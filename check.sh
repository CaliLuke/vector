#!/usr/bin/env bash
# Shared entry point for local checks, Husky/Prek hooks, and CI.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

mode=--full
scope=all
for argument in "$@"; do
  case "$argument" in
    --fast|--full|--fix) mode="$argument" ;;
    --go) scope=go ;;
    --typescript) scope=typescript ;;
    --build) scope=build ;;
    *) echo "Usage: ./check.sh [--fast|--full|--fix] [--go|--typescript|--build]" >&2; exit 2 ;;
  esac
done

failed=()
run_gate() {
  local name="$1"
  shift
  if ! "$@"; then
    failed+=("$name")
  fi
}

if [[ "$scope" == build || ( "$scope" == all && "$mode" == --full ) ]]; then
  run_gate "Frontend build" pnpm build
  run_gate "Component Storybooks" node scripts/quality/run-bounded.mjs --seconds 1200 --memory-mib 4096 --heap-mib 2048 -- pnpm --workspace-concurrency=1 --filter @plane/ui --filter @plane/propel build-storybook
fi
if [[ "$scope" == all || "$scope" == go ]]; then
  run_gate Go node scripts/quality/run-bounded.mjs --seconds 3600 --memory-mib 6144 --heap-mib 512 -- env GOMEMLIMIT=2GiB GOMAXPROCS=2 bash scripts/quality/go.sh "$mode"
fi
if [[ "$scope" == all || "$scope" == typescript ]]; then
  run_gate TypeScript bash scripts/quality/typescript.sh "$mode"
fi

if ((${#failed[@]})); then
  printf '\nFailed quality gates:\n'
  printf '  - %s\n' "${failed[@]}"
  exit 1
fi
printf '\nAll selected quality gates passed.\n'
