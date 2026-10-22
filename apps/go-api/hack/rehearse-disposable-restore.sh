#!/bin/sh

set -eu

if [ "$#" -ne 2 ]; then
  echo "usage: rehearse-disposable-restore.sh SOURCE_POSTGRES_URL TARGET_POSTGRES_URL" >&2
  exit 2
fi

source_postgres_url=$1
target_postgres_url=$2
if [ "$source_postgres_url" = "$target_postgres_url" ]; then
  echo "source and target PostgreSQL URLs must be different" >&2
  exit 2
fi
umask 077
backup_file=$(mktemp "${TMPDIR:-/tmp}/plane-m1-restore.XXXXXX")
trap 'rm -f "$backup_file"' EXIT HUP INT TERM

dump_started=$(date +%s)
if [ -n "${PLANE_POSTGRES_CONTAINER:-}" ]; then
  docker exec "$PLANE_POSTGRES_CONTAINER" pg_dump \
    --format=custom \
    --no-owner \
    --no-privileges \
    "$source_postgres_url" >"$backup_file"
else
  pg_dump \
    --format=custom \
    --no-owner \
    --no-privileges \
    --file="$backup_file" \
    "$source_postgres_url"
fi
dump_finished=$(date +%s)

restore_started=$(date +%s)
if [ -n "${PLANE_POSTGRES_CONTAINER:-}" ]; then
  docker exec -i "$PLANE_POSTGRES_CONTAINER" pg_restore \
    --clean \
    --if-exists \
    --no-owner \
    --no-privileges \
    --exit-on-error \
    --dbname="$target_postgres_url" <"$backup_file"
else
  pg_restore \
    --clean \
    --if-exists \
    --no-owner \
    --no-privileges \
    --exit-on-error \
    --dbname="$target_postgres_url" \
    "$backup_file"
fi
restore_finished=$(date +%s)

if backup_bytes=$(stat -f '%z' "$backup_file" 2>/dev/null); then
  :
else
  backup_bytes=$(stat -c '%s' "$backup_file")
fi

echo "profile=disposable-m1"
echo "dump_seconds=$((dump_finished - dump_started))"
echo "restore_seconds=$((restore_finished - restore_started))"
echo "backup_bytes=$backup_bytes"
