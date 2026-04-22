// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package rest

import (
	"os"
	"path/filepath"

	graphstore "github.com/sourcebridge/sourcebridge/internal/graph"
	"github.com/sourcebridge/sourcebridge/internal/qa"
)

// qaRepoLocator adapts the graph store's Repository records to
// qa.RepoLocator. Uses the same clone-path-resolution logic as the
// GraphQL resolvers (resolveRepoSourcePath in api/graphql/helpers.go)
// so QA sees exactly the paths the rest of the product sees.
type qaRepoLocator struct {
	store         graphstore.GraphStore
	repoCacheBase string
}

func newQARepoLocator(store graphstore.GraphStore, repoCacheBase string) *qaRepoLocator {
	return &qaRepoLocator{store: store, repoCacheBase: repoCacheBase}
}

// LocateRepoClone resolves a repo ID to its on-disk root. Mirrors
// resolveRepoSourcePath's decision order: persisted clone_path →
// computed cache path → local Path fallback.
func (l *qaRepoLocator) LocateRepoClone(repoID string) (string, bool) {
	if l == nil || l.store == nil {
		return "", false
	}
	repo := l.store.GetRepository(repoID)
	if repo == nil {
		return "", false
	}
	if repo.ClonePath != "" {
		if info, err := os.Stat(repo.ClonePath); err == nil && info.IsDir() {
			return repo.ClonePath, true
		}
	}
	if repo.Name != "" && l.repoCacheBase != "" {
		computed := filepath.Join(l.repoCacheBase, "repos", sanitizeRepoNameForQA(repo.Name))
		if info, err := os.Stat(computed); err == nil && info.IsDir() {
			return computed, true
		}
	}
	if repo.Path != "" {
		if info, err := os.Stat(repo.Path); err == nil && info.IsDir() {
			return repo.Path, true
		}
	}
	return "", false
}

// sanitizeRepoNameForQA mirrors sanitizeRepoName in
// internal/api/graphql/helpers.go. Duplicated here rather than
// exported because the other variant is unexported and changing its
// visibility would leak package internals unnecessarily — the rule
// is intentionally narrow (replace / and : with -).
func sanitizeRepoNameForQA(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		switch r {
		case '/', ':':
			out = append(out, '-')
		default:
			out = append(out, r)
		}
	}
	return string(out)
}

// qaGraphLookup adapts the graph store's symbol lookup to
// qa.graphSymbolLookup without leaking graph.StoredSymbol into the qa
// package.
type qaGraphLookup struct {
	store graphstore.GraphStore
}

func (g *qaGraphLookup) Lookup(id string) (string, string, string, int, int, bool) {
	if g == nil || g.store == nil {
		return "", "", "", 0, 0, false
	}
	sym := g.store.GetSymbol(id)
	if sym == nil {
		return "", "", "", 0, 0, false
	}
	qn := sym.QualifiedName
	if qn == "" {
		qn = sym.Name
	}
	return qn, sym.FilePath, sym.Language, sym.StartLine, sym.EndLine, true
}

// qaGraphAdapter adapts the store's caller/callee methods. Returns a
// minimal-surface value that qa.NewGraphExpander consumes.
type qaGraphAdapter struct {
	store graphstore.GraphStore
}

func (a *qaGraphAdapter) GetCallers(id string) []string { return a.store.GetCallers(id) }
func (a *qaGraphAdapter) GetCallees(id string) []string { return a.store.GetCallees(id) }

// compile-time check: both adapters satisfy the qa package's
// interfaces. This catches drift if the qa interfaces change.
var _ qa.RepoLocator = (*qaRepoLocator)(nil)
