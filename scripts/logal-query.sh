#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
source scripts/dev-env.sh
# SQLite WAL readers and the writer must use the same filesystem/kernel.
# Keep the database in Colima and query it there, never through a host bind mount.
command_name="${1:-logs}"
if [[ $# -gt 0 ]]; then shift; fi
exec docker compose -p plane -f docker-compose-local.yml exec -T logal \
    logal "$command_name" --db /data/otel.debug.sqlite "$@"
