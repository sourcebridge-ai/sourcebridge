// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package clustering

import (
	"math"
	"sort"

	"github.com/sourcebridge/sourcebridge/internal/graph"
)

// ComputeModularity calculates the Newman–Girvan modularity Q for the given
// partition. Q ranges from −1 to 1; values above ~0.3 indicate meaningful
// community structure.
//
// Formula: Q = (1/2m) * Σ_ij [ A_ij − k_i*k_j/(2m) ] * δ(c_i, c_j)
//
// where m is the number of edges, A_ij is 1 when an edge exists between i
// and j, k_i is the degree of node i, and δ(c_i, c_j) is 1 when i and j
// are in the same cluster.
//
// We treat the directed graph as undirected for Q calculation (each directed
// edge contributes degree 1 to both endpoints), consistent with LPA's
// undirected neighbourhood propagation.
func ComputeModularity(edges []graph.CallEdge, labels map[string]string) float64 {
	if len(edges) == 0 || len(labels) == 0 {
		return 0
	}

	// Build undirected degree map and edge set.
	degree := make(map[string]int, len(labels))
	type edgeKey struct{ a, b string }
	edgeSet := make(map[edgeKey]bool, len(edges))

	for _, e := range edges {
		// Only count edges where both endpoints are in the partition.
		_, hasA := labels[e.CallerID]
		_, hasB := labels[e.CalleeID]
		if !hasA || !hasB {
			continue
		}
		// Undirected: normalise key so a ≤ b.
		a, b := e.CallerID, e.CalleeID
		if a > b {
			a, b = b, a
		}
		k := edgeKey{a, b}
		if !edgeSet[k] {
			edgeSet[k] = true
			degree[a]++
			degree[b]++
		}
	}

	m := float64(len(edgeSet))
	if m == 0 {
		return 0
	}
	twoM := 2 * m

	// Sum the modularity contribution of each within-cluster edge pair.
	q := 0.0
	for k := range edgeSet {
		a, b := k.a, k.b
		if labels[a] != labels[b] {
			continue
		}
		// A_ij = 1; subtract expected k_i * k_j / (2m).
		q += 1.0 - float64(degree[a])*float64(degree[b])/twoM
	}
	// Include the diagonal correction: each node i contributes
	// (0 − k_i^2/(2m)) * δ(c_i,c_i) = −k_i^2/(2m).
	for id, d := range degree {
		if _, ok := labels[id]; ok {
			q -= float64(d) * float64(d) / twoM
		}
	}

	return math.Round(q/m*100) / 100 // round to 2dp
}

// SizeDistribution computes min, max, p50, and p95 cluster sizes from the
// provided clusters slice. Returns zeroes for an empty input.
func SizeDistribution(clusters []Cluster) (min, max, p50, p95 int) {
	if len(clusters) == 0 {
		return
	}
	sizes := make([]int, len(clusters))
	for i, c := range clusters {
		sizes[i] = c.Size
	}
	sort.Ints(sizes)

	min = sizes[0]
	max = sizes[len(sizes)-1]
	p50 = percentile(sizes, 50)
	p95 = percentile(sizes, 95)
	return
}

// percentile returns the p-th percentile value from a sorted slice.
func percentile(sorted []int, p int) int {
	if len(sorted) == 0 {
		return 0
	}
	idx := (p * len(sorted)) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
