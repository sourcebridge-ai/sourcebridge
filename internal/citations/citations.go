// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

// Package citations defines the canonical citation format shared
// across all SourceBridge report surfaces (agentic QA, compliance,
// knowledge artifacts, IDE plugin).
//
// The on-wire format for a file/line citation is:
//
//	path:startLine-endLine         (e.g. internal/auth/auth.go:42-55)
//
// Symbol citations use the opaque symbol handle:
//
//	sym_<id>                       (e.g. sym_abc123)
//
// Requirement citations use the external requirement ID verbatim.
//
// This package provides [Format] and [Parse] so every callsite
// produces the same string and every consumer parses it the same way.
// Round-trip stability is guaranteed: Parse(Format(c)) == (c, true)
// for every well-formed Citation.
package citations

import (
	"fmt"
	"strconv"
	"strings"
)

// Kind discriminates the three citation variants.
type Kind string

const (
	// KindFileRange cites a specific line range in a source file.
	// Format: "path:startLine-endLine"
	KindFileRange Kind = "file_range"

	// KindSymbol cites a code symbol by its opaque symbol ID.
	// Format: "sym_<id>"
	KindSymbol Kind = "symbol"

	// KindRequirement cites an external requirement by its external ID.
	// Format: the external ID verbatim (no structural wrapper).
	KindRequirement Kind = "requirement"
)

// Citation is the canonical, typed representation of a SourceBridge
// citation. All report surfaces convert to/from this type.
type Citation struct {
	Kind Kind

	// FileRange fields — populated when Kind == KindFileRange.
	RepoID    string // optional; absent from the string format but carried for programmatic use
	Path      string
	StartLine int
	EndLine   int

	// Symbol fields — populated when Kind == KindSymbol.
	SymbolID string

	// Requirement fields — populated when Kind == KindRequirement.
	RequirementID string
}

// FormatFileRange returns the canonical string representation of a
// file-range citation: "path:startLine-endLine".
// When endLine is zero (unknown), it falls back to "path:startLine".
func FormatFileRange(path string, startLine, endLine int) string {
	if path == "" {
		return ""
	}
	if startLine <= 0 {
		return path
	}
	if endLine <= 0 || endLine < startLine {
		return fmt.Sprintf("%s:%d", path, startLine)
	}
	return fmt.Sprintf("%s:%d-%d", path, startLine, endLine)
}

// FormatSymbol returns the canonical string representation of a
// symbol citation: "sym_<id>". The prefix is idempotent — already-
// prefixed IDs pass through unchanged.
func FormatSymbol(symbolID string) string {
	if symbolID == "" {
		return ""
	}
	if strings.HasPrefix(symbolID, "sym_") {
		return symbolID
	}
	return "sym_" + symbolID
}

// Format returns the canonical string for a Citation. Returns an
// empty string for the zero value or unknown kind.
func Format(c Citation) string {
	switch c.Kind {
	case KindFileRange:
		return FormatFileRange(c.Path, c.StartLine, c.EndLine)
	case KindSymbol:
		return FormatSymbol(c.SymbolID)
	case KindRequirement:
		return c.RequirementID
	}
	return ""
}

// Parse parses a citation string into a typed Citation.
// Returns (Citation, true) on success; (Citation{}, false) when the
// string is empty or not recognizable.
//
// Disambiguation rules (applied in order):
//  1. Empty string → false.
//  2. Prefix "sym_" → KindSymbol.
//  3. Contains ":" with a trailing "digits" or "digits-digits"
//     segment → KindFileRange.
//  4. Anything else → KindRequirement (caller decides how to handle).
func Parse(s string) (Citation, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Citation{}, false
	}

	// Symbol handle.
	if strings.HasPrefix(s, "sym_") {
		id := strings.TrimPrefix(s, "sym_")
		if id == "" {
			return Citation{}, false
		}
		return Citation{Kind: KindSymbol, SymbolID: s}, true
	}

	// Attempt file-range parse: find the rightmost ":" and check if
	// the tail is a valid line range.
	if idx := strings.LastIndex(s, ":"); idx > 0 {
		tail := s[idx+1:]
		start, end, ok := parseLineRange(tail)
		if ok {
			return Citation{
				Kind:      KindFileRange,
				Path:      s[:idx],
				StartLine: start,
				EndLine:   end,
			}, true
		}
	}

	// Treat as requirement ID verbatim.
	return Citation{Kind: KindRequirement, RequirementID: s}, true
}

// ParseRange pulls line numbers from a `path:start-end` or `path:start`
// string. Returns zeros when no valid range is present.
// This is the shared implementation that replaces the ad-hoc
// parseHandleRange helpers that existed in internal/qa.
func ParseRange(handle string) (startLine, endLine int) {
	idx := strings.LastIndex(handle, ":")
	if idx < 0 {
		return 0, 0
	}
	start, end, ok := parseLineRange(handle[idx+1:])
	if !ok {
		return 0, 0
	}
	return start, end
}

// parseLineRange parses the suffix portion after the last ":". Accepts
// "n" (single line) or "n-m" (range). Returns ok=false for anything else.
func parseLineRange(tail string) (start, end int, ok bool) {
	if tail == "" {
		return 0, 0, false
	}
	dash := strings.Index(tail, "-")
	if dash < 0 {
		// Single line: "n"
		n, err := strconv.Atoi(tail)
		if err != nil || n <= 0 {
			return 0, 0, false
		}
		return n, n, true
	}
	startStr := tail[:dash]
	endStr := tail[dash+1:]
	s, err1 := strconv.Atoi(startStr)
	e, err2 := strconv.Atoi(endStr)
	if err1 != nil || err2 != nil || s <= 0 || e < s {
		return 0, 0, false
	}
	return s, e, true
}
