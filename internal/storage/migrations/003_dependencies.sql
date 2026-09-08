ALTER TABLE analyses ADD COLUMN dependency_count INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS dependencies (
	id TEXT PRIMARY KEY,
	analysis_id TEXT NOT NULL,
	source_request_id TEXT NOT NULL,
	target_request_id TEXT NOT NULL,
	source_path TEXT NOT NULL,
	target_location TEXT NOT NULL,
	target_path TEXT NOT NULL,
	confidence TEXT NOT NULL,
	reason TEXT NOT NULL,
	FOREIGN KEY (analysis_id) REFERENCES analyses(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS dependencies_unique_evidence_idx
ON dependencies (analysis_id, source_request_id, target_request_id, source_path, target_location, target_path);