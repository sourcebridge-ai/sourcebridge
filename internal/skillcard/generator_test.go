// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package skillcard_test

import (
	"strings"
	"testing"
	"time"

	"github.com/sourcebridge/sourcebridge/internal/skillcard"
)

func TestRender_ThreeClusters(t *testing.T) {
	indexedAt := time.Date(2026, 4, 27, 9, 45, 0, 0, time.UTC)
	input := skillcard.RepoSummary{
		RepoName:  "payments-service",
		RepoID:    "abc123",
		ServerURL: "https://sb.example.com",
		IndexedAt: indexedAt,
		Clusters: []skillcard.ClusterSummary{
			{
				Label:              "auth",
				MemberCount:        14,
				Packages:           []string{"auth", "middleware", "session"},
				RepresentativeSyms: []string{"TokenStore.Rotate", "Session.Validate", "OAuthFlow.Begin"},
				Warnings: []skillcard.Warning{
					{
						Symbol: "TokenStore.Rotate",
						Kind:   "cross-package-callers",
						Detail: "TokenStore.Rotate() has callers in auth, api, and worker — coordinate changes across all of them.",
					},
					{
						Symbol: "Session.Validate",
						Kind:   "hot-path",
						Detail: "Session.Validate() is on the hot path (highest in-degree in cluster, 8 callers).",
					},
				},
			},
			{
				Label:       "billing",
				MemberCount: 9,
				Packages:    []string{"billing", "stripe"},
				Warnings: []skillcard.Warning{
					{
						Symbol: "InvoiceJob.Run",
						Kind:   "hot-path",
						Detail: "InvoiceJob.Run() is on the hot path (highest in-degree in cluster, 3 callers).",
					},
				},
			},
			{
				Label:       "storage",
				MemberCount: 11,
				Packages:    []string{"db"},
				Warnings:    nil,
			},
		},
	}

	out := skillcard.Render(input)

	// Must include start/end markers.
	if !strings.Contains(out, "<!-- sourcebridge:start -->") {
		t.Error("missing start marker")
	}
	if !strings.Contains(out, "<!-- sourcebridge:end -->") {
		t.Error("missing end marker")
	}

	// Header block.
	if !strings.Contains(out, "# SourceBridge — payments-service") {
		t.Error("missing repo name in header")
	}
	if !strings.Contains(out, "Repo ID: abc123") {
		t.Error("missing repo ID")
	}
	if !strings.Contains(out, "Indexed: 2026-04-27") {
		t.Error("missing indexed date")
	}
	if !strings.Contains(out, "Server: https://sb.example.com") {
		t.Error("missing server URL")
	}

	// Try this first prompt names the two largest clusters (auth and billing).
	if !strings.Contains(out, "auth") || !strings.Contains(out, "billing") {
		t.Error("Try this first prompt should name both the largest and second-largest clusters")
	}
	// The prompt should use the cross-cluster comparison form for 2+ clusters.
	if !strings.Contains(out, "Compare") {
		t.Error("Try this first prompt should use the Compare form for multiple clusters")
	}

	// Per-cluster sections.
	if !strings.Contains(out, "## Subsystem: auth") {
		t.Error("missing auth subsystem heading")
	}
	if !strings.Contains(out, "14 symbols") {
		t.Error("missing auth member count")
	}
	if !strings.Contains(out, "3 packages (auth, middleware, session)") {
		t.Error("missing auth packages")
	}
	if !strings.Contains(out, "Watch out: TokenStore.Rotate()") {
		t.Error("missing cross-package-callers warning")
	}
	if !strings.Contains(out, "Watch out: Session.Validate()") {
		t.Error("missing hot-path warning")
	}

	if !strings.Contains(out, "## Subsystem: billing") {
		t.Error("missing billing subsystem heading")
	}
	if !strings.Contains(out, "9 symbols") {
		t.Error("missing billing member count")
	}

	if !strings.Contains(out, "## Subsystem: storage") {
		t.Error("missing storage subsystem heading")
	}
	if !strings.Contains(out, "11 symbols") {
		t.Error("missing storage member count")
	}
}

func TestRender_EmptyClusters(t *testing.T) {
	input := skillcard.RepoSummary{
		RepoName:  "empty-repo",
		RepoID:    "xyz789",
		ServerURL: "http://localhost:8080",
		IndexedAt: time.Now(),
		Clusters:  nil,
	}
	out := skillcard.Render(input)

	if !strings.Contains(out, "# SourceBridge — empty-repo") {
		t.Error("missing header for empty cluster list")
	}
	// No Subsystem sections should appear.
	if strings.Contains(out, "## Subsystem:") {
		t.Error("should not emit subsystem sections for empty cluster list")
	}
	if !strings.Contains(out, "<!-- sourcebridge:end -->") {
		t.Error("missing end marker")
	}
}

func TestRender_SingleCluster(t *testing.T) {
	input := skillcard.RepoSummary{
		RepoName:  "tiny-service",
		RepoID:    "tiny1",
		ServerURL: "https://sb.example.com",
		IndexedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Clusters: []skillcard.ClusterSummary{
			{
				Label:       "core",
				MemberCount: 5,
				Packages:    []string{"core"},
			},
		},
	}
	out := skillcard.Render(input)
	if !strings.Contains(out, "## Subsystem: core") {
		t.Error("missing core section")
	}
	if !strings.Contains(out, "5 symbols") {
		t.Error("missing symbol count")
	}
	// Single package — singular form.
	if !strings.Contains(out, "1 package (core)") {
		t.Error("expected singular 'package'")
	}
}

func TestGenerate_SectionFields(t *testing.T) {
	input := skillcard.RepoSummary{
		RepoName: "test-repo",
		RepoID:   "r1",
		Clusters: []skillcard.ClusterSummary{
			{Label: "alpha", MemberCount: 3, Packages: []string{"a", "b"}},
			{Label: "beta", MemberCount: 7, Packages: []string{"b"}},
		},
	}
	sections := skillcard.Generate(input)
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	if sections[0].ClusterLabel != "alpha" {
		t.Errorf("expected ClusterLabel=alpha, got %q", sections[0].ClusterLabel)
	}
	if sections[1].ClusterLabel != "beta" {
		t.Errorf("expected ClusterLabel=beta, got %q", sections[1].ClusterLabel)
	}
	if !strings.HasPrefix(sections[0].Heading, "## Subsystem:") {
		t.Error("heading should start with ## Subsystem:")
	}
}
