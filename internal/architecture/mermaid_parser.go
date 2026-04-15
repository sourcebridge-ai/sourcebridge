// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package architecture

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ParseMermaid parses a Mermaid flowchart string into a DiagramDocument.
// It handles nodes, edges, subgraphs, and labels.
func ParseMermaid(repoID, mermaidSource string) (*DiagramDocument, error) {
	// Extract the Mermaid code block if wrapped in fences
	source := extractMermaidBlock(mermaidSource)
	if source == "" {
		return nil, fmt.Errorf("no valid Mermaid content found")
	}

	lines := strings.Split(source, "\n")

	doc := &DiagramDocument{
		ID:           fmt.Sprintf("imp-%s-%d", repoID, time.Now().Unix()),
		RepositoryID: repoID,
		SourceKind:   SourceImportedMermaid,
		ViewType:     ViewSystem,
		Title:        "Imported Diagram",
		Nodes:        make([]DiagramNode, 0),
		Edges:        make([]DiagramEdge, 0),
		Groups:       make([]DiagramGroup, 0),
		RawMermaid:   mermaidSource,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	// Track state
	nodeMap := make(map[string]*DiagramNode) // id → node
	edgeCount := 0
	var currentSubgraph string
	direction := "LR"

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}

		// flowchart direction
		if strings.HasPrefix(line, "flowchart ") || strings.HasPrefix(line, "graph ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				direction = parts[1]
			}
			continue
		}

		// direction within subgraph
		if strings.HasPrefix(line, "direction ") {
			continue
		}

		// Style directives
		if strings.HasPrefix(line, "style ") || strings.HasPrefix(line, "classDef ") || strings.HasPrefix(line, "class ") {
			continue
		}

		// Subgraph start
		if strings.HasPrefix(line, "subgraph ") {
			sg := parseSubgraph(line)
			if sg != nil {
				doc.Groups = append(doc.Groups, *sg)
				currentSubgraph = sg.ID
			}
			continue
		}

		// Subgraph end
		if line == "end" {
			currentSubgraph = ""
			continue
		}

		// Try to parse as edge (must check before node because edges contain node refs)
		if edge := parseEdge(line); edge != nil {
			edgeCount++
			edge.ID = fmt.Sprintf("e%d", edgeCount)
			edge.Provenance = ProvenanceImported
			doc.Edges = append(doc.Edges, *edge)

			// Ensure both nodes exist
			ensureNode(nodeMap, edge.FromNodeID, currentSubgraph)
			ensureNode(nodeMap, edge.ToNodeID, currentSubgraph)
			continue
		}

		// Try to parse as node definition
		if node := parseNodeDef(line); node != nil {
			node.Provenance = ProvenanceImported
			if currentSubgraph != "" {
				node.GroupID = currentSubgraph
			}
			if existing, ok := nodeMap[node.ID]; ok {
				// Update existing node with new info
				if node.Label != "" && node.Label != node.ID {
					existing.Label = node.Label
				}
				if node.Description != "" {
					existing.Description = node.Description
				}
				if currentSubgraph != "" {
					existing.GroupID = currentSubgraph
				}
			} else {
				nodeMap[node.ID] = node
			}
			continue
		}
	}

	// Convert node map to slice
	for _, n := range nodeMap {
		doc.Nodes = append(doc.Nodes, *n)
	}

	// Sort nodes for stable output
	sortNodes(doc.Nodes)

	doc.LayoutHints = &LayoutHints{Direction: direction}

	return doc, nil
}

// extractMermaidBlock strips markdown code fences if present.
func extractMermaidBlock(source string) string {
	source = strings.TrimSpace(source)

	// Try to find a fenced block
	reFence := regexp.MustCompile("(?s)```(?:mermaid)?\\s*\\n(.*?)\\n```")
	if m := reFence.FindStringSubmatch(source); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}

	// If it starts with flowchart/graph, treat the whole thing as Mermaid
	if strings.HasPrefix(source, "flowchart ") || strings.HasPrefix(source, "graph ") {
		return source
	}

	return source
}

// ── Edge patterns ───────────────────────────────────────────────────────────

// Matches: A --> B, A -->|"label"| B, A -- "label" --> B, A -.-> B, etc.
var edgePatterns = []*regexp.Regexp{
	// A -->|"label"| B  or  A -->|label| B
	regexp.MustCompile(`^(\S+)\s+--+>?\|"?([^"|]*)"?\|\s+(\S+)$`),
	// A -- "label" --> B
	regexp.MustCompile(`^(\S+)\s+--\s+"([^"]*)"\s+--+>\s+(\S+)$`),
	// A --> B  or  A -.-> B  or  A ==> B
	regexp.MustCompile(`^(\S+)\s+[-=.]+>+\s+(\S+)$`),
}

func parseEdge(line string) *DiagramEdge {
	for i, re := range edgePatterns {
		m := re.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		switch i {
		case 0: // labeled edge with |...|
			return &DiagramEdge{
				FromNodeID: cleanNodeRef(m[1]),
				ToNodeID:   cleanNodeRef(m[3]),
				Label:      m[2],
				Kind:       inferEdgeKind(m[2]),
			}
		case 1: // labeled edge with -- "..." -->
			return &DiagramEdge{
				FromNodeID: cleanNodeRef(m[1]),
				ToNodeID:   cleanNodeRef(m[3]),
				Label:      m[2],
				Kind:       inferEdgeKind(m[2]),
			}
		case 2: // unlabeled edge
			return &DiagramEdge{
				FromNodeID: cleanNodeRef(m[1]),
				ToNodeID:   cleanNodeRef(m[2]),
				Kind:       EdgeCall,
			}
		}
	}
	return nil
}

// cleanNodeRef strips shape delimiters that might be attached to a node reference.
func cleanNodeRef(s string) string {
	s = strings.TrimSpace(s)
	// Remove any trailing shape chars
	for _, suffix := range []string{")", "]", "}", ">", "|"} {
		s = strings.TrimRight(s, suffix)
	}
	for _, prefix := range []string{"(", "[", "{", "<"} {
		s = strings.TrimLeft(s, prefix)
	}
	return s
}

// ── Node patterns ───────────────────────────────────────────────────────────

// Matches: id["label"], id("label"), id{"label"}, id["label\nsubtitle"]
var nodeDefPattern = regexp.MustCompile(`^(\S+?)[\[\(\{>]+"?([^"\]\)\}]*)"?[\]\)\}<]+$`)

// Simple node: just an id on a line by itself (rare but valid)
var simpleNodePattern = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)$`)

func parseNodeDef(line string) *DiagramNode {
	if m := nodeDefPattern.FindStringSubmatch(line); m != nil {
		id := m[1]
		rawLabel := m[2]

		// Split on \n for label + description
		parts := strings.SplitN(rawLabel, "\\n", 2)
		label := strings.TrimSpace(parts[0])
		desc := ""
		if len(parts) > 1 {
			desc = strings.TrimSpace(parts[1])
		}

		// Unescape Mermaid quotes
		label = strings.ReplaceAll(label, "#quot;", `"`)

		if label == "" {
			label = id
		}

		return &DiagramNode{
			ID:          id,
			Label:       label,
			Kind:        NodeComponent,
			Description: desc,
		}
	}

	if m := simpleNodePattern.FindStringSubmatch(line); m != nil {
		return &DiagramNode{
			ID:    m[1],
			Label: m[1],
			Kind:  NodeComponent,
		}
	}

	return nil
}

// ── Subgraph parsing ────────────────────────────────────────────────────────

// Matches: subgraph id["label"], subgraph id[label], subgraph label
var subgraphPattern = regexp.MustCompile(`^subgraph\s+(\S+?)[\[\(]"?([^"\]\)]*)"?[\]\)]$`)
var subgraphSimplePattern = regexp.MustCompile(`^subgraph\s+(.+)$`)

func parseSubgraph(line string) *DiagramGroup {
	if m := subgraphPattern.FindStringSubmatch(line); m != nil {
		id := m[1]
		label := strings.TrimSpace(m[2])
		if label == "" || label == " " {
			label = id
		}
		return &DiagramGroup{
			ID:    id,
			Label: label,
			Kind:  inferGroupKind(label),
		}
	}

	if m := subgraphSimplePattern.FindStringSubmatch(line); m != nil {
		label := strings.TrimSpace(m[1])
		id := SanitizeNodeID(label)
		return &DiagramGroup{
			ID:    id,
			Label: label,
			Kind:  inferGroupKind(label),
		}
	}

	return nil
}

// ── Inference helpers ───────────────────────────────────────────────────────

func inferEdgeKind(label string) EdgeKind {
	lower := strings.ToLower(label)
	switch {
	case strings.Contains(lower, "http") || strings.Contains(lower, "request") || strings.Contains(lower, "api"):
		return EdgeRequest
	case strings.Contains(lower, "dispatch") || strings.Contains(lower, "queue") || strings.Contains(lower, "publish"):
		return EdgeDispatch
	case strings.Contains(lower, "read") || strings.Contains(lower, "query") || strings.Contains(lower, "fetch"):
		return EdgeRead
	case strings.Contains(lower, "write") || strings.Contains(lower, "store") || strings.Contains(lower, "persist") || strings.Contains(lower, "insert"):
		return EdgeWrite
	case strings.Contains(lower, "event") || strings.Contains(lower, "emit") || strings.Contains(lower, "notify"):
		return EdgeEvent
	case strings.Contains(lower, "call") || strings.Contains(lower, "invoke"):
		return EdgeCall
	default:
		return EdgeOther
	}
}

func inferGroupKind(label string) GroupKind {
	lower := strings.ToLower(label)
	switch {
	case strings.Contains(lower, "interface") || strings.Contains(lower, "client") || strings.Contains(lower, "ui") || strings.Contains(lower, "frontend"):
		return GroupInterfaces
	case strings.Contains(lower, "external") || strings.Contains(lower, "third") || strings.Contains(lower, "provider"):
		return GroupExternal
	case strings.Contains(lower, "storage") || strings.Contains(lower, "database") || strings.Contains(lower, "db") || strings.Contains(lower, "cache"):
		return GroupStorage
	case strings.Contains(lower, "worker") || strings.Contains(lower, "execution") || strings.Contains(lower, "processing"):
		return GroupExecution
	case strings.Contains(lower, "platform") || strings.Contains(lower, "core") || strings.Contains(lower, "service"):
		return GroupPlatform
	default:
		return GroupCustom
	}
}

func ensureNode(nodeMap map[string]*DiagramNode, id, currentSubgraph string) {
	if _, ok := nodeMap[id]; !ok {
		nodeMap[id] = &DiagramNode{
			ID:         id,
			Label:      id,
			Kind:       NodeComponent,
			GroupID:    currentSubgraph,
			Provenance: ProvenanceImported,
		}
	}
}

func sortNodes(nodes []DiagramNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
}
