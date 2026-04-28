// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

// Style guide for this renderer:
//
//   - Every line must contain a concrete fact derived from graph or cluster data.
//   - No generic advice ("use get_callers before renaming", "check docs first").
//   - No document-level headers like "## Working in this repo" or "## Overview".
//   - Header block is ≤10 lines — the model must actually read it.
//   - "Try this first" prompt names real cluster labels from this repo.
//   - Watch out: bullets cite the specific symbol and the packages it crosses.
//   - No filler. Omit a warning rather than invent a vague one.

package skillcard

import (
	"fmt"
	"strings"
)

const (
	markerStart = "<!-- sourcebridge:start -->"
	markerEnd   = "<!-- sourcebridge:end -->"
)

// renderHeader produces the ≤10-line header block for the generated region.
// The "Try this first" prompt dynamically names the two largest clusters.
func renderHeader(input RepoSummary) string {
	indexedDate := input.IndexedAt.UTC().Format("2006-01-02")

	var sb strings.Builder
	sb.WriteString(markerStart + "\n")
	fmt.Fprintf(&sb, "# SourceBridge — %s\n", input.RepoName)
	fmt.Fprintf(&sb, "Repo ID: %s | Indexed: %s | Server: %s\n",
		input.RepoID, indexedDate, input.ServerURL)
	fmt.Fprintf(&sb, "Refresh: `sourcebridge setup claude --repo-id %s`\n", input.RepoID)
	sb.WriteString("\n")
	sb.WriteString("Before refactoring, run `get_subsystem(<symbol>)` to see what cluster it lives in.\n")
	sb.WriteString("\n")
	sb.WriteString("Try this first:\n")
	sb.WriteString(`  "` + buildTryThisPrompt(input.Clusters) + `"` + "\n")
	return sb.String()
}

// buildTryThisPrompt constructs the "Try this first" string from the cluster list.
// Clusters must be ordered largest-first (by MemberCount) before this is called.
// When two or more clusters are present, the prompt names both so the agent
// exercises subsystem awareness AND cross-cluster relationships.
func buildTryThisPrompt(clusters []ClusterSummary) string {
	if len(clusters) == 0 {
		return "List the subsystems of this repo."
	}
	if len(clusters) == 1 {
		return fmt.Sprintf(
			"List the subsystems of this repo, then show me the top symbols in the %s cluster.",
			clusters[0].Label,
		)
	}
	// Two or more clusters: name the two largest so the agent can explore
	// cross-cluster relationships, not just the dominant subsystem.
	return fmt.Sprintf(
		"Compare the %s and %s clusters: which symbols cross between them?",
		clusters[0].Label,
		clusters[1].Label,
	)
}

// renderSection produces the Section for a single cluster.
func renderSection(c ClusterSummary) Section {
	heading := fmt.Sprintf("## Subsystem: %s", c.Label)

	var sb strings.Builder
	// Summary line: "<N> symbols · <M> packages (pkg1, pkg2, pkg3)"
	if len(c.Packages) > 0 {
		pkgList := packageList(c.Packages)
		fmt.Fprintf(&sb, "%d symbols · %d %s (%s)\n",
			c.MemberCount,
			len(c.Packages),
			pluralize("package", len(c.Packages)),
			pkgList,
		)
	} else {
		fmt.Fprintf(&sb, "%d symbols\n", c.MemberCount)
	}

	// Watch out: bullets — one per Warning, no filler.
	for _, w := range c.Warnings {
		if w.Detail == "" {
			continue
		}
		fmt.Fprintf(&sb, "Watch out: %s\n", w.Detail)
	}

	return Section{
		Heading:      heading,
		Body:         sb.String(),
		ClusterLabel: c.Label,
	}
}

// Render assembles the complete generated region from a RepoSummary.
// The returned string includes the start and end markers.
func Render(input RepoSummary) string {
	sections := Generate(input)

	var sb strings.Builder
	sb.WriteString(renderHeader(input))

	for _, sec := range sections {
		sb.WriteString("\n")
		sb.WriteString(sec.Heading + "\n")
		sb.WriteString(sec.Body)
	}

	sb.WriteString(markerEnd + "\n")
	return sb.String()
}

// packageList formats up to 5 package names as a comma-separated string.
func packageList(pkgs []string) string {
	cap := 5
	if len(pkgs) < cap {
		cap = len(pkgs)
	}
	return strings.Join(pkgs[:cap], ", ")
}

// pluralize returns the singular or plural form of word based on count.
func pluralize(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "s"
}
