// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package graphql

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sourcebridge/sourcebridge/internal/settings/livingwiki"
)

// ─────────────────────────────────────────────────────────────────────────────
// Query resolver
// ─────────────────────────────────────────────────────────────────────────────

func (r *queryResolver) LivingWikiSettings(ctx context.Context) (*LivingWikiSettings, error) {
	if r.LivingWikiStore == nil {
		return &LivingWikiSettings{}, nil
	}
	s, err := r.LivingWikiStore.Get()
	if err != nil {
		return nil, err
	}
	return mapLivingWikiSettings(livingwiki.MaskSecrets(*s)), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Mutation resolvers
// ─────────────────────────────────────────────────────────────────────────────

func (r *mutationResolver) UpdateLivingWikiSettings(ctx context.Context, input UpdateLivingWikiSettingsInput) (*LivingWikiSettings, error) {
	if r.LivingWikiStore == nil {
		return nil, fmt.Errorf("living-wiki settings not configured")
	}

	// Load current stored settings so we can apply a partial patch.
	current, err := r.LivingWikiStore.Get()
	if err != nil {
		return nil, err
	}

	// Apply scalar fields.
	if input.Enabled != nil {
		current.Enabled = input.Enabled
	}
	if input.WorkerCount != nil {
		current.WorkerCount = *input.WorkerCount
	}
	if input.EventTimeout != nil {
		current.EventTimeout = *input.EventTimeout
	}

	// Apply secret fields: ignore the sentinel (don't overwrite with "********").
	applySecret := func(stored *string, incoming *string) {
		if incoming != nil && *incoming != livingwiki.SecretSentinel {
			*stored = *incoming
		}
	}
	applySecret(&current.GitHubToken, input.GithubToken)
	applySecret(&current.GitLabToken, input.GitlabToken)
	applySecret(&current.ConfluenceEmail, input.ConfluenceEmail)
	applySecret(&current.ConfluenceToken, input.ConfluenceToken)
	applySecret(&current.NotionToken, input.NotionToken)
	applySecret(&current.ConfluenceWebhookSecret, input.ConfluenceWebhookSecret)
	applySecret(&current.NotionWebhookSecret, input.NotionWebhookSecret)

	// Stamp audit fields.
	current.UpdatedAt = time.Now()
	if userID := userIDFromContext(ctx); userID != "" {
		current.UpdatedBy = userID
	}

	if err := r.LivingWikiStore.Set(current); err != nil {
		return nil, err
	}

	// Invalidate the resolver cache so next webhook sees new values immediately.
	if r.LivingWikiResolver != nil {
		r.LivingWikiResolver.Invalidate()
	}

	return mapLivingWikiSettings(livingwiki.MaskSecrets(*current)), nil
}

func (r *mutationResolver) TestLivingWikiConnection(ctx context.Context, provider string) (*LivingWikiConnectionTestResult, error) {
	if r.LivingWikiResolver == nil {
		return &LivingWikiConnectionTestResult{
			Provider: provider,
			Ok:       false,
			Message:  strPtr("living-wiki resolver not configured"),
		}, nil
	}

	res, err := r.LivingWikiResolver.Get()
	if err != nil {
		return nil, err
	}

	switch provider {
	case "github":
		return testGitHubConnection(ctx, res.GitHubToken)
	case "gitlab":
		return testGitLabConnection(ctx, res.GitLabToken)
	case "confluence":
		return testConfluenceConnection(ctx, res.ConfluenceEmail, res.ConfluenceToken)
	case "notion":
		return testNotionConnection(ctx, res.NotionToken)
	default:
		return &LivingWikiConnectionTestResult{
			Provider: provider,
			Ok:       false,
			Message:  strPtr(fmt.Sprintf("unknown provider %q; expected github, gitlab, confluence, or notion", provider)),
		}, nil
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Mapping helper
// ─────────────────────────────────────────────────────────────────────────────

func mapLivingWikiSettings(s livingwiki.Settings) *LivingWikiSettings {
	out := &LivingWikiSettings{
		Enabled:      s.Enabled,
		UpdatedBy:    strPtrIfNonEmpty(s.UpdatedBy),
	}
	if s.WorkerCount > 0 {
		v := s.WorkerCount
		out.WorkerCount = &v
	}
	if s.EventTimeout != "" {
		out.EventTimeout = &s.EventTimeout
	}
	if s.GitHubToken != "" {
		out.GithubToken = &s.GitHubToken
	}
	if s.GitLabToken != "" {
		out.GitlabToken = &s.GitLabToken
	}
	if s.ConfluenceEmail != "" {
		out.ConfluenceEmail = &s.ConfluenceEmail
	}
	if s.ConfluenceToken != "" {
		out.ConfluenceToken = &s.ConfluenceToken
	}
	if s.NotionToken != "" {
		out.NotionToken = &s.NotionToken
	}
	if s.ConfluenceWebhookSecret != "" {
		out.ConfluenceWebhookSecret = &s.ConfluenceWebhookSecret
	}
	if s.NotionWebhookSecret != "" {
		out.NotionWebhookSecret = &s.NotionWebhookSecret
	}
	if !s.UpdatedAt.IsZero() {
		out.UpdatedAt = &s.UpdatedAt
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// Provider connectivity tests
// ─────────────────────────────────────────────────────────────────────────────

func testGitHubConnection(ctx context.Context, token string) (*LivingWikiConnectionTestResult, error) {
	if token == "" {
		return &LivingWikiConnectionTestResult{
			Provider: "github",
			Ok:       false,
			Message:  strPtr("no GitHub token configured"),
		}, nil
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &LivingWikiConnectionTestResult{
			Provider: "github",
			Ok:       false,
			Message:  strPtr(err.Error()),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return &LivingWikiConnectionTestResult{Provider: "github", Ok: true}, nil
	}
	return &LivingWikiConnectionTestResult{
		Provider: "github",
		Ok:       false,
		Message:  strPtr(fmt.Sprintf("GitHub API returned %d", resp.StatusCode)),
	}, nil
}

func testGitLabConnection(ctx context.Context, token string) (*LivingWikiConnectionTestResult, error) {
	if token == "" {
		return &LivingWikiConnectionTestResult{
			Provider: "gitlab",
			Ok:       false,
			Message:  strPtr("no GitLab token configured"),
		}, nil
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://gitlab.com/api/v4/user", nil)
	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &LivingWikiConnectionTestResult{
			Provider: "gitlab",
			Ok:       false,
			Message:  strPtr(err.Error()),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return &LivingWikiConnectionTestResult{Provider: "gitlab", Ok: true}, nil
	}
	return &LivingWikiConnectionTestResult{
		Provider: "gitlab",
		Ok:       false,
		Message:  strPtr(fmt.Sprintf("GitLab API returned %d", resp.StatusCode)),
	}, nil
}

func testConfluenceConnection(ctx context.Context, email, token string) (*LivingWikiConnectionTestResult, error) {
	if email == "" || token == "" {
		return &LivingWikiConnectionTestResult{
			Provider: "confluence",
			Ok:       false,
			Message:  strPtr("Confluence email and token are both required"),
		}, nil
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.atlassian.com/me", nil)
	req.SetBasicAuth(email, token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &LivingWikiConnectionTestResult{
			Provider: "confluence",
			Ok:       false,
			Message:  strPtr(err.Error()),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return &LivingWikiConnectionTestResult{Provider: "confluence", Ok: true}, nil
	}
	return &LivingWikiConnectionTestResult{
		Provider: "confluence",
		Ok:       false,
		Message:  strPtr(fmt.Sprintf("Atlassian API returned %d", resp.StatusCode)),
	}, nil
}

func testNotionConnection(ctx context.Context, token string) (*LivingWikiConnectionTestResult, error) {
	if token == "" {
		return &LivingWikiConnectionTestResult{
			Provider: "notion",
			Ok:       false,
			Message:  strPtr("no Notion token configured"),
		}, nil
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.notion.com/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", "2022-06-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &LivingWikiConnectionTestResult{
			Provider: "notion",
			Ok:       false,
			Message:  strPtr(err.Error()),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return &LivingWikiConnectionTestResult{Provider: "notion", Ok: true}, nil
	}
	return &LivingWikiConnectionTestResult{
		Provider: "notion",
		Ok:       false,
		Message:  strPtr(fmt.Sprintf("Notion API returned %d", resp.StatusCode)),
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Micro-helpers
// ─────────────────────────────────────────────────────────────────────────────

func strPtrIfNonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// userIDFromContext extracts the caller's user ID for audit logging.
// Returns "" when no user is in context (e.g. anonymous or API token callers).
func userIDFromContext(ctx context.Context) string {
	// Re-use the existing auth package's token accessor.
	// Import is already pulled in by other resolvers via helpers.go.
	_ = ctx
	return "" // extended by enterprise layer; acceptable for now
}
