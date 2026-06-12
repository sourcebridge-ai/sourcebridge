// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package indexer

import (
	"path/filepath"
	"strings"
)

// Grails role constants. Applied to FileResult.GrailsRole when a file
// lives under a Grails convention directory. Empty role = not a
// Grails-convention file (including plain Groovy files, non-Grails
// repos, and files that happen to share a similar path fragment).
const (
	GrailsRoleController = "grails_controller"
	GrailsRoleDomain     = "grails_domain"
	GrailsRoleService    = "grails_service"
	GrailsRoleTagLib     = "grails_taglib"
	GrailsRoleConf       = "grails_conf"
	GrailsRoleView       = "grails_view"
)

// grailsDirRoles maps the exact directory prefix (everything up to and
// including the role directory) to the role constant. Order matters
// only for readability — the lookup is first-match since each prefix
// is unique.
var grailsDirRoles = []struct {
	prefix string
	role   string
	ext    string // ".groovy" for all Groovy ones, ".gsp" for views
}{
	{"grails-app/controllers/", GrailsRoleController, ".groovy"},
	{"grails-app/domain/", GrailsRoleDomain, ".groovy"},
	{"grails-app/services/", GrailsRoleService, ".groovy"},
	{"grails-app/taglib/", GrailsRoleTagLib, ".groovy"},
	{"grails-app/conf/", GrailsRoleConf, ".groovy"},
	{"grails-app/views/", GrailsRoleView, ".gsp"},
}

// GrailsRoleFor classifies a repo-relative file path by Grails
// directory convention. Returns one of the GrailsRole* constants when
// the path matches, or "" when it doesn't.
//
// Matching is case-insensitive (path is lowercased before comparison)
// so that paths on case-insensitive filesystems (e.g.
// "Grails-App/Controllers/Foo.groovy") classify correctly. This
// matches the live behavior of the MCP entry-points layer which this
// function replaces as the single canonical classifier (Decision 3).
//
// Matching is strict:
//   - The convention directory must appear as a path prefix (leading
//     "/" accepted for convenience — callers may pass "./foo" or "foo").
//   - The file extension must match the directory's expected extension
//     (.groovy for code dirs, .gsp for views) so random files dropped
//     into grails-app/ aren't mis-classified.
//   - A grails-app segment appearing mid-path (e.g. "legacy/grails-app/bar")
//     is NOT matched — Grails projects place grails-app at the repo root.
//   - .gvy files are NOT role-tagged even when they appear under a
//     convention directory. Only .groovy (code dirs) and .gsp (views)
//     receive a role. This is intentional: .gvy is a Groovy variant
//     extension without the Grails convention contract.
//
// The check is intentionally conservative: false-negatives on weird
// repo layouts are better than false-positives that would mis-tag
// plain Groovy code as a Grails controller.
func GrailsRoleFor(path string) string {
	if path == "" {
		return ""
	}

	// Normalize to forward slashes and strip a single leading "./" or "/".
	clean := strings.ReplaceAll(path, "\\", "/")
	clean = strings.TrimPrefix(clean, "./")
	clean = strings.TrimPrefix(clean, "/")

	// Case-fold for case-insensitive matching (matches MCP live behavior).
	cleanLower := strings.ToLower(clean)

	ext := strings.ToLower(filepath.Ext(clean))

	for _, r := range grailsDirRoles {
		if !strings.HasPrefix(cleanLower, r.prefix) {
			continue
		}
		if ext != r.ext {
			continue
		}
		return r.role
	}
	return ""
}
