// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package skillcard

import (
	"fmt"
	"sort"
	"strings"
)

// CallEdge represents a caller → callee relationship.
// Using a local type keeps this package free of graph-package imports.
type CallEdge struct {
	CallerID string
	CalleeID string
}

// SymbolMeta carries the minimum per-symbol metadata needed for warning derivation.
type SymbolMeta struct {
	// QualifiedName is the fully qualified symbol name used in warning Detail text.
	QualifiedName string
	// Package is the top-level package path (e.g. "auth", "api/rest").
	Package string
	// ClusterLabel is the label of the cluster this symbol belongs to.
	ClusterLabel string
}

// DeriveWarnings computes call-graph-derived warnings for all clusters at once.
// It returns a map from cluster label → []Warning.
//
// Graph data is provided as two slim slices — no graph-package types are imported.
// Symbols not present in the meta map are silently skipped so the function
// degrades gracefully when the index is partial.
//
// Warning kinds produced in v1:
//   - "cross-package-callers": a symbol whose callers span 3+ distinct top-level packages.
//   - "hot-path": the highest-in-degree symbol in each cluster.
//
// Extension: add new Kind values and populate them here without touching render.go.
func DeriveWarnings(edges []CallEdge, symbols map[string]SymbolMeta) map[string][]Warning {
	if len(edges) == 0 || len(symbols) == 0 {
		return nil
	}

	// Build caller-package sets per callee symbol.
	// callerPackages[calleeID] = set of package paths of all direct callers.
	callerPackages := make(map[string]map[string]struct{})
	// Build raw in-degree per symbol.
	inDegree := make(map[string]int)

	for _, e := range edges {
		callerMeta, callerOK := symbols[e.CallerID]
		callee, calleeOK := symbols[e.CalleeID]
		_ = callee
		if !calleeOK {
			continue
		}
		inDegree[e.CalleeID]++
		if !callerOK {
			continue
		}
		if callerPackages[e.CalleeID] == nil {
			callerPackages[e.CalleeID] = make(map[string]struct{})
		}
		callerPackages[e.CalleeID][topLevelPackage(callerMeta.Package)] = struct{}{}
	}

	// Group symbols by cluster label for hot-path detection.
	// clusterInDegree[label][symbolID] = inDegree count
	clusterInDegree := make(map[string]map[string]int)
	for symID, meta := range symbols {
		label := meta.ClusterLabel
		if label == "" {
			continue
		}
		if clusterInDegree[label] == nil {
			clusterInDegree[label] = make(map[string]int)
		}
		clusterInDegree[label][symID] = inDegree[symID]
	}

	result := make(map[string][]Warning)

	// Cross-package-callers warnings (threshold: 3+ distinct packages).
	const crossPackageThreshold = 3
	for calleeID, pkgSet := range callerPackages {
		if len(pkgSet) < crossPackageThreshold {
			continue
		}
		callee, ok := symbols[calleeID]
		if !ok {
			continue
		}
		pkgs := make([]string, 0, len(pkgSet))
		for p := range pkgSet {
			pkgs = append(pkgs, p)
		}
		sort.Strings(pkgs)
		label := callee.ClusterLabel
		detail := fmt.Sprintf(
			"%s has callers in %s — coordinate changes across all of them.",
			callee.QualifiedName,
			joinPackageList(pkgs),
		)
		result[label] = append(result[label], Warning{
			Symbol: callee.QualifiedName,
			Kind:   "cross-package-callers",
			Detail: detail,
		})
	}

	// Hot-path warnings: the highest-in-degree symbol per cluster.
	for label, degrees := range clusterInDegree {
		bestID := ""
		bestDeg := 0
		for symID, deg := range degrees {
			if deg > bestDeg || (deg == bestDeg && symID < bestID) {
				bestDeg = deg
				bestID = symID
			}
		}
		if bestID == "" || bestDeg == 0 {
			continue
		}
		sym, ok := symbols[bestID]
		if !ok {
			continue
		}
		detail := fmt.Sprintf(
			"%s is on the hot path (highest in-degree in cluster, %d callers).",
			sym.QualifiedName,
			bestDeg,
		)
		result[label] = append(result[label], Warning{
			Symbol: sym.QualifiedName,
			Kind:   "hot-path",
			Detail: detail,
		})
	}

	// Stable ordering within each cluster: cross-package-callers first, then hot-path.
	for label := range result {
		sort.Slice(result[label], func(i, j int) bool {
			ki, kj := result[label][i].Kind, result[label][j].Kind
			if ki != kj {
				// cross-package-callers sorts before hot-path
				return ki < kj
			}
			return result[label][i].Symbol < result[label][j].Symbol
		})
	}

	return result
}

// topLevelPackage extracts the first path segment from a package path.
// "auth/middleware" → "auth", "api/rest" → "api", "billing" → "billing".
func topLevelPackage(pkg string) string {
	if idx := strings.IndexByte(pkg, '/'); idx >= 0 {
		return pkg[:idx]
	}
	return pkg
}

// joinPackageList formats a sorted slice of package names as a human-readable
// list: ["a", "b", "c"] → "a, b, and c".
func joinPackageList(pkgs []string) string {
	switch len(pkgs) {
	case 0:
		return ""
	case 1:
		return pkgs[0]
	case 2:
		return pkgs[0] + " and " + pkgs[1]
	default:
		return strings.Join(pkgs[:len(pkgs)-1], ", ") + ", and " + pkgs[len(pkgs)-1]
	}
}
