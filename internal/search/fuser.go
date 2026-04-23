// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package search

import "sort"

// RRFK is the reciprocal-rank-fusion constant. k=60 is the
// "industry-default" dampener that biases against the long tail of any
// single backend dominating the top of the fused list. Plan §Target
// Architecture / §Fuse.
const RRFK = 60.0

// fuse combines per-adapter candidate lists into one ranked list of
// Results. The fused score is the standard reciprocal-rank-fusion
// score: Σ 1/(k + rank_i) across contributing adapters.
//
// `hydrate` resolves an entity ID → *Result with metadata. The fuser
// does not touch storage; each call is O(n) over the combined
// candidate set.
//
// The function also records which adapter contributed to each result
// in Result.Signals so downstream presentation can render signal
// chips without re-running the adapters.
func fuse(
	adapterResults []AdapterResult,
	hydrate func(*Candidate) *Result,
) []*Result {
	// by entity, accumulate scored signals
	type agg struct {
		fused float64
		res   *Result
	}
	aggMap := make(map[string]*agg)

	for _, ar := range adapterResults {
		if ar.Unavailable {
			continue
		}
		for _, c := range ar.Candidates {
			key := c.EntityType + ":" + c.EntityID
			a, ok := aggMap[key]
			if !ok {
				res := hydrate(c)
				if res == nil {
					continue
				}
				a = &agg{res: res}
				aggMap[key] = a
			}
			rrf := 1.0 / (RRFK + float64(c.Rank))
			a.fused += rrf
			// Record the adapter's signal on the result. We normalise
			// different adapter scores into the 0-1 band.
			switch c.AdapterID {
			case "exact":
				if c.RawScore > a.res.Signals.Exact {
					a.res.Signals.Exact = c.RawScore
				}
			case "fts":
				if c.RawScore > a.res.Signals.Lexical {
					a.res.Signals.Lexical = c.RawScore
				}
			case "vector":
				if c.RawScore > a.res.Signals.Semantic {
					a.res.Signals.Semantic = c.RawScore
				}
			case "graph":
				if c.RawScore > a.res.Signals.Graph {
					a.res.Signals.Graph = c.RawScore
				}
			}
		}
	}

	out := make([]*Result, 0, len(aggMap))
	for _, a := range aggMap {
		a.res.Score = a.fused
		out = append(out, a.res)
	}

	// Primary: fused score DESC. Tie-breaker: (repo, entity_type,
	// entity_id) ASC — deterministic for pagination stability per
	// plan §Pagination & Stable Ordering.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		if out[i].RepoID != out[j].RepoID {
			return out[i].RepoID < out[j].RepoID
		}
		if out[i].EntityType != out[j].EntityType {
			return out[i].EntityType < out[j].EntityType
		}
		return out[i].EntityID < out[j].EntityID
	})
	return out
}

// rrfAcrossRanked is a convenience helper for the GraphQL adapter's
// cross-repo merge: given N per-repo ranked lists, return one list in
// second-stage RRF order using each source list's local rank only.
// Duplicate (repo, entity_type, entity_id) keys are deduped; the max
// fused score wins.
func rrfAcrossRanked(perRepo [][]*Result) []*Result {
	type agg struct {
		fused float64
		res   *Result
	}
	aggMap := make(map[string]*agg)
	for _, list := range perRepo {
		for rank, r := range list {
			key := r.RepoID + ":" + r.EntityType + ":" + r.EntityID
			a, ok := aggMap[key]
			if !ok {
				a = &agg{res: r}
				aggMap[key] = a
			}
			a.fused += 1.0 / (RRFK + float64(rank))
		}
	}
	out := make([]*Result, 0, len(aggMap))
	for _, a := range aggMap {
		a.res.Score = a.fused
		out = append(out, a.res)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		if out[i].RepoID != out[j].RepoID {
			return out[i].RepoID < out[j].RepoID
		}
		if out[i].EntityType != out[j].EntityType {
			return out[i].EntityType < out[j].EntityType
		}
		return out[i].EntityID < out[j].EntityID
	})
	return out
}
