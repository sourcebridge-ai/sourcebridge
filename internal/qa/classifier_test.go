// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package qa

import (
	"testing"
)

func TestClassifyQuestion(t *testing.T) {
	cases := []struct {
		q    string
		want QuestionKind
	}{
		{"What is the architecture?", KindArchitecture},
		{"High level overview please", KindArchitecture},
		{"1000 foot view", KindArchitecture},
		{"Describe the subsystem boundaries", KindArchitecture},

		{"How does the flow work?", KindExecutionFlow},
		{"Trace the request path", KindExecutionFlow},
		{"What happens when I click submit?", KindExecutionFlow},
		{"how does auth work", KindExecutionFlow},

		{"Which requirement covers billing?", KindRequirementCoverage},
		{"Explain REQ-123", KindRequirementCoverage},

		{"Where is the user model defined?", KindOwnership},
		{"which file has the handler", KindOwnership},

		{"Show the user schema", KindDataModel},
		{"What's the data model", KindDataModel},
		{"describe the entity relationships", KindDataModel},

		{"Any risk in this change?", KindRiskReview},
		{"security vulnerability review", KindRiskReview},

		{"What does this function do?", KindBehavior},
		{"default fall-through", KindBehavior},
	}
	for _, c := range cases {
		if got := ClassifyQuestion(c.q); got != c.want {
			t.Errorf("classify(%q) = %q, want %q", c.q, got, c.want)
		}
	}
}

func TestPathBoosts_TestFilesPenalty(t *testing.T) {
	b := PathBoosts("internal/foo/bar_test.go", "how does foo work?", KindBehavior)
	if b.Delta >= 0 {
		t.Errorf("expected negative delta for _test.go, got %+v", b)
	}
	hasTest := false
	for _, r := range b.Reasons {
		if r == "penalty:test" {
			hasTest = true
		}
	}
	if !hasTest {
		t.Errorf("expected penalty:test reason, got %v", b.Reasons)
	}
}

func TestPathBoosts_ArchitectureDiagramMarker(t *testing.T) {
	b := PathBoosts("internal/architecture/diagram.go", "how is the architecture diagram generated?", KindArchitecture)
	if b.Delta < 20 {
		t.Errorf("expected strong positive delta for arch diagram path marker, got %+v", b)
	}
}

func TestPathBoosts_FlowRoutes(t *testing.T) {
	b := PathBoosts("internal/api/routes/auth/handler.go", "how does the auth flow work?", KindExecutionFlow)
	if b.Delta < 8 {
		t.Errorf("expected flow boost >= 8, got %+v", b)
	}
}

func TestPathBoosts_SchemaResolver(t *testing.T) {
	b := PathBoosts("internal/api/graphql/schema.resolvers.go", "what does discussCode do", KindBehavior)
	if b.Delta < 3 {
		t.Errorf("expected base resolver boost, got %+v", b)
	}
}

func TestPathBoosts_NonProductPenalty(t *testing.T) {
	b := PathBoosts("docs/essays/x.md", "how does foo work", KindBehavior)
	found := false
	for _, r := range b.Reasons {
		if r == "penalty:non-product" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected penalty:non-product, got %v", b.Reasons)
	}
}
