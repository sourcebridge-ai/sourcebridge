// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package markdown_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sourcebridge/sourcebridge/internal/livingwiki/ast"
	"github.com/sourcebridge/sourcebridge/internal/livingwiki/manifest"
	"github.com/sourcebridge/sourcebridge/internal/livingwiki/markdown"
)

// ---- test helpers ----

// memWikiWriter captures WriteFiles calls in memory.
type memWikiWriter struct {
	files map[string][]byte
	calls int
}

func newMemWikiWriter() *memWikiWriter {
	return &memWikiWriter{files: make(map[string][]byte)}
}

func (m *memWikiWriter) WriteFiles(_ context.Context, files map[string][]byte) error {
	m.calls++
	for k, v := range files {
		cp := make([]byte, len(v))
		copy(cp, v)
		m.files[k] = cp
	}
	return nil
}

func buildSimplePage(id string) ast.Page {
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
					Heading: &ast.HeadingContent{Level: 1, Text: "Test page"},
				},
			},
			{
				ID:    "bp001",
				Kind:  ast.BlockKindParagraph,
				Owner: ast.OwnerGenerated,
				Content: ast.BlockContent{
					Paragraph: &ast.ParagraphContent{Markdown: "This is a test paragraph."},
				},
			},
		},
	}
}

// ---- GitHubWikiSink tests ----

// TestGitHubWikiSink_InstantPush verifies that with Delay=0 the write reaches
// the writer immediately without needing an explicit Flush call.
func TestGitHubWikiSink_InstantPush(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	writer := newMemWikiWriter()
	cfg := markdown.DelayQueueConfig{Delay: 0} // instant
	sink := markdown.NewGitHubWikiSink(writer, cfg, nil)

	page := buildSimplePage("arch.auth")
	if err := sink.WritePage(ctx, page); err != nil {
		t.Fatalf("WritePage: %v", err)
	}

	// With Delay=0, the flush is inline, so the writer must already have the file.
	if writer.calls != 1 {
		t.Errorf("expected 1 WriteFiles call, got %d", writer.calls)
	}
	content, ok := writer.files["arch.auth.md"]
	if !ok {
		t.Fatalf("expected file arch.auth.md in writer, keys: %v", fileMapKeys(writer.files))
	}
	if !strings.Contains(string(content), "Test page") {
		t.Error("rendered content missing heading text")
	}
	if !strings.Contains(string(content), "sourcebridge:block") {
		t.Error("rendered content missing block ID markers")
	}
}

// TestGitHubWikiSink_DelayedPush verifies that with a positive Delay the write
// is buffered and only reaches the writer after FlushDue is called with a
// sufficiently advanced time.
func TestGitHubWikiSink_DelayedPush(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	writer := newMemWikiWriter()
	cfg := markdown.DelayQueueConfig{
		Delay:       24 * time.Hour,
		DigestEmail: "team@example.com",
	}
	sink := markdown.NewGitHubWikiSink(writer, cfg, nil)

	page := buildSimplePage("arch.billing")
	if err := sink.WritePage(ctx, page); err != nil {
		t.Fatalf("WritePage: %v", err)
	}

	// Nothing pushed yet.
	if writer.calls != 0 {
		t.Errorf("expected 0 WriteFiles calls before flush, got %d", writer.calls)
	}
	if sink.PendingCount() != 1 {
		t.Errorf("expected 1 pending item, got %d", sink.PendingCount())
	}

	// Force flush via Flush() (bypasses delay).
	if err := sink.Flush(ctx); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	if writer.calls != 1 {
		t.Errorf("expected 1 WriteFiles call after flush, got %d", writer.calls)
	}
	if sink.PendingCount() != 0 {
		t.Errorf("expected 0 pending items after flush, got %d", sink.PendingCount())
	}
	if _, ok := writer.files["arch.billing.md"]; !ok {
		t.Error("expected arch.billing.md to be written after flush")
	}
}

// TestGitHubWikiSink_CoalescesWrites verifies that two writes for the same page
// are coalesced into one file write with the latest content.
func TestGitHubWikiSink_CoalescesWrites(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	writer := newMemWikiWriter()
	cfg := markdown.DelayQueueConfig{Delay: 24 * time.Hour}
	sink := markdown.NewGitHubWikiSink(writer, cfg, nil)

	page1 := buildSimplePage("arch.auth")
	page2 := buildSimplePage("arch.auth")
	// Overwrite the paragraph text in the second version.
	page2.Blocks[1].Content.Paragraph.Markdown = "Updated paragraph."

	if err := sink.WritePage(ctx, page1); err != nil {
		t.Fatalf("WritePage (first): %v", err)
	}
	if err := sink.WritePage(ctx, page2); err != nil {
		t.Fatalf("WritePage (second): %v", err)
	}

	if err := sink.Flush(ctx); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	// Only one WriteFiles call, with one file.
	if writer.calls != 1 {
		t.Errorf("expected 1 WriteFiles call (coalesced), got %d", writer.calls)
	}
	content := string(writer.files["arch.auth.md"])
	if !strings.Contains(content, "Updated paragraph.") {
		t.Error("coalesced write should contain the latest content")
	}
}

// TestGitLabWikiSink_OutputPath verifies that the GitLab sink writes to the
// correct flat path (no subdirectory).
func TestGitLabWikiSink_OutputPath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	writer := newMemWikiWriter()
	cfg := markdown.DelayQueueConfig{Delay: 0}
	sink := markdown.NewGitLabWikiSink(writer, cfg, nil)

	page := buildSimplePage("system_overview")
	if err := sink.WritePage(ctx, page); err != nil {
		t.Fatalf("WritePage: %v", err)
	}

	if _, ok := writer.files["system_overview.md"]; !ok {
		t.Errorf("expected file system_overview.md, got files: %v", fileMapKeys(writer.files))
	}
}

// TestDelayQueue_FlushDue_LeavesNonDueItems verifies that items not yet due
// are preserved after a partial flush.
func TestDelayQueue_FlushDue_LeavesNonDueItems(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	writer := newMemWikiWriter()
	flushed := make(map[string][]byte)
	flushFn := func(_ context.Context, files map[string][]byte) error {
		for k, v := range files {
			flushed[k] = v
		}
		return writer.WriteFiles(ctx, files)
	}

	cfg := markdown.DelayQueueConfig{Delay: 2 * time.Hour}
	q := markdown.NewDelayQueue(cfg, nil, flushFn)

	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)

	// Item enqueued at now (due after 2h).
	if err := q.Enqueue(ctx, markdown.QueuedWrite{
		PageID:   "arch.auth",
		Path:     "arch.auth.md",
		Content:  []byte("content-a"),
		QueuedAt: now,
	}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// Item enqueued at now+1h (not yet due at now+1h30m).
	if err := q.Enqueue(ctx, markdown.QueuedWrite{
		PageID:   "arch.billing",
		Path:     "arch.billing.md",
		Content:  []byte("content-b"),
		QueuedAt: now.Add(1 * time.Hour),
	}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// Flush at now+2h30m — first item is due (>2h), second is not (only 1h30m).
	if err := q.FlushDue(ctx, now.Add(2*time.Hour+30*time.Minute)); err != nil {
		t.Fatalf("FlushDue: %v", err)
	}

	if _, ok := flushed["arch.auth.md"]; !ok {
		t.Error("arch.auth.md should have been flushed")
	}
	if _, ok := flushed["arch.billing.md"]; ok {
		t.Error("arch.billing.md should NOT have been flushed yet")
	}
	if q.PendingCount() != 1 {
		t.Errorf("expected 1 remaining item, got %d", q.PendingCount())
	}
}

// TestDelayQueue_DigestSent verifies that the email digest is sent on flush.
func TestDelayQueue_DigestSent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var digestTo, digestSubject, digestBody string
	notifier := &captureNotifier{}
	writer := newMemWikiWriter()

	cfg := markdown.DelayQueueConfig{
		Delay:          0,
		DigestEmail:    "team@example.com",
		DigestSchedule: "daily at 09:00 UTC",
	}
	q := markdown.NewDelayQueue(cfg, notifier, func(_ context.Context, files map[string][]byte) error {
		return writer.WriteFiles(ctx, files)
	})

	if err := q.Enqueue(ctx, markdown.QueuedWrite{
		PageID:  "arch.auth",
		Path:    "arch.auth.md",
		Content: []byte("content"),
	}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	digestTo = notifier.lastTo
	digestSubject = notifier.lastSubject
	digestBody = notifier.lastBody

	if digestTo != "team@example.com" {
		t.Errorf("digest to: got %q, want team@example.com", digestTo)
	}
	if !strings.Contains(digestSubject, "wiki update") {
		t.Errorf("unexpected digest subject: %q", digestSubject)
	}
	if !strings.Contains(digestBody, "arch.auth") {
		t.Errorf("digest body should mention page ID, got: %q", digestBody)
	}
	if !strings.Contains(digestBody, "daily at 09:00 UTC") {
		t.Errorf("digest body should mention schedule, got: %q", digestBody)
	}
}

// captureNotifier captures the last SendDigest call.
type captureNotifier struct {
	lastTo      string
	lastSubject string
	lastBody    string
}

func (n *captureNotifier) SendDigest(_ context.Context, to, subject, body string) error {
	n.lastTo = to
	n.lastSubject = subject
	n.lastBody = body
	return nil
}

// fileMapKeys returns the keys of a map for error messages.
func fileMapKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
