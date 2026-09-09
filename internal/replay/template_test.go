package replay

import (
	"net/http"
	"strings"
	"testing"

	"protocollens/internal/domain"
)

func TestBuildTemplateRedactsSensitiveHeadersQueryAndBodies(t *testing.T) {
	req := domain.Request{
		ID:     "req-1",
		Method: "POST",
		URL:    "https://example.com/api/items?token=raw&page=1",
		Headers: map[string][]string{
			"Authorization": {"Bearer <REDACTED>"},
			"Content-Type":  {"application/json"},
		},
		Body: []byte(`{"name":"safe","password":"raw","nested":{"api_key":"raw"}}`),
	}
	template, err := BuildTemplate("a1", req)
	if err != nil {
		t.Fatal(err)
	}
	text := template.URL + template.Body + template.Headers.Get("Authorization")
	if strings.Contains(text, "raw") {
		t.Fatalf("template leaked secret: %#v", template)
	}
	if !template.RequiresReview || !strings.Contains(template.URL, "token=%3CREDACTED%3E") || !strings.Contains(template.Body, Redacted) || template.Query.Get("page") != "1" {
		t.Fatalf("template redaction failed: %#v", template)
	}
}

func TestBuildTemplateRedactsFormAndOmitsOpaqueBody(t *testing.T) {
	form := domain.Request{ID: "req-1", Method: "POST", URL: "https://example.com/login", Headers: map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}}, Body: []byte("username=me&csrf=raw")}
	template, err := BuildTemplate("a1", form)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(template.Body, "raw") || !strings.Contains(template.Body, "csrf=%3CREDACTED%3E") {
		t.Fatalf("form body = %q", template.Body)
	}
	opaque := domain.Request{ID: "req-2", Method: "POST", URL: "https://example.com/upload", Headers: map[string][]string{"Content-Type": {"application/octet-stream"}}, Body: []byte{0, 1, 2}}
	template, err = BuildTemplate("a1", opaque)
	if err != nil {
		t.Fatal(err)
	}
	if template.BodyAvailable || template.BodyReason != "opaque_body_omitted" || template.Body != "" {
		t.Fatalf("opaque template = %#v", template)
	}
}

func TestSafeResponseHeadersRedactSensitiveResponseHeaders(t *testing.T) {
	headers := safeResponseHeaders(http.Header{"Set-Cookie": {"session=raw"}, "Www-Authenticate": {"Bearer raw"}, "Content-Type": {"text/plain"}})
	if headers.Get("Set-Cookie") != Redacted || headers.Get("Www-Authenticate") != Redacted || headers.Get("Content-Type") != "text/plain" {
		t.Fatalf("headers = %#v", headers)
	}
}
