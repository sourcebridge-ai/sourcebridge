// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package graphql

import (
	"os"
	"path/filepath"
	"testing"

	commonv1 "github.com/sourcebridge/sourcebridge/gen/go/common/v1"
	graphstore "github.com/sourcebridge/sourcebridge/internal/graph"
)

func TestIsGitURL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		// Scheme-prefixed URLs are unambiguously remote.
		{"https://github.com/user/repo.git", true},
		{"http://github.com/user/repo", true},
		{"git://github.com/user/repo", true},
		{"git@github.com:user/repo.git", true},
		// Host-shaped .git suffix: hostname (contains dot) before first slash.
		{"github.com/user/repo.git", true},
		// Local paths — must be false regardless of .git suffix (codex M fix).
		{"/home/user/project", false},
		{"./relative/path", false},
		{"/home/user/repos/myrepo.git", false}, // absolute local bare repo
		{"./myrepo.git", false},                // explicit relative local bare repo
		{"../myrepo.git", false},               // parent-relative local bare repo
		// Bare name without hostname prefix: ambiguous, treated as local because
		// there is no hostname-shaped prefix before the first slash.
		// Pre-Slice-7 graphql accepted this (myrepo.git → true), but the
		// consolidated pathutil.IsGitURL correctly rejects it to avoid
		// misclassifying local bare-repo directories as remote clone targets.
		{"myrepo.git", false},
		{"", false},
	}
	for _, tc := range tests {
		got := isGitURL(tc.input)
		if got != tc.want {
			t.Errorf("isGitURL(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestSafeJoinPath(t *testing.T) {
	root := t.TempDir()

	// Valid relative path
	got, err := safeJoinPath(root, "src/main.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(root, "src", "main.go")
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}

	// Strip leading ./
	got, err = safeJoinPath(root, "./src/main.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}

	// Reject absolute paths
	_, err = safeJoinPath(root, "/etc/passwd")
	if err == nil {
		t.Error("expected error for absolute path")
	}

	// Reject path traversal
	_, err = safeJoinPath(root, "../../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestReadSourceFile(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "src")
	os.MkdirAll(subdir, 0o755)
	os.WriteFile(filepath.Join(subdir, "main.go"), []byte("package main\n"), 0o644)

	content, err := readSourceFile(root, "src/main.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "package main\n" {
		t.Errorf("got %q, want %q", content, "package main\n")
	}

	// Missing file
	_, err = readSourceFile(root, "nonexistent.go")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestExtractSymbolContext(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5"

	// Normal range
	got := extractSymbolContext(content, 2, 4)
	if got != "line2\nline3\nline4" {
		t.Errorf("got %q, want %q", got, "line2\nline3\nline4")
	}

	// Start before line 1 — returns "" (CA-151: aligned with qa/REST contract)
	got = extractSymbolContext(content, 0, 2)
	if got != "" {
		t.Errorf("got %q, want %q (non-positive start must return empty)", got, "")
	}

	// End beyond file length
	got = extractSymbolContext(content, 4, 100)
	if got != "line4\nline5" {
		t.Errorf("got %q, want %q", got, "line4\nline5")
	}

	// Start beyond file length
	got = extractSymbolContext(content, 100, 200)
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}

	// Single line
	got = extractSymbolContext(content, 3, 3)
	if got != "line3" {
		t.Errorf("got %q, want %q", got, "line3")
	}
}

// TestExtractSymbolContext_NonPositiveStart_ReturnsEmpty pins the post-CA-151
// contract: a non-positive startLine must return "" (not the entire file from
// line 1). Both callers in schema.resolvers.go pass sym.StartLine which the
// indexer guarantees is >= 1, so this path should never fire in production.
func TestExtractSymbolContext_NonPositiveStart_ReturnsEmpty(t *testing.T) {
	content := "a\nb\nc\nd\ne"
	for _, start := range []int{0, -1, -100} {
		if got := extractSymbolContext(content, start, 5); got != "" {
			t.Errorf("extractSymbolContext(content, %d, 5) = %q, want empty string", start, got)
		}
	}
}

func TestLanguageToProto(t *testing.T) {
	tests := []struct {
		input string
		want  commonv1.Language
	}{
		{"GO", commonv1.Language_LANGUAGE_GO},
		{"go", commonv1.Language_LANGUAGE_GO},
		{"Python", commonv1.Language_LANGUAGE_PYTHON},
		{"TYPESCRIPT", commonv1.Language_LANGUAGE_TYPESCRIPT},
		{"unknown", commonv1.Language_LANGUAGE_UNSPECIFIED},
		{"", commonv1.Language_LANGUAGE_UNSPECIFIED},
	}
	for _, tc := range tests {
		got := languageToProto(tc.input)
		if got != tc.want {
			t.Errorf("languageToProto(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestDeriveLanguage(t *testing.T) {
	tests := []struct {
		path string
		want commonv1.Language
	}{
		{"main.go", commonv1.Language_LANGUAGE_GO},
		{"app.py", commonv1.Language_LANGUAGE_PYTHON},
		{"index.ts", commonv1.Language_LANGUAGE_TYPESCRIPT},
		{"component.tsx", commonv1.Language_LANGUAGE_TYPESCRIPT},
		{"script.js", commonv1.Language_LANGUAGE_JAVASCRIPT},
		{"App.jsx", commonv1.Language_LANGUAGE_JAVASCRIPT},
		{"Service.java", commonv1.Language_LANGUAGE_JAVA},
		{"lib.rs", commonv1.Language_LANGUAGE_RUST},
		{"Program.cs", commonv1.Language_LANGUAGE_CSHARP},
		{"main.cpp", commonv1.Language_LANGUAGE_CPP},
		{"utils.h", commonv1.Language_LANGUAGE_CPP},
		{"app.rb", commonv1.Language_LANGUAGE_RUBY},
		{"index.php", commonv1.Language_LANGUAGE_PHP},
		{"readme.md", commonv1.Language_LANGUAGE_UNSPECIFIED},
		{"", commonv1.Language_LANGUAGE_UNSPECIFIED},
	}
	for _, tc := range tests {
		got := deriveLanguage(tc.path)
		if got != tc.want {
			t.Errorf("deriveLanguage(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestConfidenceFromFloat(t *testing.T) {
	tests := []struct {
		input float64
		want  Confidence
	}{
		{1.0, ConfidenceVerified},
		{0.95, ConfidenceHigh},
		{0.8, ConfidenceHigh},
		{0.5, ConfidenceMedium},
		{0.79, ConfidenceMedium},
		{0.3, ConfidenceLow},
		{0.0, ConfidenceLow},
	}
	for _, tc := range tests {
		got := confidenceFromFloat(tc.input)
		if got != tc.want {
			t.Errorf("confidenceFromFloat(%v) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Groovy enum-mapper tests (Phase 3 — M7, M8, M-MAPSYM)
// ---------------------------------------------------------------------------

// TestLanguageToProto_Groovy ensures the languageToProtoMap entry added in M7
// resolves "groovy" (and its uppercase form) to LANGUAGE_GROOVY.
func TestLanguageToProto_Groovy(t *testing.T) {
	cases := []struct {
		input string
		want  commonv1.Language
	}{
		{"groovy", commonv1.Language_LANGUAGE_GROOVY},
		{"GROOVY", commonv1.Language_LANGUAGE_GROOVY},
		{"Groovy", commonv1.Language_LANGUAGE_GROOVY},
	}
	for _, tc := range cases {
		got := languageToProto(tc.input)
		if got != tc.want {
			t.Errorf("languageToProto(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// TestDeriveLanguage_Groovy ensures .groovy/.gradle/.gvy extensions (added in
// M8) resolve to LANGUAGE_GROOVY via deriveLanguage.
func TestDeriveLanguage_Groovy(t *testing.T) {
	cases := []struct {
		path string
		want commonv1.Language
	}{
		{"Service.groovy", commonv1.Language_LANGUAGE_GROOVY},
		{"build.gradle", commonv1.Language_LANGUAGE_GROOVY},
		{"Script.gvy", commonv1.Language_LANGUAGE_GROOVY},
	}
	for _, tc := range cases {
		got := deriveLanguage(tc.path)
		if got != tc.want {
			t.Errorf("deriveLanguage(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

// TestMapSymbol_GroovyNormalizesToEnum is the M-MAPSYM regression guard.
// mapSymbol used to cast s.Language directly as Language(s.Language) — a
// raw lowercase string. languageStringToGraphQL uppercases before calling
// IsValid, so "groovy" must produce LanguageGroovy, not LanguageUnknown.
func TestMapSymbol_GroovyNormalizesToEnum(t *testing.T) {
	// Test the helper directly — mapSymbol is unexported and requires a
	// full graph.Symbol which depends on the store. Testing the helper
	// is equivalent since mapSymbol's language path is now a one-liner
	// delegating to languageStringToGraphQL.
	cases := []struct {
		input string
		want  Language
	}{
		{"groovy", LanguageGroovy},
		{"GROOVY", LanguageGroovy},
		{"go", LanguageGo},
		{"python", LanguagePython},
		{"notareal", LanguageUnknown},
		{"", LanguageUnknown},
	}
	for _, tc := range cases {
		got := languageStringToGraphQL(tc.input)
		if got != tc.want {
			t.Errorf("languageStringToGraphQL(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}

	// Direct mapSymbol call: verify the full mapping path from a stored symbol
	// whose Language is "groovy" produces a GraphQL symbol with LanguageGroovy.
	// This exercises mapSymbol itself, not just the helper it delegates to.
	sym := &graphstore.StoredSymbol{
		ID:            "test:groovy:sym",
		RepoID:        "repo1",
		Name:          "MyGroovyClass",
		QualifiedName: "com.example.MyGroovyClass",
		Kind:          "class",
		Language:      "groovy",
		FilePath:      "src/MyGroovyClass.groovy",
	}
	mapped := mapSymbol(sym)
	if mapped.Language != LanguageGroovy {
		t.Errorf("mapSymbol with Language=%q: got %v, want %v", sym.Language, mapped.Language, LanguageGroovy)
	}
}
