#!/usr/bin/env bash
# Shared entry point for local checks, Husky/Prek hooks, and CI.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

# Manual checks and Husky must resolve tools from the same PATH so a successful
# manual stage remains reusable by publication hooks. Preserve all other entries.
while [[ "$PATH" == node_modules/.bin:* || "$PATH" == "$ROOT/node_modules/.bin:"* ]]; do
  PATH="${PATH#*:}"
done
export PATH="$ROOT/node_modules/.bin:$PATH"

mode=--full
scope=all
fresh=()
status=false
for argument in "$@"; do
  case "$argument" in
    --fast|--full|--fix) mode="$argument" ;;
    --go) scope=go ;;
    --typescript) scope=typescript ;;
    --build) scope=build ;;
    --fresh) fresh=(--fresh) ;;
    --status) status=true ;;
    *) echo "Usage: ./check.sh [--fast|--full|--fix] [--go|--typescript|--build] [--fresh|--status]" >&2; exit 2 ;;
  esac
done

if [[ "$status" == true ]]; then
  exec node scripts/quality/full-check.mjs --status
fi
source scripts/dev-env.sh
if [[ "$mode" == --full && "$scope" != build ]]; then
  exec node scripts/quality/full-check.mjs "$scope" "${fresh[@]}"
fi
if ((${#fresh[@]})); then
  echo "--fresh applies only to full checks." >&2
  exit 2
fi

run_gate() {
  local name="$1"
  shift
  printf '\n▶ %s\n' "$name"
  if ! "$@"; then
    printf '\nFailed gate: %s; stopping before later stages.\n' "$name" >&2
    exit 1
  fi
}

if [[ "$scope" == build ]]; then
  run_gate "Frontend build" pnpm build
  run_gate "Component Storybooks" node scripts/quality/run-bounded.mjs --seconds 1200 --memory-mib 4096 --heap-mib 2048 -- pnpm --workspace-concurrency=1 --filter @plane/ui --filter @plane/propel build-storybook
fi
if [[ "$scope" == all || "$scope" == go ]]; then
  run_gate Go node scripts/quality/run-bounded.mjs --seconds 3600 --memory-mib 6144 --heap-mib 512 -- env GOMEMLIMIT=2GiB GOMAXPROCS=2 bash scripts/quality/go.sh "$mode"
fi
if [[ "$scope" == all || "$scope" == typescript ]]; then
  run_gate TypeScript bash scripts/quality/typescript.sh "$mode"
fi
printf '\nAll selected quality gates passed.\n'
