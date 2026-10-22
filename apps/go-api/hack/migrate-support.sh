#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 POSTGRES_URL" >&2
  exit 2
fi

exec go run ./cmd/support-migrate --database-url "$1" up
