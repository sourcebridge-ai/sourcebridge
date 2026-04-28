// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package skillcard_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sourcebridge/sourcebridge/internal/skillcard"
)

func TestSidecar_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	sc := &skillcard.Sidecar{
		RepoID:         "abc123",
		ServerURL:      "https://sb.example.com",
		LastIndexAt:    "2026-04-27T09:45:00Z",
		GeneratedFiles: []string{".claude/CLAUDE.md"},
		WrittenHash:    "deadbeef",
	}

	if err := skillcard.WriteSidecar(dir, sc); err != nil {
		t.Fatalf("WriteSidecar: %v", err)
	}

	got, err := skillcard.ReadSidecar(dir)
	if err != nil {
		t.Fatalf("ReadSidecar: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil sidecar")
	}
	if got.RepoID != sc.RepoID {
		t.Errorf("RepoID: want %q, got %q", sc.RepoID, got.RepoID)
	}
	if got.ServerURL != sc.ServerURL {
		t.Errorf("ServerURL: want %q, got %q", sc.ServerURL, got.ServerURL)
	}
	if got.LastIndexAt != sc.LastIndexAt {
		t.Errorf("LastIndexAt: want %q, got %q", sc.LastIndexAt, got.LastIndexAt)
	}
	if got.Version != skillcard.SidecarVersion {
		t.Errorf("Version: want %d, got %d", skillcard.SidecarVersion, got.Version)
	}
	if got.WrittenHash != sc.WrittenHash {
		t.Errorf("WrittenHash: want %q, got %q", sc.WrittenHash, got.WrittenHash)
	}
}

func TestSidecar_NotExist(t *testing.T) {
	sc, err := skillcard.ReadSidecar(t.TempDir())
	if err != nil {
		t.Fatalf("expected nil error for missing sidecar, got: %v", err)
	}
	if sc != nil {
		t.Error("expected nil sidecar for missing file")
	}
}

func TestSidecar_MigrationV0(t *testing.T) {
	dir := t.TempDir()
	// Write a v0 sidecar (no version field).
	v0 := `{"repo_id":"r1","server_url":"http://localhost","last_index_at":"2026-01-01T00:00:00Z","generated_files":[".claude/CLAUDE.md"]}`
	path := filepath.Join(dir, ".claude", "sourcebridge.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(v0), 0o644); err != nil {
		t.Fatal(err)
	}

	sc, err := skillcard.ReadSidecar(dir)
	if err != nil {
		t.Fatalf("ReadSidecar: %v", err)
	}
	if sc.Version != skillcard.SidecarVersion {
		t.Errorf("expected version migrated to %d, got %d", skillcard.SidecarVersion, sc.Version)
	}
	if sc.RepoID != "r1" {
		t.Errorf("expected RepoID=r1, got %q", sc.RepoID)
	}
}

func TestSidecar_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude", "sourcebridge.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not json{"), 0o644); err != nil {
		t.Fatal(err)
	}

	sc, err := skillcard.ReadSidecar(dir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if sc != nil {
		t.Error("expected nil sidecar for invalid JSON")
	}
}
