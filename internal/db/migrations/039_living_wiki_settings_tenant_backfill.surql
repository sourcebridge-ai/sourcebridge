-- Migration 039: Backfill tenant_id on the existing lw_settings row.
--
-- Migration 036 added `tenant_id` to lw_settings (DEFAULT 'default') as
-- backwards-compat for the global settings record. SurrealDB DEFAULT only
-- applies on row creation, not retroactively, so the pre-existing
-- `lw_settings:default` row from migration 035 still has tenant_id = NONE.
-- Subsequent UPSERTs that omit tenant_id fail schema validation:
--
--   Found NONE for field `tenant_id`, with record `lw_settings:default`,
--   but expected a string
--
-- Backfill any rows where tenant_id is missing.

UPDATE lw_settings SET tenant_id = 'default' WHERE tenant_id IS NONE;

-- Schema version marker
UPSERT schema_version:v39 SET version = 39;
