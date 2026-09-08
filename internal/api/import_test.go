package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestDependenciesEndpointReturnsDependenciesWithoutMatchedValues(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, t.TempDir()+"/api.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	importAnalysis := app.NewImportAnalysis(har.NewImporter(1024*1024), store)
	mux := http.NewServeMux()
	registerRoutes(mux, store, importAnalysis)

	body := strings.NewReader(`{"log":{"entries":[
		{"startedDateTime":"2026-01-02T03:04:05Z","time":10,
		 "request":{"method":"GET","url":"https://example.com/api/source","headers":[]},
		 "response":{"status":200,"headers":[{"name":"Content-Type","value":"application/json"}],"content":{"mimeType":"application/json","text":"{\"id\":\"dep_secret_12345\"}"}}},
		{"startedDateTime":"2026-01-02T03:04:06Z","time":10,
		 "request":{"method":"GET","url":"https://example.com/api/items/dep_secret_12345","headers":[]},
		 "response":{"status":200,"headers":[]}}
	]}}`)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/import/har", body))
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var analysis analysisResponse
	if err := json.Unmarshal(w.Body.Bytes(), &analysis); err != nil {
		t.Fatal(err)
	}
	if analysis.DependencyCount != 1 {
		t.Fatalf("dependency count = %d", analysis.DependencyCount)
	}

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/analyses/"+analysis.ID+"/dependencies", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "dep_secret_12345") {
		t.Fatalf("response leaked matched value: %s", w.Body.String())
	}

	var response dependenciesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Dependencies) != 1 || response.Dependencies[0].TargetPath != "/api/items/{value}" {
		t.Fatalf("dependencies = %#v", response.Dependencies)
	}
}

func TestDependenciesEndpointMissingAnalysisReturnsNotFound(t *testing.T) {
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
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/analyses/missing/dependencies", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestDependenciesEndpointExistingAnalysisWithNoDependenciesReturnsEmptyList(t *testing.T) {
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
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/analyses/"+analysis.ID+"/dependencies", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var response dependenciesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Dependencies) != 0 {
		t.Fatalf("dependencies = %#v", response.Dependencies)
	}
}

func TestWorkflowEndpointReturnsGraphWithoutMatchedValues(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, t.TempDir()+"/api.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	importAnalysis := app.NewImportAnalysis(har.NewImporter(1024*1024), store)
	mux := http.NewServeMux()
	registerRoutes(mux, store, importAnalysis)

	input, err := os.Open("../../examples/har/dependency.har")
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/import/har", input))
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var analysis analysisResponse
	if err := json.Unmarshal(w.Body.Bytes(), &analysis); err != nil {
		t.Fatal(err)
	}

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/analyses/"+analysis.ID+"/workflow", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	for _, value := range []string{"item_12345", "next_abc123", "p_12345", "form_98765"} {
		if strings.Contains(w.Body.String(), value) {
			t.Fatalf("workflow response leaked matched value %q: %s", value, w.Body.String())
		}
	}
	var response workflowResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Nodes) != 6 || len(response.Edges) != 4 {
		t.Fatalf("workflow = %#v", response)
	}
}

func TestWorkflowEndpointExistingAnalysisWithNoDependenciesReturnsNodesAndEmptyEdges(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, t.TempDir()+"/api.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	importAnalysis := app.NewImportAnalysis(har.NewImporter(1024*1024), store)
	mux := http.NewServeMux()
	registerRoutes(mux, store, importAnalysis)

	body := strings.NewReader(`{"log":{"entries":[
		{"startedDateTime":"2026-01-02T03:04:05Z","time":15,"request":{"method":"GET","url":"https://example.com/api/items?page=1","headers":[]},"response":{"status":200,"headers":[]}},
		{"startedDateTime":"2026-01-02T03:04:06Z","time":16,"request":{"method":"GET","url":"https://example.com/api/items?page=2","headers":[]},"response":{"status":200,"headers":[]}}
	]}}`)
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
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/analyses/"+analysis.ID+"/workflow", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var response workflowResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Nodes) != 2 || len(response.Edges) != 0 {
		t.Fatalf("workflow = %#v", response)
	}
}

func TestWorkflowEndpointMissingAnalysisReturnsNotFound(t *testing.T) {
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
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/analyses/missing/workflow", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}
