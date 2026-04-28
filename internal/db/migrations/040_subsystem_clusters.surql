-- Migration 040: Subsystem clustering tables
-- Adds cluster and cluster_member tables for label-propagation clustering
-- results, plus a cluster_graph_edge_hash field on ca_repository for
-- delta-skip optimization.

DEFINE TABLE cluster SCHEMAFULL;
DEFINE FIELD repo_id    ON cluster TYPE string;
DEFINE FIELD label      ON cluster TYPE string;
DEFINE FIELD llm_label  ON cluster TYPE option<string>;
DEFINE FIELD size       ON cluster TYPE int;
DEFINE FIELD edge_hash  ON cluster TYPE string;
DEFINE FIELD partial    ON cluster TYPE bool DEFAULT false;
DEFINE FIELD created_at ON cluster TYPE datetime;
DEFINE FIELD updated_at ON cluster TYPE datetime;

DEFINE TABLE cluster_member SCHEMAFULL;
DEFINE FIELD cluster_id ON cluster_member TYPE record<cluster>;
DEFINE FIELD symbol_id  ON cluster_member TYPE string;
DEFINE FIELD repo_id    ON cluster_member TYPE string;

DEFINE INDEX idx_cluster_repo   ON cluster        FIELDS repo_id;
DEFINE INDEX idx_member_cluster ON cluster_member FIELDS cluster_id;
DEFINE INDEX idx_member_symbol  ON cluster_member FIELDS symbol_id, repo_id;

-- Add cluster_graph_edge_hash to the existing repository table.
-- Stores the SHA-256 of sorted (caller_id, callee_id) pairs so the
-- clustering job can skip re-runs when the call graph hasn't changed.
DEFINE FIELD IF NOT EXISTS cluster_graph_edge_hash ON ca_repository TYPE string DEFAULT '';

UPSERT schema_version:v40 SET version = 40;
