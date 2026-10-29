#!/usr/bin/env bash
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
export NODE_COMPILE_CACHE="$ROOT/node_modules/.cache/frontend-node"

# Keep each shard within the process guard without adding concurrent workers.
# Node partitions the complete file list; a failed shard still fails the suite.
failed=0
for shard in 1 2; do
  printf '\n▶ Frontend behavior tests (shard %s/2)\n' "$shard"
  node scripts/quality/run-bounded.mjs --seconds 300 --memory-mib 1024 --heap-mib 384 -- \
    node --test --test-timeout=15000 --test-concurrency=2 --test-shard="$shard/2" \
    tests/frontend/*.test.mjs || failed=1
done
exit "$failed"
