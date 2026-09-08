package sqlite

import (
	"context"

	"protocollens/internal/domain"
)

func (s *Store) ListDependencies(ctx context.Context, analysisID string) ([]domain.Dependency, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, analysis_id, source_request_id, target_request_id, source_path,
	target_location, target_path, confidence, reason
FROM dependencies
WHERE analysis_id = ?
ORDER BY source_request_id, target_request_id, source_path, target_location, target_path`, analysisID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dependencies []domain.Dependency
	for rows.Next() {
		var dependency domain.Dependency
		var confidence string
		err := rows.Scan(
			&dependency.ID,
			&dependency.AnalysisID,
			&dependency.SourceRequestID,
			&dependency.TargetRequestID,
			&dependency.SourcePath,
			&dependency.TargetLocation,
			&dependency.TargetPath,
			&confidence,
			&dependency.Reason,
		)
		if err != nil {
			return nil, err
		}
		dependency.Confidence = domain.DependencyConfidence(confidence)
		dependencies = append(dependencies, dependency)
	}
	return dependencies, rows.Err()
}
