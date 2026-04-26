// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package livingwiki

import "time"

// Settings holds the living-wiki configuration as stored in the DB (via the
// admin UI). Zero/empty values mean "not configured by UI; use env-var or
// built-in default".
//
// Secret fields (tokens, webhook secrets) are stored encrypted at rest. The
// resolver returns the sentinel "********" for any field that has been set,
// so clients can detect "a value exists" without reading the value back.
type Settings struct {
	// --- Orchestration ---

	// Enabled is the master on/off switch. nil means "not set by UI".
	Enabled *bool `json:"enabled,omitempty"`

	// WorkerCount controls Dispatcher goroutine count. 0 = not set by UI.
	WorkerCount int `json:"worker_count,omitempty"`

	// EventTimeout is the per-event handler deadline. Empty = not set by UI.
	// Stored as a Go duration string (e.g. "5m").
	EventTimeout string `json:"event_timeout,omitempty"`

	// --- Source integrations (encrypted at rest) ---

	// GitHubToken is a Personal Access Token or GitHub App installation token.
	GitHubToken string `json:"github_token,omitempty"`

	// GitLabToken is a GitLab PRIVATE-TOKEN.
	GitLabToken string `json:"gitlab_token,omitempty"`

	// ConfluenceEmail is the Atlassian account email for Basic auth.
	ConfluenceEmail string `json:"confluence_email,omitempty"`

	// ConfluenceToken is the Atlassian API token.
	ConfluenceToken string `json:"confluence_token,omitempty"`

	// NotionToken is the Notion integration secret.
	NotionToken string `json:"notion_token,omitempty"`

	// --- Webhook validation secrets (encrypted at rest) ---

	// ConfluenceWebhookSecret is the HMAC-SHA256 secret for Confluence webhooks.
	ConfluenceWebhookSecret string `json:"confluence_webhook_secret,omitempty"`

	// NotionWebhookSecret is reserved for future Notion webhook validation.
	NotionWebhookSecret string `json:"notion_webhook_secret,omitempty"`

	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by,omitempty"`
}

// SecretSentinel is returned in place of a plaintext secret value when one
// has been stored. The UI displays this to communicate "a value exists".
const SecretSentinel = "********"

// secretFields lists field names that carry credentials and must be encrypted
// at rest. The Store implementation uses this list; callers use MaskSecrets.
var secretFields = []string{
	"github_token",
	"gitlab_token",
	"confluence_email",
	"confluence_token",
	"notion_token",
	"confluence_webhook_secret",
	"notion_webhook_secret",
}

// MaskSecrets returns a copy of s where every secret field that has a
// non-empty value is replaced by [SecretSentinel]. This is the struct that
// GraphQL resolvers return to the UI.
func MaskSecrets(s Settings) Settings {
	if s.GitHubToken != "" {
		s.GitHubToken = SecretSentinel
	}
	if s.GitLabToken != "" {
		s.GitLabToken = SecretSentinel
	}
	if s.ConfluenceEmail != "" {
		s.ConfluenceEmail = SecretSentinel
	}
	if s.ConfluenceToken != "" {
		s.ConfluenceToken = SecretSentinel
	}
	if s.NotionToken != "" {
		s.NotionToken = SecretSentinel
	}
	if s.ConfluenceWebhookSecret != "" {
		s.ConfluenceWebhookSecret = SecretSentinel
	}
	if s.NotionWebhookSecret != "" {
		s.NotionWebhookSecret = SecretSentinel
	}
	return s
}

// Store is the persistence interface for living-wiki settings.
// Implementations: [MemStore] (tests) and the SurrealDB store in internal/db.
type Store interface {
	// Get returns the current settings, or a zero-value Settings if none have
	// been saved yet. Secrets are returned decrypted.
	Get() (*Settings, error)

	// Set persists s. Secret fields are encrypted before writing.
	Set(s *Settings) error
}
