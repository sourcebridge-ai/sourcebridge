// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package skillcard

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// SidecarVersion is the current schema version for .claude/sourcebridge.json.
// Bump this when the schema changes in a backwards-incompatible way.
const SidecarVersion = 1

// Sidecar is the contents of .claude/sourcebridge.json.
// It records the repo ID, server URL, and the list of files managed by setup.
type Sidecar struct {
	// Version allows future migration without breaking existing installs.
	Version int `json:"version"`
	// RepoID is the stable SourceBridge repository identifier.
	RepoID string `json:"repo_id"`
	// ServerURL is the SourceBridge instance the data was fetched from.
	ServerURL string `json:"server_url"`
	// LastIndexAt is the ISO-8601 timestamp of the most recent index run.
	LastIndexAt string `json:"last_index_at"`
	// GeneratedFiles lists all files written or managed by setup claude.
	GeneratedFiles []string `json:"generated_files"`
	// WrittenHash is the SHA-256 of the generated region as written to
	// CLAUDE.md. Used to detect user edits on the next run.
	WrittenHash string `json:"written_hash,omitempty"`
}

// ReadSidecar reads and parses .claude/sourcebridge.json from baseDir.
// Returns (nil, nil) when the file does not exist.
// Returns (nil, err) on parse failures or unsupported versions.
func ReadSidecar(baseDir string) (*Sidecar, error) {
	path := sidecarPath(baseDir)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading sidecar: %w", err)
	}

	var sc Sidecar
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, fmt.Errorf("parsing sidecar: %w", err)
	}

	// Forward-compat: reject versions we can't handle.
	if sc.Version > SidecarVersion {
		return nil, fmt.Errorf("sidecar version %d is newer than supported version %d — upgrade sourcebridge", sc.Version, SidecarVersion)
	}

	// Migrate v0 (no version field) → v1.
	if sc.Version == 0 {
		sc.Version = SidecarVersion
	}

	return &sc, nil
}

// WriteSidecar serialises sc to .claude/sourcebridge.json under baseDir.
// Parent directories are created as needed.
func WriteSidecar(baseDir string, sc *Sidecar) error {
	sc.Version = SidecarVersion
	data, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling sidecar: %w", err)
	}
	path := sidecarPath(baseDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating .claude/ directory: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing sidecar: %w", err)
	}
	return nil
}

func sidecarPath(baseDir string) string {
	return filepath.Join(baseDir, ".claude", "sourcebridge.json")
}
