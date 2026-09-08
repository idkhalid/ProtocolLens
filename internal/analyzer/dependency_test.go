package analyzer

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"protocollens/internal/domain"
)

func TestDependencyAnalyzerFindsPathQueryJSONAndFormTargets(t *testing.T) {
	exchanges := []domain.Exchange{
		{
			Request: domain.Request{ID: "req-1", Timestamp: ts(1), URL: "https://example.com/api/source"},
			Response: domain.Response{
				Headers: map[string][]string{"Content-Type": {"application/json"}},
				Body: []byte(`{
					"data": {"id": "item_12345", "cursor": "next_abc123", "product_id": "p_12345", "form_id": "form_98765", "short": "en", "status": 200},
					"items": [{"id": "array_12345"}],
					"unused": "does_not_match"
				}`),
			},
		},
		{Request: domain.Request{ID: "req-2", Timestamp: ts(2), URL: "https://example.com/api/items/item_12345"}},
		{Request: domain.Request{ID: "req-3", Timestamp: ts(3), URL: "https://example.com/api/items?cursor=next_abc123"}},
		{
			Request: domain.Request{
				ID:        "req-4",
				Timestamp: ts(4),
				URL:       "https://example.com/api/order",
				Headers:   map[string][]string{"Content-Type": {"application/json"}},
				Body:      []byte(`{"product":{"id":"p_12345"},"array":[{"id":"array_12345"}]}`),
			},
		},
		{
			Request: domain.Request{
				ID:        "req-5",
				Timestamp: ts(5),
				URL:       "https://example.com/api/form",
				Headers:   map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
				Body:      []byte("form_id=form_98765"),
			},
		},
	}

	got := DependencyAnalyzer{}.Analyze(exchanges)
	assertDependency(t, got, "req-1", "req-2", "$.data.id", "path", "/api/items/{value}", "response_json_to_path")
	assertDependency(t, got, "req-1", "req-3", "$.data.cursor", "query", "cursor", "response_json_to_query")
	assertDependency(t, got, "req-1", "req-4", "$.data.product_id", "body_json", "$.product.id", "response_json_to_json_body")
	assertDependency(t, got, "req-1", "req-4", "$.items[0].id", "body_json", "$.array[0].id", "response_json_to_json_body")
	assertDependency(t, got, "req-1", "req-5", "$.data.form_id", "body_form", "form_id", "response_json_to_form_body")
	assertNoTarget(t, got, "does_not_match")
	assertNoSourcePath(t, got, "$.data.short")
	assertNoSourcePath(t, got, "$.data.status")
	assertNoRawValues(t, got, "item_12345", "next_abc123", "p_12345", "form_98765", "array_12345")
}

func TestDependencyAnalyzerRespectsCaptureOrder(t *testing.T) {
	exchanges := []domain.Exchange{
		{Request: domain.Request{ID: "req-2", Timestamp: ts(2), URL: "https://example.com/api/items/later_12345"}},
		{Request: domain.Request{ID: "req-1", Timestamp: ts(1), URL: "https://example.com/api/source"}, Response: domain.Response{Body: []byte(`{"id":"later_12345"}`)}},
	}
	got := DependencyAnalyzer{}.Analyze(exchanges)
	assertDependency(t, got, "req-1", "req-2", "$.id", "path", "/api/items/{value}", "response_json_to_path")
}

func TestDependencyAnalyzerDoesNotInferBackwardsForEntryOrderTies(t *testing.T) {
	exchanges := []domain.Exchange{
		{Request: domain.Request{ID: "req-2", Timestamp: ts(1), URL: "https://example.com/api/items/tied_12345"}},
		{Request: domain.Request{ID: "req-1", Timestamp: ts(1), URL: "https://example.com/api/source"}, Response: domain.Response{Body: []byte(`{"id":"tied_12345"}`)}},
	}
	if got := (DependencyAnalyzer{}).Analyze(exchanges); len(got) != 0 {
		t.Fatalf("dependencies = %#v", got)
	}
}

func TestDependencyAnalyzerPreservesMissingTimestampEntryOrder(t *testing.T) {
	exchanges := []domain.Exchange{
		{Request: domain.Request{ID: "req-2", URL: "https://example.com/api/items/missing_12345"}},
		{Request: domain.Request{ID: "req-1", URL: "https://example.com/api/source"}, Response: domain.Response{Body: []byte(`{"id":"missing_12345"}`)}},
	}
	if got := (DependencyAnalyzer{}).Analyze(exchanges); len(got) != 0 {
		t.Fatalf("dependencies = %#v", got)
	}
}

func TestDependencyAnalyzerDeterministicTargetOrdering(t *testing.T) {
	exchanges := []domain.Exchange{
		{Request: domain.Request{ID: "req-1", Timestamp: ts(1), URL: "https://example.com/api/source"}, Response: domain.Response{Body: []byte(`{"id":"same_12345"}`)}},
		{Request: domain.Request{ID: "req-2", Timestamp: ts(2), URL: "https://example.com/api/items?b=same_12345&a=same_12345"}},
		{Request: domain.Request{ID: "req-3", Timestamp: ts(3), URL: "https://example.com/api/form", Headers: map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}}, Body: []byte("z=same_12345&a=same_12345")}},
	}

	want := "req-2:query:a\nreq-2:query:b\nreq-3:body_form:a\nreq-3:body_form:z\n"
	for n := 0; n < 20; n++ {
		if got := dependencyOrder(DependencyAnalyzer{}.Analyze(exchanges)); got != want {
			t.Fatalf("order run %d = %q, want %q", n, got, want)
		}
	}
}

func TestDependencyAnalyzerHandlesMissingContentTypeAndMalformedJSON(t *testing.T) {
	valid := DependencyAnalyzer{}.Analyze([]domain.Exchange{
		{Request: domain.Request{ID: "req-1", Timestamp: ts(1), URL: "https://example.com/api/source"}, Response: domain.Response{Body: []byte(`{"id":"missing_ct_123"}`)}},
		{Request: domain.Request{ID: "req-2", Timestamp: ts(2), URL: "https://example.com/api/items/missing_ct_123"}},
	})
	assertDependency(t, valid, "req-1", "req-2", "$.id", "path", "/api/items/{value}", "response_json_to_path")

	for _, body := range [][]byte{[]byte(`{"id"`), []byte(`{"id":"trailing_12345"} null`)} {
		invalid := DependencyAnalyzer{}.Analyze([]domain.Exchange{
			{Request: domain.Request{ID: "req-1", Timestamp: ts(1), URL: "https://example.com/api/source"}, Response: domain.Response{Body: body}},
			{Request: domain.Request{ID: "req-2", Timestamp: ts(2), URL: "https://example.com/api/items/trailing_12345"}},
		})
		if len(invalid) != 0 {
			t.Fatalf("dependencies = %#v", invalid)
		}
	}
}

func TestDependencyAnalyzerSuppressesGenericCandidates(t *testing.T) {
	got := DependencyAnalyzer{}.Analyze([]domain.Exchange{
		{Request: domain.Request{ID: "req-1", Timestamp: ts(1), URL: "https://example.com/api/source"}, Response: domain.Response{Body: []byte(`{
			"empty":"", "short":"abc", "zero":0, "one":1, "method":"POST", "status":404,
			"currency":"USD", "language":"en", "truth":true, "nothing":null
		}`)}},
		{Request: domain.Request{ID: "req-2", Timestamp: ts(2), URL: "https://example.com/api/abc/0/1/POST/404/USD/en"}},
	})
	if len(got) != 0 {
		t.Fatalf("dependencies = %#v", got)
	}
}

func TestDependencyAnalyzerSuppressesDuplicatesAndClassifiesConfidence(t *testing.T) {
	got := DependencyAnalyzer{}.Analyze([]domain.Exchange{
		{Request: domain.Request{ID: "req-1", Timestamp: ts(1), URL: "https://example.com/api/source"}, Response: domain.Response{Body: []byte(`{"a":"dup_12345","b":"dup_12345","specific":"unique_12345"}`)}},
		{Request: domain.Request{ID: "req-2", Timestamp: ts(2), URL: "https://example.com/api/items/dup_12345/unique_12345"}},
	})
	dup := assertDependency(t, got, "req-1", "req-2", "$.a", "path", "/api/items/{value}/{value}", "response_json_to_path")
	if dup.Confidence != domain.DependencyConfidenceMedium {
		t.Fatalf("dup confidence = %s", dup.Confidence)
	}
	unique := assertDependency(t, got, "req-1", "req-2", "$.specific", "path", "/api/items/{value}/{value}", "response_json_to_path")
	if unique.Confidence != domain.DependencyConfidenceHigh {
		t.Fatalf("unique confidence = %s", unique.Confidence)
	}
	assertNoRawValues(t, got, "dup_12345", "unique_12345")
}

func TestDependencyAnalyzerLimitsJSONTraversalAndDependencies(t *testing.T) {
	exact := bodyJSONScalars([]byte(jsonArray(maxScalarsPerBody)), nil)
	if len(exact) != maxScalarsPerBody {
		t.Fatalf("exact scalar count = %d", len(exact))
	}
	over := bodyJSONScalars([]byte(jsonArray(maxScalarsPerBody+1)), nil)
	if len(over) != maxScalarsPerBody {
		t.Fatalf("over scalar count = %d", len(over))
	}
	if got := bodyJSONScalars([]byte(nestedJSON(maxJSONDepth, "deep_12345")), nil); len(got) != 1 {
		t.Fatalf("exact depth scalars = %#v", got)
	}
	if got := bodyJSONScalars([]byte(nestedJSON(maxJSONDepth+1, "deep_12345")), nil); len(got) != 0 {
		t.Fatalf("over depth scalars = %#v", got)
	}

	var exchanges []domain.Exchange
	for source := 0; source < 5; source++ {
		exchanges = append(exchanges, domain.Exchange{
			Request:  domain.Request{ID: fmt.Sprintf("req-source-%d", source), Timestamp: ts(1 + source), URL: "https://example.com/api/source"},
			Response: domain.Response{Body: []byte(jsonObject(maxScalarsPerBody))},
		})
	}
	for n := 0; n < maxScalarsPerBody; n++ {
		value := fmt.Sprintf("cap_%05d", n)
		exchanges = append(exchanges, domain.Exchange{Request: domain.Request{ID: fmt.Sprintf("req-target-%05d", n), Timestamp: ts(100 + n), URL: "https://example.com/api/items/" + value}})
	}
	if got := (DependencyAnalyzer{}).Analyze(exchanges); len(got) != maxDependenciesPerAnalysis {
		t.Fatalf("dependency count = %d", len(got))
	}
}

func ts(second int) time.Time {
	return time.Date(2026, 1, 2, 3, 4, second, 0, time.UTC)
}

func assertDependency(t *testing.T, dependencies []domain.Dependency, source, target, sourcePath, targetLocation, targetPath, reason string) domain.Dependency {
	t.Helper()
	for _, dependency := range dependencies {
		if dependency.SourceRequestID == source && dependency.TargetRequestID == target && dependency.SourcePath == sourcePath && dependency.TargetLocation == targetLocation && dependency.TargetPath == targetPath && dependency.Reason == reason {
			return dependency
		}
	}
	t.Fatalf("missing dependency %s %s %s %s %s %s in %#v", source, target, sourcePath, targetLocation, targetPath, reason, dependencies)
	return domain.Dependency{}
}

func assertNoSourcePath(t *testing.T, dependencies []domain.Dependency, sourcePath string) {
	t.Helper()
	for _, dependency := range dependencies {
		if dependency.SourcePath == sourcePath {
			t.Fatalf("unexpected source path %s in %#v", sourcePath, dependencies)
		}
	}
}

func assertNoTarget(t *testing.T, dependencies []domain.Dependency, value string) {
	t.Helper()
	for _, dependency := range dependencies {
		if strings.Contains(dependency.TargetPath, value) {
			t.Fatalf("target path leaked %q in %#v", value, dependency)
		}
	}
}

func assertNoRawValues(t *testing.T, dependencies []domain.Dependency, values ...string) {
	t.Helper()
	for _, dependency := range dependencies {
		fields := []string{dependency.SourcePath, dependency.TargetLocation, dependency.TargetPath, dependency.Reason}
		for _, field := range fields {
			for _, value := range values {
				if strings.Contains(field, value) {
					t.Fatalf("dependency leaked %q in %#v", value, dependency)
				}
			}
		}
	}
}

func dependencyOrder(dependencies []domain.Dependency) string {
	var out strings.Builder
	for _, dependency := range dependencies {
		out.WriteString(dependency.TargetRequestID)
		out.WriteByte(':')
		out.WriteString(dependency.TargetLocation)
		out.WriteByte(':')
		out.WriteString(dependency.TargetPath)
		out.WriteByte('\n')
	}
	return out.String()
}

func jsonArray(count int) string {
	var b strings.Builder
	b.WriteByte('[')
	for n := 0; n < count; n++ {
		if n > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%q", fmt.Sprintf("arr_%05d", n))
	}
	b.WriteByte(']')
	return b.String()
}

func jsonObject(count int) string {
	var b strings.Builder
	b.WriteByte('{')
	for n := 0; n < count; n++ {
		if n > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "\"k%05d\":\"cap_%05d\"", n, n)
	}
	b.WriteByte('}')
	return b.String()
}

func nestedJSON(depth int, value string) string {
	out := fmt.Sprintf("%q", value)
	for n := 0; n < depth; n++ {
		out = `{"x":` + out + `}`
	}
	return out
}
