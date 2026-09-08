# ProtocolLens

ProtocolLens is a Go-first developer tool for analyzing observable client-side web workflows from browser traffic. The current milestone imports HAR files, normalizes HTTP exchanges, aggregates endpoints, stores results in SQLite, and exposes them through a CLI, HTTP API, and thin React UI.

## Current Capabilities

- Import HAR 1.2 JSON
- Normalize requests and responses into Go domain models
- Aggregate endpoints by method, host, and path
- Persist analyses, exchanges, and endpoints in SQLite
- Analyze HAR files from the CLI
- Import and inspect analyses through the HTTP API
- Upload HAR files and view endpoint summaries in React

## Architecture

The core product is Go. React is only a presentation layer.

```text
HAR -> Go importer -> normalizer -> endpoint analyzer -> SQLite -> CLI/API -> React
```

See `docs/architecture.md` and `docs/data-flow.md`.

## Quick Start

```sh
go run ./cmd/protocollens analyze examples/har/simple.har
go run ./cmd/protocollens-server
```

In another shell:

```sh
cd web
npm install
npm run dev
```

## CLI Usage

```sh
protocollens analyze file.har
```

The command prints the analysis ID, request count, endpoint count, and import duration.

## API Usage

```sh
curl -X POST --data-binary @examples/har/simple.har http://localhost:8080/api/v1/import/har
curl http://localhost:8080/api/v1/analyses/{id}
curl http://localhost:8080/api/v1/analyses/{id}/endpoints
```

## Development

```sh
go test ./...
go vet ./...
go build ./...
```

Frontend:

```sh
cd web
npm run lint
npm run typecheck
npm run build
```

Docker:

```sh
docker compose up --build
```

## Security Boundaries

Imported HAR files are untrusted input and are size-limited. Request and response bodies are trimmed before normalization. Sensitive headers such as `Authorization`, `Cookie`, and `Set-Cookie` are redacted during normalization. Replay, code generation, authentication bypass, browser impersonation, CAPTCHA solving, and token validation are not implemented in this milestone.

Captured credentials may exist inside HAR input. Do not import sensitive production traffic unless it has been sanitized.

## Project Status

Initial vertical slice. The next useful backend additions are session artifact detection, dependency inference, and explicit replay with redaction.

## License

No license has been selected yet.
