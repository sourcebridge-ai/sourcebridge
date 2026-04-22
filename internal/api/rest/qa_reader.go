// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package rest

import (
	"github.com/sourcebridge/sourcebridge/internal/knowledge"
	"github.com/sourcebridge/sourcebridge/internal/settings/comprehension"
)

// qaUnderstandingReader adapts the server's knowledge + summary stores
// to the qa.UnderstandingReader interface without leaking either store
// type into internal/qa.
type qaUnderstandingReader struct {
	knowledge knowledge.KnowledgeStore
	summaries comprehension.SummaryNodeStore
}

func (a qaUnderstandingReader) GetRepositoryUnderstanding(repoID string, scope knowledge.ArtifactScope) *knowledge.RepositoryUnderstanding {
	if a.knowledge == nil {
		return nil
	}
	return a.knowledge.GetRepositoryUnderstanding(repoID, scope)
}

func (a qaUnderstandingReader) GetSummaryNodes(corpusID string) ([]comprehension.SummaryNode, error) {
	if a.summaries == nil {
		return nil, nil
	}
	return a.summaries.GetSummaryNodes(corpusID)
}
