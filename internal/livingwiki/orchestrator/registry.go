// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package orchestrator

import (
	"github.com/sourcebridge/sourcebridge/internal/reports/templates"
	"github.com/sourcebridge/sourcebridge/internal/reports/templates/architecture"
	"github.com/sourcebridge/sourcebridge/internal/reports/templates/apiref"
	"github.com/sourcebridge/sourcebridge/internal/reports/templates/glossary"
	"github.com/sourcebridge/sourcebridge/internal/reports/templates/sysoverview"
)

// MapRegistry is a simple map-backed [TemplateRegistry]. Construct via
// [NewDefaultRegistry] for the built-in template set, or [NewMapRegistry]
// to compose a custom set.
type MapRegistry struct {
	m map[string]templates.Template
}

// NewMapRegistry creates a registry from the provided templates.
func NewMapRegistry(ts ...templates.Template) *MapRegistry {
	r := &MapRegistry{m: make(map[string]templates.Template, len(ts))}
	for _, t := range ts {
		r.m[t.ID()] = t
	}
	return r
}

// NewDefaultRegistry creates a registry containing all built-in templates for
// the A1.P1 wiki taxonomy: architecture, api_reference, system_overview, glossary.
//
// The architecture template's [templates.Template.Generate] entry point handles
// the general case; the orchestrator calls [architecture.Template.GeneratePackagePage]
// directly for per-package pages via the [PlannedPage.PackageInfo] field.
func NewDefaultRegistry() *MapRegistry {
	return NewMapRegistry(
		architecture.New(),
		apiref.New(),
		sysoverview.New(),
		glossary.New(),
	)
}

// Compile-time interface check.
var _ TemplateRegistry = (*MapRegistry)(nil)

// Lookup returns the template for the given ID.
func (r *MapRegistry) Lookup(id string) (templates.Template, bool) {
	t, ok := r.m[id]
	return t, ok
}
