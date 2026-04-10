// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package reports

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemStore is an in-memory Store implementation for tests.
type MemStore struct {
	mu        sync.RWMutex
	reports   map[string]*Report
	templates map[string]*ReportTemplate
	brandings map[string]*ReportBranding
	evidence  map[string][]ReportEvidence // keyed by reportID
}

func NewMemStore() *MemStore {
	return &MemStore{
		reports:   make(map[string]*Report),
		templates: make(map[string]*ReportTemplate),
		brandings: make(map[string]*ReportBranding),
		evidence:  make(map[string][]ReportEvidence),
	}
}

// --- Reports ---

func (m *MemStore) CreateReport(r *Report) (*Report, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	r.CreatedAt = time.Now()
	r.UpdatedAt = time.Now()
	cp := *r
	m.reports[r.ID] = &cp
	return &cp, nil
}

func (m *MemStore) GetReport(id string) (*Report, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.reports[id]
	if !ok {
		return nil, nil
	}
	cp := *r
	return &cp, nil
}

func (m *MemStore) ListReports(limit int) ([]Report, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Report, 0, len(m.reports))
	for _, r := range m.reports {
		result = append(result, *r)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *MemStore) UpdateReportStatus(id string, status ReportStatus, progress float64, phase, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.reports[id]
	if !ok {
		return fmt.Errorf("report not found: %s", id)
	}
	r.Status = status
	r.Progress = progress
	r.ProgressPhase = phase
	r.ProgressMessage = message
	r.UpdatedAt = time.Now()
	return nil
}

func (m *MemStore) UpdateReportCompleted(id string, sectionCount, wordCount, evidenceCount int, contentDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.reports[id]
	if !ok {
		return fmt.Errorf("report not found: %s", id)
	}
	r.Status = StatusReady
	r.Progress = 1.0
	r.SectionCount = sectionCount
	r.WordCount = wordCount
	r.EvidenceCount = evidenceCount
	r.ContentDir = contentDir
	now := time.Now()
	r.CompletedAt = &now
	r.UpdatedAt = now
	return nil
}

func (m *MemStore) SetReportFailed(id string, code, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.reports[id]
	if !ok {
		return fmt.Errorf("report not found: %s", id)
	}
	r.Status = StatusFailed
	r.ErrorCode = code
	r.ErrorMessage = message
	r.UpdatedAt = time.Now()
	return nil
}

func (m *MemStore) DeleteReport(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.reports, id)
	delete(m.evidence, id)
	return nil
}

func (m *MemStore) MarkReportsStale(repoID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.reports {
		var repoIDs []string
		_ = json.Unmarshal([]byte(fmt.Sprintf("%v", r.RepositoryIDs)), &repoIDs)
		for _, rid := range r.RepositoryIDs {
			if rid == repoID {
				r.Stale = true
				r.UpdatedAt = time.Now()
				break
			}
		}
	}
	return nil
}

// --- Templates ---

func (m *MemStore) CreateTemplate(t *ReportTemplate) (*ReportTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	cp := *t
	m.templates[t.ID] = &cp
	return &cp, nil
}

func (m *MemStore) GetTemplate(id string) (*ReportTemplate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.templates[id]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (m *MemStore) ListTemplates() ([]ReportTemplate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]ReportTemplate, 0, len(m.templates))
	for _, t := range m.templates {
		result = append(result, *t)
	}
	return result, nil
}

func (m *MemStore) DeleteTemplate(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.templates, id)
	return nil
}

// --- Branding ---

func (m *MemStore) CreateBranding(b *ReportBranding) (*ReportBranding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	b.CreatedAt = time.Now()
	b.UpdatedAt = time.Now()
	cp := *b
	m.brandings[b.ID] = &cp
	return &cp, nil
}

func (m *MemStore) GetBranding(id string) (*ReportBranding, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.brandings[id]
	if !ok {
		return nil, nil
	}
	cp := *b
	return &cp, nil
}

func (m *MemStore) ListBrandings() ([]ReportBranding, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]ReportBranding, 0, len(m.brandings))
	for _, b := range m.brandings {
		result = append(result, *b)
	}
	return result, nil
}

func (m *MemStore) UpdateBranding(b *ReportBranding) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.brandings[b.ID]
	if !ok {
		return fmt.Errorf("branding not found: %s", b.ID)
	}
	b.CreatedAt = existing.CreatedAt
	b.UpdatedAt = time.Now()
	cp := *b
	m.brandings[b.ID] = &cp
	return nil
}

func (m *MemStore) DeleteBranding(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.brandings, id)
	return nil
}

// --- Evidence ---

func (m *MemStore) StoreEvidence(items []ReportEvidence) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, item := range items {
		if item.ID == "" {
			item.ID = uuid.New().String()
		}
		item.CreatedAt = time.Now()
		m.evidence[item.ReportID] = append(m.evidence[item.ReportID], item)
	}
	return nil
}

func (m *MemStore) GetEvidence(reportID string) ([]ReportEvidence, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.evidence[reportID]
	cp := make([]ReportEvidence, len(items))
	copy(cp, items)
	return cp, nil
}

func (m *MemStore) DeleteEvidence(reportID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.evidence, reportID)
	return nil
}
