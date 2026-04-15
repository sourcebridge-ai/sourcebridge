// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package architecture

import (
	"fmt"
	"sort"
	"strings"
)

// ImportMode controls how aggressively normalization modifies the diagram.
type ImportMode string

const (
	ImportPreserve ImportMode = "preserve"
	ImportImprove  ImportMode = "improve"
	ImportSimplify ImportMode = "simplify"
)

// NormalizationResult tracks what the normalizer changed.
type NormalizationResult struct {
	NodesMerged    int      `json:"nodes_merged"`
	LabelsImproved int      `json:"labels_improved"`
	EdgesRemoved   int      `json:"edges_removed"`
	GroupsAdded    int      `json:"groups_added"`
	NodesClassified int     `json:"nodes_classified"`
	Diagnostics    []string `json:"diagnostics"`
}

// Normalize applies heuristic normalization to a DiagramDocument.
// In Preserve mode, it only classifies nodes/edges and removes duplicates.
// In Improve mode, it also improves labels, merges weak nodes, and infers groups.
// In Simplify mode, it aggressively collapses to a system-level view.
func Normalize(doc *DiagramDocument, mode ImportMode) *NormalizationResult {
	result := &NormalizationResult{
		Diagnostics: make([]string, 0),
	}

	// Step 1: Always — deduplicate edges
	result.EdgesRemoved += deduplicateEdges(doc)

	// Step 2: Always — classify node kinds from labels
	result.NodesClassified += classifyNodeKinds(doc)

	// Step 3: Always — classify edge kinds from labels
	classifyEdgeKinds(doc)

	if mode == ImportPreserve {
		return result
	}

	// Step 4: Improve — improve generic labels
	result.LabelsImproved += improveGenericLabels(doc)

	// Step 5: Improve — remove low-signal reciprocal edges
	result.EdgesRemoved += removeReciprocalEdges(doc)

	// Step 6: Improve — infer groups if none exist
	if len(doc.Groups) == 0 {
		result.GroupsAdded += inferGroups(doc)
	}

	if mode == ImportImprove {
		return result
	}

	// Step 7: Simplify — merge weakly distinct nodes
	result.NodesMerged += mergeWeakNodes(doc, 10) // target max 10 nodes for system view

	// Step 8: Simplify — remove isolated nodes
	removeIsolatedNodes(doc)

	doc.ViewType = ViewSystem
	return result
}

// ── Step 1: Deduplicate edges ───────────────────────────────────────────────

func deduplicateEdges(doc *DiagramDocument) int {
	seen := make(map[string]bool)
	unique := make([]DiagramEdge, 0, len(doc.Edges))
	removed := 0

	for _, e := range doc.Edges {
		key := fmt.Sprintf("%s->%s:%s", e.FromNodeID, e.ToNodeID, e.Label)
		if seen[key] {
			removed++
			continue
		}
		seen[key] = true
		unique = append(unique, e)
	}

	doc.Edges = unique
	return removed
}

// ── Step 2: Classify node kinds ─────────────────────────────────────────────

func classifyNodeKinds(doc *DiagramDocument) int {
	classified := 0
	for i := range doc.Nodes {
		n := &doc.Nodes[i]
		if n.Kind != NodeComponent {
			continue // already classified
		}
		newKind := inferNodeKind(n.Label, n.Description)
		if newKind != NodeComponent {
			n.Kind = newKind
			n.Provenance = ProvenanceInferredNorm
			classified++
		}
	}
	return classified
}

func inferNodeKind(label, description string) NodeKind {
	lower := strings.ToLower(label + " " + description)
	switch {
	case containsAny(lower, "database", "postgres", "mysql", "sqlite", "surreal", "mongo", "s3", "storage engine"):
		return NodeStorage
	case containsAny(lower, "redis", "memcache", "cache layer", "cache"):
		return NodeCache
	case containsAny(lower, "queue", "rabbitmq", "kafka", "nats", "pubsub", "sqs"):
		return NodeQueue
	case containsAny(lower, "worker", "processor", "job runner", "background"):
		return NodeWorker
	case containsAny(lower, "api server", "http server", "rest api", "graphql api", "grpc server"):
		return NodeService
	case containsAny(lower, "web ui", "frontend", "dashboard", "client", "browser", "mobile app"):
		return NodeInterface
	case containsAny(lower, "user", "actor", "admin", "developer"):
		return NodeActor
	case containsAny(lower, "external", "third-party", "provider", "llm", "stripe", "github", "smtp", "cloud"):
		return NodeExternal
	default:
		return NodeComponent
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// ── Step 3: Classify edge kinds ─────────────────────────────────────────────

func classifyEdgeKinds(doc *DiagramDocument) {
	for i := range doc.Edges {
		e := &doc.Edges[i]
		if e.Kind != EdgeOther && e.Kind != "" {
			continue
		}
		if e.Label != "" {
			e.Kind = inferEdgeKind(e.Label)
		} else {
			e.Kind = inferEdgeKindFromNodes(doc, e)
		}
	}
}

func inferEdgeKindFromNodes(doc *DiagramDocument, e *DiagramEdge) EdgeKind {
	var toNode *DiagramNode
	for j := range doc.Nodes {
		if doc.Nodes[j].ID == e.ToNodeID {
			toNode = &doc.Nodes[j]
			break
		}
	}
	if toNode == nil {
		return EdgeCall
	}

	switch toNode.Kind {
	case NodeStorage:
		return EdgeRead
	case NodeQueue:
		return EdgeDispatch
	case NodeCache:
		return EdgeRead
	case NodeExternal:
		return EdgeRequest
	default:
		return EdgeCall
	}
}

// ── Step 4: Improve generic labels ──────────────────────────────────────────

var genericLabels = map[string]bool{
	"primary flow":   true,
	"major flow":     true,
	"interaction":    true,
	"connection":     true,
	"link":           true,
	"data flow":      true,
	"communicates":   true,
	"sends":          true,
	"receives":       true,
	"processes":      true,
	"main":           true,
	"flow":           true,
}

func improveGenericLabels(doc *DiagramDocument) int {
	improved := 0
	for i := range doc.Edges {
		e := &doc.Edges[i]
		if e.Label == "" {
			continue
		}
		if genericLabels[strings.ToLower(e.Label)] {
			better := suggestBetterEdgeLabel(doc, e)
			if better != e.Label {
				e.Label = better
				improved++
			}
		}
	}
	return improved
}

func suggestBetterEdgeLabel(doc *DiagramDocument, e *DiagramEdge) string {
	var toNode *DiagramNode
	for j := range doc.Nodes {
		if doc.Nodes[j].ID == e.ToNodeID {
			toNode = &doc.Nodes[j]
			break
		}
	}
	if toNode == nil {
		return e.Label
	}

	switch toNode.Kind {
	case NodeStorage:
		return "reads/writes data"
	case NodeQueue:
		return "dispatches jobs"
	case NodeCache:
		return "caches data"
	case NodeExternal:
		return fmt.Sprintf("calls %s", strings.ToLower(toNode.Label))
	case NodeWorker:
		return "delegates work"
	case NodeService:
		return "sends requests"
	case NodeInterface:
		return "serves UI"
	default:
		return fmt.Sprintf("calls %s", strings.ToLower(toNode.Label))
	}
}

// ── Step 5: Remove reciprocal edges ─────────────────────────────────────────

func removeReciprocalEdges(doc *DiagramDocument) int {
	edgeSet := make(map[string]int) // "A->B" → index
	for i, e := range doc.Edges {
		key := fmt.Sprintf("%s->%s", e.FromNodeID, e.ToNodeID)
		edgeSet[key] = i
	}

	removed := 0
	toRemove := make(map[int]bool)

	for _, e := range doc.Edges {
		reverseKey := fmt.Sprintf("%s->%s", e.ToNodeID, e.FromNodeID)
		if reverseIdx, ok := edgeSet[reverseKey]; ok {
			reverseEdge := doc.Edges[reverseIdx]
			// If both edges have the same or no label, remove the reverse
			if reverseEdge.Label == e.Label || reverseEdge.Label == "" {
				if !toRemove[reverseIdx] {
					forwardKey := fmt.Sprintf("%s->%s", e.FromNodeID, e.ToNodeID)
					forwardIdx := edgeSet[forwardKey]
					if !toRemove[forwardIdx] {
						toRemove[reverseIdx] = true
					}
				}
			}
		}
	}

	if len(toRemove) == 0 {
		return 0
	}

	filtered := make([]DiagramEdge, 0, len(doc.Edges)-len(toRemove))
	for i, e := range doc.Edges {
		if !toRemove[i] {
			filtered = append(filtered, e)
		} else {
			removed++
		}
	}
	doc.Edges = filtered
	return removed
}

// ── Step 6: Infer groups ────────────────────────────────────────────────────

func inferGroups(doc *DiagramDocument) int {
	groupMap := map[GroupKind]*DiagramGroup{
		GroupInterfaces: {ID: "g_interfaces", Label: "Interfaces", Kind: GroupInterfaces},
		GroupPlatform:   {ID: "g_platform", Label: "Core Platform", Kind: GroupPlatform},
		GroupExecution:  {ID: "g_execution", Label: "Execution", Kind: GroupExecution},
		GroupStorage:    {ID: "g_storage", Label: "Storage", Kind: GroupStorage},
		GroupExternal:   {ID: "g_external", Label: "External Systems", Kind: GroupExternal},
	}

	// Map node kinds to group kinds
	kindToGroup := map[NodeKind]GroupKind{
		NodeInterface: GroupInterfaces,
		NodeActor:     GroupInterfaces,
		NodeService:   GroupPlatform,
		NodeComponent: GroupPlatform,
		NodeWorker:    GroupExecution,
		NodeQueue:     GroupExecution,
		NodeStorage:   GroupStorage,
		NodeCache:     GroupStorage,
		NodeExternal:  GroupExternal,
	}

	usedGroups := make(map[GroupKind]bool)
	for i := range doc.Nodes {
		n := &doc.Nodes[i]
		if n.GroupID != "" {
			continue // already grouped
		}
		if gk, ok := kindToGroup[n.Kind]; ok {
			n.GroupID = groupMap[gk].ID
			n.Provenance = ProvenanceInferredNorm
			usedGroups[gk] = true
		}
	}

	// Only add groups that actually have members
	added := 0
	for gk, used := range usedGroups {
		if used {
			doc.Groups = append(doc.Groups, *groupMap[gk])
			added++
		}
	}

	return added
}

// ── Step 7: Merge weak nodes ────────────────────────────────────────────────

func mergeWeakNodes(doc *DiagramDocument, targetMax int) int {
	if len(doc.Nodes) <= targetMax {
		return 0
	}

	// Score nodes by connectivity
	type nodeScore struct {
		id    string
		score int
	}
	scores := make(map[string]int)
	for _, e := range doc.Edges {
		scores[e.FromNodeID]++
		scores[e.ToNodeID]++
	}

	scoreList := make([]nodeScore, 0, len(doc.Nodes))
	for _, n := range doc.Nodes {
		scoreList = append(scoreList, nodeScore{id: n.ID, score: scores[n.ID]})
	}
	sort.Slice(scoreList, func(i, j int) bool {
		return scoreList[i].score < scoreList[j].score
	})

	// Merge lowest-scoring nodes into their group peers or a catch-all
	merged := 0
	mergeMap := make(map[string]string) // old ID → surviving ID

	toMerge := len(doc.Nodes) - targetMax
	for _, ns := range scoreList {
		if merged >= toMerge {
			break
		}
		// Find the node
		var node *DiagramNode
		for j := range doc.Nodes {
			if doc.Nodes[j].ID == ns.id {
				node = &doc.Nodes[j]
				break
			}
		}
		if node == nil {
			continue
		}

		// Find a neighbor to merge into (the most-connected neighbor)
		bestNeighbor := ""
		bestScore := -1
		for _, e := range doc.Edges {
			neighbor := ""
			if e.FromNodeID == ns.id {
				neighbor = e.ToNodeID
			} else if e.ToNodeID == ns.id {
				neighbor = e.FromNodeID
			}
			if neighbor != "" && scores[neighbor] > bestScore {
				if _, alreadyMerged := mergeMap[neighbor]; !alreadyMerged {
					bestNeighbor = neighbor
					bestScore = scores[neighbor]
				}
			}
		}

		if bestNeighbor != "" {
			mergeMap[ns.id] = bestNeighbor
			merged++
		}
	}

	if merged == 0 {
		return 0
	}

	// Apply merges to edges
	for i := range doc.Edges {
		if target, ok := mergeMap[doc.Edges[i].FromNodeID]; ok {
			doc.Edges[i].FromNodeID = target
		}
		if target, ok := mergeMap[doc.Edges[i].ToNodeID]; ok {
			doc.Edges[i].ToNodeID = target
		}
	}

	// Remove self-edges created by merges
	filtered := make([]DiagramEdge, 0, len(doc.Edges))
	for _, e := range doc.Edges {
		if e.FromNodeID != e.ToNodeID {
			filtered = append(filtered, e)
		}
	}
	doc.Edges = filtered

	// Remove merged nodes
	surviving := make([]DiagramNode, 0, len(doc.Nodes)-merged)
	for _, n := range doc.Nodes {
		if _, wasMerged := mergeMap[n.ID]; !wasMerged {
			surviving = append(surviving, n)
		}
	}
	doc.Nodes = surviving

	// Deduplicate edges again after merge
	deduplicateEdges(doc)

	return merged
}

// ── Step 8: Remove isolated nodes ───────────────────────────────────────────

func removeIsolatedNodes(doc *DiagramDocument) {
	connected := make(map[string]bool)
	for _, e := range doc.Edges {
		connected[e.FromNodeID] = true
		connected[e.ToNodeID] = true
	}

	filtered := make([]DiagramNode, 0, len(doc.Nodes))
	for _, n := range doc.Nodes {
		if connected[n.ID] {
			filtered = append(filtered, n)
		}
	}
	doc.Nodes = filtered
}
