package golang

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"protocollens/internal/generator"
	"protocollens/internal/replay"
)

func TestGenerateGolang(t *testing.T) {
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
		},
		BodyStruct: map[string]any{
			"password": replay.Redacted,
		},
	}

	out, err := Generate(in)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.Code, `params.Add("token", os.Getenv("TOKEN"))`) {
		t.Fatalf("missing query env:\n%s", out.Code)
	}

	if !strings.Contains(out.Code, `req.Header.Set("Authorization", "Bearer "+os.Getenv("AUTHORIZATION"))`) {
		t.Fatalf("missing header env:\n%s", out.Code)
	}

	if !strings.Contains(out.Code, `"password": os.Getenv("PASSWORD")`) {
		t.Fatalf("missing json env:\n%s", out.Code)
	}

	if !strings.Contains(out.Code, `req, err := http.NewRequest("POST", rawURL, bodyReader)`) {
		t.Fatalf("missing http request:\n%s", out.Code)
	}
}
