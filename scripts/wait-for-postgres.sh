#!/usr/bin/env bash
set -euo pipefail

host="${POSTGRES_HOST:-localhost}"
port="${POSTGRES_PORT:-5432}"
user="${POSTGRES_USER:-rms}"
db="${POSTGRES_DB:-rms}"

echo "Waiting for PostgreSQL at ${host}:${port}..."
for i in {1..60}; do
  if pg_isready -h "$host" -p "$port" -U "$user" -d "$db" >/dev/null 2>&1; then
    echo "PostgreSQL is ready."
    exit 0
  fi
  sleep 1
done

echo "Timed out waiting for PostgreSQL."
exit 1
