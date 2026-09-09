package replay

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestRedirectPolicyBlocksUnsafeAndStripsCrossOriginSecrets(t *testing.T) {
	policy := NewDestinationPolicy([]uint16{80, 443})
	policy.Resolver = resolver{"host": {{IP: net.ParseIP("93.184.216.34")}}}
	ex := &Executor{Policy: policy, Timeout: time.Second}

	blocked, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1/", nil)
	via, _ := http.NewRequest(http.MethodGet, "https://example.com/", nil)
	if err := ex.checkRedirect(blocked, []*http.Request{via}); !errors.Is(err, ErrDestinationBlocked) {
		t.Fatalf("blocked redirect err = %v", err)
	}

	next, _ := http.NewRequest(http.MethodGet, "https://other.example/", nil)
	next.Header.Set("Authorization", "Bearer secret")
	next.Header.Set("Cookie", "session=secret")
	next.Header.Set("Proxy-Authorization", "secret")
	next.Header.Set("X-Api-Key", "secret")
	next.Header.Set("X-Csrf-Token", "secret")
	if err := ex.checkRedirect(next, []*http.Request{via}); err != nil {
		t.Fatalf("public redirect err = %v", err)
	}
	if next.Header.Get("Authorization") != "" || next.Header.Get("Cookie") != "" || next.Header.Get("Proxy-Authorization") != "" || next.Header.Get("X-Api-Key") != "" || next.Header.Get("X-Csrf-Token") != "" {
		t.Fatalf("sensitive headers not stripped: %#v", next.Header)
	}

	if err := ex.checkRedirect(next, []*http.Request{via, via, via, via, via}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("redirect loop err = %v", err)
	}
}

func TestExecutorMapsTimeout(t *testing.T) {
	ex := testExecutor(roundTrip(func(*http.Request) (*http.Response, error) { return nil, context.DeadlineExceeded }))
	_, err := ex.Execute(context.Background(), Request{Method: http.MethodGet, URL: "https://example.com/"})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("timeout err = %v", err)
	}
}

func TestSameOrigin(t *testing.T) {
	a, _ := url.Parse("https://example.com/a")
	b, _ := url.Parse("https://example.com/b")
	c, _ := url.Parse("https://other.example/b")
	if !sameOrigin(a, b) || sameOrigin(a, c) {
		t.Fatal("sameOrigin mismatch")
	}
}
