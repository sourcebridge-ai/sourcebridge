-- Migration 021: First-class repository understanding
-- Formalizes the persisted summary tree as a shared understanding record and
-- links downstream artifacts to the understanding revision they were built on.

DEFINE TABLE IF NOT EXISTS ca_repository_understanding SCHEMAFULL;

DEFINE FIELD IF NOT EXISTS repo_id ON ca_repository_understanding TYPE string;
DEFINE FIELD IF NOT EXISTS scope_type ON ca_repository_understanding TYPE string DEFAULT 'repository';
DEFINE FIELD IF NOT EXISTS scope_key ON ca_repository_understanding TYPE string DEFAULT 'repository:';
DEFINE FIELD IF NOT EXISTS scope_path ON ca_repository_understanding TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS corpus_id ON ca_repository_understanding TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS revision_fp ON ca_repository_understanding TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS strategy ON ca_repository_understanding TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS stage ON ca_repository_understanding TYPE string DEFAULT 'building_tree';
DEFINE FIELD IF NOT EXISTS tree_status ON ca_repository_understanding TYPE string DEFAULT 'missing';
DEFINE FIELD IF NOT EXISTS cached_nodes ON ca_repository_understanding TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS total_nodes ON ca_repository_understanding TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS model_used ON ca_repository_understanding TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS metadata ON ca_repository_understanding TYPE string DEFAULT '{}';
DEFINE FIELD IF NOT EXISTS error_code ON ca_repository_understanding TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS error_message ON ca_repository_understanding TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS created_at ON ca_repository_understanding TYPE datetime DEFAULT time::now();
DEFINE FIELD IF NOT EXISTS updated_at ON ca_repository_understanding TYPE datetime DEFAULT time::now();

DEFINE INDEX IF NOT EXISTS idx_repo_understanding_repo ON ca_repository_understanding FIELDS repo_id;
DEFINE INDEX IF NOT EXISTS idx_repo_understanding_scope ON ca_repository_understanding FIELDS repo_id, scope_key UNIQUE;

DEFINE FIELD IF NOT EXISTS understanding_id ON ca_knowledge_artifact TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS understanding_revision_fp ON ca_knowledge_artifact TYPE string DEFAULT '';

UPSERT schema_version:v21 SET version = 21;
