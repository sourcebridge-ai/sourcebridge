// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package reports

import "time"

// ReportType identifies the kind of report.
type ReportType string

const (
	TypeArchitectureBaseline ReportType = "architecture_baseline"
	TypeSWOT                 ReportType = "swot"
	TypeEnvironmentEval      ReportType = "environment_eval"
	TypeDueDiligence         ReportType = "due_diligence"
	TypeComplianceGap        ReportType = "compliance_gap"
	TypePortfolioHealth      ReportType = "portfolio_health"
)

// Audience controls language, depth, and framing.
type Audience string

const (
	AudienceCSuite           Audience = "c_suite"
	AudienceExecutive        Audience = "executive"
	AudienceTechnicalLead    Audience = "technical_leadership"
	AudienceDeveloper        Audience = "developer"
	AudienceCompliance       Audience = "compliance"
	AudienceNonTechnical     Audience = "non_technical"
)

// LOEMode controls how effort estimates are framed.
type LOEMode string

const (
	LOEHumanHours LOEMode = "human_hours"
	LOEAIAssisted LOEMode = "ai_assisted"
)

// ReportStatus tracks the report lifecycle.
type ReportStatus string

const (
	StatusPending    ReportStatus = "pending"
	StatusCollecting ReportStatus = "collecting"
	StatusGenerating ReportStatus = "generating"
	StatusRendering  ReportStatus = "rendering"
	StatusReady      ReportStatus = "ready"
	StatusFailed     ReportStatus = "failed"
)

// Report is the core domain object.
type Report struct {
	ID               string       `json:"id,omitempty"`
	Name             string       `json:"name"`
	ReportType       ReportType   `json:"reportType"`
	Audience         Audience     `json:"audience"`
	RepositoryIDs    []string     `json:"repositoryIds"`
	SelectedSections []string     `json:"selectedSections"`
	IncludeDiagrams  bool         `json:"includeDiagrams"`
	OutputFormats    []string     `json:"outputFormats"`
	LOEMode          LOEMode      `json:"loeMode"`
	BrandingID       string       `json:"brandingId,omitempty"`
	TemplateID       string       `json:"templateId,omitempty"`
	Status           ReportStatus `json:"status"`
	Progress         float64      `json:"progress"`
	ProgressPhase    string       `json:"progressPhase,omitempty"`
	ProgressMessage  string       `json:"progressMessage,omitempty"`
	ErrorCode        string       `json:"errorCode,omitempty"`
	ErrorMessage     string       `json:"errorMessage,omitempty"`
	Version          int          `json:"version"`
	ContentDir       string       `json:"contentDir,omitempty"`
	SectionCount     int          `json:"sectionCount"`
	WordCount        int          `json:"wordCount"`
	EvidenceCount    int          `json:"evidenceCount"`
	Stale            bool         `json:"stale"`
	CreatedBy        string       `json:"createdBy,omitempty"`
	CreatedAt        time.Time    `json:"createdAt"`
	UpdatedAt        time.Time    `json:"updatedAt"`
	CompletedAt      *time.Time   `json:"completedAt,omitempty"`
}

// ReportTemplate is a saved wizard configuration for one-click regeneration.
type ReportTemplate struct {
	ID               string     `json:"id,omitempty"`
	Name             string     `json:"name"`
	ReportType       ReportType `json:"reportType"`
	Audience         Audience   `json:"audience"`
	RepositoryIDs    []string   `json:"repositoryIds"`
	SelectedSections []string   `json:"selectedSections"`
	IncludeDiagrams  bool       `json:"includeDiagrams"`
	OutputFormats    []string   `json:"outputFormats"`
	LOEMode          LOEMode    `json:"loeMode"`
	BrandingID       string     `json:"brandingId,omitempty"`
	CreatedBy        string     `json:"createdBy,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// ReportBranding holds visual customization for rendered reports.
type ReportBranding struct {
	ID            string    `json:"id,omitempty"`
	Name          string    `json:"name"`
	LogoPath      string    `json:"logoPath,omitempty"`
	PrimaryColor  string    `json:"primaryColor"`
	AccentColor   string    `json:"accentColor"`
	CoverTitle    string    `json:"coverTitle,omitempty"`
	CoverSubtitle string    `json:"coverSubtitle,omitempty"`
	FooterText    string    `json:"footerText,omitempty"`
	CreatedBy     string    `json:"createdBy,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// ReportEvidence is a single piece of evidence backing a report claim.
type ReportEvidence struct {
	ID           string    `json:"id,omitempty"`
	ReportID     string    `json:"reportId"`
	EvidenceID   string    `json:"evidenceId"`
	Category     string    `json:"category"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	SourceType   string    `json:"sourceType"`
	SourceRepoID string    `json:"sourceRepoId,omitempty"`
	FilePath     string    `json:"filePath,omitempty"`
	LineStart    int       `json:"lineStart,omitempty"`
	LineEnd      int       `json:"lineEnd,omitempty"`
	CodeSnippet  string    `json:"codeSnippet,omitempty"`
	RawData      string    `json:"rawData,omitempty"`
	Severity     string    `json:"severity"`
	CreatedAt    time.Time `json:"createdAt"`
}

// SectionDefinition describes a report section and its dependencies.
type SectionDefinition struct {
	Key          string   `json:"key"`
	Title        string   `json:"title"`
	Category     string   `json:"category"`
	Description  string   `json:"description"`
	DependsOn    []string `json:"dependsOn,omitempty"`
	DataSources  []string `json:"dataSources"`
	MinWordCount int      `json:"minWordCount"`
}

// ReportTypeDefinition describes a report type and its available sections.
type ReportTypeDefinition struct {
	Type        ReportType          `json:"type"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Sections    []SectionDefinition `json:"sections"`
}
