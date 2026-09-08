.PHONY: dev build test vet web

dev:
	docker compose up --build

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

web:
	cd web && npm run build
