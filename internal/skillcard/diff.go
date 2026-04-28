// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package skillcard

import (
	"fmt"
	"io"
	"strings"
)

// DiffAction classifies a planned file operation for --dry-run output.
type DiffAction struct {
	// Tag is one of "CREATE", "MODIFY", "SKIP — user-modified",
	// "SKIP — orphan marker", "UNCHANGED".
	Tag string
	// Path is the file path relative to the repository root.
	Path string
	// Detail is an optional inline annotation (e.g. "+1 line: .claude/sourcebridge.json").
	Detail string
}

// PrintDiff writes the --dry-run summary to w.
// Format (per the plan):
//
//	[CREATE]   .claude/CLAUDE.md
//	[MODIFY]   .gitignore  (+1 line: .claude/sourcebridge.json)
//	[SKIP — user-modified]  .claude/CLAUDE.md (subsystem: billing section)
//
//	3 files would be written, 1 skipped.
func PrintDiff(w io.Writer, actions []DiffAction) {
	written := 0
	skipped := 0
	unchanged := 0

	for _, a := range actions {
		tag := "[" + a.Tag + "]"
		line := fmt.Sprintf("%-30s %s", tag, a.Path)
		if a.Detail != "" {
			line += "  " + a.Detail
		}
		fmt.Fprintln(w, line)

		switch {
		case strings.HasPrefix(a.Tag, "SKIP"):
			skipped++
		case a.Tag == "UNCHANGED":
			unchanged++
		default:
			written++
		}
	}

	fmt.Fprintln(w)
	parts := []string{}
	if written > 0 {
		parts = append(parts, fmt.Sprintf("%d %s would be written", written, pluralize("file", written)))
	}
	if unchanged > 0 {
		parts = append(parts, fmt.Sprintf("%d unchanged", unchanged))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipped))
	}
	if len(parts) == 0 {
		fmt.Fprintln(w, "No changes.")
		return
	}
	fmt.Fprintln(w, strings.Join(parts, ", ")+".")
}
