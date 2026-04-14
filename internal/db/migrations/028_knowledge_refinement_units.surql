-- Migration 028: durable section-level refinement units

DEFINE TABLE IF NOT EXISTS ca_knowledge_refinement SCHEMALESS;

DEFINE FIELD IF NOT EXISTS artifact_id ON ca_knowledge_refinement TYPE string;
DEFINE FIELD IF NOT EXISTS section_key ON ca_knowledge_refinement TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS section_title ON ca_knowledge_refinement TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS refinement_type ON ca_knowledge_refinement TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS status ON ca_knowledge_refinement TYPE string DEFAULT 'queued';
DEFINE FIELD IF NOT EXISTS attempt_count ON ca_knowledge_refinement TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS understanding_id ON ca_knowledge_refinement TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS evidence_revision_fp ON ca_knowledge_refinement TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS renderer_version ON ca_knowledge_refinement TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS last_error ON ca_knowledge_refinement TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS metadata ON ca_knowledge_refinement TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS created_at ON ca_knowledge_refinement TYPE datetime DEFAULT time::now();
DEFINE FIELD IF NOT EXISTS updated_at ON ca_knowledge_refinement TYPE datetime DEFAULT time::now();

DEFINE INDEX IF NOT EXISTS idx_knowledge_refinement_artifact ON TABLE ca_knowledge_refinement COLUMNS artifact_id;
DEFINE INDEX IF NOT EXISTS uniq_knowledge_refinement_unit ON TABLE ca_knowledge_refinement COLUMNS artifact_id, section_key, refinement_type UNIQUE;
