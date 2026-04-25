// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeFilesToDir writes a map of relative file paths → content to root.
// Missing intermediate directories are created. Existing files are overwritten.
func writeFilesToDir(root string, files map[string][]byte) error {
	for relPath, content := range files {
		absPath := filepath.Join(root, relPath)
		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("orchestrator: creating directory %q: %w", dir, err)
		}
		if err := os.WriteFile(absPath, content, 0o644); err != nil {
			return fmt.Errorf("orchestrator: writing file %q: %w", absPath, err)
		}
	}
	return nil
}
