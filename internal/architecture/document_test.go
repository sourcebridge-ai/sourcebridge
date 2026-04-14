// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package architecture

import (
	"strings"
	"testing"
)

func TestParseMermaidBasicFlowchart(t *testing.T) {
	input := `flowchart LR
    A["API Server"]
    B["Database"]
    A --> B
`
	doc, err := ParseMermaid("test-repo", input)
	if err != nil {
		t.Fatalf("ParseMermaid failed: %v", err)
	}

	if len(doc.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(doc.Nodes))
	}
	if len(doc.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(doc.Edges))
	}

	// Check labels were parsed
	nodeMap := make(map[string]string)
	for _, n := range doc.Nodes {
		nodeMap[n.ID] = n.Label
	}
	if nodeMap["A"] != "API Server" {
		t.Errorf("expected node A label 'API Server', got '%s'", nodeMap["A"])
	}
	if nodeMap["B"] != "Database" {
		t.Errorf("expected node B label 'Database', got '%s'", nodeMap["B"])
	}
}

func TestParseMermaidWithSubgraphs(t *testing.T) {
	input := `flowchart TB
    subgraph core["Core Platform"]
        api["API Server"]
        worker["Worker"]
    end
    subgraph external["External"]
        llm["LLM Provider"]
    end
    api --> worker
    worker --> llm
`
	doc, err := ParseMermaid("test-repo", input)
	if err != nil {
		t.Fatalf("ParseMermaid failed: %v", err)
	}

	if len(doc.Groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(doc.Groups))
	}
	if len(doc.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(doc.Edges))
	}

	// Check group assignment
	for _, n := range doc.Nodes {
		if n.ID == "api" && n.GroupID != "core" {
			t.Errorf("expected api node in group 'core', got '%s'", n.GroupID)
		}
		if n.ID == "llm" && n.GroupID != "external" {
			t.Errorf("expected llm node in group 'external', got '%s'", n.GroupID)
		}
	}
}

func TestParseMermaidWithLabels(t *testing.T) {
	input := `flowchart LR
    A["Client"]
    B["Server"]
    A -->|"HTTP requests"| B
`
	doc, err := ParseMermaid("test-repo", input)
	if err != nil {
		t.Fatalf("ParseMermaid failed: %v", err)
	}

	if len(doc.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(doc.Edges))
	}
	if doc.Edges[0].Label != "HTTP requests" {
		t.Errorf("expected edge label 'HTTP requests', got '%s'", doc.Edges[0].Label)
	}
	if doc.Edges[0].Kind != EdgeRequest {
		t.Errorf("expected edge kind 'request', got '%s'", doc.Edges[0].Kind)
	}
}

func TestParseMermaidFencedBlock(t *testing.T) {
	input := "```mermaid\nflowchart LR\n    A --> B\n```"

	doc, err := ParseMermaid("test-repo", input)
	if err != nil {
		t.Fatalf("ParseMermaid failed: %v", err)
	}
	if len(doc.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(doc.Nodes))
	}
}

func TestDocumentFromDiagramResult(t *testing.T) {
	result := &DiagramResult{
		MermaidSource: "flowchart LR\n    A --> B",
		Modules: []ModuleNode{
			{Path: "internal/api", SymbolCount: 50, FileCount: 10, OutboundEdges: []EdgeInfo{{TargetPath: "internal/db", CallCount: 5}}},
			{Path: "internal/db", SymbolCount: 30, FileCount: 5},
		},
		Level:        "MODULE",
		TotalModules: 2,
		ShownModules: 2,
	}

	doc := DocumentFromDiagramResult("repo-1", result)

	if len(doc.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(doc.Nodes))
	}
	if len(doc.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(doc.Edges))
	}
	if doc.SourceKind != SourceDeterministic {
		t.Errorf("expected source_kind 'deterministic', got '%s'", doc.SourceKind)
	}
}

func TestGenerateMermaidRoundTrip(t *testing.T) {
	// Build a document manually
	doc := &DiagramDocument{
		Nodes: []DiagramNode{
			{ID: "api", Label: "API Server", Kind: NodeService},
			{ID: "db", Label: "Database", Kind: NodeStorage, GroupID: "storage_group"},
			{ID: "web", Label: "Web UI", Kind: NodeInterface},
		},
		Edges: []DiagramEdge{
			{ID: "e1", FromNodeID: "api", ToNodeID: "db", Label: "queries", Kind: EdgeRead},
			{ID: "e2", FromNodeID: "web", ToNodeID: "api", Kind: EdgeRequest},
		},
		Groups: []DiagramGroup{
			{ID: "storage_group", Label: "Storage Layer", Kind: GroupStorage},
		},
		LayoutHints: &LayoutHints{Direction: "LR"},
	}

	mermaid := doc.GenerateMermaid()

	// Verify key elements are present
	if !strings.Contains(mermaid, "flowchart LR") {
		t.Error("expected 'flowchart LR' in output")
	}
	if !strings.Contains(mermaid, "API Server") {
		t.Error("expected 'API Server' label in output")
	}
	if !strings.Contains(mermaid, "Database") {
		t.Error("expected 'Database' label in output")
	}
	if !strings.Contains(mermaid, "queries") {
		t.Error("expected edge label 'queries' in output")
	}
	if !strings.Contains(mermaid, "subgraph") {
		t.Error("expected subgraph in output")
	}
	if !strings.Contains(mermaid, "Storage Layer") {
		t.Error("expected 'Storage Layer' subgraph label in output")
	}

	// Parse the generated Mermaid back
	reparsed, err := ParseMermaid("test", mermaid)
	if err != nil {
		t.Fatalf("round-trip ParseMermaid failed: %v", err)
	}
	if len(reparsed.Nodes) != 3 {
		t.Errorf("round-trip: expected 3 nodes, got %d", len(reparsed.Nodes))
	}
	if len(reparsed.Edges) != 2 {
		t.Errorf("round-trip: expected 2 edges, got %d", len(reparsed.Edges))
	}
}

func TestJSONRoundTrip(t *testing.T) {
	doc := &DiagramDocument{
		ID:           "test-1",
		RepositoryID: "repo-1",
		SourceKind:   SourceDeterministic,
		ViewType:     ViewSystem,
		Title:        "Test Diagram",
		Nodes: []DiagramNode{
			{ID: "a", Label: "Node A", Kind: NodeService, Provenance: ProvenanceGraphBacked},
		},
		Edges: []DiagramEdge{
			{ID: "e1", FromNodeID: "a", ToNodeID: "b", Kind: EdgeCall, Provenance: ProvenanceGraphBacked},
		},
	}

	jsonStr, err := doc.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	reparsed, err := DocumentFromJSON(jsonStr)
	if err != nil {
		t.Fatalf("DocumentFromJSON failed: %v", err)
	}

	if reparsed.ID != doc.ID {
		t.Errorf("expected ID '%s', got '%s'", doc.ID, reparsed.ID)
	}
	if len(reparsed.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(reparsed.Nodes))
	}
	if reparsed.Nodes[0].Provenance != ProvenanceGraphBacked {
		t.Errorf("expected provenance 'graph_backed', got '%s'", reparsed.Nodes[0].Provenance)
	}
}
