package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"protocollens/internal/app"
	"protocollens/internal/capture/har"
	"protocollens/internal/storage/sqlite"
)

func TestImportHARFlow(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, t.TempDir()+"/api.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	importAnalysis := app.NewImportAnalysis(har.NewImporter(1024*1024), store)
	mux := http.NewServeMux()
	registerRoutes(mux, store, importAnalysis)

	body := strings.NewReader(`{"log":{"entries":[{
		"startedDateTime":"2026-01-02T03:04:05Z",
		"time":15,
		"request":{"method":"GET","url":"https://example.com/api/items?page=1","headers":[]},
		"response":{"status":200,"headers":[{"name":"Content-Type","value":"application/json"}]}
	}]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/import/har", body)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestEndpointsMissingAnalysisReturnsNotFound(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, t.TempDir()+"/api.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	importAnalysis := app.NewImportAnalysis(har.NewImporter(1024*1024), store)
	mux := http.NewServeMux()
	registerRoutes(mux, store, importAnalysis)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analyses/missing/endpoints", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestSessionsEndpointReturnsArtifacts(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, t.TempDir()+"/api.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	importAnalysis := app.NewImportAnalysis(har.NewImporter(1024*1024), store)
	mux := http.NewServeMux()
	registerRoutes(mux, store, importAnalysis)

	body := strings.NewReader(`{"log":{"entries":[{
		"startedDateTime":"2026-01-02T03:04:05Z",
		"time":15,
		"request":{"method":"GET","url":"https://example.com/api/items","headers":[
			{"name":"Authorization","value":"Bearer test-token-value"},
			{"name":"Cookie","value":"session_id=fake-session-value"}
		]},
		"response":{"status":200,"headers":[
			{"name":"Set-Cookie","value":"session_id=fake-session-value; Path=/; HttpOnly; Secure"}
		]}
	}]}}`)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/import/har", body))
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var analysis analysisResponse
	if err := json.Unmarshal(w.Body.Bytes(), &analysis); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/analyses/"+analysis.ID+"/sessions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "test-token-value") || strings.Contains(w.Body.String(), "fake-session-value") {
		t.Fatalf("response leaked secret: %s", w.Body.String())
	}

	var response sessionArtifactsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Artifacts) != 3 {
		t.Fatalf("artifacts = %#v", response.Artifacts)
	}
}

func TestSessionsEndpointMissingAnalysisReturnsNotFound(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, t.TempDir()+"/api.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	importAnalysis := app.NewImportAnalysis(har.NewImporter(1024*1024), store)
	mux := http.NewServeMux()
	registerRoutes(mux, store, importAnalysis)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/analyses/missing/sessions", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestSessionsEndpointExistingAnalysisWithNoArtifactsReturnsEmptyList(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, t.TempDir()+"/api.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	importAnalysis := app.NewImportAnalysis(har.NewImporter(1024*1024), store)
	mux := http.NewServeMux()
	registerRoutes(mux, store, importAnalysis)

	body := strings.NewReader(`{"log":{"entries":[{
		"startedDateTime":"2026-01-02T03:04:05Z",
		"time":15,
		"request":{"method":"GET","url":"https://example.com/api/items","headers":[]},
		"response":{"status":200,"headers":[]}
	}]}}`)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/import/har", body))
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var analysis analysisResponse
	if err := json.Unmarshal(w.Body.Bytes(), &analysis); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/analyses/"+analysis.ID+"/sessions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var response sessionArtifactsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Artifacts) != 0 {
		t.Fatalf("artifacts = %#v", response.Artifacts)
	}
}
