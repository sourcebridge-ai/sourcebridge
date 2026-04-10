-- Migration 020: Reports tables
-- Stores report records, templates, branding, and evidence.

DEFINE TABLE IF NOT EXISTS ca_report SCHEMAFULL;

DEFINE FIELD IF NOT EXISTS name ON ca_report TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS report_type ON ca_report TYPE string DEFAULT 'architecture_baseline';
DEFINE FIELD IF NOT EXISTS audience ON ca_report TYPE string DEFAULT 'technical_leadership';
DEFINE FIELD IF NOT EXISTS repository_ids ON ca_report TYPE string DEFAULT '[]';
DEFINE FIELD IF NOT EXISTS selected_sections ON ca_report TYPE string DEFAULT '[]';
DEFINE FIELD IF NOT EXISTS include_diagrams ON ca_report TYPE bool DEFAULT false;
DEFINE FIELD IF NOT EXISTS output_formats ON ca_report TYPE string DEFAULT '["markdown"]';
DEFINE FIELD IF NOT EXISTS loe_mode ON ca_report TYPE string DEFAULT 'human_hours';
DEFINE FIELD IF NOT EXISTS branding_id ON ca_report TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS template_id ON ca_report TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS status ON ca_report TYPE string DEFAULT 'pending';
DEFINE FIELD IF NOT EXISTS progress ON ca_report TYPE float DEFAULT 0.0;
DEFINE FIELD IF NOT EXISTS progress_phase ON ca_report TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS progress_message ON ca_report TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS error_code ON ca_report TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS error_message ON ca_report TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS version ON ca_report TYPE int DEFAULT 1;
DEFINE FIELD IF NOT EXISTS content_dir ON ca_report TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS section_count ON ca_report TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS word_count ON ca_report TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS evidence_count ON ca_report TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS stale ON ca_report TYPE bool DEFAULT false;
DEFINE FIELD IF NOT EXISTS created_by ON ca_report TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS created_at ON ca_report TYPE datetime DEFAULT time::now();
DEFINE FIELD IF NOT EXISTS updated_at ON ca_report TYPE datetime DEFAULT time::now();
DEFINE FIELD IF NOT EXISTS completed_at ON ca_report TYPE option<datetime>;

DEFINE INDEX IF NOT EXISTS idx_ca_report_status ON ca_report FIELDS status;
DEFINE INDEX IF NOT EXISTS idx_ca_report_type ON ca_report FIELDS report_type;
DEFINE INDEX IF NOT EXISTS idx_ca_report_created ON ca_report FIELDS created_at;

-- Report templates for one-click quick generate
DEFINE TABLE IF NOT EXISTS ca_report_template SCHEMAFULL;

DEFINE FIELD IF NOT EXISTS name ON ca_report_template TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS report_type ON ca_report_template TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS audience ON ca_report_template TYPE string DEFAULT 'technical_leadership';
DEFINE FIELD IF NOT EXISTS repository_ids ON ca_report_template TYPE string DEFAULT '[]';
DEFINE FIELD IF NOT EXISTS selected_sections ON ca_report_template TYPE string DEFAULT '[]';
DEFINE FIELD IF NOT EXISTS include_diagrams ON ca_report_template TYPE bool DEFAULT false;
DEFINE FIELD IF NOT EXISTS output_formats ON ca_report_template TYPE string DEFAULT '["markdown"]';
DEFINE FIELD IF NOT EXISTS loe_mode ON ca_report_template TYPE string DEFAULT 'human_hours';
DEFINE FIELD IF NOT EXISTS branding_id ON ca_report_template TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS created_by ON ca_report_template TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS created_at ON ca_report_template TYPE datetime DEFAULT time::now();
DEFINE FIELD IF NOT EXISTS updated_at ON ca_report_template TYPE datetime DEFAULT time::now();

-- Report branding (logo, colors, cover text)
DEFINE TABLE IF NOT EXISTS ca_report_branding SCHEMAFULL;

DEFINE FIELD IF NOT EXISTS name ON ca_report_branding TYPE string DEFAULT 'Default';
DEFINE FIELD IF NOT EXISTS logo_path ON ca_report_branding TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS primary_color ON ca_report_branding TYPE string DEFAULT '#2563eb';
DEFINE FIELD IF NOT EXISTS accent_color ON ca_report_branding TYPE string DEFAULT '#059669';
DEFINE FIELD IF NOT EXISTS cover_title ON ca_report_branding TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS cover_subtitle ON ca_report_branding TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS footer_text ON ca_report_branding TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS created_by ON ca_report_branding TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS created_at ON ca_report_branding TYPE datetime DEFAULT time::now();
DEFINE FIELD IF NOT EXISTS updated_at ON ca_report_branding TYPE datetime DEFAULT time::now();

-- Report evidence items
DEFINE TABLE IF NOT EXISTS ca_report_evidence SCHEMAFULL;

DEFINE FIELD IF NOT EXISTS report_id ON ca_report_evidence TYPE string;
DEFINE FIELD IF NOT EXISTS evidence_id ON ca_report_evidence TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS category ON ca_report_evidence TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS title ON ca_report_evidence TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS description ON ca_report_evidence TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS source_type ON ca_report_evidence TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS source_repo_id ON ca_report_evidence TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS file_path ON ca_report_evidence TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS line_start ON ca_report_evidence TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS line_end ON ca_report_evidence TYPE int DEFAULT 0;
DEFINE FIELD IF NOT EXISTS code_snippet ON ca_report_evidence TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS raw_data ON ca_report_evidence TYPE string DEFAULT '';
DEFINE FIELD IF NOT EXISTS severity ON ca_report_evidence TYPE string DEFAULT 'info';
DEFINE FIELD IF NOT EXISTS created_at ON ca_report_evidence TYPE datetime DEFAULT time::now();

DEFINE INDEX IF NOT EXISTS idx_ca_report_evidence_report ON ca_report_evidence FIELDS report_id;
DEFINE INDEX IF NOT EXISTS idx_ca_report_evidence_category ON ca_report_evidence FIELDS report_id, category;

UPSERT schema_version:v20 SET version = 20;
