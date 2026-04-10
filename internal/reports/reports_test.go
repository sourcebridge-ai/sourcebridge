// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package reports

import (
	"testing"
)

func TestMemStoreReportCRUD(t *testing.T) {
	store := NewMemStore()

	// Create
	r, err := store.CreateReport(&Report{
		Name:             "Test Report",
		ReportType:       TypeArchitectureBaseline,
		Audience:         AudienceTechnicalLead,
		RepositoryIDs:    []string{"repo-1", "repo-2"},
		SelectedSections: []string{"executive_summary", "testing"},
		OutputFormats:    []string{"markdown"},
		Status:           StatusPending,
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.ID == "" {
		t.Error("expected non-empty ID")
	}

	// Get
	got, err := store.GetReport(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Test Report" {
		t.Errorf("expected name 'Test Report', got %q", got.Name)
	}

	// List
	list, err := store.ListReports(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 report, got %d", len(list))
	}

	// Update status
	err = store.UpdateReportStatus(r.ID, StatusGenerating, 0.5, "generating", "Building sections")
	if err != nil {
		t.Fatal(err)
	}
	got, _ = store.GetReport(r.ID)
	if got.Status != StatusGenerating || got.Progress != 0.5 {
		t.Errorf("expected generating/0.5, got %s/%f", got.Status, got.Progress)
	}

	// Complete
	err = store.UpdateReportCompleted(r.ID, 7, 3000, 12, "/data/reports/test/v1")
	if err != nil {
		t.Fatal(err)
	}
	got, _ = store.GetReport(r.ID)
	if got.Status != StatusReady || got.SectionCount != 7 || got.WordCount != 3000 {
		t.Errorf("expected ready/7/3000, got %s/%d/%d", got.Status, got.SectionCount, got.WordCount)
	}

	// Fail
	r2, _ := store.CreateReport(&Report{Name: "Fail test", Status: StatusPending})
	err = store.SetReportFailed(r2.ID, "LLM_ERROR", "model crashed")
	if err != nil {
		t.Fatal(err)
	}
	got, _ = store.GetReport(r2.ID)
	if got.Status != StatusFailed || got.ErrorCode != "LLM_ERROR" {
		t.Errorf("expected failed/LLM_ERROR, got %s/%s", got.Status, got.ErrorCode)
	}

	// Delete
	_ = store.DeleteReport(r.ID)
	got, _ = store.GetReport(r.ID)
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestMemStoreTemplateCRUD(t *testing.T) {
	store := NewMemStore()

	tpl, err := store.CreateTemplate(&ReportTemplate{
		Name:       "Monthly Review",
		ReportType: TypePortfolioHealth,
		Audience:   AudienceExecutive,
	})
	if err != nil {
		t.Fatal(err)
	}
	if tpl.ID == "" {
		t.Error("expected non-empty ID")
	}

	got, _ := store.GetTemplate(tpl.ID)
	if got.Name != "Monthly Review" {
		t.Errorf("expected 'Monthly Review', got %q", got.Name)
	}

	list, _ := store.ListTemplates()
	if len(list) != 1 {
		t.Errorf("expected 1 template, got %d", len(list))
	}

	_ = store.DeleteTemplate(tpl.ID)
	got, _ = store.GetTemplate(tpl.ID)
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestMemStoreBrandingCRUD(t *testing.T) {
	store := NewMemStore()

	b, err := store.CreateBranding(&ReportBranding{
		Name:         "Client Brand",
		PrimaryColor: "#ff0000",
		AccentColor:  "#00ff00",
	})
	if err != nil {
		t.Fatal(err)
	}
	if b.ID == "" {
		t.Error("expected non-empty ID")
	}

	b.PrimaryColor = "#0000ff"
	err = store.UpdateBranding(b)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := store.GetBranding(b.ID)
	if got.PrimaryColor != "#0000ff" {
		t.Errorf("expected #0000ff, got %s", got.PrimaryColor)
	}

	list, _ := store.ListBrandings()
	if len(list) != 1 {
		t.Errorf("expected 1 branding, got %d", len(list))
	}

	_ = store.DeleteBranding(b.ID)
	got, _ = store.GetBranding(b.ID)
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestMemStoreEvidence(t *testing.T) {
	store := NewMemStore()

	err := store.StoreEvidence([]ReportEvidence{
		{ReportID: "r1", EvidenceID: "E-SEC-01", Category: "security", Title: "No auth", Severity: "critical"},
		{ReportID: "r1", EvidenceID: "E-SEC-02", Category: "security", Title: "XSS", Severity: "high"},
		{ReportID: "r2", EvidenceID: "E-ENG-01", Category: "engineering", Title: "No tests", Severity: "medium"},
	})
	if err != nil {
		t.Fatal(err)
	}

	items, _ := store.GetEvidence("r1")
	if len(items) != 2 {
		t.Errorf("expected 2 evidence items for r1, got %d", len(items))
	}

	items, _ = store.GetEvidence("r2")
	if len(items) != 1 {
		t.Errorf("expected 1 evidence item for r2, got %d", len(items))
	}

	_ = store.DeleteEvidence("r1")
	items, _ = store.GetEvidence("r1")
	if len(items) != 0 {
		t.Errorf("expected 0 after delete, got %d", len(items))
	}
}

func TestSectionsForType(t *testing.T) {
	sections := SectionsForType(TypeArchitectureBaseline)
	if len(sections) < 30 {
		t.Errorf("expected 30+ arch baseline sections, got %d", len(sections))
	}

	sections = SectionsForType(TypeSWOT)
	if len(sections) != 5 {
		t.Errorf("expected 5 SWOT sections, got %d", len(sections))
	}

	sections = SectionsForType("nonexistent")
	if sections != nil {
		t.Error("expected nil for unknown type")
	}
}

func TestDefaultSectionsForAudience(t *testing.T) {
	// C-Suite should exclude Engineering and Appendix sections
	csuite := DefaultSectionsForAudience(TypeArchitectureBaseline, AudienceCSuite)
	all := DefaultSectionsForAudience(TypeArchitectureBaseline, AudienceDeveloper)

	if len(csuite) >= len(all) {
		t.Errorf("expected C-Suite to have fewer sections than Developer: c-suite=%d, dev=%d", len(csuite), len(all))
	}

	// Verify executive_summary is in both
	hasExecCSuite := false
	hasExecDev := false
	for _, k := range csuite {
		if k == "executive_summary" {
			hasExecCSuite = true
		}
	}
	for _, k := range all {
		if k == "executive_summary" {
			hasExecDev = true
		}
	}
	if !hasExecCSuite || !hasExecDev {
		t.Error("expected executive_summary in both audiences")
	}

	// Verify appendix_owasp is not in C-Suite but is in Developer
	hasAppendixCSuite := false
	hasAppendixDev := false
	for _, k := range csuite {
		if k == "appendix_owasp" {
			hasAppendixCSuite = true
		}
	}
	for _, k := range all {
		if k == "appendix_owasp" {
			hasAppendixDev = true
		}
	}
	if hasAppendixCSuite {
		t.Error("C-Suite should not have appendix_owasp")
	}
	if !hasAppendixDev {
		t.Error("Developer should have appendix_owasp")
	}
}

func TestAllReportTypes(t *testing.T) {
	types := AllReportTypes()
	if len(types) != 6 {
		t.Errorf("expected 6 report types, got %d", len(types))
	}
	for _, rt := range types {
		if rt.Title == "" {
			t.Errorf("report type %s has empty title", rt.Type)
		}
		if len(rt.Sections) == 0 {
			t.Errorf("report type %s has no sections", rt.Type)
		}
	}
}
