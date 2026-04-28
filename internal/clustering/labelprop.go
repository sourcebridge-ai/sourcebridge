// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors
//
// Implements label propagation as described in:
//   Raghavan, U. N., Albert, R., & Kumara, S. (2007).
//   "Near linear time algorithm to detect community structures
//    in large-scale networks." Physical Review E, 76(3).

package clustering

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"sort"

	"github.com/sourcebridge/sourcebridge/internal/graph"
)

const (
	// maxIterations caps LPA to prevent infinite loops on poorly-connected
	// or adversarially-structured graphs.
	maxIterations = 50
	// relabelThreshold is the fraction of nodes that must change label in a
	// single iteration for LPA to continue. Below this rate the community
	// structure is considered stable.
	relabelThreshold = 0.01
)

// LPAResult is returned by RunLPA.
type LPAResult struct {
	// Labels maps symbol ID → cluster label (a symbol ID that is the
	// canonical representative for the cluster).
	Labels map[string]string
	// Iterations is the number of full sweeps executed.
	Iterations int
	// Partial is true when the run terminated at maxIterations rather than
	// converging via the relabel-rate check.
	Partial bool
}

// RunLPA executes label-propagation clustering on the provided call graph.
// The seed is derived from (repoID, commitSHA) to guarantee that identical
// inputs produce identical outputs across re-runs.
//
// edges is the full set of directed caller→callee relationships. nodeIDs is
// the deduplicated set of all node identifiers that should appear in the
// result (including isolates — nodes with no edges). Isolates each become
// their own single-node cluster.
func RunLPA(edges []graph.CallEdge, nodeIDs []string, seed [32]byte) LPAResult {
	if len(nodeIDs) == 0 {
		return LPAResult{Labels: make(map[string]string)}
	}

	// ---- Build adjacency lists (undirected) --------------------------------
	// LPA on a directed graph treats each edge as undirected for label
	// propagation purposes: neighbours of v = callers(v) ∪ callees(v).
	neighbors := make(map[string][]string, len(nodeIDs))
	for _, id := range nodeIDs {
		neighbors[id] = nil // ensure every node appears, even isolates
	}
	for _, e := range edges {
		if _, ok := neighbors[e.CallerID]; ok {
			neighbors[e.CallerID] = append(neighbors[e.CallerID], e.CalleeID)
		}
		if _, ok := neighbors[e.CalleeID]; ok {
			neighbors[e.CalleeID] = append(neighbors[e.CalleeID], e.CallerID)
		}
	}

	// ---- Initialise labels -------------------------------------------------
	// Each node starts as its own label.
	labels := make(map[string]string, len(nodeIDs))
	for _, id := range nodeIDs {
		labels[id] = id
	}

	// ---- Seeded RNG --------------------------------------------------------
	// Convert the first 8 bytes of the seed to an int64 for rand.New.
	seedInt := int64(binary.LittleEndian.Uint64(seed[:8]))
	rng := rand.New(rand.NewSource(seedInt)) //nolint:gosec // seeded, non-crypto RNG is intentional

	// ---- Shuffle order for the update sweep --------------------------------
	// A deterministic random order prevents the propagation from being biased
	// by the order nodeIDs was built.
	order := make([]string, len(nodeIDs))
	copy(order, nodeIDs)
	rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })

	// ---- Iteration loop ----------------------------------------------------
	iterations := 0
	partial := false
	for iter := 0; iter < maxIterations; iter++ {
		iterations++
		changed := 0

		// Reshuffle each iteration for better convergence.
		rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })

		for _, v := range order {
			nbrs := neighbors[v]
			if len(nbrs) == 0 {
				// Isolate — keep its own label.
				continue
			}
			// Count neighbour labels.
			freq := make(map[string]int, len(nbrs))
			for _, nb := range nbrs {
				freq[labels[nb]]++
			}
			best := dominantLabel(freq, rng)
			if best != labels[v] {
				labels[v] = best
				changed++
			}
		}

		rate := float64(changed) / float64(len(nodeIDs))
		if rate < relabelThreshold {
			break
		}
		if iter == maxIterations-1 {
			partial = true
		}
	}

	return LPAResult{
		Labels:     labels,
		Iterations: iterations,
		Partial:    partial,
	}
}

// dominantLabel returns the label with the highest frequency among the
// provided counts. Ties are broken by choosing the lexicographically smallest
// label among the tied candidates, then applying the seeded RNG for a random
// tie-break among equal-frequency, equal-lex candidates.
//
// The lex-first step makes the tie-breaking deterministic across
// single-iteration runs; the RNG handles the pathological case where multiple
// labels share the exact same value.
func dominantLabel(freq map[string]int, rng *rand.Rand) string {
	maxFreq := 0
	for _, f := range freq {
		if f > maxFreq {
			maxFreq = f
		}
	}
	// Collect all labels at maxFreq and sort for determinism.
	candidates := make([]string, 0, 4)
	for lbl, f := range freq {
		if f == maxFreq {
			candidates = append(candidates, lbl)
		}
	}
	sort.Strings(candidates)
	if len(candidates) == 1 {
		return candidates[0]
	}
	return candidates[rng.Intn(len(candidates))]
}

// BuildSeed constructs a deterministic 32-byte seed from a (repoID, commitSHA)
// pair. Re-indexing the same commit always produces the same clusters.
func BuildSeed(repoID, commitSHA string) [32]byte {
	h := sha256.New()
	h.Write([]byte(repoID))
	h.Write([]byte(commitSHA))
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}
