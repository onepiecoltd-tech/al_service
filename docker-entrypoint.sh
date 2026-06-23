#!/bin/sh
set -e

# Apply pending DB migrations, then start the API. The DB runs in a separate
# compose project, so retry a few times in case it isn't accepting
# connections the instant this container starts.
n=0
until goose -dir /app/db/migrations postgres "$DATABASE_URL" up; do
  n=$((n + 1))
  if [ "$n" -ge 10 ]; then
    echo "migrations failed after $n attempts — is the database reachable?" >&2
    exit 1
  fi
  echo "database not ready, retrying migrations ($n)…" >&2
  sleep 3
done

exec /app/api
