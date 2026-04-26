-- Migration 035: Living-wiki settings table
-- Stores a single "default" record with all living-wiki configuration.
-- Secret fields (tokens, webhook secrets) are stored as AES-256-GCM
-- ciphertext (base64-encoded) by the Go layer before writing.

DEFINE TABLE IF NOT EXISTS lw_settings SCHEMAFULL;

-- Orchestration knobs
DEFINE FIELD IF NOT EXISTS enabled_set ON lw_settings TYPE bool DEFAULT false;
DEFINE FIELD IF NOT EXISTS enabled ON lw_settings TYPE bool DEFAULT false;
DEFINE FIELD IF NOT EXISTS worker_count ON lw_settings TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS event_timeout ON lw_settings TYPE string DEFAULT '';

-- Source integration credentials (AES-256-GCM encrypted by Go layer)
DEFINE FIELD IF NOT EXISTS github_token ON lw_settings TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS gitlab_token ON lw_settings TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS confluence_email ON lw_settings TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS confluence_token ON lw_settings TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS notion_token ON lw_settings TYPE string DEFAULT '';

-- Webhook secrets (AES-256-GCM encrypted by Go layer)
DEFINE FIELD IF NOT EXISTS confluence_webhook_secret ON lw_settings TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS notion_webhook_secret ON lw_settings TYPE string DEFAULT '';

-- Audit fields
DEFINE FIELD IF NOT EXISTS updated_by ON lw_settings TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS updated_at ON lw_settings TYPE datetime DEFAULT time::now();

UPSERT schema_version:v35 SET version = 35;
