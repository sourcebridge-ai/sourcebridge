// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package architecture

import (
	"testing"
)

func TestNormalizePreserveMode(t *testing.T) {
	doc := &DiagramDocument{
		Nodes: []DiagramNode{
			{ID: "api", Label: "API Server", Kind: NodeComponent},
			{ID: "db", Label: "PostgreSQL Database", Kind: NodeComponent},
		},
		Edges: []DiagramEdge{
			{ID: "e1", FromNodeID: "api", ToNodeID: "db"},
			{ID: "e2", FromNodeID: "api", ToNodeID: "db"}, // duplicate
		},
	}

	result := Normalize(doc, ImportPreserve)

	// Should deduplicate edges
	if result.EdgesRemoved != 1 {
		t.Errorf("expected 1 edge removed, got %d", result.EdgesRemoved)
	}
	if len(doc.Edges) != 1 {
		t.Errorf("expected 1 edge after dedup, got %d", len(doc.Edges))
	}

	// Should classify node kinds
	if result.NodesClassified < 1 {
		t.Error("expected at least 1 node classified")
	}

	// DB node should be classified as storage
	for _, n := range doc.Nodes {
		if n.ID == "db" && n.Kind != NodeStorage {
			t.Errorf("expected db node kind 'storage', got '%s'", n.Kind)
		}
	}
}

func TestNormalizeImproveMode(t *testing.T) {
	doc := &DiagramDocument{
		Nodes: []DiagramNode{
			{ID: "web", Label: "Web UI", Kind: NodeComponent},
			{ID: "api", Label: "API Server", Kind: NodeComponent},
			{ID: "worker", Label: "Background Worker", Kind: NodeComponent},
			{ID: "db", Label: "PostgreSQL", Kind: NodeComponent},
			{ID: "llm", Label: "LLM Provider", Kind: NodeComponent},
		},
		Edges: []DiagramEdge{
			{ID: "e1", FromNodeID: "web", ToNodeID: "api", Label: "primary flow"},
			{ID: "e2", FromNodeID: "api", ToNodeID: "worker"},
			{ID: "e3", FromNodeID: "api", ToNodeID: "db"},
			{ID: "e4", FromNodeID: "worker", ToNodeID: "llm"},
			{ID: "e5", FromNodeID: "worker", ToNodeID: "db"},
		},
	}

	result := Normalize(doc, ImportImprove)

	// Should classify all 5 nodes
	if result.NodesClassified < 3 {
		t.Errorf("expected at least 3 nodes classified, got %d", result.NodesClassified)
	}

	// Should improve the generic "primary flow" label
	if result.LabelsImproved < 1 {
		t.Errorf("expected at least 1 label improved, got %d", result.LabelsImproved)
	}

	// Should infer groups since none exist
	if result.GroupsAdded < 1 {
		t.Errorf("expected at least 1 group added, got %d", result.GroupsAdded)
	}
	if len(doc.Groups) < 1 {
		t.Errorf("expected groups to be created, got %d", len(doc.Groups))
	}

	// Check that nodes have group assignments
	grouped := 0
	for _, n := range doc.Nodes {
		if n.GroupID != "" {
			grouped++
		}
	}
	if grouped < 3 {
		t.Errorf("expected at least 3 nodes grouped, got %d", grouped)
	}
}

func TestNormalizeSimplifyMode(t *testing.T) {
	// Create a document with more than 10 nodes
	doc := &DiagramDocument{
		Nodes: make([]DiagramNode, 0),
		Edges: make([]DiagramEdge, 0),
	}

	// Add 15 nodes with varying connectivity
	for i := 0; i < 15; i++ {
		doc.Nodes = append(doc.Nodes, DiagramNode{
			ID:    nodeID(i),
			Label: nodeName(i),
			Kind:  NodeComponent,
		})
	}

	// Add edges — first 5 nodes are highly connected, rest are sparse
	edgeID := 0
	for i := 0; i < 5; i++ {
		for j := i + 1; j < 5; j++ {
			edgeID++
			doc.Edges = append(doc.Edges, DiagramEdge{
				ID: eid(edgeID), FromNodeID: nodeID(i), ToNodeID: nodeID(j),
			})
		}
	}
	// Sparse connections for remaining nodes
	for i := 5; i < 15; i++ {
		edgeID++
		doc.Edges = append(doc.Edges, DiagramEdge{
			ID: eid(edgeID), FromNodeID: nodeID(i), ToNodeID: nodeID(0),
		})
	}

	result := Normalize(doc, ImportSimplify)

	if result.NodesMerged < 1 {
		t.Errorf("expected nodes to be merged in simplify mode, got %d merged", result.NodesMerged)
	}
	if len(doc.Nodes) > 10 {
		t.Errorf("expected at most 10 nodes after simplify, got %d", len(doc.Nodes))
	}
	if doc.ViewType != ViewSystem {
		t.Errorf("expected view type 'system' after simplify, got '%s'", doc.ViewType)
	}
}

func TestDeduplicateReciprocalEdges(t *testing.T) {
	doc := &DiagramDocument{
		Nodes: []DiagramNode{
			{ID: "a", Label: "A"}, {ID: "b", Label: "B"},
		},
		Edges: []DiagramEdge{
			{ID: "e1", FromNodeID: "a", ToNodeID: "b", Label: "calls"},
			{ID: "e2", FromNodeID: "b", ToNodeID: "a", Label: "calls"},
		},
	}

	removed := removeReciprocalEdges(doc)
	if removed != 1 {
		t.Errorf("expected 1 reciprocal edge removed, got %d", removed)
	}
	if len(doc.Edges) != 1 {
		t.Errorf("expected 1 edge remaining, got %d", len(doc.Edges))
	}
}

func TestInferNodeKind(t *testing.T) {
	tests := []struct {
		label    string
		expected NodeKind
	}{
		{"PostgreSQL Database", NodeStorage},
		{"Redis Cache", NodeCache},
		{"RabbitMQ Queue", NodeQueue},
		{"Background Worker", NodeWorker},
		{"REST API Server", NodeService},
		{"Web UI Dashboard", NodeInterface},
		{"Admin User", NodeActor},
		{"Stripe Provider", NodeExternal},
		{"Something Else", NodeComponent},
	}

	for _, tt := range tests {
		got := inferNodeKind(tt.label, "")
		if got != tt.expected {
			t.Errorf("inferNodeKind(%q) = %s, want %s", tt.label, got, tt.expected)
		}
	}
}

// helpers

func nodeID(i int) string  { return string(rune('a' + i)) }
func nodeName(i int) string { return string(rune('A'+i)) + " Module" }
func eid(i int) string     { return "e" + string(rune('0'+i)) }
