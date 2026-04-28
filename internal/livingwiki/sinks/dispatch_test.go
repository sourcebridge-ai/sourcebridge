// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package sinks_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sourcebridge/sourcebridge/internal/livingwiki/ast"
	"github.com/sourcebridge/sourcebridge/internal/livingwiki/markdown"
	"github.com/sourcebridge/sourcebridge/internal/livingwiki/sinks"
)

// ─────────────────────────────────────────────────────────────────────────────
// Test doubles
// ─────────────────────────────────────────────────────────────────────────────

// countingSinkWriter counts WritePage calls and optionally injects errors.
type countingSinkWriter struct {
	kind    markdown.SinkKind
	calls   int32
	pageIDs []string
	// errOnPageID maps page ID → error to return for that specific page.
	errOnPageID map[string]error
	// authErr, if set, is returned for every WritePage call (simulates 401).
	authErr error
}

func newCountingWriter(kind markdown.SinkKind) *countingSinkWriter {
	return &countingSinkWriter{kind: kind, errOnPageID: map[string]error{}}
}

func (c *countingSinkWriter) Kind() markdown.SinkKind { return c.kind }
func (c *countingSinkWriter) WritePage(_ context.Context, page ast.Page) error {
	atomic.AddInt32(&c.calls, 1)
	c.pageIDs = append(c.pageIDs, page.ID)
	if c.authErr != nil {
		return c.authErr
	}
	if err, ok := c.errOnPageID[page.ID]; ok {
		return err
	}
	return nil
}

func (c *countingSinkWriter) callCount() int { return int(atomic.LoadInt32(&c.calls)) }

// makePages creates n test pages with IDs "page-1", "page-2", …
func makePages(n int) []ast.Page {
	pages := make([]ast.Page, n)
	for i := range pages {
		pages[i] = ast.Page{ID: fmt.Sprintf("page-%d", i+1)}
	}
	return pages
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestDispatchHappyPath verifies that all pages are written to all sinks.
func TestDispatchHappyPath(t *testing.T) {
	t.Parallel()

	pages := makePages(5)
	w := newCountingWriter(markdown.SinkKindConfluence)
	writers := []sinks.NamedSinkWriter{{Name: "eng-docs", Writer: w}}

	result, err := sinks.DispatchPagesNamed(context.Background(), pages, writers, nil, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	summary, ok := result.PerSink["eng-docs"]
	if !ok {
		t.Fatal("expected summary for eng-docs")
	}
	if summary.PagesWritten != 5 {
		t.Errorf("expected 5 pages written, got %d", summary.PagesWritten)
	}
	if summary.PagesFailed != 0 {
		t.Errorf("expected 0 pages failed, got %d", summary.PagesFailed)
	}
	if summary.Error != nil {
		t.Errorf("expected nil sink error, got %v", summary.Error)
	}
	if w.callCount() != 5 {
		t.Errorf("expected 5 WritePage calls, got %d", w.callCount())
	}
}

// TestDispatchPerPageFailure verifies that a per-page error is collected without
// stopping the sink — remaining pages continue.
func TestDispatchPerPageFailure(t *testing.T) {
	t.Parallel()

	pages := makePages(5)
	w := newCountingWriter(markdown.SinkKindConfluence)
	w.errOnPageID["page-3"] = errors.New("transient write error on page-3")

	writers := []sinks.NamedSinkWriter{{Name: "eng-docs", Writer: w}}
	result, err := sinks.DispatchPagesNamed(context.Background(), pages, writers, nil, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	summary := result.PerSink["eng-docs"]
	if summary.PagesWritten != 4 {
		t.Errorf("expected 4 written, got %d", summary.PagesWritten)
	}
	if summary.PagesFailed != 1 {
		t.Errorf("expected 1 failed, got %d", summary.PagesFailed)
	}
	if len(summary.FailedPageIDs) != 1 || summary.FailedPageIDs[0] != "page-3" {
		t.Errorf("expected FailedPageIDs=[page-3], got %v", summary.FailedPageIDs)
	}
	if summary.Error != nil {
		t.Errorf("expected nil sink-level error (per-page errors don't stop the sink), got %v", summary.Error)
	}
	// All 5 pages were attempted even though page-3 errored.
	if w.callCount() != 5 {
		t.Errorf("expected 5 WritePage calls (sink continues after per-page error), got %d", w.callCount())
	}
}

// TestDispatchAuthFailureStopsSink verifies that a 401/403 error stops the sink
// immediately and surfaces via SinkWriteSummary.Error.
func TestDispatchAuthFailureStopsSink(t *testing.T) {
	t.Parallel()

	pages := makePages(5)
	w := newCountingWriter(markdown.SinkKindConfluence)
	// Simulate a 401 Unauthorized from Confluence.
	w.authErr = &markdown.ConfluenceAPIError{StatusCode: http.StatusUnauthorized, Message: "unauthorized"}

	writers := []sinks.NamedSinkWriter{{Name: "eng-docs", Writer: w}}
	result, err := sinks.DispatchPagesNamed(context.Background(), pages, writers, nil, nil)

	if err != nil {
		t.Fatalf("unexpected ctx error: %v", err)
	}
	summary := result.PerSink["eng-docs"]
	if summary.Error == nil {
		t.Fatal("expected non-nil sink error for auth failure")
	}
	// The sink should stop after the first auth error — only 1 write attempted.
	if w.callCount() != 1 {
		t.Errorf("expected sink to stop after first auth error (1 call), got %d", w.callCount())
	}
	if summary.PagesWritten != 0 {
		t.Errorf("expected 0 pages written on auth error, got %d", summary.PagesWritten)
	}
}

// TestDispatchMultipleSinksParallel verifies that two sinks run independently
// and both receive all pages.
func TestDispatchMultipleSinksParallel(t *testing.T) {
	t.Parallel()

	pages := makePages(3)
	w1 := newCountingWriter(markdown.SinkKindConfluence)
	w2 := newCountingWriter(markdown.SinkKindNotion)

	writers := []sinks.NamedSinkWriter{
		{Name: "confluence-sink", Writer: w1},
		{Name: "notion-sink", Writer: w2},
	}
	result, err := sinks.DispatchPagesNamed(context.Background(), pages, writers, nil, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := result.PerSink["confluence-sink"].PagesWritten; got != 3 {
		t.Errorf("confluence: expected 3 written, got %d", got)
	}
	if got := result.PerSink["notion-sink"].PagesWritten; got != 3 {
		t.Errorf("notion: expected 3 written, got %d", got)
	}
	if w1.callCount() != 3 {
		t.Errorf("confluence writer called %d times, expected 3", w1.callCount())
	}
	if w2.callCount() != 3 {
		t.Errorf("notion writer called %d times, expected 3", w2.callCount())
	}
}

// TestDispatchWithRateLimiter verifies that the rate limiter's Allow method is
// called before each write, and Record after each successful write.
func TestDispatchWithRateLimiter(t *testing.T) {
	t.Parallel()

	pages := makePages(3)
	w := newCountingWriter(markdown.SinkKindConfluence)

	rl := &countingRateLimiter{}
	writers := []sinks.NamedSinkWriter{{Name: "eng", Writer: w}}
	_, _ = sinks.DispatchPagesNamed(context.Background(), pages, writers, rl, nil)

	if rl.allowCalls != 3 {
		t.Errorf("expected 3 Allow calls, got %d", rl.allowCalls)
	}
	if rl.recordCalls != 3 {
		t.Errorf("expected 3 Record calls (all writes succeeded), got %d", rl.recordCalls)
	}
}

// TestDispatchSummaryStatus verifies the status classification logic.
func TestDispatchSummaryStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   sinks.DispatchResult
		expected string
	}{
		{
			name:     "empty result",
			result:   sinks.DispatchResult{PerSink: map[string]sinks.SinkWriteSummary{}},
			expected: "ok",
		},
		{
			name: "all written",
			result: sinks.DispatchResult{PerSink: map[string]sinks.SinkWriteSummary{
				"s1": {PagesWritten: 3, PagesFailed: 0},
			}},
			expected: "ok",
		},
		{
			name: "some failed",
			result: sinks.DispatchResult{PerSink: map[string]sinks.SinkWriteSummary{
				"s1": {PagesWritten: 2, PagesFailed: 1},
			}},
			expected: "partial",
		},
		{
			name: "all failed",
			result: sinks.DispatchResult{PerSink: map[string]sinks.SinkWriteSummary{
				"s1": {PagesWritten: 0, PagesFailed: 3},
			}},
			expected: "failed",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := sinks.DispatchSummaryStatus(tc.result)
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

// TestDispatchEmptyPages returns ok immediately without calling any writers.
func TestDispatchEmptyPages(t *testing.T) {
	t.Parallel()

	w := newCountingWriter(markdown.SinkKindConfluence)
	writers := []sinks.NamedSinkWriter{{Name: "eng", Writer: w}}

	result, err := sinks.DispatchPagesNamed(context.Background(), nil, writers, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.PerSink) != 0 {
		t.Errorf("expected empty PerSink, got %d entries", len(result.PerSink))
	}
	if w.callCount() != 0 {
		t.Errorf("expected 0 WritePage calls, got %d", w.callCount())
	}
}

// TestDispatchContextCancellation stops dispatch when the context is cancelled.
func TestDispatchContextCancellation(t *testing.T) {
	t.Parallel()

	pages := makePages(100)
	ctx, cancel := context.WithCancel(context.Background())

	blockingWriter := &blockOnWriteWriter{cancel: cancel}
	writers := []sinks.NamedSinkWriter{{Name: "slow", Writer: blockingWriter}}

	_, err := sinks.DispatchPagesNamed(ctx, pages, writers, nil, nil)
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test double helpers
// ─────────────────────────────────────────────────────────────────────────────

// countingRateLimiter records Allow and Record calls for assertion.
type countingRateLimiter struct {
	allowCalls  int
	recordCalls int
}

func (r *countingRateLimiter) Allow(_ context.Context, _ markdown.SinkKind) error {
	r.allowCalls++
	return nil
}

func (r *countingRateLimiter) Record(_ markdown.SinkKind) {
	r.recordCalls++
}

// blockOnWriteWriter cancels the context on the first WritePage call, then
// blocks briefly so the cancellation propagates before the next page.
type blockOnWriteWriter struct {
	cancel context.CancelFunc
}

func (b *blockOnWriteWriter) Kind() markdown.SinkKind { return markdown.SinkKindConfluence }
func (b *blockOnWriteWriter) WritePage(ctx context.Context, _ ast.Page) error {
	b.cancel()
	// Give a tiny window for the context to propagate.
	time.Sleep(5 * time.Millisecond)
	return ctx.Err()
}
