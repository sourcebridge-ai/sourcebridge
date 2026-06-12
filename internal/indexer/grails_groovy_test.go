// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

//go:build !nogroovy

// Grails integration tests that exercise ParseFile on Groovy content.
// Gated behind !nogroovy because without the grammar registered, ParseFile
// returns an empty FileResult (no LanguageConfig) and the assertions would
// fail rather than skip.
//
// TestGrailsRoleFor (pure path classifier, no grammar) lives in grails_test.go
// with no build tag, so it runs even under -tags nogroovy.

package indexer

import (
	"context"
	"testing"
)

// TestGrailsRoleIntegrated confirms that ParseFile attaches the Grails role
// to the returned FileResult at parse time so downstream consumers don't
// need a second classifier call.
func TestGrailsRoleIntegrated(t *testing.T) {
	parser := NewParser()

	// Read real Grails-shape content from the fixture tree, but parse it
	// under the repo-root-relative path a real Grails project would have.
	// The classifier only tags files whose path starts at grails-app/<role>/,
	// so we can't pass the fixture disk path directly.
	content := readFixture(t, "grails-fixture/grails-app/controllers/FooController.groovy")
	repoRelPath := "grails-app/controllers/FooController.groovy"

	result, err := parser.ParseFile(context.Background(), repoRelPath, "groovy", content)
	if err != nil {
		t.Fatal(err)
	}

	if result.GrailsRole != GrailsRoleController {
		t.Errorf("FileResult.GrailsRole = %q, want %q", result.GrailsRole, GrailsRoleController)
	}
	if len(result.Symbols) == 0 {
		t.Error("expected at least one symbol from FooController.groovy")
	}
}

// TestGrailsRoleIntegratedAllConventions walks ALL six convention fixture files
// through ParseFile (or GrailsRoleFor for .gsp) and asserts (a) the expected
// GrailsRole and (b) empirically-verified lower-bound symbol counts.
//
// Symbol counts were determined empirically by running the parser against the
// actual fixture files and observing the output (Phase 2 probe run):
//
//	controllers/FooController.groovy — class + 2 methods captured by both
//	  ClassQuery and MethodQuery = 5 total; lower bound >= 3 (class + 2 methods)
//	domain/Foo.groovy          — 1 (just the Foo class; field declarations
//	  are not extracted as symbols for any language)
//	services/FooService.groovy — 5 (class + 2 methods via each query path);
//	  lower bound >= 3
//	taglib/FooTagLib.groovy    — 1 (FooTagLib class only; the "def greet = {...}"
//	  closure-assignment does NOT parse as a function_definition — it is a
//	  declaration node — so greet does NOT appear as a symbol)
//	conf/Config.groovy         — 0 (pure Grails DSL blocks; no class or def)
//	views/index.gsp            — path-classifier only (see separate sub-test)
func TestGrailsRoleIntegratedAllConventions(t *testing.T) {
	parser := NewParser()

	cases := []struct {
		name        string
		fixturePath string // relative to tests/fixtures/multi-lang-repo/
		repoRelPath string // repo-root-relative path used for role classification
		wantRole    string
		minSymbols  int
	}{
		{
			// Empirical: 5 symbols (class + 2 methods via ClassQuery AND MethodQuery
			// both fire; lower bound is 3: FooController + index + show).
			name:        "controller",
			fixturePath: "grails-fixture/grails-app/controllers/FooController.groovy",
			repoRelPath: "grails-app/controllers/FooController.groovy",
			wantRole:    GrailsRoleController,
			minSymbols:  3,
		},
		{
			// Empirical: 1 symbol (Foo class only; field declarations are not
			// extracted as symbols for any language by design).
			name:        "domain",
			fixturePath: "grails-fixture/grails-app/domain/Foo.groovy",
			repoRelPath: "grails-app/domain/Foo.groovy",
			wantRole:    GrailsRoleDomain,
			minSymbols:  1,
		},
		{
			// Empirical: 5 symbols (FooService class + findAll + count, each
			// captured by both FunctionQuery and MethodQuery; lower bound is 3).
			name:        "service",
			fixturePath: "grails-fixture/grails-app/services/FooService.groovy",
			repoRelPath: "grails-app/services/FooService.groovy",
			wantRole:    GrailsRoleService,
			minSymbols:  3,
		},
		{
			// Empirical: 1 symbol (FooTagLib class only).
			// The "def greet = { attrs -> ... }" is a closure-assignment, which
			// the grammar parses as a declaration node, NOT a function_definition.
			// So greet does NOT appear as a function symbol — taglib yields 1,
			// not the 2 that might naively be expected.
			name:        "taglib",
			fixturePath: "grails-fixture/grails-app/taglib/FooTagLib.groovy",
			repoRelPath: "grails-app/taglib/FooTagLib.groovy",
			wantRole:    GrailsRoleTagLib,
			minSymbols:  1,
		},
		{
			// Empirical: 0 symbols (pure Grails DSL; no class or def declarations).
			name:        "conf",
			fixturePath: "grails-fixture/grails-app/conf/Config.groovy",
			repoRelPath: "grails-app/conf/Config.groovy",
			wantRole:    GrailsRoleConf,
			minSymbols:  0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := readFixture(t, tc.fixturePath)
			result, err := parser.ParseFile(context.Background(), tc.repoRelPath, "groovy", content)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			if result.GrailsRole != tc.wantRole {
				t.Errorf("GrailsRole = %q; want %q", result.GrailsRole, tc.wantRole)
			}
			if len(result.Symbols) < tc.minSymbols {
				t.Errorf("symbol count = %d; want >= %d (symbols: %v)",
					len(result.Symbols), tc.minSymbols, symbolNames(result.Symbols))
			}
		})
	}

	// .gsp view: path-classifier only.
	//
	// A real ScanRepository → IndexRepository run produces NO FileResult for
	// .gsp files: DetectLanguage has no .gsp case and the indexer skips files
	// where GetLanguageConfig(language) == nil.  We therefore assert via
	// GrailsRoleFor only (NOT ParseFile), documenting the path-classifier
	// behavior without implying .gsp files are scanned.
	t.Run("view_gsp_path_classifier_only", func(t *testing.T) {
		role := GrailsRoleFor("grails-app/views/index.gsp")
		if role != GrailsRoleView {
			t.Errorf("GrailsRoleFor(gsp) = %q; want %q", role, GrailsRoleView)
		}
		// Negative assertion: .gsp is NOT scanned — GetLanguageConfig returns nil
		// for the empty-string language that DetectLanguage returns for .gsp.
		if cfg := GetLanguageConfig(""); cfg != nil {
			t.Error("expected nil LanguageConfig for empty language string (gsp scan path)")
		}
	})
}
