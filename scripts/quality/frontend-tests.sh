#!/usr/bin/env bash
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
node_options=()
if [[ "${1:-}" == --coverage ]]; then
  export VECTOR_FRONTEND_COVERAGE="$ROOT/coverage/frontend"
  rm -rf -- "$VECTOR_FRONTEND_COVERAGE"
  mkdir -p "$VECTOR_FRONTEND_COVERAGE/raw" "$VECTOR_FRONTEND_COVERAGE/sources"
  export NODE_V8_COVERAGE="$VECTOR_FRONTEND_COVERAGE/raw"
  export NODE_DISABLE_COMPILE_CACHE=1
  unset NODE_COMPILE_CACHE
  node_options=(--import "$ROOT/tests/frontend/helpers/coverage-preload.mjs")
else
  export NODE_COMPILE_CACHE="$ROOT/node_modules/.cache/frontend-node"
fi

# Keep each shard within the process guard without adding concurrent workers.
# Node partitions the complete file list; a failed shard still fails the suite.
# Capture it once: adding a test between shards must not change their partition.
test_files=(tests/frontend/*.test.mjs apps/web/tests/*.test.mjs)
failed=0
for shard in 1 2; do
  printf '\n▶ Frontend behavior tests (shard %s/2)\n' "$shard"
  node scripts/quality/run-bounded.mjs --seconds 300 --memory-mib 1024 --heap-mib 384 -- \
    node "${node_options[@]}" --test --test-timeout=15000 --test-concurrency=2 --test-shard="$shard/2" \
    "${test_files[@]}" || failed=1
done
if [[ "${1:-}" == --coverage && "$failed" == 0 ]]; then
  node scripts/quality/run-bounded.mjs --seconds 600 --memory-mib 3072 --heap-mib 1024 -- \
    node scripts/quality/frontend-coverage.mjs --directory "$VECTOR_FRONTEND_COVERAGE" || failed=1
fi
exit "$failed"
