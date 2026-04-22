// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package search

import (
	"testing"
)

func TestFuse_BasicRRF(t *testing.T) {
	mk := func(adapter string, ids ...string) AdapterResult {
		cands := make([]*Candidate, len(ids))
		for i, id := range ids {
			cands[i] = &Candidate{
				EntityID:   id,
				EntityType: "symbol",
				AdapterID:  adapter,
				Rank:       i,
				RawScore:   float64(len(ids) - i),
			}
		}
		return AdapterResult{AdapterID: adapter, Candidates: cands}
	}
	hydrate := func(c *Candidate) *Result {
		return &Result{
			EntityType: c.EntityType,
			EntityID:   c.EntityID,
			RepoID:     "r1",
			Title:      c.EntityID,
		}
	}

	// FTS: A, B, C
	// Vector: C, D, A
	// A is ranked 0 (fts) and 2 (vector) → 1/60 + 1/62 ≈ 0.03279
	// C is ranked 2 (fts) and 0 (vector) → 1/62 + 1/60 ≈ same = tie with A
	// B at 1/61; D at 1/61
	out := fuse([]AdapterResult{
		mk("fts", "A", "B", "C"),
		mk("vector", "C", "D", "A"),
	}, hydrate)

	if len(out) != 4 {
		t.Fatalf("expected 4 results, got %d", len(out))
	}
	// Top two should be A and C (tie broken by lexical entity_id).
	if out[0].EntityID != "A" || out[1].EntityID != "C" {
		t.Errorf("unexpected top two: %q %q", out[0].EntityID, out[1].EntityID)
	}
	// Signals set on A.
	if out[0].Signals.Lexical == 0 {
		t.Errorf("A should have lexical signal, got 0")
	}
	if out[0].Signals.Semantic == 0 {
		t.Errorf("A should have semantic signal, got 0")
	}
}

func TestFuse_StableTieBreak(t *testing.T) {
	ar := AdapterResult{
		AdapterID: "fts",
		Candidates: []*Candidate{
			{EntityID: "z", EntityType: "symbol", AdapterID: "fts", Rank: 0, RawScore: 1.0},
			{EntityID: "a", EntityType: "symbol", AdapterID: "fts", Rank: 0, RawScore: 1.0},
		},
	}
	hydrate := func(c *Candidate) *Result {
		return &Result{EntityType: c.EntityType, EntityID: c.EntityID, RepoID: "r1"}
	}
	out := fuse([]AdapterResult{ar}, hydrate)
	if len(out) != 2 {
		t.Fatalf("want 2, got %d", len(out))
	}
	// Both tied on score — tie-break is entity_id ASC, so 'a' first.
	if out[0].EntityID != "a" {
		t.Errorf("tie-break broken: first = %q, want 'a'", out[0].EntityID)
	}
}

func TestFuse_IgnoresUnavailable(t *testing.T) {
	hydrate := func(c *Candidate) *Result {
		return &Result{EntityType: c.EntityType, EntityID: c.EntityID, RepoID: "r1"}
	}
	out := fuse([]AdapterResult{
		{AdapterID: "fts", Unavailable: true},
		{AdapterID: "vector", Candidates: []*Candidate{
			{EntityID: "x", EntityType: "symbol", AdapterID: "vector", Rank: 0},
		}},
	}, hydrate)
	if len(out) != 1 || out[0].EntityID != "x" {
		t.Fatalf("unexpected: %+v", out)
	}
}

func TestRRFAcrossRanked_DuplicatesDedupedPerRepo(t *testing.T) {
	// Two repos, one shared entity id but different repo_id → must
	// not dedupe cross-repo.
	a := []*Result{{RepoID: "r1", EntityType: "symbol", EntityID: "a"}}
	b := []*Result{{RepoID: "r2", EntityType: "symbol", EntityID: "a"}}
	out := rrfAcrossRanked([][]*Result{a, b})
	if len(out) != 2 {
		t.Fatalf("expected 2 distinct (repo, id) results, got %d", len(out))
	}
}
