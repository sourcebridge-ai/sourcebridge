// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

// Package markdown — static_sinks.go implements the static-site sink writers
// (Workstream A1.P5):
//
//   - [BackstageTechDocsSink] — writes to docs/<page-id>.md (Backstage TechDocs convention)
//   - [MkDocsSink]            — writes to docs/<page-id>.md
//   - [DocusaurusSink]        — writes to docs/<page-id>.md with Docusaurus frontmatter
//   - [VitePressSink]         — writes to docs/<page-id>.md with VitePress frontmatter
//
// These sinks are thin wrappers around the markdown writer. Their only job is:
//
//  1. Choose the output path (always docs/<page-id>.md).
//  2. Add tool-specific frontmatter above the SourceBridge frontmatter.
//
// Because these tools build from committed source files, these sinks do not
// have delay queues or edit-policy mechanics — they write straight to the repo
// via the injected [StaticSiteWriter]. Reconciliation is handled by the
// git_repo sink layer; these sinks only control path and frontmatter shape.
//
// # Frontmatter layering for Docusaurus and VitePress
//
// Both tools consume top-level YAML frontmatter keys. The SourceBridge
// frontmatter is nested under the `sourcebridge:` key, which these tools
// ignore. The layered output looks like:
//
//	---
//	id: arch.auth
//	title: "Architecture: internal/auth"
//	sidebar_position: 1          # Docusaurus
//	---
//	sourcebridge:
//	  page_id: arch.auth
//	  ...
//	---
//
// Backstage TechDocs and MkDocs do not require tool-specific keys, so they
// emit the SourceBridge frontmatter unchanged.
package markdown

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/sourcebridge/sourcebridge/internal/livingwiki/ast"
)

// StaticSiteWriter writes rendered source-tree files to the repository.
// Each call to WriteFiles commits the given files to the source tree;
// reconciliation and branch management are handled at the git_repo sink layer.
type StaticSiteWriter interface {
	// WriteFiles writes the given files to the repository source tree.
	// Path keys are repo-root-relative (e.g. "docs/arch.auth.md").
	WriteFiles(ctx context.Context, files map[string][]byte) error
}

// staticDocsPath returns the docs-directory path for a page.
// All static-site sinks use docs/<page-id>.md.
func staticDocsPath(pageID string) string {
	return "docs/" + pageID + ".md"
}

// BackstageTechDocsSink writes wiki pages to the docs/ directory following the
// Backstage TechDocs convention. The output is standard SourceBridge markdown
// with no additional tool-specific frontmatter, because TechDocs reads the
// mkdocs.yml site nav, not YAML frontmatter keys.
type BackstageTechDocsSink struct {
	writer StaticSiteWriter
}

// NewBackstageTechDocsSink creates a [BackstageTechDocsSink] that writes via writer.
func NewBackstageTechDocsSink(writer StaticSiteWriter) *BackstageTechDocsSink {
	return &BackstageTechDocsSink{writer: writer}
}

// WritePage renders page and writes it to docs/<page-id>.md.
func (s *BackstageTechDocsSink) WritePage(ctx context.Context, page ast.Page) error {
	content, err := renderPageToBytes(page)
	if err != nil {
		return fmt.Errorf("backstage_techdocs: rendering page %q: %w", page.ID, err)
	}
	return s.writer.WriteFiles(ctx, map[string][]byte{
		staticDocsPath(page.ID): content,
	})
}

// MkDocsSink writes wiki pages to the docs/ directory following the MkDocs
// convention (docs/<page-id>.md). Frontmatter is the standard SourceBridge
// YAML; MkDocs uses the page filename and mkdocs.yml nav for site structure.
type MkDocsSink struct {
	writer StaticSiteWriter
}

// NewMkDocsSink creates a [MkDocsSink] that writes via writer.
func NewMkDocsSink(writer StaticSiteWriter) *MkDocsSink {
	return &MkDocsSink{writer: writer}
}

// WritePage renders page and writes it to docs/<page-id>.md.
func (s *MkDocsSink) WritePage(ctx context.Context, page ast.Page) error {
	content, err := renderPageToBytes(page)
	if err != nil {
		return fmt.Errorf("mkdocs: rendering page %q: %w", page.ID, err)
	}
	return s.writer.WriteFiles(ctx, map[string][]byte{
		staticDocsPath(page.ID): content,
	})
}

// DocusaurusSink writes wiki pages to the docs/ directory with Docusaurus-
// flavored frontmatter prepended before the SourceBridge frontmatter.
//
// The Docusaurus frontmatter adds:
//
//	id: <page-id>
//	title: <derived from page-id>
//	sidebar_position: <1-based ordinal among pages>
//
// The SourceBridge frontmatter is preserved as-is under the `sourcebridge:` key.
type DocusaurusSink struct {
	writer StaticSiteWriter
}

// NewDocusaurusSink creates a [DocusaurusSink] that writes via writer.
func NewDocusaurusSink(writer StaticSiteWriter) *DocusaurusSink {
	return &DocusaurusSink{writer: writer}
}

// WritePage renders page with Docusaurus frontmatter and writes it to
// docs/<page-id>.md.
func (s *DocusaurusSink) WritePage(ctx context.Context, page ast.Page) error {
	content, err := renderWithDocusaurusFrontmatter(page)
	if err != nil {
		return fmt.Errorf("docusaurus: rendering page %q: %w", page.ID, err)
	}
	return s.writer.WriteFiles(ctx, map[string][]byte{
		staticDocsPath(page.ID): content,
	})
}

// VitePressSink writes wiki pages to the docs/ directory with VitePress-
// flavored frontmatter prepended before the SourceBridge frontmatter.
//
// The VitePress frontmatter adds:
//
//	title: <derived from page-id>
//	description: <SourceBridge-generated page description>
//	outline: deep
//
// The SourceBridge frontmatter is preserved unchanged under the `sourcebridge:` key.
type VitePressSink struct {
	writer StaticSiteWriter
}

// NewVitePressSink creates a [VitePressSink] that writes via writer.
func NewVitePressSink(writer StaticSiteWriter) *VitePressSink {
	return &VitePressSink{writer: writer}
}

// WritePage renders page with VitePress frontmatter and writes it to
// docs/<page-id>.md.
func (s *VitePressSink) WritePage(ctx context.Context, page ast.Page) error {
	content, err := renderWithVitePressFrontmatter(page)
	if err != nil {
		return fmt.Errorf("vitepress: rendering page %q: %w", page.ID, err)
	}
	return s.writer.WriteFiles(ctx, map[string][]byte{
		staticDocsPath(page.ID): content,
	})
}

// ---- frontmatter helpers ----

// renderWithDocusaurusFrontmatter produces a markdown document with the
// Docusaurus-specific top-level frontmatter block layered above the
// SourceBridge frontmatter.
//
// Output format:
//
//	---
//	id: arch.auth
//	title: "arch auth"
//	sidebar_position: 1
//	---
//	sourcebridge:
//	  page_id: arch.auth
//	  ...
//	---
//	<blocks>
func renderWithDocusaurusFrontmatter(page ast.Page) ([]byte, error) {
	title := humanTitle(page.ID)
	toolFM := fmt.Sprintf("---\nid: %s\ntitle: %q\nsidebar_position: 1\n---\n",
		page.ID, title)

	pageBytes, err := renderPageToBytes(page)
	if err != nil {
		return nil, err
	}

	return layerFrontmatter(toolFM, pageBytes), nil
}

// renderWithVitePressFrontmatter produces a markdown document with
// VitePress-specific top-level frontmatter layered above the SourceBridge one.
//
// Output format:
//
//	---
//	title: "arch auth"
//	outline: deep
//	---
//	sourcebridge:
//	  page_id: arch.auth
//	  ...
//	---
//	<blocks>
func renderWithVitePressFrontmatter(page ast.Page) ([]byte, error) {
	title := humanTitle(page.ID)
	toolFM := fmt.Sprintf("---\ntitle: %q\noutline: deep\n---\n", title)

	pageBytes, err := renderPageToBytes(page)
	if err != nil {
		return nil, err
	}

	return layerFrontmatter(toolFM, pageBytes), nil
}

// layerFrontmatter places toolFrontmatter directly before the SourceBridge
// frontmatter. If the page has no SourceBridge frontmatter, the tool
// frontmatter is prepended to the raw content.
//
// The layering is:
//
//	<toolFrontmatter>           ← Docusaurus / VitePress keys
//	<sourcebridgeFrontmatter>   ← sourcebridge: block (unchanged)
//	<body>                      ← block content
func layerFrontmatter(toolFrontmatter string, sourcebridgePage []byte) []byte {
	var buf bytes.Buffer
	writeStringTo(&buf, toolFrontmatter)
	buf.Write(sourcebridgePage)
	return buf.Bytes()
}

// writeStringTo writes s to w, ignoring errors (buf.Buffer.WriteString never errors).
func writeStringTo(w io.Writer, s string) {
	_, _ = io.WriteString(w, s)
}

// humanTitle converts a page ID like "arch.auth" into a human-readable title
// "arch auth" by replacing dots and underscores with spaces.
func humanTitle(pageID string) string {
	r := strings.NewReplacer(".", " ", "_", " ", "-", " ")
	return r.Replace(pageID)
}
