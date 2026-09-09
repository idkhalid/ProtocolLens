package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"protocollens/internal/app"
	"protocollens/internal/capture/har"
	"protocollens/internal/replay"
	"protocollens/internal/storage/sqlite"
)

type replayRoundTrip func(*http.Request) (*http.Response, error)

func (rt replayRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return rt(req) }

type replayResolver map[string][]net.IPAddr

func (r replayResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return r["host"], nil
}

func TestReplayDisabledReturnsForbidden(t *testing.T) {
	store, importAnalysis := testReplayStore(t)
	mux := http.NewServeMux()
	registerRoutes(mux, store, importAnalysis)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/replay", strings.NewReader(`{"method":"GET","url":"https://example.com"}`)))
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "replay_disabled") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestReplayTemplateRedactsSecrets(t *testing.T) {
	store, importAnalysis := testReplayStore(t)
	mux := http.NewServeMux()
	policy := replay.NewDestinationPolicy([]uint16{80, 443})
	policy.Resolver = replayResolver{"host": {{IP: net.ParseIP("93.184.216.34")}}}
	executor := &replay.Executor{Policy: policy, Client: &http.Client{Transport: replayRoundTrip(func(*http.Request) (*http.Response, error) { t.Fatal("template executed network"); return nil, nil })}, Timeout: time.Second, MaxRequestSize: 1024, MaxResponseSize: 1024}
	registerRoutes(mux, store, importAnalysis, NewReplayRoutes(app.NewGetReplayTemplate(store), app.NewExecuteReplay(false, executor, 1), 1024))

	analysisID := importReplayHAR(t, mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/analyses/"+analysisID+"/requests/req-000001/replay-template", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	for _, secret := range []string{"test-token-value", "fake-session-value", "example-api-key", "raw-token"} {
		if strings.Contains(w.Body.String(), secret) {
			t.Fatalf("template leaked %q: %s", secret, w.Body.String())
		}
	}
	var template replay.Template
	if err := json.Unmarshal(w.Body.Bytes(), &template); err != nil {
		t.Fatal(err)
	}
	if template.Query.Get("token") != replay.Redacted || !strings.Contains(template.Body, replay.Redacted) || template.Headers.Get("Api-Key") != replay.Redacted || !template.RequiresReview {
		t.Fatalf("template missing redaction: %#v", template)
	}
}

func TestReplayAPISuccessAndRequestTooLarge(t *testing.T) {
	store, importAnalysis := testReplayStore(t)
	mux := http.NewServeMux()
	policy := replay.NewDestinationPolicy([]uint16{80, 443})
	policy.Resolver = replayResolver{"host": {{IP: net.ParseIP("93.184.216.34")}}}
	executor := &replay.Executor{
		Policy: policy,
		Client: &http.Client{Transport: replayRoundTrip(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Request: req, Header: http.Header{"Content-Type": {"application/json"}, "Set-Cookie": {"session=raw"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), ContentLength: 11}, nil
		}), Timeout: time.Second},
		Timeout: time.Second, MaxRequestSize: 8, MaxResponseSize: 1024,
	}
	registerRoutes(mux, store, importAnalysis, NewReplayRoutes(app.NewGetReplayTemplate(store), app.NewExecuteReplay(true, executor, 1), 8))

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/replay", strings.NewReader(`{"method":"POST","url":"https://example.com/api","body":"too large"}`)))
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("large status=%d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/replay", strings.NewReader(`{"method":"GET","url":"https://example.com/api","headers":{"Accept":["application/json"]}}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var out replayResponse
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.StatusCode != 200 || out.Headers["Set-Cookie"][0] != replay.Redacted || !out.BodyAvailable || !strings.Contains(out.Body, "ok") {
		t.Fatalf("response = %#v", out)
	}
}

func testReplayStore(t *testing.T) (*sqlite.Store, *app.ImportAnalysis) {
	t.Helper()
	store, err := sqlite.Open(context.Background(), t.TempDir()+"/api.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, app.NewImportAnalysis(har.NewImporter(1024*1024), store)
}

func importReplayHAR(t *testing.T, mux *http.ServeMux) string {
	t.Helper()
	body := bytes.NewBufferString(`{"log":{"entries":[{"startedDateTime":"2026-01-02T03:04:05Z","time":15,"request":{"method":"POST","url":"https://example.com/api/items?token=raw-token&page=1","headers":[{"name":"Authorization","value":"Bearer test-token-value"},{"name":"Cookie","value":"session_id=fake-session-value"},{"name":"API-Key","value":"example-api-key"},{"name":"Content-Type","value":"application/json"}],"postData":{"mimeType":"application/json","text":"{\"name\":\"safe\",\"password\":\"raw-pass\"}"}},"response":{"status":200,"headers":[]}}]}}`)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/import/har", body))
	if w.Code != http.StatusCreated {
		t.Fatalf("import status=%d body=%s", w.Code, w.Body.String())
	}
	var analysis analysisResponse
	if err := json.Unmarshal(w.Body.Bytes(), &analysis); err != nil {
		t.Fatal(err)
	}
	return analysis.ID
}
