// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

// Package credentials provides the credential-injection port for the
// living-wiki runtime. HTTP clients receive a [Snapshot] per call rather than
// accepting tokens at construction time, so credential rotation takes effect
// on the next job without requiring a process restart.
//
// # Per-job invariant
//
// The at-most-one-rotation-per-job invariant is enforced by calling [Take]
// once at the top of each orchestrator job and threading the resulting
// [Snapshot] through all subsequent port calls within that job. Mid-job
// rotations are ignored; the next job picks up the new values.
//
// # HTTP client contract
//
// HTTP client constructors ([markdown.NewHTTPConfluenceClient],
// [markdown.NewHTTPNotionClient], [orchestrator.NewGitHubClient],
// [orchestrator.NewGitLabClient]) accept no token or getter parameters.
// Each public method on those clients accepts a [Snapshot] parameter so
// credentials are injected at the call site, not at construction time.
package credentials

import (
	"context"
	"fmt"

	"github.com/sourcebridge/sourcebridge/internal/settings/livingwiki"
)

// Broker is the credential-injection port for living-wiki runtime components.
// HTTP clients accept Broker (or a typed sub-interface), not the resolver
// type directly. Future credential sources — Vault, AWS Secrets Manager,
// per-tenant resolvers — implement Broker without touching client constructors.
type Broker interface {
	// GitHub returns the current GitHub PAT or App installation token.
	GitHub(ctx context.Context) (string, error)
	// GitLab returns the current GitLab PRIVATE-TOKEN.
	GitLab(ctx context.Context) (string, error)
	// ConfluenceSite returns the Atlassian Cloud site subdomain
	// (e.g. "mycompany" for mycompany.atlassian.net).
	ConfluenceSite(ctx context.Context) (string, error)
	// Confluence returns the Atlassian account email and API token for Basic auth.
	Confluence(ctx context.Context) (email, token string, err error)
	// Notion returns the current Notion integration secret.
	Notion(ctx context.Context) (string, error)
}

// Snapshot is the per-job immutable credential record. It is captured once at
// the start of each orchestrator job by [Take] and threaded through all HTTP
// calls within that job. Mid-job rotation does not affect an in-flight job;
// the next job captures a fresh Snapshot.
//
// Snapshot is a value type. Passing it by value guarantees that callers cannot
// mutate the orchestrator's copy.
type Snapshot struct {
	GitHubToken     string
	GitLabToken     string
	ConfluenceSite  string
	ConfluenceEmail string
	ConfluenceToken string
	NotionToken     string
}

// Take calls b once for each credential and returns an immutable Snapshot.
// If any credential call fails, Take returns a zero Snapshot and the error —
// partial snapshots are never returned so callers cannot accidentally use a
// half-populated credential set.
//
// The orchestrator calls Take at the start of every Generate invocation.
func Take(ctx context.Context, b Broker) (Snapshot, error) {
	gh, err := b.GitHub(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("credentials: GitHub: %w", err)
	}

	gl, err := b.GitLab(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("credentials: GitLab: %w", err)
	}

	cfSite, err := b.ConfluenceSite(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("credentials: ConfluenceSite: %w", err)
	}

	cfEmail, cfToken, err := b.Confluence(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("credentials: Confluence: %w", err)
	}

	nt, err := b.Notion(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("credentials: Notion: %w", err)
	}

	return Snapshot{
		GitHubToken:     gh,
		GitLabToken:     gl,
		ConfluenceSite:  cfSite,
		ConfluenceEmail: cfEmail,
		ConfluenceToken: cfToken,
		NotionToken:     nt,
	}, nil
}

// ResolverBroker wraps the living-wiki settings [livingwiki.Resolver] to
// implement [Broker]. Token calls benefit from the resolver's 30-second TTL
// cache, so repeated calls within a scheduling tick do not hammer the DB.
//
// Construct via [NewResolverBroker].
type ResolverBroker struct {
	r *livingwiki.Resolver
}

// NewResolverBroker returns a [Broker] backed by the given Resolver.
func NewResolverBroker(r *livingwiki.Resolver) Broker {
	return &ResolverBroker{r: r}
}

// GitHub implements [Broker].
func (b *ResolverBroker) GitHub(ctx context.Context) (string, error) {
	res, err := b.r.Get()
	if err != nil {
		return "", fmt.Errorf("resolverBroker: GitHub: %w", err)
	}
	return res.GitHubToken, nil
}

// GitLab implements [Broker].
func (b *ResolverBroker) GitLab(ctx context.Context) (string, error) {
	res, err := b.r.Get()
	if err != nil {
		return "", fmt.Errorf("resolverBroker: GitLab: %w", err)
	}
	return res.GitLabToken, nil
}

// ConfluenceSite implements [Broker].
func (b *ResolverBroker) ConfluenceSite(ctx context.Context) (string, error) {
	res, err := b.r.Get()
	if err != nil {
		return "", fmt.Errorf("resolverBroker: ConfluenceSite: %w", err)
	}
	return res.ConfluenceSite, nil
}

// Confluence implements [Broker].
func (b *ResolverBroker) Confluence(ctx context.Context) (email, token string, err error) {
	res, err := b.r.Get()
	if err != nil {
		return "", "", fmt.Errorf("resolverBroker: Confluence: %w", err)
	}
	return res.ConfluenceEmail, res.ConfluenceToken, nil
}

// Notion implements [Broker].
func (b *ResolverBroker) Notion(ctx context.Context) (string, error) {
	res, err := b.r.Get()
	if err != nil {
		return "", fmt.Errorf("resolverBroker: Notion: %w", err)
	}
	return res.NotionToken, nil
}

// Compile-time interface check.
var _ Broker = (*ResolverBroker)(nil)
