package har

import (
	"strings"
	"testing"
)

func TestImporterReadsHAR(t *testing.T) {
	entries, err := NewImporter(1024).Import(strings.NewReader(`{
		"log": {"entries": [{
			"startedDateTime": "2026-01-02T03:04:05Z",
			"time": 12.5,
			"request": {"method": "GET", "url": "https://example.com/api/items?page=1", "headers": []},
			"response": {"status": 200, "headers": [{"name":"Content-Type","value":"application/json"}]}
		}]}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := entries[0].Request.URL; got != "https://example.com/api/items?page=1" {
		t.Fatalf("url = %q", got)
	}
}

func TestImporterRejectsOversizedHAR(t *testing.T) {
	_, err := NewImporter(10).Import(strings.NewReader(`{"log":{"entries":[]}}`))
	if err == nil {
		t.Fatal("expected oversized HAR error")
	}
}
