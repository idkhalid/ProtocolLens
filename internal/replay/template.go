package replay

import (
	"bytes"
	"encoding/json"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"protocollens/internal/domain"
)

const Redacted = "<REDACTED>"

type Template struct {
	AnalysisID     string      `json:"analysisId"`
	RequestID      string      `json:"requestId"`
	Method         string      `json:"method"`
	URL            string      `json:"url"`
	Scheme         string      `json:"scheme"`
	Host           string      `json:"host"`
	Port           string      `json:"port"`
	Path           string      `json:"path"`
	Query          url.Values  `json:"query"`
	Headers        http.Header `json:"headers"`
	Body           string      `json:"body"`
	BodyAvailable  bool        `json:"bodyAvailable"`
	BodyReason     string      `json:"bodyReason,omitempty"`
	ContentType    string      `json:"contentType,omitempty"`
	RequiresReview bool        `json:"requiresReview"`
}

func BuildTemplate(analysisID string, request domain.Request) (Template, error) {
	u, err := url.Parse(request.URL)
	if err != nil {
		return Template{}, err
	}
	query := u.Query()
	review := false
	for name, values := range query {
		if sensitiveName(name) {
			for n := range values {
				values[n] = Redacted
			}
			query[name] = values
			review = true
		}
	}
	u.RawQuery = query.Encode()
	t := Template{AnalysisID: analysisID, RequestID: request.ID, Method: request.Method, URL: u.String(), Scheme: u.Scheme, Host: u.Hostname(), Port: u.Port(), Path: u.EscapedPath(), Query: query, Headers: http.Header{}, BodyAvailable: true, ContentType: headerValue(request.Headers, "Content-Type"), RequiresReview: review}
	for name, values := range request.Headers {
		copyValues := append([]string(nil), values...)
		for _, value := range copyValues {
			if strings.Contains(value, Redacted) {
				t.RequiresReview = true
			}
		}
		t.Headers[http.CanonicalHeaderKey(name)] = copyValues
	}
	body, available, reason, redacted := safeBody(request.Body, t.ContentType)
	t.Body = body
	t.BodyAvailable = available
	t.BodyReason = reason
	if redacted {
		t.RequiresReview = true
	}
	return t, nil
}

func safeBody(body []byte, contentType string) (string, bool, string, bool) {
	if len(body) == 0 {
		return "", true, "", false
	}
	mediaType, _, _ := mime.ParseMediaType(contentType)
	mediaType = strings.ToLower(mediaType)
	switch mediaType {
	case "application/json":
		var value any
		if err := json.Unmarshal(body, &value); err != nil {
			return "", false, "unsafe_unstructured_body", false
		}
		redacted := redactJSON(value)
		var out bytes.Buffer
		encoder := json.NewEncoder(&out)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(value); err != nil {
			return "", false, "unsafe_unstructured_body", false
		}
		return strings.TrimSuffix(out.String(), "\n"), true, "", redacted
	case "application/x-www-form-urlencoded":
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return "", false, "unsafe_unstructured_body", false
		}
		redacted := false
		for name, items := range values {
			if sensitiveName(name) {
				for n := range items {
					items[n] = Redacted
				}
				values[name] = items
				redacted = true
			}
		}
		return values.Encode(), true, "", redacted
	default:
		if strings.HasPrefix(mediaType, "text/") {
			return string(body), true, "", false
		}
		return "", false, "opaque_body_omitted", false
	}
}

func redactJSON(value any) bool {
	redacted := false
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			if sensitiveName(key) {
				v[key] = Redacted
				redacted = true
				continue
			}
			if redactJSON(item) {
				redacted = true
			}
		}
	case []any:
		for _, item := range v {
			if redactJSON(item) {
				redacted = true
			}
		}
	}
	return redacted
}

func sensitiveName(name string) bool {
	name = strings.ToLower(strings.ReplaceAll(name, "-", "_"))
	switch name {
	case "token", "access_token", "auth", "authorization", "api_key", "apikey", "key", "secret", "password", "passwd", "session", "session_id", "csrf", "xsrf", "x_csrf_token", "x_csrftoken", "x_xsrf_token", "x_xsrftoken":
		return true
	default:
		return false
	}
}

func headerValue(headers map[string][]string, name string) string {
	for key, values := range headers {
		if strings.EqualFold(key, name) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}
