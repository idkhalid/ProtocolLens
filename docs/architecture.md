# Architecture

ProtocolLens is a Go application with a React visualization layer. The backend owns import, normalization, analysis, persistence, and HTTP APIs. The frontend uploads files and displays API results.

## Backend Boundaries

`internal/domain` contains stable product concepts: analyses, HTTP exchanges, and endpoint summaries. It does not import HAR, SQLite, API, or frontend code.

`internal/capture/har` reads HAR 1.2 JSON into HAR-specific structs and applies basic input limits.

`internal/normalize` converts captured data into `domain.Exchange`, including URL parsing, header casing, query extraction, timestamp handling, and body decoding.

`internal/analyzer` runs explicit analyzers over normalized domain data. The first implemented analyzer aggregates endpoints by method, host, and path.

`internal/storage/sqlite` persists analyses, exchanges, and endpoint summaries with `database/sql`.

`internal/app` is the shared application layer used by both the CLI and HTTP API.

`internal/api` exposes the v1 HTTP endpoints with `net/http`.

## Frontend

`web` is a Vite React app. It uploads HAR files and displays analysis summaries and endpoints returned by the Go API. It does not parse HAR or infer protocol behavior.

## Deferred

Replay, generation, dependency inference, session analysis, workflow graphs, Playwright import, WebSocket/SSE analysis, and authentication are intentionally absent until a real slice needs them.
