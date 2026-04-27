-- Migration 036: Per-repo living-wiki settings + job results tables
--
-- lw_repo_settings: one row per (tenant_id, repo_id) living-wiki opt-in.
-- lw_job_results:   one row per completed (or in-progress) living-wiki job.
--
-- Also adds tenant_id to the global lw_settings table (backcompat for
-- existing single-tenant installs — existing rows get 'default').

-- ─────────────────────────────────────────────────────────────────────────────
-- Per-repo settings
-- ─────────────────────────────────────────────────────────────────────────────

DEFINE TABLE IF NOT EXISTS lw_repo_settings SCHEMAFULL;

DEFINE FIELD IF NOT EXISTS tenant_id           ON TABLE lw_repo_settings TYPE string DEFAULT 'default';
DEFINE FIELD IF NOT EXISTS repo_id             ON TABLE lw_repo_settings TYPE string;
DEFINE FIELD IF NOT EXISTS enabled             ON TABLE lw_repo_settings TYPE bool DEFAULT false;
DEFINE FIELD IF NOT EXISTS mode                ON TABLE lw_repo_settings TYPE string DEFAULT 'PR_REVIEW';
DEFINE FIELD IF NOT EXISTS sinks               ON TABLE lw_repo_settings TYPE array DEFAULT [];
DEFINE FIELD IF NOT EXISTS exclude_paths       ON TABLE lw_repo_settings TYPE array DEFAULT [];
DEFINE FIELD IF NOT EXISTS stale_when_strategy ON TABLE lw_repo_settings TYPE string DEFAULT 'DIRECT';
DEFINE FIELD IF NOT EXISTS max_pages_per_job   ON TABLE lw_repo_settings TYPE int DEFAULT 50;
DEFINE FIELD IF NOT EXISTS last_run_at         ON TABLE lw_repo_settings TYPE option<datetime>;
DEFINE FIELD IF NOT EXISTS disabled_at         ON TABLE lw_repo_settings TYPE option<datetime>;
DEFINE FIELD IF NOT EXISTS updated_at          ON TABLE lw_repo_settings TYPE datetime DEFAULT time::now();
DEFINE FIELD IF NOT EXISTS updated_by          ON TABLE lw_repo_settings TYPE string DEFAULT '';

-- Unique composite key: one settings row per (tenant, repo).
DEFINE INDEX IF NOT EXISTS idx_lw_repo_settings_pk
    ON TABLE lw_repo_settings COLUMNS tenant_id, repo_id UNIQUE;

-- Accelerates ListEnabledRepos used by R6 scheduler.
DEFINE INDEX IF NOT EXISTS idx_lw_repo_settings_enabled
    ON TABLE lw_repo_settings COLUMNS tenant_id, enabled;

-- ─────────────────────────────────────────────────────────────────────────────
-- Backcompat: add tenant_id to the global settings table (migration 035)
-- ─────────────────────────────────────────────────────────────────────────────

DEFINE FIELD IF NOT EXISTS tenant_id ON TABLE lw_settings TYPE string DEFAULT 'default';
DEFINE INDEX IF NOT EXISTS idx_lw_settings_tenant ON TABLE lw_settings COLUMNS tenant_id;

-- ─────────────────────────────────────────────────────────────────────────────
-- Per-job result records
-- ─────────────────────────────────────────────────────────────────────────────

DEFINE TABLE IF NOT EXISTS lw_job_results SCHEMAFULL;

DEFINE FIELD IF NOT EXISTS tenant_id             ON TABLE lw_job_results TYPE string DEFAULT 'default';
DEFINE FIELD IF NOT EXISTS repo_id               ON TABLE lw_job_results TYPE string;
DEFINE FIELD IF NOT EXISTS job_id                ON TABLE lw_job_results TYPE string;
DEFINE FIELD IF NOT EXISTS started_at            ON TABLE lw_job_results TYPE datetime;
DEFINE FIELD IF NOT EXISTS completed_at          ON TABLE lw_job_results TYPE option<datetime>;
DEFINE FIELD IF NOT EXISTS pages_planned         ON TABLE lw_job_results TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS pages_generated       ON TABLE lw_job_results TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS pages_excluded        ON TABLE lw_job_results TYPE int DEFAULT 0;
-- excluded_page_ids: used by retryExcludedOnly to build the allowlist for the
-- next job (avoids re-generating pages that were intentionally excluded).
DEFINE FIELD IF NOT EXISTS excluded_page_ids     ON TABLE lw_job_results TYPE array DEFAULT [];
-- generated_page_titles: historical record of what this job produced, so the
-- "Generated pages" collapsible in the UI can show the last job's output
-- without querying the live PageStore.
DEFINE FIELD IF NOT EXISTS generated_page_titles ON TABLE lw_job_results TYPE array DEFAULT [];
DEFINE FIELD IF NOT EXISTS exclusion_reasons     ON TABLE lw_job_results TYPE array DEFAULT [];
DEFINE FIELD IF NOT EXISTS status                ON TABLE lw_job_results TYPE string DEFAULT 'running';
DEFINE FIELD IF NOT EXISTS error_message         ON TABLE lw_job_results TYPE string DEFAULT '';

-- Accelerates "latest job for repo" lookups used by the settings panel summary.
DEFINE INDEX IF NOT EXISTS idx_lw_job_results_repo
    ON TABLE lw_job_results COLUMNS tenant_id, repo_id;

-- ─────────────────────────────────────────────────────────────────────────────
-- Schema version marker
-- ─────────────────────────────────────────────────────────────────────────────

UPSERT schema_version:v36 SET version = 36;
