package app_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"protocollens/internal/app"
	"protocollens/internal/capture/har"
	"protocollens/internal/storage/sqlite"
)

func TestImportAnalysisDoesNotPersistRawSecrets(t *testing.T) {
	ctx := context.Background()
	dbPath := t.TempDir() + "/secrets.db"
	store, err := sqlite.Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}

	input := strings.NewReader(`{"log":{"entries":[{
		"startedDateTime":"2026-01-02T03:04:05Z",
		"time":15,
		"request":{"method":"GET","url":"https://example.com/api/items","headers":[
			{"name":"Authorization","value":"Bearer test-token-value"},
			{"name":"Cookie","value":"session_id=fake-session-value"},
			{"name":"X-API-Key","value":"example-api-key"}
		]},
		"response":{"status":200,"headers":[
			{"name":"Set-Cookie","value":"session_id=fake-session-value; Path=/; HttpOnly"}
		]}
	}]}}`)
	analysis, err := app.NewImportAnalysis(har.NewImporter(1024*1024), store).Execute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if analysis.SessionCount != 4 {
		t.Fatalf("session count = %d", analysis.SessionCount)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	contents := string(data)
	for _, secret := range []string{"test-token-value", "fake-session-value", "example-api-key"} {
		if strings.Contains(contents, secret) {
			t.Fatalf("database contains raw secret %q", secret)
		}
	}
}
