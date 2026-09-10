# Development

## Backend

```sh
go test ./...
go vet ./...
go build ./...
go run ./cmd/protocollens-server
```

## CLI

```sh
go run ./cmd/protocollens analyze examples/har/simple.har
go run ./cmd/protocollens version
```

Release builds can inject the release version:

```sh
go build -ldflags "-X protocollens/internal/version.Version=v0.1.0" -o protocollens ./cmd/protocollens
go build -ldflags "-X protocollens/internal/version.Version=v0.1.0" -o protocollens-server ./cmd/protocollens-server
```

## Frontend

```sh
cd web
npm ci
npm run dev
npm run lint
npm run typecheck
npm run build
```

The frontend expects `VITE_API_URL`, defaulting to `http://localhost:8080`.

## Browser Capture Adapter (Optional)

```sh
cd browser
npm ci
npm run typecheck
npm run build
npx playwright install chromium
```

The compiled `browser/dist/capture.js` must exist before CLI or workbench capture can run. Normal API server use does not require Node, Playwright, Chromium, or `browser/dist`.

## Docker

```sh
docker compose up --build
```

The API stores SQLite data in `./data` by default.

## Configuration

| Variable | Default | Notes |
| --- | --- | --- |
| `PROTOCOLLENS_ADDR` | `:8080` | Server bind address. Local capture requires a loopback bind when enabled. |
| `PROTOCOLLENS_DATABASE_PATH` | `data/protocollens.db` | SQLite database path. |
| `PROTOCOLLENS_DATA_DIR` | `data` | Reserved data directory setting. |
| `PROTOCOLLENS_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error`. |
| `PROTOCOLLENS_MAX_UPLOAD_SIZE` | `52428800` | HAR upload/read limit in bytes. |
| `PROTOCOLLENS_REPLAY_ENABLED` | `false` | Safe Replay and Benchmark network execution gate. |
| `PROTOCOLLENS_REPLAY_ALLOWED_PORTS` | `80,443` | Comma-separated outbound replay ports. |
| `PROTOCOLLENS_REPLAY_TIMEOUT` | `10s` | Replay request timeout, capped at `60s`. |
| `PROTOCOLLENS_REPLAY_MAX_REQUEST_SIZE` | `1048576` | Replay/benchmark request body limit. |
| `PROTOCOLLENS_REPLAY_MAX_RESPONSE_SIZE` | `2097152` | Replay response body read limit. |
| `PROTOCOLLENS_REPLAY_MAX_CONCURRENT` | `4` | Global active replay request limit. |
| `PROTOCOLLENS_LOCAL_CAPTURE_ENABLED` | `false` | Local Playwright capture gate. |

Malformed numeric, duration, port, and boolean values fall back to safe defaults. Network side effects are disabled by default.

## HTTP Replay Development

Replay is disabled by default:

```sh
PROTOCOLLENS_REPLAY_ENABLED=false
```

For local development against allowed public destinations:

```sh
PROTOCOLLENS_REPLAY_ENABLED=true go run ./cmd/protocollens-server
```

Public deployments should evaluate replay carefully. Even with SSRF checks and response limits, enabling replay on a public instance can create an outbound HTTP relay surface.
