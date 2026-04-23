// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// UpstreamHead is the result of a cached git ls-remote lookup against a
// repository's canonical remote.
type UpstreamHead struct {
	// CommitSHA is the upstream HEAD for the requested branch, or empty if
	// the check failed / the remote is unreachable.
	CommitSHA string
	// CheckedAt is the wall-clock time of the lookup.
	CheckedAt time.Time
	// Err is non-nil when the lookup failed. Callers should treat a
	// non-nil Err as "unknown" rather than "behind" — network blips
	// should not spook the UI.
	Err error
}

// Fresh returns true if the cached head is still within TTL of now.
func (u *UpstreamHead) Fresh(ttl time.Duration) bool {
	if u == nil || u.CheckedAt.IsZero() {
		return false
	}
	return time.Since(u.CheckedAt) < ttl
}

// UpstreamCache is an in-process TTL cache for remote HEAD lookups.
// Kept small on purpose — the goal is to absorb polling bursts from
// multiple concurrent page viewers, not to be a long-term store. On
// process restart the cache is warm again within one polling cycle.
type UpstreamCache struct {
	mu      sync.Mutex
	entries map[string]*UpstreamHead
	ttl     time.Duration
	// inflight deduplicates concurrent requests for the same key so a
	// thundering herd produces a single git ls-remote call.
	inflight map[string]*inflightCall
}

type inflightCall struct {
	done chan struct{}
	head *UpstreamHead
}

// NewUpstreamCache returns a new cache with the given TTL. A typical TTL
// is 30 seconds — long enough to absorb page-refresh polling, short
// enough that "N commits behind" updates feel responsive.
func NewUpstreamCache(ttl time.Duration) *UpstreamCache {
	return &UpstreamCache{
		entries:  make(map[string]*UpstreamHead),
		ttl:      ttl,
		inflight: make(map[string]*inflightCall),
	}
}

// Lookup returns the cached upstream head, refreshing if stale. The
// fetch is synchronous and short (one git ls-remote). Callers should
// invoke this with a context that has a sensible timeout — we suggest
// ~10 seconds so a hung remote doesn't block page load.
func (c *UpstreamCache) Lookup(ctx context.Context, remoteURL, branch, token string) *UpstreamHead {
	if remoteURL == "" {
		return &UpstreamHead{Err: fmt.Errorf("no remote url")}
	}

	key := cacheKey(remoteURL, branch)

	// Fast path: cached + fresh.
	c.mu.Lock()
	if cached, ok := c.entries[key]; ok && cached.Fresh(c.ttl) {
		out := *cached
		c.mu.Unlock()
		return &out
	}

	// Dedupe concurrent callers.
	if inflight, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		<-inflight.done
		if inflight.head != nil {
			out := *inflight.head
			return &out
		}
		return &UpstreamHead{Err: fmt.Errorf("upstream check failed")}
	}

	// We're the elected fetcher.
	call := &inflightCall{done: make(chan struct{})}
	c.inflight[key] = call
	c.mu.Unlock()

	head := fetchUpstreamHead(ctx, remoteURL, branch, token)

	c.mu.Lock()
	c.entries[key] = head
	call.head = head
	delete(c.inflight, key)
	close(call.done)
	c.mu.Unlock()

	out := *head
	return &out
}

// Peek returns the most recent cached entry without triggering a
// refresh. Useful for first-render paths that want to return fast and
// let a follow-up poll pull a fresh value.
func (c *UpstreamCache) Peek(remoteURL, branch string) *UpstreamHead {
	if remoteURL == "" {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.entries[cacheKey(remoteURL, branch)]; ok {
		out := *entry
		return &out
	}
	return nil
}

func cacheKey(remoteURL, branch string) string {
	return remoteURL + "\x00" + branch
}

// fetchUpstreamHead runs `git ls-remote --heads <url> <branch>` and
// parses the first matching SHA. Token (if provided) is injected into
// HTTPS URLs using the same x-access-token trick we use elsewhere.
func fetchUpstreamHead(ctx context.Context, remoteURL, branch, token string) *UpstreamHead {
	now := time.Now().UTC()

	// If no branch is provided, ls-remote HEAD. Otherwise ls-remote the
	// specific ref. Ls-remote does not require a working-tree clone.
	url := remoteURL
	if token != "" {
		if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
			for _, prefix := range []string{"https://", "http://"} {
				if strings.HasPrefix(url, prefix) {
					url = prefix + "x-access-token:" + token + "@" + url[len(prefix):]
					break
				}
			}
		}
	}

	args := []string{"ls-remote"}
	if branch != "" {
		args = append(args, "--heads", url, branch)
	} else {
		args = append(args, "--symref", url, "HEAD")
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	// Reduce friction for first-time remotes; we never prompt.
	cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=/bin/true")
	out, err := cmd.Output()
	if err != nil {
		return &UpstreamHead{CheckedAt: now, Err: fmt.Errorf("ls-remote: %w", err)}
	}
	sha := parseLsRemoteHead(string(out), branch)
	if sha == "" {
		return &UpstreamHead{CheckedAt: now, Err: fmt.Errorf("ls-remote: no matching ref in output")}
	}
	return &UpstreamHead{CommitSHA: sha, CheckedAt: now}
}

// parseLsRemoteHead extracts the SHA from `git ls-remote` output. When
// `branch` is empty, it parses the HEAD line from `--symref` output
// (which has a leading `ref:` line we must skip). When `branch` is
// set, it matches the first line whose ref ends with `/branch`.
func parseLsRemoteHead(output, branch string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip `ref: refs/heads/main\tHEAD` lines.
		if strings.HasPrefix(line, "ref:") {
			continue
		}
		// Normal format: "<sha>\t<ref>"
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		sha, ref := parts[0], parts[1]
		if len(sha) < 7 {
			continue
		}
		if branch == "" {
			// --symref HEAD emits the HEAD line last; just take the first
			// non-ref SHA we see.
			return sha
		}
		// ref looks like "refs/heads/<branch>" — match the suffix.
		if strings.HasSuffix(ref, "/"+branch) {
			return sha
		}
	}
	return ""
}
