// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

// Package sinks implements the per-sink write dispatch layer for the living-wiki
// cold-start runner. After the orchestrator generates pages into the proposed_ast
// store, the runner calls DispatchPagesNamed to push each page to every
// configured sink.
//
// # Concurrency model
//
// Pages are dispatched to all sinks in parallel (one goroutine per sink kind).
// Within each sink goroutine, pages are written sequentially so the per-sink
// rate limiter can gate each write without coordination overhead between sinks.
//
// # Failure semantics
//
//   - A per-page write error is collected in FailedPageIDs but does NOT stop
//     the sink. All remaining pages are still attempted.
//   - A non-recoverable error (e.g. HTTP 401 Unauthorized) stops the sink
//     immediately and is stored in SinkWriteSummary.Error.
//   - When all sinks fail with non-recoverable errors the caller should classify
//     the job as FAILED_AUTH or FAILED_TRANSIENT depending on the error kind.
package sinks

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sourcebridge/sourcebridge/internal/livingwiki/ast"
	"github.com/sourcebridge/sourcebridge/internal/livingwiki/markdown"
	"github.com/sourcebridge/sourcebridge/internal/livingwiki/metrics"
)

// SinkWriter is the unified write contract every sink kind implements.
// One instance per (repo, sink kind, integration name) per job.
type SinkWriter interface {
	// Kind returns the rate-limiter bucket key for this sink.
	Kind() markdown.SinkKind

	// WritePage pushes one generated page to the sink. A non-nil error from
	// WritePage is classified by IsAuthError before deciding whether to stop the
	// sink or continue with the next page.
	WritePage(ctx context.Context, page ast.Page) error
}

// NamedSinkWriter pairs a SinkWriter with its human-readable integration name.
// The name is used as the key in DispatchResult.PerSink so callers can
// identify each sink instance when a repo has multiple sinks of the same kind.
type NamedSinkWriter struct {
	// Name is the integration name (e.g. "eng-docs", "product-wiki").
	Name string
	// Writer is the underlying SinkWriter implementation.
	Writer SinkWriter
}

// SinkWriteSummary records the outcome of dispatching pages to one sink.
type SinkWriteSummary struct {
	// Kind identifies the sink type (e.g. SinkKindConfluence).
	Kind markdown.SinkKind
	// IntegrationName is the human-readable label for this sink instance.
	IntegrationName string
	// PagesWritten is the count of pages that were successfully pushed.
	PagesWritten int
	// PagesFailed is the count of pages that returned a per-page error.
	PagesFailed int
	// FailedPageIDs lists the IDs of pages whose write calls returned an error.
	FailedPageIDs []string
	// Error is set when a non-recoverable error stopped this sink early (e.g.
	// a 401 Unauthorized from the external API). When nil the sink ran to
	// completion (even if some individual page writes failed).
	Error error
}

// DispatchResult aggregates per-sink outcomes for one dispatch call.
type DispatchResult struct {
	// PerSink maps integration name → summary. Keys are NamedSinkWriter.Name.
	PerSink map[string]SinkWriteSummary
}

// IsAuthError reports whether err represents an authentication/authorization
// failure that should stop the sink and classify the job as FAILED_AUTH.
// Recognises HTTP 401 and 403 status codes from the confluence and notion
// typed API errors.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	var ce *markdown.ConfluenceAPIError
	if errors.As(err, &ce) {
		return ce.StatusCode == http.StatusUnauthorized || ce.StatusCode == http.StatusForbidden
	}
	var ne *markdown.NotionAPIError
	if errors.As(err, &ne) {
		return ne.StatusCode == http.StatusUnauthorized || ne.StatusCode == http.StatusForbidden
	}
	return false
}

// IsMissingCredentialsError reports whether err is an *ErrMissingCredentials.
func IsMissingCredentialsError(err error) bool {
	var e *ErrMissingCredentials
	return errors.As(err, &e)
}

// DispatchPagesNamed pushes each generated page to every configured sink.
//
// Sinks run in parallel goroutines (one per named writer). Within each sink,
// pages are written sequentially so the rate limiter can gate individual calls
// without cross-sink coordination.
//
// rateLimiter may be nil (no rate limiting). metricsCollector may be nil (no
// per-write latency recording).
//
// The function always returns a DispatchResult. An error is returned only when
// the context is cancelled; per-sink failures are recorded in the result.
func DispatchPagesNamed(
	ctx context.Context,
	pages []ast.Page,
	writers []NamedSinkWriter,
	rateLimiter markdown.SinkRateLimiter,
	metricsCollector *metrics.Collector,
) (DispatchResult, error) {
	result := DispatchResult{
		PerSink: make(map[string]SinkWriteSummary, len(writers)),
	}
	if len(pages) == 0 || len(writers) == 0 {
		return result, nil
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, nsw := range writers {
		nsw := nsw // capture for goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			summary := dispatchToSink(ctx, pages, nsw.Writer, rateLimiter, metricsCollector)
			summary.IntegrationName = nsw.Name
			mu.Lock()
			result.PerSink[nsw.Name] = summary
			mu.Unlock()
		}()
	}

	wg.Wait()

	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	return result, nil
}

// dispatchToSink runs the write loop for a single sink, returning a summary.
func dispatchToSink(
	ctx context.Context,
	pages []ast.Page,
	sw SinkWriter,
	rateLimiter markdown.SinkRateLimiter,
	mc *metrics.Collector,
) SinkWriteSummary {
	summary := SinkWriteSummary{
		Kind: sw.Kind(),
	}

	for _, page := range pages {
		if ctx.Err() != nil {
			break
		}

		// Gate on rate limiter before each write.
		if rateLimiter != nil {
			if err := rateLimiter.Allow(ctx, sw.Kind()); err != nil {
				// Context cancelled — stop this sink.
				break
			}
		}

		writeStart := time.Now()
		err := sw.WritePage(ctx, page)
		writeDuration := time.Since(writeStart).Seconds()

		if mc != nil {
			mc.RecordSinkWrite(string(sw.Kind()), writeDuration)
		}
		if rateLimiter != nil {
			rateLimiter.Record(sw.Kind())
		}

		if err != nil {
			if IsAuthError(err) {
				// Non-recoverable: stop this sink immediately.
				summary.Error = fmt.Errorf("sink %s: auth error writing page %q: %w", sw.Kind(), page.ID, err)
				summary.PagesFailed++
				summary.FailedPageIDs = append(summary.FailedPageIDs, page.ID)
				return summary
			}
			// Per-page failure: record and continue.
			summary.PagesFailed++
			summary.FailedPageIDs = append(summary.FailedPageIDs, page.ID)
			continue
		}

		summary.PagesWritten++
	}

	return summary
}

// DispatchSummaryStatus classifies a DispatchResult into one of three terminal
// job statuses:
//
//   - "ok"      — all sinks wrote all pages without error
//   - "partial" — some pages or sinks failed but at least one page landed
//   - "failed"  — all sinks failed (zero writes total)
func DispatchSummaryStatus(result DispatchResult) string {
	if len(result.PerSink) == 0 {
		return "ok"
	}
	totalWritten := 0
	totalFailed := 0
	for _, s := range result.PerSink {
		totalWritten += s.PagesWritten
		totalFailed += s.PagesFailed
	}
	if totalWritten == 0 && totalFailed > 0 {
		return "failed"
	}
	if totalFailed > 0 {
		return "partial"
	}
	return "ok"
}
