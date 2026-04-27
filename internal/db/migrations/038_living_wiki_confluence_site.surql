-- Migration 038: Add confluence_site to living-wiki settings
-- Stores the Atlassian Cloud site subdomain (e.g. "mycompany" for
-- mycompany.atlassian.net). Not a secret; stored in plaintext.
-- Existing rows default to empty string (no-op for current deployments).

DEFINE FIELD IF NOT EXISTS confluence_site ON lw_settings TYPE string DEFAULT '';

UPSERT schema_version:v38 SET version = 38;
