// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

// Package markdown — git_wiki_sinks.go implements the github_wiki and
// gitlab_wiki sink writers (Workstream A1.P3).
//
// # GitHub wiki sink
//
// GitHub automatically creates a companion `<repo>.wiki.git` repository for
// every repository. Pages are committed as markdown files to that repo's default
// branch (usually `master`). [GitHubWikiSink] renders an [ast.Page] to
// markdown via [Write] and hands the bytes off to a [WikiRepoWriter] (the same
// interface as the source-repo RepoWriter, pointing at the `.wiki.git` remote
// instead of the source remote).
//
// # GitLab wiki sink
//
// GitLab's wiki is the same model. [GitLabWikiSink] is structurally identical to
// [GitHubWikiSink]; it exists as a distinct named type so callers can distinguish
// the two in audit logs and settings UIs.
//
// # Delay queue
//
// Unlike the git_repo sink, built-in wikis have no PR review step. To give
// maintainers visibility before content lands, both sinks run writes through a
// [DelayQueue] before pushing. The delay is configurable down to "instant"
// (Delay=0). The default is 24 hours with an optional email digest on flush.
//
// # Edit policy
//
// Default edit policy for both sinks is require_review_before_promote (G.2).
// Policy enforcement is the governance layer's responsibility; the sink only writes.
package markdown

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/sourcebridge/sourcebridge/internal/livingwiki/ast"
)

// WikiRepoWriter writes rendered wiki files to the wiki remote.
// For GitHub the concrete implementation points at `<repo>.wiki.git`;
// for GitLab it points at the GitLab wiki git remote or API.
//
// The interface is intentionally identical to orchestrator.RepoWriter so the
// same git-push plumbing serves both source-repo and wiki-repo writes.
type WikiRepoWriter interface {
	// WriteFiles writes the given files to the wiki remote.
	// Path keys are wiki-root-relative (e.g. "arch.auth.md").
	WriteFiles(ctx context.Context, files map[string][]byte) error
}

// flushSentinel is a time far in the future used by Flush() to make all queued
// items appear due without a special code path.
var flushSentinel = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)

// GitHubWikiSink writes wiki pages to a GitHub built-in wiki (`<repo>.wiki.git`).
// Writes are buffered through a [DelayQueue] before being pushed. Set
// [DelayQueueConfig.Delay] to 0 for immediate push.
type GitHubWikiSink struct {
	writer WikiRepoWriter
	queue  *DelayQueue
}

// NewGitHubWikiSink creates a GitHubWikiSink that writes via writer, buffered
// by a [DelayQueue] with the given config and email notifier. notifier may be nil.
func NewGitHubWikiSink(writer WikiRepoWriter, cfg DelayQueueConfig, notifier EmailNotifier) *GitHubWikiSink {
	s := &GitHubWikiSink{writer: writer}
	s.queue = NewDelayQueue(cfg, notifier, func(ctx context.Context, files map[string][]byte) error {
		return writer.WriteFiles(ctx, files)
	})
	return s
}

// WritePage renders page to markdown and enqueues it for flush after the
// configured delay. When Delay is zero the push happens inline.
func (s *GitHubWikiSink) WritePage(ctx context.Context, page ast.Page) error {
	rendered, err := renderPageToBytes(page)
	if err != nil {
		return fmt.Errorf("github_wiki: rendering page %q: %w", page.ID, err)
	}
	return s.queue.Enqueue(ctx, QueuedWrite{
		PageID:  page.ID,
		Path:    wikiPagePath(page.ID),
		Content: rendered,
	})
}

// Flush pushes all pending writes immediately, bypassing the configured delay.
// Useful for "publish now" operations from the SourceBridge UI.
func (s *GitHubWikiSink) Flush(ctx context.Context) error {
	return s.queue.FlushDue(ctx, flushSentinel)
}

// PendingCount returns the number of page writes buffered but not yet pushed.
func (s *GitHubWikiSink) PendingCount() int {
	return s.queue.PendingCount()
}

// GitLabWikiSink writes wiki pages to a GitLab built-in wiki.
// Structurally identical to [GitHubWikiSink]; provided as a distinct type for
// audit-log and settings-UI differentiation.
type GitLabWikiSink struct {
	writer WikiRepoWriter
	queue  *DelayQueue
}

// NewGitLabWikiSink creates a GitLabWikiSink that writes via writer, buffered
// by a [DelayQueue] with the given config and email notifier. notifier may be nil.
func NewGitLabWikiSink(writer WikiRepoWriter, cfg DelayQueueConfig, notifier EmailNotifier) *GitLabWikiSink {
	s := &GitLabWikiSink{writer: writer}
	s.queue = NewDelayQueue(cfg, notifier, func(ctx context.Context, files map[string][]byte) error {
		return writer.WriteFiles(ctx, files)
	})
	return s
}

// WritePage renders page to markdown and enqueues it for flush after the
// configured delay.
func (s *GitLabWikiSink) WritePage(ctx context.Context, page ast.Page) error {
	rendered, err := renderPageToBytes(page)
	if err != nil {
		return fmt.Errorf("gitlab_wiki: rendering page %q: %w", page.ID, err)
	}
	return s.queue.Enqueue(ctx, QueuedWrite{
		PageID:  page.ID,
		Path:    wikiPagePath(page.ID),
		Content: rendered,
	})
}

// Flush pushes all pending writes immediately, bypassing the configured delay.
func (s *GitLabWikiSink) Flush(ctx context.Context) error {
	return s.queue.FlushDue(ctx, flushSentinel)
}

// PendingCount returns the number of page writes buffered but not yet pushed.
func (s *GitLabWikiSink) PendingCount() int {
	return s.queue.PendingCount()
}

// ---- shared helpers ----

// renderPageToBytes renders a page to markdown bytes.
func renderPageToBytes(page ast.Page) ([]byte, error) {
	var buf bytes.Buffer
	if err := Write(&buf, page); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// wikiPagePath returns the wiki-root-relative file path for a page ID.
// GitHub and GitLab wiki repos store pages flat by default.
// Example: "arch.auth" → "arch.auth.md"
func wikiPagePath(pageID string) string {
	return pageID + ".md"
}
