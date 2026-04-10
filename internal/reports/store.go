// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package reports

// Store is the persistence interface for reports, templates, and branding.
type Store interface {
	// Reports
	CreateReport(r *Report) (*Report, error)
	GetReport(id string) (*Report, error)
	ListReports(limit int) ([]Report, error)
	UpdateReportStatus(id string, status ReportStatus, progress float64, phase, message string) error
	UpdateReportCompleted(id string, sectionCount, wordCount, evidenceCount int, contentDir string) error
	SetReportFailed(id string, code, message string) error
	DeleteReport(id string) error
	MarkReportsStale(repoID string) error

	// Templates
	CreateTemplate(t *ReportTemplate) (*ReportTemplate, error)
	GetTemplate(id string) (*ReportTemplate, error)
	ListTemplates() ([]ReportTemplate, error)
	DeleteTemplate(id string) error

	// Branding
	CreateBranding(b *ReportBranding) (*ReportBranding, error)
	GetBranding(id string) (*ReportBranding, error)
	ListBrandings() ([]ReportBranding, error)
	UpdateBranding(b *ReportBranding) error
	DeleteBranding(id string) error

	// Evidence
	StoreEvidence(items []ReportEvidence) error
	GetEvidence(reportID string) ([]ReportEvidence, error)
	DeleteEvidence(reportID string) error
}
