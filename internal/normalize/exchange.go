package normalize

import (
	"encoding/base64"
	"fmt"
	"net/textproto"
	"net/url"
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
				Duration:   time.Duration(entry.Time * float64(time.Millisecond)),
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
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	return domain.Endpoint{
		Method: strings.ToUpper(request.Method),
		Host:   strings.ToLower(u.Host),
		Path:   path,
	}, nil
}

func headers(values []har.NameValue) map[string]string {
	out := make(map[string]string, len(values))
	for _, header := range values {
		name := strings.TrimSpace(header.Name)
		if name == "" {
			continue
		}
		out[textproto.CanonicalMIMEHeaderKey(name)] = header.Value
	}
	return out
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
