// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

// TestGrailsRoleFor exercises the pure path-classifier with no grammar
// dependency.  This file carries no build tag so that -tags nogroovy builds
// still exercise the path-classification contract.

package indexer

import "testing"

// TestGrailsRoleFor exercises the path-classifier directly.  No grammar
// dependency — runs even with -tags nogroovy.
func TestGrailsRoleFor(t *testing.T) {
	cases := []struct {
		name string
		path string
		want string
	}{
		// ---- positive cases: every convention dir + expected extension ----
		{
			name: "controller under grails-app/controllers",
			path: "grails-app/controllers/example/FooController.groovy",
			want: GrailsRoleController,
		},
		{
			name: "domain under grails-app/domain",
			path: "grails-app/domain/example/Foo.groovy",
			want: GrailsRoleDomain,
		},
		{
			name: "service under grails-app/services",
			path: "grails-app/services/example/FooService.groovy",
			want: GrailsRoleService,
		},
		{
			name: "taglib under grails-app/taglib",
			path: "grails-app/taglib/example/FooTagLib.groovy",
			want: GrailsRoleTagLib,
		},
		{
			name: "conf under grails-app/conf",
			path: "grails-app/conf/Config.groovy",
			want: GrailsRoleConf,
		},
		{
			name: "view under grails-app/views",
			path: "grails-app/views/foo/index.gsp",
			want: GrailsRoleView,
		},
		{
			name: "tolerates leading ./ prefix",
			path: "./grails-app/controllers/FooController.groovy",
			want: GrailsRoleController,
		},
		{
			name: "tolerates leading / prefix",
			path: "/grails-app/controllers/FooController.groovy",
			want: GrailsRoleController,
		},
		{
			name: "backslash-normalized Windows-style path",
			path: `grails-app\services\example\FooService.groovy`,
			want: GrailsRoleService,
		},
		// ---- case-fold contract (Decision 3): mixed-case path must match ----
		// Without case-folding this would return "" — this case pins the
		// case-insensitive matching that makes GrailsRoleFor consistent with
		// the live MCP behavior (which used startsWithFold/endsWithFold).
		{
			name: "mixed-case path classifies correctly (case-fold contract)",
			path: "Grails-App/Controllers/FooController.groovy",
			want: GrailsRoleController,
		},

		// ---- negative cases: no role, empty string expected ----
		{
			name: "empty path",
			path: "",
			want: "",
		},
		{
			name: "plain Groovy file outside grails-app",
			path: "src/main/groovy/example/Foo.groovy",
			want: "",
		},
		{
			name: "test spec, not a grails-app file",
			path: "src/test/groovy/example/FooSpec.groovy",
			want: "",
		},
		{
			name: "non-Groovy file in a convention dir is NOT tagged",
			path: "grails-app/controllers/FooController.java",
			want: "",
		},
		{
			name: "non-.gsp file in views/ is NOT a grails_view",
			path: "grails-app/views/foo/index.groovy",
			want: "",
		},
		{
			name: "grails-app as a mid-path segment does NOT match",
			path: "legacy/grails-app/controllers/FooController.groovy",
			want: "",
		},
		{
			name: "unknown grails-app subdir (no convention)",
			path: "grails-app/unknown/Foo.groovy",
			want: "",
		},
		{
			name: "top-level build.gradle is not a Grails file",
			path: "build.gradle",
			want: "",
		},
		// ---- .gvy contract: .gvy in a convention dir is NOT role-tagged ----
		// Only .groovy (code dirs) and .gsp (views) receive a role.  .gvy is
		// a Groovy variant extension but is not part of the Grails convention
		// contract.  Documented in grails.go's GrailsRoleFor docstring.
		{
			name: ".gvy file in services/ is NOT role-tagged",
			path: "grails-app/services/FooService.gvy",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := GrailsRoleFor(tc.path)
			if got != tc.want {
				t.Errorf("GrailsRoleFor(%q) = %q; want %q", tc.path, got, tc.want)
			}
		})
	}
}
