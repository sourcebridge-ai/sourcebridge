// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSetupClaude_E2E exercises the full setup flow against an in-process
// test server that returns known cluster data (with packages and warnings).
// It verifies that the written CLAUDE.md contains:
//   - Subsystem headings
//   - "N symbols · M packages (...)" summary lines
//   - "Watch out:" lines for clusters with graph-derived advisories
//   - The "Compare X and Y clusters:" prompt form when 2+ clusters present
func TestSetupClaude_E2E(t *testing.T) {
	// Build the fake clusters response that the test server will return.
	fakeResp := map[string]interface{}{
		"repo_id":      "test-repo-123",
		"status":       "ready",
		"retrieved_at": time.Now().UTC().Format(time.RFC3339),
		"clusters": []map[string]interface{}{
			{
				"id":           "cluster:auth",
				"label":        "auth",
				"member_count": 14,
				"representative_symbols": []string{
					"TokenStore.Rotate",
					"Session.Validate",
					"OAuthFlow.Begin",
				},
				"partial":  false,
				"packages": []string{"auth", "middleware", "session"},
				"warnings": []map[string]interface{}{
					{
						"symbol": "TokenStore.Rotate",
						"kind":   "cross-package-callers",
						"detail": "TokenStore.Rotate has callers in auth, api, and worker — coordinate changes across all of them.",
					},
					{
						"symbol": "Session.Validate",
						"kind":   "hot-path",
						"detail": "Session.Validate is on the hot path (highest in-degree in cluster, 8 callers).",
					},
				},
			},
			{
				"id":           "cluster:billing",
				"label":        "billing",
				"member_count": 9,
				"representative_symbols": []string{
					"InvoiceJob.Run",
				},
				"partial":  false,
				"packages": []string{"billing", "stripe"},
				"warnings": []map[string]interface{}{
					{
						"symbol": "InvoiceJob.Run",
						"kind":   "hot-path",
						"detail": "InvoiceJob.Run is on the hot path (highest in-degree in cluster, 3 callers).",
					},
				},
			},
			{
				"id":           "cluster:storage",
				"label":        "storage",
				"member_count": 11,
				"representative_symbols": []string{
					"TxManager.Commit",
				},
				"partial":  false,
				"packages": []string{"db"},
				"warnings": nil,
			},
		},
	}

	// Fake repo info response.
	fakeRepo := map[string]interface{}{
		"id":   "test-repo-123",
		"name": "payments-service",
		"path": "/tmp/payments-service",
	}

	// Fake repositories list (for lookupRepoByPath, though we pass --repo-id directly).
	fakeRepos := []map[string]interface{}{fakeRepo}

	// Spin up the test server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/healthz":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case strings.HasSuffix(r.URL.Path, "/clusters"):
			if err := json.NewEncoder(w).Encode(fakeResp); err != nil {
				t.Errorf("encoding clusters response: %v", err)
			}
		case strings.HasPrefix(r.URL.Path, "/api/v1/repositories/test-repo-123"):
			if err := json.NewEncoder(w).Encode(fakeRepo); err != nil {
				t.Errorf("encoding repo response: %v", err)
			}
		case r.URL.Path == "/api/v1/repositories":
			if err := json.NewEncoder(w).Encode(fakeRepos); err != nil {
				t.Errorf("encoding repos response: %v", err)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Create a temp directory for the output files.
	tmpDir := t.TempDir()

	// Override the flags for this test run.
	origServer := setupClaudeServer
	origRepoID := setupClaudeRepoID
	origDryRun := setupClaudeDryRun
	origNoMCP := setupClaudeNoMCP
	origCI := setupClaudeCI
	origForce := setupClaudeForce
	origCommit := setupClaudeCommitConfig
	origNoSkills := setupClaudeNoSkills

	setupClaudeServer = srv.URL
	setupClaudeRepoID = "test-repo-123"
	setupClaudeDryRun = false
	setupClaudeNoMCP = true    // skip .mcp.json for cleaner test
	setupClaudeCI = false
	setupClaudeForce = false
	setupClaudeCommitConfig = true // skip .gitignore patching
	setupClaudeNoSkills = false

	defer func() {
		setupClaudeServer = origServer
		setupClaudeRepoID = origRepoID
		setupClaudeDryRun = origDryRun
		setupClaudeNoMCP = origNoMCP
		setupClaudeCI = origCI
		setupClaudeForce = origForce
		setupClaudeCommitConfig = origCommit
		setupClaudeNoSkills = origNoSkills
	}()

	// Change working directory to tmpDir so output files go there.
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir to tmpDir: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	// Run the command. The cobra command must have a non-nil context because
	// runSetupClaude calls cmd.Context() to create the request context.
	setupClaudeCmd.SetContext(context.Background())
	if err := runSetupClaude(setupClaudeCmd, nil); err != nil {
		t.Fatalf("runSetupClaude: %v", err)
	}

	// Read the generated CLAUDE.md.
	claudePath := filepath.Join(tmpDir, ".claude", "CLAUDE.md")
	data, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatalf("reading CLAUDE.md: %v", err)
	}
	content := string(data)

	t.Logf("Generated CLAUDE.md:\n%s", content)

	// --- Golden assertions ---

	// Start/end markers.
	assertContains(t, content, "<!-- sourcebridge:start -->", "missing start marker")
	assertContains(t, content, "<!-- sourcebridge:end -->", "missing end marker")

	// Header block.
	assertContains(t, content, "# SourceBridge — payments-service", "missing repo name")
	assertContains(t, content, "Repo ID: test-repo-123", "missing repo ID")
	assertContains(t, content, "Server: "+srv.URL, "missing server URL")

	// "Try this first" should compare the two largest clusters (auth=14 and storage=11
	// after sorting by member count descending: auth>storage>billing).
	assertContains(t, content, "Compare the auth and storage clusters", "Try this first should use Compare form for 2+ clusters")

	// Subsystem headings.
	assertContains(t, content, "## Subsystem: auth", "missing auth heading")
	assertContains(t, content, "## Subsystem: billing", "missing billing heading")
	assertContains(t, content, "## Subsystem: storage", "missing storage heading")

	// Package summary lines — the headline fix.
	assertContains(t, content, "14 symbols · 3 packages (auth, middleware, session)", "missing auth packages summary")
	assertContains(t, content, "9 symbols · 2 packages (billing, stripe)", "missing billing packages summary")
	assertContains(t, content, "11 symbols · 1 package (db)", "missing storage packages summary")

	// Watch out lines — the headline fix.
	assertContains(t, content, "Watch out: TokenStore.Rotate", "missing cross-package-callers warning")
	assertContains(t, content, "Watch out: Session.Validate", "missing hot-path warning for auth")
	assertContains(t, content, "Watch out: InvoiceJob.Run", "missing hot-path warning for billing")

	// Storage has no warnings — the section between "## Subsystem: storage" and
	// the next "## Subsystem:" heading should not contain any "Watch out:" lines.
	storageIdx := strings.Index(content, "## Subsystem: storage")
	if storageIdx < 0 {
		t.Fatal("## Subsystem: storage not found")
	}
	nextHeadingIdx := strings.Index(content[storageIdx+1:], "## Subsystem:")
	var storageSection string
	if nextHeadingIdx >= 0 {
		storageSection = content[storageIdx : storageIdx+1+nextHeadingIdx]
	} else {
		// Last section before end marker.
		storageSection = content[storageIdx:]
	}
	if strings.Contains(storageSection, "Watch out:") {
		t.Errorf("storage section should have no Watch out lines; got:\n%s", storageSection)
	}
}

func assertContains(t *testing.T, content, substr, msg string) {
	t.Helper()
	if !strings.Contains(content, substr) {
		t.Errorf("%s: %q not found in output", msg, substr)
	}
}
