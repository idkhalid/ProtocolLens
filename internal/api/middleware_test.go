package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSAllowsOnlyLocalOrigins(t *testing.T) {
	handler := cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))

	local := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	handler.ServeHTTP(local, req)
	if local.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("local CORS origin = %q", local.Header().Get("Access-Control-Allow-Origin"))
	}

	remote := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://evil.example")
	handler.ServeHTTP(remote, req)
	if remote.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("remote CORS origin = %q", remote.Header().Get("Access-Control-Allow-Origin"))
	}
}
