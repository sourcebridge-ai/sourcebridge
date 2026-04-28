// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package skillcard_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sourcebridge/sourcebridge/internal/skillcard"
)

const generated = `<!-- sourcebridge:start -->
# SourceBridge — test-repo
Repo ID: r1 | Indexed: 2026-04-27 | Server: http://localhost
Refresh: ` + "`sourcebridge setup claude --repo-id r1`" + `

## Subsystem: auth
5 symbols · 1 package (auth)
<!-- sourcebridge:end -->
`

func TestMergeFileWithHash_Create(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude", "CLAUDE.md")

	result, hash, err := skillcard.MergeFileWithHash(path, generated, "", skillcard.MergeOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "create" {
		t.Errorf("expected action=create, got %q", result.Action)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "<!-- sourcebridge:start -->") {
		t.Error("file missing start marker")
	}
}

func TestMergeFileWithHash_Unchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude", "CLAUDE.md")

	// First write.
	_, hash, _ := skillcard.MergeFileWithHash(path, generated, "", skillcard.MergeOptions{})

	// Second write with same content — should be unchanged.
	result, _, err := skillcard.MergeFileWithHash(path, generated, hash, skillcard.MergeOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "unchanged" {
		t.Errorf("expected action=unchanged, got %q", result.Action)
	}
}

func TestMergeFileWithHash_OrphanMarker(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude", "CLAUDE.md")

	// Write a file with start marker but no end marker.
	orphan := "<!-- sourcebridge:start -->\n# SourceBridge — test\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(orphan), 0o644); err != nil {
		t.Fatal(err)
	}

	result, _, err := skillcard.MergeFileWithHash(path, generated, "", skillcard.MergeOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "skip-orphan-marker" {
		t.Errorf("expected skip-orphan-marker, got %q", result.Action)
	}

	// With --force the orphan should be repaired.
	result2, _, err2 := skillcard.MergeFileWithHash(path, generated, "", skillcard.MergeOptions{Force: true})
	if err2 != nil {
		t.Fatalf("unexpected error with force: %v", err2)
	}
	if result2.Action != "update" {
		t.Errorf("expected update after --force, got %q", result2.Action)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "<!-- sourcebridge:end -->") {
		t.Error("repaired file should contain end marker")
	}
}

func TestMergeFileWithHash_UserModified(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude", "CLAUDE.md")

	// First write.
	_, hash, _ := skillcard.MergeFileWithHash(path, generated, "", skillcard.MergeOptions{})

	// Simulate user edit: append a line inside the generated region.
	data, _ := os.ReadFile(path)
	modified := strings.Replace(string(data), "## Subsystem: auth", "## Subsystem: auth\nUser added this.", 1)
	_ = os.WriteFile(path, []byte(modified), 0o644)

	// Re-run without --force: should skip.
	result, _, err := skillcard.MergeFileWithHash(path, generated, hash, skillcard.MergeOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "skip-user-modified" {
		t.Errorf("expected skip-user-modified, got %q", result.Action)
	}

	// Re-run with --force: should overwrite.
	result2, _, err2 := skillcard.MergeFileWithHash(path, generated, hash, skillcard.MergeOptions{Force: true})
	if err2 != nil {
		t.Fatalf("unexpected error with force: %v", err2)
	}
	if result2.Action != "update" {
		t.Errorf("expected update after --force, got %q", result2.Action)
	}
}

func TestMergeFileWithHash_CIMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude", "CLAUDE.md")

	// First write.
	_, hash, _ := skillcard.MergeFileWithHash(path, generated, "", skillcard.MergeOptions{})

	// Simulate user edit.
	data, _ := os.ReadFile(path)
	modified := strings.Replace(string(data), "## Subsystem: auth", "## Subsystem: auth\nUser change.", 1)
	_ = os.WriteFile(path, []byte(modified), 0o644)

	// CI mode should return an error.
	_, _, err := skillcard.MergeFileWithHash(path, generated, hash, skillcard.MergeOptions{CI: true})
	if err == nil {
		t.Error("expected error in CI mode for user-modified section")
	}
}

func TestMergeFileWithHash_DryRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude", "CLAUDE.md")

	result, _, err := skillcard.MergeFileWithHash(path, generated, "", skillcard.MergeOptions{DryRun: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "create" {
		t.Errorf("expected create in dry-run, got %q", result.Action)
	}

	// File should NOT have been created.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("dry-run should not create the file")
	}
}

func TestMergeFileWithHash_PreservesUserContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude", "CLAUDE.md")

	// Create a file with user content before the generated region.
	userContent := "# My Project Notes\n\nThis is user-written.\n\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(userContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := skillcard.MergeFileWithHash(path, generated, "", skillcard.MergeOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "My Project Notes") {
		t.Error("user content should be preserved")
	}
	if !strings.Contains(string(data), "<!-- sourcebridge:start -->") {
		t.Error("generated region should be appended")
	}
}
