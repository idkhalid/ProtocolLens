package normalize

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"sort"
	"strings"
	"time"

	"protocollens/internal/capture/har"
	"protocollens/internal/domain"
)

func FromHAR(entries []har.Entry) ([]domain.Exchange, error) {
	exchanges := make([]domain.Exchange, 0, len(entries))
	for n, entry := range entries {
		u, err := url.Parse(entry.Request.URL)
		if err != nil {
			return nil, fmt.Errorf("parse request URL %q: %w", entry.Request.URL, err)
		}
		if u.Scheme == "" || u.Host == "" {
			return nil, fmt.Errorf("request URL must be absolute: %q", entry.Request.URL)
		}

		timestamp, _ := time.Parse(time.RFC3339Nano, entry.StartedDateTime)
		if timestamp.IsZero() {
			timestamp = time.Now().UTC()
		}

		exchanges = append(exchanges, domain.Exchange{
			Request: domain.Request{
				ID:        fmt.Sprintf("req-%06d", n+1),
				Method:    strings.ToUpper(entry.Request.Method),
				URL:       u.String(),
				Headers:   headers(entry.Request.Headers),
				Query:     u.Query(),
				Body:      requestBody(entry.Request.PostData),
				Timestamp: timestamp,
			},
			Response: domain.Response{
				StatusCode: entry.Response.Status,
				Headers:    headers(entry.Response.Headers),
				Body:       responseBody(entry.Response.Content),
				Duration:   duration(entry.Time),
			},
		})
	}
	return exchanges, nil
}

func EndpointOf(request domain.Request) (domain.Endpoint, error) {
	u, err := url.Parse(request.URL)
	if err != nil {
		return domain.Endpoint{}, err
	}
	if u.Scheme == "" || u.Host == "" {
		return domain.Endpoint{}, fmt.Errorf("request URL must be absolute: %q", request.URL)
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	return domain.Endpoint{
		Method: strings.ToUpper(request.Method),
		Host:   host(u),
		Path:   path,
	}, nil
}

func host(u *url.URL) string {
	name := strings.ToLower(u.Hostname())
	port := u.Port()
	if port == "" || (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		return name
	}
	if strings.Contains(name, ":") {
		return net.JoinHostPort(name, port)
	}
	return name + ":" + port
}

func duration(ms float64) time.Duration {
	if ms <= 0 {
		return 0
	}
	return time.Duration(ms * float64(time.Millisecond))
}

func headers(values []har.NameValue) map[string][]string {
	out := make(map[string][]string, len(values))
	for _, header := range values {
		name := strings.TrimSpace(header.Name)
		if name == "" {
			continue
		}
		canonical := textproto.CanonicalMIMEHeaderKey(name)
		value, ok := safeHeader(canonical, header.Value)
		if ok {
			out[canonical] = append(out[canonical], value)
		}
	}
	return out
}

func safeHeader(name, value string) (string, bool) {
	switch name {
	case "Authorization":
		if strings.HasPrefix(strings.ToLower(value), "bearer ") {
			return "Bearer <REDACTED>", true
		}
		return "<REDACTED>", true
	case "Cookie":
		names := cookieNames(value)
		if len(names) == 0 {
			return "", false
		}
		return strings.Join(names, "; "), true
	case "Set-Cookie":
		return setCookieMetadata(value)
	case "Proxy-Authorization", "X-Api-Key", "Api-Key", "X-Csrf-Token", "X-Csrftoken", "X-Xsrf-Token", "X-Xsrftoken", "Csrf-Token":
		return "<REDACTED>", true
	default:
		return value, true
	}
}

func cookieNames(value string) []string {
	seen := map[string]bool{}
	var names []string
	for _, part := range strings.Split(value, ";") {
		name, _, ok := strings.Cut(strings.TrimSpace(part), "=")
		name = strings.TrimSpace(name)
		if ok && name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func setCookieMetadata(value string) (string, bool) {
	first, _, _ := strings.Cut(value, ";")
	if !strings.Contains(first, "=") {
		return "", false
	}
	response := http.Response{Header: http.Header{"Set-Cookie": []string{value}}}
	cookies := response.Cookies()
	if len(cookies) == 0 || cookies[0].Name == "" {
		return "", false
	}

	cookie := cookies[0]
	parts := []string{cookie.Name}
	if cookie.Path != "" {
		parts = append(parts, "Path="+cookie.Path)
	}
	if cookie.Domain != "" {
		parts = append(parts, "Domain="+cookie.Domain)
	}
	if cookie.HttpOnly {
		parts = append(parts, "HttpOnly")
	}
	if cookie.Secure {
		parts = append(parts, "Secure")
	}
	if value := sameSite(cookie.SameSite); value != "" {
		parts = append(parts, "SameSite="+value)
	}
	if cookie.MaxAge != 0 {
		parts = append(parts, "Max-Age")
	}
	if !cookie.Expires.IsZero() {
		parts = append(parts, "Expires")
	}
	return strings.Join(parts, "; "), true
}

func sameSite(value http.SameSite) string {
	switch value {
	case http.SameSiteLaxMode:
		return "Lax"
	case http.SameSiteStrictMode:
		return "Strict"
	case http.SameSiteNoneMode:
		return "None"
	default:
		return ""
	}
}

func requestBody(postData *har.PostData) []byte {
	if postData == nil {
		return nil
	}
	return []byte(postData.Text)
}

func responseBody(content *har.Content) []byte {
	if content == nil || content.Text == "" {
		return nil
	}
	if strings.EqualFold(content.Encoding, "base64") {
		decoded, err := base64.StdEncoding.DecodeString(content.Text)
		if err == nil {
			return decoded
		}
	}
	return []byte(content.Text)
}
