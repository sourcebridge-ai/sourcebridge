// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package knowledge

import "testing"

func TestMarkAllStale(t *testing.T) {
	s := NewMemStore()

	// Create two ready artifacts and one pending artifact for the same repo.
	a1, _ := s.StoreKnowledgeArtifact(&Artifact{
		RepositoryID: "repo-1",
		Type:         ArtifactCliffNotes,
		Audience:     AudienceDeveloper,
		Depth:        DepthMedium,
		Status:       StatusPending,
	})
	_ = s.UpdateKnowledgeArtifactStatus(a1.ID, StatusReady)

	a2, _ := s.StoreKnowledgeArtifact(&Artifact{
		RepositoryID: "repo-1",
		Type:         ArtifactLearningPath,
		Audience:     AudienceBeginner,
		Depth:        DepthSummary,
		Status:       StatusPending,
	})
	_ = s.UpdateKnowledgeArtifactStatus(a2.ID, StatusReady)

	pendingArtifact, _ := s.StoreKnowledgeArtifact(&Artifact{
		RepositoryID: "repo-1",
		Type:         ArtifactCodeTour,
		Audience:     AudienceDeveloper,
		Depth:        DepthDeep,
		Status:       StatusPending,
	})

	// Artifact for a different repo — should not be affected.
	otherRepo, _ := s.StoreKnowledgeArtifact(&Artifact{
		RepositoryID: "repo-2",
		Type:         ArtifactCliffNotes,
		Audience:     AudienceDeveloper,
		Depth:        DepthMedium,
		Status:       StatusPending,
	})
	_ = s.UpdateKnowledgeArtifactStatus(otherRepo.ID, StatusReady)

	MarkAllStale(s, "repo-1")

	// Both ready artifacts should now be stale.
	if a := s.GetKnowledgeArtifact(a1.ID); !a.Stale {
		t.Fatalf("expected a1 to be stale")
	}
	if a := s.GetKnowledgeArtifact(a2.ID); !a.Stale {
		t.Fatalf("expected a2 to be stale")
	}

	// Pending artifact should NOT be marked stale.
	if a := s.GetKnowledgeArtifact(pendingArtifact.ID); a.Stale {
		t.Fatalf("expected pending artifact to remain non-stale")
	}

	// Other repo should NOT be affected.
	if a := s.GetKnowledgeArtifact(otherRepo.ID); a.Stale {
		t.Fatalf("expected other-repo artifact to remain non-stale")
	}
}

func TestMarkAllStaleNilStore(t *testing.T) {
	// Should not panic.
	MarkAllStale(nil, "repo-1")
}

func TestMarkAllStaleIdempotent(t *testing.T) {
	s := NewMemStore()

	a, _ := s.StoreKnowledgeArtifact(&Artifact{
		RepositoryID: "repo-1",
		Type:         ArtifactCliffNotes,
		Audience:     AudienceDeveloper,
		Depth:        DepthMedium,
		Status:       StatusPending,
	})
	_ = s.UpdateKnowledgeArtifactStatus(a.ID, StatusReady)

	MarkAllStale(s, "repo-1")
	MarkAllStale(s, "repo-1") // second call — already stale, should be no-op

	if fetched := s.GetKnowledgeArtifact(a.ID); !fetched.Stale {
		t.Fatalf("expected artifact to still be stale after double mark")
	}
}

func TestRefreshClearsStale(t *testing.T) {
	s := NewMemStore()

	a, _ := s.StoreKnowledgeArtifact(&Artifact{
		RepositoryID: "repo-1",
		Type:         ArtifactCliffNotes,
		Audience:     AudienceDeveloper,
		Depth:        DepthMedium,
		Status:       StatusPending,
	})
	_ = s.UpdateKnowledgeArtifactStatus(a.ID, StatusReady)
	_ = s.MarkKnowledgeArtifactStale(a.ID, true)

	if fetched := s.GetKnowledgeArtifact(a.ID); !fetched.Stale {
		t.Fatalf("expected artifact to be stale")
	}

	// Clearing stale flag simulates what happens after a successful refresh.
	_ = s.MarkKnowledgeArtifactStale(a.ID, false)

	if fetched := s.GetKnowledgeArtifact(a.ID); fetched.Stale {
		t.Fatalf("expected artifact to no longer be stale after refresh")
	}
}

func TestMarkAllStaleMarksRepositoryUnderstandingNeedsRefresh(t *testing.T) {
	s := NewMemStore()

	_, err := s.StoreRepositoryUnderstanding(&RepositoryUnderstanding{
		RepositoryID: "repo-1",
		Scope:        (&ArtifactScope{ScopeType: ScopeRepository}).NormalizePtr(),
		RevisionFP:   "rev-1",
		Stage:        UnderstandingReady,
		TreeStatus:   UnderstandingTreeComplete,
		CachedNodes:  10,
		TotalNodes:   10,
	})
	if err != nil {
		t.Fatalf("StoreRepositoryUnderstanding: %v", err)
	}

	MarkAllStale(s, "repo-1")

	u := s.GetRepositoryUnderstanding("repo-1", ArtifactScope{ScopeType: ScopeRepository})
	if u == nil {
		t.Fatal("expected repository understanding to remain present")
	}
	if u.Stage != UnderstandingNeedsRefresh {
		t.Fatalf("expected repository understanding to be marked needs_refresh, got %q", u.Stage)
	}
}
