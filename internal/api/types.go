package api

import (
	"time"

	"protocollens/internal/domain"
)

type analysisResponse struct {
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	RequestCount   int       `json:"requestCount"`
	EndpointCount  int       `json:"endpointCount"`
	ImportDuration int64     `json:"importDurationMs"`
}

type endpointResponse struct {
	Method            string   `json:"method"`
	Host              string   `json:"host"`
	Path              string   `json:"path"`
	RequestCount      int      `json:"requestCount"`
	StatusCodes       []int    `json:"statusCodes"`
	AverageDurationMs float64  `json:"averageDurationMs"`
	MinDurationMs     float64  `json:"minDurationMs"`
	MaxDurationMs     float64  `json:"maxDurationMs"`
	ContentTypes      []string `json:"contentTypes"`
}

func toAnalysisResponse(analysis domain.Analysis) analysisResponse {
	return analysisResponse{
		ID:             analysis.ID,
		CreatedAt:      analysis.CreatedAt,
		RequestCount:   analysis.RequestCount,
		EndpointCount:  analysis.EndpointCount,
		ImportDuration: analysis.ImportDuration,
	}
}

func toAnalysisResponses(analyses []domain.Analysis) []analysisResponse {
	out := make([]analysisResponse, len(analyses))
	for n, analysis := range analyses {
		out[n] = toAnalysisResponse(analysis)
	}
	return out
}

func toEndpointResponses(endpoints []domain.EndpointSummary) []endpointResponse {
	out := make([]endpointResponse, len(endpoints))
	for n, endpoint := range endpoints {
		out[n] = endpointResponse{
			Method:            endpoint.Method,
			Host:              endpoint.Host,
			Path:              endpoint.Path,
			RequestCount:      endpoint.RequestCount,
			StatusCodes:       endpoint.StatusCodes,
			AverageDurationMs: endpoint.AverageDurationMs,
			MinDurationMs:     endpoint.MinDurationMs,
			MaxDurationMs:     endpoint.MaxDurationMs,
			ContentTypes:      endpoint.ContentTypes,
		}
	}
	return out
}
