// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package graph

import (
	"sort"
	"testing"

	"github.com/sourcebridge/sourcebridge/internal/indexer"
)

// TestRecomputePackageDependencies_BasicAggregation verifies that after indexing
// a repo with two packages where one imports the other, RecomputePackageDependencies
// produces the expected Imports and ImportedBy edges.
func TestRecomputePackageDependencies_BasicAggregation(t *testing.T) {
	store := NewStore()

	// Seed: internal/auth imports internal/jwt.
	result := &indexer.IndexResult{
		RepoName: "test-repo",
		RepoPath: "/tmp/test-pkgdeps",
		Files: []indexer.FileResult{
			{
				Path:      "internal/auth/auth.go",
				Language:  "go",
				LineCount: 20,
				Imports: []indexer.Import{
					{Path: "internal/jwt", Line: 3},
					{Path: "fmt", Line: 1},
				},
			},
			{
				Path:      "internal/jwt/jwt.go",
				Language:  "go",
				LineCount: 10,
				Imports:   []indexer.Import{},
			},
			{
				Path:      "internal/api/handler.go",
				Language:  "go",
				LineCount: 15,
				Imports: []indexer.Import{
					{Path: "internal/auth", Line: 2},
				},
			},
		},
	}

	repo, err := store.StoreIndexResult(t.Context(), result)
	if err != nil {
		t.Fatalf("StoreIndexResult: %v", err)
	}

	store.RecomputePackageDependencies(t.Context(), repo.ID)

	deps := store.GetPackageDependencies(t.Context(), repo.ID)
	if len(deps) == 0 {
		t.Fatal("expected non-empty package dependencies after recompute")
	}

	// Build lookup by package name.
	byPkg := make(map[string]*StoredPackageDependencies)
	for _, d := range deps {
		byPkg[d.Package] = d
	}

	// internal/auth should import internal/jwt (fmt is external, no inbound from repo).
	authDeps, ok := byPkg["internal/auth"]
	if !ok {
		t.Fatal("expected package_dep record for internal/auth")
	}
	if !containsStr(authDeps.Imports, "internal/jwt") {
		t.Errorf("internal/auth.Imports should contain internal/jwt; got %v", authDeps.Imports)
	}
	// internal/auth should be imported by internal/api.
	if !containsStr(authDeps.ImportedBy, "internal/api") {
		t.Errorf("internal/auth.ImportedBy should contain internal/api; got %v", authDeps.ImportedBy)
	}

	// internal/jwt should appear in importedBy because internal/auth imports it.
	jwtDeps, ok := byPkg["internal/jwt"]
	if !ok {
		t.Fatal("expected package_dep record for internal/jwt")
	}
	if !containsStr(jwtDeps.ImportedBy, "internal/auth") {
		t.Errorf("internal/jwt.ImportedBy should contain internal/auth; got %v", jwtDeps.ImportedBy)
	}
}

// TestRecomputePackageDependencies_Idempotent verifies that calling
// RecomputePackageDependencies twice produces the same result (no duplicate edges).
func TestRecomputePackageDependencies_Idempotent(t *testing.T) {
	store := NewStore()

	result := &indexer.IndexResult{
		RepoName: "test-repo",
		RepoPath: "/tmp/test-pkgdeps-idempotent",
		Files: []indexer.FileResult{
			{
				Path:     "pkg/a/a.go",
				Language: "go",
				Imports:  []indexer.Import{{Path: "pkg/b", Line: 1}},
			},
			{
				Path:     "pkg/b/b.go",
				Language: "go",
				Imports:  nil,
			},
		},
	}

	repo, err := store.StoreIndexResult(t.Context(), result)
	if err != nil {
		t.Fatalf("StoreIndexResult: %v", err)
	}

	store.RecomputePackageDependencies(t.Context(), repo.ID)
	store.RecomputePackageDependencies(t.Context(), repo.ID) // second call

	deps := store.GetPackageDependencies(t.Context(), repo.ID)

	// Count entries for pkg/a — must be exactly 1.
	var aCount int
	for _, d := range deps {
		if d.Package == "pkg/a" {
			aCount++
		}
	}
	if aCount != 1 {
		t.Errorf("expected exactly 1 package_dep for pkg/a after two recomputes, got %d", aCount)
	}

	// pkg/a.Imports must contain pkg/b exactly once.
	for _, d := range deps {
		if d.Package == "pkg/a" {
			sorted := make([]string, len(d.Imports))
			copy(sorted, d.Imports)
			sort.Strings(sorted)
			var prev string
			for _, imp := range sorted {
				if imp == prev {
					t.Errorf("duplicate import %q in pkg/a.Imports after idempotent recompute", imp)
				}
				prev = imp
			}
		}
	}
}

// TestGetPackageDependencies_EmptyBeforeRecompute verifies that GetPackageDependencies
// returns an empty slice for a freshly indexed repo before RecomputePackageDependencies is called.
func TestGetPackageDependencies_EmptyBeforeRecompute(t *testing.T) {
	store := NewStore()

	result := &indexer.IndexResult{
		RepoName: "test-repo",
		RepoPath: "/tmp/test-pkgdeps-empty",
		Files: []indexer.FileResult{
			{
				Path:    "main.go",
				Imports: []indexer.Import{{Path: "fmt", Line: 1}},
			},
		},
	}

	repo, err := store.StoreIndexResult(t.Context(), result)
	if err != nil {
		t.Fatalf("StoreIndexResult: %v", err)
	}

	deps := store.GetPackageDependencies(t.Context(), repo.ID)
	if len(deps) != 0 {
		t.Errorf("expected empty deps before RecomputePackageDependencies, got %d entries", len(deps))
	}
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// TestStoreIndexResult_GrailsRoleRoundTrips verifies that GrailsRole set on a
// FileResult is carried through to graph.File at all three in-memory construction
// sites: StoreIndexResult (checkpoint 1) and ReplaceIndexResult (checkpoint 2).
// The MergeIndexResult site is covered separately in merge_index_result_test.go.
func TestStoreIndexResult_GrailsRoleRoundTrips(t *testing.T) {
	store := NewStore()

	result := &indexer.IndexResult{
		RepoName: "grails-test-repo",
		RepoPath: "/tmp/grails-test",
		Files: []indexer.FileResult{
			{
				Path:       "grails-app/controllers/com/example/BookController.groovy",
				Language:   "groovy",
				LineCount:  42,
				GrailsRole: indexer.GrailsRoleController,
				Symbols: []indexer.Symbol{
					{ID: "book-list", Name: "list", QualifiedName: "com.example.BookController.list", Kind: indexer.SymbolMethod, Language: "groovy", FilePath: "grails-app/controllers/com/example/BookController.groovy", StartLine: 10, EndLine: 15},
				},
			},
			{
				Path:      "grails-app/domain/com/example/Book.groovy",
				Language:  "groovy",
				LineCount: 20,
				// GrailsRole deliberately omitted — non-controller file, role = "".
			},
		},
	}

	// Checkpoint 1: StoreIndexResult.
	repo, err := store.StoreIndexResult(t.Context(), result)
	if err != nil {
		t.Fatalf("StoreIndexResult: %v", err)
	}
	assertGrailsRoles(t, store, repo.ID, "after StoreIndexResult")

	// Checkpoint 2: ReplaceIndexResult — same files, same roles.
	if _, err := store.ReplaceIndexResult(t.Context(), repo.ID, result); err != nil {
		t.Fatalf("ReplaceIndexResult: %v", err)
	}
	assertGrailsRoles(t, store, repo.ID, "after ReplaceIndexResult")
}

// assertGrailsRoles is a helper shared between checkpoints.
func assertGrailsRoles(t *testing.T, store *Store, repoID, label string) {
	t.Helper()
	files := store.GetFiles(t.Context(), repoID)
	if len(files) != 2 {
		t.Fatalf("%s: expected 2 files, got %d", label, len(files))
	}
	for _, f := range files {
		switch f.Path {
		case "grails-app/controllers/com/example/BookController.groovy":
			if f.GrailsRole != indexer.GrailsRoleController {
				t.Errorf("%s: controller file GrailsRole = %q, want %q", label, f.GrailsRole, indexer.GrailsRoleController)
			}
		case "grails-app/domain/com/example/Book.groovy":
			if f.GrailsRole != "" {
				t.Errorf("%s: domain file GrailsRole = %q, want %q (empty)", label, f.GrailsRole, "")
			}
		default:
			t.Errorf("%s: unexpected file path %q", label, f.Path)
		}
	}
}
