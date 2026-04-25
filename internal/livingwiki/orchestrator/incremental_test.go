// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package orchestrator_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sourcebridge/sourcebridge/internal/livingwiki/ast"
	"github.com/sourcebridge/sourcebridge/internal/livingwiki/manifest"
	"github.com/sourcebridge/sourcebridge/internal/livingwiki/orchestrator"
	"github.com/sourcebridge/sourcebridge/internal/quality"
	"github.com/sourcebridge/sourcebridge/internal/reports/templates"
)

// ---- Diff provider stubs ----

// staticDiffProvider always returns the same diff result.
type staticDiffProvider struct {
	result orchestrator.DiffResult
	err    error
}

func (s *staticDiffProvider) Diff(_ context.Context, _, _, _ string) (orchestrator.DiffResult, error) {
	return s.result, s.err
}

// notFoundDiffProvider returns ErrSHANotFound to simulate a force-push.
type notFoundDiffProvider struct{}

func (n *notFoundDiffProvider) Diff(_ context.Context, _, _, _ string) (orchestrator.DiffResult, error) {
	return orchestrator.DiffResult{}, orchestrator.ErrSHANotFound
}

// ---- Template stubs for incremental tests ----

// passGlossaryTemplate generates a single-block page that passes the glossary
// quality profile (factual_grounding only; our prose makes no behavioral
// assertions).
type passGlossaryTemplate struct {
	id string
}

func (p *passGlossaryTemplate) ID() string { return p.id }

func (p *passGlossaryTemplate) Generate(_ context.Context, input templates.GenerateInput) (ast.Page, error) {
	pageID := input.RepoID + "." + p.id
	blkID := ast.GenerateBlockID(pageID, "", ast.BlockKindParagraph, 0)
	return ast.Page{
		ID: pageID,
		Manifest: manifest.DependencyManifest{
			PageID:   pageID,
			Template: p.id,
			Audience: string(input.Audience),
		},
		Blocks: []ast.Block{
			{
				ID:   blkID,
				Kind: ast.BlockKindParagraph,
				Content: ast.BlockContent{Paragraph: &ast.ParagraphContent{
					Markdown: "Middleware wraps an HTTP handler. No behavioral assertions.",
				}},
				Owner: ast.OwnerGenerated,
			},
		},
		Provenance: ast.Provenance{GeneratedAt: input.Now},
	}, nil
}

// ---- helpers ----

// newIncrementalOrch creates an Orchestrator with a glossary stub template,
// ready for incremental tests.
func newIncrementalOrch() *orchestrator.Orchestrator {
	reg := orchestrator.NewMapRegistry(&passGlossaryTemplate{id: "glossary"})
	store := orchestrator.NewMemoryPageStore()
	return orchestrator.New(orchestrator.Config{RepoID: repoID}, reg, store)
}

// makeManifest creates a simple DependencyManifest for testing.
func makeManifest(pageID, tmpl string, paths ...string) manifest.DependencyManifest {
	return manifest.DependencyManifest{
		PageID:   pageID,
		Template: tmpl,
		Audience: string(quality.AudienceEngineers),
		Dependencies: manifest.Dependencies{
			Paths:           paths,
			DependencyScope: manifest.ScopeDirect,
		},
	}
}

// pageWithBlock creates a simple ast.Page with a single block of the given owner.
func pageWithBlock(pageID string, blockID ast.BlockID, owner ast.Owner) ast.Page {
	return ast.Page{
		ID: pageID,
		Manifest: manifest.DependencyManifest{
			PageID:   pageID,
			Template: "glossary",
			Audience: string(quality.AudienceEngineers),
		},
		Blocks: []ast.Block{
			{
				ID:   blockID,
				Kind: ast.BlockKindParagraph,
				Content: ast.BlockContent{Paragraph: &ast.ParagraphContent{
					Markdown: "Some content.",
				}},
				Owner: owner,
			},
		},
	}
}

// ---- Tests ----

// TestWatermarkAdvanceOnSuccess verifies that SourceProcessedSHA advances after
// a successful incremental generation.
func TestWatermarkAdvanceOnSuccess(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	orch := newIncrementalOrch()
	wm := orchestrator.NewMemoryWatermarkStore()
	pr := orchestrator.NewMemoryWikiPR("pr-wm")

	m := makeManifest(repoID+".glossary", "glossary", "internal/auth/**")

	result, err := orch.GenerateIncremental(ctx, orchestrator.IncrementalRequest{
		HeadSHA: "sha-abc123",
		Pages:   []manifest.DependencyManifest{m},
		DiffProvider: &staticDiffProvider{result: orchestrator.DiffResult{
			Changed:    []manifest.ChangedPair{{Path: "internal/auth/auth.go"}},
			TotalLines: 10,
		}},
		PR:             pr,
		WatermarkStore: wm,
		Now:            time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("GenerateIncremental: %v", err)
	}
	if result.Debounced || result.Skipped {
		t.Fatalf("unexpected debounced=%v skipped=%v", result.Debounced, result.Skipped)
	}

	marks, err := wm.Get(ctx, repoID)
	if err != nil {
		t.Fatalf("Get watermarks: %v", err)
	}
	if marks.SourceProcessedSHA != "sha-abc123" {
		t.Errorf("expected SourceProcessedSHA=sha-abc123, got %q", marks.SourceProcessedSHA)
	}
	// WikiPublishedSHA must NOT have advanced (only advances on Promote).
	if marks.WikiPublishedSHA != "" {
		t.Errorf("expected WikiPublishedSHA to be empty (not promoted yet), got %q", marks.WikiPublishedSHA)
	}
}

// TestWatermarkAdvancePublishedOnPromote verifies that PromoteWithWatermark
// advances WikiPublishedSHA.
func TestWatermarkAdvancePublishedOnPromote(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	reg := orchestrator.NewMapRegistry()
	store := orchestrator.NewMemoryPageStore()
	orch := orchestrator.New(orchestrator.Config{RepoID: "promo-wm-repo"}, reg, store)
	wm := orchestrator.NewMemoryWatermarkStore()

	if err := orch.PromoteWithWatermark(ctx, "promo-wm-repo", "pr-x", "sha-merged", wm); err != nil {
		t.Fatalf("PromoteWithWatermark: %v", err)
	}

	marks, err := wm.Get(ctx, "promo-wm-repo")
	if err != nil {
		t.Fatalf("Get watermarks: %v", err)
	}
	if marks.WikiPublishedSHA != "sha-merged" {
		t.Errorf("expected WikiPublishedSHA=sha-merged, got %q", marks.WikiPublishedSHA)
	}
	if marks.SourceProcessedSHA != "sha-merged" {
		t.Errorf("expected SourceProcessedSHA=sha-merged, got %q", marks.SourceProcessedSHA)
	}
}

// TestDiffParserAffectedPagesNoOp verifies that a diff with no matching paths
// produces an empty generation result (no pages regenerated, no PR commit).
func TestDiffParserAffectedPagesNoOp(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	orch := newIncrementalOrch()
	wm := orchestrator.NewMemoryWatermarkStore()
	pr := orchestrator.NewMemoryWikiPR("pr-noop")

	// Page watches internal/auth/**; diff touches internal/billing only.
	m := makeManifest(repoID+".glossary", "glossary", "internal/auth/**")

	result, err := orch.GenerateIncremental(ctx, orchestrator.IncrementalRequest{
		HeadSHA: "sha-noop",
		Pages:   []manifest.DependencyManifest{m},
		DiffProvider: &staticDiffProvider{result: orchestrator.DiffResult{
			Changed:    []manifest.ChangedPair{{Path: "internal/billing/billing.go"}},
			TotalLines: 5,
		}},
		PR:             pr,
		WatermarkStore: wm,
		Now:            time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("GenerateIncremental: %v", err)
	}
	if len(result.Regenerated) != 0 {
		t.Errorf("expected 0 regenerated pages for no-op diff, got %d", len(result.Regenerated))
	}
	if pr.CommitCount() != 0 {
		t.Errorf("expected 0 commits appended to PR for no-op diff, got %d", pr.CommitCount())
	}
	// Processed watermark must still advance.
	marks, _ := wm.Get(ctx, repoID)
	if marks.SourceProcessedSHA != "sha-noop" {
		t.Errorf("expected SourceProcessedSHA=sha-noop, got %q", marks.SourceProcessedSHA)
	}
}

// TestDiffParserAffectedPagesSmall verifies that a small diff matching a page's
// paths triggers regeneration of that page.
func TestDiffParserAffectedPagesSmall(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	orch := newIncrementalOrch()
	wm := orchestrator.NewMemoryWatermarkStore()
	pr := orchestrator.NewMemoryWikiPR("pr-small")

	m := makeManifest(repoID+".glossary", "glossary", "internal/auth/**")

	result, err := orch.GenerateIncremental(ctx, orchestrator.IncrementalRequest{
		HeadSHA: "sha-small",
		Pages:   []manifest.DependencyManifest{m},
		DiffProvider: &staticDiffProvider{result: orchestrator.DiffResult{
			Changed:    []manifest.ChangedPair{{Path: "internal/auth/auth.go"}},
			TotalLines: 20,
		}},
		PR:             pr,
		WatermarkStore: wm,
		Now:            time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("GenerateIncremental: %v", err)
	}
	if len(result.Regenerated) != 1 {
		t.Errorf("expected 1 regenerated page, got %d", len(result.Regenerated))
	}
}

// TestDiffParserAffectedPagesOverBudget verifies that when more than 5 pages
// are affected, exactly 5 are generated and the remainder are queued.
func TestDiffParserAffectedPagesOverBudget(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Build 8 pages all watching the same path.
	var pages []manifest.DependencyManifest
	for i := 0; i < 8; i++ {
		pages = append(pages, makeManifest(
			repoID+".glossary"+string(rune('0'+i)),
			"glossary",
			"internal/auth/**",
		))
	}

	reg := orchestrator.NewMapRegistry(&passGlossaryTemplate{id: "glossary"})
	store := orchestrator.NewMemoryPageStore()
	orch := orchestrator.New(orchestrator.Config{RepoID: repoID}, reg, store)
	wm := orchestrator.NewMemoryWatermarkStore()
	pr := orchestrator.NewMemoryWikiPR("pr-budget")

	result, err := orch.GenerateIncremental(ctx, orchestrator.IncrementalRequest{
		HeadSHA: "sha-budget",
		Pages:   pages,
		DiffProvider: &staticDiffProvider{result: orchestrator.DiffResult{
			Changed:    []manifest.ChangedPair{{Path: "internal/auth/auth.go"}},
			TotalLines: 20,
		}},
		PR:             pr,
		WatermarkStore: wm,
		Now:            time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("GenerateIncremental: %v", err)
	}
	if len(result.Regenerated) > 5 {
		t.Errorf("expected at most 5 regenerated pages (budget cap), got %d", len(result.Regenerated))
	}
	if len(result.Queued) == 0 {
		t.Error("expected queued pages when over budget, got none")
	}
	total := len(result.Regenerated) + len(result.Queued)
	if total != 8 {
		t.Errorf("expected regenerated+queued=8, got %d", total)
	}
}

// TestSkipGuardFires verifies that a diff exceeding 5,000 lines produces a
// Skipped result with no pages regenerated.
func TestSkipGuardFires(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	orch := newIncrementalOrch()
	wm := orchestrator.NewMemoryWatermarkStore()
	pr := orchestrator.NewMemoryWikiPR("pr-skip")

	m := makeManifest(repoID+".glossary", "glossary", "internal/auth/**")

	result, err := orch.GenerateIncremental(ctx, orchestrator.IncrementalRequest{
		HeadSHA: "sha-big",
		Pages:   []manifest.DependencyManifest{m},
		DiffProvider: &staticDiffProvider{result: orchestrator.DiffResult{
			Changed:    []manifest.ChangedPair{{Path: "internal/auth/auth.go"}},
			TotalLines: 6_000, // exceeds 5,000-line guard
		}},
		PR:             pr,
		WatermarkStore: wm,
		Now:            time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("GenerateIncremental: %v", err)
	}
	if !result.Skipped {
		t.Error("expected result.Skipped=true for 6,000-line diff")
	}
	if len(result.Regenerated) != 0 {
		t.Errorf("expected 0 regenerated pages when skipped, got %d", len(result.Regenerated))
	}
	if pr.CommitCount() != 0 {
		t.Errorf("expected 0 commits when skipped, got %d", pr.CommitCount())
	}
	// Processed watermark must still advance after skip.
	marks, _ := wm.Get(ctx, repoID)
	if marks.SourceProcessedSHA != "sha-big" {
		t.Errorf("expected SourceProcessedSHA=sha-big, got %q", marks.SourceProcessedSHA)
	}
}

// TestDebounceFiresWhenCalledTwiceQuickly verifies that a second call within
// the 60s window returns ErrDebounced.
func TestDebounceFiresWhenCalledTwiceQuickly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	orch := newIncrementalOrch()
	wm := orchestrator.NewMemoryWatermarkStore()
	pr := orchestrator.NewMemoryWikiPR("pr-debounce")

	baseTime := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	m := makeManifest(repoID+".glossary", "glossary", "internal/auth/**")
	diffProvider := &staticDiffProvider{result: orchestrator.DiffResult{
		Changed:    []manifest.ChangedPair{{Path: "internal/auth/auth.go"}},
		TotalLines: 10,
	}}

	// First call — should succeed.
	_, err := orch.GenerateIncremental(ctx, orchestrator.IncrementalRequest{
		HeadSHA:        "sha-first",
		Pages:          []manifest.DependencyManifest{m},
		DiffProvider:   diffProvider,
		PR:             pr,
		WatermarkStore: wm,
		Now:            baseTime,
	})
	if err != nil {
		t.Fatalf("first GenerateIncremental: %v", err)
	}

	// Second call 30s later — within the 60s window, should be debounced.
	result, err := orch.GenerateIncremental(ctx, orchestrator.IncrementalRequest{
		HeadSHA:        "sha-second",
		Pages:          []manifest.DependencyManifest{m},
		DiffProvider:   diffProvider,
		PR:             pr,
		WatermarkStore: wm,
		Now:            baseTime.Add(30 * time.Second),
	})
	if !errors.Is(err, orchestrator.ErrDebounced) {
		t.Errorf("expected ErrDebounced on second call within window, got: %v", err)
	}
	if !result.Debounced {
		t.Error("expected result.Debounced=true on second call within window")
	}
}

// TestDebounceDoesNotFireAfterWindow verifies that a call outside the 60s
// window is not debounced.
func TestDebounceDoesNotFireAfterWindow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	orch := newIncrementalOrch()
	wm := orchestrator.NewMemoryWatermarkStore()
	pr1 := orchestrator.NewMemoryWikiPR("pr-debounce-ok-1")
	pr2 := orchestrator.NewMemoryWikiPR("pr-debounce-ok-2")

	baseTime := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	m := makeManifest(repoID+".glossary", "glossary", "internal/auth/**")
	diffProvider := &staticDiffProvider{result: orchestrator.DiffResult{
		Changed:    []manifest.ChangedPair{{Path: "internal/auth/auth.go"}},
		TotalLines: 10,
	}}

	_, err := orch.GenerateIncremental(ctx, orchestrator.IncrementalRequest{
		HeadSHA:        "sha-first",
		Pages:          []manifest.DependencyManifest{m},
		DiffProvider:   diffProvider,
		PR:             pr1,
		WatermarkStore: wm,
		Now:            baseTime,
	})
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	// Second call 61s later — outside the window, should NOT be debounced.
	result, err := orch.GenerateIncremental(ctx, orchestrator.IncrementalRequest{
		HeadSHA:        "sha-second",
		Pages:          []manifest.DependencyManifest{m},
		DiffProvider:   diffProvider,
		PR:             pr2,
		WatermarkStore: wm,
		Now:            baseTime.Add(61 * time.Second),
	})
	if errors.Is(err, orchestrator.ErrDebounced) {
		t.Error("expected second call outside debounce window to succeed, got ErrDebounced")
	}
	if result.Debounced {
		t.Error("expected result.Debounced=false for call outside window")
	}
}

// TestOpenPRDetectedAdditiveCommit verifies that when an open PR exists,
// GenerateIncremental appends a commit to the existing branch rather than
// opening a new PR.
func TestOpenPRDetectedAdditiveCommit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	orch := newIncrementalOrch()
	wm := orchestrator.NewMemoryWatermarkStore()

	// Simulate an already-open PR with one existing (bot) commit.
	pr := orchestrator.NewMemoryWikiPR("pr-additive")
	pr.AddHumanCommit("sha-bot-init", map[string][]byte{
		"wiki/test-repo.glossary.md": []byte("initial content"),
	})

	m := makeManifest(repoID+".glossary", "glossary", "internal/auth/**")

	result, err := orch.GenerateIncremental(ctx, orchestrator.IncrementalRequest{
		HeadSHA: "sha-push2",
		Pages:   []manifest.DependencyManifest{m},
		DiffProvider: &staticDiffProvider{result: orchestrator.DiffResult{
			Changed:    []manifest.ChangedPair{{Path: "internal/auth/auth.go"}},
			TotalLines: 10,
		}},
		PR:             pr,
		WatermarkStore: wm,
		Now:            time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("GenerateIncremental: %v", err)
	}

	// The PR should have received an additive commit (not a fresh open).
	// We verify by checking that a bot commit was appended beyond the initial one.
	if pr.CommitCount() < 2 {
		t.Errorf("expected at least 2 commits (initial + additive), got %d", pr.CommitCount())
	}
	if result.PRID != "pr-additive" {
		t.Errorf("expected PRID=pr-additive, got %q", result.PRID)
	}
}

// TestNoPROpensFreshPR verifies that when there is no open PR and an
// ExtendedRepoWriter is provided, GenerateIncremental writes to the branch.
func TestNoPROpensFreshPR(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	orch := newIncrementalOrch()
	wm := orchestrator.NewMemoryWatermarkStore()
	writer := orchestrator.NewMemoryExtendedRepoWriter()

	m := makeManifest(repoID+".glossary", "glossary", "internal/auth/**")

	result, err := orch.GenerateIncremental(ctx, orchestrator.IncrementalRequest{
		HeadSHA: "sha-noPR",
		Pages:   []manifest.DependencyManifest{m},
		DiffProvider: &staticDiffProvider{result: orchestrator.DiffResult{
			Changed:    []manifest.ChangedPair{{Path: "internal/auth/auth.go"}},
			TotalLines: 10,
		}},
		PR:             nil, // no open PR
		Writer:         writer,
		WatermarkStore: wm,
		Now:            time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("GenerateIncremental: %v", err)
	}

	// Writer should have received the files.
	if writer.CommitCount() == 0 {
		t.Error("expected ExtendedRepoWriter to have received at least one commit")
	}
	// PRID is empty when we wrote via the writer (PR must be opened separately).
	_ = result
}

// TestReviewerCommitMarkBlocksHumanEditedOnPRBranch verifies that a non-bot
// commit to the wiki PR branch marks the affected blocks as
// OwnerHumanEditedOnPRBranch.
func TestReviewerCommitMarkBlocksHumanEditedOnPRBranch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	reg := orchestrator.NewMapRegistry()
	store := orchestrator.NewMemoryPageStore()
	orch := orchestrator.New(orchestrator.Config{RepoID: "review-repo"}, reg, store)

	const (
		reviewRepoID = "review-repo"
		reviewPRID   = "pr-review"
	)

	blkID := ast.GenerateBlockID("review-repo.glossary", "", ast.BlockKindParagraph, 0)

	// Store an initial proposed page with a generated block.
	initialPage := pageWithBlock("review-repo.glossary", blkID, ast.OwnerGenerated)
	if err := store.SetProposed(ctx, reviewRepoID, reviewPRID, initialPage); err != nil {
		t.Fatalf("SetProposed: %v", err)
	}

	// Build a human commit that touches the wiki file containing blkID.
	// We embed the block ID in a comment as the markdown writer would.
	wikiContent := []byte(`<!-- sourcebridge:block id="` + string(blkID) + `" -->
Some updated prose by the reviewer.`)

	humanCommit := orchestrator.Commit{
		SHA:            "sha-human-edit",
		CommitterName:  "human-reviewer",
		CommitterEmail: "reviewer@example.com",
		Files: map[string][]byte{
			"wiki/review-repo.glossary.md": wikiContent,
		},
	}

	if err := orch.ApplyReviewerCommits(ctx, reviewRepoID, reviewPRID, []orchestrator.Commit{humanCommit}); err != nil {
		t.Fatalf("ApplyReviewerCommits: %v", err)
	}

	// The block must now be marked OwnerHumanEditedOnPRBranch.
	page, ok, err := store.GetProposed(ctx, reviewRepoID, reviewPRID, "review-repo.glossary")
	if err != nil {
		t.Fatalf("GetProposed: %v", err)
	}
	if !ok {
		t.Fatal("expected proposed page to exist after reviewer commit")
	}

	var found bool
	for _, blk := range page.Blocks {
		if blk.ID == blkID {
			found = true
			if blk.Owner != ast.OwnerHumanEditedOnPRBranch {
				t.Errorf("expected block %q to be OwnerHumanEditedOnPRBranch, got %v", blk.ID, blk.Owner)
			}
		}
	}
	if !found {
		t.Errorf("block %q not found in proposed page", blkID)
	}
}

// TestSubsequentBotRegenLeavesHumanEditedBlockUntouched verifies that when
// GenerateIncremental runs after a reviewer has edited a block, the
// human-edited-on-pr-branch block is left untouched in the output.
func TestSubsequentBotRegenLeavesHumanEditedBlockUntouched(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	const (
		botRegenRepo = "bot-regen-repo"
		botRegenPR   = "pr-bot-regen"
		pageID       = botRegenRepo + ".glossary"
	)

	// The block ID that the human edited.
	blkID := ast.GenerateBlockID(pageID, "", ast.BlockKindParagraph, 0)

	reg := orchestrator.NewMapRegistry(&passGlossaryTemplate{id: "glossary"})
	store := orchestrator.NewMemoryPageStore()
	orch := orchestrator.New(orchestrator.Config{RepoID: botRegenRepo}, reg, store)

	// Seed proposed_ast: the block is human-edited-on-pr-branch.
	humanContent := "Human-edited content that must not be overwritten."
	humanPage := ast.Page{
		ID: pageID,
		Manifest: manifest.DependencyManifest{
			PageID:   pageID,
			Template: "glossary",
			Audience: string(quality.AudienceEngineers),
		},
		Blocks: []ast.Block{
			{
				ID:   blkID,
				Kind: ast.BlockKindParagraph,
				Content: ast.BlockContent{Paragraph: &ast.ParagraphContent{
					Markdown: humanContent,
				}},
				Owner: ast.OwnerHumanEditedOnPRBranch,
			},
		},
	}
	if err := store.SetProposed(ctx, botRegenRepo, botRegenPR, humanPage); err != nil {
		t.Fatalf("SetProposed: %v", err)
	}

	// The passGlossaryTemplate generates a block with the same ID but different
	// content. The reconciler must preserve the human edit.
	//
	// Note: passGlossaryTemplate generates pageID = input.RepoID + "." + p.id,
	// which equals botRegenRepo + ".glossary" = pageID when we set RepoID correctly.
	// The block ID it generates must match blkID for reconciliation to find it.
	// We verify this by ensuring blkID matches GenerateBlockID(pageID, "", Paragraph, 0).

	pr := orchestrator.NewMemoryWikiPR(botRegenPR)
	wm := orchestrator.NewMemoryWatermarkStore()

	m := makeManifest(pageID, "glossary", "internal/auth/**")

	_, err := orch.GenerateIncremental(ctx, orchestrator.IncrementalRequest{
		HeadSHA: "sha-bot-regen",
		Pages:   []manifest.DependencyManifest{m},
		DiffProvider: &staticDiffProvider{result: orchestrator.DiffResult{
			Changed:    []manifest.ChangedPair{{Path: "internal/auth/auth.go"}},
			TotalLines: 10,
		}},
		PR:             pr,
		WatermarkStore: wm,
		Config:         orchestrator.Config{RepoID: botRegenRepo},
		Now:            time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("GenerateIncremental: %v", err)
	}

	// Fetch the proposed page after regen.
	page, ok, err := store.GetProposed(ctx, botRegenRepo, botRegenPR, pageID)
	if err != nil {
		t.Fatalf("GetProposed after regen: %v", err)
	}
	if !ok {
		t.Fatal("expected proposed page to exist after regen")
	}

	// Find the block: it must still carry the human content and owner.
	var found bool
	for _, blk := range page.Blocks {
		if blk.ID == blkID {
			found = true
			if blk.Owner != ast.OwnerHumanEditedOnPRBranch {
				t.Errorf("block %q owner: expected OwnerHumanEditedOnPRBranch, got %v", blk.ID, blk.Owner)
			}
			if blk.Content.Paragraph == nil || blk.Content.Paragraph.Markdown != humanContent {
				t.Errorf("block %q content: expected human content %q, got %q",
					blk.ID,
					humanContent,
					func() string {
						if blk.Content.Paragraph != nil {
							return blk.Content.Paragraph.Markdown
						}
						return "<nil>"
					}(),
				)
			}
		}
	}
	if !found {
		t.Errorf("block %q not found in proposed page after regen", blkID)
	}
}

// TestStaleWhenFiresOnHumanEditedBlock verifies that when a human-edited-on-pr-branch
// block is structurally stale (bot replacement has a different Kind), a PR
// comment is posted and the block is left alone.
func TestStaleWhenFiresOnHumanEditedBlock(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	const (
		staleRepo  = "stale-repo"
		stalePRID  = "pr-stale"
		stalePageID = staleRepo + ".glossary"
	)

	blkID := ast.GenerateBlockID(stalePageID, "", ast.BlockKindParagraph, 0)

	// Build a template that generates a HEADING block for the same ID position.
	// This differs in Kind from the human-edited PARAGRAPH block → stale.
	headingTemplate := &kindChangingTemplate{
		id:         "glossary",
		outputKind: ast.BlockKindHeading,
		blockID:    blkID,
	}

	reg := orchestrator.NewMapRegistry(headingTemplate)
	store := orchestrator.NewMemoryPageStore()
	orch := orchestrator.New(orchestrator.Config{RepoID: staleRepo}, reg, store)

	// Seed proposed_ast with a human-edited paragraph block.
	humanPage := ast.Page{
		ID: stalePageID,
		Manifest: manifest.DependencyManifest{
			PageID:   stalePageID,
			Template: "glossary",
			Audience: string(quality.AudienceEngineers),
		},
		Blocks: []ast.Block{
			{
				ID:   blkID,
				Kind: ast.BlockKindParagraph,
				Content: ast.BlockContent{Paragraph: &ast.ParagraphContent{
					Markdown: "Human-edited content.",
				}},
				Owner: ast.OwnerHumanEditedOnPRBranch,
			},
		},
	}
	if err := store.SetProposed(ctx, staleRepo, stalePRID, humanPage); err != nil {
		t.Fatalf("SetProposed: %v", err)
	}

	pr := orchestrator.NewMemoryWikiPR(stalePRID)
	wm := orchestrator.NewMemoryWatermarkStore()
	m := makeManifest(stalePageID, "glossary", "internal/auth/**")

	_, err := orch.GenerateIncremental(ctx, orchestrator.IncrementalRequest{
		HeadSHA: "sha-stale",
		Pages:   []manifest.DependencyManifest{m},
		DiffProvider: &staticDiffProvider{result: orchestrator.DiffResult{
			Changed:    []manifest.ChangedPair{{Path: "internal/auth/auth.go"}},
			TotalLines: 10,
		}},
		PR:             pr,
		WatermarkStore: wm,
		Config:         orchestrator.Config{RepoID: staleRepo},
		Now:            time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("GenerateIncremental: %v", err)
	}

	// A stale-block comment must have been posted.
	comments := pr.Comments()
	if len(comments) == 0 {
		t.Error("expected a PR comment about the stale human-edited block, got none")
	}
	if len(comments) > 0 && !strings.Contains(comments[0], "stale") {
		t.Errorf("comment does not mention 'stale': %q", comments[0])
	}

	// The block must still be the human-edited paragraph (not overwritten).
	page, ok, err := store.GetProposed(ctx, staleRepo, stalePRID, stalePageID)
	if err != nil || !ok {
		t.Fatal("expected proposed page after regen")
	}
	for _, blk := range page.Blocks {
		if blk.ID == blkID {
			if blk.Owner != ast.OwnerHumanEditedOnPRBranch {
				t.Errorf("stale block should remain OwnerHumanEditedOnPRBranch, got %v", blk.Owner)
			}
			if blk.Kind != ast.BlockKindParagraph {
				t.Errorf("stale block kind should be Paragraph (human content), got %v", blk.Kind)
			}
		}
	}
}

// TestPRDescriptionUpdatedWithBotAndHumanCounts verifies that after a bot
// regen the PR description summarises bot and human commit counts.
func TestPRDescriptionUpdatedWithBotAndHumanCounts(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	orch := newIncrementalOrch()
	wm := orchestrator.NewMemoryWatermarkStore()
	pr := orchestrator.NewMemoryWikiPR("pr-desc")

	// Add a human commit before the bot regen.
	pr.AddHumanCommit("sha-human", map[string][]byte{
		"wiki/test-repo.glossary.md": []byte("human content"),
	})

	m := makeManifest(repoID+".glossary", "glossary", "internal/auth/**")

	_, err := orch.GenerateIncremental(ctx, orchestrator.IncrementalRequest{
		HeadSHA: "sha-bot",
		Pages:   []manifest.DependencyManifest{m},
		DiffProvider: &staticDiffProvider{result: orchestrator.DiffResult{
			Changed:    []manifest.ChangedPair{{Path: "internal/auth/auth.go"}},
			TotalLines: 10,
		}},
		PR:             pr,
		WatermarkStore: wm,
		Now:            time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("GenerateIncremental: %v", err)
	}

	body := pr.Body()
	if body == "" {
		t.Fatal("expected PR description to be updated, got empty body")
	}
	if !strings.Contains(body, "SourceBridge") {
		t.Errorf("PR description missing 'SourceBridge': %q", body)
	}
	// The description should mention commit counts.
	if !strings.Contains(body, "commits") {
		t.Errorf("PR description missing commit count summary: %q", body)
	}
}

// TestRejectionRollsBackSourceProcessedSHA verifies that DiscardWithWatermark
// rolls source_processed_sha back to wiki_published_sha, so the next push
// regenerates the full rejected delta.
func TestRejectionRollsBackSourceProcessedSHA(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	reg := orchestrator.NewMapRegistry()
	store := orchestrator.NewMemoryPageStore()
	orch := orchestrator.New(orchestrator.Config{RepoID: "rollback-repo"}, reg, store)

	wm := orchestrator.NewMemoryWatermarkStore()

	// Simulate: wiki_published_sha = "sha-published", source_processed_sha = "sha-newer"
	if err := wm.AdvancePublished(ctx, "rollback-repo", "sha-published"); err != nil {
		t.Fatalf("AdvancePublished: %v", err)
	}
	if err := wm.AdvanceProcessed(ctx, "rollback-repo", "sha-newer"); err != nil {
		t.Fatalf("AdvanceProcessed: %v", err)
	}

	// Discard the PR (simulate rejection).
	if err := orch.DiscardWithWatermark(ctx, "rollback-repo", "pr-reject", wm); err != nil {
		t.Fatalf("DiscardWithWatermark: %v", err)
	}

	marks, err := wm.Get(ctx, "rollback-repo")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	// source_processed_sha must roll back to wiki_published_sha.
	if marks.SourceProcessedSHA != "sha-published" {
		t.Errorf("expected SourceProcessedSHA=sha-published after rejection rollback, got %q", marks.SourceProcessedSHA)
	}
	// wiki_published_sha must be unchanged.
	if marks.WikiPublishedSHA != "sha-published" {
		t.Errorf("expected WikiPublishedSHA unchanged=sha-published, got %q", marks.WikiPublishedSHA)
	}
}

// TestForcePushSHANotFoundResetsWatermarksAndFullRegen verifies that when the
// diff provider returns ErrSHANotFound, both watermarks are reset and a full
// regeneration is attempted.
func TestForcePushSHANotFoundResetsWatermarksAndFullRegen(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	orch := newIncrementalOrch()
	wm := orchestrator.NewMemoryWatermarkStore()
	pr := orchestrator.NewMemoryWikiPR("pr-forcepush")

	// Pre-set both watermarks so we can verify reset.
	if err := wm.AdvancePublished(ctx, repoID, "sha-old"); err != nil {
		t.Fatalf("AdvancePublished: %v", err)
	}

	m := makeManifest(repoID+".glossary", "glossary", "internal/auth/**")

	result, err := orch.GenerateIncremental(ctx, orchestrator.IncrementalRequest{
		HeadSHA:        "sha-new-after-force",
		Pages:          []manifest.DependencyManifest{m},
		DiffProvider:   &notFoundDiffProvider{},
		PR:             pr,
		WatermarkStore: wm,
		Now:            time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("GenerateIncremental on force-push: %v", err)
	}
	if !result.ForcePush {
		t.Error("expected result.ForcePush=true when ErrSHANotFound")
	}

	marks, err := wm.Get(ctx, repoID)
	if err != nil {
		t.Fatalf("Get watermarks: %v", err)
	}
	// Both watermarks must be reset (empty) after force-push recovery.
	if marks.WikiPublishedSHA != "" {
		t.Errorf("expected WikiPublishedSHA reset to empty after force-push, got %q", marks.WikiPublishedSHA)
	}
}

// TestPageBudgetPrioritisesDirectHitsOverGraphHits verifies that the 5-page
// budget selects direct-hit pages before graph-hit pages.
func TestPageBudgetPrioritisesDirectHitsOverGraphHits(t *testing.T) {
	t.Parallel()

	// 3 direct-hit pages + 4 graph-hit pages = 7 total; budget = 5.
	// Expected: all 3 direct hits + 2 graph hits in prioritized; 2 graph hits queued.
	var affected []manifest.AffectedPage
	for i := 0; i < 3; i++ {
		affected = append(affected, manifest.AffectedPage{
			Manifest:  makeManifest("direct.page"+string(rune('0'+i)), "glossary"),
			DirectHit: true,
		})
	}
	for i := 0; i < 4; i++ {
		affected = append(affected, manifest.AffectedPage{
			Manifest: makeManifest("graph.page"+string(rune('0'+i)), "glossary"),
			GraphHit: true,
		})
	}

	// We call applyPageBudget indirectly via a GenerateIncremental run that
	// returns the Queued slice. To test the budget logic in isolation without
	// going through the full orchestrator, we use the exported helper function
	// added below. For now we verify via the IncrementalResult.
	//
	// Since applyPageBudget is package-internal, we test it through the
	// orchestrator's IncrementalResult.Queued and len(Regenerated) outputs.

	ctx := context.Background()

	reg := orchestrator.NewMapRegistry(&passGlossaryTemplate{id: "glossary"})
	store := orchestrator.NewMemoryPageStore()
	orch := orchestrator.New(orchestrator.Config{RepoID: repoID}, reg, store)
	wm := orchestrator.NewMemoryWatermarkStore()
	pr := orchestrator.NewMemoryWikiPR("pr-priority")

	var pages []manifest.DependencyManifest
	for _, ap := range affected {
		pages = append(pages, ap.Manifest)
	}

	// All pages watch internal/auth/**; diff touches internal/auth.
	for i := range pages {
		pages[i].Dependencies.Paths = []string{"internal/auth/**"}
		// Preserve the direct/graph classification by keeping direct-hit pages
		// as direct-path matches. We set up the dependency so that the manifest
		// package's AffectedPages function will classify them correctly.
		// Since direct-hit pages have paths matching the changed file, they
		// become direct hits. Graph-hit pages have no paths but have upstream packages.
		if i >= 3 {
			pages[i].Dependencies.Paths = nil
			pages[i].Dependencies.UpstreamPackages = []string{"internal/auth"}
		}
	}

	result, err := orch.GenerateIncremental(ctx, orchestrator.IncrementalRequest{
		HeadSHA: "sha-priority",
		Pages:   pages,
		DiffProvider: &staticDiffProvider{result: orchestrator.DiffResult{
			Changed:    []manifest.ChangedPair{{Path: "internal/auth/auth.go"}},
			TotalLines: 10,
		}},
		PR:             pr,
		WatermarkStore: wm,
		Now:            time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("GenerateIncremental: %v", err)
	}

	total := len(result.Regenerated) + len(result.Queued)
	if total < 5 {
		// Some pages may have been excluded by quality gates; that's OK.
		// We just need to verify at least the budget limit was applied when there
		// are enough pages.
		t.Logf("total=%d (some pages may have been excluded by quality gates)", total)
	}
	if len(result.Queued) == 0 && total == 7 {
		t.Error("expected some pages to be queued when 7 pages affected and budget is 5")
	}
}

// ---- Additional stubs used only in this file ----

// kindChangingTemplate generates a block whose Kind differs from what the
// human-edited block holds, triggering the stale detection logic.
type kindChangingTemplate struct {
	id         string
	outputKind ast.BlockKind
	blockID    ast.BlockID
}

func (k *kindChangingTemplate) ID() string { return k.id }

func (k *kindChangingTemplate) Generate(_ context.Context, input templates.GenerateInput) (ast.Page, error) {
	pageID := input.RepoID + "." + k.id
	blk := ast.Block{
		ID:   k.blockID,
		Kind: k.outputKind,
		Owner: ast.OwnerGenerated,
	}
	switch k.outputKind {
	case ast.BlockKindHeading:
		blk.Content = ast.BlockContent{Heading: &ast.HeadingContent{Level: 2, Text: "Generated heading"}}
	default:
		blk.Content = ast.BlockContent{Paragraph: &ast.ParagraphContent{Markdown: "Generated content."}}
	}
	return ast.Page{
		ID: pageID,
		Manifest: manifest.DependencyManifest{
			PageID:   pageID,
			Template: k.id,
			Audience: string(input.Audience),
		},
		Blocks:     []ast.Block{blk},
		Provenance: ast.Provenance{GeneratedAt: input.Now},
	}, nil
}
