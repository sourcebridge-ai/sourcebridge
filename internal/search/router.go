// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package search

import (
	"regexp"
	"strings"
)

// RouterOutput is the structured view of a parsed query. It tells the
// service which adapters to invoke and with what cleaned-up inputs.
type RouterOutput struct {
	Class       QueryClass
	Cleaned     string // the query with operators / filters stripped
	Filters     Filters
	Structural  []StructuralOp
	Seed        string // for structural queries, the right-hand side
	Quoted      string // the phrase body for ClassPhrase
	WantExact   bool   // run symbol-exact adapter
	WantLexical bool   // run FTS adapter
	WantVector  bool   // run semantic adapter
	WantGraph   bool   // run structural adapter
}

// Regex definitions. Kept here so the test suite can exercise them
// directly via the package-level classify helpers.
var (
	// An identifier-shaped token: starts with a letter/underscore,
	// contains only id-chars (. : for qualified names).
	reIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.:]*$`)
	// Filter modifier: kind:function, lang:go, path:internal/**
	reFilter = regexp.MustCompile(`\b(kind|lang|path):([^\s]+)`)
	// Structural operator: calls:foo, callers:foo, impl:Interface
	reStructural = regexp.MustCompile(`\b(calls|callers|impl):([A-Za-z_][A-Za-z0-9_.:]*)`)
	// Quoted phrase
	reQuoted = regexp.MustCompile(`"([^"]+)"`)
)

// Classify parses a raw query into a RouterOutput. The function is
// pure: no DB calls, no allocations on the hot path beyond what the
// regexes already perform.
func Classify(raw string) RouterOutput {
	out := RouterOutput{}
	q := strings.TrimSpace(raw)
	if q == "" {
		return out
	}

	// --- extract filters ---
	for _, m := range reFilter.FindAllStringSubmatch(q, -1) {
		switch m[1] {
		case "kind":
			out.Filters.Kind = m[2]
		case "lang":
			out.Filters.Language = m[2]
		case "path":
			out.Filters.FilePath = m[2]
		}
	}
	q = reFilter.ReplaceAllString(q, "")

	// --- extract structural operators ---
	if sm := reStructural.FindAllStringSubmatch(q, -1); len(sm) > 0 {
		for _, m := range sm {
			switch m[1] {
			case "calls":
				out.Structural = append(out.Structural, OpCalls)
			case "callers":
				out.Structural = append(out.Structural, OpCallers)
			case "impl":
				out.Structural = append(out.Structural, OpImpl)
			}
			// First seed wins.
			if out.Seed == "" {
				out.Seed = m[2]
			}
		}
		q = reStructural.ReplaceAllString(q, "")
	}

	// --- extract quoted phrase ---
	if pm := reQuoted.FindStringSubmatch(q); len(pm) >= 2 {
		out.Quoted = pm[1]
		q = reQuoted.ReplaceAllString(q, "")
	}

	out.Cleaned = strings.TrimSpace(strings.Join(strings.Fields(q), " "))

	switch {
	// Structural query: graph-only. No need for lexical / vector.
	case len(out.Structural) > 0 && out.Cleaned == "" && out.Quoted == "":
		out.Class = ClassStructural
		out.WantGraph = true

	// Phrase-only query.
	case out.Quoted != "" && out.Cleaned == "":
		out.Class = ClassPhrase
		out.Cleaned = out.Quoted
		out.WantLexical = true

	// Identifier-shaped token (has separator / CamelCase / ALL_CAPS).
	case reIdentifier.MatchString(out.Cleaned) && !strings.ContainsAny(out.Cleaned, " \t") && hasIdentShape(out.Cleaned):
		out.Class = ClassIdentifier
		out.WantExact = true
		out.WantLexical = true
		// Skip vector on identifiers (plan §Query Router — "cost not
		// justified").
		out.WantVector = false

	// Mixed: has an identifier token inside a natural-language phrase.
	case containsIdentToken(out.Cleaned):
		out.Class = ClassMixed
		out.WantExact = true
		out.WantLexical = true
		out.WantVector = true

	// Fallback natural language: ≥ 1 word, no identifier-shaped token.
	case out.Cleaned != "":
		out.Class = ClassNaturalLng
		out.WantLexical = true
		out.WantVector = true

	// Only structural + other modifiers — still structural.
	case len(out.Structural) > 0:
		out.Class = ClassStructural
		out.WantGraph = true
	}

	// If any structural ops were parsed, always run the graph adapter.
	if len(out.Structural) > 0 {
		out.WantGraph = true
	}

	return out
}

// containsIdentToken returns true when at least one whitespace-split
// token looks like a code identifier rather than an English word.
//
// A bare lowercase word ("login", "cache", "handler") is not an
// identifier for routing purposes — it is indistinguishable from
// ordinary prose and classifying on it would bias every
// natural-language query toward the mixed arm.
//
// A token is identifier-shaped when it matches the identifier regex
// AND carries at least one of:
//
//   - a separator char (`.`, `:`, `_`)
//   - both uppercase and lowercase letters (CamelCase / mixedCase)
//   - is all-caps and at least two characters (SCREAMING_CASE style
//     constants)
func containsIdentToken(s string) bool {
	for _, tok := range strings.Fields(s) {
		if !reIdentifier.MatchString(tok) {
			continue
		}
		if hasIdentShape(tok) {
			return true
		}
	}
	return false
}

func hasIdentShape(tok string) bool {
	hasUpper, hasLower, hasSep := false, false, false
	for _, r := range tok {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r == '.' || r == ':' || r == '_':
			hasSep = true
		}
	}
	if hasSep {
		return true
	}
	if hasUpper && hasLower {
		return true
	}
	if hasUpper && !hasLower && len(tok) > 1 {
		return true
	}
	return false
}
