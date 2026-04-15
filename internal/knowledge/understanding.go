// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package knowledge

import "strings"

// RevisionFingerprint returns the most stable revision identifier available for
// artifact freshness comparisons.
func RevisionFingerprint(rev SourceRevision) string {
	if strings.TrimSpace(rev.CommitSHA) != "" {
		return strings.TrimSpace(rev.CommitSHA)
	}
	if strings.TrimSpace(rev.ContentFingerprint) != "" {
		return strings.TrimSpace(rev.ContentFingerprint)
	}
	return strings.TrimSpace(rev.DocsFingerprint)
}

// ArtifactRefreshAvailable reports whether the artifact was built on an older
// understanding revision than the current shared understanding.
func ArtifactRefreshAvailable(a *Artifact, u *RepositoryUnderstanding) bool {
	if a == nil || u == nil {
		return false
	}
	if strings.TrimSpace(a.RendererVersion) != "" && a.RendererVersion != RendererVersionForArtifact(a.Type) {
		return true
	}
	if a.UnderstandingRevisionFP == "" || u.RevisionFP == "" {
		return false
	}
	return a.UnderstandingRevisionFP != u.RevisionFP
}

// RendererVersionForArtifact returns the current renderer schema/prompt
// version for a given artifact type.
func RendererVersionForArtifact(t ArtifactType) string {
	switch t {
	case ArtifactCliffNotes:
		return "cliff_notes:v2"
	case ArtifactArchitectureDiagram:
		return "architecture_diagram:v1"
	case ArtifactLearningPath:
		return "learning_path:v1"
	case ArtifactCodeTour:
		return "code_tour:v1"
	case ArtifactWorkflowStory:
		return "workflow_story:v1"
	default:
		return "artifact:v1"
	}
}

func SectionKeyForTitle(title string) string {
	key := strings.TrimSpace(strings.ToLower(title))
	key = strings.ReplaceAll(key, "&", "and")
	replacer := strings.NewReplacer(" ", "_", "-", "_", "/", "_")
	key = replacer.Replace(key)
	for strings.Contains(key, "__") {
		key = strings.ReplaceAll(key, "__", "_")
	}
	return strings.Trim(key, "_")
}
