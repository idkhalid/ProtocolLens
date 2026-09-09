package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"protocollens/internal/app"
	"protocollens/internal/generator"
	"protocollens/internal/replay"
)

func TestGenerateClientNetworkAbsence(t *testing.T) {
	store, importAnalysis := testReplayStore(t)
	mux := http.NewServeMux()
	
	executor := &replay.Executor{
		Client: &http.Client{Transport: replayRoundTrip(func(*http.Request) (*http.Response, error) { 
			t.Fatal("generator executed network")
			return nil, nil 
		})}, 
		Timeout: time.Second,
	}

	getTemplate := app.NewGetReplayTemplate(store)
	generateClient := app.NewGenerateClient(getTemplate)
	registerRoutes(mux, store, importAnalysis, NewReplayRoutes(getTemplate, app.NewExecuteReplay(true, executor, 1), generateClient, 1024))

	analysisID := importReplayHAR(t, mux)
	
	reqBody := `{"analysisId":"` + analysisID + `", "requestId":"req-000001", "target":"curl"}`
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/generate", strings.NewReader(reqBody)))
	
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	
	var out generator.Output
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	
	if out.Target != "curl" {
		t.Fatalf("expected curl, got %s", out.Target)
	}
	
	// Secret absence
	if strings.Contains(out.Code, "test-token-value") || strings.Contains(out.Code, "fake-session-value") {
		t.Fatalf("generated code leaked secrets: %s", out.Code)
	}
	if strings.Contains(out.Code, replay.Redacted) {
		t.Fatalf("generated code leaked <REDACTED> literal: %s", out.Code)
	}
}

func TestGenerateClientInvalidTarget(t *testing.T) {
	store, importAnalysis := testReplayStore(t)
	mux := http.NewServeMux()
	
	getTemplate := app.NewGetReplayTemplate(store)
	generateClient := app.NewGenerateClient(getTemplate)
	registerRoutes(mux, store, importAnalysis, NewReplayRoutes(getTemplate, nil, generateClient, 1024))

	analysisID := importReplayHAR(t, mux)
	
	reqBody := `{"analysisId":"` + analysisID + `", "requestId":"req-000001", "target":"invalid"}`
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/generate", strings.NewReader(reqBody)))
	
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGenerateClientMissingAnalysis(t *testing.T) {
	store, importAnalysis := testReplayStore(t)
	mux := http.NewServeMux()
	
	getTemplate := app.NewGetReplayTemplate(store)
	generateClient := app.NewGenerateClient(getTemplate)
	registerRoutes(mux, store, importAnalysis, NewReplayRoutes(getTemplate, nil, generateClient, 1024))

	reqBody := `{"analysisId":"not-found", "requestId":"req-000001", "target":"curl"}`
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/generate", strings.NewReader(reqBody)))
	
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
