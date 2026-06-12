// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

// Internal white-box tests for confluence_storage.go helpers.
package markdown

import "testing"

// TestConfluenceCodeLanguage_Groovy verifies that Groovy and Gradle language
// identifiers map to Confluence's "groovy" code-macro language value, not
// the "none" fallback (M-CONF, Phase 5).
func TestConfluenceCodeLanguage_Groovy(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"groovy", "groovy"},
		{"GROOVY", "groovy"},  // confluenceCodeLanguage lowercases before lookup
		{"Groovy", "groovy"},
		{"gradle", "groovy"},  // Gradle files use Groovy syntax; macro uses "groovy"
		{"GRADLE", "groovy"},
	}
	for _, tc := range tests {
		got := confluenceCodeLanguage(tc.input)
		if got != tc.want {
			t.Errorf("confluenceCodeLanguage(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestConfluenceCodeLanguage_UnknownFallsBackToNone verifies that an unknown
// language returns "none" (the existing default).
func TestConfluenceCodeLanguage_UnknownFallsBackToNone(t *testing.T) {
	got := confluenceCodeLanguage("xyzzy")
	if got != "none" {
		t.Errorf("confluenceCodeLanguage(%q) = %q, want %q", "xyzzy", got, "none")
	}
}

// TestConfluenceCodeLanguage_EmptyFallsBackToNone verifies that an empty
// language hint returns "none".
func TestConfluenceCodeLanguage_EmptyFallsBackToNone(t *testing.T) {
	got := confluenceCodeLanguage("")
	if got != "none" {
		t.Errorf("confluenceCodeLanguage(%q) = %q, want %q", "", got, "none")
	}
}
