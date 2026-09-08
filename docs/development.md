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

## Docker

```sh
docker compose up --build
```

The API stores SQLite data in `./data`.
