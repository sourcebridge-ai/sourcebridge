// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package graphql

import (
	"context"

	"google.golang.org/grpc/metadata"
)

// withModelMetadata enriches a context with the API's effective LLM
// configuration for a single worker call. The API owns the DB-backed
// LLM settings, so the worker should not rely on deployment env vars
// as the runtime source of truth.
func (r *Resolver) withModelMetadata(ctx context.Context, operationGroup string) context.Context {
	if r.Config == nil {
		return ctx
	}

	model := r.Config.LLM.ModelForOperation(operationGroup)
	pairs := []string{
		"x-sb-llm-provider", r.Config.LLM.Provider,
		"x-sb-llm-base-url", r.Config.LLM.BaseURL,
		"x-sb-llm-api-key", r.Config.LLM.APIKey,
		"x-sb-llm-draft-model", r.Config.LLM.DraftModel,
		"x-sb-operation", operationGroup,
	}
	if model != "" {
		pairs = append(pairs, "x-sb-model", model)
	}
	md := metadata.Pairs(pairs...)
	return metadata.NewOutgoingContext(ctx, md)
}
