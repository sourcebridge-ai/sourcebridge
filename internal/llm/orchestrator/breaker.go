// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package orchestrator

import (
	"sync"
	"time"

	"github.com/sourcebridge/sourcebridge/internal/llm"
)

type breakerState struct {
	consecutiveComputeFailures int
	openUntil                  time.Time
}

// subsystemBreaker applies a short cooldown after repeated provider
// compute failures for a subsystem. This prevents the queue from
// immediately re-hammering an unstable backend.
type subsystemBreaker struct {
	mu        sync.Mutex
	threshold int
	cooldown  time.Duration
	states    map[llm.Subsystem]*breakerState
}

func newSubsystemBreaker(threshold int, cooldown time.Duration) *subsystemBreaker {
	if threshold <= 0 || cooldown <= 0 {
		return &subsystemBreaker{threshold: 0, cooldown: 0, states: map[llm.Subsystem]*breakerState{}}
	}
	return &subsystemBreaker{
		threshold: threshold,
		cooldown:  cooldown,
		states:    make(map[llm.Subsystem]*breakerState),
	}
}

func (b *subsystemBreaker) waitDuration(subsystem llm.Subsystem) time.Duration {
	if b == nil || b.threshold <= 0 || b.cooldown <= 0 || subsystem == "" {
		return 0
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	state := b.states[subsystem]
	if state == nil || state.openUntil.IsZero() {
		return 0
	}
	now := time.Now()
	if !state.openUntil.After(now) {
		state.openUntil = time.Time{}
		return 0
	}
	return state.openUntil.Sub(now)
}

func (b *subsystemBreaker) recordSuccess(subsystem llm.Subsystem) {
	if b == nil || subsystem == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	state := b.states[subsystem]
	if state == nil {
		return
	}
	state.consecutiveComputeFailures = 0
	state.openUntil = time.Time{}
}

func (b *subsystemBreaker) recordFailure(subsystem llm.Subsystem, code string) {
	if b == nil || b.threshold <= 0 || b.cooldown <= 0 || subsystem == "" {
		return
	}
	if code != "PROVIDER_COMPUTE" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	state := b.states[subsystem]
	if state == nil {
		state = &breakerState{}
		b.states[subsystem] = state
	}
	state.consecutiveComputeFailures++
	if state.consecutiveComputeFailures >= b.threshold {
		state.openUntil = time.Now().Add(b.cooldown)
	}
}
