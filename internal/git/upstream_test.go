// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package git

import (
	"testing"
	"time"
)

func TestParseLsRemoteHead_BranchMatch(t *testing.T) {
	out := "deadbeef1234567890abcdef\trefs/heads/main\n"
	sha := parseLsRemoteHead(out, "main")
	if sha != "deadbeef1234567890abcdef" {
		t.Fatalf("got %q", sha)
	}
}

func TestParseLsRemoteHead_BranchMismatch(t *testing.T) {
	out := "deadbeef1234567890abcdef\trefs/heads/develop\n"
	sha := parseLsRemoteHead(out, "main")
	if sha != "" {
		t.Fatalf("expected empty, got %q", sha)
	}
}

func TestParseLsRemoteHead_SymrefHead(t *testing.T) {
	// --symref HEAD output has a leading ref: line and then the sha/HEAD
	// pair.
	out := "ref: refs/heads/main\tHEAD\n" +
		"1234567890abcdef1234567890abcdef\tHEAD\n"
	sha := parseLsRemoteHead(out, "")
	if sha != "1234567890abcdef1234567890abcdef" {
		t.Fatalf("got %q", sha)
	}
}

func TestParseLsRemoteHead_Empty(t *testing.T) {
	if got := parseLsRemoteHead("", "main"); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestUpstreamHead_Fresh(t *testing.T) {
	// Nil / zero-time heads are never fresh.
	var nilHead *UpstreamHead
	if nilHead.Fresh(time.Minute) {
		t.Error("nil head should not be fresh")
	}
	zeroHead := &UpstreamHead{}
	if zeroHead.Fresh(time.Minute) {
		t.Error("zero-time head should not be fresh")
	}
	recent := &UpstreamHead{CheckedAt: time.Now().Add(-10 * time.Second)}
	if !recent.Fresh(time.Minute) {
		t.Error("10s-old head should be fresh within 1min TTL")
	}
	stale := &UpstreamHead{CheckedAt: time.Now().Add(-2 * time.Minute)}
	if stale.Fresh(time.Minute) {
		t.Error("2min-old head should be stale within 1min TTL")
	}
}

func TestUpstreamCache_PeekAndCacheKey(t *testing.T) {
	c := NewUpstreamCache(time.Minute)
	// Peek with no entry returns nil.
	if got := c.Peek("https://example.com/repo.git", "main"); got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
	// Manually seed via the internal map to verify Peek works.
	c.mu.Lock()
	c.entries[cacheKey("https://example.com/repo.git", "main")] = &UpstreamHead{
		CommitSHA: "cafe",
		CheckedAt: time.Now(),
	}
	c.mu.Unlock()
	if got := c.Peek("https://example.com/repo.git", "main"); got == nil || got.CommitSHA != "cafe" {
		t.Errorf("peek: got %+v", got)
	}
	// Peek with a different branch is a cache miss.
	if got := c.Peek("https://example.com/repo.git", "develop"); got != nil {
		t.Errorf("expected miss for different branch, got %+v", got)
	}
}
