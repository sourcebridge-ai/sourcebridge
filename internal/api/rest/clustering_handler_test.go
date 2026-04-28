// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sourcebridge/sourcebridge/internal/clustering"
	"github.com/sourcebridge/sourcebridge/internal/config"
	graphstore "github.com/sourcebridge/sourcebridge/internal/graph"
)

// fakeClusterStore embeds *graph.Store (satisfying GraphStore) and adds
// ClusterStore methods backed by in-memory state. This avoids SurrealDB in
// unit tests while exercising the real handler logic.
type fakeClusterStore struct {
	*graphstore.Store
	clusters map[string][]clustering.Cluster // repoID → clusters
}

func newFakeClusterStore() *fakeClusterStore {
	return &fakeClusterStore{
		Store:    graphstore.NewStore(),
		clusters: make(map[string][]clustering.Cluster),
	}
}

func (f *fakeClusterStore) GetClusters(_ context.Context, repoID string) ([]clustering.Cluster, error) {
	return f.clusters[repoID], nil
}

func (f *fakeClusterStore) GetClusterByID(_ context.Context, clusterID string) (*clustering.Cluster, error) {
	for _, cs := range f.clusters {
		for _, c := range cs {
			if c.ID == clusterID {
				cp := c
				return &cp, nil
			}
		}
	}
	return nil, nil
}

func (f *fakeClusterStore) GetClusterForSymbol(_ context.Context, repoID, symbolID string) (*clustering.Cluster, error) {
	for _, c := range f.clusters[repoID] {
		for _, m := range c.Members {
			if m.SymbolID == symbolID {
				cp := c
				return &cp, nil
			}
		}
	}
	return nil, nil
}

func (f *fakeClusterStore) SaveClusters(_ context.Context, repoID string, clusters []clustering.Cluster) error {
	f.clusters[repoID] = append(f.clusters[repoID], clusters...)
	return nil
}

func (f *fakeClusterStore) ReplaceClusters(_ context.Context, repoID string, clusters []clustering.Cluster) error {
	f.clusters[repoID] = clusters
	return nil
}

func (f *fakeClusterStore) DeleteClusters(_ context.Context, repoID string) error {
	delete(f.clusters, repoID)
	return nil
}

func (f *fakeClusterStore) SetClusterLLMLabel(_ context.Context, clusterID, label string) error {
	for repoID, cs := range f.clusters {
		for i, c := range cs {
			if c.ID == clusterID {
				f.clusters[repoID][i].LLMLabel = &label
				return nil
			}
		}
	}
	return clustering.ErrClusterNotFound
}

func (f *fakeClusterStore) GetRepoEdgeHash(_ context.Context, repoID string) (string, error) {
	return "", nil
}

func (f *fakeClusterStore) SetRepoEdgeHash(_ context.Context, repoID, hash string) error {
	return nil
}

// TestHandleListClusters_PackagesAndWarnings verifies that the REST handler
// computes and returns packages and warnings server-side using the symbol
// metadata and call edges available in the store.
func TestHandleListClusters_PackagesAndWarnings(t *testing.T) {
	store := newFakeClusterStore()
	const repoID = "repo-abc"

	// Inject two symbols in different files/packages.
	// auth/token.go → package "auth"
	// api/handler.go → package "api"
	store.InjectSymbolForTest(repoID, &graphstore.StoredSymbol{
		ID:            "sym-token",
		Name:          "TokenStore.Rotate",
		QualifiedName: "TokenStore.Rotate",
		FilePath:      "internal/auth/token.go",
	})
	store.InjectSymbolForTest(repoID, &graphstore.StoredSymbol{
		ID:            "sym-session",
		Name:          "Session.Validate",
		QualifiedName: "Session.Validate",
		FilePath:      "internal/auth/session.go",
	})
	store.InjectSymbolForTest(repoID, &graphstore.StoredSymbol{
		ID:            "sym-api",
		Name:          "API.Handle",
		QualifiedName: "API.Handle",
		FilePath:      "api/handler.go",
	})
	store.InjectSymbolForTest(repoID, &graphstore.StoredSymbol{
		ID:            "sym-worker",
		Name:          "Worker.Run",
		QualifiedName: "Worker.Run",
		FilePath:      "worker/job.go",
	})
	store.InjectSymbolForTest(repoID, &graphstore.StoredSymbol{
		ID:            "sym-middleware",
		Name:          "Middleware.Wrap",
		QualifiedName: "Middleware.Wrap",
		FilePath:      "internal/middleware/wrap.go",
	})

	// Inject call edges:
	// sym-api → sym-token (from api package to auth cluster)
	// sym-worker → sym-token (from worker package to auth cluster)
	// sym-middleware → sym-token (from middleware package to auth cluster)
	// → TokenStore.Rotate has 3 distinct caller packages (api, worker, middleware)
	// → should get cross-package-callers warning
	//
	// sym-api → sym-session (one caller)
	// → Session.Validate has 1 caller; hot-path if highest in cluster
	store.InjectCallEdgesForTest(repoID, []graphstore.CallEdge{
		{CallerID: "sym-api", CalleeID: "sym-token"},
		{CallerID: "sym-worker", CalleeID: "sym-token"},
		{CallerID: "sym-middleware", CalleeID: "sym-token"},
		{CallerID: "sym-api", CalleeID: "sym-session"},
	})

	// Create the auth cluster with members.
	now := time.Now()
	authCluster := clustering.Cluster{
		ID:     "cluster:auth",
		RepoID: repoID,
		Label:  "auth",
		Size:   2,
		Members: []clustering.ClusterMember{
			{ClusterID: "cluster:auth", SymbolID: "sym-token", RepoID: repoID},
			{ClusterID: "cluster:auth", SymbolID: "sym-session", RepoID: repoID},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	_ = store.ReplaceClusters(context.Background(), repoID, []clustering.Cluster{authCluster})

	// Set up the server with the OSS edition (subsystem_clustering is registered
	// globally in registry_data.go for OSS + Enterprise).
	cfg := &config.Config{Edition: "oss"}
	srv := &Server{cfg: cfg, store: store}

	// Wire up chi routing so URL params work.
	r := chi.NewRouter()
	r.Get("/api/v1/repositories/{repo_id}/clusters", srv.handleListClusters)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/repositories/"+repoID+"/clusters", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp clustersResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if len(resp.Clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(resp.Clusters))
	}

	c := resp.Clusters[0]
	if c.Label != "auth" {
		t.Errorf("expected label=auth, got %q", c.Label)
	}

	// Packages should include "auth" derived from internal/auth/*.go files.
	if len(c.Packages) == 0 {
		t.Error("expected packages to be non-empty")
	}
	containsAuth := false
	for _, p := range c.Packages {
		if p == "auth" {
			containsAuth = true
		}
	}
	if !containsAuth {
		t.Errorf("expected packages to include 'auth'; got %v", c.Packages)
	}

	// Warnings should include a cross-package-callers warning for TokenStore.Rotate
	// (called from 3 distinct packages: api, worker, middleware).
	if len(c.Warnings) == 0 {
		t.Error("expected non-empty warnings")
	}
	var hasCrossPackage bool
	for _, w := range c.Warnings {
		if w.Kind == "cross-package-callers" && strings.Contains(w.Detail, "TokenStore.Rotate") {
			hasCrossPackage = true
		}
	}
	if !hasCrossPackage {
		t.Errorf("expected cross-package-callers warning for TokenStore.Rotate; got %v", c.Warnings)
	}
}
