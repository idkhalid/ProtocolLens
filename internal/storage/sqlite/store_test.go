package sqlite

import (
	"context"
	"testing"
	"time"

	"protocollens/internal/domain"
)

func TestStorePersistsAnalysisAndEndpoints(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, t.TempDir()+"/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	analysis := domain.Analysis{ID: "a1", CreatedAt: time.Now().UTC(), RequestCount: 1, EndpointCount: 1, SessionCount: 1}
	endpoint := domain.EndpointSummary{
		Method:            "GET",
		Host:              "example.com",
		Path:              "/api",
		RequestCount:      1,
		StatusCodes:       []int{200},
		AverageDurationMs: 12,
		ContentTypes:      []string{"application/json"},
	}

	artifact := domain.SessionArtifact{
		ID:             "s1",
		AnalysisID:     "a1",
		Type:           domain.SessionArtifactCookie,
		Name:           "session_id",
		Source:         "response_cookie",
		FirstRequestID: "r1",
		FirstSeenAt:    analysis.CreatedAt,
		Occurrences:    1,
		Metadata:       map[string]string{"HttpOnly": "true"},
	}

	err = store.SaveAnalysis(ctx, analysis, []domain.Exchange{{
		Request: domain.Request{ID: "r1", Method: "GET", URL: "https://example.com/api"},
	}}, []domain.EndpointSummary{endpoint}, []domain.SessionArtifact{artifact})
	if err != nil {
		t.Fatal(err)
	}

	got, err := store.GetAnalysis(ctx, "a1")
	if err != nil {
		t.Fatal(err)
	}
	if got.RequestCount != 1 || got.EndpointCount != 1 || got.SessionCount != 1 {
		t.Fatalf("analysis = %#v", got)
	}

	endpoints, err := store.ListEndpoints(ctx, "a1")
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 1 || endpoints[0].Path != "/api" {
		t.Fatalf("endpoints = %#v", endpoints)
	}

	artifacts, err := store.ListSessionArtifacts(ctx, "a1")
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 || artifacts[0].Name != "session_id" || artifacts[0].Metadata["HttpOnly"] != "true" {
		t.Fatalf("artifacts = %#v", artifacts)
	}
}

func TestOpenAppliesMigrationsOnce(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/migrate.db"

	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	store, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
}
