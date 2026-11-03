#!/usr/bin/env bash
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
if [[ "${1:-}" == --coverage ]]; then
  export VECTOR_FRONTEND_COVERAGE="$ROOT/coverage/frontend"
  rm -rf -- "$VECTOR_FRONTEND_COVERAGE"
  mkdir -p "$VECTOR_FRONTEND_COVERAGE/raw" "$VECTOR_FRONTEND_COVERAGE/sources"
fi

# Keep each shard within the process guard without adding concurrent workers.
# Capture it once: adding a test between shards must not change their partition.
test_files=(tests/frontend/*.test.mjs apps/web/tests/*.test.mjs)
shard_directory="$(mktemp -d "${TMPDIR:-/tmp}/vector-frontend-shards.XXXXXX")" || exit 1
trap 'rm -rf -- "$shard_directory"' EXIT
bun scripts/quality/frontend-test-shards.mjs "$shard_directory" "${test_files[@]}" || exit 1
failed=0
budget_seconds=600
budget_started=$SECONDS
for shard in 1 2 3; do
  shard_files=()
  while IFS= read -r -d '' file; do shard_files+=("$file"); done < "$shard_directory/$shard"
  if ((${#shard_files[@]} == 0)); then continue; fi
  printf '\n▶ Frontend behavior tests (shard %s/3)\n' "$shard"
  remaining=$((budget_seconds - SECONDS + budget_started))
  if ((remaining <= 0)); then
    printf '[frontend] cumulative test deadline exceeded %ss\n' "$budget_seconds" >&2
    failed=1
    break
  fi
  shard_seconds=$((remaining < 300 ? remaining : 300))
  bun scripts/quality/run-bounded.mjs --seconds "$shard_seconds" --memory-mib 1024 -- \
    bun scripts/quality/frontend-test-workers.mjs \
    "${shard_files[@]}" || { failed=1; break; }
done
if [[ "${1:-}" == --coverage && "$failed" == 0 ]]; then
  bun scripts/quality/run-bounded.mjs --seconds 600 --memory-mib 3072 -- \
    bun scripts/quality/frontend-coverage.mjs --directory "$VECTOR_FRONTEND_COVERAGE" || failed=1
fi
exit "$failed"
