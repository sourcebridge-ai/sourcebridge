// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestLane_UnboundedNoOp(t *testing.T) {
	l := NewLane("x", 0)
	rel, err := l.Acquire(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rel() // must not panic
	rel() // sync.Once-style release is idempotent for no-op too
}

func TestLane_BoundsConcurrency(t *testing.T) {
	l := NewLane("qa.synthesize", 2)

	rel1, err := l.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	rel2, err := l.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Third acquire must block until we release one.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := l.Acquire(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}

	rel1()
	// Now there's a slot; third should succeed promptly.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel2()
	rel3, err := l.Acquire(ctx2)
	if err != nil {
		t.Fatalf("expected success after release, got %v", err)
	}
	rel2()
	rel3()
}

func TestLane_ReleaseIdempotent(t *testing.T) {
	l := NewLane("t", 1)
	rel, err := l.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	rel()
	rel() // second call must not double-release (would deadlock or panic)
	// Prove the slot is free:
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	rel2, err := l.Acquire(ctx)
	if err != nil {
		t.Fatalf("expected free slot, got %v", err)
	}
	rel2()
}

func TestLanes_Registry(t *testing.T) {
	ls := NewLanes()
	ls.Register(NewLane(LaneQASynthesize, 3))
	if got := ls.Get(LaneQASynthesize).Capacity(); got != 3 {
		t.Errorf("capacity = %d, want 3", got)
	}
	// Unregistered lane returns a no-op permissive lane, never nil.
	if l := ls.Get("nonexistent"); l == nil {
		t.Fatal("expected non-nil permissive lane")
	}
}

func TestLane_FairnessUnderLoad(t *testing.T) {
	l := NewLane("q", 2)
	var running atomic.Int32
	var maxRunning atomic.Int32
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			rel, _ := l.Acquire(context.Background())
			n := running.Add(1)
			for {
				prev := maxRunning.Load()
				if n <= prev || maxRunning.CompareAndSwap(prev, n) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			running.Add(-1)
			rel()
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	if got := maxRunning.Load(); got > 2 {
		t.Errorf("max concurrent = %d, want <= 2", got)
	}
}
