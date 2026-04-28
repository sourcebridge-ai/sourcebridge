// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package skillcard

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// PatchGitignore idempotently adds line to the .gitignore file at path.
// If the file does not exist, it is created with just the entry.
// If the entry already exists (as an exact line), the file is not modified.
// The function returns true when it wrote a change, false when the file was
// already correct.
func PatchGitignore(path, line string) (bool, error) {
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("reading %s: %w", path, err)
	}

	content := string(existing)

	// Check whether the line is already present (exact match on any line).
	for _, l := range strings.Split(content, "\n") {
		if strings.TrimSpace(l) == strings.TrimSpace(line) {
			return false, nil
		}
	}

	// Append the entry, ensuring a trailing newline before and after.
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += line + "\n"

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return false, fmt.Errorf("writing %s: %w", path, err)
	}
	return true, nil
}
