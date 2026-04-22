// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package indexer

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	tsts "github.com/smacker/go-tree-sitter/typescript/typescript"
)

// LanguageConfig holds tree-sitter configuration for a language.
type LanguageConfig struct {
	Name     string
	Language *sitter.Language
	// Queries for extracting symbols
	FunctionQuery string
	ClassQuery    string
	ImportQuery   string
	MethodQuery   string
	// Queries for call graph and doc comments
	CallQuery       string
	DocCommentQuery string
	// Test file patterns
	TestFilePatterns []string
	TestFuncPattern  string
}

// Registry maps language names to their tree-sitter configurations.
var Registry = map[string]*LanguageConfig{
	"go":         goConfig(),
	"python":     pythonConfig(),
	"typescript": typescriptConfig(),
	"javascript": javascriptConfig(),
	"java":       javaConfig(),
	"rust":       rustConfig(),
	"ruby":       rubyConfig(),
	"php":        phpConfig(),
}

func goConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:     "go",
		Language: golang.GetLanguage(),
		FunctionQuery: `(function_declaration
			name: (identifier) @name) @func`,
		ClassQuery: `(type_declaration
			(type_spec
				name: (type_identifier) @name
				type: (struct_type))) @struct`,
		ImportQuery: `(import_spec
			path: (interpreted_string_literal) @path)`,
		MethodQuery: `(method_declaration
			receiver: (parameter_list
				(parameter_declaration
					type: [(pointer_type (type_identifier) @receiver) (type_identifier) @receiver]))
			name: (field_identifier) @name) @method`,
		CallQuery: `(call_expression
			function: [
				(identifier) @callee
				(selector_expression field: (field_identifier) @callee)
			]) @call`,
		DocCommentQuery: `(comment) @comment`,
		TestFilePatterns: []string{"_test.go"},
		TestFuncPattern:  "^Test",
	}
}

func pythonConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:     "python",
		Language: python.GetLanguage(),
		FunctionQuery: `(function_definition
			name: (identifier) @name) @func`,
		ClassQuery: `(class_definition
			name: (identifier) @name) @class`,
		ImportQuery: `[
			(import_statement
				name: (dotted_name) @path)
			(import_from_statement
				module_name: (dotted_name) @path)
		]`,
		MethodQuery: `(class_definition
			body: (block
				(function_definition
					name: (identifier) @name) @method))`,
		CallQuery: `(call
			function: [
				(identifier) @callee
				(attribute attribute: (identifier) @callee)
			]) @call`,
		DocCommentQuery: `(expression_statement (string) @docstring)`,
		TestFilePatterns: []string{"test_", "_test.py"},
		TestFuncPattern:  "^test_",
	}
}

func typescriptConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:     "typescript",
		Language: tsts.GetLanguage(),
		FunctionQuery: `[
			(function_declaration
				name: (identifier) @name) @func
			(lexical_declaration
				(variable_declarator
					name: (identifier) @name
					value: (arrow_function))) @func
		]`,
		ClassQuery: `(class_declaration
			name: (type_identifier) @name) @class`,
		ImportQuery: `(import_statement
			source: (string) @path)`,
		MethodQuery: `(class_declaration
			body: (class_body
				(method_definition
					name: (property_identifier) @name) @method))`,
		CallQuery: `(call_expression
			function: [
				(identifier) @callee
				(member_expression property: (property_identifier) @callee)
			]) @call`,
		DocCommentQuery: `(comment) @comment`,
		TestFilePatterns: []string{".test.ts", ".test.tsx", ".spec.ts", ".spec.tsx"},
		TestFuncPattern:  "^(test|it|describe)",
	}
}

func javascriptConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:     "javascript",
		Language: javascript.GetLanguage(),
		FunctionQuery: `[
			(function_declaration
				name: (identifier) @name) @func
			(lexical_declaration
				(variable_declarator
					name: (identifier) @name
					value: (arrow_function))) @func
		]`,
		ClassQuery: `(class_declaration
			name: (identifier) @name) @class`,
		ImportQuery: `(import_statement
			source: (string) @path)`,
		MethodQuery: `(class_declaration
			body: (class_body
				(method_definition
					name: (property_identifier) @name) @method))`,
		CallQuery: `(call_expression
			function: [
				(identifier) @callee
				(member_expression property: (property_identifier) @callee)
			]) @call`,
		DocCommentQuery: `(comment) @comment`,
		TestFilePatterns: []string{".test.js", ".test.jsx", ".spec.js"},
		TestFuncPattern:  "^(test|it|describe)",
	}
}

func javaConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:     "java",
		Language: java.GetLanguage(),
		FunctionQuery: `(method_declaration
			name: (identifier) @name) @func`,
		ClassQuery: `(class_declaration
			name: (identifier) @name) @class`,
		ImportQuery: `(import_declaration
			(scoped_identifier) @path)`,
		MethodQuery: `(class_declaration
			body: (class_body
				(method_declaration
					name: (identifier) @name) @method))`,
		CallQuery: `(method_invocation
			name: (identifier) @callee) @call`,
		DocCommentQuery: `(block_comment) @comment`,
		TestFilePatterns: []string{"Test.java", "Tests.java"},
		TestFuncPattern:  "^test",
	}
}

func rustConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:     "rust",
		Language: rust.GetLanguage(),
		FunctionQuery: `(function_item
			name: (identifier) @name) @func`,
		ClassQuery: `[
			(struct_item
				name: (type_identifier) @name) @struct
			(enum_item
				name: (type_identifier) @name) @enum
			(trait_item
				name: (type_identifier) @name) @trait
		]`,
		ImportQuery: `(use_declaration
			argument: (_) @path)`,
		MethodQuery: `(impl_item
			body: (declaration_list
				(function_item
					name: (identifier) @name) @method))`,
		CallQuery: `(call_expression
			function: [
				(identifier) @callee
				(field_expression field: (field_identifier) @callee)
			]) @call`,
		DocCommentQuery: `(line_comment) @comment`,
		TestFilePatterns: []string{},
		TestFuncPattern:  "^test_",
	}
}

func rubyConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:     "ruby",
		Language: ruby.GetLanguage(),
		// Captures top-level `def foo` methods and `def self.foo` singleton
		// methods (which tree-sitter-ruby emits as a distinct node type).
		// Methods nested inside a class/module body are also captured at
		// the file scope — the MethodQuery adds the class-scoped overlay
		// for classification.
		FunctionQuery: `[
			(method
				name: (identifier) @name) @func
			(singleton_method
				name: (identifier) @name) @func
		]`,
		// Ruby has both `class` and `module` as container nodes with the
		// same name-binding shape; we emit both as "class" for symbol
		// classification so downstream filtering treats modules like
		// Java-package-equivalents.
		ClassQuery: `[
			(class
				name: (constant) @name) @class
			(module
				name: (constant) @name) @class
		]`,
		// Ruby has no dedicated import node. The convention is:
		//   require "foo"
		//   require_relative "bar"
		//   load "baz.rb"
		//   autoload :Qux, "qux"
		// All three are ordinary method calls with a string literal
		// argument. The #match? predicate keeps the capture noise-free.
		ImportQuery: `((call
			method: (identifier) @method
			arguments: (argument_list (string (string_content) @path)))
			(#match? @method "^(require|require_relative|load|autoload)$"))`,
		// Methods defined inside a class or module body.
		MethodQuery: `[
			(class
				body: (body_statement
					(method
						name: (identifier) @name) @method))
			(module
				body: (body_statement
					(method
						name: (identifier) @name) @method))
		]`,
		// Ruby calls: `foo(...)`, `obj.foo(...)`, and `obj.foo` (no args).
		// The "call" node covers parenthesized forms; bare `foo` refs are
		// grammar-ambiguous and we ignore them here to avoid over-capture.
		CallQuery: `(call
			method: (identifier) @callee) @call`,
		// Ruby comments use `#` line comments; the grammar also emits
		// `=begin...=end` blocks as a single `(comment)` node.
		DocCommentQuery: `(comment) @comment`,
		// Common Ruby test conventions:
		//   minitest:  test_helper.rb, *_test.rb with `def test_…`
		//   rspec:     spec/ dirs with *_spec.rb, but those don't use `def`
		//              methods — they use `it "…"`, which this indexer
		//              can't meaningfully tag by function name. Only the
		//              minitest shape is captured here.
		TestFilePatterns: []string{"_test.rb", "_spec.rb"},
		TestFuncPattern:  "^test_",
	}
}

func phpConfig() *LanguageConfig {
	return &LanguageConfig{
		Name:     "php",
		Language: php.GetLanguage(),
		// Top-level `function foo(...)`.
		FunctionQuery: `(function_definition
			name: (name) @name) @func`,
		// `class Foo`, `interface Foo`, `trait Foo` — all emitted as the
		// "class" classification for downstream symbol kind.
		ClassQuery: `[
			(class_declaration
				name: (name) @name) @class
			(interface_declaration
				name: (name) @name) @class
			(trait_declaration
				name: (name) @name) @class
		]`,
		// `use Foo\Bar;` and `use Foo\Bar\Baz;` — the namespace_use_clause
		// wraps a qualified name (or plain name).
		ImportQuery: `[
			(namespace_use_declaration
				(namespace_use_clause (qualified_name) @path))
			(namespace_use_declaration
				(namespace_use_clause (name) @path))
		]`,
		// Methods inside a class / interface / trait body.
		MethodQuery: `[
			(class_declaration
				body: (declaration_list
					(method_declaration
						name: (name) @name) @method))
			(interface_declaration
				body: (declaration_list
					(method_declaration
						name: (name) @name) @method))
			(trait_declaration
				body: (declaration_list
					(method_declaration
						name: (name) @name) @method))
		]`,
		// PHP calls: bare function, method on an object, and static method.
		CallQuery: `[
			(function_call_expression
				function: (name) @callee) @call
			(member_call_expression
				name: (name) @callee) @call
			(scoped_call_expression
				name: (name) @callee) @call
		]`,
		// PHPDoc blocks are `/** … */`; the grammar emits them as
		// `(comment)` nodes, same as single-line `//` and `#` comments.
		DocCommentQuery: `(comment) @comment`,
		// PHPUnit convention: *Test.php files, `public function testFoo`.
		TestFilePatterns: []string{"Test.php"},
		TestFuncPattern:  "^test",
	}
}

// GetLanguageConfig returns the config for a language name, or nil if unsupported.
func GetLanguageConfig(lang string) *LanguageConfig {
	return Registry[lang]
}

// SupportedLanguages returns the list of supported language names.
func SupportedLanguages() []string {
	langs := make([]string, 0, len(Registry))
	for name := range Registry {
		langs = append(langs, name)
	}
	return langs
}
