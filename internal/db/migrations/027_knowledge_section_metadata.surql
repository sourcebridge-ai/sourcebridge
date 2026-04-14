DEFINE FIELD IF NOT EXISTS metadata ON ca_knowledge_section TYPE string DEFAULT '';

UPDATE ca_knowledge_section SET metadata = string::trim(metadata ?? '');
