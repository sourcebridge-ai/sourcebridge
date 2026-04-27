// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package livingwiki_test

import (
	"context"
	"testing"
	"time"

	"github.com/sourcebridge/sourcebridge/internal/settings/livingwiki"
)

const testTenant = "default"

func TestRepoSettingsMemStore_GetNilForUnconfigured(t *testing.T) {
	store := livingwiki.NewRepoSettingsMemStore()
	got, err := store.GetRepoSettings(context.Background(), testTenant, "repo-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for unconfigured repo, got %+v", got)
	}
}

func TestRepoSettingsMemStore_RoundTrip(t *testing.T) {
	store := livingwiki.NewRepoSettingsMemStore()

	want := livingwiki.RepositoryLivingWikiSettings{
		TenantID:          testTenant,
		RepoID:            "repo-a",
		Enabled:           true,
		Mode:              livingwiki.RepoWikiModePRReview,
		StaleWhenStrategy: livingwiki.StaleStrategyDirect,
		MaxPagesPerJob:    50,
		Sinks: []livingwiki.RepoWikiSink{
			{
				Kind:            livingwiki.RepoWikiSinkConfluence,
				IntegrationName: "test",
				Audience:        livingwiki.RepoWikiAudienceEngineer,
			},
		},
		UpdatedAt: time.Now().UTC().Truncate(time.Second),
	}

	if err := store.SetRepoSettings(context.Background(), want); err != nil {
		t.Fatalf("SetRepoSettings: %v", err)
	}

	got, err := store.GetRepoSettings(context.Background(), testTenant, "repo-a")
	if err != nil {
		t.Fatalf("GetRepoSettings: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil settings after Set")
	}
	if got.Enabled != want.Enabled {
		t.Errorf("Enabled: got %v, want %v", got.Enabled, want.Enabled)
	}
	if got.Mode != want.Mode {
		t.Errorf("Mode: got %q, want %q", got.Mode, want.Mode)
	}
	if len(got.Sinks) != 1 {
		t.Errorf("Sinks length: got %d, want 1", len(got.Sinks))
	} else if got.Sinks[0].IntegrationName != "test" {
		t.Errorf("Sinks[0].IntegrationName: got %q, want %q", got.Sinks[0].IntegrationName, "test")
	}
}

func TestRepoSettingsMemStore_DisablePersistsSinks(t *testing.T) {
	store := livingwiki.NewRepoSettingsMemStore()

	original := livingwiki.RepositoryLivingWikiSettings{
		TenantID: testTenant,
		RepoID:   "repo-b",
		Enabled:  true,
		Mode:     livingwiki.RepoWikiModeDirectPublish,
		Sinks: []livingwiki.RepoWikiSink{
			{
				Kind:            livingwiki.RepoWikiSinkGitRepo,
				IntegrationName: "my-git",
				Audience:        livingwiki.RepoWikiAudienceProduct,
			},
		},
	}
	if err := store.SetRepoSettings(context.Background(), original); err != nil {
		t.Fatalf("SetRepoSettings: %v", err)
	}

	// Soft-disable: set enabled=false but leave sinks intact.
	now := time.Now()
	disabled := original
	disabled.Enabled = false
	disabled.DisabledAt = &now
	if err := store.SetRepoSettings(context.Background(), disabled); err != nil {
		t.Fatalf("SetRepoSettings disable: %v", err)
	}

	got, err := store.GetRepoSettings(context.Background(), testTenant, "repo-b")
	if err != nil {
		t.Fatalf("GetRepoSettings: %v", err)
	}
	if got.Enabled {
		t.Error("expected Enabled=false after disable")
	}
	if got.DisabledAt == nil {
		t.Error("expected DisabledAt to be set")
	}
	if len(got.Sinks) != 1 || got.Sinks[0].IntegrationName != "my-git" {
		t.Errorf("sinks were cleared on disable, got %+v", got.Sinks)
	}
	if got.Mode != livingwiki.RepoWikiModeDirectPublish {
		t.Errorf("mode was changed on disable, got %q", got.Mode)
	}
}

func TestRepoSettingsMemStore_ListEnabledRepos(t *testing.T) {
	store := livingwiki.NewRepoSettingsMemStore()

	enabled := livingwiki.RepositoryLivingWikiSettings{
		TenantID: testTenant,
		RepoID:   "repo-enabled",
		Enabled:  true,
	}
	disabled := livingwiki.RepositoryLivingWikiSettings{
		TenantID: testTenant,
		RepoID:   "repo-disabled",
		Enabled:  false,
	}
	otherTenant := livingwiki.RepositoryLivingWikiSettings{
		TenantID: "other-tenant",
		RepoID:   "repo-other",
		Enabled:  true,
	}

	for _, s := range []livingwiki.RepositoryLivingWikiSettings{enabled, disabled, otherTenant} {
		if err := store.SetRepoSettings(context.Background(), s); err != nil {
			t.Fatalf("SetRepoSettings: %v", err)
		}
	}

	list, err := store.ListEnabledRepos(context.Background(), testTenant)
	if err != nil {
		t.Fatalf("ListEnabledRepos: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 enabled repo for %q, got %d", testTenant, len(list))
	}
	if len(list) > 0 && list[0].RepoID != "repo-enabled" {
		t.Errorf("expected repo-enabled, got %q", list[0].RepoID)
	}
}

func TestRepoSettingsMemStore_RepositoriesUsingSink(t *testing.T) {
	store := livingwiki.NewRepoSettingsMemStore()

	repoA := livingwiki.RepositoryLivingWikiSettings{
		TenantID: testTenant,
		RepoID:   "repo-a",
		Sinks: []livingwiki.RepoWikiSink{
			{Kind: livingwiki.RepoWikiSinkConfluence, IntegrationName: "target-sink"},
		},
	}
	repoB := livingwiki.RepositoryLivingWikiSettings{
		TenantID: testTenant,
		RepoID:   "repo-b",
		Sinks: []livingwiki.RepoWikiSink{
			{Kind: livingwiki.RepoWikiSinkNotion, IntegrationName: "other-sink"},
		},
	}

	for _, s := range []livingwiki.RepositoryLivingWikiSettings{repoA, repoB} {
		if err := store.SetRepoSettings(context.Background(), s); err != nil {
			t.Fatalf("SetRepoSettings: %v", err)
		}
	}

	results, err := store.RepositoriesUsingSink(context.Background(), testTenant, "target-sink")
	if err != nil {
		t.Fatalf("RepositoriesUsingSink: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for target-sink, got %d", len(results))
	}
	if len(results) > 0 && results[0].RepoID != "repo-a" {
		t.Errorf("expected repo-a, got %q", results[0].RepoID)
	}

	// Query for unknown sink returns empty.
	empty, err := store.RepositoriesUsingSink(context.Background(), testTenant, "no-such-sink")
	if err != nil {
		t.Fatalf("RepositoriesUsingSink (empty): %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0 results for no-such-sink, got %d", len(empty))
	}
}

// TestDefaultRepoEditPolicy covers the canonical defaultEditPolicy table from
// the plan. The two most important entries are:
//   - git_repo → PROPOSE_PR   (proposal-first, native PR concept)
//   - backstage_techdocs → DIRECT_PUBLISH  (no PR concept)
func TestDefaultRepoEditPolicy(t *testing.T) {
	cases := []struct {
		kind RepoWikiSinkKind
		want RepoWikiEditPolicy
	}{
		{livingwiki.RepoWikiSinkGitRepo, livingwiki.RepoWikiEditPolicyProposePR},
		{livingwiki.RepoWikiSinkConfluence, livingwiki.RepoWikiEditPolicyProposePR},
		{livingwiki.RepoWikiSinkNotion, livingwiki.RepoWikiEditPolicyProposePR},
		{livingwiki.RepoWikiSinkGitHubWiki, livingwiki.RepoWikiEditPolicyProposePR},
		{livingwiki.RepoWikiSinkGitLabWiki, livingwiki.RepoWikiEditPolicyProposePR},
		{livingwiki.RepoWikiSinkBackstageTechDocs, livingwiki.RepoWikiEditPolicyDirectPublish},
		{livingwiki.RepoWikiSinkMkDocs, livingwiki.RepoWikiEditPolicyDirectPublish},
		{livingwiki.RepoWikiSinkDocusaurus, livingwiki.RepoWikiEditPolicyDirectPublish},
		{livingwiki.RepoWikiSinkVitePress, livingwiki.RepoWikiEditPolicyDirectPublish},
	}
	for _, tc := range cases {
		got, ok := livingwiki.DefaultRepoEditPolicy[tc.kind]
		if !ok {
			t.Errorf("DefaultRepoEditPolicy missing entry for %q", tc.kind)
			continue
		}
		if got != tc.want {
			t.Errorf("DefaultRepoEditPolicy[%q] = %q, want %q", tc.kind, got, tc.want)
		}
	}
}

// TestEffectiveEditPolicyFallback verifies that EffectiveEditPolicy falls back
// to the DefaultRepoEditPolicy table when EditPolicy is not explicitly set,
// and honours an explicit override when set.
func TestEffectiveEditPolicyFallback(t *testing.T) {
	// No explicit policy — should fall back to map.
	sink := livingwiki.RepoWikiSink{
		Kind: livingwiki.RepoWikiSinkGitRepo,
	}
	if got := sink.EffectiveEditPolicy(); got != livingwiki.RepoWikiEditPolicyProposePR {
		t.Errorf("fallback for GIT_REPO: got %q, want PROPOSE_PR", got)
	}

	// Explicit override — should be honoured.
	sinkOverride := livingwiki.RepoWikiSink{
		Kind:       livingwiki.RepoWikiSinkGitRepo,
		EditPolicy: livingwiki.RepoWikiEditPolicyDirectPublish,
	}
	if got := sinkOverride.EffectiveEditPolicy(); got != livingwiki.RepoWikiEditPolicyDirectPublish {
		t.Errorf("override for GIT_REPO: got %q, want DIRECT_PUBLISH", got)
	}

	// backstage_techdocs without override → DIRECT_PUBLISH
	sinkBST := livingwiki.RepoWikiSink{
		Kind: livingwiki.RepoWikiSinkBackstageTechDocs,
	}
	if got := sinkBST.EffectiveEditPolicy(); got != livingwiki.RepoWikiEditPolicyDirectPublish {
		t.Errorf("fallback for BACKSTAGE_TECHDOCS: got %q, want DIRECT_PUBLISH", got)
	}
}

// Use livingwiki and time to avoid "declared and not used" errors on import.
type RepoWikiSinkKind = livingwiki.RepoWikiSinkKind
type RepoWikiEditPolicy = livingwiki.RepoWikiEditPolicy
