package sqlite

import (
	"context"
	"encoding/json"

	"protocollens/internal/domain"
)

func (s *Store) ListExchanges(ctx context.Context, analysisID string) ([]domain.Exchange, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT request_json, response_json
FROM exchanges
WHERE analysis_id = ?
ORDER BY id`, analysisID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exchanges []domain.Exchange
	for rows.Next() {
		var exchange domain.Exchange
		var requestJSON string
		var responseJSON string
		if err := rows.Scan(&requestJSON, &responseJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(requestJSON), &exchange.Request); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(responseJSON), &exchange.Response); err != nil {
			return nil, err
		}
		exchanges = append(exchanges, exchange)
	}
	return exchanges, rows.Err()
}
