// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

// gitcli.go implements the DiffProvider and ExtendedRepoWriter ports using the
// git CLI (os/exec).  It follows the same pattern as internal/git/local.go:
// spawn git, parse output, return errors with context.
//
// # DiffProvider
//
// [GitCLIDiffProvider] runs:
//
//	git diff --name-only --diff-filter=ACMR <baseSHA>..<headSHA>
//
// When baseSHA is empty (first run) it returns all files tracked in headSHA:
//
//	git ls-tree -r --name-only <headSHA>
//
// A fatal: "bad object" or "unknown revision" exit from git indicates the
// baseSHA is no longer reachable — force-push scenario — and is translated to
// [ErrSHANotFound].
//
// # ExtendedRepoWriter
//
// [GitCLIRepoWriter] writes files to a local clone:
//   - [WriteFiles]: write to the configured branch, stage, commit, push.
//   - [AppendCommitToBranch]: same but with a caller-supplied message.
//   - [ListCommitsOnBranch]: git log --pretty=format, parse into [Commit] slices.
//
// Commits are stamped with [SourceBridgeCommitterName] / [SourceBridgeCommitterEmail].
package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sourcebridge/sourcebridge/internal/livingwiki/manifest"
)

// GitCLIConfig holds configuration for git CLI adapters.
type GitCLIConfig struct {
	// ClonePath is the local filesystem path to the git working tree.
	ClonePath string

	// RemoteName is the git remote to push to (default "origin").
	RemoteName string

	// Timeout is the per-operation timeout passed to each git invocation.
	// When zero, defaults to 60 seconds.
	Timeout time.Duration
}

func (c GitCLIConfig) remoteName() string {
	if c.RemoteName == "" {
		return "origin"
	}
	return c.RemoteName
}

func (c GitCLIConfig) timeout() time.Duration {
	if c.Timeout <= 0 {
		return 60 * time.Second
	}
	return c.Timeout
}

// ─────────────────────────────────────────────────────────────────────────────
// GitCLIDiffProvider
// ─────────────────────────────────────────────────────────────────────────────

// GitCLIDiffProvider implements [DiffProvider] via the git CLI.
type GitCLIDiffProvider struct {
	cfg GitCLIConfig
}

// NewGitCLIDiffProvider creates a [GitCLIDiffProvider] for the clone at cfg.ClonePath.
func NewGitCLIDiffProvider(cfg GitCLIConfig) *GitCLIDiffProvider {
	return &GitCLIDiffProvider{cfg: cfg}
}

// Compile-time interface check.
var _ DiffProvider = (*GitCLIDiffProvider)(nil)

// Diff returns the changed file pairs between baseSHA and headSHA.
// When baseSHA is empty, all files at headSHA are treated as changed.
// Returns [ErrSHANotFound] when baseSHA cannot be resolved (force-push).
func (p *GitCLIDiffProvider) Diff(ctx context.Context, _ string, baseSHA, headSHA string) (DiffResult, error) {
	ctx, cancel := context.WithTimeout(ctx, p.cfg.timeout())
	defer cancel()

	if baseSHA == "" {
		return p.allFiles(ctx, headSHA)
	}
	return p.diffSHAs(ctx, baseSHA, headSHA)
}

// diffSHAs runs `git diff --name-only --diff-filter=ACMR baseSHA..headSHA`.
func (p *GitCLIDiffProvider) diffSHAs(ctx context.Context, baseSHA, headSHA string) (DiffResult, error) {
	out, err := p.runGit(ctx, "diff", "--name-only", "--diff-filter=ACMR",
		baseSHA+".."+headSHA)
	if err != nil {
		if isSHANotFound(err, string(out)) {
			return DiffResult{}, ErrSHANotFound
		}
		return DiffResult{}, fmt.Errorf("gitcli diff %s..%s: %w", baseSHA, headSHA, err)
	}

	// Count lines for the skip-guard.
	statOut, statErr := p.runGit(ctx, "diff", "--stat", baseSHA+".."+headSHA)
	totalLines := 0
	if statErr == nil {
		totalLines = parseDiffStatLines(string(statOut))
	}

	changed := parseNameOnlyOutput(string(out))
	return DiffResult{Changed: changed, TotalLines: totalLines}, nil
}

// allFiles returns all tracked files at headSHA as if every file changed.
func (p *GitCLIDiffProvider) allFiles(ctx context.Context, headSHA string) (DiffResult, error) {
	out, err := p.runGit(ctx, "ls-tree", "-r", "--name-only", headSHA)
	if err != nil {
		if isSHANotFound(err, string(out)) {
			return DiffResult{}, ErrSHANotFound
		}
		return DiffResult{}, fmt.Errorf("gitcli ls-tree %s: %w", headSHA, err)
	}
	changed := parseNameOnlyOutput(string(out))
	// TotalLines is unknown for a full-tree listing; return 0 so the caller
	// does not trigger the skip guard.
	return DiffResult{Changed: changed, TotalLines: 0}, nil
}

func (p *GitCLIDiffProvider) runGit(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = p.cfg.ClonePath
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return stderr.Bytes(), fmt.Errorf("%w; stderr: %s", err, stderr.String())
	}
	return out, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// GitCLIRepoWriter
// ─────────────────────────────────────────────────────────────────────────────

// GitCLIRepoWriter implements [ExtendedRepoWriter] via the git CLI.
type GitCLIRepoWriter struct {
	cfg GitCLIConfig
}

// NewGitCLIRepoWriter creates a [GitCLIRepoWriter] for the clone at cfg.ClonePath.
func NewGitCLIRepoWriter(cfg GitCLIConfig) *GitCLIRepoWriter {
	return &GitCLIRepoWriter{cfg: cfg}
}

// Compile-time interface checks.
var _ RepoWriter = (*GitCLIRepoWriter)(nil)
var _ ExtendedRepoWriter = (*GitCLIRepoWriter)(nil)

// WriteFiles checks out the configured branch (creating it if necessary),
// writes files, stages them, commits as the SourceBridge bot, and pushes.
// It uses the PRBranch from the orchestrator config; callers should inject
// the desired branch via [AppendCommitToBranch].
func (w *GitCLIRepoWriter) WriteFiles(ctx context.Context, files map[string][]byte) error {
	return w.AppendCommitToBranch(ctx, "sourcebridge/wiki-update", files,
		"wiki: update (sourcebridge)")
}

// AppendCommitToBranch checks out branch (creating it if it does not exist),
// writes files, stages all changes, commits with the given message as the
// SourceBridge bot committer, and pushes to the remote.
func (w *GitCLIRepoWriter) AppendCommitToBranch(ctx context.Context, branch string, files map[string][]byte, message string) error {
	ctx, cancel := context.WithTimeout(ctx, w.cfg.timeout())
	defer cancel()

	// Checkout or create the branch.
	if err := w.checkoutOrCreate(ctx, branch); err != nil {
		return fmt.Errorf("gitcli: checkout branch %q: %w", branch, err)
	}

	// Write files to disk.
	for path, content := range files {
		absPath := filepath.Join(w.cfg.ClonePath, path)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			return fmt.Errorf("gitcli: mkdir %q: %w", filepath.Dir(absPath), err)
		}
		if err := os.WriteFile(absPath, content, 0o644); err != nil {
			return fmt.Errorf("gitcli: write %q: %w", path, err)
		}
	}

	// Stage all changes (only the files we wrote, not any pre-existing changes).
	for path := range files {
		if _, err := w.runGit(ctx, "add", "--", path); err != nil {
			return fmt.Errorf("gitcli: git add %q: %w", path, err)
		}
	}

	// Check if there is anything to commit (staged diff).
	_, diffErr := w.runGit(ctx, "diff", "--cached", "--quiet")
	if diffErr == nil {
		// Exit code 0 means no changes staged — nothing to commit.
		return nil
	}

	// Commit with the SourceBridge bot identity.
	_, err := w.runGitWithEnv(ctx,
		[]string{
			"GIT_COMMITTER_NAME=" + SourceBridgeCommitterName,
			"GIT_COMMITTER_EMAIL=" + SourceBridgeCommitterEmail,
		},
		"commit",
		"--author="+SourceBridgeCommitterName+" <"+SourceBridgeCommitterEmail+">",
		"-m", message,
	)
	if err != nil {
		return fmt.Errorf("gitcli: git commit: %w", err)
	}

	// Push to remote.
	if _, pushErr := w.runGit(ctx, "push", w.cfg.remoteName(), branch); pushErr != nil {
		return fmt.Errorf("gitcli: git push %s %s: %w", w.cfg.remoteName(), branch, pushErr)
	}

	return nil
}

// ListCommitsOnBranch returns commits on branch that were authored at or after
// since, in oldest-first order.
//
// Format string: `%H|%an|%ae|%at|%s` (hash, author name, author email,
// unix timestamp, subject).
func (w *GitCLIRepoWriter) ListCommitsOnBranch(ctx context.Context, branch string, since time.Time) ([]Commit, error) {
	ctx, cancel := context.WithTimeout(ctx, w.cfg.timeout())
	defer cancel()

	args := []string{
		"log",
		"--pretty=format:%H|%cn|%ce|%ct|%s",
		branch,
	}
	if !since.IsZero() {
		args = append(args, "--after="+strconv.FormatInt(since.Unix(), 10))
	}

	out, err := w.runGit(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("gitcli: git log %s: %w", branch, err)
	}

	return parseGitLog(string(out)), nil
}

// checkoutOrCreate switches to branch, creating it from the current HEAD when
// it does not exist in the local clone.
func (w *GitCLIRepoWriter) checkoutOrCreate(ctx context.Context, branch string) error {
	// Try a plain checkout first.
	_, err := w.runGit(ctx, "checkout", branch)
	if err == nil {
		return nil
	}
	// Branch does not exist locally — create it.
	_, err = w.runGit(ctx, "checkout", "-b", branch)
	return err
}

func (w *GitCLIRepoWriter) runGit(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = w.cfg.ClonePath
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("%w; stderr: %s", err, stderr.String())
	}
	return out, nil
}

func (w *GitCLIRepoWriter) runGitWithEnv(ctx context.Context, env []string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = w.cfg.ClonePath
	cmd.Env = append(os.Environ(), env...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("%w; stderr: %s", err, stderr.String())
	}
	return out, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Parsing helpers
// ─────────────────────────────────────────────────────────────────────────────

// parseNameOnlyOutput converts newline-delimited file paths to ChangedPairs.
func parseNameOnlyOutput(output string) []manifest.ChangedPair {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var pairs []manifest.ChangedPair
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pairs = append(pairs, manifest.ChangedPair{Path: line})
	}
	return pairs
}

// parseDiffStatLines extracts the total insertions+deletions from `git diff --stat`
// output.  The last line is typically: "N files changed, M insertions(+), K deletions(-)"
func parseDiffStatLines(output string) int {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return 0
	}
	summary := lines[len(lines)-1]
	total := 0
	for _, field := range strings.Fields(summary) {
		n, err := strconv.Atoi(field)
		if err != nil {
			continue
		}
		total += n
	}
	return total
}

// parseGitLog parses lines in the format `%H|%cn|%ce|%ct|%s` into Commit values.
func parseGitLog(output string) []Commit {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	commits := make([]Commit, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Split on '|', but the subject (%s) may contain '|'.
		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 4 {
			continue
		}
		c := Commit{
			SHA:            parts[0],
			CommitterName:  parts[1],
			CommitterEmail: parts[2],
		}
		// parts[3] is the unix timestamp; parts[4] (if present) is the subject.
		_ = parts[3] // timestamp not exposed on Commit struct currently
		commits = append(commits, c)
	}
	// git log emits newest-first; reverse to oldest-first.
	for i, j := 0, len(commits)-1; i < j; i, j = i+1, j-1 {
		commits[i], commits[j] = commits[j], commits[i]
	}
	return commits
}

// isSHANotFound reports whether a git command failure indicates a missing SHA.
// git exits 128 with one of several messages in stderr depending on the command:
//   - "fatal: bad object"
//   - "fatal: unknown revision"
//   - "fatal: not a valid object name"
//   - "fatal: Invalid revision range"
func isSHANotFound(err error, output string) bool {
	if err == nil {
		return false
	}
	low := strings.ToLower(output)
	return strings.Contains(low, "bad object") ||
		strings.Contains(low, "unknown revision") ||
		strings.Contains(low, "not a valid object name") ||
		strings.Contains(low, "invalid revision range")
}
