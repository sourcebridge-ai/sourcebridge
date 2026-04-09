-- Migration 014: Persist structured failure metadata on knowledge artifacts

DEFINE FIELD IF NOT EXISTS error_code ON ca_knowledge_artifact TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS error_message ON ca_knowledge_artifact TYPE string DEFAULT '';

UPDATE ca_knowledge_artifact SET
  error_code = '',
  error_message = ''
WHERE error_code = NONE OR error_code = NULL OR error_message = NONE OR error_message = NULL;

UPSERT schema_version:v14 SET version = 14;
