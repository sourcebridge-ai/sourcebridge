// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package rest

import (
	"encoding/json"
	"testing"
	"time"

	graphstore "github.com/sourcebridge/sourcebridge/internal/graph"
	"github.com/sourcebridge/sourcebridge/internal/indexer"
)

// ---------------------------------------------------------------------------
// get_entry_points: stored-role and path-fallback tests (CA-549, Phase 3)
//
// Coverage:
//   - StoredRoleFiresClassification: when graph.File.GrailsRole is set
//     (stored-first path), get_entry_points fires grails_controller_action
//     classification without re-deriving from the path.
//   - EmptyRoleFallsBackToPathDerivation: when graph.File.GrailsRole is ""
//     (pre-migration / un-reindexed row), classification still fires via
//     the path re-derivation fallback (grailsRoleFromPath), matching the
//     behaviour before CA-549.
// ---------------------------------------------------------------------------

// buildEntryPointsHarness constructs a dedicated store + handler seeded with
// a single Grails controller file. Unlike newTestHarness (which always has
// only Go files), this helper lets callers control both the GrailsRole field
// and the file path directly to exercise the stored-first vs. path-fallback
// paths independently.
//
// filePath controls whether grailsRoleFromPath would independently classify
// the file:
//   - a grails-app/controllers/... path: classified by BOTH stored role AND
//     path derivation — use this for the path-fallback test only.
//   - a src/main/groovy/... path: NOT classified by path derivation — use
//     this for the stored-first test so that only the stored GrailsRole value
//     can produce a grails_controller_action entry point.
func buildEntryPointsHarness(t *testing.T, storedRole, filePath string) (*mcpTestHarness, string) {
	t.Helper()

	store := graphstore.NewStore()
	worker := &mockWorkerCaller{available: true}
	ks := newMockKnowledgeStore()

	result := &indexer.IndexResult{
		RepoName: "grails-ep-test-repo",
		RepoPath: "/tmp/grails-ep-test-repo",
		Files: []indexer.FileResult{
			{
				Path:       filePath,
				Language:   "groovy",
				GrailsRole: storedRole,
				Symbols: []indexer.Symbol{
					{
						Name:      "list",
						Kind:      "method",
						Language:  "groovy",
						FilePath:  filePath,
						StartLine: 10,
						EndLine:   20,
					},
				},
			},
		},
	}

	repo, err := store.StoreIndexResult(t.Context(), result)
	if err != nil {
		t.Fatalf("StoreIndexResult: %v", err)
	}

	h := newMCPHandler(store, ks, worker, "", 1*time.Hour, 30*time.Second, 100, nil)

	harness := &mcpTestHarness{
		handler: h,
		store:   store,
		worker:  worker,
		ks:      ks,
		repoID:  repo.ID,
	}
	return harness, repo.ID
}

// parseEntryPointsResult extracts the entry_points array from a tools/call
// response. Returns the slice of entry-point maps and a bool indicating
// whether the response was an error.
func parseEntryPointsResult(t *testing.T, resp jsonRPCResponse) ([]map[string]interface{}, bool) {
	t.Helper()
	text, isErr := parseToolText(resp)
	if isErr {
		return nil, true
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("unmarshal entry_points payload: %v\nraw: %s", err, text)
	}
	raw, ok := payload["entry_points"]
	if !ok {
		return nil, false
	}
	rawSlice, ok := raw.([]interface{})
	if !ok {
		return nil, false
	}
	eps := make([]map[string]interface{}, 0, len(rawSlice))
	for _, item := range rawSlice {
		if m, ok := item.(map[string]interface{}); ok {
			eps = append(eps, m)
		}
	}
	return eps, false
}

// hasKind reports whether any entry point in eps has the given kind string.
func hasKind(eps []map[string]interface{}, kind string) bool {
	for _, ep := range eps {
		if ep["kind"] == kind {
			return true
		}
	}
	return false
}

// TestGetEntryPoints_StoredRoleFiresClassification verifies the stored-first
// path (Decision 2, CA-549): when graph.File.GrailsRole is set to
// indexer.GrailsRoleController, get_entry_points fires
// grails_controller_action classification via the stored value — it does
// NOT need to re-derive the role from the path.
//
// The fixture uses a NON-classifying path (src/main/groovy/…) so that
// grailsRoleFromPath returns "" for this file. Classification can therefore
// only succeed via the stored GrailsRole field. If the implementation is
// changed to ignore f.GrailsRole (reverting the stored-first branch), this
// test will fail because the path fallback produces no role for this path.
func TestGetEntryPoints_StoredRoleFiresClassification(t *testing.T) {
	h, repoID := buildEntryPointsHarness(t, indexer.GrailsRoleController, "src/main/groovy/com/example/BookController.groovy")
	sess := h.createSession()

	resp := h.sendRPC(sess, 1, "tools/call", map[string]interface{}{
		"name": "get_entry_points",
		"arguments": map[string]interface{}{
			"repository_id": repoID,
			"precision":     "framework_aware",
		},
	})

	eps, isErr := parseEntryPointsResult(t, resp)
	if isErr {
		t.Fatal("get_entry_points returned an error response; want success")
	}
	if !hasKind(eps, "grails_controller_action") {
		t.Errorf("no grails_controller_action entry point found; got %d entry points: %v", len(eps), eps)
	}
}

// TestGetEntryPoints_EmptyRoleFallsBackToPathDerivation verifies the
// path-derivation fallback (Decision 2, CA-549): when graph.File.GrailsRole
// is "" (simulating a pre-migration or un-reindexed row), get_entry_points
// still fires grails_controller_action classification via grailsRoleFromPath,
// which keys off the grails-app/controllers/ path convention.
func TestGetEntryPoints_EmptyRoleFallsBackToPathDerivation(t *testing.T) {
	h, repoID := buildEntryPointsHarness(t, "", "grails-app/controllers/com/example/BookController.groovy")
	sess := h.createSession()

	resp := h.sendRPC(sess, 2, "tools/call", map[string]interface{}{
		"name": "get_entry_points",
		"arguments": map[string]interface{}{
			"repository_id": repoID,
			"precision":     "framework_aware",
		},
	})

	eps, isErr := parseEntryPointsResult(t, resp)
	if isErr {
		t.Fatal("get_entry_points returned an error response; want success")
	}
	if !hasKind(eps, "grails_controller_action") {
		t.Errorf("path-derivation fallback did not fire: no grails_controller_action entry point; got %d entry points: %v", len(eps), eps)
	}
}
