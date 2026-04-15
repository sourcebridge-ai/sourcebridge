-- Migration 022: Understanding follow-on metadata, dependencies, and job mode

DEFINE FIELD IF NOT EXISTS renderer_version ON ca_knowledge_artifact TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS generation_mode ON ca_knowledge_artifact TYPE string DEFAULT 'understanding_first';

DEFINE FIELD IF NOT EXISTS section_key ON ca_knowledge_section TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS refinement_status ON ca_knowledge_section TYPE string DEFAULT '';

DEFINE TABLE IF NOT EXISTS ca_knowledge_dependency SCHEMAFULL;
DEFINE FIELD IF NOT EXISTS artifact_id ON ca_knowledge_dependency TYPE string;
DEFINE FIELD IF NOT EXISTS dependency_type ON ca_knowledge_dependency TYPE string;
DEFINE FIELD IF NOT EXISTS target_id ON ca_knowledge_dependency TYPE string;
DEFINE FIELD IF NOT EXISTS target_revision_fp ON ca_knowledge_dependency TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS metadata ON ca_knowledge_dependency TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS created_at ON ca_knowledge_dependency TYPE datetime DEFAULT time::now();
DEFINE INDEX IF NOT EXISTS idx_knowledge_dependency_artifact ON ca_knowledge_dependency FIELDS artifact_id;

DEFINE FIELD IF NOT EXISTS priority ON ca_llm_job TYPE string DEFAULT 'interactive';
DEFINE FIELD IF NOT EXISTS generation_mode ON ca_llm_job TYPE string DEFAULT '';

UPSERT schema_version:v22 SET version = 22;
