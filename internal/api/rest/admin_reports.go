// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	"github.com/sourcebridge/sourcebridge/internal/reports"
)

// --- Report CRUD ---

func (s *Server) handleListReports(w http.ResponseWriter, r *http.Request) {
	if s.reportStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reports not configured"})
		return
	}
	list, err := s.reportStore.ListReports(50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleGetReport(w http.ResponseWriter, r *http.Request) {
	if s.reportStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reports not configured"})
		return
	}
	id := chi.URLParam(r, "id")
	report, err := s.reportStore.GetReport(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if report == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "report not found"})
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleCreateReport(w http.ResponseWriter, r *http.Request) {
	if s.reportStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reports not configured"})
		return
	}
	var report reports.Report
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		return
	}
	if report.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if len(report.RepositoryIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "at least one repository is required"})
		return
	}
	report.Status = reports.StatusPending
	userID, _ := currentActorIdentity(r)
	report.CreatedBy = userID

	created, err := s.reportStore.CreateReport(&report)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// TODO: Enqueue report generation via the Python worker.
	// For now, the report is created in pending status and the
	// frontend polls for status changes.

	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleDeleteReport(w http.ResponseWriter, r *http.Request) {
	if s.reportStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reports not configured"})
		return
	}
	id := chi.URLParam(r, "id")
	_ = s.reportStore.DeleteEvidence(id)
	if err := s.reportStore.DeleteReport(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Report content download ---

func (s *Server) handleDownloadReportFile(w http.ResponseWriter, r *http.Request) {
	if s.reportStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reports not configured"})
		return
	}
	id := chi.URLParam(r, "id")
	format := chi.URLParam(r, "format") // "markdown", "pdf", "docx"
	report, err := s.reportStore.GetReport(id)
	if err != nil || report == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "report not found"})
		return
	}
	if report.ContentDir == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "report content not yet generated"})
		return
	}

	var filename string
	var contentType string
	switch format {
	case "markdown", "md":
		filename = "report.md"
		contentType = "text/markdown"
	case "pdf":
		filename = "report.pdf"
		contentType = "application/pdf"
	case "docx":
		filename = "report.docx"
		contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("unknown format: %s", format)})
		return
	}

	path := filepath.Join(report.ContentDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found: " + filename})
		return
	}

	safeName := report.Name
	if safeName == "" {
		safeName = "report"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, safeName, format))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// --- Report types metadata ---

func (s *Server) handleListReportTypes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, reports.AllReportTypes())
}

func (s *Server) handleGetDefaultSections(w http.ResponseWriter, r *http.Request) {
	reportType := reports.ReportType(r.URL.Query().Get("report_type"))
	audience := reports.Audience(r.URL.Query().Get("audience"))
	if reportType == "" {
		reportType = reports.TypeArchitectureBaseline
	}
	if audience == "" {
		audience = reports.AudienceTechnicalLead
	}
	sections := reports.DefaultSectionsForAudience(reportType, audience)
	allSections := reports.SectionsForType(reportType)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"reportType":      reportType,
		"audience":        audience,
		"defaultSections": sections,
		"allSections":     allSections,
	})
}

// --- Templates ---

func (s *Server) handleListReportTemplates(w http.ResponseWriter, r *http.Request) {
	if s.reportStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reports not configured"})
		return
	}
	list, err := s.reportStore.ListTemplates()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateReportTemplate(w http.ResponseWriter, r *http.Request) {
	if s.reportStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reports not configured"})
		return
	}
	var tpl reports.ReportTemplate
	if err := json.NewDecoder(r.Body).Decode(&tpl); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		return
	}
	userID, _ := currentActorIdentity(r)
	tpl.CreatedBy = userID
	created, err := s.reportStore.CreateTemplate(&tpl)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleDeleteReportTemplate(w http.ResponseWriter, r *http.Request) {
	if s.reportStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reports not configured"})
		return
	}
	id := chi.URLParam(r, "id")
	if err := s.reportStore.DeleteTemplate(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Evidence ---

func (s *Server) handleGetReportEvidence(w http.ResponseWriter, r *http.Request) {
	if s.reportStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reports not configured"})
		return
	}
	id := chi.URLParam(r, "id")
	items, err := s.reportStore.GetEvidence(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// --- Report regeneration ---

func (s *Server) handleRegenerateReport(w http.ResponseWriter, r *http.Request) {
	if s.reportStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reports not configured"})
		return
	}
	id := chi.URLParam(r, "id")
	report, err := s.reportStore.GetReport(id)
	if err != nil || report == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "report not found"})
		return
	}

	// Increment version and reset status
	report.Version++
	report.Status = reports.StatusPending
	report.Progress = 0
	report.ProgressPhase = ""
	report.ProgressMessage = ""
	report.ErrorCode = ""
	report.ErrorMessage = ""
	report.Stale = false

	if err := s.reportStore.UpdateReportStatus(id, reports.StatusPending, 0, "", "Queued for regeneration"); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// TODO: Enqueue regeneration job
	writeJSON(w, http.StatusOK, report)
}

// --- Report Markdown content (for preview) ---

func (s *Server) handleGetReportMarkdown(w http.ResponseWriter, r *http.Request) {
	if s.reportStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reports not configured"})
		return
	}
	id := chi.URLParam(r, "id")
	report, err := s.reportStore.GetReport(id)
	if err != nil || report == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "report not found"})
		return
	}
	if report.ContentDir == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "content not yet generated"})
		return
	}
	path := filepath.Join(report.ContentDir, "report.md")
	data, err := os.ReadFile(path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "markdown file not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"markdown": string(data)})
}
