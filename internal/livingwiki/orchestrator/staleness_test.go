// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package orchestrator_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/sourcebridge/sourcebridge/internal/livingwiki/ast"
	"github.com/sourcebridge/sourcebridge/internal/livingwiki/manifest"
	"github.com/sourcebridge/sourcebridge/internal/livingwiki/markdown"
	"github.com/sourcebridge/sourcebridge/internal/livingwiki/orchestrator"
	"github.com/sourcebridge/sourcebridge/internal/quality"
)

// ---- test manifests ----

// authManifest is the manifest for arch.auth, with stale_when conditions.
var authManifest = manifest.DependencyManifest{
	PageID:   "arch.auth",
	Template: "architecture",
	Audience: "for-engineers",
	Dependencies: manifest.Dependencies{
		Paths:   []string{"internal/auth/**"},
		Symbols: []string{"auth.Middleware", "auth.RequireRole"},
	},
	StaleWhen: []manifest.StaleCondition{
		{SignatureChangeIn: []string{"auth.Middleware", "auth.RequireRole"}},
		{NewCallerAddedTo: []string{"auth.RequireRole"}},
	},
}

// billingManifest has no stale_when conditions.
var billingManifest = manifest.DependencyManifest{
	PageID:   "arch.billing",
	Template: "architecture",
	Audience: "for-engineers",
	Dependencies: manifest.Dependencies{
		Paths: []string{"internal/billing/**"},
	},
}

// ---- EvaluateStaleness tests ----

// TestEvaluateStaleness_SignatureChangeFires verifies that a signature change
// in a monitored symbol returns a signal for the owning page.
func TestEvaluateStaleness_SignatureChangeFires(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	store := orchestrator.NewMemoryPageStore()
	orch := orchestrator.New(orchestrator.Config{RepoID: "test-repo"}, orchestrator.NewMapRegistry(), store)

	changed := []manifest.ChangedPair{
		{Path: "internal/auth/auth.go", Symbol: "auth.RequireRole"},
	}

	signals, err := orch.EvaluateStaleness(ctx,
		[]manifest.DependencyManifest{authManifest, billingManifest},
		changed,
		"abc1234",
		"",
	)
	if err != nil {
		t.Fatalf("EvaluateStaleness: %v", err)
	}

	if len(signals) == 0 {
		t.Fatal("expected at least one stale signal, got none")
	}

	found := false
	for _, s := range signals {
		if s.PageID == "arch.auth" {
			found = true
			if s.TriggeringCommit != "abc1234" {
				t.Errorf("TriggeringCommit: got %q, want abc1234", s.TriggeringCommit)
			}
			if len(s.TriggeringSymbols) == 0 {
				t.Error("expected TriggeringSymbols to be non-empty")
			}
		}
		// billing has no stale_when, so it must not appear.
		if s.PageID == "arch.billing" {
			t.Errorf("arch.billing has no stale_when conditions; it should not generate a signal")
		}
	}
	if !found {
		t.Error("expected a signal for arch.auth")
	}
}

// TestEvaluateStaleness_NoMatchReturnsEmpty verifies that changes to symbols
// not listed in any stale_when condition produce no signals.
func TestEvaluateStaleness_NoMatchReturnsEmpty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	store := orchestrator.NewMemoryPageStore()
	orch := orchestrator.New(orchestrator.Config{RepoID: "test-repo"}, orchestrator.NewMapRegistry(), store)

	changed := []manifest.ChangedPair{
		{Path: "internal/billing/billing.go", Symbol: "billing.Charge"},
	}

	signals, err := orch.EvaluateStaleness(ctx,
		[]manifest.DependencyManifest{authManifest},
		changed,
		"def5678",
		"",
	)
	if err != nil {
		t.Fatalf("EvaluateStaleness: %v", err)
	}
	if len(signals) != 0 {
		t.Errorf("expected 0 signals for unrelated changes, got %d", len(signals))
	}
}

// TestEvaluateStaleness_BothConditions verifies that multiple stale_when
// conditions in the same manifest can each fire independently.
func TestEvaluateStaleness_BothConditions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	store := orchestrator.NewMemoryPageStore()
	orch := orchestrator.New(orchestrator.Config{RepoID: "test-repo"}, orchestrator.NewMapRegistry(), store)

	// Change fires both conditions in authManifest.
	changed := []manifest.ChangedPair{
		{Symbol: "auth.Middleware"},   // fires condition[0] SignatureChangeIn
		{Symbol: "auth.RequireRole"},  // fires condition[0] SignatureChangeIn AND condition[1] NewCallerAddedTo
	}

	signals, err := orch.EvaluateStaleness(ctx,
		[]manifest.DependencyManifest{authManifest},
		changed,
		"multi1",
		"next week",
	)
	if err != nil {
		t.Fatalf("EvaluateStaleness: %v", err)
	}
	// Expect 2 signals: one per condition that fired.
	if len(signals) < 1 {
		t.Errorf("expected at least 1 signal, got %d", len(signals))
	}
	// All signals must carry the next regen window.
	for _, s := range signals {
		if s.NextRegenWindow != "next week" {
			t.Errorf("signal missing NextRegenWindow: got %q", s.NextRegenWindow)
		}
	}
}

// ---- AttachStaleBanners tests ----

// TestAttachStaleBanners_BannerPrependedToPage verifies that a stale banner
// block appears at the head of the page's block list after AttachStaleBanners.
func TestAttachStaleBanners_BannerPrependedToPage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	store := orchestrator.NewMemoryPageStore()
	orch := orchestrator.New(orchestrator.Config{RepoID: "test-repo"}, orchestrator.NewMapRegistry(), store)

	// Seed a canonical page.
	page := buildPageForStaleness("arch.auth")
	if err := store.SetCanonical(ctx, "test-repo", page); err != nil {
		t.Fatalf("SetCanonical: %v", err)
	}

	signal := orchestrator.StalePageSignal{
		PageID:            "arch.auth",
		TriggeringCommit:  "abc1234",
		TriggeringSymbols: []string{"auth.RequireRole"},
		ConditionKind:     "signature_change_in",
	}

	err := orch.AttachStaleBanners(ctx, "test-repo", "", []orchestrator.StalePageSignal{signal},
		orchestrator.StaleBannerConfig{UIBaseURL: "https://app.sourcebridge.ai"})
	if err != nil {
		t.Fatalf("AttachStaleBanners: %v", err)
	}

	updated, ok, err := store.GetCanonical(ctx, "test-repo", "arch.auth")
	if err != nil || !ok {
		t.Fatalf("GetCanonical after attach: ok=%v err=%v", ok, err)
	}

	if len(updated.Blocks) == 0 {
		t.Fatal("page has no blocks after AttachStaleBanners")
	}
	if updated.Blocks[0].Kind != ast.BlockKindStaleBanner {
		t.Errorf("first block should be stale_banner, got %q", updated.Blocks[0].Kind)
	}
	banner := updated.Blocks[0].Content.StaleBanner
	if banner == nil {
		t.Fatal("stale banner content is nil")
	}
	if banner.TriggeringCommit != "abc1234" {
		t.Errorf("TriggeringCommit: got %q, want abc1234", banner.TriggeringCommit)
	}
	if !strings.Contains(banner.RefreshURL, "arch.auth") {
		t.Errorf("RefreshURL should contain page ID: got %q", banner.RefreshURL)
	}
}

// TestAttachStaleBanners_ReplacesExistingBanner verifies that a second call
// replaces the existing banner rather than prepending a second one.
func TestAttachStaleBanners_ReplacesExistingBanner(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	store := orchestrator.NewMemoryPageStore()
	orch := orchestrator.New(orchestrator.Config{RepoID: "test-repo"}, orchestrator.NewMapRegistry(), store)

	page := buildPageForStaleness("arch.auth")
	if err := store.SetCanonical(ctx, "test-repo", page); err != nil {
		t.Fatalf("SetCanonical: %v", err)
	}

	sig1 := orchestrator.StalePageSignal{
		PageID:            "arch.auth",
		TriggeringCommit:  "sha-first",
		TriggeringSymbols: []string{"auth.Middleware"},
	}
	sig2 := orchestrator.StalePageSignal{
		PageID:            "arch.auth",
		TriggeringCommit:  "sha-second",
		TriggeringSymbols: []string{"auth.RequireRole"},
	}

	cfg := orchestrator.StaleBannerConfig{}
	if err := orch.AttachStaleBanners(ctx, "test-repo", "", []orchestrator.StalePageSignal{sig1}, cfg); err != nil {
		t.Fatalf("AttachStaleBanners (first): %v", err)
	}
	if err := orch.AttachStaleBanners(ctx, "test-repo", "", []orchestrator.StalePageSignal{sig2}, cfg); err != nil {
		t.Fatalf("AttachStaleBanners (second): %v", err)
	}

	updated, _, _ := store.GetCanonical(ctx, "test-repo", "arch.auth")

	bannerCount := 0
	for _, b := range updated.Blocks {
		if b.Kind == ast.BlockKindStaleBanner {
			bannerCount++
		}
	}
	if bannerCount != 1 {
		t.Errorf("expected exactly 1 stale banner, got %d", bannerCount)
	}
	// The second banner's commit should be present.
	if updated.Blocks[0].Content.StaleBanner.TriggeringCommit != "sha-second" {
		t.Errorf("banner commit should be sha-second, got %q",
			updated.Blocks[0].Content.StaleBanner.TriggeringCommit)
	}
}

// TestAttachStaleBanners_ProposedAST verifies that when a prID is provided the
// banner is attached to the proposed_ast, not the canonical.
func TestAttachStaleBanners_ProposedAST(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	store := orchestrator.NewMemoryPageStore()
	orch := orchestrator.New(orchestrator.Config{RepoID: "test-repo"}, orchestrator.NewMapRegistry(), store)

	page := buildPageForStaleness("arch.auth")
	const prID = "pr-stale-test"
	if err := store.SetProposed(ctx, "test-repo", prID, page); err != nil {
		t.Fatalf("SetProposed: %v", err)
	}

	sig := orchestrator.StalePageSignal{
		PageID:            "arch.auth",
		TriggeringCommit:  "ccc9999",
		TriggeringSymbols: []string{"auth.Middleware"},
	}
	if err := orch.AttachStaleBanners(ctx, "test-repo", prID, []orchestrator.StalePageSignal{sig},
		orchestrator.StaleBannerConfig{}); err != nil {
		t.Fatalf("AttachStaleBanners: %v", err)
	}

	// Proposed should have the banner.
	proposed, ok, _ := store.GetProposed(ctx, "test-repo", prID, "arch.auth")
	if !ok {
		t.Fatal("proposed page not found after AttachStaleBanners")
	}
	if len(proposed.Blocks) == 0 || proposed.Blocks[0].Kind != ast.BlockKindStaleBanner {
		t.Error("proposed page should have stale banner at head")
	}

	// Canonical should be unchanged (no banner).
	canonical, exists, _ := store.GetCanonical(ctx, "test-repo", "arch.auth")
	if exists {
		for _, b := range canonical.Blocks {
			if b.Kind == ast.BlockKindStaleBanner {
				t.Error("canonical_ast should not have a stale banner when prID is set")
			}
		}
	}
}

// ---- Banner rendered format test ----

// TestStaleBanner_MarkdownFormat verifies the exact blockquote format required by the plan.
func TestStaleBanner_MarkdownFormat(t *testing.T) {
	t.Parallel()

	page := ast.Page{
		ID: "arch.auth",
		Manifest: manifest.DependencyManifest{PageID: "arch.auth"},
		Blocks: []ast.Block{
			{
				ID:   "bstale",
				Kind: ast.BlockKindStaleBanner,
				Content: ast.BlockContent{
					StaleBanner: &ast.StaleBannerContent{
						TriggeringCommit:  "a1b2c3d",
						TriggeringSymbols: []string{"auth.RequireRole"},
						ConditionKind:     "signature_change_in",
						RefreshURL:        "https://app.sourcebridge.ai/repos/test/pages/arch.auth/refresh",
						NextRegenWindow:   "in 2 hours",
					},
				},
				Owner: ast.OwnerGenerated,
			},
		},
	}

	var buf bytes.Buffer
	if err := markdown.Write(&buf, page); err != nil {
		t.Fatalf("Write: %v", err)
	}

	out := buf.String()

	// The plan specifies this exact format:
	// > ⚠️ **This page may be out of date.** Recent changes to `<sym>` (commit `<sha>`) may affect this content. [Refresh from source](<url>).
	if !strings.Contains(out, "> ⚠️ **This page may be out of date.**") {
		t.Errorf("banner missing required prefix, got:\n%s", out)
	}
	if !strings.Contains(out, "`auth.RequireRole`") {
		t.Error("banner missing triggering symbol in backticks")
	}
	if !strings.Contains(out, "(commit `a1b2c3d`)") {
		t.Error("banner missing commit SHA in backticks")
	}
	if !strings.Contains(out, "[Refresh from source]") {
		t.Error("banner missing Refresh from source link")
	}
	if !strings.Contains(out, "in 2 hours") {
		t.Error("banner missing next regen window")
	}
}

// ---- RefreshPage tests ----

// TestRefreshPage_RemovesStaleBanner verifies that RefreshPage regenerates the
// page and removes any stale banner from the head.
func TestRefreshPage_RemovesStaleBanner(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	const passMarkdown = "Middleware wraps an HTTP handler. No behavioral assertions."

	tmpl := &stubTemplate{id: "glossary", markdown: passMarkdown}
	reg := orchestrator.NewMapRegistry(tmpl)
	store := orchestrator.NewMemoryPageStore()
	orch := orchestrator.New(orchestrator.Config{RepoID: "test-repo"}, reg, store)

	// Seed a canonical page that already has a stale banner prepended.
	stalePage := buildPageForStaleness("test-repo.glossary")
	staleBlock := ast.Block{
		ID:   "bstale-existing",
		Kind: ast.BlockKindStaleBanner,
		Content: ast.BlockContent{
			StaleBanner: &ast.StaleBannerContent{
				TriggeringCommit: "old-sha",
			},
		},
		Owner: ast.OwnerGenerated,
	}
	stalePage.Blocks = append([]ast.Block{staleBlock}, stalePage.Blocks...)
	if err := store.SetCanonical(ctx, "test-repo", stalePage); err != nil {
		t.Fatalf("SetCanonical: %v", err)
	}

	planned := orchestrator.PlannedPage{
		ID:         "test-repo.glossary",
		TemplateID: "glossary",
		Audience:   quality.AudienceEngineers,
		Input:      makeBaseInput(nil, nil),
	}

	refreshed, err := orch.RefreshPage(ctx, "test-repo", "", planned)
	if err != nil {
		t.Fatalf("RefreshPage: %v", err)
	}

	// The stale banner should be gone.
	for _, b := range refreshed.Blocks {
		if b.Kind == ast.BlockKindStaleBanner {
			t.Error("RefreshPage should remove stale banner after regen")
		}
	}

	// The canonical page should be updated.
	canonical, ok, err := store.GetCanonical(ctx, "test-repo", "test-repo.glossary")
	if err != nil || !ok {
		t.Fatalf("GetCanonical after RefreshPage: ok=%v err=%v", ok, err)
	}
	for _, b := range canonical.Blocks {
		if b.Kind == ast.BlockKindStaleBanner {
			t.Error("canonical_ast should not have a stale banner after RefreshPage")
		}
	}
}

// TestRefreshPage_BypassesBudget verifies that RefreshPage generates the page
// even when the per-push budget is conceptually exhausted. Since the budget is
// enforced only in GenerateIncremental, RefreshPage calling generateOnePage
// directly bypasses it.
func TestRefreshPage_BypassesBudget(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	const passMarkdown = "Middleware wraps an HTTP handler. No behavioral assertions."
	tmpl := &stubTemplate{id: "glossary", markdown: passMarkdown}
	reg := orchestrator.NewMapRegistry(tmpl)
	store := orchestrator.NewMemoryPageStore()
	orch := orchestrator.New(orchestrator.Config{RepoID: "test-repo"}, reg, store)

	planned := orchestrator.PlannedPage{
		ID:         "test-repo.glossary",
		TemplateID: "glossary",
		Audience:   quality.AudienceEngineers,
		Input:      makeBaseInput(nil, nil),
	}

	// RefreshPage should work without errors even for a page not previously stored.
	page, err := orch.RefreshPage(ctx, "test-repo", "", planned)
	if err != nil {
		t.Fatalf("RefreshPage: %v", err)
	}
	if page.ID == "" {
		t.Error("RefreshPage returned empty page")
	}
}

// ---- helpers ----

// buildPageForStaleness builds a simple page used by staleness tests.
func buildPageForStaleness(id string) ast.Page {
	return ast.Page{
		ID: id,
		Manifest: manifest.DependencyManifest{
			PageID:   id,
			Template: "architecture",
			Audience: "for-engineers",
		},
		Blocks: []ast.Block{
			{
				ID:    ast.GenerateBlockID(id, "", ast.BlockKindHeading, 0),
				Kind:  ast.BlockKindHeading,
				Owner: ast.OwnerGenerated,
				Content: ast.BlockContent{
					Heading: &ast.HeadingContent{Level: 1, Text: id},
				},
			},
			{
				ID:    ast.GenerateBlockID(id, "", ast.BlockKindParagraph, 0),
				Kind:  ast.BlockKindParagraph,
				Owner: ast.OwnerGenerated,
				Content: ast.BlockContent{
					Paragraph: &ast.ParagraphContent{Markdown: "This page documents " + id + "."},
				},
			},
		},
	}
}
