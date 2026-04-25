// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package markdown_test

import (
	"context"
	"strings"
	"testing"

	"github.com/sourcebridge/sourcebridge/internal/livingwiki/ast"
	"github.com/sourcebridge/sourcebridge/internal/livingwiki/manifest"
	"github.com/sourcebridge/sourcebridge/internal/livingwiki/markdown"
)

// memStaticWriter captures WriteFiles calls for static-site sinks.
type memStaticWriter struct {
	files map[string][]byte
}

func newMemStaticWriter() *memStaticWriter {
	return &memStaticWriter{files: make(map[string][]byte)}
}

func (m *memStaticWriter) WriteFiles(_ context.Context, files map[string][]byte) error {
	for k, v := range files {
		cp := make([]byte, len(v))
		copy(cp, v)
		m.files[k] = cp
	}
	return nil
}

func buildPageWithTitle(id, title string) ast.Page {
	return ast.Page{
		ID: id,
		Manifest: manifest.DependencyManifest{
			PageID:   id,
			Template: "architecture",
			Audience: "for-engineers",
		},
		Blocks: []ast.Block{
			{
				ID:    "bh001",
				Kind:  ast.BlockKindHeading,
				Owner: ast.OwnerGenerated,
				Content: ast.BlockContent{
					Heading: &ast.HeadingContent{Level: 1, Text: title},
				},
			},
			{
				ID:    "bp001",
				Kind:  ast.BlockKindParagraph,
				Owner: ast.OwnerGenerated,
				Content: ast.BlockContent{
					Paragraph: &ast.ParagraphContent{Markdown: "Sample paragraph. (internal/auth/auth.go:1-10)"},
				},
			},
		},
	}
}

// TestBackstageTechDocsSink_OutputPath verifies the output path convention.
func TestBackstageTechDocsSink_OutputPath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	writer := newMemStaticWriter()
	sink := markdown.NewBackstageTechDocsSink(writer)

	page := buildPageWithTitle("arch.auth", "Architecture: internal/auth")
	if err := sink.WritePage(ctx, page); err != nil {
		t.Fatalf("WritePage: %v", err)
	}

	if _, ok := writer.files["docs/arch.auth.md"]; !ok {
		t.Errorf("expected docs/arch.auth.md, got: %v", fileMapKeys(writer.files))
	}
}

// TestBackstageTechDocsSink_ContentHasFrontmatter verifies the output has
// SourceBridge frontmatter (Backstage reads mkdocs.yml, not extra keys).
func TestBackstageTechDocsSink_ContentHasFrontmatter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	writer := newMemStaticWriter()
	sink := markdown.NewBackstageTechDocsSink(writer)

	page := buildPageWithTitle("arch.auth", "Auth")
	if err := sink.WritePage(ctx, page); err != nil {
		t.Fatalf("WritePage: %v", err)
	}

	content := string(writer.files["docs/arch.auth.md"])
	if !strings.Contains(content, "sourcebridge:") {
		t.Error("Backstage output should contain SourceBridge frontmatter")
	}
	// Backstage does NOT add tool-specific keys — verify.
	if strings.Contains(content, "sidebar_position") {
		t.Error("Backstage output should not contain sidebar_position (that is Docusaurus)")
	}
}

// TestMkDocsSink_OutputPath verifies MkDocs uses docs/<page-id>.md.
func TestMkDocsSink_OutputPath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	writer := newMemStaticWriter()
	sink := markdown.NewMkDocsSink(writer)

	page := buildPageWithTitle("system_overview", "System Overview")
	if err := sink.WritePage(ctx, page); err != nil {
		t.Fatalf("WritePage: %v", err)
	}

	if _, ok := writer.files["docs/system_overview.md"]; !ok {
		t.Errorf("expected docs/system_overview.md, got: %v", fileMapKeys(writer.files))
	}
}

// TestDocusaurusSink_OutputPath verifies the Docusaurus sink path.
func TestDocusaurusSink_OutputPath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	writer := newMemStaticWriter()
	sink := markdown.NewDocusaurusSink(writer)

	page := buildPageWithTitle("arch.auth", "Auth")
	if err := sink.WritePage(ctx, page); err != nil {
		t.Fatalf("WritePage: %v", err)
	}

	if _, ok := writer.files["docs/arch.auth.md"]; !ok {
		t.Errorf("expected docs/arch.auth.md, got: %v", fileMapKeys(writer.files))
	}
}

// TestDocusaurusSink_Frontmatter verifies Docusaurus-specific frontmatter keys.
func TestDocusaurusSink_Frontmatter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	writer := newMemStaticWriter()
	sink := markdown.NewDocusaurusSink(writer)

	page := buildPageWithTitle("arch.auth", "Auth")
	if err := sink.WritePage(ctx, page); err != nil {
		t.Fatalf("WritePage: %v", err)
	}

	content := string(writer.files["docs/arch.auth.md"])

	// Must have Docusaurus-specific top-level keys.
	if !strings.Contains(content, "sidebar_position:") {
		t.Error("Docusaurus output missing sidebar_position key")
	}
	if !strings.Contains(content, `id: arch.auth`) {
		t.Error("Docusaurus output missing id key")
	}
	// Must still have SourceBridge frontmatter nested under sourcebridge:.
	if !strings.Contains(content, "sourcebridge:") {
		t.Error("Docusaurus output missing SourceBridge frontmatter")
	}
	// Must have page blocks.
	if !strings.Contains(content, "sourcebridge:block") {
		t.Error("Docusaurus output missing block ID markers")
	}
}

// TestDocusaurusSink_ToolFrontmatterBeforeSourceBridge verifies that the
// Docusaurus frontmatter appears BEFORE the SourceBridge frontmatter in the
// document, so tools reading top-level YAML keys see their own keys first.
func TestDocusaurusSink_ToolFrontmatterBeforeSourceBridge(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	writer := newMemStaticWriter()
	sink := markdown.NewDocusaurusSink(writer)

	page := buildPageWithTitle("arch.auth", "Auth")
	if err := sink.WritePage(ctx, page); err != nil {
		t.Fatalf("WritePage: %v", err)
	}

	content := string(writer.files["docs/arch.auth.md"])

	// The Docusaurus "id:" key must come before "sourcebridge:" in the document.
	idIdx := strings.Index(content, "id: arch.auth")
	sbIdx := strings.Index(content, "sourcebridge:")
	if idIdx < 0 {
		t.Fatal("could not find 'id: arch.auth' in output")
	}
	if sbIdx < 0 {
		t.Fatal("could not find 'sourcebridge:' in output")
	}
	if idIdx > sbIdx {
		t.Errorf("Docusaurus id: key (offset %d) must appear before sourcebridge: (offset %d)", idIdx, sbIdx)
	}
}

// TestVitePressSink_OutputPath verifies the VitePress sink path.
func TestVitePressSink_OutputPath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	writer := newMemStaticWriter()
	sink := markdown.NewVitePressSink(writer)

	page := buildPageWithTitle("arch.auth", "Auth")
	if err := sink.WritePage(ctx, page); err != nil {
		t.Fatalf("WritePage: %v", err)
	}

	if _, ok := writer.files["docs/arch.auth.md"]; !ok {
		t.Errorf("expected docs/arch.auth.md, got: %v", fileMapKeys(writer.files))
	}
}

// TestVitePressSink_Frontmatter verifies VitePress-specific frontmatter keys.
func TestVitePressSink_Frontmatter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	writer := newMemStaticWriter()
	sink := markdown.NewVitePressSink(writer)

	page := buildPageWithTitle("arch.auth", "Auth")
	if err := sink.WritePage(ctx, page); err != nil {
		t.Fatalf("WritePage: %v", err)
	}

	content := string(writer.files["docs/arch.auth.md"])

	if !strings.Contains(content, "outline: deep") {
		t.Error("VitePress output missing 'outline: deep' key")
	}
	if !strings.Contains(content, "title:") {
		t.Error("VitePress output missing title key")
	}
	if !strings.Contains(content, "sourcebridge:") {
		t.Error("VitePress output missing SourceBridge frontmatter")
	}
}

// TestVitePressSink_ToolFrontmatterBeforeSourceBridge verifies ordering.
func TestVitePressSink_ToolFrontmatterBeforeSourceBridge(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	writer := newMemStaticWriter()
	sink := markdown.NewVitePressSink(writer)

	page := buildPageWithTitle("arch.auth", "Auth")
	if err := sink.WritePage(ctx, page); err != nil {
		t.Fatalf("WritePage: %v", err)
	}

	content := string(writer.files["docs/arch.auth.md"])

	outlineIdx := strings.Index(content, "outline: deep")
	sbIdx := strings.Index(content, "sourcebridge:")
	if outlineIdx < 0 {
		t.Fatal("could not find 'outline: deep' in output")
	}
	if sbIdx < 0 {
		t.Fatal("could not find 'sourcebridge:' in output")
	}
	if outlineIdx > sbIdx {
		t.Errorf("VitePress outline: key (offset %d) must appear before sourcebridge: (offset %d)",
			outlineIdx, sbIdx)
	}
}
