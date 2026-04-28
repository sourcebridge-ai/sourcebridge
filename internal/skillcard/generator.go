// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

// Package skillcard generates the .claude/CLAUDE.md skill card and supporting
// configuration files for Claude Code integration.
//
// Input types are native (no imports from clustering or wiki packages) so the
// package is independently testable and the dependency direction stays clean.
// Translation from clustering.Cluster → skillcard.ClusterSummary happens at
// the CLI call site.
package skillcard

import "time"

// RepoSummary is the top-level input to Generate. It carries everything needed
// to produce the CLAUDE.md header block and per-subsystem sections.
type RepoSummary struct {
	// RepoName is the human-readable repository name (e.g. "payments-service").
	RepoName string
	// RepoID is the stable SourceBridge repository identifier.
	RepoID string
	// ServerURL is the SourceBridge instance the data was fetched from.
	ServerURL string
	// IndexedAt is the timestamp of the most recent completed index run.
	IndexedAt time.Time
	// Clusters is the ordered list of subsystem clusters, largest first.
	Clusters []ClusterSummary
}

// ClusterSummary is a lightweight view of a single subsystem cluster.
// It contains only what is needed to render the CLAUDE.md section —
// the full member list is intentionally omitted.
type ClusterSummary struct {
	// Label is the heuristic or LLM-assigned cluster name (e.g. "auth").
	Label string
	// MemberCount is the total number of symbols in the cluster.
	MemberCount int
	// Packages is the deduplicated set of top-level package paths that
	// appear in this cluster's member list.
	Packages []string
	// RepresentativeSyms holds the top symbols by in-degree, capped at 5.
	RepresentativeSyms []string
	// Warnings is the list of call-graph-derived advisories for this cluster.
	Warnings []Warning
}

// Warning is a concrete, call-graph-derived advisory for a cluster.
// All fields must be populated — empty Detail strings are never rendered.
type Warning struct {
	// Symbol is the qualified symbol name the warning applies to.
	Symbol string
	// Kind classifies the warning type. Known values:
	//   "cross-package-callers" — symbol has callers in 3+ distinct top-level packages.
	//   "hot-path"             — symbol has the highest in-degree in the cluster.
	// Other kinds may be added without changing the rendering path.
	Kind string
	// Detail is the human-readable advisory text rendered verbatim under
	// "Watch out:". It must be a concrete statement derived from graph data,
	// not generic advice.
	Detail string
}

// Section is the rendered output for a single cluster, ready to be assembled
// into the CLAUDE.md generated region.
type Section struct {
	// Heading is the ## Subsystem: <label> line.
	Heading string
	// Body is the rendered markdown body (summary line + Watch out: bullets).
	Body string
	// ClusterLabel is the raw label used for section-hash keying.
	ClusterLabel string
}

// Generate produces the rendered sections for all clusters in the summary.
// The caller is responsible for assembling these into the full CLAUDE.md
// document via writer.MergeFileWithHash.
func Generate(input RepoSummary) []Section {
	sections := make([]Section, 0, len(input.Clusters))
	for _, c := range input.Clusters {
		sections = append(sections, renderSection(c))
	}
	return sections
}
