package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"protocollens/internal/domain"
)

func (s *Store) SaveAnalysis(ctx context.Context, analysis domain.Analysis, exchanges []domain.Exchange, endpoints []domain.EndpointSummary) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
INSERT INTO analyses (id, created_at, request_count, endpoint_count, import_duration_ms)
VALUES (?, ?, ?, ?, ?)`,
		analysis.ID,
		analysis.CreatedAt.Format(time.RFC3339Nano),
		analysis.RequestCount,
		analysis.EndpointCount,
		analysis.ImportDuration,
	)
	if err != nil {
		return err
	}

	for _, exchange := range exchanges {
		requestJSON, err := json.Marshal(exchange.Request)
		if err != nil {
			return err
		}
		responseJSON, err := json.Marshal(exchange.Response)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO exchanges (id, analysis_id, request_json, response_json)
VALUES (?, ?, ?, ?)`,
			exchange.Request.ID,
			analysis.ID,
			string(requestJSON),
			string(responseJSON),
		)
		if err != nil {
			return err
		}
	}

	for _, endpoint := range endpoints {
		statusCodes, err := json.Marshal(endpoint.StatusCodes)
		if err != nil {
			return err
		}
		contentTypes, err := json.Marshal(endpoint.ContentTypes)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO endpoints (
	analysis_id, method, host, path, request_count, status_codes_json,
	average_duration_ms, min_duration_ms, max_duration_ms, content_types_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			analysis.ID,
			endpoint.Method,
			endpoint.Host,
			endpoint.Path,
			endpoint.RequestCount,
			string(statusCodes),
			endpoint.AverageDurationMs,
			endpoint.MinDurationMs,
			endpoint.MaxDurationMs,
			string(contentTypes),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) GetAnalysis(ctx context.Context, id string) (domain.Analysis, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, request_count, endpoint_count, import_duration_ms
FROM analyses
WHERE id = ?`, id)
	analysis, err := scanAnalysis(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Analysis{}, domain.ErrNotFound
	}
	return analysis, err
}

func (s *Store) ListAnalyses(ctx context.Context) ([]domain.Analysis, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, created_at, request_count, endpoint_count, import_duration_ms
FROM analyses
ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var analyses []domain.Analysis
	for rows.Next() {
		analysis, err := scanAnalysis(rows)
		if err != nil {
			return nil, err
		}
		analyses = append(analyses, analysis)
	}
	return analyses, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAnalysis(row scanner) (domain.Analysis, error) {
	var analysis domain.Analysis
	var createdAt string
	if err := row.Scan(&analysis.ID, &createdAt, &analysis.RequestCount, &analysis.EndpointCount, &analysis.ImportDuration); err != nil {
		return domain.Analysis{}, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return domain.Analysis{}, fmt.Errorf("parse created_at: %w", err)
	}
	analysis.CreatedAt = parsed
	return analysis, nil
}
