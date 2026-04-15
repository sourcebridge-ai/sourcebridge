// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package architecture

import (
	"strings"
	"testing"
)

// TestFullPipeline_DeterministicToEditAndBack tests the complete flow:
// deterministic diagram → structured model → edit → regenerate Mermaid → parse back
func TestFullPipeline_DeterministicToEditAndBack(t *testing.T) {
	// Step 1: Build a deterministic diagram result
	result := &DiagramResult{
		MermaidSource: "flowchart LR\n    api --> db",
		Modules: []ModuleNode{
			{Path: "internal/api", SymbolCount: 50, FileCount: 10,
				OutboundEdges: []EdgeInfo{
					{TargetPath: "internal/db", CallCount: 15},
					{TargetPath: "internal/worker", CallCount: 3},
				}},
			{Path: "internal/db", SymbolCount: 30, FileCount: 5},
			{Path: "internal/worker", SymbolCount: 20, FileCount: 4,
				OutboundEdges: []EdgeInfo{
					{TargetPath: "internal/db", CallCount: 8},
				}},
			{Path: "web/src", SymbolCount: 100, FileCount: 25,
				OutboundEdges: []EdgeInfo{
					{TargetPath: "internal/api", CallCount: 20},
				}},
		},
		Level:        "MODULE",
		TotalModules: 4,
		ShownModules: 4,
	}

	// Step 2: Convert to DiagramDocument
	doc := DocumentFromDiagramResult("repo-test", result)

	if len(doc.Nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(doc.Nodes))
	}
	if len(doc.Edges) != 4 {
		t.Fatalf("expected 4 edges, got %d", len(doc.Edges))
	}
	if doc.SourceKind != SourceDeterministic {
		t.Errorf("expected source_kind deterministic, got %s", doc.SourceKind)
	}

	// Step 3: Normalize in Improve mode
	normResult := Normalize(doc, ImportImprove)

	if normResult.NodesClassified < 1 {
		t.Error("expected at least 1 node classified")
	}
	if len(doc.Groups) < 1 {
		t.Error("expected groups to be inferred")
	}

	// Step 4: Generate Mermaid from the structured model
	mermaid := doc.GenerateMermaid()

	if !strings.Contains(mermaid, "flowchart") {
		t.Error("generated Mermaid should contain 'flowchart'")
	}
	if !strings.Contains(mermaid, "subgraph") {
		t.Error("generated Mermaid should contain subgraphs after normalization")
	}

	// Step 5: Simulate user edit — rename a node, add a node
	for i := range doc.Nodes {
		if doc.Nodes[i].Label == "internal/api" {
			doc.Nodes[i].Label = "API Server"
			doc.Nodes[i].Kind = NodeService
		}
	}
	doc.Nodes = append(doc.Nodes, DiagramNode{
		ID:         "new_cache",
		Label:      "Redis Cache",
		Kind:       NodeCache,
		Provenance: ProvenanceUserAdded,
	})
	doc.Edges = append(doc.Edges, DiagramEdge{
		ID:         "e-new",
		FromNodeID: SanitizeNodeID("internal/api"),
		ToNodeID:   "new_cache",
		Label:      "caches data",
		Kind:       EdgeRead,
		Provenance: ProvenanceUserAdded,
	})
	doc.SourceKind = SourceUserEdited

	// Step 6: Regenerate Mermaid after edit
	editedMermaid := doc.GenerateMermaid()

	if !strings.Contains(editedMermaid, "API Server") {
		t.Error("edited Mermaid should contain renamed 'API Server'")
	}
	if !strings.Contains(editedMermaid, "Redis Cache") {
		t.Error("edited Mermaid should contain added 'Redis Cache'")
	}
	if !strings.Contains(editedMermaid, "caches data") {
		t.Error("edited Mermaid should contain edge label 'caches data'")
	}

	// Step 7: JSON round-trip
	jsonStr, err := doc.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	restored, err := DocumentFromJSON(jsonStr)
	if err != nil {
		t.Fatalf("DocumentFromJSON failed: %v", err)
	}

	if len(restored.Nodes) != 5 {
		t.Errorf("expected 5 nodes after restore, got %d", len(restored.Nodes))
	}
	if restored.SourceKind != SourceUserEdited {
		t.Errorf("expected source_kind user_edited, got %s", restored.SourceKind)
	}

	t.Logf("Pipeline complete: %d nodes, %d edges, %d groups", len(doc.Nodes), len(doc.Edges), len(doc.Groups))
}

// TestFullPipeline_ImportMermaid tests: Mermaid import → normalize → export → re-import stability
func TestFullPipeline_ImportMermaid(t *testing.T) {
	// A realistic architecture diagram
	input := `flowchart TB
    subgraph clients["Clients"]
        web["Web UI"]
        cli["CLI"]
    end
    subgraph core["Core Platform"]
        api["API Server"]
        worker["Background Worker"]
    end
    subgraph storage["Data Layer"]
        db["PostgreSQL Database"]
        redis["Redis Cache"]
    end
    subgraph external["External"]
        llm["LLM Provider"]
        stripe["Stripe"]
    end
    web -->|"HTTP requests"| api
    cli -->|"HTTP requests"| api
    api --> worker
    api --> db
    api --> redis
    worker -->|"calls"| llm
    worker --> db
    api -->|"billing"| stripe
`

	// Step 1: Parse
	doc, err := ParseMermaid("test-repo", input)
	if err != nil {
		t.Fatalf("ParseMermaid failed: %v", err)
	}

	if len(doc.Groups) < 3 {
		t.Errorf("expected at least 3 groups, got %d", len(doc.Groups))
	}

	// Step 2: Normalize (Improve mode)
	normResult := Normalize(doc, ImportImprove)
	t.Logf("Normalization: classified=%d, labels_improved=%d, edges_removed=%d",
		normResult.NodesClassified, normResult.LabelsImproved, normResult.EdgesRemoved)

	// Verify node kind classification
	nodeKinds := make(map[string]NodeKind)
	for _, n := range doc.Nodes {
		nodeKinds[n.ID] = n.Kind
	}

	if nodeKinds["db"] != NodeStorage {
		t.Errorf("expected db to be storage, got %s", nodeKinds["db"])
	}
	if nodeKinds["redis"] != NodeCache {
		t.Errorf("expected redis to be cache, got %s", nodeKinds["redis"])
	}
	if nodeKinds["llm"] != NodeExternal {
		t.Errorf("expected llm to be external, got %s", nodeKinds["llm"])
	}

	// Step 3: Export to Mermaid
	exported := doc.GenerateMermaid()

	if !strings.Contains(exported, "flowchart") {
		t.Error("exported Mermaid should start with flowchart")
	}

	// Step 4: Re-import the exported Mermaid (round-trip stability)
	doc2, err := ParseMermaid("test-repo", exported)
	if err != nil {
		t.Fatalf("Round-trip ParseMermaid failed: %v", err)
	}

	// Should have same number of nodes and edges (or close)
	if absInt(len(doc2.Nodes)-len(doc.Nodes)) > 1 {
		t.Errorf("round-trip node count drift: %d → %d", len(doc.Nodes), len(doc2.Nodes))
	}
	if absInt(len(doc2.Edges)-len(doc.Edges)) > 1 {
		t.Errorf("round-trip edge count drift: %d → %d", len(doc.Edges), len(doc2.Edges))
	}

	t.Logf("Import pipeline: %d nodes, %d edges, %d groups → exported → re-imported: %d nodes, %d edges",
		len(doc.Nodes), len(doc.Edges), len(doc.Groups),
		len(doc2.Nodes), len(doc2.Edges))
}

// TestNormalizeModes_QualityComparison tests all three modes on the same input
func TestNormalizeModes_QualityComparison(t *testing.T) {
	makeDoc := func() *DiagramDocument {
		doc, _ := ParseMermaid("test", `flowchart LR
    A["Web Client"]
    B["API Gateway"]
    C["Auth Service"]
    D["User Service"]
    E["Order Service"]
    F["Payment Gateway"]
    G["PostgreSQL"]
    H["Redis"]
    I["Kafka"]
    J["Email Service"]
    K["S3 Storage"]
    L["Monitoring"]
    A --> B
    B --> C
    B --> D
    B --> E
    C --> G
    D --> G
    E --> G
    E --> F
    E -->|"events"| I
    I --> J
    D --> H
    E --> H
    J --> K
    L --> B
`)
		return doc
	}

	// Preserve mode
	docP := makeDoc()
	rP := Normalize(docP, ImportPreserve)
	t.Logf("Preserve: nodes=%d, edges=%d, groups=%d, classified=%d",
		len(docP.Nodes), len(docP.Edges), len(docP.Groups), rP.NodesClassified)

	// Improve mode
	docI := makeDoc()
	rI := Normalize(docI, ImportImprove)
	t.Logf("Improve: nodes=%d, edges=%d, groups=%d, classified=%d, labels_improved=%d",
		len(docI.Nodes), len(docI.Edges), len(docI.Groups), rI.NodesClassified, rI.LabelsImproved)

	// Simplify mode
	docS := makeDoc()
	rS := Normalize(docS, ImportSimplify)
	t.Logf("Simplify: nodes=%d, edges=%d, groups=%d, merged=%d",
		len(docS.Nodes), len(docS.Edges), len(docS.Groups), rS.NodesMerged)

	// Simplify should have fewer nodes
	if len(docS.Nodes) >= len(docP.Nodes) {
		t.Error("simplify should reduce node count compared to preserve")
	}

	// Improve should have groups
	if len(docI.Groups) == 0 {
		t.Error("improve mode should infer groups")
	}

	// All modes should produce valid Mermaid
	for _, doc := range []*DiagramDocument{docP, docI, docS} {
		m := doc.GenerateMermaid()
		if !strings.Contains(m, "flowchart") {
			t.Error("all modes should produce valid Mermaid")
		}
	}
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
