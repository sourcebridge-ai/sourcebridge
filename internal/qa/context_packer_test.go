// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package qa

import "testing"

// TestExtFromPath verifies extFromPath's language-hint mapping for every
// extension family it supports. Inputs are full paths — extFromPath takes a
// path, strips the extension via strings.LastIndex, and switches on the bare
// ext string. Extension-only inputs (e.g. ".rb") would return "" because the
// function looks for a dot inside the string; use real path shapes.
//
// Source of truth: DetectLanguage in internal/git/local.go — extFromPath must
// mirror its extension→language assignments exactly.
func TestExtFromPath(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		// pre-existing languages
		{"main.go", "go"},
		{"app/server.py", "python"},
		{"web/index.ts", "typescript"},
		{"web/App.tsx", "tsx"},
		{"src/app.js", "javascript"},
		{"src/App.jsx", "jsx"},
		{"src/Main.java", "java"},
		{"lib/parser.rs", "rust"},
		{"grails-app/conf/Application.groovy", "groovy"},
		{"build.gradle", "groovy"},
		{"app.gvy", "groovy"},
		{"docs/README.md", "markdown"},

		// CA-549: ruby/php/csharp/cpp parity — mirrors DetectLanguage
		{"app/models/user.rb", "ruby"},
		{"src/Controller.php", "php"},
		{"a/b/Foo.cs", "csharp"},
		{"src/engine.cpp", "cpp"},
		{"lib/util.cc", "cpp"},
		{"src/parser.cxx", "cpp"},
		// .c and .h map to "cpp" per DetectLanguage (C is classified as cpp)
		{"lib/alloc.c", "cpp"},
		{"include/types.h", "cpp"},
		{"include/vector.hpp", "cpp"},

		// negative: unknown extension → empty string
		{"README.unknownext", ""},
		{"Makefile", ""},   // no dot in the last segment
		{".gitignore", ""}, // dot at index 0 → ext is "gitignore" → not in switch
	}
	for _, c := range cases {
		if got := extFromPath(c.path); got != c.want {
			t.Errorf("extFromPath(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}
