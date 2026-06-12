// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

//go:build integration

// CA-549: GrailsRole persists to ca_file through both write paths.
//
// This is the formal proof that migration 061 + the SCHEMAFULL DEFINE FIELD +
// buildIndexBatch write + surrealFile/toFile() read compose correctly against a
// real SurrealDB v2.6.5 container. The in-memory MemStore tests (Phase 1) cannot
// prove the SCHEMAFULL behavior: a field written without a DEFINE FIELD is
// silently dropped by SurrealDB on persist, which is what this test guards.
//
// Two-checkpoint shape (pinned by plan reviewer):
//   Checkpoint 1 — after StoreIndexResult, grails_role round-trips.
//   Checkpoint 2 — after ReplaceIndexResult (re-index cycle), grails_role still
//                  round-trips (proves the shared buildIndexBatch covers both paths).
//
// A non-Grails file is included in the fixture and asserted to carry GrailsRole == ""
// at both checkpoints (the '' DEFAULT / no-role sentinel).

package db

import (
	"testing"

	"github.com/sourcebridge/sourcebridge/internal/graph"
	"github.com/sourcebridge/sourcebridge/internal/indexer"
)

func TestGrailsRolePersistsAfterStoreAndReplace(t *testing.T) {
	s := startSurrealContainer(t)
	store := NewSurrealStore(s)

	// Fixture: one Grails controller file + one plain Go file.
	// GrailsRole is set explicitly via the typed constant — matches the
	// integration-test convention in replace_index_result_integration_test.go.
	initial := &indexer.IndexResult{
		RepoName:   "grails-test-repo",
		RepoPath:   "/tmp/grails-test-repo",
		TotalFiles: 2,
		Files: []indexer.FileResult{
			{
				Path:       "grails-app/controllers/com/example/BookController.groovy",
				Language:   "groovy",
				LineCount:  20,
				GrailsRole: indexer.GrailsRoleController,
				Symbols: []indexer.Symbol{
					{
						ID:            "sym-book-list",
						Name:          "list",
						QualifiedName: "BookController.list",
						Kind:          indexer.SymbolFunction,
						Language:      "groovy",
						FilePath:      "grails-app/controllers/com/example/BookController.groovy",
						StartLine:     5,
						EndLine:       10,
					},
				},
			},
			{
				Path:      "src/main/go/util.go",
				Language:  "go",
				LineCount: 8,
				// GrailsRole intentionally absent — should round-trip as "".
				Symbols: []indexer.Symbol{
					{
						ID:            "sym-util-helper",
						Name:          "helper",
						QualifiedName: "main.helper",
						Kind:          indexer.SymbolFunction,
						Language:      "go",
						FilePath:      "src/main/go/util.go",
						StartLine:     1,
						EndLine:       8,
					},
				},
			},
		},
	}

	// ── Checkpoint 1: StoreIndexResult ────────────────────────────────────────
	repo, err := store.StoreIndexResult(t.Context(), initial)
	if err != nil {
		t.Fatalf("StoreIndexResult: %v", err)
	}

	files := store.GetFiles(t.Context(), repo.ID)
	if len(files) != 2 {
		t.Fatalf("checkpoint 1: want 2 files, got %d", len(files))
	}

	assertGrailsRoles(t, "checkpoint 1 (after StoreIndexResult)", files)

	// ── Checkpoint 2: ReplaceIndexResult (re-index cycle) ─────────────────────
	// Same content — exercises ReplaceIndexResult via the shared buildIndexBatch.
	// This proves the CA-304 structural symmetry: one edit to buildIndexBatch
	// covers both Store and Replace paths.
	reindex := &indexer.IndexResult{
		RepoName:   initial.RepoName,
		RepoPath:   initial.RepoPath,
		TotalFiles: initial.TotalFiles,
		Files:      initial.Files,
	}
	_, err = store.ReplaceIndexResult(t.Context(), repo.ID, reindex)
	if err != nil {
		t.Fatalf("ReplaceIndexResult: %v", err)
	}

	filesAfter := store.GetFiles(t.Context(), repo.ID)
	if len(filesAfter) != 2 {
		t.Fatalf("checkpoint 2: want 2 files, got %d", len(filesAfter))
	}

	assertGrailsRoles(t, "checkpoint 2 (after ReplaceIndexResult)", filesAfter)
}

// assertGrailsRoles checks that among the returned files:
//   - the Grails controller file carries GrailsRole == indexer.GrailsRoleController
//   - the plain Go file carries GrailsRole == "" (the no-role sentinel / DEFAULT '')
func assertGrailsRoles(t *testing.T, label string, files []*graph.File) {
	t.Helper()
	found := map[string]string{} // path → grails_role
	for _, f := range files {
		found[f.Path] = f.GrailsRole
	}

	controllerPath := "grails-app/controllers/com/example/BookController.groovy"
	plainPath := "src/main/go/util.go"

	role, ok := found[controllerPath]
	if !ok {
		t.Errorf("%s: controller file not found in GetFiles result", label)
	} else if role != indexer.GrailsRoleController {
		t.Errorf("%s: controller file GrailsRole = %q, want %q (SCHEMAFULL drop? migration 061 missing?)",
			label, role, indexer.GrailsRoleController)
	}

	role, ok = found[plainPath]
	if !ok {
		t.Errorf("%s: plain Go file not found in GetFiles result", label)
	} else if role != "" {
		t.Errorf("%s: plain Go file GrailsRole = %q, want %q", label, role, "")
	}
}
