// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package quality_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// requiredSections are the three sections every audience profile must contain.
// Matching is case-insensitive and based on whether the string appears as a
// markdown heading anywhere in the file.
var requiredAudienceSections = []string{
	"voice rules",
	"section schema",
	"length envelope",
}

// audienceProfiles lists the expected audience profile files relative to
// the repository root. These paths match the plan's Q.1 spec.
var audienceProfiles = []string{
	"prompts/audience/for-engineers.md",
	"prompts/audience/for-product.md",
	"prompts/audience/for-operators.md",
}

// voiceProfiles lists the expected voice profile files per Q.3.
var voiceProfiles = []string{
	"prompts/voice/engineer-to-engineer.md",
	"prompts/voice/engineer-to-pm.md",
	"prompts/voice/engineer-to-operator.md",
}

// TestAudienceProfiles_ExistAndHaveRequiredSections loads all three
// audience profile files and asserts:
//  1. The file exists.
//  2. It is non-empty (more than 100 bytes).
//  3. It contains all three required sections as markdown headings.
func TestAudienceProfiles_ExistAndHaveRequiredSections(t *testing.T) {
	repoRoot := repoRootFromTestFile(t)

	for _, relPath := range audienceProfiles {
		relPath := relPath
		t.Run(relPath, func(t *testing.T) {
			absPath := filepath.Join(repoRoot, relPath)
			content, err := os.ReadFile(absPath)
			if err != nil {
				t.Fatalf("cannot read %s: %v", relPath, err)
			}
			if len(content) < 100 {
				t.Errorf("%s: file is suspiciously short (%d bytes)", relPath, len(content))
			}

			lower := strings.ToLower(string(content))
			for _, section := range requiredAudienceSections {
				// Accept the section title appearing anywhere in a heading line.
				found := false
				for _, line := range strings.Split(lower, "\n") {
					if strings.HasPrefix(strings.TrimSpace(line), "#") &&
						strings.Contains(line, section) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("%s: missing required section %q", relPath, section)
				}
			}
		})
	}
}

// TestVoiceProfiles_ExistAndHaveExamples loads all three voice profile
// files and asserts:
//  1. The file exists.
//  2. It has both positive and negative examples (the words "positive"
//     and "negative" must appear as headings).
//  3. The negative examples section is non-trivial (> 50 words).
func TestVoiceProfiles_ExistAndHaveExamples(t *testing.T) {
	repoRoot := repoRootFromTestFile(t)

	for _, relPath := range voiceProfiles {
		relPath := relPath
		t.Run(relPath, func(t *testing.T) {
			absPath := filepath.Join(repoRoot, relPath)
			content, err := os.ReadFile(absPath)
			if err != nil {
				t.Fatalf("cannot read %s: %v", relPath, err)
			}

			lower := strings.ToLower(string(content))

			for _, required := range []string{"positive", "negative"} {
				found := false
				for _, line := range strings.Split(lower, "\n") {
					if strings.HasPrefix(strings.TrimSpace(line), "#") &&
						strings.Contains(line, required) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("%s: missing %q examples section", relPath, required)
				}
			}

			// Negative examples section must be non-trivial.
			negIdx := strings.Index(lower, "negative")
			if negIdx >= 0 {
				after := lower[negIdx:]
				words := strings.Fields(after)
				if len(words) < 50 {
					t.Errorf("%s: negative examples section appears too short (%d words after heading)", relPath, len(words))
				}
			}
		})
	}
}

// repoRootFromTestFile walks up from the test file's directory until it
// finds a go.mod file, returning that directory as the repo root.
// This makes the test work regardless of where `go test` is invoked from.
func repoRootFromTestFile(t *testing.T) string {
	t.Helper()
	// Start from the package directory (internal/quality) and walk up.
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("cannot get abs path: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod in any parent directory")
		}
		dir = parent
	}
}
