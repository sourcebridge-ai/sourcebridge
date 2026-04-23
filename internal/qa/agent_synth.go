// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package qa

import (
	"context"
	"fmt"

	reasoningv1 "github.com/sourcebridge/sourcebridge/gen/go/reasoning/v1"
)

// WorkerAgentSynthesizer adapts a worker client + its
// capability-probe result into the orchestrator's AgentSynthesizer
// interface. Constructed once on server startup after a successful
// GetProviderCapabilities call; the capability bit is cached on the
// adapter so the loop's gate is a cheap local read.
type WorkerAgentSynthesizer struct {
	worker       agentWorkerCaller
	toolsEnabled bool
}

// agentWorkerCaller is the narrow surface we need from
// *worker.Client. Kept as an interface so tests can inject a fake
// without importing grpc.
type agentWorkerCaller interface {
	AnswerQuestionWithTools(ctx context.Context, req *reasoningv1.AnswerQuestionWithToolsRequest) (*reasoningv1.AnswerQuestionWithToolsResponse, error)
	IsAvailable() bool
}

// NewWorkerAgentSynthesizer constructs the adapter. `toolsEnabled`
// comes from the GetProviderCapabilities probe; when false, the
// loop must not be entered.
func NewWorkerAgentSynthesizer(worker agentWorkerCaller, toolsEnabled bool) *WorkerAgentSynthesizer {
	return &WorkerAgentSynthesizer{worker: worker, toolsEnabled: toolsEnabled}
}

// SupportsTools mirrors the capability bit.
func (s *WorkerAgentSynthesizer) SupportsTools() bool {
	if s == nil || s.worker == nil {
		return false
	}
	if !s.worker.IsAvailable() {
		return false
	}
	return s.toolsEnabled
}

// AnswerQuestionWithTools translates the Go-native AgentTurnRequest
// into proto, calls the worker, and translates the response back.
func (s *WorkerAgentSynthesizer) AnswerQuestionWithTools(
	ctx context.Context,
	req AgentTurnRequest,
) (AgentTurn, error) {
	if s == nil || s.worker == nil {
		return AgentTurn{}, fmt.Errorf("agent synth: worker client is nil")
	}

	protoMsgs := make([]*reasoningv1.AgentMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		protoMsgs = append(protoMsgs, toProtoAgentMessage(m))
	}

	protoTools := make([]*reasoningv1.ToolSchema, 0, len(req.Tools))
	for _, t := range req.Tools {
		protoTools = append(protoTools, &reasoningv1.ToolSchema{
			Name:            t.Name,
			Description:     t.Description,
			InputSchemaJson: t.InputSchemaJSON,
		})
	}

	resp, err := s.worker.AnswerQuestionWithTools(ctx, &reasoningv1.AnswerQuestionWithToolsRequest{
		RepositoryId: req.RepositoryID,
		Messages:     protoMsgs,
		Tools:        protoTools,
		MaxTokens:    int32(req.MaxTokens),
	})
	if err != nil {
		return AgentTurn{}, err
	}
	if !resp.GetCapabilitySupported() {
		return AgentTurn{}, fmt.Errorf("agent synth: capability_supported=false (%s)", resp.GetTerminationHint())
	}

	turn := AgentTurn{
		Role:            AgentRoleAssistant,
		Text:            resp.GetTurn().GetText(),
		TerminationHint: resp.GetTerminationHint(),
	}
	for _, c := range resp.GetTurn().GetToolCalls() {
		turn.ToolCalls = append(turn.ToolCalls, ToolCall{
			CallID: c.GetCallId(),
			Name:   c.GetName(),
			Args:   []byte(c.GetArgsJson()),
		})
	}
	if u := resp.GetUsage(); u != nil {
		turn.Model = u.GetModel()
		turn.InputTokens = int(u.GetInputTokens())
		turn.OutputTokens = int(u.GetOutputTokens())
	}
	return turn, nil
}

func toProtoAgentMessage(m AgentMessage) *reasoningv1.AgentMessage {
	out := &reasoningv1.AgentMessage{
		Role: string(m.Role),
		Text: m.Text,
	}
	for _, c := range m.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, &reasoningv1.ToolCall{
			CallId:   c.CallID,
			Name:     c.Name,
			ArgsJson: string(c.Args),
		})
	}
	for _, r := range m.ToolResults {
		out.ToolResults = append(out.ToolResults, &reasoningv1.ToolResult{
			CallId:   r.CallID,
			Ok:       r.OK,
			DataJson: string(r.Data),
			Error:    r.Error,
			Hint:     r.Hint,
		})
	}
	return out
}
