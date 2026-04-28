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

func TestPatchGitignore_Create(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")

	changed, err := skillcard.PatchGitignore(path, ".claude/sourcebridge.json")
	if err != nil {
		t.Fatalf("PatchGitignore: %v", err)
	}
	if !changed {
		t.Error("expected changed=true for new file")
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), ".claude/sourcebridge.json") {
		t.Error("entry not written")
	}
}

func TestPatchGitignore_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")

	// First call.
	_, _ = skillcard.PatchGitignore(path, ".claude/sourcebridge.json")

	// Second call — should report no change.
	changed, err := skillcard.PatchGitignore(path, ".claude/sourcebridge.json")
	if err != nil {
		t.Fatalf("PatchGitignore second call: %v", err)
	}
	if changed {
		t.Error("expected changed=false on duplicate entry")
	}

	// File should not contain duplicate lines.
	data, _ := os.ReadFile(path)
	count := strings.Count(string(data), ".claude/sourcebridge.json")
	if count != 1 {
		t.Errorf("expected 1 occurrence, got %d", count)
	}
}

func TestPatchGitignore_AppendToExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")

	existing := "*.log\nnode_modules/\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := skillcard.PatchGitignore(path, ".claude/sourcebridge.json")
	if err != nil {
		t.Fatalf("PatchGitignore: %v", err)
	}
	if !changed {
		t.Error("expected changed=true when appending to existing file")
	}

	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "*.log") {
		t.Error("existing content should be preserved")
	}
	if !strings.Contains(content, ".claude/sourcebridge.json") {
		t.Error("new entry should be present")
	}
}

func TestPatchGitignore_ExistingEntryNoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")

	// File exists but no trailing newline — the entry is already there.
	if err := os.WriteFile(path, []byte(".claude/sourcebridge.json"), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := skillcard.PatchGitignore(path, ".claude/sourcebridge.json")
	if err != nil {
		t.Fatalf("PatchGitignore: %v", err)
	}
	if changed {
		t.Error("expected changed=false when entry already present without trailing newline")
	}
}
