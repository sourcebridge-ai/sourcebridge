// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package search

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type fakeEmbedder struct {
	model string
	calls atomic.Int64
	fail  atomic.Bool
}

func (f *fakeEmbedder) Model() string { return f.model }
func (f *fakeEmbedder) Embed(_ context.Context, q string) ([]float32, bool) {
	f.calls.Add(1)
	if f.fail.Load() {
		return nil, false
	}
	// Deterministic fake vector derived from query length.
	v := []float32{float32(len(q)), 1.0, 0.0}
	return v, true
}

func TestCachedEmbedder_HitsCache(t *testing.T) {
	fe := &fakeEmbedder{model: "m1"}
	ce := NewCachedEmbedder(fe, 100, time.Minute, 5, time.Second)

	a, ok := ce.Embed(context.Background(), "hello world")
	if !ok || len(a) == 0 {
		t.Fatalf("first call should succeed")
	}
	b, ok := ce.Embed(context.Background(), "hello world")
	if !ok || len(b) == 0 {
		t.Fatalf("second call should succeed")
	}
	if fe.calls.Load() != 1 {
		t.Errorf("second call should hit cache, calls=%d", fe.calls.Load())
	}
}

func TestCachedEmbedder_CircuitOpens(t *testing.T) {
	fe := &fakeEmbedder{model: "m1"}
	fe.fail.Store(true)
	ce := NewCachedEmbedder(fe, 100, time.Minute, 3, 200*time.Millisecond)

	for i := 0; i < 3; i++ {
		if _, ok := ce.Embed(context.Background(), "q"); ok {
			t.Fatalf("expected failure at attempt %d", i)
		}
	}
	if !ce.CircuitOpen() {
		t.Fatal("breaker should be open after 3 consecutive failures")
	}
	// Additional calls should not reach the backend.
	before := fe.calls.Load()
	ce.Embed(context.Background(), "q2")
	if fe.calls.Load() != before {
		t.Errorf("open breaker should short-circuit; before=%d after=%d", before, fe.calls.Load())
	}

	// Wait for half-open recovery window.
	time.Sleep(220 * time.Millisecond)
	if ce.CircuitOpen() {
		t.Fatal("breaker should be half-open / closed after openDuration")
	}
}

func TestCachedEmbedder_SuccessResetsBreaker(t *testing.T) {
	fe := &fakeEmbedder{model: "m1"}
	fe.fail.Store(true)
	ce := NewCachedEmbedder(fe, 100, time.Minute, 3, time.Second)

	for i := 0; i < 2; i++ {
		_, _ = ce.Embed(context.Background(), "q")
	}
	fe.fail.Store(false)
	// Third call succeeds → resets counter.
	if _, ok := ce.Embed(context.Background(), "q"); !ok {
		t.Fatal("expected success after recovery")
	}
	// Now fail twice — should not open (counter reset).
	fe.fail.Store(true)
	for i := 0; i < 2; i++ {
		_, _ = ce.Embed(context.Background(), "q2")
	}
	if ce.CircuitOpen() {
		t.Fatal("breaker should not be open with 2 < threshold fails after reset")
	}
}
