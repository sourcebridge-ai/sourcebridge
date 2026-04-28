// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package skillcard_test

import (
	"strings"
	"testing"

	"github.com/sourcebridge/sourcebridge/internal/skillcard"
)

func TestDeriveWarnings_Empty(t *testing.T) {
	result := skillcard.DeriveWarnings(nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty result for nil input, got %v", result)
	}
}

func TestDeriveWarnings_HotPath(t *testing.T) {
	// Three symbols in the "auth" cluster; sym2 is called the most.
	edges := []skillcard.CallEdge{
		{CallerID: "sym1", CalleeID: "sym2"},
		{CallerID: "sym3", CalleeID: "sym2"},
		{CallerID: "sym4", CalleeID: "sym2"},
		{CallerID: "sym1", CalleeID: "sym3"},
	}
	symbols := map[string]skillcard.SymbolMeta{
		"sym1": {QualifiedName: "Auth.Login", Package: "auth", ClusterLabel: "auth"},
		"sym2": {QualifiedName: "Session.Validate", Package: "auth", ClusterLabel: "auth"},
		"sym3": {QualifiedName: "Token.Refresh", Package: "auth", ClusterLabel: "auth"},
		"sym4": {QualifiedName: "OAuthFlow.Begin", Package: "middleware", ClusterLabel: "auth"},
	}

	result := skillcard.DeriveWarnings(edges, symbols)
	authWarnings, ok := result["auth"]
	if !ok {
		t.Fatal("expected warnings for auth cluster")
	}

	var found bool
	for _, w := range authWarnings {
		if w.Kind == "hot-path" && strings.Contains(w.Detail, "Session.Validate") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected hot-path warning for Session.Validate; got %v", authWarnings)
	}
}

func TestDeriveWarnings_CrossPackageCallers(t *testing.T) {
	// sym_target is called from three different top-level packages.
	edges := []skillcard.CallEdge{
		{CallerID: "a1", CalleeID: "target"},
		{CallerID: "b1", CalleeID: "target"},
		{CallerID: "c1", CalleeID: "target"},
	}
	symbols := map[string]skillcard.SymbolMeta{
		"a1":     {QualifiedName: "api.Handler", Package: "api", ClusterLabel: "api"},
		"b1":     {QualifiedName: "billing.Processor", Package: "billing", ClusterLabel: "billing"},
		"c1":     {QualifiedName: "worker.Job", Package: "worker", ClusterLabel: "worker"},
		"target": {QualifiedName: "Store.Save", Package: "storage", ClusterLabel: "storage"},
	}

	result := skillcard.DeriveWarnings(edges, symbols)
	storageWarnings := result["storage"]

	var found bool
	for _, w := range storageWarnings {
		if w.Kind == "cross-package-callers" && strings.Contains(w.Detail, "Store.Save") {
			found = true
			if !strings.Contains(w.Detail, "api") || !strings.Contains(w.Detail, "billing") || !strings.Contains(w.Detail, "worker") {
				t.Errorf("cross-package warning should name all 3 packages; detail: %s", w.Detail)
			}
		}
	}
	if !found {
		t.Errorf("expected cross-package-callers warning for Store.Save; got %v", storageWarnings)
	}
}

func TestDeriveWarnings_CrossPackageBelowThreshold(t *testing.T) {
	// Only 2 caller packages — should NOT trigger cross-package warning.
	edges := []skillcard.CallEdge{
		{CallerID: "a1", CalleeID: "target"},
		{CallerID: "b1", CalleeID: "target"},
	}
	symbols := map[string]skillcard.SymbolMeta{
		"a1":     {QualifiedName: "api.Handler", Package: "api", ClusterLabel: "api"},
		"b1":     {QualifiedName: "billing.Processor", Package: "billing", ClusterLabel: "billing"},
		"target": {QualifiedName: "Store.Save", Package: "storage", ClusterLabel: "storage"},
	}

	result := skillcard.DeriveWarnings(edges, symbols)
	for _, w := range result["storage"] {
		if w.Kind == "cross-package-callers" {
			t.Errorf("should not emit cross-package warning for 2 packages; got: %v", w)
		}
	}
}

func TestDeriveWarnings_StableOrder(t *testing.T) {
	// Both warning types in same cluster — cross-package-callers should come first.
	edges := []skillcard.CallEdge{
		{CallerID: "a1", CalleeID: "target"},
		{CallerID: "b1", CalleeID: "target"},
		{CallerID: "c1", CalleeID: "target"},
		{CallerID: "a1", CalleeID: "other"},
		{CallerID: "a2", CalleeID: "other"},
	}
	symbols := map[string]skillcard.SymbolMeta{
		"a1":     {QualifiedName: "p1.Fn", Package: "p1", ClusterLabel: "core"},
		"a2":     {QualifiedName: "p1.Fn2", Package: "p1", ClusterLabel: "core"},
		"b1":     {QualifiedName: "p2.Fn", Package: "p2", ClusterLabel: "core"},
		"c1":     {QualifiedName: "p3.Fn", Package: "p3", ClusterLabel: "core"},
		"target": {QualifiedName: "Core.Handle", Package: "core", ClusterLabel: "core"},
		"other":  {QualifiedName: "Core.Process", Package: "core", ClusterLabel: "core"},
	}

	result := skillcard.DeriveWarnings(edges, symbols)
	warnings := result["core"]
	if len(warnings) < 2 {
		t.Fatalf("expected at least 2 warnings; got %d", len(warnings))
	}
	if warnings[0].Kind != "cross-package-callers" {
		t.Errorf("first warning should be cross-package-callers, got %q", warnings[0].Kind)
	}
}
