# Architecture

ProtocolLens is a Go application with a React visualization layer. The backend owns import, normalization, analysis, persistence, graph construction, and HTTP APIs. The frontend uploads files, starts local-only captures when enabled, and displays API results.

## Backend Boundaries

`internal/domain` contains stable product concepts: analyses, HTTP exchanges, endpoint summaries, session artifacts, and dependencies. It does not import HAR, SQLite, API, or frontend code.

`internal/capture/har` reads HAR 1.2 JSON into HAR-specific structs and applies basic input limits.

`internal/capture/playwright` owns the shared T7A/T7B browser capture lifecycle: initial URL validation through the replay destination policy, Node invocation, bounded duration, process-tree cancellation, temporary HAR cleanup, and adapter availability checks.

`internal/normalize` converts captured data into `domain.Exchange`, including URL parsing, header casing, query extraction, timestamp handling, safe sensitive header metadata, and body decoding.

`internal/analyzer` runs explicit analyzers over normalized domain data. Endpoint analysis aggregates method, host, and path. Session analysis detects safe metadata for cookies, bearer authorization, CSRF headers, and explicit API key headers. Dependency analysis performs deterministic observable data-flow inference from earlier response JSON scalar values into later request paths, query parameters, JSON bodies, and form bodies.

`internal/storage/sqlite` persists analyses, exchanges, endpoint summaries, session artifacts, and dependencies with `database/sql`. Workflow graphs are derived from existing exchanges and dependencies, not stored as duplicated graph JSON.

`internal/graph` builds deterministic workflow graphs from captured request instances and dependency records. It owns graph construction rules without React Flow concepts.

`internal/app` is the shared application layer used by both the CLI and HTTP API. `LocalCapture` allows one active local capture and imports only completed HAR output.

`internal/api` exposes the v1 HTTP endpoints with `net/http`. `POST /api/v1/local/capture` is disabled by default, requires `PROTOCOLLENS_LOCAL_CAPTURE_ENABLED=true`, refuses non-loopback server binds, and rejects non-local Host/Origin/connection metadata before Chromium can launch. `GET /api/v1/capabilities` reports safe booleans only.

`internal/replay` owns safe HTTP replay execution. It validates schemes, ports, hostnames, resolved IP addresses, redirects, request size, response size, and response header redaction before returning an ephemeral result. Replay is disabled by default and is never persisted.

`internal/generator` creates reproducible HTTP client code (cURL, Python httpx, Go net/http) from safe replay templates, replacing sensitive secrets with environment variable lookups.

## Playwright Capture Adapter

The `browser` directory contains a standalone TypeScript project acting purely as a local-first Playwright browser traffic capture adapter. It relies entirely on the Go analysis pipeline for endpoint, session, and dependency analysis. It performs no network analysis itself.

Both `protocollens capture` and the T7B React Capture workbench call the same Go capture lifecycle. The CLI remains a local developer command. The workbench route remains local-only and disabled on normal public deployments, so the Go server does not require Playwright, Node, Chromium, or `browser/dist` for primary use.

Browser capture has a wider outbound trust surface than Safe Replay because page execution can load subresources. The initial navigation target still uses the shared destination policy; ProtocolLens does not provide a remote browser service.

## Frontend

`web` is a Vite React app. It uploads HAR files, optionally starts local browser capture through the guarded local API route, and displays analysis summaries, endpoints, session artifacts, dependencies, and workflow graphs returned by the Go API. It may calculate display coordinates, but it does not parse HAR, infer protocol behavior, reconstruct dependencies, or calculate confidence.

## Deferred

Session edges, cookie lifecycle graphs, WebSocket/SSE analysis, JWT/OAuth inference, header dependency inference, multipart replay reconstruction, and authentication are intentionally absent until a real slice needs them.