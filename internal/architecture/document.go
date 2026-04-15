// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package architecture

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ── Node kinds ──────────────────────────────────────────────────────────────

type NodeKind string

const (
	NodeActor     NodeKind = "actor"
	NodeInterface NodeKind = "interface"
	NodeService   NodeKind = "service"
	NodeWorker    NodeKind = "worker"
	NodeStorage   NodeKind = "storage"
	NodeExternal  NodeKind = "external"
	NodeComponent NodeKind = "component"
	NodeQueue     NodeKind = "queue"
	NodeCache     NodeKind = "cache"
)

// ── Edge kinds ──────────────────────────────────────────────────────────────

type EdgeKind string

const (
	EdgeRequest  EdgeKind = "request"
	EdgeDispatch EdgeKind = "dispatch"
	EdgeRead     EdgeKind = "read"
	EdgeWrite    EdgeKind = "write"
	EdgeCall     EdgeKind = "call"
	EdgeEvent    EdgeKind = "event"
	EdgeDepends  EdgeKind = "depends"
	EdgeOther    EdgeKind = "other"
)

// ── Group kinds ─────────────────────────────────────────────────────────────

type GroupKind string

const (
	GroupInterfaces GroupKind = "interfaces"
	GroupPlatform   GroupKind = "platform"
	GroupExecution  GroupKind = "execution"
	GroupStorage    GroupKind = "storage"
	GroupExternal   GroupKind = "external"
	GroupCustom     GroupKind = "custom"
)

// ── Source kinds ─────────────────────────────────────────────────────────────

type SourceKind string

const (
	SourceDeterministic SourceKind = "deterministic"
	SourceAIGenerated   SourceKind = "ai_generated"
	SourceImportedMermaid SourceKind = "imported_mermaid"
	SourceUserEdited    SourceKind = "user_edited"
)

// ── View types ──────────────────────────────────────────────────────────────

type ViewType string

const (
	ViewSystem    ViewType = "system"
	ViewExecution ViewType = "execution"
	ViewDetailed  ViewType = "detailed"
)

// ── Provenance ──────────────────────────────────────────────────────────────

type Provenance string

const (
	ProvenanceGraphBacked    Provenance = "graph_backed"
	ProvenanceUnderstanding  Provenance = "understanding_backed"
	ProvenanceImported       Provenance = "imported"
	ProvenanceUserAdded      Provenance = "user_added"
	ProvenanceInferredNorm   Provenance = "inferred_by_normalizer"
	ProvenanceInferredAI     Provenance = "inferred_by_ai"
)

// ── Core model ──────────────────────────────────────────────────────────────

// DiagramDocument is the structured source of truth for an architecture diagram.
type DiagramDocument struct {
	ID           string        `json:"id"`
	RepositoryID string        `json:"repository_id"`
	ArtifactID   string        `json:"artifact_id,omitempty"`
	SourceKind   SourceKind    `json:"source_kind"`
	ViewType     ViewType      `json:"view_type"`
	Title        string        `json:"title"`
	Summary      string        `json:"summary,omitempty"`
	Nodes        []DiagramNode `json:"nodes"`
	Edges        []DiagramEdge `json:"edges"`
	Groups       []DiagramGroup `json:"groups"`
	LayoutHints  *LayoutHints  `json:"layout_hints,omitempty"`
	RawMermaid   string        `json:"raw_mermaid_source,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// DiagramNode represents a component in the architecture diagram.
type DiagramNode struct {
	ID          string     `json:"id"`
	Label       string     `json:"label"`
	Kind        NodeKind   `json:"kind"`
	Description string     `json:"description,omitempty"`
	GroupID     string     `json:"group_id,omitempty"`
	SourceRefs  []string   `json:"source_refs,omitempty"`
	Provenance  Provenance `json:"provenance"`
	SymbolCount int        `json:"symbol_count,omitempty"`
	FileCount   int        `json:"file_count,omitempty"`
	PositionX   *float64   `json:"position_x,omitempty"`
	PositionY   *float64   `json:"position_y,omitempty"`
}

// DiagramEdge represents a relationship between two nodes.
type DiagramEdge struct {
	ID         string     `json:"id"`
	FromNodeID string     `json:"from_node_id"`
	ToNodeID   string     `json:"to_node_id"`
	Label      string     `json:"label,omitempty"`
	Kind       EdgeKind   `json:"kind"`
	Provenance Provenance `json:"provenance"`
	Confidence string     `json:"confidence,omitempty"`
	SourceRefs []string   `json:"source_refs,omitempty"`
	CallCount  int        `json:"call_count,omitempty"`
}

// DiagramGroup represents a logical grouping of nodes (rendered as subgraphs).
type DiagramGroup struct {
	ID    string    `json:"id"`
	Label string    `json:"label"`
	Kind  GroupKind `json:"kind"`
}

// LayoutHints provides optional rendering guidance.
type LayoutHints struct {
	Direction string `json:"direction,omitempty"` // "LR" or "TB"
}

// ── Conversion: DiagramResult → DiagramDocument ─────────────────────────────

// DocumentFromDiagramResult converts the deterministic DiagramResult into a
// structured DiagramDocument.
func DocumentFromDiagramResult(repoID string, result *DiagramResult) *DiagramDocument {
	now := time.Now().UTC()
	doc := &DiagramDocument{
		ID:           fmt.Sprintf("det-%s", repoID),
		RepositoryID: repoID,
		SourceKind:   SourceDeterministic,
		ViewType:     ViewDetailed,
		Title:        "Architecture Diagram",
		Nodes:        make([]DiagramNode, 0, len(result.Modules)),
		Edges:        make([]DiagramEdge, 0),
		Groups:       make([]DiagramGroup, 0),
		RawMermaid:   result.MermaidSource,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if len(result.Modules) > 15 {
		doc.LayoutHints = &LayoutHints{Direction: "TB"}
	} else {
		doc.LayoutHints = &LayoutHints{Direction: "LR"}
	}

	edgeID := 0
	for _, mod := range result.Modules {
		nodeID := SanitizeNodeID(mod.Path)
		doc.Nodes = append(doc.Nodes, DiagramNode{
			ID:          nodeID,
			Label:       mod.Path,
			Kind:        NodeComponent,
			Provenance:  ProvenanceGraphBacked,
			SymbolCount: mod.SymbolCount,
			FileCount:   mod.FileCount,
			SourceRefs:  []string{mod.Path},
		})

		for _, edge := range mod.OutboundEdges {
			edgeID++
			doc.Edges = append(doc.Edges, DiagramEdge{
				ID:         fmt.Sprintf("e%d", edgeID),
				FromNodeID: nodeID,
				ToNodeID:   SanitizeNodeID(edge.TargetPath),
				Kind:       EdgeCall,
				Provenance: ProvenanceGraphBacked,
				CallCount:  edge.CallCount,
			})
		}
	}

	return doc
}

// ── Mermaid generation from DiagramDocument ─────────────────────────────────

// GenerateMermaid produces Mermaid flowchart syntax from a DiagramDocument.
func (doc *DiagramDocument) GenerateMermaid() string {
	var b strings.Builder

	direction := "LR"
	if doc.LayoutHints != nil && doc.LayoutHints.Direction != "" {
		direction = doc.LayoutHints.Direction
	}
	if len(doc.Nodes) > 15 {
		direction = "TB"
	}

	b.WriteString(fmt.Sprintf("flowchart %s\n", direction))

	// Build group membership map
	groupNodes := make(map[string][]DiagramNode)
	ungrouped := make([]DiagramNode, 0)
	for _, n := range doc.Nodes {
		if n.GroupID != "" {
			groupNodes[n.GroupID] = append(groupNodes[n.GroupID], n)
		} else {
			ungrouped = append(ungrouped, n)
		}
	}

	// Emit grouped nodes as subgraphs
	groupMap := make(map[string]DiagramGroup)
	for _, g := range doc.Groups {
		groupMap[g.ID] = g
	}

	// Sort group IDs for stable output
	groupIDs := make([]string, 0, len(groupNodes))
	for gid := range groupNodes {
		groupIDs = append(groupIDs, gid)
	}
	sort.Strings(groupIDs)

	for _, gid := range groupIDs {
		nodes := groupNodes[gid]
		g, ok := groupMap[gid]
		label := gid
		if ok {
			label = g.Label
		}
		sgID := SanitizeNodeID(gid)
		b.WriteString(fmt.Sprintf("    subgraph %s[\"%s\"]\n", sgID, escapeLabel(label)))
		for _, n := range nodes {
			writeNodeDef(&b, n, "        ")
		}
		b.WriteString("    end\n\n")
	}

	// Emit ungrouped nodes
	for _, n := range ungrouped {
		writeNodeDef(&b, n, "    ")
	}
	if len(ungrouped) > 0 {
		b.WriteString("\n")
	}

	// Emit edges
	nodeSet := make(map[string]bool)
	for _, n := range doc.Nodes {
		nodeSet[n.ID] = true
	}
	for _, e := range doc.Edges {
		if !nodeSet[e.FromNodeID] || !nodeSet[e.ToNodeID] {
			continue
		}
		if e.Label != "" {
			b.WriteString(fmt.Sprintf("    %s -->|\"%s\"| %s\n", e.FromNodeID, escapeLabel(e.Label), e.ToNodeID))
		} else if e.CallCount > 1 {
			b.WriteString(fmt.Sprintf("    %s -->|\"%d calls\"| %s\n", e.FromNodeID, e.CallCount, e.ToNodeID))
		} else {
			b.WriteString(fmt.Sprintf("    %s --> %s\n", e.FromNodeID, e.ToNodeID))
		}
	}

	// Style external groups
	for _, g := range doc.Groups {
		if g.Kind == GroupExternal {
			sgID := SanitizeNodeID(g.ID)
			b.WriteString(fmt.Sprintf("\n    style %s fill:none,stroke:#475569,stroke-dasharray:5 5\n", sgID))
		}
	}

	return b.String()
}

func writeNodeDef(b *strings.Builder, n DiagramNode, indent string) {
	label := escapeLabel(n.Label)
	shape := nodeShape(n.Kind)

	extra := ""
	if n.SymbolCount > 0 {
		extra = fmt.Sprintf("\\n%d symbols", n.SymbolCount)
	}
	if n.Description != "" && n.SymbolCount == 0 {
		extra = fmt.Sprintf("\\n%s", escapeLabel(n.Description))
	}

	b.WriteString(fmt.Sprintf("%s%s%s\"%s%s\"%s\n", indent, n.ID, shape[0:1], label, extra, shape[1:2]))
}

// nodeShape returns the Mermaid delimiters for different node kinds.
func nodeShape(kind NodeKind) string {
	switch kind {
	case NodeStorage, NodeCache:
		return "[]" // cylinder-ish (database)
	case NodeExternal:
		return "()" // stadium shape
	case NodeActor:
		return ">]" // flag shape
	case NodeQueue:
		return "[]" // cylinder
	default:
		return "[]" // rectangle (default)
	}
}

// ── JSON serialization ──────────────────────────────────────────────────────

// ToJSON serializes the document to JSON.
func (doc *DiagramDocument) ToJSON() (string, error) {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DocumentFromJSON deserializes a DiagramDocument from JSON.
func DocumentFromJSON(data string) (*DiagramDocument, error) {
	var doc DiagramDocument
	if err := json.Unmarshal([]byte(data), &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}
