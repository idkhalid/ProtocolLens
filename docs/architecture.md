# Architecture

ProtocolLens is a Go application with a React visualization layer. The backend owns import, normalization, analysis, persistence, graph construction, and HTTP APIs. The frontend uploads files and displays API results.

## Backend Boundaries

`internal/domain` contains stable product concepts: analyses, HTTP exchanges, endpoint summaries, session artifacts, and dependencies. It does not import HAR, SQLite, API, or frontend code.

`internal/capture/har` reads HAR 1.2 JSON into HAR-specific structs and applies basic input limits.

`internal/normalize` converts captured data into `domain.Exchange`, including URL parsing, header casing, query extraction, timestamp handling, safe sensitive header metadata, and body decoding.

`internal/analyzer` runs explicit analyzers over normalized domain data. Endpoint analysis aggregates method, host, and path. Session analysis detects safe metadata for cookies, bearer authorization, CSRF headers, and explicit API key headers. Dependency analysis performs deterministic observable data-flow inference from earlier response JSON scalar values into later request paths, query parameters, JSON bodies, and form bodies.

`internal/storage/sqlite` persists analyses, exchanges, endpoint summaries, session artifacts, and dependencies with `database/sql`. Workflow graphs are derived from existing exchanges and dependencies, not stored as duplicated graph JSON.

`internal/graph` builds deterministic workflow graphs from captured request instances and dependency records. It owns graph construction rules without React Flow concepts.

`internal/app` is the shared application layer used by both the CLI and HTTP API.

`internal/api` exposes the v1 HTTP endpoints with `net/http`.

## Frontend

`web` is a Vite React app. It uploads HAR files and displays analysis summaries, endpoints, session artifacts, dependencies, and workflow graphs returned by the Go API. It may calculate display coordinates, but it does not parse HAR, infer protocol behavior, reconstruct dependencies, or calculate confidence.

## Deferred

Replay, generation, session edges, cookie lifecycle graphs, Playwright import, WebSocket/SSE analysis, JWT/OAuth inference, header dependency inference, multipart parsing, and authentication are intentionally absent until a real slice needs them.