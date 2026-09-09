package replay

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrInvalidRequest  = errors.New("invalid_replay_request")
	ErrRequestTooLarge = errors.New("request_too_large")
	ErrTimeout         = errors.New("replay_timeout")
)

type Request struct {
	Method          string
	URL             string
	Headers         http.Header
	Body            []byte
	FollowRedirects bool
}

type Result struct {
	StatusCode    int
	Headers       http.Header
	Body          []byte
	BodyAvailable bool
	ContentLength int64
	ContentType   string
	Duration      time.Duration
	Truncated     bool
	FinalURL      string
}

type Executor struct {
	Client          *http.Client
	Policy          DestinationPolicy
	Timeout         time.Duration
	MaxRequestSize  int64
	MaxResponseSize int64
}

func NewExecutor(policy DestinationPolicy, timeout time.Duration, maxRequestSize, maxResponseSize int64) *Executor {
	ex := &Executor{Policy: policy, Timeout: timeout, MaxRequestSize: maxRequestSize, MaxResponseSize: maxResponseSize}
	ex.Client = &http.Client{Transport: policy.Transport(5 * time.Second), Timeout: timeout, CheckRedirect: ex.checkRedirect}
	return ex
}

func (e *Executor) Execute(ctx context.Context, in Request) (Result, error) {
	method := strings.ToUpper(strings.TrimSpace(in.Method))
	if !allowedMethod(method) || strings.Contains(in.URL, "<REDACTED>") || bytes.Contains(in.Body, []byte("<REDACTED>")) {
		return Result{}, ErrInvalidRequest
	}
	if int64(len(in.Body)) > e.MaxRequestSize {
		return Result{}, ErrRequestTooLarge
	}
	u, err := e.Policy.ValidateURL(ctx, in.URL)
	if err != nil {
		return Result{}, err
	}
	baseClient := e.Client
	if baseClient == nil {
		baseClient = &http.Client{Transport: e.Policy.Transport(5 * time.Second), Timeout: e.Timeout, CheckRedirect: e.checkRedirect}
	}
	client := *baseClient
	if !in.FollowRedirects {
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	} else if client.CheckRedirect == nil {
		client.CheckRedirect = e.checkRedirect
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(in.Body))
	if err != nil {
		return Result{}, ErrInvalidRequest
	}
	for name, values := range in.Headers {
		if !allowedHeader(name) {
			return Result{}, ErrInvalidRequest
		}
		for _, value := range values {
			if strings.Contains(value, "<REDACTED>") || strings.ContainsAny(value, "\r\n") {
				return Result{}, ErrInvalidRequest
			}
			req.Header.Add(name, value)
		}
	}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "Client.Timeout") {
			return Result{}, ErrTimeout
		}
		return Result{}, err
	}
	defer resp.Body.Close()
	result := Result{StatusCode: resp.StatusCode, Headers: safeResponseHeaders(resp.Header), Duration: time.Since(start), FinalURL: resp.Request.URL.String(), ContentLength: resp.ContentLength, ContentType: resp.Header.Get("Content-Type")}
	limit := e.MaxResponseSize
	if limit <= 0 {
		limit = 2 << 20
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return Result{}, err
	}
	if int64(len(body)) > limit {
		result.Truncated = true
		body = body[:limit]
	}
	if textBody(result.ContentType, body) {
		result.Body = body
		result.BodyAvailable = true
	}
	return result, nil
}

func (e *Executor) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 5 {
		return ErrInvalidRequest
	}
	if _, err := e.Policy.ValidateURL(req.Context(), req.URL.String()); err != nil {
		return err
	}
	prev := via[len(via)-1].URL
	if !sameOrigin(prev, req.URL) {
		stripCrossOriginCredentials(req.Header)
	}
	return nil
}

func allowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions:
		return true
	default:
		return false
	}
}

func allowedHeader(name string) bool {
	if name == "" || strings.ContainsAny(name, "\r\n:") {
		return false
	}
	switch http.CanonicalHeaderKey(name) {
	case "Accept-Encoding", "Connection", "Content-Length", "Host", "Proxy-Connection", "Keep-Alive", "Transfer-Encoding", "Te", "Trailer", "Upgrade", "Proxy-Authorization":
		return false
	default:
		return true
	}
}

func stripCrossOriginCredentials(headers http.Header) {
	for _, name := range []string{
		"Authorization",
		"Cookie",
		"Proxy-Authorization",
		"X-Api-Key",
		"Api-Key",
		"X-Csrf-Token",
		"X-Csrftoken",
		"X-Xsrf-Token",
		"X-Xsrftoken",
		"Csrf-Token",
	} {
		headers.Del(name)
	}
}

func safeResponseHeaders(headers http.Header) http.Header {
	out := http.Header{}
	for name, values := range headers {
		canonical := http.CanonicalHeaderKey(name)
		if canonical == "Set-Cookie" || canonical == "Proxy-Authenticate" || canonical == "Www-Authenticate" {
			out[canonical] = []string{"<REDACTED>"}
			continue
		}
		out[canonical] = append([]string(nil), values...)
	}
	return out
}

func textBody(contentType string, body []byte) bool {
	contentType = strings.ToLower(contentType)
	return utf8.Valid(body) && (strings.HasPrefix(contentType, "text/") || strings.Contains(contentType, "application/json") || strings.Contains(contentType, "application/xml") || strings.Contains(contentType, "application/javascript"))
}

func sameOrigin(a, b *url.URL) bool {
	return a.Scheme == b.Scheme && a.Host == b.Host
}
