// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package orchestrator

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sourcebridge/sourcebridge/internal/llm"
)

// newTestOrchestrator returns an orchestrator wired up against the
// in-memory job store with a short debounce window so tests finish fast.
func newTestOrchestrator(t *testing.T, cfg Config) *Orchestrator {
	t.Helper()
	if cfg.ProgressDebounce == 0 {
		cfg.ProgressDebounce = 5 * time.Millisecond
	}
	if cfg.Retry.MaxAttempts == 0 {
		cfg.Retry = RetryPolicy{MaxAttempts: 1} // no retries by default in tests
	}
	store := llm.NewMemStore()
	orch := New(store, cfg)
	t.Cleanup(func() { _ = orch.Shutdown(2 * time.Second) })
	return orch
}

// waitFor polls the supplied condition until it returns true or the
// timeout elapses. Used instead of time.Sleep to keep tests responsive.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}

func TestOrchestratorEnqueueRunsJobToCompletion(t *testing.T) {
	orch := newTestOrchestrator(t, Config{MaxConcurrency: 2})

	var ran atomic.Bool
	job, err := orch.Enqueue(&llm.EnqueueRequest{
		Subsystem: llm.SubsystemKnowledge,
		JobType:   "cliff_notes",
		TargetKey: "repo-1:cliff_notes:dev:medium",
		Run: func(rt llm.Runtime) error {
			rt.ReportProgress(0.5, "mid", "halfway")
			rt.ReportTokens(1000, 500)
			rt.ReportSnapshotBytes(12345)
			ran.Store(true)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if job == nil || job.ID == "" {
		t.Fatal("expected a job with an id")
	}
	waitFor(t, time.Second, func() bool {
		stored := orch.GetJob(job.ID)
		return stored != nil && stored.Status == llm.StatusReady
	})
	if !ran.Load() {
		t.Fatal("expected the run closure to have executed")
	}
	stored := orch.GetJob(job.ID)
	if stored.InputTokens != 1000 || stored.OutputTokens != 500 {
		t.Fatalf("expected tokens persisted (1000/500), got %d/%d", stored.InputTokens, stored.OutputTokens)
	}
	if stored.SnapshotBytes != 12345 {
		t.Fatalf("expected snapshot bytes 12345, got %d", stored.SnapshotBytes)
	}
	if stored.Progress != 1.0 {
		t.Fatalf("expected terminal progress 1.0, got %v", stored.Progress)
	}
}

func TestOrchestratorDedupeReturnsSameJob(t *testing.T) {
	orch := newTestOrchestrator(t, Config{MaxConcurrency: 1})

	// Use a channel gate so the first job parks until we signal it.
	gate := make(chan struct{})
	firstReq := &llm.EnqueueRequest{
		Subsystem: llm.SubsystemKnowledge,
		JobType:   "cliff_notes",
		TargetKey: "repo-1:dedupe",
		Run: func(rt llm.Runtime) error {
			<-gate
			return nil
		},
	}
	first, err := orch.Enqueue(firstReq)
	if err != nil {
		t.Fatalf("first enqueue failed: %v", err)
	}

	// Give the worker a beat to pick up the job so it's firmly
	// in-flight by the time we call Enqueue again.
	waitFor(t, time.Second, func() bool {
		j := orch.GetJob(first.ID)
		return j != nil && j.Status == llm.StatusGenerating
	})

	// Second request with the same target key should dedupe.
	var secondRan atomic.Bool
	second, err := orch.Enqueue(&llm.EnqueueRequest{
		Subsystem: llm.SubsystemKnowledge,
		JobType:   "cliff_notes",
		TargetKey: "repo-1:dedupe",
		Run: func(rt llm.Runtime) error {
			secondRan.Store(true)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("second enqueue failed: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected dedupe to return first job id %q, got %q", first.ID, second.ID)
	}

	// Release the gate so the job can finish and Shutdown drains cleanly.
	close(gate)
	waitFor(t, time.Second, func() bool {
		j := orch.GetJob(first.ID)
		return j != nil && j.Status == llm.StatusReady
	})
	if secondRan.Load() {
		t.Fatal("second run closure should not have executed after dedupe")
	}
}

func TestOrchestratorBoundedConcurrency(t *testing.T) {
	const maxConcurrency = 3
	const total = 10
	orch := newTestOrchestrator(t, Config{MaxConcurrency: maxConcurrency})

	var running atomic.Int32
	var maxSeen atomic.Int32
	release := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(total)
	for i := 0; i < total; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := orch.Enqueue(&llm.EnqueueRequest{
				Subsystem: llm.SubsystemKnowledge,
				JobType:   "cliff_notes",
				TargetKey: fmt.Sprintf("repo-1:concurrency:%d", i),
				Run: func(rt llm.Runtime) error {
					n := running.Add(1)
					for {
						cur := maxSeen.Load()
						if n <= cur || maxSeen.CompareAndSwap(cur, n) {
							break
						}
					}
					<-release
					running.Add(-1)
					return nil
				},
			})
			if err != nil {
				t.Errorf("enqueue %d failed: %v", i, err)
			}
		}(i)
	}
	wg.Wait() // all enqueue calls returned

	// Wait until the pool is fully occupied.
	waitFor(t, time.Second, func() bool {
		return running.Load() == int32(maxConcurrency)
	})

	// Release the jobs and let everything drain.
	close(release)
	waitFor(t, 2*time.Second, func() bool {
		return running.Load() == 0
	})

	if maxSeen.Load() > int32(maxConcurrency) {
		t.Fatalf("concurrency breached ceiling: observed %d, cap %d", maxSeen.Load(), maxConcurrency)
	}
	if maxSeen.Load() != int32(maxConcurrency) {
		t.Fatalf("expected to saturate at %d concurrent jobs, peak was %d", maxConcurrency, maxSeen.Load())
	}
}

func TestOrchestratorRetryOnTransientError(t *testing.T) {
	orch := newTestOrchestrator(t, Config{
		MaxConcurrency: 1,
		Retry: RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: 5 * time.Millisecond,
			MaxBackoff:     10 * time.Millisecond,
			Multiplier:     1.5,
		},
	})

	var attempts atomic.Int32
	job, err := orch.Enqueue(&llm.EnqueueRequest{
		Subsystem: llm.SubsystemKnowledge,
		JobType:   "cliff_notes",
		TargetKey: "repo-1:retry-transient",
		Run: func(rt llm.Runtime) error {
			n := attempts.Add(1)
			if n < 3 {
				return errors.New("LLM returned empty content (transient)")
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	waitFor(t, 2*time.Second, func() bool {
		j := orch.GetJob(job.ID)
		return j != nil && j.Status == llm.StatusReady
	})

	final := orch.GetJob(job.ID)
	if final.Status != llm.StatusReady {
		t.Fatalf("expected status ready, got %q", final.Status)
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
	if final.RetryCount != 2 {
		t.Fatalf("expected retry_count 2 (two retries after the first attempt), got %d", final.RetryCount)
	}
}

func TestOrchestratorNonRetryableFailsFast(t *testing.T) {
	orch := newTestOrchestrator(t, Config{
		MaxConcurrency: 1,
		Retry: RetryPolicy{
			MaxAttempts:    5, // lots of retries available
			InitialBackoff: 1 * time.Millisecond,
			MaxBackoff:     5 * time.Millisecond,
			Multiplier:     2.0,
		},
	})

	var attempts atomic.Int32
	job, err := orch.Enqueue(&llm.EnqueueRequest{
		Subsystem: llm.SubsystemKnowledge,
		JobType:   "cliff_notes",
		TargetKey: "repo-1:snapshot-too-large",
		Run: func(rt llm.Runtime) error {
			attempts.Add(1)
			return errors.New("snapshot too large (cliff_notes:repository): ~45000 tokens exceeds budget 24000")
		},
	})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	waitFor(t, 2*time.Second, func() bool {
		j := orch.GetJob(job.ID)
		return j != nil && j.Status == llm.StatusFailed
	})

	if attempts.Load() != 1 {
		t.Fatalf("expected exactly 1 attempt for non-retryable error, got %d", attempts.Load())
	}
	final := orch.GetJob(job.ID)
	if final.ErrorCode != "SNAPSHOT_TOO_LARGE" {
		t.Fatalf("expected error_code SNAPSHOT_TOO_LARGE, got %q", final.ErrorCode)
	}
}

func TestOrchestratorPublishesEvents(t *testing.T) {
	orch := newTestOrchestrator(t, Config{MaxConcurrency: 1})
	events, unsubscribe := orch.Subscribe()
	defer unsubscribe()

	_, err := orch.Enqueue(&llm.EnqueueRequest{
		Subsystem: llm.SubsystemKnowledge,
		JobType:   "cliff_notes",
		TargetKey: "repo-1:events",
		Run: func(rt llm.Runtime) error {
			rt.ReportProgress(0.25, "building", "building")
			time.Sleep(15 * time.Millisecond) // cross the debounce window
			rt.ReportProgress(0.75, "finishing", "nearly done")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	seen := make(map[llm.EventKind]int)
	timeout := time.After(2 * time.Second)
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return
			}
			seen[ev.Kind]++
			if seen[llm.EventCompleted] > 0 {
				if seen[llm.EventCreated] == 0 || seen[llm.EventStarted] == 0 {
					t.Fatalf("expected created and started events before completed, got %+v", seen)
				}
				return
			}
		case <-timeout:
			t.Fatalf("did not receive completed event in time; saw %+v", seen)
		}
	}
}

func TestOrchestratorMetricsRecord(t *testing.T) {
	orch := newTestOrchestrator(t, Config{MaxConcurrency: 1})

	for i := 0; i < 5; i++ {
		_, err := orch.Enqueue(&llm.EnqueueRequest{
			Subsystem: llm.SubsystemKnowledge,
			JobType:   "cliff_notes",
			TargetKey: fmt.Sprintf("repo-1:metrics:%d", i),
			Run: func(rt llm.Runtime) error {
				time.Sleep(2 * time.Millisecond)
				return nil
			},
		})
		if err != nil {
			t.Fatalf("enqueue %d failed: %v", i, err)
		}
	}

	waitFor(t, 2*time.Second, func() bool {
		snap := orch.Metrics()
		return snap.Overall.Total == 5 && snap.Overall.Succeeded == 5
	})

	snap := orch.Metrics()
	if snap.Overall.SuccessRate != 1.0 {
		t.Fatalf("expected overall success rate 1.0, got %v", snap.Overall.SuccessRate)
	}
	if bucket, ok := snap.ByJobType["knowledge/cliff_notes"]; !ok {
		t.Fatalf("expected knowledge/cliff_notes bucket in metrics, got keys %v", snap.ByJobType)
	} else if bucket.Total != 5 {
		t.Fatalf("expected 5 samples in knowledge/cliff_notes, got %d", bucket.Total)
	}
}

func TestIsRetryableClassification(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"empty content retryable", errors.New("LLM returned empty content (ctx)"), true},
		{"deadline retryable", errors.New("context deadline exceeded"), true},
		{"snapshot too large not retryable", errors.New("snapshot too large (ctx): 30000 > 24000"), false},
		{"connection refused retryable", errors.New("dial tcp: connection refused"), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsRetryable(tc.err)
			if got != tc.want {
				t.Fatalf("IsRetryable(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestBackoffGrowsExponentially(t *testing.T) {
	p := RetryPolicy{
		MaxAttempts:    5,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     10 * time.Second,
		Multiplier:     2.0,
	}
	if got := p.BackoffFor(1); got != 0 {
		t.Fatalf("BackoffFor(1) = %v, want 0", got)
	}
	if got := p.BackoffFor(2); got != 100*time.Millisecond {
		t.Fatalf("BackoffFor(2) = %v, want 100ms", got)
	}
	if got := p.BackoffFor(3); got != 200*time.Millisecond {
		t.Fatalf("BackoffFor(3) = %v, want 200ms", got)
	}
	if got := p.BackoffFor(4); got != 400*time.Millisecond {
		t.Fatalf("BackoffFor(4) = %v, want 400ms", got)
	}
}
