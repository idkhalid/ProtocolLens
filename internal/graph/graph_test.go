package graph

import (
	"errors"
	"strings"
	"testing"
	"time"

	"protocollens/internal/domain"
)

func TestBuilderCreatesNodesAndDependencyEdges(t *testing.T) {
	exchanges := []domain.Exchange{
		exchange("req-1", 1, "GET", "https://example.com/api/bootstrap", 200),
		exchange("req-2", 2, "GET", "https://example.com/api/items?cursor=raw_value", 201),
		exchange("req-3", 3, "POST", "https://example.com/api/alone?secret=raw_value", 204),
	}
	dependencies := []domain.Dependency{{
		ID: "dep-1", SourceRequestID: "req-1", TargetRequestID: "req-2", SourcePath: "$.cursor",
		TargetLocation: "query", TargetPath: "cursor", Confidence: domain.DependencyConfidenceHigh, Reason: "response_json_to_query",
	}}

	got, err := Builder{}.Build(exchanges, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Nodes) != 3 || len(got.Edges) != 1 {
		t.Fatalf("graph = %#v", got)
	}
	if got.Nodes[1].ID != "req-2" || got.Nodes[1].Order != 1 || got.Nodes[1].Method != "GET" || got.Nodes[1].Host != "example.com" || got.Nodes[1].Path != "/api/items" || got.Nodes[1].StatusCode != 201 {
		t.Fatalf("node = %#v", got.Nodes[1])
	}
	if got.Edges[0].ID != "dep-1" || got.Edges[0].Source != "req-1" || got.Edges[0].Target != "req-2" || got.Edges[0].Type != EdgeTypeDataDependency || got.Edges[0].TargetPath != "cursor" {
		t.Fatalf("edge = %#v", got.Edges[0])
	}
	assertGraphDoesNotContain(t, got, "raw_value")
}

func TestBuilderOrdersNodesWithT3Semantics(t *testing.T) {
	equal := []domain.Exchange{
		exchange("req-2", 1, "GET", "https://example.com/two", 200),
		exchange("req-1", 1, "GET", "https://example.com/one", 200),
		exchange("req-0", 0, "GET", "https://example.com/zero", 200),
	}
	got, err := Builder{}.Build(equal, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertNodeOrder(t, got, "req-0", "req-2", "req-1")

	missing := []domain.Exchange{
		{Request: domain.Request{ID: "req-a", Method: "GET", URL: "https://example.com/a"}},
		{Request: domain.Request{ID: "req-b", Method: "GET", URL: "https://example.com/b"}},
	}
	got, err = Builder{}.Build(missing, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertNodeOrder(t, got, "req-a", "req-b")
}

func TestBuilderReportsMissingDependencyReferences(t *testing.T) {
	exchanges := []domain.Exchange{exchange("req-1", 1, "GET", "https://example.com/api/bootstrap", 200)}
	dependencies := []domain.Dependency{
		{ID: "dep-missing-source", SourceRequestID: "missing-source", TargetRequestID: "req-1", SourcePath: "$.id", TargetLocation: "path", TargetPath: "/items/{value}", Confidence: domain.DependencyConfidenceHigh, Reason: "response_json_to_path"},
		{ID: "dep-missing-target", SourceRequestID: "req-1", TargetRequestID: "missing-target", SourcePath: "$.id", TargetLocation: "path", TargetPath: "/items/{value}", Confidence: domain.DependencyConfidenceHigh, Reason: "response_json_to_path"},
	}

	got, err := Builder{}.Build(exchanges, dependencies)
	if len(got.Nodes) != 1 || len(got.Edges) != 0 {
		t.Fatalf("graph = %#v", got)
	}
	var refErr ReferenceError
	if !errors.As(err, &refErr) || len(refErr.Missing) != 2 {
		t.Fatalf("err = %#v", err)
	}
}

func TestBuilderSuppressesDuplicateEdges(t *testing.T) {
	exchanges := []domain.Exchange{
		exchange("req-1", 1, "GET", "https://example.com/api/source", 200),
		exchange("req-2", 2, "GET", "https://example.com/api/items/{value}", 200),
	}
	dependency := domain.Dependency{ID: "dep-1", SourceRequestID: "req-1", TargetRequestID: "req-2", SourcePath: "$.id", TargetLocation: "path", TargetPath: "/api/items/{value}", Confidence: domain.DependencyConfidenceHigh, Reason: "response_json_to_path"}

	got, err := Builder{}.Build(exchanges, []domain.Dependency{dependency, dependency})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Edges) != 1 {
		t.Fatalf("edges = %#v", got.Edges)
	}
}

func assertNodeOrder(t *testing.T, graph Graph, ids ...string) {
	t.Helper()
	if len(graph.Nodes) != len(ids) {
		t.Fatalf("nodes = %#v", graph.Nodes)
	}
	for n, id := range ids {
		if graph.Nodes[n].RequestID != id || graph.Nodes[n].Order != n {
			t.Fatalf("node %d = %#v", n, graph.Nodes[n])
		}
	}
}

func assertGraphDoesNotContain(t *testing.T, graph Graph, value string) {
	t.Helper()
	for _, node := range graph.Nodes {
		if strings.Contains(node.Path, value) || strings.Contains(node.Host, value) {
			t.Fatalf("node leaked %q: %#v", value, node)
		}
	}
	for _, edge := range graph.Edges {
		if strings.Contains(edge.SourcePath, value) || strings.Contains(edge.TargetPath, value) || strings.Contains(edge.TargetLocation, value) || strings.Contains(edge.Reason, value) {
			t.Fatalf("edge leaked %q: %#v", value, edge)
		}
	}
}

func exchange(id string, second int, method, url string, status int) domain.Exchange {
	return domain.Exchange{
		Request: domain.Request{
			ID:        id,
			Method:    method,
			URL:       url,
			Timestamp: time.Date(2026, 1, 2, 3, 4, second, 0, time.UTC),
		},
		Response: domain.Response{StatusCode: status, Duration: time.Duration(second) * time.Millisecond},
	}
}

func TestBuilderMergesPathRedactionForSharedTarget(t *testing.T) {
	exchanges := []domain.Exchange{
		exchange("req-1", 1, "GET", "https://example.com/source-a", 200),
		exchange("req-2", 2, "GET", "https://example.com/source-b", 200),
		exchange("req-3", 3, "GET", "https://example.com/api/items/secret_a/secret_b", 200),
	}
	dependencies := []domain.Dependency{
		{ID: "dep-a", SourceRequestID: "req-1", TargetRequestID: "req-3", SourcePath: "$.a", TargetLocation: "path", TargetPath: "/api/items/{value}/secret_b", Confidence: domain.DependencyConfidenceHigh, Reason: "response_json_to_path"},
		{ID: "dep-b", SourceRequestID: "req-2", TargetRequestID: "req-3", SourcePath: "$.b", TargetLocation: "path", TargetPath: "/api/items/secret_a/{value}", Confidence: domain.DependencyConfidenceHigh, Reason: "response_json_to_path"},
	}

	got, err := Builder{}.Build(exchanges, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	if got.Nodes[2].Path != "/api/items/{value}/{value}" {
		t.Fatalf("target node path = %q", got.Nodes[2].Path)
	}
	assertGraphDoesNotContain(t, got, "secret_a")
	assertGraphDoesNotContain(t, got, "secret_b")
}
