// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package rest

import (
	"context"
	"encoding/json"

	"github.com/sourcebridge/sourcebridge/internal/entrypoints"
	"github.com/sourcebridge/sourcebridge/internal/indexer"
)

// Phase 1b — get_entry_points.
//
// Adapts stored graph.Symbols + FileResults into the
// internal/entrypoints classifier and returns the result. Two modes:
// "basic" is language-agnostic (main funcs + HTTP-verb-named
// functions); "framework_aware" adds per-framework detection
// (Grails controllers, FastAPI/Flask decorators, Go http.ResponseWriter
// signatures, Next.js API routes).

func (h *mcpHandler) getEntryPointsToolDef() mcpToolDefinition {
	return mcpToolDefinition{
		Name:        "get_entry_points",
		Description: "Return structured entry points across the indexed repo — main funcs, HTTP routes, Grails controller actions, message handlers. `precision: \"basic\"` uses language-agnostic heuristics; `precision: \"framework_aware\"` adds per-framework detectors.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"repository_id": map[string]interface{}{"type": "string", "description": "Repository ID"},
				"kind": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"main", "http_route", "cli_command", "message_handler", "scheduled_job", "grails_controller_action", "any"},
					"description": "Filter to a single entry-point kind (default: any)",
				},
				"precision": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"basic", "framework_aware"},
					"description": "Classifier precision (default: framework_aware)",
				},
				"limit": map[string]interface{}{"type": "integer", "description": "Max results (default: 200, cap 1000)"},
			},
			"required": []string{"repository_id"},
		},
	}
}

type entryPointsResult struct {
	RepositoryID string                   `json:"repository_id"`
	Precision    string                   `json:"precision"`
	EntryPoints  []entrypoints.EntryPoint `json:"entry_points"`
	Total        int                      `json:"total"`
	Truncated    bool                     `json:"truncated,omitempty"`
}

func (h *mcpHandler) callGetEntryPoints(ctx context.Context, session *mcpSession, args json.RawMessage) (interface{}, error) {
	var params struct {
		RepositoryID string `json:"repository_id"`
		Kind         string `json:"kind"`
		Precision    string `json:"precision"`
		Limit        int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, errInvalidArguments(err.Error())
	}
	if err := h.checkRepoAccess(ctx, session, params.RepositoryID); err != nil {
		return nil, err
	}

	precision := entrypoints.PrecisionFrameworkAware
	if params.Precision == string(entrypoints.PrecisionBasic) {
		precision = entrypoints.PrecisionBasic
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}

	// Pull every symbol once. The classifier is a linear pass; we
	// don't need pagination here — the Phase 2 pagination work
	// adds cursors for all list tools as a uniform treatment.
	storedSymbols, _ := h.store.GetSymbols(ctx, params.RepositoryID, nil, nil, 0, 0)
	symbols := make([]entrypoints.Symbol, 0, len(storedSymbols))
	for _, s := range storedSymbols {
		symbols = append(symbols, entrypoints.Symbol{
			ID:        s.ID,
			Name:      s.Name,
			Kind:      s.Kind,
			Language:  s.Language,
			FilePath:  s.FilePath,
			StartLine: s.StartLine,
			EndLine:   s.EndLine,
			Signature: s.Signature,
			IsTest:    s.IsTest,
		})
	}

	// File metadata — Grails role is what the framework-aware classifier
	// reads. Stored-first policy (Decision 2, CA-549): read the role from
	// graph.File.GrailsRole when it is non-empty (set by the indexer and
	// persisted to ca_file via migration 061); fall back to path-derivation
	// via grailsRoleFromPath when the stored value is empty. Empty means a
	// pre-migration row or a file that has not been reindexed since CA-549
	// landed — path re-derivation gives the exact same result as before, so
	// Grails classification is never silently regressed during the upgrade
	// window. After a reindex, the stored value wins and path re-derivation
	// never runs.
	//
	// Forward-compatibility note: any future consumer that builds an
	// entrypoints.File from a graph.File MUST implement the same
	// stored-first/path-fallback contract. A consumer that reads
	// f.GrailsRole unconditionally (without the empty-value path fallback)
	// silently regresses Grails classification for pre-migration and
	// un-reindexed rows. The fallback is not an MCP-local quirk; it is the
	// read contract for stored Grails roles until every install has
	// reindexed.
	storedFiles := h.store.GetFiles(ctx, params.RepositoryID)
	files := make([]entrypoints.File, 0, len(storedFiles))
	for _, f := range storedFiles {
		role := f.GrailsRole
		if role == "" {
			role = grailsRoleFromPath(f.Path)
		}
		files = append(files, entrypoints.File{
			Path:       f.Path,
			Language:   f.Language,
			GrailsRole: role,
		})
	}

	all := entrypoints.Classify(symbols, files, precision)

	// Kind filter.
	if params.Kind != "" && params.Kind != "any" {
		filtered := all[:0]
		for _, ep := range all {
			if string(ep.Kind) == params.Kind {
				filtered = append(filtered, ep)
			}
		}
		all = filtered
	}

	total := len(all)
	truncated := false
	if total > limit {
		all = all[:limit]
		truncated = true
	}

	return entryPointsResult{
		RepositoryID: params.RepositoryID,
		Precision:    string(precision),
		EntryPoints:  all,
		Total:        total,
		Truncated:    truncated,
	}, nil
}

// grailsRoleFromPath delegates to the canonical indexer.GrailsRoleFor
// classifier. It is the fallback for pre-migration rows and un-reindexed
// files whose graph.File.GrailsRole is empty (see the storedFiles loop
// above). indexer.GrailsRoleFor is a pure, stateless path classifier —
// no I/O, no grammar dependency — so the import and the per-call
// invocation are both safe.
func grailsRoleFromPath(path string) string {
	return indexer.GrailsRoleFor(path)
}
