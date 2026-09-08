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
	SessionCount   int       `json:"sessionArtifactCount"`
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

type sessionArtifactsResponse struct {
	Artifacts []sessionArtifactResponse `json:"artifacts"`
}

type sessionArtifactResponse struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"`
	Name           string            `json:"name"`
	Source         string            `json:"source"`
	FirstRequestID string            `json:"firstRequestId"`
	FirstSeenAt    time.Time         `json:"firstSeenAt"`
	Occurrences    int               `json:"occurrences"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

func toAnalysisResponse(analysis domain.Analysis) analysisResponse {
	return analysisResponse{
		ID:             analysis.ID,
		CreatedAt:      analysis.CreatedAt,
		RequestCount:   analysis.RequestCount,
		EndpointCount:  analysis.EndpointCount,
		SessionCount:   analysis.SessionCount,
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

func toSessionArtifactsResponse(artifacts []domain.SessionArtifact) sessionArtifactsResponse {
	out := make([]sessionArtifactResponse, len(artifacts))
	for n, artifact := range artifacts {
		out[n] = sessionArtifactResponse{
			ID:             artifact.ID,
			Type:           string(artifact.Type),
			Name:           artifact.Name,
			Source:         artifact.Source,
			FirstRequestID: artifact.FirstRequestID,
			FirstSeenAt:    artifact.FirstSeenAt,
			Occurrences:    artifact.Occurrences,
			Metadata:       artifact.Metadata,
		}
	}
	return sessionArtifactsResponse{Artifacts: out}
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
