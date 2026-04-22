// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package qa

import (
	"testing"
	"time"
)

func TestAsksTotal14d_Accumulates(t *testing.T) {
	resetForTest()
	for i := 0; i < 5; i++ {
		CountAsk()
	}
	if n := AsksTotal14d(); n != 5 {
		t.Errorf("expected 5, got %d", n)
	}
}

func TestAsksTotal14d_IgnoresStaleSlots(t *testing.T) {
	resetForTest()
	// Inject a slot that's 30 days old — must not be counted.
	stale := utcDay(time.Now()) - 30
	asksMu.Lock()
	asksRing[stale%int64(len(asksRing))] = askBucket{day: stale, count: 1000}
	asksMu.Unlock()
	CountAsk() // today = 1
	if n := AsksTotal14d(); n != 1 {
		t.Errorf("stale slot should be ignored, got %d", n)
	}
}

func TestServerSideEnabled_Roundtrip(t *testing.T) {
	resetForTest()
	if ServerSideEnabled() {
		t.Error("default must be false")
	}
	SetServerSideEnabled(true)
	if !ServerSideEnabled() {
		t.Error("expected true after set")
	}
	SetServerSideEnabled(false)
	if ServerSideEnabled() {
		t.Error("expected false after re-set")
	}
}
