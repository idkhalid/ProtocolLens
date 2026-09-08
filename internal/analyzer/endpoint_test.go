package analyzer

import (
	"testing"
	"time"

	"protocollens/internal/domain"
)

func TestEndpointAnalyzerAggregatesByMethodHostPath(t *testing.T) {
	exchanges := []domain.Exchange{
		exchange("GET", "https://example.com/api/items?page=1", 200, 10*time.Millisecond),
		exchange("GET", "https://example.com/api/items?page=2", 201, 30*time.Millisecond),
		exchange("POST", "https://example.com/api/items", 204, 20*time.Millisecond),
	}

	got := EndpointAnalyzer{}.Analyze(exchanges)
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	first := got[0]
	if first.Method != "GET" || first.Path != "/api/items" || first.RequestCount != 2 {
		t.Fatalf("summary = %#v", first)
	}
	if first.AverageDurationMs != 20 || first.MinDurationMs != 10 || first.MaxDurationMs != 30 {
		t.Fatalf("durations = %#v", first)
	}
	if len(first.StatusCodes) != 2 || first.StatusCodes[0] != 200 || first.StatusCodes[1] != 201 {
		t.Fatalf("status codes = %#v", first.StatusCodes)
	}
}

func exchange(method, rawURL string, status int, duration time.Duration) domain.Exchange {
	return domain.Exchange{
		Request: domain.Request{
			Method: method,
			URL:    rawURL,
		},
		Response: domain.Response{
			StatusCode: status,
			Headers:    map[string][]string{"Content-Type": {"application/json"}},
			Duration:   duration,
		},
	}
}
