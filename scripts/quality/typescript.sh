#!/usr/bin/env bash
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

MODE="${1:---full}"

run_gate() {
  local name="$1"
  shift
  printf '\n▶ %s\n' "$name"
  if ! bun scripts/quality/run-bounded.mjs --seconds 600 --memory-mib 3072 -- "$@"; then
    printf '\nFailed gate: %s; stopping before later stages.\n' "$name" >&2
    exit 1
  fi
  return 0
}

case "$MODE" in
  --fast)
    run_gate "Oxlint syntax diagnostic ratchet" bun scripts/quality/check-oxlint-baseline.mjs fast
    run_gate "Oxfmt check" bun run oxfmt --check .
    run_gate "Test runner regression tests" bun test --timeout 15000 ./scripts/quality/run-bounded.test.mjs ./scripts/quality/frontend-tests.test.mjs ./scripts/quality/frontend-coverage.test.mjs
    run_gate "Quality-ratchet fixture tests" bun scripts/quality/test-ratchets.mjs
    ;;
  --full)
    run_gate "Generated route types" bash scripts/quality/route-types.sh
    run_gate "Oxlint type-aware rules and TypeScript diagnostics" env GOMEMLIMIT=1GiB GOMAXPROCS=2 bun scripts/quality/check-oxlint-baseline.mjs full
    run_gate "Oxfmt check" bun run oxfmt --check .
    run_gate "Test runner regression tests" bun test --timeout 15000 ./scripts/quality/run-bounded.test.mjs ./scripts/quality/frontend-tests.test.mjs ./scripts/quality/frontend-coverage.test.mjs
    run_gate "Quality-ratchet fixture tests" bun scripts/quality/test-ratchets.mjs
    run_gate "Knip unused-code/dependency ratchet" bun scripts/quality/check-knip-ratchet.mjs
    run_gate "jscpd duplicate baseline" bun run check:duplicates
    run_gate "Constants endpoint tests" bun run --filter @plane/constants test
    run_gate "Reset Live coverage artifact" rm -f -- apps/live/coverage/coverage-summary.json
    run_gate "Live tests with coverage" bun run --filter live test:coverage
    run_gate "Complete Live source inventory" bun scripts/quality/complete-coverage-scope.mjs apps/live
    run_gate "Live coverage ratchet" bun scripts/quality/check-coverage-ratchet.mjs --root apps/live --summary coverage/coverage-summary.json --baseline ../../.quality/coverage-live-baseline.json --scope-file .quality/coverage-scope.json --scope-key apps/live --new-file-min 80
    run_gate "Reset codemods coverage artifact" rm -f -- packages/codemods/coverage/coverage-summary.json
    run_gate "Codemods tests with coverage" bun run --filter @plane/codemods test:coverage
    run_gate "Complete Codemods source inventory" bun scripts/quality/complete-coverage-scope.mjs packages/codemods
    run_gate "Codemods coverage ratchet" bun scripts/quality/check-coverage-ratchet.mjs --root packages/codemods --summary coverage/coverage-summary.json --baseline ../../.quality/coverage-codemods-baseline.json --scope-file .quality/coverage-scope.json --scope-key packages/codemods --new-file-min 80
    run_gate "Reset editor coverage artifact" rm -f -- packages/editor/coverage/coverage-summary.json
    run_gate "Editor tests with coverage" bun run --filter @plane/editor test:coverage
    run_gate "Complete Editor source inventory" bun scripts/quality/complete-coverage-scope.mjs packages/editor
    run_gate "Editor coverage ratchet" bun scripts/quality/check-coverage-ratchet.mjs --root packages/editor --summary coverage/coverage-summary.json --baseline ../../.quality/coverage-editor-baseline.json --scope-file .quality/coverage-scope.json --scope-key packages/editor --new-file-min 80
    run_gate "Reset UI coverage artifact" rm -f -- packages/ui/coverage/coverage-summary.json
    run_gate "UI tests with coverage" bun run --filter @plane/ui test:coverage
    run_gate "Complete UI source inventory" bun scripts/quality/complete-coverage-scope.mjs packages/ui
    run_gate "UI coverage ratchet" bun scripts/quality/check-coverage-ratchet.mjs --root packages/ui --summary coverage/coverage-summary.json --baseline ../../.quality/coverage-ui-baseline.json --scope-file .quality/coverage-scope.json --scope-key packages/ui --new-file-min 80
    # test:frontend has its own tighter process guard; avoid nested process groups.
    bun run test:frontend:coverage || exit 1
    run_gate "Frontend source coverage ratchets" bun scripts/quality/check-frontend-coverage.mjs
    ;;
  --fix)
    run_gate "Oxlint safe fixes" bun run oxlint --fix .
    run_gate "Oxfmt fixes" bun run oxfmt .
    ;;
  *)
    printf 'Usage: %s [--fast|--full|--fix]\n' "$0" >&2
    exit 2
    ;;
esac

printf '\nTypeScript quality gates passed.\n'
