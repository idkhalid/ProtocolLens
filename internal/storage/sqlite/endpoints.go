package sqlite

import (
	"context"
	"encoding/json"

	"protocollens/internal/domain"
)

func (s *Store) ListEndpoints(ctx context.Context, analysisID string) ([]domain.EndpointSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT method, host, path, request_count, status_codes_json,
	average_duration_ms, min_duration_ms, max_duration_ms, content_types_json
FROM endpoints
WHERE analysis_id = ?
ORDER BY host, path, method`, analysisID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []domain.EndpointSummary
	for rows.Next() {
		var endpoint domain.EndpointSummary
		var statusCodesJSON string
		var contentTypesJSON string
		err := rows.Scan(
			&endpoint.Method,
			&endpoint.Host,
			&endpoint.Path,
			&endpoint.RequestCount,
			&statusCodesJSON,
			&endpoint.AverageDurationMs,
			&endpoint.MinDurationMs,
			&endpoint.MaxDurationMs,
			&contentTypesJSON,
		)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(statusCodesJSON), &endpoint.StatusCodes); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(contentTypesJSON), &endpoint.ContentTypes); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, endpoint)
	}
	return endpoints, rows.Err()
}
