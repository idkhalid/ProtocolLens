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
```

## Frontend

```sh
cd web
npm install
npm run dev
```

The frontend expects `VITE_API_URL`, defaulting to `http://localhost:8080`.

## Browser Capture Adapter (Optional)

```sh
cd browser
npm install
npm run typecheck
npm run build
npx playwright install chromium
```

The compiled `browser/dist/capture.js` must exist before the CLI capture command can be used.

## Docker

```sh
docker compose up --build
```

The API stores SQLite data in `./data`.

## HTTP replay development

Replay is disabled by default:

```sh
PROTOCOLLENS_REPLAY_ENABLED=false
```

For local development against allowed public destinations:

```sh
PROTOCOLLENS_REPLAY_ENABLED=true go run ./cmd/protocollens-server
```

Replay configuration:

```sh
PROTOCOLLENS_REPLAY_ALLOWED_PORTS=80,443
PROTOCOLLENS_REPLAY_TIMEOUT=10s
PROTOCOLLENS_REPLAY_MAX_REQUEST_SIZE=1048576
PROTOCOLLENS_REPLAY_MAX_RESPONSE_SIZE=2097152
PROTOCOLLENS_REPLAY_MAX_CONCURRENT=4
```

Public deployments should evaluate replay carefully. Even with SSRF checks and response limits, enabling replay on a public instance can create an outbound HTTP relay surface.