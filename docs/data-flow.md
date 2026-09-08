# Data Flow

```text
HAR file
  |
  v
HAR Importer
  |
  v
Normalizer
  |
  v
domain.Exchange
  |
  v
Endpoint Analyzer
  |
  +--> Session Analyzer
  |
  v
SQLite
  |
  +--> CLI summary
  |
  v
HTTP API
  |
  v
React UI
```

Endpoint normalization keeps method, host, and path. Query values are retained on the request but ignored for endpoint grouping, so `/api/items?page=1` and `/api/items?page=2` map to the same endpoint.

Sensitive headers are normalized into safe metadata before analysis and storage. Cookie names and Set-Cookie attributes may be retained; token and cookie values are discarded.
