#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 POSTGRES_URL" >&2
  exit 2
fi

exec go tool river migrate-up --database-url "$1" --line main
