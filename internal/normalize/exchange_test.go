package normalize

import (
	"testing"

	"protocollens/internal/capture/har"
	"protocollens/internal/domain"
)

func TestEndpointOfDropsQuery(t *testing.T) {
	endpoint, err := EndpointOf(domain.Request{
		Method: "get",
		URL:    "https://Example.com/api/items?page=1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if endpoint.Method != "GET" || endpoint.Host != "example.com" || endpoint.Path != "/api/items" {
		t.Fatalf("endpoint = %#v", endpoint)
	}
}

func TestEndpointOfDropsDefaultPort(t *testing.T) {
	endpoint, err := EndpointOf(domain.Request{
		Method: "GET",
		URL:    "https://example.com:443/api/items",
	})
	if err != nil {
		t.Fatal(err)
	}
	if endpoint.Host != "example.com" {
		t.Fatalf("host = %q", endpoint.Host)
	}
}

func TestEndpointOfKeepsNonDefaultPort(t *testing.T) {
	endpoint, err := EndpointOf(domain.Request{
		Method: "GET",
		URL:    "https://example.com:8443/api/items",
	})
	if err != nil {
		t.Fatal(err)
	}
	if endpoint.Host != "example.com:8443" {
		t.Fatalf("host = %q", endpoint.Host)
	}
}

func TestFromHARRejectsRelativeURL(t *testing.T) {
	_, err := FromHAR([]har.Entry{{
		Request: har.Request{Method: "GET", URL: "/api/items"},
	}})
	if err == nil {
		t.Fatal("expected relative URL error")
	}
}

func TestFromHARClampsNegativeDuration(t *testing.T) {
	exchanges, err := FromHAR([]har.Entry{{
		Request: har.Request{Method: "GET", URL: "https://example.com/api/items"},
		Time:    -1,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if exchanges[0].Response.Duration != 0 {
		t.Fatalf("duration = %s", exchanges[0].Response.Duration)
	}
}

func TestFromHARRedactsSensitiveHeaders(t *testing.T) {
	exchanges, err := FromHAR([]har.Entry{{
		Request: har.Request{
			Method: "GET",
			URL:    "https://example.com/api/items",
			Headers: []har.NameValue{
				{Name: "Authorization", Value: "Bearer secret-token"},
				{Name: "Cookie", Value: "session=secret"},
				{Name: "Accept", Value: "application/json"},
			},
		},
		Response: har.Response{
			Headers: []har.NameValue{{Name: "Set-Cookie", Value: "session=secret"}},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if exchanges[0].Request.Headers["Authorization"][0] != "Bearer <REDACTED>" {
		t.Fatalf("authorization = %q", exchanges[0].Request.Headers["Authorization"])
	}
	if exchanges[0].Request.Headers["Cookie"][0] != "session" {
		t.Fatalf("cookie = %q", exchanges[0].Request.Headers["Cookie"])
	}
	if exchanges[0].Request.Headers["Accept"][0] != "application/json" {
		t.Fatalf("accept = %q", exchanges[0].Request.Headers["Accept"])
	}
	if exchanges[0].Response.Headers["Set-Cookie"][0] != "session" {
		t.Fatalf("set-cookie = %q", exchanges[0].Response.Headers["Set-Cookie"])
	}
}

func TestFromHARPreservesMultipleSetCookiesSafely(t *testing.T) {
	exchanges, err := FromHAR([]har.Entry{{
		Request: har.Request{Method: "GET", URL: "https://example.com/api/items"},
		Response: har.Response{
			Headers: []har.NameValue{
				{Name: "Set-Cookie", Value: "session_id=fake-session-value; Path=/; HttpOnly; Secure; SameSite=Lax"},
				{Name: "set-cookie", Value: "theme=dark; Path=/"},
				{Name: "Set-Cookie", Value: "malformed-cookie"},
			},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	got := exchanges[0].Response.Headers["Set-Cookie"]
	if len(got) != 2 {
		t.Fatalf("set-cookie len = %d: %#v", len(got), got)
	}
	if got[0] != "session_id; Path=/; HttpOnly; Secure; SameSite=Lax" || got[1] != "theme; Path=/" {
		t.Fatalf("set-cookie = %#v", got)
	}
}
