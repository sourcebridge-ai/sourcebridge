-- Migration 015: LLM job orchestration table
-- Introduces ca_llm_job, the unified persistence layer for every LLM-backed
-- job in the system (knowledge artifacts, reasoning, requirements, linking).
-- The job record captures lifecycle state, progress, errors, retry metadata,
-- and usage metrics so a single Monitor page can render activity across all
-- subsystems.

DEFINE TABLE IF NOT EXISTS ca_llm_job SCHEMAFULL;

DEFINE FIELD IF NOT EXISTS subsystem ON ca_llm_job TYPE string;
DEFINE FIELD IF NOT EXISTS job_type ON ca_llm_job TYPE string;
DEFINE FIELD IF NOT EXISTS target_key ON ca_llm_job TYPE string;
DEFINE FIELD IF NOT EXISTS strategy ON ca_llm_job TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS model ON ca_llm_job TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS status ON ca_llm_job TYPE string DEFAULT 'pending';
DEFINE FIELD IF NOT EXISTS progress ON ca_llm_job TYPE float DEFAULT 0.0;
DEFINE FIELD IF NOT EXISTS progress_phase ON ca_llm_job TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS progress_message ON ca_llm_job TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS error_code ON ca_llm_job TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS error_message ON ca_llm_job TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS retry_count ON ca_llm_job TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS max_attempts ON ca_llm_job TYPE int DEFAULT 1;
DEFINE FIELD IF NOT EXISTS timeout_sec ON ca_llm_job TYPE int DEFAULT 300;
DEFINE FIELD IF NOT EXISTS input_tokens ON ca_llm_job TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS output_tokens ON ca_llm_job TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS snapshot_bytes ON ca_llm_job TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS artifact_id ON ca_llm_job TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS repo_id ON ca_llm_job TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS created_at ON ca_llm_job TYPE datetime DEFAULT time::now();
DEFINE FIELD IF NOT EXISTS started_at ON ca_llm_job TYPE option<datetime>;
DEFINE FIELD IF NOT EXISTS updated_at ON ca_llm_job TYPE datetime DEFAULT time::now();
DEFINE FIELD IF NOT EXISTS completed_at ON ca_llm_job TYPE option<datetime>;

-- Monitor "Now Running" tile query: SELECT WHERE status IN ('pending','generating') ORDER BY updated_at DESC
DEFINE INDEX IF NOT EXISTS idx_llm_job_status ON ca_llm_job FIELDS status, updated_at;

-- Monitor "Recent history" query: SELECT WHERE subsystem = ? AND status IN (...) ORDER BY updated_at DESC
DEFINE INDEX IF NOT EXISTS idx_llm_job_subsystem_status ON ca_llm_job FIELDS subsystem, status, updated_at;

-- In-flight dedupe lookup: SELECT WHERE target_key = ? AND status IN ('pending','generating')
DEFINE INDEX IF NOT EXISTS idx_llm_job_target ON ca_llm_job FIELDS target_key, status;

-- Per-repo filtering for the repo-scoped popover
DEFINE INDEX IF NOT EXISTS idx_llm_job_repo ON ca_llm_job FIELDS repo_id, updated_at;

UPSERT schema_version:v15 SET version = 15;
