// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package rest_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sourcebridge/sourcebridge/internal/api/rest"
	"github.com/sourcebridge/sourcebridge/internal/auth"
	"github.com/sourcebridge/sourcebridge/internal/config"
	"github.com/sourcebridge/sourcebridge/internal/livingwiki/orchestrator"
	"github.com/sourcebridge/sourcebridge/internal/livingwiki/webhook"
)

// buildTestDispatcher returns a started webhook.Dispatcher backed by in-memory stores.
func buildTestDispatcher(t *testing.T) *webhook.Dispatcher {
	t.Helper()
	pageStore := orchestrator.NewMemoryPageStore()
	watermarkStore := orchestrator.NewMemoryWatermarkStore()
	registry := orchestrator.NewDefaultRegistry()
	orch := orchestrator.New(orchestrator.Config{}, registry, pageStore)

	deps := webhook.DispatcherDeps{
		Orchestrator:   orch,
		WatermarkStore: watermarkStore,
	}
	cfg := webhook.DispatcherConfig{
		WorkerCount:  1,
		EventTimeout: 200 * time.Millisecond,
	}
	d := webhook.NewDispatcher(deps, cfg)
	if err := d.Start(context.Background()); err != nil {
		t.Fatalf("dispatcher.Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = d.Stop(ctx)
	})
	return d
}

// buildTestServer returns a Server with minimal config. It does NOT start an
// HTTP listener — callers use httptest to invoke the handler directly.
func buildTestServer(t *testing.T, opts ...rest.ServerOption) *rest.Server {
	t.Helper()
	cfg := config.Defaults()
	cfg.Server.HTTPPort = 0
	cfg.Security.JWTSecret = "test-secret-32-bytes-long-padding"
	jwtMgr := auth.NewJWTManager(cfg.Security.JWTSecret, 60, "")
	localAuth := auth.NewLocalAuth(jwtMgr, nil)
	return rest.NewServer(cfg, localAuth, jwtMgr, nil, nil, opts...)
}

// TestWebhookRoutes_WithDispatcher verifies that /webhooks/confluence and
// /webhooks/notion-poll are reachable (non-404) when a dispatcher is wired.
// Without a valid signature, Confluence returns 401; without a repo_id, the
// Notion endpoint returns 400. Either response proves the route is registered.
func TestWebhookRoutes_WithDispatcher(t *testing.T) {
	d := buildTestDispatcher(t)
	s := buildTestServer(t, rest.WithLivingWikiDispatcher(d))

	cases := []struct {
		path    string
		body    string
		wantNot int // must NOT return this status
	}{
		{
			path:    "/webhooks/confluence",
			body:    `{"eventType":"page_updated"}`,
			wantNot: http.StatusNotFound,
		},
		{
			path:    "/webhooks/notion-poll",
			body:    `{"page_id":"p1"}`, // missing repo_id → 400, not 404
			wantNot: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, req)

			if w.Code == tc.wantNot {
				t.Errorf("%s: got %d, want not %d", tc.path, w.Code, tc.wantNot)
			}
		})
	}
}

// TestWebhookRoutes_WithoutDispatcher verifies that the 503 stub routes are
// registered when no dispatcher is wired, so senders get 503 not 404.
func TestWebhookRoutes_WithoutDispatcher(t *testing.T) {
	s := buildTestServer(t) // no WithLivingWikiDispatcher → nil dispatcher

	paths := []string{"/webhooks/confluence", "/webhooks/notion-poll"}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, req)

			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("%s: got %d, want %d", path, w.Code, http.StatusServiceUnavailable)
			}
		})
	}
}

// TestShutdown_WithDispatcher verifies that Server.Shutdown drains the
// dispatcher within the 30-second window and does not block indefinitely.
func TestShutdown_WithDispatcher(t *testing.T) {
	d := buildTestDispatcher(t)
	s := buildTestServer(t, rest.WithLivingWikiDispatcher(d))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown must return before the context expires.
	done := make(chan error, 1)
	go func() { done <- s.Shutdown(ctx) }()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Shutdown: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("Shutdown blocked past context deadline")
	}
}

// TestShutdown_NilDispatcher verifies that Server.Shutdown works normally when
// no dispatcher was wired (embedded mode / kill-switch path).
func TestShutdown_NilDispatcher(t *testing.T) {
	s := buildTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown (no dispatcher): %v", err)
	}
}
