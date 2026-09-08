ALTER TABLE analyses ADD COLUMN session_artifact_count INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS session_artifacts (
	id TEXT PRIMARY KEY,
	analysis_id TEXT NOT NULL,
	type TEXT NOT NULL,
	name TEXT NOT NULL,
	source TEXT NOT NULL,
	first_request_id TEXT NOT NULL,
	first_seen_at TEXT NOT NULL,
	occurrences INTEGER NOT NULL,
	metadata_json TEXT NOT NULL,
	FOREIGN KEY (analysis_id) REFERENCES analyses(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS session_artifacts_analysis_id_idx ON session_artifacts (analysis_id);
CREATE INDEX IF NOT EXISTS session_artifacts_analysis_type_idx ON session_artifacts (analysis_id, type);