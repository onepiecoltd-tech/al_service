#!/bin/sh
set -e

# Apply pending DB migrations, then start the API. Postgres is already
# reachable here because compose gates this container on its healthcheck.
goose -dir /app/db/migrations postgres "$DATABASE_URL" up

exec /app/api
