// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

//go:build !nogroovy

// Groovy language support. Gated behind the `nogroovy` build tag so
// operators can disable the Groovy binding if the cgo grammar ever
// misbehaves on their input — build with `-tags nogroovy` to exclude.
//
// See `internal/indexer/testdata/groovy/NODES.md` for the empirical
// grammar notes these queries are derived from.

package indexer

import (
	"github.com/smacker/go-tree-sitter/groovy"
)

func init() {
	Registry["groovy"] = groovyConfig()
}

func groovyConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:     "groovy",
		Language: groovy.GetLanguage(),

		// Top-level `def foo()` AND class methods both parse as
		// `function_definition` in the Groovy grammar. MethodQuery
		// narrows to class-scoped ones for the SymbolMethod kind.
		FunctionQuery: `(function_definition
			function: (identifier) @name) @func`,

		// `class Foo { ... }` — body wraps in `(closure)` rather than a
		// typed class-body node. The closure wrapping also applies to
		// method bodies and nested scopes.
		//
		// Multi-annotation classes (`@A @B class X`) emit ERROR nodes
		// in the current grammar version and are NOT captured here —
		// documented in NODES.md as a known limitation.
		ClassQuery: `(class_definition
			name: (identifier) @name) @class`,

		// Class-scoped methods. The `body: (closure ...)` wrap is the
		// grammar's convention for class bodies.
		MethodQuery: `(class_definition
			body: (closure
				(function_definition
					function: (identifier) @name) @method))`,

		// `import foo.Bar`, `import foo.*`, `import static foo.Bar.X` —
		// all three emit `groovy_import` with a `qualified_name` payload.
		// Note: node kind is `groovy_import`, NOT `import_declaration`.
		ImportQuery: `(groovy_import
			import: (qualified_name) @path)`,

		// Two call kinds — `function_call` covers `foo(x)` and
		// `juxt_function_call` covers juxtaposition / command-chain
		// syntax like Gradle DSL (`plugins { id 'application' }`).
		CallQuery: `[
			(function_call
				function: (identifier) @callee) @call
			(juxt_function_call
				function: (identifier) @callee) @call
		]`,

		// Groovydoc (`/** ... */`) emits a distinct `groovy_doc` node —
		// separate from the generic `(comment)` used for line and block
		// comments. Union both so downstream doc extraction sees both.
		DocCommentQuery: `[
			(groovy_doc) @comment
			(comment) @comment
		]`,

		// Spock / JUnit test conventions. Spock's `def "should foo"()`
		// emits an ERROR node in the current grammar — method names
		// aren't capturable — so test detection is primarily file-based.
		// The filename patterns catch every Spock spec; the func pattern
		// is a fallback for JUnit-style Groovy tests where the method
		// IS a plain identifier.
		TestFilePatterns: []string{"Spec.groovy", "Test.groovy", "Tests.groovy"},
		TestFuncPattern:  "^(test|should)",
	}
}
