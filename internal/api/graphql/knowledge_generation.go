package graphql

import (
	"context"
	"fmt"

	graphstore "github.com/sourcebridge/sourcebridge/internal/graph"
)

const knowledgeWorkerUnavailableMessage = "AI features are unavailable — worker not connected"

func (r *Resolver) requireKnowledgeGenerationSupport() error {
	if r.Worker == nil {
		return fmt.Errorf(knowledgeWorkerUnavailableMessage)
	}
	if r.KnowledgeStore == nil {
		return fmt.Errorf("knowledge store not configured")
	}
	return nil
}

func (r *Resolver) loadKnowledgeRepository(ctx context.Context, repositoryID string) (*graphstore.Repository, error) {
	repo := r.getStore(ctx).GetRepository(repositoryID)
	if repo == nil {
		return nil, fmt.Errorf("repository %s not found", repositoryID)
	}
	return repo, nil
}
