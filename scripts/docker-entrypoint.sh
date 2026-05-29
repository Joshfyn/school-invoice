#!/bin/sh
set -e

if [ "${RUN_MIGRATIONS:-true}" = "true" ]; then
  echo "Running database migrations..."
  migrate -path /app/migrations -database "${DATABASE_URL}" up
fi

echo "Starting API server..."
exec ./main
