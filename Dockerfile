# syntax=docker/dockerfile:1

# --- build stage ---
FROM golang:1.26-alpine AS build
WORKDIR /src

# goose (migration CLI) is installed here so the runtime image can apply
# migrations on startup without needing the Go toolchain.
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api

# --- runtime stage ---
FROM alpine:3.20
WORKDIR /app
# CA certs for outbound HTTPS (Gemini, Google token verification).
RUN apk add --no-cache ca-certificates

COPY --from=build /out/api /app/api
COPY --from=build /go/bin/goose /usr/local/bin/goose
COPY db/migrations /app/db/migrations
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

EXPOSE 8080
ENTRYPOINT ["/app/docker-entrypoint.sh"]
