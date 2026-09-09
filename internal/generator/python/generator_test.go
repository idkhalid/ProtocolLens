package python

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"protocollens/internal/generator"
	"protocollens/internal/replay"
)

func TestGeneratePython(t *testing.T) {
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

	if !strings.Contains(out.Code, `import httpx`) || !strings.Contains(out.Code, `import os`) {
		t.Fatalf("missing imports:\n%s", out.Code)
	}

	if !strings.Contains(out.Code, `"token": os.environ["TOKEN"]`) {
		t.Fatalf("missing param env:\n%s", out.Code)
	}

	if !strings.Contains(out.Code, `"Authorization": f"Bearer {os.environ['AUTHORIZATION']}"`) {
		t.Fatalf("missing header env:\n%s", out.Code)
	}

	if !strings.Contains(out.Code, `"password": os.environ["PASSWORD"]`) {
		t.Fatalf("missing json env:\n%s", out.Code)
	}

	if !strings.Contains(out.Code, `with httpx.Client() as client:`) {
		t.Fatalf("missing httpx call:\n%s", out.Code)
	}
}
