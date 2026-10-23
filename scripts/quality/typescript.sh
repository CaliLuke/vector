#!/usr/bin/env bash
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

MODE="${1:---full}"
FAILED=()

run_gate() {
  local name="$1"
  shift
  printf '\n▶ %s\n' "$name"
  if ! node scripts/quality/run-bounded.mjs --seconds 600 --memory-mib 3072 --heap-mib 1024 -- "$@"; then
    FAILED+=("$name")
  fi
  return 0
}

case "$MODE" in
  --fast)
    run_gate "Oxlint syntax diagnostic ratchet" node scripts/quality/check-oxlint-baseline.mjs fast
    run_gate "Oxfmt check" pnpm exec oxfmt --check .
    run_gate "Process guard regression tests" node --test --test-concurrency=1 scripts/quality/run-bounded.test.mjs
    run_gate "Quality-ratchet fixture tests" node scripts/quality/test-ratchets.mjs
    ;;
  --full)
    run_gate "Generated route types" bash scripts/quality/route-types.sh
    run_gate "Oxlint type-aware rules and TypeScript diagnostics" env GOMEMLIMIT=1GiB GOMAXPROCS=2 node scripts/quality/check-oxlint-baseline.mjs full
    run_gate "Oxfmt check" pnpm exec oxfmt --check .
    run_gate "Process guard regression tests" node --test --test-concurrency=1 scripts/quality/run-bounded.test.mjs
    run_gate "Quality-ratchet fixture tests" node scripts/quality/test-ratchets.mjs
    run_gate "Knip unused-code/dependency ratchet" node scripts/quality/check-knip-ratchet.mjs
    run_gate "jscpd duplicate baseline" pnpm exec jscpd --config jscpd.json --baseline .quality/jscpd-baseline.json --fail-on-new-clones
    run_gate "Constants endpoint tests" pnpm --filter @plane/constants test
    run_gate "Reset Live coverage artifact" rm -f -- apps/live/coverage/coverage-summary.json
    run_gate "Live tests with coverage" pnpm --filter live test:coverage
    run_gate "Live coverage ratchet" node scripts/quality/check-coverage-ratchet.mjs --root apps/live --summary coverage/coverage-summary.json --baseline ../../.quality/coverage-live-baseline.json --scope-file .quality/coverage-scope.json --scope-key apps/live --new-file-min 80
    run_gate "Reset codemods coverage artifact" rm -f -- packages/codemods/coverage/coverage-summary.json
    run_gate "Codemods tests with coverage" pnpm --filter @plane/codemods test:coverage
    run_gate "Codemods coverage ratchet" node scripts/quality/check-coverage-ratchet.mjs --root packages/codemods --summary coverage/coverage-summary.json --baseline ../../.quality/coverage-codemods-baseline.json --scope-file .quality/coverage-scope.json --scope-key packages/codemods --new-file-min 80
    run_gate "Reset editor coverage artifact" rm -f -- packages/editor/coverage/coverage-summary.json
    run_gate "Editor tests with coverage" pnpm --filter @plane/editor test:coverage
    run_gate "Editor coverage ratchet" node scripts/quality/check-coverage-ratchet.mjs --root packages/editor --summary coverage/coverage-summary.json --baseline ../../.quality/coverage-editor-baseline.json --scope-file .quality/coverage-scope.json --scope-key packages/editor --new-file-min 80
    run_gate "Reset UI coverage artifact" rm -f -- packages/ui/coverage/coverage-summary.json
    run_gate "UI tests with coverage" pnpm --filter @plane/ui test:coverage
    run_gate "UI coverage ratchet" node scripts/quality/check-coverage-ratchet.mjs --root packages/ui --summary coverage/coverage-summary.json --baseline ../../.quality/coverage-ui-baseline.json --scope-file .quality/coverage-scope.json --scope-key packages/ui --new-file-min 80
    # test:frontend has its own tighter process guard; avoid nested process groups.
    pnpm test:frontend || FAILED+=("Frontend behavior tests")
    ;;
  --fix)
    run_gate "Oxlint safe fixes" pnpm exec oxlint --fix .
    run_gate "Oxfmt fixes" pnpm exec oxfmt .
    ;;
  *)
    printf 'Usage: %s [--fast|--full|--fix]\n' "$0" >&2
    exit 2
    ;;
esac

if ((${#FAILED[@]})); then
  printf '\nFailed gates:\n'
  printf '  - %s\n' "${FAILED[@]}"
  exit 1
fi

printf '\nTypeScript quality gates passed.\n'
