package replay

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (rt roundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return rt(req) }

func testExecutor(rt http.RoundTripper) *Executor {
	policy := NewDestinationPolicy([]uint16{80, 443})
	policy.Resolver = resolver{"host": {{IP: net.ParseIP("93.184.216.34")}}}
	return &Executor{Policy: policy, Client: &http.Client{Transport: rt, Timeout: time.Second}, Timeout: time.Second, MaxRequestSize: 16, MaxResponseSize: 5}
}

func TestExecutorSendsRequestAndTruncatesTextResponse(t *testing.T) {
	ex := testExecutor(roundTrip(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.Header.Get("X-Test") != "yes" {
			t.Fatalf("request = %s %#v", req.Method, req.Header)
		}
		body, _ := io.ReadAll(req.Body)
		if string(body) != "hello" {
			t.Fatalf("body = %q", body)
		}
		return &http.Response{StatusCode: 200, Request: req, Header: http.Header{"Content-Type": {"text/plain"}}, Body: io.NopCloser(strings.NewReader("abcdef")), ContentLength: 6}, nil
	}))
	result, err := ex.Execute(context.Background(), Request{Method: "POST", URL: "https://example.com/api", Headers: http.Header{"X-Test": {"yes"}}, Body: []byte("hello")})
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != 200 || string(result.Body) != "abcde" || !result.Truncated || !result.BodyAvailable {
		t.Fatalf("result = %#v", result)
	}
}

func TestExecutorRejectsInvalidRequests(t *testing.T) {
	ex := testExecutor(roundTrip(func(*http.Request) (*http.Response, error) { t.Fatal("network executed"); return nil, nil }))
	cases := []Request{
		{Method: "TRACE", URL: "https://example.com/"},
		{Method: "GET", URL: "https://example.com/?token=<REDACTED>"},
		{Method: "GET", URL: "https://example.com/", Headers: http.Header{"Authorization": {"Bearer <REDACTED>"}}},
		{Method: "POST", URL: "https://example.com/", Body: []byte("<REDACTED>")},
		{Method: "GET", URL: "https://example.com/", Headers: http.Header{"Proxy-Authorization": {"x"}}},
		{Method: "GET", URL: "https://example.com/", Headers: http.Header{"Host": {"other.example"}}},
		{Method: "GET", URL: "https://example.com/", Headers: http.Header{"Content-Length": {"999"}}},
		{Method: "GET", URL: "https://example.com/", Headers: http.Header{"Accept-Encoding": {"gzip"}}},
	}
	for _, req := range cases {
		if _, err := ex.Execute(context.Background(), req); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("%#v error = %v", req, err)
		}
	}
	if _, err := ex.Execute(context.Background(), Request{Method: "POST", URL: "https://example.com/", Body: []byte("this body is too long")}); !errors.Is(err, ErrRequestTooLarge) {
		t.Fatalf("large body error = %v", err)
	}
}

func TestExecutorDoesNotExposeBinaryBody(t *testing.T) {
	ex := testExecutor(roundTrip(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Request: req, Header: http.Header{"Content-Type": {"application/octet-stream"}}, Body: io.NopCloser(strings.NewReader("abc")), ContentLength: 3}, nil
	}))
	result, err := ex.Execute(context.Background(), Request{Method: "GET", URL: "https://example.com/bin"})
	if err != nil {
		t.Fatal(err)
	}
	if result.BodyAvailable || len(result.Body) != 0 {
		t.Fatalf("result = %#v", result)
	}
}
