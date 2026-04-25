// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package citations_test

import (
	"testing"

	"github.com/sourcebridge/sourcebridge/internal/citations"
)

// TestFormatFileRange covers the canonical string for every edge case
// the callers in qa/ and knowledge/ can produce.
func TestFormatFileRange(t *testing.T) {
	tests := []struct {
		path  string
		start int
		end   int
		want  string
	}{
		// Normal range.
		{"internal/auth/auth.go", 42, 55, "internal/auth/auth.go:42-55"},
		// Same start and end treated as range.
		{"foo.go", 1, 1, "foo.go:1-1"},
		// No end line: single-line form.
		{"foo.go", 10, 0, "foo.go:10"},
		// End before start: treated as unknown.
		{"foo.go", 10, 5, "foo.go:10"},
		// No start line: just the path.
		{"foo.go", 0, 0, "foo.go"},
		// Empty path.
		{"", 1, 5, ""},
	}
	for _, tc := range tests {
		got := citations.FormatFileRange(tc.path, tc.start, tc.end)
		if got != tc.want {
			t.Errorf("FormatFileRange(%q, %d, %d) = %q, want %q", tc.path, tc.start, tc.end, got, tc.want)
		}
	}
}

// TestFormatSymbol verifies the idempotent sym_ prefix.
func TestFormatSymbol(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"abc123", "sym_abc123"},
		{"sym_abc123", "sym_abc123"}, // already prefixed
		{"", ""},
	}
	for _, tc := range tests {
		got := citations.FormatSymbol(tc.id)
		if got != tc.want {
			t.Errorf("FormatSymbol(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}
}

// TestParse covers every disambiguation branch.
func TestParse(t *testing.T) {
	tests := []struct {
		input     string
		wantKind  citations.Kind
		wantPath  string
		wantStart int
		wantEnd   int
		wantSym   string
		wantReq   string
		wantOK    bool
	}{
		// File range.
		{
			input:     "internal/auth/auth.go:42-55",
			wantKind:  citations.KindFileRange,
			wantPath:  "internal/auth/auth.go",
			wantStart: 42,
			wantEnd:   55,
			wantOK:    true,
		},
		// Single-line form.
		{
			input:     "foo.go:10",
			wantKind:  citations.KindFileRange,
			wantPath:  "foo.go",
			wantStart: 10,
			wantEnd:   10,
			wantOK:    true,
		},
		// Symbol.
		{
			input:    "sym_abc123",
			wantKind: citations.KindSymbol,
			wantSym:  "sym_abc123",
			wantOK:   true,
		},
		// Requirement (no structure).
		{
			input:   "REQ-001",
			wantKind: citations.KindRequirement,
			wantReq: "REQ-001",
			wantOK:  true,
		},
		// Empty string.
		{
			input:  "",
			wantOK: false,
		},
		// Whitespace-only.
		{
			input:  "   ",
			wantOK: false,
		},
		// Path with colons but no valid range (e.g. a URL that slipped in).
		{
			input:   "https://example.com/path",
			wantKind: citations.KindRequirement,
			wantReq: "https://example.com/path",
			wantOK:  true,
		},
		// Deep path with a numeric-looking segment in the middle.
		{
			input:     "internal/auth/v2/auth.go:100-200",
			wantKind:  citations.KindFileRange,
			wantPath:  "internal/auth/v2/auth.go",
			wantStart: 100,
			wantEnd:   200,
			wantOK:    true,
		},
		// Bare sym_ prefix with no id → parse fails.
		{
			input:  "sym_",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		c, ok := citations.Parse(tc.input)
		if ok != tc.wantOK {
			t.Errorf("Parse(%q): ok=%v, want %v", tc.input, ok, tc.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if c.Kind != tc.wantKind {
			t.Errorf("Parse(%q): kind=%q, want %q", tc.input, c.Kind, tc.wantKind)
		}
		switch c.Kind {
		case citations.KindFileRange:
			if c.Path != tc.wantPath {
				t.Errorf("Parse(%q): path=%q, want %q", tc.input, c.Path, tc.wantPath)
			}
			if c.StartLine != tc.wantStart {
				t.Errorf("Parse(%q): startLine=%d, want %d", tc.input, c.StartLine, tc.wantStart)
			}
			if c.EndLine != tc.wantEnd {
				t.Errorf("Parse(%q): endLine=%d, want %d", tc.input, c.EndLine, tc.wantEnd)
			}
		case citations.KindSymbol:
			if c.SymbolID != tc.wantSym {
				t.Errorf("Parse(%q): symbolID=%q, want %q", tc.input, c.SymbolID, tc.wantSym)
			}
		case citations.KindRequirement:
			if c.RequirementID != tc.wantReq {
				t.Errorf("Parse(%q): requirementID=%q, want %q", tc.input, c.RequirementID, tc.wantReq)
			}
		}
	}
}

// TestRoundTrip proves Format(Parse(s)) == s for all well-formed inputs.
func TestRoundTrip(t *testing.T) {
	cases := []string{
		"internal/auth/auth.go:42-55",
		"foo.go:10-10",
		"sym_abc123",
		"sym_already-prefixed",
		"REQ-001",
		"JIRA-1234",
	}
	for _, s := range cases {
		c, ok := citations.Parse(s)
		if !ok {
			t.Errorf("Parse(%q) returned ok=false, expected success", s)
			continue
		}
		got := citations.Format(c)
		if got != s {
			t.Errorf("Format(Parse(%q)) = %q, want %q", s, got, s)
		}
	}
}

// TestParseRange verifies the shared helper that replaces the ad-hoc
// parseHandleRange in internal/qa/agent_reference.go.
func TestParseRange(t *testing.T) {
	tests := []struct {
		handle     string
		wantStart  int
		wantEnd    int
	}{
		{"internal/auth/auth.go:42-55", 42, 55},
		{"foo.go:1-1", 1, 1},
		{"foo.go:10", 10, 10},
		{"foo.go", 0, 0},
		{"", 0, 0},
		{"sym_abc", 0, 0},
		{"path:not-a-number", 0, 0},
	}
	for _, tc := range tests {
		s, e := citations.ParseRange(tc.handle)
		if s != tc.wantStart || e != tc.wantEnd {
			t.Errorf("ParseRange(%q) = (%d, %d), want (%d, %d)", tc.handle, s, e, tc.wantStart, tc.wantEnd)
		}
	}
}

// TestFormat_ZeroValue ensures Format never panics on a zero Citation.
func TestFormat_ZeroValue(t *testing.T) {
	got := citations.Format(citations.Citation{})
	if got != "" {
		t.Errorf("Format(zero) = %q, want empty string", got)
	}
}
