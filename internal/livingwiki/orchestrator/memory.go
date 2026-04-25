// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package orchestrator

import (
	"context"
	"fmt"
	"sync"

	"github.com/sourcebridge/sourcebridge/internal/livingwiki/ast"
)

// MemoryPageStore is an in-memory implementation of [PageStore].
// It is intended for tests and local development. It does not persist across
// process restarts.
type MemoryPageStore struct {
	mu        sync.RWMutex
	canonical map[string]ast.Page  // key: repoID + "/" + pageID
	proposed  map[string]ast.Page  // key: repoID + "/" + prID + "/" + pageID
}

// NewMemoryPageStore returns an empty in-memory page store.
func NewMemoryPageStore() *MemoryPageStore {
	return &MemoryPageStore{
		canonical: make(map[string]ast.Page),
		proposed:  make(map[string]ast.Page),
	}
}

// Compile-time interface check.
var _ PageStore = (*MemoryPageStore)(nil)

func (m *MemoryPageStore) GetCanonical(_ context.Context, repoID, pageID string) (ast.Page, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.canonical[canonicalKey(repoID, pageID)]
	return p, ok, nil
}

func (m *MemoryPageStore) SetCanonical(_ context.Context, repoID string, page ast.Page) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.canonical[canonicalKey(repoID, page.ID)] = page
	return nil
}

func (m *MemoryPageStore) GetProposed(_ context.Context, repoID, prID, pageID string) (ast.Page, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.proposed[proposedKey(repoID, prID, pageID)]
	return p, ok, nil
}

func (m *MemoryPageStore) SetProposed(_ context.Context, repoID, prID string, page ast.Page) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.proposed[proposedKey(repoID, prID, page.ID)] = page
	return nil
}

func (m *MemoryPageStore) ListProposed(_ context.Context, repoID, prID string) ([]ast.Page, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	prefix := repoID + "/" + prID + "/"
	var pages []ast.Page
	for k, p := range m.proposed {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			pages = append(pages, p)
		}
	}
	return pages, nil
}

func (m *MemoryPageStore) DeleteProposed(_ context.Context, repoID, prID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := repoID + "/" + prID + "/"
	for k := range m.proposed {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			delete(m.proposed, k)
		}
	}
	return nil
}

func (m *MemoryPageStore) PromoteProposed(ctx context.Context, repoID, prID string) error {
	proposed, err := m.ListProposed(ctx, repoID, prID)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range proposed {
		// Get the current canonical page (may not exist — that's fine for cold-start).
		canonical, _ := m.canonical[canonicalKey(repoID, p.ID)]
		promoted := ast.Promote(canonical, p)
		m.canonical[canonicalKey(repoID, promoted.ID)] = promoted
	}
	// Discard proposed pages after promotion.
	prefix := repoID + "/" + prID + "/"
	for k := range m.proposed {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			delete(m.proposed, k)
		}
	}
	return nil
}

func canonicalKey(repoID, pageID string) string {
	return repoID + "/" + pageID
}

func proposedKey(repoID, prID, pageID string) string {
	return repoID + "/" + prID + "/" + pageID
}

// MemoryWikiPR is an in-memory implementation of [WikiPR].
// It captures the files and state for test inspection.
type MemoryWikiPR struct {
	mu     sync.Mutex
	id     string
	branch string
	title  string
	body   string
	files  map[string][]byte
	opened bool
	merged bool
	closed bool
}

// NewMemoryWikiPR returns a MemoryWikiPR with the given ID.
func NewMemoryWikiPR(id string) *MemoryWikiPR {
	return &MemoryWikiPR{id: id}
}

// Compile-time interface check.
var _ WikiPR = (*MemoryWikiPR)(nil)

func (m *MemoryWikiPR) ID() string { return m.id }

func (m *MemoryWikiPR) Open(_ context.Context, branch, title, body string, files map[string][]byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.opened {
		return fmt.Errorf("MemoryWikiPR: PR %q is already open", m.id)
	}
	m.branch = branch
	m.title = title
	m.body = body
	m.files = make(map[string][]byte, len(files))
	for k, v := range files {
		cp := make([]byte, len(v))
		copy(cp, v)
		m.files[k] = cp
	}
	m.opened = true
	return nil
}

func (m *MemoryWikiPR) Merged(_ context.Context) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.merged, nil
}

func (m *MemoryWikiPR) Closed(_ context.Context) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed, nil
}

// SimulateMerge marks the PR as merged (for test control).
func (m *MemoryWikiPR) SimulateMerge() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.merged = true
}

// SimulateClose marks the PR as closed without merge (for test control).
func (m *MemoryWikiPR) SimulateClose() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
}

// IsOpen reports whether the PR has been opened.
func (m *MemoryWikiPR) IsOpen() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.opened
}

// Branch returns the branch name passed to Open.
func (m *MemoryWikiPR) Branch() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.branch
}

// Title returns the PR title.
func (m *MemoryWikiPR) Title() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.title
}

// Files returns a copy of the files committed to this PR.
func (m *MemoryWikiPR) Files() map[string][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string][]byte, len(m.files))
	for k, v := range m.files {
		cp := make([]byte, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

// FilesystemRepoWriter is a [RepoWriter] that writes files to a local directory.
// The root directory is the base; file paths from the orchestrator are joined
// to this root. Suitable for tests and the FilesystemRepoWriter integration test.
type FilesystemRepoWriter struct {
	root string
}

// NewFilesystemRepoWriter creates a FilesystemRepoWriter that writes to root.
func NewFilesystemRepoWriter(root string) *FilesystemRepoWriter {
	return &FilesystemRepoWriter{root: root}
}

// Compile-time interface check.
var _ RepoWriter = (*FilesystemRepoWriter)(nil)

func (f *FilesystemRepoWriter) WriteFiles(_ context.Context, files map[string][]byte) error {
	// Import os and path/filepath locally to avoid cluttering the package-level imports.
	// We use the standard library via the helper below.
	return writeFilesToDir(f.root, files)
}
