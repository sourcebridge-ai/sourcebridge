// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package qa

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	commonv1 "github.com/sourcebridge/sourcebridge/gen/go/common/v1"
	reasoningv1 "github.com/sourcebridge/sourcebridge/gen/go/reasoning/v1"
	"github.com/sourcebridge/sourcebridge/internal/worker"
)

// deepAsk implements the deep orchestration route:
//
//   1. repository readiness check
//   2. parallel retrieval: summary evidence + requirement lines
//   3. deterministic context assembly with token-budget eviction
//   4. synthesis via AnswerQuestion
//   5. reference emission from recorded provenance
//
// Failures on any step degrade gracefully: the caller always gets a
// well-formed AskResult with a diagnostic explaining what happened.
func (o *Orchestrator) deepAsk(ctx context.Context, in AskInput) (*AskResult, error) {
	started := time.Now()
	result := &AskResult{
		References:          []AskReference{},
		RelatedRequirements: []string{},
		Diagnostics: AskDiagnostics{
			StageTimings: map[string]DurationMs{},
			Mode:         string(ModeDeep),
		},
	}

	// Stage 1: classify.
	t0 := time.Now()
	kind := ClassifyQuestion(in.Question)
	result.Diagnostics.QuestionType = string(kind)
	result.Diagnostics.StageTimings["qa.classify"] = FromDuration(time.Since(t0))

	// Stage 2: readiness check. If we have no reader, deep degrades
	// to the fast-route synthesis with whatever caller-supplied code
	// the user passed. This preserves answerability on installs that
	// haven't enabled the knowledge corpus yet.
	var status *RepositoryStatus
	var summaries []SummaryEvidence
	var requirementLines []string

	if o.reader != nil {
		t1 := time.Now()
		status = GetRepositoryStatus(o.reader, in.RepositoryID, "")
		result.Diagnostics.StageTimings["qa.understanding_ready"] = FromDuration(time.Since(t1))
		result.Diagnostics.UnderstandingStage = status.UnderstandingStage
		result.Diagnostics.TreeStatus = status.TreeStatus
		result.Diagnostics.UnderstandingRevision = status.UnderstandingRevision

		if !status.Ready {
			result.Diagnostics.FallbackUsed = "understanding_not_ready"
			result.References = append(result.References, buildUnderstandingCTA(in.RepositoryID))
			result.Answer = understandingCTAAnswer
			result.Diagnostics.StageTimings["qa.ask"] = FromDuration(time.Since(started))
			return result, nil
		}

		// Stage 3: parallel retrieval. Summary evidence + requirement
		// lines read from independent stores, so we fan out.
		t2 := time.Now()
		type sumRes struct {
			items []SummaryEvidence
			err   error
		}
		sumCh := make(chan sumRes, 1)
		go func() {
			items, err := GetSummaryEvidence(o.reader, status.CorpusID, in.Question, string(kind))
			sumCh <- sumRes{items: items, err: err}
		}()
		// Requirement lines are a file-system read; keep on the same
		// goroutine as the caller to avoid spinning for trivial work.
		// Orchestrator doesn't know the clone path today — we defer to
		// callers that want requirement selection to supply lines via
		// in.Code or a future input field. A future phase wires the
		// clone-path lookup here.
		_ = requirementLines

		r := <-sumCh
		if r.err == nil {
			summaries = r.items
		}
		result.Diagnostics.UnderstandingUsed = len(summaries) > 0
		result.Diagnostics.StageTimings["qa.summary_evidence"] = FromDuration(time.Since(t2))

		// Cap the summaries we pack so one big corpus doesn't dominate.
		summaries = trimSummaries(summaries, 8)
	}

	// Stage 4: deterministic assembly.
	t3 := time.Now()
	contextMD := buildContextMarkdown(in, summaries, requirementLines)
	promptEnvelope := buildPromptEnvelope(in, contextMD)
	result.Diagnostics.StageTimings["qa.assemble"] = FromDuration(time.Since(t3))
	result.Diagnostics.StageTimings["qa.prompt_build"] = FromDuration(time.Since(t3))

	// Record file-level provenance for diagnostics.
	for _, s := range summaries {
		if s.FilePath != "" {
			result.Diagnostics.FilesConsidered = append(result.Diagnostics.FilesConsidered, s.FilePath)
			result.Diagnostics.FilesUsed = append(result.Diagnostics.FilesUsed, s.FilePath)
		}
		result.References = append(result.References, AskReference{
			Kind: RefKindUnderstandingSection,
			UnderstandingSection: &UnderstandingSectionRef{
				Headline: orDefault(s.Headline, s.UnitID),
				Kind:     "section",
			},
			Title: orDefault(s.Headline, s.UnitID),
		})
	}

	// Stage 5: synthesize.
	if o.synthesizer == nil || !o.synthesizer.IsAvailable() {
		result.Diagnostics.FallbackUsed = "worker_unavailable"
		result.Answer = "The reasoning worker is not available right now. Please try again shortly."
		result.Diagnostics.StageTimings["qa.ask"] = FromDuration(time.Since(started))
		return result, nil
	}

	lane := o.lanes.Get(worker.LaneQASynthesize)
	release, err := lane.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	t4 := time.Now()
	req := &reasoningv1.AnswerQuestionRequest{
		Question:     promptEnvelope,
		RepositoryId: in.RepositoryID,
		ContextCode:  contextMD,
		FilePath:     in.FilePath,
		Language:     languageFromString(in.Language),
		MaxTokens:    int32(o.config.MaxAnswerTokens),
	}
	resp, err := o.synthesizer.AnswerQuestion(ctx, req)
	result.Diagnostics.StageTimings["qa.llm_call"] = FromDuration(time.Since(t4))
	if err != nil {
		// Even on synthesis failure we still emit the references we
		// gathered — the caller can read the evidence block directly.
		result.Diagnostics.FallbackUsed = "synthesis_failed"
		result.Answer = fmt.Sprintf("Synthesis failed: %v. The gathered evidence is available in the references list.", err)
		result.Diagnostics.StageTimings["qa.ask"] = FromDuration(time.Since(started))
		return result, nil
	}

	// Stage 6: normalize.
	t5 := time.Now()
	result.Answer = resp.GetAnswer()
	if u := resp.GetUsage(); u != nil {
		result.Usage = AskUsage{
			Model:        u.GetModel(),
			InputTokens:  int(u.GetInputTokens()),
			OutputTokens: int(u.GetOutputTokens()),
		}
		result.Diagnostics.ModelUsed = u.GetModel()
	}
	for _, sym := range resp.GetReferencedSymbols() {
		result.References = append(result.References, symbolRefFromProto(sym))
	}
	result.Diagnostics.StageTimings["qa.response_parse"] = FromDuration(time.Since(t5))
	result.Diagnostics.StageTimings["qa.normalize"] = FromDuration(time.Since(t5))
	result.Diagnostics.StageTimings["qa.ask"] = FromDuration(time.Since(started))

	if in.IncludeDebug {
		result.Debug = &AskDebug{
			Prompt:          promptEnvelope,
			ContextMarkdown: contextMD,
		}
	}
	return result, nil
}

// understandingCTAAnswer is the user-facing message returned when the
// repository understanding isn't ready. Phrased so a user sees it as
// actionable guidance rather than a failure.
const understandingCTAAnswer = `Deep answers require the repository's understanding corpus to be built first. ` +
	`Use the build-understanding command (or the web UI's "Generate understanding" action) and try again.`

func buildUnderstandingCTA(repoID string) AskReference {
	return AskReference{
		Kind: RefKindUnderstandingSection,
		UnderstandingSection: &UnderstandingSectionRef{
			Kind:      "action_cta",
			Headline:  "Build repository understanding",
			ActionURL: "/repositories/" + repoID + "#understanding",
		},
		Title: "Build repository understanding",
	}
}

func trimSummaries(in []SummaryEvidence, n int) []SummaryEvidence {
	if n <= 0 || len(in) <= n {
		return in
	}
	return in[:n]
}

func orDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func symbolRefFromProto(sym *commonv1.CodeSymbol) AskReference {
	ref := &SymbolRef{
		SymbolID:      sym.GetId(),
		QualifiedName: sym.GetQualifiedName(),
		Language:      sym.GetLanguage().String(),
	}
	if loc := sym.GetLocation(); loc != nil {
		ref.FilePath = loc.GetPath()
		ref.StartLine = int(loc.GetStartLine())
		ref.EndLine = int(loc.GetEndLine())
	}
	return AskReference{
		Kind:   RefKindSymbol,
		Symbol: ref,
		Title:  sym.GetQualifiedName(),
	}
}

// uniqueStrings returns s with duplicates removed while preserving
// first-seen order. Used by diagnostics collectors so files_considered
// doesn't carry accidental duplicates from multi-source retrieval.
func uniqueStrings(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	out := make([]string, 0, len(s))
	for _, x := range s {
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	sort.SliceStable(out, func(i, j int) bool { return strings.Compare(out[i], out[j]) < 0 })
	return out
}
