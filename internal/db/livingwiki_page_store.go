// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package db

import (
	"context"
	"encoding/json"
	"fmt"

	surrealdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"

	"github.com/sourcebridge/sourcebridge/internal/livingwiki/ast"
	"github.com/sourcebridge/sourcebridge/internal/livingwiki/orchestrator"
)

// LivingWikiPageStore persists canonical and proposed wiki page ASTs in
// SurrealDB. It implements [orchestrator.PageStore].
//
// AST serialisation: the ast.Page value is marshalled as JSON and stored in the
// ast_json column. This keeps the SurrealDB schema simple while preserving full
// fidelity — the AST types are stable and the JSON encoding is used elsewhere
// for wire transport.
type LivingWikiPageStore struct {
	client *SurrealDB
}

// NewLivingWikiPageStore creates a store backed by the given SurrealDB client.
func NewLivingWikiPageStore(client *SurrealDB) *LivingWikiPageStore {
	return &LivingWikiPageStore{client: client}
}

// Compile-time interface check.
var _ orchestrator.PageStore = (*LivingWikiPageStore)(nil)

// ─────────────────────────────────────────────────────────────────────────────
// SurrealDB DTO
// ─────────────────────────────────────────────────────────────────────────────

type surrealLWPage struct {
	ID        *models.RecordID `json:"id,omitempty"`
	RepoID    string           `json:"repo_id"`
	PageID    string           `json:"page_id"`
	Kind      string           `json:"kind"` // "canonical" | "proposed"
	PRID      string           `json:"pr_id"`
	ASTJson   string           `json:"ast_json"`
	UpdatedAt surrealTime      `json:"updated_at"`
}

// ─────────────────────────────────────────────────────────────────────────────
// PageStore implementation
// ─────────────────────────────────────────────────────────────────────────────

func (s *LivingWikiPageStore) GetCanonical(ctx context.Context, repoID, pageID string) (ast.Page, bool, error) {
	db := s.client.DB()
	if db == nil {
		return ast.Page{}, false, fmt.Errorf("livingwiki page store: database not connected")
	}
	sql := `SELECT * FROM lw_pages WHERE repo_id = $repo_id AND page_id = $page_id AND kind = 'canonical' LIMIT 1`
	rows, err := queryOne[[]surrealLWPage](ctx, db, sql, map[string]any{
		"repo_id": repoID,
		"page_id": pageID,
	})
	if err != nil || len(rows) == 0 {
		return ast.Page{}, false, nil
	}
	page, err := unmarshalPage(rows[0].ASTJson)
	if err != nil {
		return ast.Page{}, false, fmt.Errorf("livingwiki page store: decode canonical %s/%s: %w", repoID, pageID, err)
	}
	return page, true, nil
}

func (s *LivingWikiPageStore) SetCanonical(ctx context.Context, repoID string, page ast.Page) error {
	db := s.client.DB()
	if db == nil {
		return fmt.Errorf("livingwiki page store: database not connected")
	}
	astJSON, err := marshalPage(page)
	if err != nil {
		return fmt.Errorf("livingwiki page store: encode canonical %s/%s: %w", repoID, page.ID, err)
	}
	sql := `
		UPSERT lw_pages
		SET repo_id    = $repo_id,
		    page_id    = $page_id,
		    kind       = 'canonical',
		    pr_id      = '',
		    ast_json   = $ast_json,
		    updated_at = time::now()
		WHERE repo_id = $repo_id AND page_id = $page_id AND kind = 'canonical'
	`
	_, err = surrealdb.Query[interface{}](ctx, db, sql, map[string]any{
		"repo_id":  repoID,
		"page_id":  page.ID,
		"ast_json": astJSON,
	})
	return err
}

func (s *LivingWikiPageStore) DeleteCanonical(ctx context.Context, repoID, pageID string) error {
	db := s.client.DB()
	if db == nil {
		return fmt.Errorf("livingwiki page store: database not connected")
	}
	sql := `DELETE FROM lw_pages WHERE repo_id = $repo_id AND page_id = $page_id AND kind = 'canonical'`
	_, err := surrealdb.Query[interface{}](ctx, db, sql, map[string]any{
		"repo_id": repoID,
		"page_id": pageID,
	})
	return err
}

func (s *LivingWikiPageStore) GetProposed(ctx context.Context, repoID, prID, pageID string) (ast.Page, bool, error) {
	db := s.client.DB()
	if db == nil {
		return ast.Page{}, false, fmt.Errorf("livingwiki page store: database not connected")
	}
	sql := `SELECT * FROM lw_pages WHERE repo_id = $repo_id AND pr_id = $pr_id AND page_id = $page_id AND kind = 'proposed' LIMIT 1`
	rows, err := queryOne[[]surrealLWPage](ctx, db, sql, map[string]any{
		"repo_id": repoID,
		"pr_id":   prID,
		"page_id": pageID,
	})
	if err != nil || len(rows) == 0 {
		return ast.Page{}, false, nil
	}
	page, err := unmarshalPage(rows[0].ASTJson)
	if err != nil {
		return ast.Page{}, false, fmt.Errorf("livingwiki page store: decode proposed %s/%s/%s: %w", repoID, prID, pageID, err)
	}
	return page, true, nil
}

func (s *LivingWikiPageStore) SetProposed(ctx context.Context, repoID, prID string, page ast.Page) error {
	db := s.client.DB()
	if db == nil {
		return fmt.Errorf("livingwiki page store: database not connected")
	}
	astJSON, err := marshalPage(page)
	if err != nil {
		return fmt.Errorf("livingwiki page store: encode proposed %s/%s/%s: %w", repoID, prID, page.ID, err)
	}
	sql := `
		UPSERT lw_pages
		SET repo_id    = $repo_id,
		    page_id    = $page_id,
		    kind       = 'proposed',
		    pr_id      = $pr_id,
		    ast_json   = $ast_json,
		    updated_at = time::now()
		WHERE repo_id = $repo_id AND pr_id = $pr_id AND page_id = $page_id AND kind = 'proposed'
	`
	_, err = surrealdb.Query[interface{}](ctx, db, sql, map[string]any{
		"repo_id":  repoID,
		"page_id":  page.ID,
		"pr_id":    prID,
		"ast_json": astJSON,
	})
	return err
}

func (s *LivingWikiPageStore) ListProposed(ctx context.Context, repoID, prID string) ([]ast.Page, error) {
	db := s.client.DB()
	if db == nil {
		return nil, fmt.Errorf("livingwiki page store: database not connected")
	}
	sql := `SELECT * FROM lw_pages WHERE repo_id = $repo_id AND pr_id = $pr_id AND kind = 'proposed'`
	rows, err := queryOne[[]surrealLWPage](ctx, db, sql, map[string]any{
		"repo_id": repoID,
		"pr_id":   prID,
	})
	if err != nil {
		// queryOne returns an error on empty result; treat as empty list.
		return []ast.Page{}, nil
	}
	pages := make([]ast.Page, 0, len(rows))
	for _, row := range rows {
		page, err := unmarshalPage(row.ASTJson)
		if err != nil {
			return nil, fmt.Errorf("livingwiki page store: decode proposed page %s: %w", row.PageID, err)
		}
		pages = append(pages, page)
	}
	return pages, nil
}

func (s *LivingWikiPageStore) DeleteProposed(ctx context.Context, repoID, prID string) error {
	db := s.client.DB()
	if db == nil {
		return fmt.Errorf("livingwiki page store: database not connected")
	}
	sql := `DELETE FROM lw_pages WHERE repo_id = $repo_id AND pr_id = $pr_id AND kind = 'proposed'`
	_, err := surrealdb.Query[interface{}](ctx, db, sql, map[string]any{
		"repo_id": repoID,
		"pr_id":   prID,
	})
	return err
}

func (s *LivingWikiPageStore) PromoteProposed(ctx context.Context, repoID, prID string) error {
	// List all proposed pages for this PR.
	proposed, err := s.ListProposed(ctx, repoID, prID)
	if err != nil {
		return fmt.Errorf("livingwiki page store: PromoteProposed list: %w", err)
	}
	if len(proposed) == 0 {
		return nil
	}

	// Fetch current canonical pages for comparison (needed by ast.Promote).
	// We do this one-by-one because the set is small (< 50 pages per repo).
	for _, pp := range proposed {
		canonical, _, err := s.GetCanonical(ctx, repoID, pp.ID)
		if err != nil {
			return fmt.Errorf("livingwiki page store: PromoteProposed get canonical %s: %w", pp.ID, err)
		}
		promoted := ast.Promote(canonical, pp)
		if err := s.SetCanonical(ctx, repoID, promoted); err != nil {
			return fmt.Errorf("livingwiki page store: PromoteProposed set canonical %s: %w", pp.ID, err)
		}
	}

	// Remove all proposed pages for this PR.
	return s.DeleteProposed(ctx, repoID, prID)
}

// ─────────────────────────────────────────────────────────────────────────────
// Encoding helpers
// ─────────────────────────────────────────────────────────────────────────────

func marshalPage(page ast.Page) (string, error) {
	b, err := json.Marshal(page)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalPage(s string) (ast.Page, error) {
	var page ast.Page
	if s == "" {
		return page, nil
	}
	if err := json.Unmarshal([]byte(s), &page); err != nil {
		return ast.Page{}, err
	}
	return page, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// LivingWikiWatermarkStore
// ─────────────────────────────────────────────────────────────────────────────

// LivingWikiWatermarkStore persists per-repo SHA watermarks in SurrealDB.
// It implements [orchestrator.WatermarkStore].
type LivingWikiWatermarkStore struct {
	client *SurrealDB
}

// NewLivingWikiWatermarkStore creates a store backed by the given SurrealDB client.
func NewLivingWikiWatermarkStore(client *SurrealDB) *LivingWikiWatermarkStore {
	return &LivingWikiWatermarkStore{client: client}
}

// Compile-time interface check.
var _ orchestrator.WatermarkStore = (*LivingWikiWatermarkStore)(nil)

type surrealLWWatermark struct {
	ID                 *models.RecordID `json:"id,omitempty"`
	RepoID             string           `json:"repo_id"`
	SourceProcessedSHA string           `json:"source_processed_sha"`
	WikiPublishedSHA   string           `json:"wiki_published_sha"`
	UpdatedAt          surrealTime      `json:"updated_at"`
}

func (s *LivingWikiWatermarkStore) Get(ctx context.Context, repoID string) (orchestrator.Watermarks, error) {
	db := s.client.DB()
	if db == nil {
		return orchestrator.Watermarks{}, fmt.Errorf("livingwiki watermark store: database not connected")
	}
	sql := `SELECT * FROM lw_watermarks WHERE repo_id = $repo_id LIMIT 1`
	rows, err := queryOne[[]surrealLWWatermark](ctx, db, sql, map[string]any{
		"repo_id": repoID,
	})
	if err != nil || len(rows) == 0 {
		// No record yet means "never generated" — return zero value watermarks.
		return orchestrator.Watermarks{}, nil
	}
	return orchestrator.Watermarks{
		SourceProcessedSHA: rows[0].SourceProcessedSHA,
		WikiPublishedSHA:   rows[0].WikiPublishedSHA,
	}, nil
}

func (s *LivingWikiWatermarkStore) AdvanceProcessed(ctx context.Context, repoID, sha string) error {
	db := s.client.DB()
	if db == nil {
		return fmt.Errorf("livingwiki watermark store: database not connected")
	}
	sql := `
		UPSERT lw_watermarks
		SET repo_id              = $repo_id,
		    source_processed_sha = $sha,
		    updated_at           = time::now()
		WHERE repo_id = $repo_id
	`
	_, err := surrealdb.Query[interface{}](ctx, db, sql, map[string]any{
		"repo_id": repoID,
		"sha":     sha,
	})
	return err
}

func (s *LivingWikiWatermarkStore) AdvancePublished(ctx context.Context, repoID, sha string) error {
	db := s.client.DB()
	if db == nil {
		return fmt.Errorf("livingwiki watermark store: database not connected")
	}
	sql := `
		UPSERT lw_watermarks
		SET repo_id              = $repo_id,
		    source_processed_sha = $sha,
		    wiki_published_sha   = $sha,
		    updated_at           = time::now()
		WHERE repo_id = $repo_id
	`
	_, err := surrealdb.Query[interface{}](ctx, db, sql, map[string]any{
		"repo_id": repoID,
		"sha":     sha,
	})
	return err
}

func (s *LivingWikiWatermarkStore) Reset(ctx context.Context, repoID, sha string) error {
	return s.AdvancePublished(ctx, repoID, sha)
}
