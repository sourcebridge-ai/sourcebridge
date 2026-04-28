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

// MCPEntry is the entry written under mcpServers.sourcebridge.
type MCPEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// MergeMCPJSON idempotently merges the SourceBridge MCP server entry into
// the .mcp.json file at path.
//
// Rules (verbatim from the plan):
//   - Parse existing .mcp.json if present.
//   - Add mcpServers.sourcebridge if absent.
//   - If present and command matches, leave existing args/env untouched.
//   - If present and command differs, return an error unless force is true.
//   - Foreign keys outside mcpServers.sourcebridge are preserved.
//   - Invalid JSON: back up to .mcp.json.sb-backup, write a fresh file with
//     only the SourceBridge entry, and set warningMsg.
func MergeMCPJSON(path, repoID string, force bool) (changed bool, warningMsg string, err error) {
	entry := MCPEntry{
		Command: "sourcebridge",
		Args:    []string{"mcp", "--repo-id", repoID},
		Env:     map[string]string{},
	}

	existing, readErr := os.ReadFile(path)
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return false, "", fmt.Errorf("reading %s: %w", path, readErr)
	}

	var doc map[string]json.RawMessage
	if len(existing) > 0 {
		if jsonErr := json.Unmarshal(existing, &doc); jsonErr != nil {
			// Invalid JSON: back up and start fresh.
			backupPath := path + ".sb-backup"
			_ = os.WriteFile(backupPath, existing, 0o644)
			warningMsg = fmt.Sprintf("Existing %s contained invalid JSON — backed up to %s and replaced.", path, backupPath)
			doc = nil
		}
	}

	if doc == nil {
		doc = make(map[string]json.RawMessage)
	}

	// Parse or create the mcpServers object.
	var mcpServers map[string]json.RawMessage
	if raw, ok := doc["mcpServers"]; ok {
		if err := json.Unmarshal(raw, &mcpServers); err != nil {
			mcpServers = make(map[string]json.RawMessage)
		}
	} else {
		mcpServers = make(map[string]json.RawMessage)
	}

	// Check for existing sourcebridge entry.
	if raw, exists := mcpServers["sourcebridge"]; exists {
		var existingEntry MCPEntry
		if err := json.Unmarshal(raw, &existingEntry); err == nil {
			if existingEntry.Command == entry.Command {
				// Same command — leave args/env untouched (idempotent).
				return false, warningMsg, nil
			}
			if !force {
				return false, "", fmt.Errorf(
					"mcpServers.sourcebridge already exists with a different command (%q). Pass --force to replace.",
					existingEntry.Command,
				)
			}
		}
	}

	// Write the SourceBridge entry.
	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return false, "", fmt.Errorf("marshaling MCP entry: %w", err)
	}
	mcpServers["sourcebridge"] = json.RawMessage(entryJSON)

	mcpServersJSON, err := json.Marshal(mcpServers)
	if err != nil {
		return false, "", fmt.Errorf("marshaling mcpServers: %w", err)
	}
	doc["mcpServers"] = json.RawMessage(mcpServersJSON)

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return false, "", fmt.Errorf("marshaling .mcp.json: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, "", fmt.Errorf("creating directories for %s: %w", path, err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		return false, "", fmt.Errorf("writing %s: %w", path, err)
	}
	return true, warningMsg, nil
}
