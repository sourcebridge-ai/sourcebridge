// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

//go:build !nogroovy

// Groovy-specific parser tests.  Gated behind !nogroovy because they call
// ParseFile on Groovy content; without the grammar registered the parse would
// return an empty FileResult (no LanguageConfig) and the assertions would fail
// rather than skip.  TestGrailsRoleFor (pure path classifier, no grammar) lives
// in grails_test.go with no build tag.

package indexer

import (
	"context"
	"testing"
)

// TestLanguageRegistry_Groovy asserts that the Groovy grammar is registered
// when built without -tags nogroovy.  The base TestLanguageRegistry in
// parser_test.go covers the 10 always-registered languages; this companion
// test covers the one grammar that is conditionally compiled.
func TestLanguageRegistry_Groovy(t *testing.T) {
	cfg := GetLanguageConfig("groovy")
	if cfg == nil {
		t.Error("expected groovy to be registered when built without -tags nogroovy")
	}
}

func TestParseGroovyFile(t *testing.T) {
	parser := NewParser()
	content := readFixture(t, "groovy/auth.groovy")

	result, err := parser.ParseFile(context.Background(), "groovy/auth.groovy", "groovy", content)
	if err != nil {
		t.Fatal(err)
	}

	funcNames := symbolNames(result.Symbols)
	t.Logf("Groovy symbols found: %v", funcNames)

	if len(result.Symbols) == 0 {
		t.Fatal("expected symbols in Groovy file")
	}
	if !containsName(funcNames, "HmacTokenVerifier") {
		t.Error("expected to find HmacTokenVerifier class")
	}
	if !containsName(funcNames, "TokenVerifier") {
		t.Error("expected to find TokenVerifier class (no-body case)")
	}
	if !containsName(funcNames, "verify") || !containsName(funcNames, "issue") {
		t.Error("expected to find verify and issue methods")
	}
	if !containsName(funcNames, "default_verifier") {
		t.Error("expected to find top-level default_verifier function (bare def)")
	}

	// Imports.  The grammar emits `groovy_import` with a `qualified_name`
	// payload whose string is the dotted path.
	if len(result.Imports) == 0 {
		t.Error("expected at least one import to be extracted from groovy_import nodes")
	}
}

func TestParseGroovyGradleFile(t *testing.T) {
	parser := NewParser()
	content := readFixture(t, "groovy/build.gradle")

	result, err := parser.ParseFile(context.Background(), "groovy/build.gradle", "groovy", content)
	if err != nil {
		t.Fatal(err)
	}

	// Gradle DSL is entirely juxt_function_call nodes with no enclosing class
	// or function — every call is at top-level.  The existing extractCalls
	// heuristic drops calls that have no enclosing function (it wires the
	// function-to-function call graph, not a file-content index), so we don't
	// assert on result.Calls here.
	//
	// What we DO assert: the parse itself succeeds without error, returns a
	// FileResult with the expected language, and the line count is non-zero.
	// That is enough to confirm the Groovy binding handles Gradle DSL.
	if result == nil {
		t.Fatal("expected non-nil FileResult for build.gradle")
	}
	if result.Language != "groovy" {
		t.Errorf("expected Language=groovy, got %q", result.Language)
	}
	if result.LineCount == 0 {
		t.Error("expected non-zero LineCount for build.gradle")
	}
	if len(result.Errors) != 0 {
		t.Errorf("unexpected parse errors: %v", result.Errors)
	}
}

func TestParseGroovySpecFile(t *testing.T) {
	parser := NewParser()
	content := readFixture(t, "groovy/FooSpec.groovy")

	result, err := parser.ParseFile(context.Background(), "groovy/FooSpec.groovy", "groovy", content)
	if err != nil {
		t.Fatal(err)
	}

	funcNames := symbolNames(result.Symbols)
	t.Logf("Groovy Spec symbols found: %v", funcNames)

	// The enclosing FooSpec class parses cleanly even though the
	// def-"string" method bodies inside emit ERROR nodes in the current
	// grammar.  Confirm the class is still captured so that Spec-level
	// browsing works.
	if !containsName(funcNames, "FooSpec") {
		t.Error("expected to find FooSpec class despite ERROR nodes for def-string methods")
	}
}
