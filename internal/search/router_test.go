// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package search

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		name        string
		q           string
		wantClass   QueryClass
		wantCleaned string
		wantFilters Filters
		wantOps     []StructuralOp
		wantSeed    string
		wantExact   bool
		wantLex     bool
		wantVector  bool
		wantGraph   bool
	}{
		{
			name:      "identifier",
			q:         "parseUser",
			wantClass: ClassIdentifier, wantCleaned: "parseUser",
			wantExact: true, wantLex: true, wantVector: false, wantGraph: false,
		},
		{
			name:      "qualified identifier",
			q:         "auth.parseUser",
			wantClass: ClassIdentifier, wantCleaned: "auth.parseUser",
			wantExact: true, wantLex: true,
		},
		{
			name:      "natural language",
			q:         "where is oidc login handled",
			wantClass: ClassNaturalLng,
			wantLex:   true, wantVector: true,
		},
		{
			name:      "phrase only",
			q:         `"parse user token"`,
			wantClass: ClassPhrase, wantCleaned: "parse user token",
			wantLex: true,
		},
		{
			name:      "structural calls",
			q:         "calls:handleOIDCLogin",
			wantClass: ClassStructural, wantSeed: "handleOIDCLogin",
			wantOps:   []StructuralOp{OpCalls},
			wantGraph: true,
		},
		{
			name:      "mixed identifier + words",
			q:         "session refresh handleRefreshToken",
			wantClass: ClassMixed,
			wantExact: true, wantLex: true, wantVector: true,
		},
		{
			name:        "filter extraction",
			q:           "parseUser lang:go kind:function",
			wantClass:   ClassIdentifier,
			wantCleaned: "parseUser",
			wantFilters: Filters{Kind: "function", Language: "go"},
			wantExact:   true, wantLex: true,
		},
		{
			name:      "path filter with natural language",
			q:         "session cache path:internal/session/**",
			wantClass: ClassNaturalLng,
			wantFilters: Filters{FilePath: "internal/session/**"},
			wantLex:   true, wantVector: true,
		},
		{
			name: "empty", q: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(tc.q)
			if got.Class != tc.wantClass {
				t.Errorf("class: got %q want %q", got.Class, tc.wantClass)
			}
			if tc.wantCleaned != "" && got.Cleaned != tc.wantCleaned {
				t.Errorf("cleaned: got %q want %q", got.Cleaned, tc.wantCleaned)
			}
			if got.Filters != tc.wantFilters {
				t.Errorf("filters: got %+v want %+v", got.Filters, tc.wantFilters)
			}
			if len(got.Structural) != len(tc.wantOps) {
				t.Errorf("structural ops: got %v want %v", got.Structural, tc.wantOps)
			}
			if tc.wantSeed != "" && got.Seed != tc.wantSeed {
				t.Errorf("seed: got %q want %q", got.Seed, tc.wantSeed)
			}
			if got.WantExact != tc.wantExact {
				t.Errorf("wantExact: got %v want %v", got.WantExact, tc.wantExact)
			}
			if got.WantLexical != tc.wantLex {
				t.Errorf("wantLexical: got %v want %v", got.WantLexical, tc.wantLex)
			}
			if got.WantVector != tc.wantVector {
				t.Errorf("wantVector: got %v want %v", got.WantVector, tc.wantVector)
			}
			if got.WantGraph != tc.wantGraph {
				t.Errorf("wantGraph: got %v want %v", got.WantGraph, tc.wantGraph)
			}
		})
	}
}
