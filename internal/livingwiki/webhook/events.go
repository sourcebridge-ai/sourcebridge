// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

// Package webhook defines the normalized event types that enterprise webhook
// handlers parse from raw GitHub/GitLab/Confluence/Notion payloads before
// handing them to the [Dispatcher].
//
// # Division of responsibility
//
// Enterprise handlers (in sourcebridge-enterprise) own:
//   - HTTP signature verification (HMAC, Confluence, Notion secrets).
//   - Raw JSON/XML parsing of the provider-specific envelope.
//   - Mapping the parsed fields to the typed events in this package.
//
// The OSS [Dispatcher] owns:
//   - Delivery-ID deduplication.
//   - Per-repo serialization (one goroutine per repo at a time).
//   - Mapping each event type to the correct [orchestrator.Orchestrator] call.
//   - Structured logging.
//
// # Event type table
//
//	EventPush                → push to main/default branch
//	EventPRBranchCommit      → human commit pushed to an open wiki PR branch
//	EventPRMerged            → wiki PR merged into main
//	EventPRClosedWithoutMerge → wiki PR closed/rejected without merge
//	EventConfluenceBlockEdit → block changed in Confluence (from webhook or poll)
//	EventNotionBlockEdit     → block changed in Notion (from automation webhook)
//	EventManualRefresh       → operator-triggered regen (UI or API)
package webhook

import "time"

// EventType is the discriminator for the webhook event union.
type EventType string

const (
	// EventPush fires when commits land on the source repo's default (or
	// watched) branch. The dispatcher runs GenerateIncremental.
	EventPush EventType = "push"

	// EventPRBranchCommit fires when a human pushes to an open wiki PR branch.
	// Commits authored by the SourceBridge bot are excluded by the enterprise
	// handler before populating [PRBranchCommitEvent.Commits].
	EventPRBranchCommit EventType = "pr_branch_commit"

	// EventPRMerged fires when the wiki PR is merged into the base branch.
	// The dispatcher calls Promote + AdvancePublished.
	EventPRMerged EventType = "pr_merged"

	// EventPRClosedWithoutMerge fires when the wiki PR is closed without merge.
	// The dispatcher calls Discard + Reset watermarks.
	EventPRClosedWithoutMerge EventType = "pr_rejected"

	// EventConfluenceBlockEdit fires when a Confluence page block is edited,
	// either via the Confluence webhook receiver or the poll endpoint.
	EventConfluenceBlockEdit EventType = "confluence_block_edit"

	// EventNotionBlockEdit fires when a Notion block is edited, either via the
	// Notion automation webhook or the poll endpoint.
	EventNotionBlockEdit EventType = "notion_block_edit"

	// EventManualRefresh fires when an operator explicitly requests regen via the
	// API or admin UI.
	EventManualRefresh EventType = "manual_refresh"
)

// WebhookEvent is the interface that all event types satisfy. Enterprise
// handlers parse the raw provider envelope and return a concrete type;
// the [Dispatcher] dispatches on EventType without needing the concrete type.
type WebhookEvent interface {
	// EventType returns the discriminator for this event.
	EventType() EventType

	// RepoID returns the opaque repository identifier the event belongs to.
	// Must be non-empty.
	RepoID() string

	// DeliveryID returns the provider-assigned delivery identifier (e.g.
	// X-GitHub-Delivery, X-GitLab-Event-UUID, a Confluence webhook UUID).
	// Empty string disables deduplication for this event.
	DeliveryID() string
}

// ─────────────────────────────────────────────────────────────────────────────
// Concrete event types
// ─────────────────────────────────────────────────────────────────────────────

// PushEvent is fired when commits arrive on the source repo's default branch.
type PushEvent struct {
	// Repo is the opaque repository identifier.
	Repo string

	// Delivery is the provider delivery ID for deduplication.
	Delivery string

	// Branch is the branch that received the push (e.g. "main").
	Branch string

	// BeforeSHA is the previous HEAD before the push.
	// Empty when this is the first push to the branch.
	BeforeSHA string

	// AfterSHA is the new HEAD after the push.
	AfterSHA string

	// PusherName is the display name of the actor who pushed.
	PusherName string

	// PusherEmail is the email of the actor who pushed.
	PusherEmail string

	// ReceivedAt is when the webhook was received.
	ReceivedAt time.Time
}

func (e PushEvent) EventType() EventType { return EventPush }
func (e PushEvent) RepoID() string       { return e.Repo }
func (e PushEvent) DeliveryID() string   { return e.Delivery }

// Commit is a single commit on a branch, carrying enough metadata for the
// reviewer-commit detection logic (mirrors orchestrator.Commit to avoid the
// webhook package importing the orchestrator package).
type Commit struct {
	// SHA is the full commit hash.
	SHA string

	// CommitterName is the name field from the committer object.
	CommitterName string

	// CommitterEmail is the email field from the committer object.
	CommitterEmail string

	// Files maps changed file paths to their post-commit contents.
	// A nil value means the file was deleted.
	Files map[string][]byte
}

// PRBranchCommitEvent fires when a human commits to an open wiki PR branch.
// The enterprise handler strips bot commits before populating Commits.
type PRBranchCommitEvent struct {
	// Repo is the opaque repository identifier.
	Repo string

	// Delivery is the provider delivery ID for deduplication.
	Delivery string

	// PRID is the platform PR identifier (e.g. GitHub PR number as a string).
	PRID string

	// Branch is the PR branch name.
	Branch string

	// Commits is the list of human commits since the last bot commit.
	// Bot commits (identified by committer name == "sourcebridge[bot]") are
	// excluded by the enterprise handler.
	Commits []Commit

	// ReceivedAt is when the webhook was received.
	ReceivedAt time.Time
}

func (e PRBranchCommitEvent) EventType() EventType { return EventPRBranchCommit }
func (e PRBranchCommitEvent) RepoID() string       { return e.Repo }
func (e PRBranchCommitEvent) DeliveryID() string   { return e.Delivery }

// PRMergedEvent fires when the wiki PR is merged into the base branch.
type PRMergedEvent struct {
	// Repo is the opaque repository identifier.
	Repo string

	// Delivery is the provider delivery ID for deduplication.
	Delivery string

	// PRID is the platform PR identifier.
	PRID string

	// MergedSHA is the commit SHA of the merge commit on the base branch.
	// Used to advance the WikiPublishedSHA watermark.
	MergedSHA string
}

func (e PRMergedEvent) EventType() EventType { return EventPRMerged }
func (e PRMergedEvent) RepoID() string       { return e.Repo }
func (e PRMergedEvent) DeliveryID() string   { return e.Delivery }

// PRRejectedEvent fires when the wiki PR is closed without merging.
type PRRejectedEvent struct {
	// Repo is the opaque repository identifier.
	Repo string

	// Delivery is the provider delivery ID for deduplication.
	Delivery string

	// PRID is the platform PR identifier.
	PRID string
}

func (e PRRejectedEvent) EventType() EventType { return EventPRClosedWithoutMerge }
func (e PRRejectedEvent) RepoID() string       { return e.Repo }
func (e PRRejectedEvent) DeliveryID() string   { return e.Delivery }

// BlockContent mirrors ast.BlockContent without importing the ast package,
// keeping the webhook package import-free of livingwiki internals.
// The dispatcher adapter (in the enterprise layer or in dispatcher.go) converts
// this to ast.BlockContent before calling the orchestrator.
type BlockContent struct {
	// ParagraphMarkdown holds the markdown text when this is a paragraph block.
	// Mutually exclusive with CodeBody / CodeLanguage.
	ParagraphMarkdown string

	// CodeLanguage is the language identifier for a code block.
	CodeLanguage string

	// CodeBody is the source text of a code block.
	CodeBody string

	// Raw is a catch-all for block kinds that don't fit the above (e.g.
	// freeform, callout). The dispatcher passes this through to the
	// orchestrator's freeform block content.
	Raw string
}

// SinkBlockEditEvent fires when a Confluence or Notion block is changed.
// The enterprise handler (or the webhook receiver in this package) constructs
// this from the parsed provider payload.
type SinkBlockEditEvent struct {
	// Repo is the opaque repository identifier.
	Repo string

	// Delivery is the provider delivery ID for deduplication.
	Delivery string

	// SinkName is the integration ID of the sink (e.g. "confluence-acme-space").
	SinkName string

	// PageID is the SourceBridge page ID (not the provider's native ID).
	PageID string

	// BlockID is the SourceBridge block ID that changed.
	BlockID string

	// NewContent is the updated block content as read from the sink.
	NewContent BlockContent

	// EditedBy is the sink's user identifier for the person who made the edit.
	EditedBy string

	// EditedAt is when the edit was observed in the sink.
	EditedAt time.Time
}

func (e SinkBlockEditEvent) EventType() EventType {
	// Both Confluence and Notion block-edit events carry the same event type
	// discriminator; callers that need to distinguish them should check SinkName.
	return EventConfluenceBlockEdit
}
func (e SinkBlockEditEvent) RepoID() string     { return e.Repo }
func (e SinkBlockEditEvent) DeliveryID() string { return e.Delivery }

// NotionBlockEditEvent is the Notion-specific variant of [SinkBlockEditEvent].
// It exists as a separate concrete type so that enterprise handlers can return
// the more specific type and the dispatcher can emit the correct EventType.
type NotionBlockEditEvent struct {
	SinkBlockEditEvent
}

func (e NotionBlockEditEvent) EventType() EventType { return EventNotionBlockEdit }

// ManualRefreshEvent fires when an operator explicitly requests wiki regen via
// the SourceBridge API or admin UI.
type ManualRefreshEvent struct {
	// Repo is the opaque repository identifier.
	Repo string

	// Delivery is the provider delivery ID for deduplication.
	// Typically a random UUID generated by the caller.
	Delivery string

	// PageID is the SourceBridge page ID to refresh.
	// Empty means refresh the entire repo.
	PageID string

	// RequestedBy is the user ID of the operator who triggered the refresh.
	RequestedBy string
}

func (e ManualRefreshEvent) EventType() EventType { return EventManualRefresh }
func (e ManualRefreshEvent) RepoID() string       { return e.Repo }
func (e ManualRefreshEvent) DeliveryID() string   { return e.Delivery }
