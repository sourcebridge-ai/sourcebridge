DEFINE FIELD IF NOT EXISTS metadata ON ca_knowledge_section TYPE string DEFAULT '';

UPDATE ca_knowledge_section SET
  section_key = string::trim(section_key ?? ''),
  refinement_status = string::trim(refinement_status ?? ''),
  metadata = string::trim(metadata ?? '');
