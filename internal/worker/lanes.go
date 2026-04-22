// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package worker

import (
	"context"
	"errors"
	"sync"
)

// Lane is a named semaphore that bounds concurrency against the
// shared Python worker. Both search (embedding) and QA (synthesis)
// hit the same worker process; without lanes they starve each other
// under load.
//
// Use Acquire/Release around the outbound RPC. Release is always
// called — even on error — via the pattern:
//
//	release, err := lane.Acquire(ctx)
//	if err != nil { return err }
//	defer release()
//	// ... RPC call ...
type Lane struct {
	name string
	ch   chan struct{}
}

// NewLane returns a lane that caps concurrent acquisitions at capacity.
// Capacity <= 0 disables the lane (acquires return a no-op release).
func NewLane(name string, capacity int) *Lane {
	l := &Lane{name: name}
	if capacity > 0 {
		l.ch = make(chan struct{}, capacity)
	}
	return l
}

// Name returns the lane name (used in logs / metrics).
func (l *Lane) Name() string { return l.name }

// Capacity returns the configured capacity. Zero means the lane is
// disabled (unbounded).
func (l *Lane) Capacity() int {
	if l.ch == nil {
		return 0
	}
	return cap(l.ch)
}

// ErrLaneClosed is returned when Acquire is called on a closed lane.
var ErrLaneClosed = errors.New("lane closed")

// Acquire blocks until the caller holds a lane slot or the context is
// cancelled. It returns a release function the caller must invoke
// (typically via defer) so slots are always returned.
//
// If the lane was created with capacity <= 0 the lane is unbounded
// and Acquire returns a no-op release.
func (l *Lane) Acquire(ctx context.Context) (func(), error) {
	if l.ch == nil {
		return func() {}, nil
	}
	select {
	case l.ch <- struct{}{}:
		var once sync.Once
		return func() {
			once.Do(func() { <-l.ch })
		}, nil
	case <-ctx.Done():
		return func() {}, ctx.Err()
	}
}

// Lanes is the canonical lane registry shared by search.Service and
// the QA orchestrator. Register lanes at startup; look them up by
// name at call sites.
type Lanes struct {
	mu    sync.RWMutex
	lanes map[string]*Lane
}

// NewLanes returns an empty registry.
func NewLanes() *Lanes {
	return &Lanes{lanes: make(map[string]*Lane)}
}

// Register adds or replaces a lane.
func (ls *Lanes) Register(lane *Lane) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.lanes[lane.name] = lane
}

// Get returns the named lane, or a permissive no-op lane (capacity 0)
// if the name is not registered. Never returns nil so callers can
// always defer their release.
func (ls *Lanes) Get(name string) *Lane {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	if l, ok := ls.lanes[name]; ok {
		return l
	}
	// Permissive default: unbounded, so unconfigured lanes don't
	// block traffic. Production operators can register explicit
	// caps; tests get the no-op path for free.
	return NewLane(name, 0)
}

// Canonical lane names shared across the codebase.
const (
	LaneSearchEmbed   = "search.embed"
	LaneQASynthesize  = "qa.synthesize"
)
