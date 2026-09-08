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

	analysis := domain.Analysis{ID: "a1", CreatedAt: time.Now().UTC(), RequestCount: 1, EndpointCount: 1}
	endpoint := domain.EndpointSummary{
		Method:            "GET",
		Host:              "example.com",
		Path:              "/api",
		RequestCount:      1,
		StatusCodes:       []int{200},
		AverageDurationMs: 12,
		ContentTypes:      []string{"application/json"},
	}

	err = store.SaveAnalysis(ctx, analysis, []domain.Exchange{{
		Request: domain.Request{ID: "r1", Method: "GET", URL: "https://example.com/api"},
	}}, []domain.EndpointSummary{endpoint})
	if err != nil {
		t.Fatal(err)
	}

	got, err := store.GetAnalysis(ctx, "a1")
	if err != nil {
		t.Fatal(err)
	}
	if got.RequestCount != 1 || got.EndpointCount != 1 {
		t.Fatalf("analysis = %#v", got)
	}

	endpoints, err := store.ListEndpoints(ctx, "a1")
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 1 || endpoints[0].Path != "/api" {
		t.Fatalf("endpoints = %#v", endpoints)
	}
}
