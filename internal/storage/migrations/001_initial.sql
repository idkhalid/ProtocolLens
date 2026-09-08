CREATE TABLE IF NOT EXISTS analyses (
	id TEXT PRIMARY KEY,
	created_at TEXT NOT NULL,
	request_count INTEGER NOT NULL,
	endpoint_count INTEGER NOT NULL,
	import_duration_ms INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS exchanges (
	id TEXT NOT NULL,
	analysis_id TEXT NOT NULL,
	request_json TEXT NOT NULL,
	response_json TEXT NOT NULL,
	PRIMARY KEY (analysis_id, id),
	FOREIGN KEY (analysis_id) REFERENCES analyses(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS endpoints (
	analysis_id TEXT NOT NULL,
	method TEXT NOT NULL,
	host TEXT NOT NULL,
	path TEXT NOT NULL,
	request_count INTEGER NOT NULL,
	status_codes_json TEXT NOT NULL,
	average_duration_ms REAL NOT NULL,
	min_duration_ms REAL NOT NULL,
	max_duration_ms REAL NOT NULL,
	content_types_json TEXT NOT NULL,
	PRIMARY KEY (analysis_id, method, host, path),
	FOREIGN KEY (analysis_id) REFERENCES analyses(id) ON DELETE CASCADE
);