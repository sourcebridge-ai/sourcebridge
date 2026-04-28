// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package rest

import (
	"encoding/json"
	"testing"
)

// TestClusteringToolsRegistered verifies that the three clustering MCP tools
// appear in the handler's tool list. The test uses the in-memory store (which
// does not implement ClusterStore), so the tools are registered but will
// return "unavailable" / empty responses rather than real cluster data.
func TestClusteringToolsRegistered(t *testing.T) {
	h := newTestHarness(t)
	sess := h.createSession()

	resp := h.sendRPC(sess, 1, "tools/list", map[string]interface{}{})
	if resp.Error != nil {
		t.Fatalf("tools/list failed: %v", resp.Error)
	}

	var body struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	raw, _ := json.Marshal(resp.Result)
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal tools/list result: %v", err)
	}

	toolSet := make(map[string]bool)
	for _, t := range body.Tools {
		toolSet[t.Name] = true
	}

	required := []string{"get_subsystems", "get_subsystem_by_id", "get_subsystem"}
	for _, name := range required {
		if !toolSet[name] {
			t.Errorf("expected tool %q in tools/list, not found", name)
		}
	}
}

// TestGetSubsystems_NoClusters verifies the schema of get_subsystems when the
// store has no clusters. The in-memory store does not implement ClusterStore,
// so the handler returns status "unavailable".
func TestGetSubsystems_NoClusters(t *testing.T) {
	h := newTestHarness(t)
	sess := h.createSession()

	resp := h.sendRPC(sess, 2, "tools/call", map[string]interface{}{
		"name": "get_subsystems",
		"arguments": map[string]interface{}{
			"repo_id": h.repoID,
		},
	})
	if resp.Error != nil {
		t.Fatalf("get_subsystems error: %v", resp.Error)
	}

	// The result is wrapped in a tools/call response with content[0].text.
	var toolResult struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	raw, _ := json.Marshal(resp.Result)
	if err := json.Unmarshal(raw, &toolResult); err != nil {
		t.Fatalf("unmarshal tool result: %v", err)
	}
	if len(toolResult.Content) == 0 {
		t.Fatal("expected at least one content item")
	}

	// Parse the JSON body.
	var body struct {
		RepoID   string      `json:"repo_id"`
		Status   string      `json:"status"`
		Clusters interface{} `json:"clusters"`
	}
	if err := json.Unmarshal([]byte(toolResult.Content[0].Text), &body); err != nil {
		t.Fatalf("unmarshal get_subsystems body: %v", err)
	}
	if body.RepoID != h.repoID {
		t.Errorf("expected repo_id %q, got %q", h.repoID, body.RepoID)
	}
	// Status should be "unavailable" or "pending" when no clusters exist.
	if body.Status != "unavailable" && body.Status != "pending" {
		t.Errorf("unexpected status %q", body.Status)
	}
}

// TestGetSubsystemByID_InvalidID verifies that get_subsystem_by_id returns an
// error (not a panic or server error) when given an invalid cluster ID.
func TestGetSubsystemByID_InvalidID(t *testing.T) {
	h := newTestHarness(t)
	sess := h.createSession()

	resp := h.sendRPC(sess, 3, "tools/call", map[string]interface{}{
		"name": "get_subsystem_by_id",
		"arguments": map[string]interface{}{
			"cluster_id": "this id has spaces and is invalid!!!",
		},
	})
	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %v", resp.Error)
	}
	// The tool should return IsError:true in the content.
	var toolResult struct {
		IsError bool `json:"isError"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	raw, _ := json.Marshal(resp.Result)
	_ = json.Unmarshal(raw, &toolResult)
	if !toolResult.IsError {
		t.Error("expected IsError=true for invalid cluster_id")
	}
}

// TestGetSubsystem_InvalidParams verifies that get_subsystem validates input IDs.
func TestGetSubsystem_InvalidParams(t *testing.T) {
	h := newTestHarness(t)
	sess := h.createSession()

	resp := h.sendRPC(sess, 4, "tools/call", map[string]interface{}{
		"name": "get_subsystem",
		"arguments": map[string]interface{}{
			"repo_id":   h.repoID,
			"symbol_id": "id with spaces!!!",
		},
	})
	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %v", resp.Error)
	}
	var toolResult struct {
		IsError bool `json:"isError"`
	}
	raw, _ := json.Marshal(resp.Result)
	_ = json.Unmarshal(raw, &toolResult)
	if !toolResult.IsError {
		t.Error("expected IsError=true for invalid symbol_id")
	}
}

// TestGetSubsystem_NoCluster verifies the response shape when a symbol is not
// in any cluster.
func TestGetSubsystem_NoCluster(t *testing.T) {
	h := newTestHarness(t)
	sess := h.createSession()

	resp := h.sendRPC(sess, 5, "tools/call", map[string]interface{}{
		"name": "get_subsystem",
		"arguments": map[string]interface{}{
			"repo_id":   h.repoID,
			"symbol_id": "somesymbolid123",
		},
	})
	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %v", resp.Error)
	}
	var toolResult struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	raw, _ := json.Marshal(resp.Result)
	_ = json.Unmarshal(raw, &toolResult)
	if len(toolResult.Content) == 0 {
		t.Fatal("expected content")
	}

	var body struct {
		RepoID   string      `json:"repo_id"`
		SymbolID string      `json:"symbol_id"`
		Cluster  interface{} `json:"cluster"`
		Message  string      `json:"message"`
	}
	_ = json.Unmarshal([]byte(toolResult.Content[0].Text), &body)
	if body.SymbolID != "somesymbolid123" {
		t.Errorf("expected symbol_id to be echoed back, got %q", body.SymbolID)
	}
}
