# Data Flow

```text
URL
  |
  v
Playwright Capture Adapter
  |
  v
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
  +--> Dependency Analyzer
  |
  v
SQLite
  |
  +--> CLI summary
  |
  +--> Workflow Graph Builder
  |
  +--> Safe Replay Template
  |       |
  |       +--> Replay Executor
  |       |
  |       +--> Client Generator
  |
  v
HTTP API
  |
  v
React UI
```

Endpoint normalization keeps method, host, and path. Query values are retained on the request but ignored for endpoint grouping, so `/api/items?page=1` and `/api/items?page=2` map to the same endpoint.

Sensitive headers are normalized into safe metadata before analysis and storage. Cookie names and Set-Cookie attributes may be retained; token and cookie values are discarded.

Dependency inference compares scalar values from earlier JSON responses against later request path segments, query values, JSON request body scalars, and form body values. Dependency records store request IDs, source JSON paths, target locations, target paths, confidence, and reason codes, not the matched values.

Workflow graph construction loads exchanges and dependencies, creates one node per captured request, and creates data-dependency edges from dependency records only. Node order follows the same deterministic timestamp and stable capture-order semantics as dependency inference. The workflow graph represents observable client-side request relationships, not the application's complete internal execution graph.
## Replay flow

```text
Captured request
  |
  v
Safe replay template
  |
  v
User review / edit
  |
  v
POST /api/v1/replay
  |
  v
Replay executor
  |
  v
Destination policy + safe transport
  |
  v
Limited response inspector
```

Replay templates are derived from normalized exchanges and do not execute network traffic. Replay execution is explicit, disabled by default, SSRF-checked during dialing and redirects, size-limited, and not written to SQLite.

## Client generation flow

```text
Captured request
  |
  v
Safe replay template
  |
  v
GenerateClient
  |
  +--> cURL
  |
  +--> Python httpx
  |
  +--> Go net/http
  |
  v
Generated client code
```

Client generation reuses the same safe replay template used by HTTP Replay. Generators do not read raw HAR data directly. Discarded secrets are not recovered. Generation performs no outbound network requests. Generated output is not persisted. Sensitive values are represented through deterministic environment-variable references. Generation logic lives in the Go backend. React only selects the target, displays generated code, and handles copy interaction.