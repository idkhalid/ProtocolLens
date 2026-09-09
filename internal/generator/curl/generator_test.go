package curl

import (
	"net/http"
	"net/url"
	"testing"

	"protocollens/internal/generator"
	"protocollens/internal/replay"
)

func TestGenerateCurl(t *testing.T) {
	in := generator.Input{
		Method: "POST",
		URL:    "https://example.com/api/items?safe=1",
		Query: url.Values{
			"safe":  {"1"},
			"token": {replay.Redacted},
		},
		Headers: http.Header{
			"Accept":       {"application/json"},
			"Authorization": {"Bearer " + replay.Redacted},
			"Content-Type": {"application/json"},
			"X-Special":    {"value with 'single quotes' and \"double\" & ampersand\nnewline"},
		},
		BodyStruct: map[string]any{
			"name":     "safe",
			"password": replay.Redacted,
			"nested": map[string]any{
				"secret": replay.Redacted,
			},
		},
	}

	out, err := Generate(in)
	if err != nil {
		t.Fatal(err)
	}

	expectedCode := `curl \
  --request POST \
  "https://example.com/api/items?safe=1&token=${TOKEN}" \
  --header 'Accept: application/json' \
  --header "Authorization: Bearer ${AUTHORIZATION}" \
  --header 'Content-Type: application/json' \
  --header 'X-Special: value with '\''single quotes'\'' and "double" & ampersand
newline' \
  --data "{
  \"name\": \"safe\",
  \"nested\": {
    \"secret\": \"${SECRET}\"
  },
  \"password\": \"${PASSWORD}\"
}"`

	if out.Code != expectedCode {
		t.Fatalf("expected:\n%s\ngot:\n%s", expectedCode, out.Code)
	}

	if len(out.EnvironmentVariables) != 4 {
		t.Fatalf("expected 4 env vars, got %v", out.EnvironmentVariables)
	}
	if out.EnvironmentVariables[0] != "AUTHORIZATION" || out.EnvironmentVariables[3] != "TOKEN" {
		t.Fatalf("env vars not sorted or wrong: %v", out.EnvironmentVariables)
	}
}
