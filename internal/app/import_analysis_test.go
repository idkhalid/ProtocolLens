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

func TestImportAnalysisDoesNotCopyDependencyValuesIntoEvidence(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, t.TempDir()+"/dependencies.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	input, err := os.Open("../../examples/har/dependency.har")
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()

	analysis, err := app.NewImportAnalysis(har.NewImporter(1024*1024), store).Execute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if analysis.RequestCount != 6 || analysis.EndpointCount != 6 || analysis.DependencyCount != 4 {
		t.Fatalf("analysis = %#v", analysis)
	}

	dependencies, err := store.ListDependencies(ctx, analysis.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(dependencies) != 4 {
		t.Fatalf("dependencies = %#v", dependencies)
	}
	for _, dependency := range dependencies {
		fields := strings.Join([]string{dependency.SourcePath, dependency.TargetLocation, dependency.TargetPath, dependency.Reason}, "\x00")
		for _, value := range []string{"item_12345", "next_abc123", "p_12345", "form_98765"} {
			if strings.Contains(fields, value) {
				t.Fatalf("dependency evidence leaked %q in %#v", value, dependency)
			}
		}
	}
}
