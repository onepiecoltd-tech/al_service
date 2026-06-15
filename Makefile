run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test -race ./...

test/cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	golangci-lint run ./...

migrate/up:
	goose -dir db/migrations postgres "$(DATABASE_URL)" up

migrate/down:
	goose -dir db/migrations postgres "$(DATABASE_URL)" down

tidy:
	go mod tidy

.PHONY: run build test test/cover lint migrate/up migrate/down tidy
