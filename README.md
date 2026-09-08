# ProtocolLens

ProtocolLens is a Go-first developer tool for analyzing observable client-side web workflows from browser traffic. The current milestone imports HAR files, normalizes HTTP exchanges, aggregates endpoints, detects session artifacts, infers deterministic request dependencies, stores results in SQLite, and exposes them through a CLI, HTTP API, and thin React UI.

## Current Capabilities

- Import HAR 1.2 JSON
- Normalize requests and responses into Go domain models
- Aggregate endpoints by method, host, and path
- Detect safe session artifact metadata for cookies, bearer authorization, CSRF headers, and API key headers
- Infer observable response JSON data reuse in later request paths, query values, JSON bodies, and form bodies
- Persist analyses, exchanges, endpoints, session artifacts, and dependencies in SQLite
- Analyze HAR files from the CLI
- Import and inspect analyses through the HTTP API
- Upload HAR files and view endpoint, session, and dependency summaries in React

## Architecture

The core product is Go. React is only a presentation layer.

```text
HAR -> Go importer -> normalizer -> endpoint/session/dependency analyzers -> SQLite -> CLI/API -> React
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

The command prints the analysis ID, request count, endpoint count, session artifact count, dependency count, and import duration.

## API Usage

```sh
curl -X POST --data-binary @examples/har/simple.har http://localhost:8080/api/v1/import/har
curl http://localhost:8080/api/v1/analyses/{id}
curl http://localhost:8080/api/v1/analyses/{id}/endpoints
curl http://localhost:8080/api/v1/analyses/{id}/sessions
curl http://localhost:8080/api/v1/analyses/{id}/dependencies
```

## Dependency Inference

Dependency inference is deterministic observable data-flow inference. It reports exact reuse of scalar values from earlier JSON responses in later request paths, query parameters, JSON request bodies, and `application/x-www-form-urlencoded` bodies.

It does not prove application semantics, inspect request headers, infer session dependencies, decode JWTs, parse HTML/XML, or analyze multipart bodies. Matched values are not stored in dependency records or returned by the dependency API.

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

Initial analysis baseline. The next useful backend additions are workflow graph assembly and explicit replay with redaction.

## License

No license has been selected yet.