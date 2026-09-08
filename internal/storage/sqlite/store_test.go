package sqlite

import (
	"context"
	"database/sql"
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

	analysis := domain.Analysis{ID: "a1", CreatedAt: time.Now().UTC(), RequestCount: 1, EndpointCount: 1, SessionCount: 1, DependencyCount: 1}
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
	}}, []domain.EndpointSummary{endpoint}, []domain.SessionArtifact{artifact}, []domain.Dependency{{ID: "d1", AnalysisID: "a1", SourceRequestID: "r1", TargetRequestID: "r2", SourcePath: "$.id", TargetLocation: "path", TargetPath: "/api/items/{value}", Confidence: domain.DependencyConfidenceHigh, Reason: "response_json_to_path"}})
	if err != nil {
		t.Fatal(err)
	}

	got, err := store.GetAnalysis(ctx, "a1")
	if err != nil {
		t.Fatal(err)
	}
	if got.RequestCount != 1 || got.EndpointCount != 1 || got.SessionCount != 1 || got.DependencyCount != 1 {
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
	dependencies, err := store.ListDependencies(ctx, "a1")
	if err != nil {
		t.Fatal(err)
	}
	if len(dependencies) != 1 || dependencies[0].SourcePath != "$.id" || dependencies[0].TargetPath != "/api/items/{value}" {
		t.Fatalf("dependencies = %#v", dependencies)
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

func TestOpenUpgradesT2DatabaseWithDependenciesMigration(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/t2.db"
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
CREATE TABLE schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE analyses (
	id TEXT PRIMARY KEY,
	created_at TEXT NOT NULL,
	request_count INTEGER NOT NULL,
	endpoint_count INTEGER NOT NULL,
	session_artifact_count INTEGER NOT NULL DEFAULT 0,
	import_duration_ms INTEGER NOT NULL
);
INSERT INTO schema_migrations (version) VALUES ('001_initial.sql'), ('002_session_artifacts.sql');
INSERT INTO analyses (id, created_at, request_count, endpoint_count, session_artifact_count, import_duration_ms)
VALUES ('a1', '2026-01-02T03:04:05Z', 1, 1, 0, 0);
`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	analysis, err := store.GetAnalysis(ctx, "a1")
	if err != nil {
		t.Fatal(err)
	}
	if analysis.DependencyCount != 0 {
		t.Fatalf("dependency count = %d", analysis.DependencyCount)
	}
}

func TestSaveAnalysisRollsBackWhenDependencyPersistenceFails(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, t.TempDir()+"/rollback.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	createdAt := time.Now().UTC()
	analysis := domain.Analysis{ID: "rollback", CreatedAt: createdAt, RequestCount: 1, EndpointCount: 1, SessionCount: 1, DependencyCount: 2}
	exchanges := []domain.Exchange{{Request: domain.Request{ID: "r1", Method: "GET", URL: "https://example.com/api"}}}
	endpoints := []domain.EndpointSummary{{Method: "GET", Host: "example.com", Path: "/api", RequestCount: 1}}
	artifacts := []domain.SessionArtifact{{ID: "s1", AnalysisID: analysis.ID, Type: domain.SessionArtifactCookie, Name: "session_id", Source: "request_cookie", FirstRequestID: "r1", FirstSeenAt: createdAt, Occurrences: 1}}
	dependencies := []domain.Dependency{
		{ID: "d1", AnalysisID: analysis.ID, SourceRequestID: "r1", TargetRequestID: "r2", SourcePath: "$.id", TargetLocation: "path", TargetPath: "/api/items/{value}", Confidence: domain.DependencyConfidenceHigh, Reason: "response_json_to_path"},
		{ID: "d2", AnalysisID: analysis.ID, SourceRequestID: "r1", TargetRequestID: "r2", SourcePath: "$.id", TargetLocation: "path", TargetPath: "/api/items/{value}", Confidence: domain.DependencyConfidenceHigh, Reason: "response_json_to_path"},
	}

	if err := store.SaveAnalysis(ctx, analysis, exchanges, endpoints, artifacts, dependencies); err == nil {
		t.Fatal("expected duplicate dependency evidence to fail")
	}
	for _, table := range []string{"analyses", "exchanges", "endpoints", "session_artifacts", "dependencies"} {
		if got := countRows(t, store, table); got != 0 {
			t.Fatalf("%s rows = %d", table, got)
		}
	}
}

func countRows(t *testing.T, store *Store, table string) int {
	t.Helper()
	var count int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
