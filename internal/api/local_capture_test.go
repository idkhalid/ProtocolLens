package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"protocollens/internal/app"
	"protocollens/internal/capture/playwright"
)

type fakeCaptureRunner struct {
	calls   atomic.Int32
	started chan struct{}
	release chan struct{}
	last    playwright.Options
}

func (r *fakeCaptureRunner) Capture(ctx context.Context, opts playwright.Options) (playwright.Result, error) {
	r.calls.Add(1)
	r.last = opts
	if r.started != nil {
		close(r.started)
	}
	if r.release != nil {
		select {
		case <-r.release:
		case <-ctx.Done():
			return playwright.Result{}, ctx.Err()
		}
	}
	f, err := os.CreateTemp("", "capture-test-*.har")
	if err != nil {
		return playwright.Result{}, err
	}
	_, _ = f.WriteString(`{"log":{"entries":[{"startedDateTime":"2026-01-02T03:04:05Z","time":15,"request":{"method":"GET","url":"https://example.com/","headers":[]},"response":{"status":200,"headers":[]}}]}}`)
	_ = f.Close()
	return playwright.Result{Path: f.Name(), Cleanup: func() { _ = os.Remove(f.Name()) }}, nil
}

func (r *fakeCaptureRunner) Available() error { return nil }

func TestLocalCaptureDisabledByDefault(t *testing.T) {
	store, importAnalysis := testReplayStore(t)
	mux := http.NewServeMux()
	registerLocalCaptureRoutes(mux, NewLocalCaptureRoutes("127.0.0.1:8080", []uint16{80, 443}, app.NewLocalCapture(false, &fakeCaptureRunner{}, importAnalysis)))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/local/capture", strings.NewReader(`{"url":"https://example.com","duration_seconds":5,"headed":false}`))
	req.Host = "localhost:8080"
	req.RemoteAddr = "127.0.0.1:12345"
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden || store == nil {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestLocalCaptureRejectsNonLocalHostAndOrigin(t *testing.T) {
	_, importAnalysis := testReplayStore(t)
	mux := http.NewServeMux()
	registerLocalCaptureRoutes(mux, NewLocalCaptureRoutes("127.0.0.1:8080", []uint16{80, 443}, app.NewLocalCapture(true, &fakeCaptureRunner{}, importAnalysis)))

	for _, tt := range []struct{ host, origin string }{{"evil.test", ""}, {"localhost:8080", "https://evil.test"}} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/local/capture", strings.NewReader(`{"url":"https://example.com","duration_seconds":5,"headed":false}`))
		req.Host = tt.host
		req.Header.Set("Origin", tt.origin)
		req.RemoteAddr = "127.0.0.1:12345"
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("%+v status=%d body=%s", tt, w.Code, w.Body.String())
		}
	}
}

func TestLocalCaptureSuccessAndBusy(t *testing.T) {
	_, importAnalysis := testReplayStore(t)
	runner := &fakeCaptureRunner{started: make(chan struct{}), release: make(chan struct{})}
	mux := http.NewServeMux()
	registerLocalCaptureRoutes(mux, NewLocalCaptureRoutes("127.0.0.1:8080", []uint16{80, 443}, app.NewLocalCapture(true, runner, importAnalysis)))

	firstDone := make(chan int, 1)
	go func() {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/local/capture", strings.NewReader(`{"url":"https://example.com","duration_seconds":5,"headed":true}`))
		req.Host = "localhost:8080"
		req.RemoteAddr = "127.0.0.1:12345"
		mux.ServeHTTP(w, req)
		firstDone <- w.Code
	}()
	<-runner.started

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/local/capture", strings.NewReader(`{"url":"https://example.com","duration_seconds":5,"headed":false}`))
	req.Host = "localhost:8080"
	req.RemoteAddr = "127.0.0.1:12345"
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusConflict || runner.calls.Load() != 1 {
		t.Fatalf("busy status=%d calls=%d body=%s", w.Code, runner.calls.Load(), w.Body.String())
	}
	close(runner.release)
	if code := <-firstDone; code != http.StatusCreated {
		t.Fatalf("first status=%d", code)
	}
	if !runner.last.Headed || runner.last.Duration != 5e9 || runner.last.URL != "https://example.com" {
		t.Fatalf("opts=%#v", runner.last)
	}
}

func TestLocalCaptureRejectsBadJSONBeforeRunner(t *testing.T) {
	_, importAnalysis := testReplayStore(t)
	runner := &fakeCaptureRunner{}
	mux := http.NewServeMux()
	registerLocalCaptureRoutes(mux, NewLocalCaptureRoutes("127.0.0.1:8080", []uint16{80, 443}, app.NewLocalCapture(true, runner, importAnalysis)))

	bodies := []string{
		`{"url":`,
		`{"url":"https://example.com","duration_seconds":5,"headed":false,"script":"no"}`,
		`{"url":"https://example.com","duration_seconds":5,"headed":false} {"url":"https://example.org"}`,
		strings.Repeat("x", captureBodyLimit+1),
	}
	for _, body := range bodies {
		w := httptest.NewRecorder()
		req := localCaptureTestRequest(body)
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest || runner.calls.Load() != 0 {
			t.Fatalf("body %q status=%d calls=%d", body, w.Code, runner.calls.Load())
		}
	}
}

func TestLocalCaptureRejectsInvalidOptionsBeforeSlot(t *testing.T) {
	_, importAnalysis := testReplayStore(t)
	runner := &fakeCaptureRunner{}
	mux := http.NewServeMux()
	registerLocalCaptureRoutes(mux, NewLocalCaptureRoutes("127.0.0.1:8080", []uint16{80, 443}, app.NewLocalCapture(true, runner, importAnalysis)))

	bodies := []string{
		`{"url":"ftp://example.com","duration_seconds":5,"headed":false}`,
		`{"url":"http://user:pass@example.com","duration_seconds":5,"headed":false}`,
		`{"url":"http://127.0.0.1","duration_seconds":5,"headed":false}`,
		`{"url":"http://10.0.0.1","duration_seconds":5,"headed":false}`,
		`{"url":"https://example.com","duration_seconds":0,"headed":false}`,
		`{"url":"https://example.com","duration_seconds":61,"headed":false}`,
	}
	for _, body := range bodies {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, localCaptureTestRequest(body))
		if w.Code != http.StatusBadRequest || runner.calls.Load() != 0 {
			t.Fatalf("body %s status=%d calls=%d", body, w.Code, runner.calls.Load())
		}
	}
}

func TestLocalCaptureLocalRequestBoundary(t *testing.T) {
	_, importAnalysis := testReplayStore(t)
	mux := http.NewServeMux()
	registerLocalCaptureRoutes(mux, NewLocalCaptureRoutes("127.0.0.1:8080", []uint16{80, 443}, app.NewLocalCapture(true, &fakeCaptureRunner{}, importAnalysis)))

	valid := []struct{ host, origin, remote string }{
		{"localhost:8080", "http://localhost:5173", "127.0.0.1:12345"},
		{"127.0.0.1:8080", "http://127.0.0.1:5173", "127.0.0.1:12345"},
		{"[::1]:8080", "http://[::1]:5173", "[::1]:12345"},
		{"localhost:8080", "", "127.0.0.1:12345"},
	}
	for _, tt := range valid {
		req := localCaptureTestRequest(`{"url":"https://example.com","duration_seconds":5,"headed":false}`)
		req.Host, req.RemoteAddr = tt.host, tt.remote
		req.Header.Set("Origin", tt.origin)
		if !localWorkbenchRequest(req) {
			t.Fatalf("expected valid local request: %+v", tt)
		}
	}

	rejected := []struct{ host, origin, remote string }{
		{"localhost.evil.example", "", "127.0.0.1:12345"},
		{"127.0.0.1.evil.example", "", "127.0.0.1:12345"},
		{"evil.example", "", "127.0.0.1:12345"},
		{"localhost:8080", "https://evil.example", "127.0.0.1:12345"},
		{"localhost:8080", "http://localhost.evil.example", "127.0.0.1:12345"},
		{"localhost:8080", "", "192.168.1.5:12345"},
		{"localhost:8080", "", "8.8.8.8:12345"},
	}
	for _, tt := range rejected {
		req := localCaptureTestRequest(`{"url":"https://example.com","duration_seconds":5,"headed":false}`)
		req.Host, req.RemoteAddr = tt.host, tt.remote
		req.Header.Set("Origin", tt.origin)
		req.Header.Set("X-Forwarded-For", "127.0.0.1")
		if localWorkbenchRequest(req) {
			t.Fatalf("expected rejected local request: %+v", tt)
		}
	}
}

func localCaptureTestRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/local/capture", strings.NewReader(body))
	req.Host = "localhost:8080"
	req.RemoteAddr = "127.0.0.1:12345"
	return req
}
