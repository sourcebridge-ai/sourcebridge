-- Migration 041: Fix array-typed fields on living-wiki tables
--
-- SurrealDB SCHEMAFULL mode treats `TYPE array` (with no element type and no
-- FLEXIBLE marker) as "an array of strictly-defined elements." Since none of
-- our array fields have per-element schema definitions, every save silently
-- dropped the array contents and stored []. This bit:
--   lw_repo_settings.sinks (array of objects)
--   lw_repo_settings.exclude_paths (array of strings)
--   lw_job_results.excluded_page_ids (array of strings)
--   lw_job_results.generated_page_titles (array of strings)
--   lw_job_results.exclusion_reasons (array of strings)
--
-- Symptom: every Enable Living Wiki click that saved sinks "succeeded" with
-- no error but the database row had sinks=[]. The cold-start dispatch then
-- silently no-op'd because sinks was empty, so no pages reached any sink.
-- Confirmed via direct SurrealDB query: UPSERT with sinks=[{...obj...}]
-- returned status:OK but the row showed sinks: [].
--
-- Fix:
--   - For arrays of objects (sinks): FLEXIBLE TYPE array. FLEXIBLE allows
--     arbitrary nested object shapes without DEFINE FIELDs for each key.
--   - For arrays of primitives: TYPE array<string>. Strict element type
--     means SurrealDB validates and stores each element correctly.
--
-- REMOVE then DEFINE rather than DEFINE OVERWRITE for compatibility with
-- older SurrealDB versions that don't support OVERWRITE.

REMOVE FIELD sinks ON TABLE lw_repo_settings;
REMOVE FIELD exclude_paths ON TABLE lw_repo_settings;
REMOVE FIELD excluded_page_ids ON TABLE lw_job_results;
REMOVE FIELD generated_page_titles ON TABLE lw_job_results;
REMOVE FIELD exclusion_reasons ON TABLE lw_job_results;

DEFINE FIELD sinks                 ON TABLE lw_repo_settings FLEXIBLE TYPE array     DEFAULT [];
DEFINE FIELD exclude_paths         ON TABLE lw_repo_settings TYPE array<string>      DEFAULT [];
DEFINE FIELD excluded_page_ids     ON TABLE lw_job_results   TYPE array<string>      DEFAULT [];
DEFINE FIELD generated_page_titles ON TABLE lw_job_results   TYPE array<string>      DEFAULT [];
DEFINE FIELD exclusion_reasons     ON TABLE lw_job_results   TYPE array<string>      DEFAULT [];

-- Schema version marker
UPSERT schema_version:v41 SET version = 41;
