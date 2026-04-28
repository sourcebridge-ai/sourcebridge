// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package clustering

import (
	"testing"

	"github.com/sourcebridge/sourcebridge/internal/graph"
)

func TestSizeDistribution_Empty(t *testing.T) {
	min, max, p50, p95 := SizeDistribution(nil)
	if min != 0 || max != 0 || p50 != 0 || p95 != 0 {
		t.Errorf("expected all zeros, got min=%d max=%d p50=%d p95=%d", min, max, p50, p95)
	}
}

func TestSizeDistribution_Single(t *testing.T) {
	clusters := []Cluster{{Size: 7}}
	min, max, p50, p95 := SizeDistribution(clusters)
	if min != 7 || max != 7 {
		t.Errorf("single cluster: expected min=max=7, got min=%d max=%d", min, max)
	}
	_ = p50
	_ = p95
}

func TestSizeDistribution_Multiple(t *testing.T) {
	clusters := []Cluster{
		{Size: 1},
		{Size: 2},
		{Size: 3},
		{Size: 4},
		{Size: 10},
	}
	min, max, p50, _ := SizeDistribution(clusters)
	if min != 1 {
		t.Errorf("expected min=1, got %d", min)
	}
	if max != 10 {
		t.Errorf("expected max=10, got %d", max)
	}
	if p50 != 3 {
		t.Errorf("expected p50=3, got %d", p50)
	}
}

func TestComputeModularity_Empty(t *testing.T) {
	q := ComputeModularity(nil, nil)
	if q != 0 {
		t.Errorf("expected Q=0 for empty graph, got %f", q)
	}
}

func TestComputeModularity_TwoComponents(t *testing.T) {
	// Two perfectly separated components should yield high Q.
	edges := []graph.CallEdge{
		{CallerID: "a", CalleeID: "b"},
		{CallerID: "b", CalleeID: "a"},
		{CallerID: "c", CalleeID: "d"},
		{CallerID: "d", CalleeID: "c"},
	}
	labels := map[string]string{"a": "1", "b": "1", "c": "2", "d": "2"}
	q := ComputeModularity(edges, labels)
	if q <= 0 {
		t.Errorf("expected Q > 0 for clean partition, got %f", q)
	}
}
