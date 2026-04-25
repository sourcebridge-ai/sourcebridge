// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package markdown

import (
	"io"

	"github.com/sourcebridge/sourcebridge/internal/livingwiki/ast"
)

// SinkWriter is the interface every sink adapter must implement.
// A SinkWriter converts an [ast.Page] into the target format and writes the
// result to w.
//
// Each implementation is expected to:
//   - Embed block IDs in the target format's native metadata field
//     (HTML comment for markdown, ac:macro for Confluence, external_id for Notion).
//   - Preserve human-edited blocks by reading back existing content and
//     applying block-level reconciliation before writing.
type SinkWriter interface {
	WritePage(w io.Writer, page ast.Page) error
}

// MarkdownWriter is the markdown sink adapter. It is the reference
// implementation of SinkWriter and the only fully implemented adapter in D.2.
// Confluence and Notion adapters ship in A1.P4.
type MarkdownWriter struct{}

// WritePage implements [SinkWriter] using [Write].
func (MarkdownWriter) WritePage(w io.Writer, page ast.Page) error {
	return Write(w, page)
}

// ConfluenceSinkWriter is the Confluence storage-XHTML sink adapter that
// satisfies the context-free [SinkWriter] interface.
//
// This path renders XHTML without block-level reconciliation. Use
// [ConfluenceWriter.WritePage] for the full reconciliation cycle when a
// [ConfluenceClient] is available.
type ConfluenceSinkWriter struct{}

// WritePage implements [SinkWriter] by rendering the page to Confluence storage
// XHTML via [WriteXHTML]. No reconciliation is performed.
func (ConfluenceSinkWriter) WritePage(w io.Writer, page ast.Page) error {
	return WriteXHTML(w, page)
}

// NotionSinkWriter is the Notion blocks sink adapter that satisfies the
// context-free [SinkWriter] interface.
//
// This path renders a JSON block array without block-level reconciliation. Use
// [NotionWriter.WritePage] for the full reconciliation cycle when a
// [NotionClient] is available.
type NotionSinkWriter struct{}

// WritePage implements [SinkWriter] by rendering the page to a JSON array of
// Notion block objects via [WriteNotionBlocks]. No reconciliation is performed.
func (NotionSinkWriter) WritePage(w io.Writer, page ast.Page) error {
	return WriteNotionBlocks(w, page)
}

// Compile-time interface checks.
var (
	_ SinkWriter = MarkdownWriter{}
	_ SinkWriter = ConfluenceSinkWriter{}
	_ SinkWriter = NotionSinkWriter{}
)
