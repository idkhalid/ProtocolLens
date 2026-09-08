package analyzer

import (
	"sort"

	"protocollens/internal/domain"
	"protocollens/internal/normalize"
)

type EndpointAnalyzer struct{}

func (EndpointAnalyzer) Analyze(exchanges []domain.Exchange) []domain.EndpointSummary {
	type aggregate struct {
		domain.Endpoint
		count        int
		totalMs      float64
		minMs        float64
		maxMs        float64
		statusCodes  map[int]bool
		contentTypes map[string]bool
	}

	byEndpoint := map[domain.Endpoint]*aggregate{}
	for _, exchange := range exchanges {
		endpoint, err := normalize.EndpointOf(exchange.Request)
		if err != nil {
			continue
		}
		durationMs := float64(exchange.Response.Duration.Microseconds()) / 1000
		agg := byEndpoint[endpoint]
		if agg == nil {
			agg = &aggregate{
				Endpoint:     endpoint,
				minMs:        durationMs,
				statusCodes:  map[int]bool{},
				contentTypes: map[string]bool{},
			}
			byEndpoint[endpoint] = agg
		}
		agg.count++
		agg.totalMs += durationMs
		if durationMs < agg.minMs {
			agg.minMs = durationMs
		}
		if durationMs > agg.maxMs {
			agg.maxMs = durationMs
		}
		if exchange.Response.StatusCode != 0 {
			agg.statusCodes[exchange.Response.StatusCode] = true
		}
		if contentType := exchange.Response.Headers["Content-Type"]; contentType != "" {
			agg.contentTypes[contentType] = true
		}
	}

	out := make([]domain.EndpointSummary, 0, len(byEndpoint))
	for _, agg := range byEndpoint {
		out = append(out, domain.EndpointSummary{
			Method:            agg.Method,
			Host:              agg.Host,
			Path:              agg.Path,
			RequestCount:      agg.count,
			StatusCodes:       ints(agg.statusCodes),
			AverageDurationMs: agg.totalMs / float64(agg.count),
			MinDurationMs:     agg.minMs,
			MaxDurationMs:     agg.maxMs,
			ContentTypes:      strings(agg.contentTypes),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Host != out[j].Host {
			return out[i].Host < out[j].Host
		}
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Method < out[j].Method
	})
	return out
}

func ints(values map[int]bool) []int {
	out := make([]int, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Ints(out)
	return out
}

func strings(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
