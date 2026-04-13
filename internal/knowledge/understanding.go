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
	if a.UnderstandingRevisionFP == "" || u.RevisionFP == "" {
		return false
	}
	return a.UnderstandingRevisionFP != u.RevisionFP
}
