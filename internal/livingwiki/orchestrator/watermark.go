// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package orchestrator

import (
	"context"
	"sync"
)

// Watermarks holds the two per-repo SHA markers used by the incremental
// generation model (A1.P2).
//
//   - SourceProcessedSHA — last commit the generator ran for. Advances after
//     every generation attempt regardless of whether the output was accepted.
//   - WikiPublishedSHA — last commit whose generated output has been merged
//     into the canonical wiki state. Advances only when a wiki PR merges
//     (via [Orchestrator.Promote]).
//
// The incremental diff target is always WikiPublishedSHA..HEAD, so an
// unmerged open PR does not cause subsequent pushes to skip changes.
type Watermarks struct {
	// SourceProcessedSHA is the last source commit the generator ran for.
	// Empty string means "never generated".
	SourceProcessedSHA string

	// WikiPublishedSHA is the last source commit whose output is in canonical.
	// Empty string means "no canonical wiki yet".
	WikiPublishedSHA string
}

// WatermarkStore persists and advances the two watermarks for each repository.
// Implementations must be safe for concurrent use.
type WatermarkStore interface {
	// Get returns the current watermarks for the given repo.
	// Returns zero-value Watermarks (empty strings) when no record exists yet.
	Get(ctx context.Context, repoID string) (Watermarks, error)

	// AdvanceProcessed sets SourceProcessedSHA to sha for the given repo.
	// It does not affect WikiPublishedSHA.
	AdvanceProcessed(ctx context.Context, repoID, sha string) error

	// AdvancePublished sets WikiPublishedSHA (and also SourceProcessedSHA) to
	// sha for the given repo. Called when a wiki PR merges.
	AdvancePublished(ctx context.Context, repoID, sha string) error

	// Reset sets both watermarks to the given sha. Used after a force-push
	// resets history or after a PR rejection to align both markers.
	Reset(ctx context.Context, repoID, sha string) error
}

// MemoryWatermarkStore is an in-memory [WatermarkStore] for tests and local
// development. It does not persist across process restarts.
type MemoryWatermarkStore struct {
	mu      sync.RWMutex
	entries map[string]Watermarks // key: repoID
}

// NewMemoryWatermarkStore returns an empty in-memory watermark store.
func NewMemoryWatermarkStore() *MemoryWatermarkStore {
	return &MemoryWatermarkStore{entries: make(map[string]Watermarks)}
}

// Compile-time interface check.
var _ WatermarkStore = (*MemoryWatermarkStore)(nil)

func (m *MemoryWatermarkStore) Get(_ context.Context, repoID string) (Watermarks, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.entries[repoID], nil
}

func (m *MemoryWatermarkStore) AdvanceProcessed(_ context.Context, repoID, sha string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	w := m.entries[repoID]
	w.SourceProcessedSHA = sha
	m.entries[repoID] = w
	return nil
}

func (m *MemoryWatermarkStore) AdvancePublished(_ context.Context, repoID, sha string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[repoID] = Watermarks{
		SourceProcessedSHA: sha,
		WikiPublishedSHA:   sha,
	}
	return nil
}

func (m *MemoryWatermarkStore) Reset(_ context.Context, repoID, sha string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[repoID] = Watermarks{
		SourceProcessedSHA: sha,
		WikiPublishedSHA:   sha,
	}
	return nil
}
