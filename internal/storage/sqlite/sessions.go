package sqlite

import (
	"context"
	"encoding/json"
	"time"

	"protocollens/internal/domain"
)

func (s *Store) ListSessionArtifacts(ctx context.Context, analysisID string) ([]domain.SessionArtifact, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, analysis_id, type, name, source, first_request_id, first_seen_at, occurrences, metadata_json
FROM session_artifacts
WHERE analysis_id = ?
ORDER BY type, name, source`, analysisID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []domain.SessionArtifact
	for rows.Next() {
		var artifact domain.SessionArtifact
		var typ string
		var firstSeenAt string
		var metadataJSON string
		err := rows.Scan(
			&artifact.ID,
			&artifact.AnalysisID,
			&typ,
			&artifact.Name,
			&artifact.Source,
			&artifact.FirstRequestID,
			&firstSeenAt,
			&artifact.Occurrences,
			&metadataJSON,
		)
		if err != nil {
			return nil, err
		}
		artifact.Type = domain.SessionArtifactType(typ)
		artifact.FirstSeenAt, err = time.Parse(time.RFC3339Nano, firstSeenAt)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(metadataJSON), &artifact.Metadata); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, rows.Err()
}
