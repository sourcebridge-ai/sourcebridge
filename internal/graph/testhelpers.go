// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package graph

// InjectSymbolForTest inserts a symbol into the in-memory store
// without going through the full indexer pipeline. It exists purely
// to keep search-layer unit tests small and readable; production
// code must use StoreIndexResult / ReplaceIndexResult so that
// files, call edges, and module rollups are populated coherently.
//
// The helper is safe to use from tests in other packages because it
// is exported; it is intentionally not surfaced on GraphStore so the
// call site is obvious.
func (s *Store) InjectSymbolForTest(repoID string, sym *StoredSymbol) {
	if sym == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sym.RepoID = repoID
	if s.symbols == nil {
		s.symbols = make(map[string]*StoredSymbol)
	}
	s.symbols[sym.ID] = sym
	s.repoSymbols[repoID] = append(s.repoSymbols[repoID], sym.ID)
}
