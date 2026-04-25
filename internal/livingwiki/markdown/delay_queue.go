// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

// Package markdown — delay_queue.go implements the configurable delay queue
// used by the GitHub and GitLab built-in wiki sinks (A1.P3).
//
// These sinks have no PR review step, so SourceBridge provides a configurable
// hold period before pushing. During the hold, changes are buffered; at flush
// time the full batch is pushed in one operation and an optional email digest
// is sent.
package markdown

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// EmailNotifier is the port for sending email digests when the delay queue
// flushes. The default in-process implementation is [NoopEmailNotifier], which
// does nothing. Wire a real SMTP or transactional-email implementation at the
// call site.
type EmailNotifier interface {
	// SendDigest sends an email digest to the configured address.
	// subject is a one-line summary; body is the full plain-text or HTML body.
	SendDigest(ctx context.Context, to, subject, body string) error
}

// NoopEmailNotifier is an [EmailNotifier] that silently discards all digests.
// It is the default when no real notifier is configured.
type NoopEmailNotifier struct{}

// SendDigest implements [EmailNotifier] by doing nothing.
func (NoopEmailNotifier) SendDigest(_ context.Context, _, _, _ string) error { return nil }

// Compile-time interface check.
var _ EmailNotifier = NoopEmailNotifier{}

// DelayQueueConfig configures a [DelayQueue].
type DelayQueueConfig struct {
	// Delay is how long a write is held before the queue flushes it.
	// Set to 0 or a negative value for "instant" (no delay — writes are flushed
	// immediately). The default 24-hour delay matches GitHub/GitLab wiki push
	// best-practice: give maintainers a chance to see the batch before it lands.
	Delay time.Duration

	// DigestEmail is the address that receives the flush digest.
	// When empty, no digest is sent.
	DigestEmail string

	// DigestSchedule is a human-readable description of the digest schedule, e.g.
	// "daily at 09:00 UTC". Included in digest emails for reader context. Optional.
	DigestSchedule string
}

// QueuedWrite is one page write buffered in the delay queue.
type QueuedWrite struct {
	// PageID is the stable page identifier.
	PageID string
	// Path is the target file path (e.g. "wiki/arch.auth.md").
	Path string
	// Content is the rendered markdown bytes.
	Content []byte
	// QueuedAt is when the write entered the queue.
	QueuedAt time.Time
}

// DelayQueue buffers wiki page writes and flushes them after the configured
// delay. It is safe for concurrent use.
//
// Typical use:
//
//	q := NewDelayQueue(cfg, notifier, flushFn)
//	q.Enqueue(ctx, write)              // called on every wiki push
//	q.FlushDue(ctx, time.Now())        // called on a ticker (e.g. every minute)
//
// The flush function is responsible for the actual push to the wiki remote.
// The delay queue is purely a time-based buffer.
type DelayQueue struct {
	cfg      DelayQueueConfig
	notifier EmailNotifier
	flushFn  FlushFunc

	mu    sync.Mutex
	items []QueuedWrite
}

// FlushFunc is called by [DelayQueue.FlushDue] with the set of writes that are
// due. The function must push the content to the wiki remote. On error, the
// items remain in the queue and will be retried on the next FlushDue call.
//
// The map key is the file path (e.g. "wiki/arch.auth.md"). Multiple enqueued
// writes for the same path are coalesced: the most-recent content wins.
type FlushFunc func(ctx context.Context, writes map[string][]byte) error

// NewDelayQueue creates a [DelayQueue] with the given config, notifier, and
// flush function. notifier may be nil (treated as [NoopEmailNotifier]).
// flushFn must be non-nil.
func NewDelayQueue(cfg DelayQueueConfig, notifier EmailNotifier, flushFn FlushFunc) *DelayQueue {
	if notifier == nil {
		notifier = NoopEmailNotifier{}
	}
	return &DelayQueue{
		cfg:      cfg,
		notifier: notifier,
		flushFn:  flushFn,
	}
}

// Enqueue adds a write to the queue. If the configured delay is zero or
// negative, FlushDue is called immediately so the write is applied inline.
func (q *DelayQueue) Enqueue(ctx context.Context, w QueuedWrite) error {
	if w.QueuedAt.IsZero() {
		w.QueuedAt = time.Now()
	}

	q.mu.Lock()
	q.items = append(q.items, w)
	immediate := q.cfg.Delay <= 0
	q.mu.Unlock()

	if immediate {
		return q.FlushDue(ctx, w.QueuedAt)
	}
	return nil
}

// FlushDue flushes all writes whose delay has elapsed as of now.
// It calls the flush function with a coalesced map of path → latest-content
// and then sends an email digest if configured.
//
// Writes that are not yet due are left in the queue.
// On flush-function error, the due writes are returned to the queue so they
// will be retried on the next call.
func (q *DelayQueue) FlushDue(ctx context.Context, now time.Time) error {
	q.mu.Lock()
	var due, remaining []QueuedWrite
	for _, item := range q.items {
		if now.Sub(item.QueuedAt) >= q.cfg.Delay {
			due = append(due, item)
		} else {
			remaining = append(remaining, item)
		}
	}
	q.mu.Unlock()

	if len(due) == 0 {
		return nil
	}

	// Coalesce: last writer for each path wins.
	coalesced := make(map[string][]byte, len(due))
	for _, item := range due {
		coalesced[item.Path] = item.Content
	}

	if err := q.flushFn(ctx, coalesced); err != nil {
		// Return due items to the queue for retry.
		q.mu.Lock()
		q.items = append(due, remaining...)
		q.mu.Unlock()
		return fmt.Errorf("delay_queue: flush failed: %w", err)
	}

	// Success: keep only remaining items.
	q.mu.Lock()
	q.items = remaining
	q.mu.Unlock()

	// Send digest if an email address is configured.
	if q.cfg.DigestEmail != "" {
		subject := fmt.Sprintf("SourceBridge wiki update: %d page(s) published", len(coalesced))
		body := buildDigestBody(due, q.cfg.DigestSchedule)
		// Digest failure is non-fatal — the push already happened.
		_ = q.notifier.SendDigest(ctx, q.cfg.DigestEmail, subject, body)
	}

	return nil
}

// PendingCount returns the number of writes currently buffered in the queue.
func (q *DelayQueue) PendingCount() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

// buildDigestBody formats the plain-text digest body.
func buildDigestBody(due []QueuedWrite, schedule string) string {
	var sb strings.Builder
	sb.WriteString("SourceBridge has published the following wiki pages:\n\n")
	seen := make(map[string]bool, len(due))
	for _, w := range due {
		if seen[w.PageID] {
			continue
		}
		seen[w.PageID] = true
		sb.WriteString("  - ")
		sb.WriteString(w.PageID)
		sb.WriteString("\n")
	}
	if schedule != "" {
		sb.WriteString("\nNext scheduled update: ")
		sb.WriteString(schedule)
		sb.WriteString("\n")
	}
	sb.WriteString("\nThis digest was sent automatically by SourceBridge.\n")
	return sb.String()
}
