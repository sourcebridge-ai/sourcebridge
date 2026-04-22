-- Migration 033: FTS analyzer + BM25 search index on ca_symbol.
--
-- Part of the hybrid retrieval rollout. Replaces the previous
-- `CONTAINS`-only path in SurrealStore.SearchContent / GetSymbols with
-- a BM25-ranked full-text index.
--
-- Analyzer rationale:
--
--   - `class` tokenizer splits on Unicode class boundaries, which
--     handles the common CamelCase / snake_case / dotted-name mix in
--     identifiers better than whitespace-only tokenizers.
--   - `lowercase` + `ascii` filters make queries case-insensitive and
--     strip diacritics, matching the existing substring-path behavior.
--
-- Index rationale:
--
--   - BM25 scoring + HIGHLIGHTS is sufficient for the Phase 1 FTS
--     contract and is retrievable via `search::score(1)` in queries.
--   - Fields `name`, `qualified_name`, `doc_comment`, `signature`
--     form the searchable projection of a symbol; keeping these in
--     one composite index means the service issues a single query
--     instead of merging per-field results.
--
-- Idempotency:
--
--   - `DEFINE ANALYZER IF NOT EXISTS` and `DEFINE INDEX IF NOT
--     EXISTS` make this safe to replay on every startup, consistent
--     with how every prior migration in this repo has shipped.
--
-- Rollback story:
--
--   - If the analyzer produces bad tokenization for a language we
--     care about, drop the index via `REMOVE INDEX`, tune analyzer,
--     and re-apply. Search falls back to the substring helpers behind
--     the feature flag while the rebuild runs.
--
-- Known caveats:
--
--   - `ca_symbol` does not yet carry a `deleted_at` column
--     (soft-delete was migration 031 for requirements / links / knowledge
--     artifacts). V1 indexes all symbols. The application-level
--     re-index path already hard-replaces symbols via
--     ReplaceIndexResult, so stale rows are not long-lived.
--
-- See thoughts/shared/plans/2026-04-22-hybrid-retrieval-search.md
-- §Phase 1.

DEFINE ANALYZER IF NOT EXISTS code_analyzer
    TOKENIZERS class
    FILTERS lowercase, ascii;

DEFINE INDEX IF NOT EXISTS idx_ca_symbol_fts
    ON ca_symbol
    FIELDS name, qualified_name, doc_comment, signature
    SEARCH ANALYZER code_analyzer BM25 HIGHLIGHTS;

UPSERT schema_version:v33 SET version = 33;
