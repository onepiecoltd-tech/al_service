# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Go REST API backend for **craftbyte/learning_languages**. PostgreSQL database, `net/http` standard library (no framework).

## Commands

```bash
make run          # start dev server (requires .env or DATABASE_URL set)
make build        # compile to bin/api
make test         # run all tests with race detector
make lint         # golangci-lint
make migrate/up   # apply DB migrations (requires goose)
make migrate/down # rollback last migration
```

Copy `.env.example` to `.env` and fill in values before running.

## Architecture

```
cmd/api/          # entrypoint — wires config, server, graceful shutdown
internal/
  config/         # env-based config (no config files)
  server/         # HTTP server + route registration (routes.go)
  middleware/     # request ID, structured logging; use middleware.Chain()
  handler/        # HTTP handlers — one file per domain
  service/        # business logic — depends on repository interfaces
  repository/     # DB queries — one file per domain, interface + pgx impl
  model/          # shared domain structs (no ORM)
  apperror/       # typed errors with HTTP status codes
db/migrations/    # goose SQL migration files
pkg/
  httputil/       # JSON response helpers: OK(), Created(), Error()
  logger/         # slog init (text in dev, JSON in prod)
```

## Key Conventions

**Layer dependencies:** `handler → service → repository`. Handlers never touch the DB; services never write HTTP responses.

**Error handling:** return `*apperror.AppError` from service/repo layers; call `httputil.Error(w, err)` in handlers — it reads the HTTP status code automatically.

**Adding a new domain** (e.g. `user`):
1. Define model in `internal/model/`
2. Define repository interface + pgx impl in `internal/repository/`
3. Implement service in `internal/service/`
4. Write handler in `internal/handler/`, register routes in `internal/server/routes.go`
5. Add migration in `db/migrations/`

**Migrations:** use `goose` with plain SQL files. Naming: `YYYYMMDDHHMMSS_description.sql`.
