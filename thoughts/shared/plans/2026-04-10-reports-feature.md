# Reports Feature — Plan

**Date:** 2026-04-10
**Author:** Jay Stuart (drafted with Claude)
**Status:** Proposed
**Scope:** Go API, Python worker, Web UI, new report engine

## Overview

SourceBridge currently generates per-repository artifacts (cliff notes, learning paths, code tours, workflow stories) but has no way to produce cross-repository professional reports. Organizations evaluating, auditing, or documenting their software portfolios need structured reports that synthesize analysis across multiple repositories into a single cohesive document.

This plan adds a **Reports** tab to SourceBridge where operators can:

1. Select one or more repositories
2. Choose a report type (Architecture Baseline, SWOT, Environment Evaluation, etc.)
3. Select which categories/sections to include
4. Generate a professional document in Markdown, PDF, or Word format
5. Download or share the result

When multiple repositories are selected, the report seamlessly integrates findings across all of them, identifying commonalities, inconsistencies, and portfolio-level patterns rather than producing separate per-repo sections.

## Report Types

### 1. Software Architecture Baseline

The flagship report type. Produces a comprehensive assessment of one or more applications covering architecture, security, operations, and organizational readiness. Based on the Hoegg Software template used for the MACU engagement.

**Sections (all optional, operator selects which to include):**

| Category | Section | Data Sources |
|---|---|---|
| **Executive** | Executive Summary | AI synthesis of all selected sections |
| **Executive** | Overall Assessment | AI synthesis + understanding scores |
| **Portfolio** | Applications Inventory | Repository metadata, file counts, languages |
| **Portfolio** | Application Access | Inferred from auth patterns in code |
| **Security** | OWASP Security Findings | Automated OWASP scan (new capability) |
| **Security** | Software Vulnerability Management | Dependency audit (npm audit, pip audit) |
| **Security** | Supply Chain Vulnerabilities | Package analysis |
| **Security** | Protecting User Information | TLS analysis, data handling patterns |
| **Security** | Compliance Considerations | Inferred from data types + industry (FERPA, GLBA, HIPAA, PCI-DSS, SOC 2) |
| **Access** | Authentication (AAA) | Auth pattern detection from code |
| **Access** | Authorization | Role/permission pattern analysis |
| **Access** | Audit Trail | Logging pattern analysis |
| **Operations** | System Availability | Infrastructure pattern analysis |
| **Operations** | Incident Response | Runbook/doc detection |
| **Operations** | Deployment Architecture | Dockerfile, CI/CD, cloud config analysis |
| **Operations** | Application Secrets | Secret pattern detection (.env, vault references) |
| **Delivery** | Requirements and Project Management | Tool detection (Linear, Jira, etc.) |
| **Delivery** | Testing | Test file detection, coverage inference |
| **Delivery** | Build and Deployment | CI/CD pipeline analysis |
| **Delivery** | Monitoring and Configuration | Observability tool detection |
| **Delivery** | Bugfixes and Patches | Git history analysis |
| **Governance** | Application Ownership and Responsibility | CODEOWNERS, contributor analysis |
| **Governance** | Access and Administrative Control | Platform access analysis |
| **Data** | Data Inventory and Handling | Schema/model analysis |
| **Data** | Third-Party Services and Dependencies | Package.json, requirements.txt analysis |
| **Data** | Secrets and Credential Management | Secret pattern analysis |
| **Data** | Backup, Recovery, and Resilience | Infrastructure config analysis |
| **Observability** | Logging, Monitoring, and Observability | Log library detection |
| **Integration** | Integration and System Interactions | API call detection, shared DB detection |
| **Operations** | Operational Workflow and Support Model | Doc/runbook analysis |
| **Engineering** | Source Control and Development Practices | Git workflow analysis (branching, PRs, reviews) |
| **Engineering** | Documentation and Knowledge Management | Doc file detection, README analysis |
| **Engineering** | Application Lifecycle and Maintenance | Dependency age, update frequency |
| **Users** | System Usage and User Base | Inferred from auth + UI patterns |
| **Users** | Training and User Enablement | Doc/guide detection |
| **Review** | Architecture Review Key Findings | AI deep analysis synthesis |
| **Appendix** | Team Members | Git contributor extraction |
| **Appendix** | OWASP Security Scan Results | Per-repo detailed findings |
| **Appendix** | Dependency Audit Results | Per-repo npm/pip audit |

### 2. SWOT Analysis

Strategic assessment of the software portfolio.

**Sections:**

| Section | Description | Data Sources |
|---|---|---|
| Strengths | What the codebase does well | Code quality metrics, test coverage, documentation, architecture patterns |
| Weaknesses | Technical debt, gaps, risks | Missing tests, outdated deps, security findings, complexity hotspots |
| Opportunities | Improvements and modernization paths | Framework upgrades, consolidation opportunities, automation gaps |
| Threats | External and internal risks | Vulnerability exposure, compliance gaps, bus factor, dependency risks |
| Recommendations | Prioritized action items | Synthesized from all quadrants |

### 3. Environment Evaluation

Assessment of the overall technology environment as ascertained from code and available files.

**Sections:**

| Section | Description |
|---|---|
| Technology Stack Summary | Languages, frameworks, databases, cloud services across all repos |
| Infrastructure Topology | Deployment targets, hosting, networking patterns |
| Development Toolchain | Build tools, linters, formatters, CI/CD |
| Code Quality Metrics | Complexity, duplication, test ratios, doc coverage |
| Security Posture | Aggregated vulnerability findings |
| Operational Maturity | Monitoring, logging, alerting, runbooks |
| Integration Map | How systems connect, shared databases, APIs |
| Compliance Readiness | Gap analysis against common frameworks |
| Cost and Resource Implications | Cloud resource patterns, scaling indicators |
| Modernization Readiness | Framework versions, migration complexity |

### 4. Technical Due Diligence

For M&A, investment, or vendor evaluation scenarios.

**Sections:**

| Section | Description |
|---|---|
| Executive Summary | Investment-grade overview |
| Technology Risk Assessment | Technical debt quantification |
| Scalability Analysis | Architecture scaling patterns |
| Team and Knowledge Risk | Bus factor, documentation gaps |
| IP and Licensing | License analysis across dependencies |
| Security and Compliance | Vulnerability and compliance summary |
| Operational Continuity | Deployment, monitoring, incident readiness |
| Estimated Remediation Effort | Prioritized backlog with rough sizing |

### 5. Compliance Gap Analysis

Assessment against a specific framework (FERPA, HIPAA, SOC 2, PCI-DSS, NIST CSF).

**Sections:**

| Section | Description |
|---|---|
| Framework Overview | Selected compliance framework summary |
| Control Mapping | Map code evidence to framework controls |
| Gap Identification | Controls with insufficient evidence |
| Risk Rating | Per-control risk assessment |
| Remediation Roadmap | Prioritized steps to close gaps |
| Evidence Inventory | What evidence exists in the codebase |

### 6. Portfolio Health Dashboard Report

Periodic summary of all indexed repositories.

**Sections:**

| Section | Description |
|---|---|
| Portfolio Overview | Repo count, total LOC, languages, activity |
| Understanding Scores | Score breakdown per repo |
| Security Summary | Aggregated findings across repos |
| Freshness Report | Stale artifacts, outdated deps |
| Activity Summary | Recent commits, active contributors |
| Cross-Repo Patterns | Shared libraries, common patterns, inconsistencies |

## Multi-Repository Analysis

When multiple repos are selected, the report engine does NOT produce isolated per-repo sections. Instead:

1. **Shared context**: The engine first identifies what the repos have in common (shared tech stack, common patterns, shared dependencies, same deployment target).

2. **Unified narrative**: Each section discusses the portfolio as a whole, calling out specific repos only when they differ from the norm. Example: "Five of seven applications use Next.js with Supabase. The transcript analyzer uses Flask/React and deploys to EC2 separately."

3. **Cross-repo findings**: The engine surfaces patterns that only become visible across repos: inconsistent auth strategies, mixed deployment models, dependency version drift, etc.

4. **Per-repo appendices**: Detailed per-repo findings (OWASP scans, dependency audits) go in appendices, not the main narrative.

## Output Formats

### Markdown
- Primary internal format
- Clean, structured with proper heading hierarchy
- Tables for structured data
- Code blocks for technical evidence
- Diagram placeholders as `[DIAGRAM: description]` blocks

### PDF
- Professional formatting with cover page
- Table of contents with page numbers
- Header/footer with report title and page numbers
- Consistent typography and spacing
- Color-coded severity badges (Critical=red, High=orange, etc.)
- Generated via a headless Chromium renderer from the Markdown (Puppeteer or similar)

### Word (.docx)
- Editable format for client delivery
- Same structure as PDF but in .docx
- Proper Word styles (Heading 1, Heading 2, Body, etc.) so users can update the TOC
- Generated via `docx` npm package or Python `python-docx`

### Diagram Placeholders
When the operator enables "Include diagram placeholders":
- Architecture diagrams: `[DIAGRAM: Deployment Architecture — show hosting topology, data flow, and external service connections]`
- Integration maps: `[DIAGRAM: System Integration Map — show how applications connect to each other and shared services]`
- Data flow: `[DIAGRAM: Data Flow Diagram — show how sensitive data moves through the system]`
- Network topology: `[DIAGRAM: Network Topology — show infrastructure layout]`

These are formatted as styled placeholder boxes in PDF/Word with enough description that someone can draw the actual diagram.

## Data Pipeline

### Phase 1: Data Collection

For each selected repository, gather:

1. **Existing SourceBridge data** (already available):
   - Repository metadata (name, path, file count, languages)
   - Symbol index (functions, classes, modules)
   - Understanding score breakdown
   - Cliff notes sections (if generated)
   - Learning path (if generated)
   - Code tour (if generated)
   - Workflow stories (if generated)
   - Requirement links (if imported)
   - Architecture diagrams (if generated)

2. **New analysis passes** (run on demand during report generation):
   - **Dependency audit**: `npm audit --json` / `pip audit --json` output
   - **OWASP scan**: Static analysis against OWASP Top 10 patterns
   - **Git history analysis**: Branching patterns, PR usage, contributor activity
   - **Auth pattern detection**: Scan for authentication/authorization patterns
   - **Secret detection**: Scan for exposed credentials, .env patterns
   - **Test detection**: Find test files, test frameworks, coverage configs
   - **CI/CD detection**: Find GitHub Actions, Dockerfile, deployment configs
   - **License scan**: Analyze dependency licenses

### Phase 2: Cross-Repository Synthesis

When multiple repos are selected:

1. Build a **portfolio context document** that describes:
   - Common technology stack
   - Shared deployment patterns
   - Consistent vs inconsistent practices
   - Portfolio-level metrics (total LOC, total deps, etc.)

2. Feed this context into every section's generation prompt so the LLM writes in portfolio terms, not per-repo terms.

### Phase 3: Section Generation

For each selected section:

1. Assemble the relevant data slice (e.g., for "Testing", gather test file counts, framework detection results, coverage configs across all selected repos)
2. Build a section-specific prompt with the data and the portfolio context
3. Call the LLM to generate the section narrative
4. Parse the result into structured Markdown

If no data is available for a section, generate a placeholder:

```markdown
## Testing

> **No automated testing evidence was detected** across the selected repositories.
> This section would typically cover test frameworks, coverage metrics, CI integration,
> and testing practices. Consider adding automated tests to improve confidence in
> code changes.
>
> _[PLACEHOLDER — Update this section after implementing testing]_
```

### Phase 4: Assembly and Rendering

1. Combine all generated sections into the final document
2. Generate the Executive Summary last (it synthesizes all sections)
3. Add cover page, TOC, and appendices
4. Render to the selected output format(s)

## Architecture

### New Go components

```
internal/reports/
  types.go           — ReportType, ReportConfig, ReportSection, ReportStatus
  store.go           — ReportStore interface (CRUD for report records)
  engine.go          — Orchestrates data collection + section generation
  collector.go       — Gathers data from graph store, knowledge store, git
  renderer_md.go     — Assembles Markdown output
  renderer_pdf.go    — Markdown -> PDF via headless Chrome
  renderer_docx.go   — Markdown -> Word via template

internal/db/
  report_store.go    — SurrealDB implementation of ReportStore
  migrations/
    020_reports.surql — ca_report table

internal/api/rest/
  admin_reports.go   — REST handlers for report CRUD + download

internal/api/graphql/
  report_resolvers.go — GraphQL queries/mutations for reports
```

### New Python components

```
workers/reports/
  __init__.py
  section_generator.py — Per-section LLM generation
  prompts/
    architecture_baseline.py — Section prompts for arch baseline
    swot.py                  — Section prompts for SWOT
    environment_eval.py      — Section prompts for env evaluation
    due_diligence.py         — Section prompts for tech DD
    compliance_gap.py        — Section prompts for compliance
    portfolio_health.py      — Section prompts for portfolio health
  analyzers/
    dependency_audit.py      — npm audit / pip audit runner
    owasp_scanner.py         — Static OWASP pattern detection
    auth_detector.py         — Authentication pattern detection
    secret_scanner.py        — Credential/secret pattern detection
    test_detector.py         — Test framework/coverage detection
    cicd_detector.py         — CI/CD pipeline detection
    git_analyzer.py          — Git history/workflow analysis
    license_scanner.py       — Dependency license analysis
```

### New Frontend

```
web/src/app/(app)/reports/
  page.tsx                   — Reports list page
  new/page.tsx               — New report wizard
  [id]/page.tsx              — Report detail/download page

web/src/components/reports/
  ReportTypeCard.tsx          — Card for selecting report type
  RepoSelector.tsx            — Multi-repo selector
  SectionSelector.tsx         — Category/section checklist
  ReportProgress.tsx          — Generation progress display
  ReportPreview.tsx           — Markdown preview
```

## Data Model

```sql
ca_report
  id uuid primary
  name string                           -- user-provided report name
  report_type string                    -- "architecture_baseline" | "swot" | "environment_eval" | etc.
  repository_ids array<string>          -- selected repos
  selected_sections array<string>       -- which sections to include
  include_diagrams bool default false   -- include diagram placeholders
  output_formats array<string>          -- ["markdown", "pdf", "docx"]
  status string                         -- "pending" | "collecting" | "generating" | "rendering" | "ready" | "failed"
  progress float default 0.0
  progress_phase string default ''
  progress_message string default ''
  error_code string default ''
  error_message string default ''
  -- Generated content
  markdown_content string default ''    -- full rendered Markdown
  pdf_path string default ''            -- path to generated PDF
  docx_path string default ''           -- path to generated Word doc
  section_count int default 0
  word_count int default 0
  -- Metadata
  created_by string default ''
  created_at datetime default time::now()
  updated_at datetime default time::now()
  completed_at option<datetime>
```

## UI Flow

### Reports List Page (`/reports`)

- Empty state: "No reports yet. Create your first report to get a professional analysis of your codebase."
- List of past reports with: name, type badge, repo count, status, date, download buttons
- "New Report" button

### New Report Wizard (`/reports/new`)

**Step 1: Report Type**
- Grid of report type cards (Architecture Baseline, SWOT, Environment Eval, Due Diligence, Compliance Gap, Portfolio Health)
- Each card has: icon, title, one-line description, "Select" button

**Step 2: Select Repositories**
- Multi-select list of all indexed repositories
- Each shows: name, status badge, understanding score, last indexed date
- "Select All" / "Deselect All" buttons
- Minimum 1 repo required

**Step 3: Configure Sections**
- Grouped checklist of all sections for the chosen report type
- "Select All" / "Deselect All" per category
- Diagram placeholder toggle
- Output format checkboxes (Markdown, PDF, Word)
- Report name input

**Step 4: Generate**
- Summary of selections
- "Generate Report" button
- Progress display with phase labels:
  - "Collecting repository data..." (10-30%)
  - "Analyzing security patterns..." (30-50%)
  - "Generating sections..." (50-80%)
  - "Assembling document..." (80-90%)
  - "Rendering output formats..." (90-100%)

### Report Detail Page (`/reports/[id]`)

- Report header with name, type, date, word count
- Markdown preview (rendered in-browser)
- Download buttons for each generated format
- "Regenerate" button
- Section navigation sidebar

## Prompt Engineering Approach

Each section gets a dedicated prompt that includes:

1. **Portfolio context** — what repos are included, their common patterns
2. **Section-specific data** — the relevant analysis results
3. **Output format instructions** — Markdown with specific heading levels
4. **Tone and style** — professional, grounded, evidence-based
5. **Quality bar** — minimum word count, specificity requirements, evidence citations

Example prompt structure for an Architecture Baseline section:

```
You are writing the "{section_title}" section of a Software Architecture Baseline report.

Portfolio: {repo_count} repositories ({repo_names})
Common stack: {common_technologies}

Evidence for this section:
{section_specific_data}

Write this section as part of a professional architecture review document.
Requirements:
- Write in third person, present tense
- Be specific: name files, technologies, versions, and patterns
- When repos differ, explain the variation
- If evidence is insufficient, note what is unknown and what should be validated
- Include "Areas for Validation / Questions" subsection where appropriate
- Minimum {word_count} words
- Use Markdown formatting: ## for section title, ### for subsections, tables where structured data fits

Output the section content in Markdown. Do not include the report title or other sections.
```

## Implementation Phases

### Phase R1: Foundation (report model + basic generation)
- SurrealDB migration for `ca_report`
- Go report store + REST endpoints
- Basic report wizard UI (type select, repo select, generate)
- Single report type: Architecture Baseline with 5 core sections
- Markdown output only
- Uses existing cliff notes + understanding scores as data

### Phase R2: Full Architecture Baseline
- All Architecture Baseline sections
- New Python analyzers: dependency audit, auth detection, test detection, git analysis
- OWASP scanner integration
- Multi-repo cross-analysis
- PDF output via Puppeteer/wkhtmltopdf

### Phase R3: Additional Report Types + Word Output
- SWOT Analysis
- Environment Evaluation
- Portfolio Health Dashboard
- Word (.docx) output
- Diagram placeholders

### Phase R4: Advanced Report Types + Polish
- Technical Due Diligence
- Compliance Gap Analysis
- Section-level regeneration (re-run one section without redoing the whole report)
- Report templates (save section selections as reusable templates)
- Report scheduling (auto-generate weekly/monthly)

## Success Criteria

1. An operator who has never used SourceBridge can generate a professional Architecture Baseline report for 3 repositories in under 5 minutes of wall-clock configuration time.
2. The generated PDF is suitable for direct client delivery without manual formatting.
3. Multi-repo reports read as a unified narrative, not disconnected per-repo sections.
4. Every claim in the report cites specific evidence (file paths, patterns, metrics).
5. Sections with insufficient data produce useful placeholders rather than fabricated content.
6. The report engine leverages existing SourceBridge analysis (cliff notes, understanding scores) rather than re-analyzing from scratch.

## Open Questions

1. **OWASP scanning scope**: Should the OWASP scanner run as a Python analyzer within the worker, or as a separate tool? The sample report shows detailed per-file findings which suggests deep static analysis.

2. **PDF rendering**: Puppeteer (headless Chrome) produces excellent PDFs but adds ~400MB to the Docker image. Alternatives: wkhtmltopdf (lighter but lower quality), WeasyPrint (Python, good quality), or a dedicated PDF microservice.

3. **Report storage**: Should full report content (Markdown, PDF, DOCX) be stored in SurrealDB, or on disk/S3? Large reports could be several MB.

4. **Concurrent report generation**: Should reports use the existing LLM orchestrator queue, or have their own? Reports make many LLM calls (one per section) and could starve artifact generation.

5. **Incremental updates**: When a repo is re-indexed, should existing reports be marked stale? Should they auto-regenerate?

## References

- Sample Architecture Baseline: MACU engagement (Hoegg Software, 2026)
- OWASP Top 10:2025: https://owasp.org/Top10/
- NIST Cybersecurity Framework: https://www.nist.gov/cyberframework
- FERPA requirements: https://studentprivacy.ed.gov/
- python-docx: https://python-docx.readthedocs.io/
- Puppeteer PDF: https://pptr.dev/api/class-page#pagepdf
