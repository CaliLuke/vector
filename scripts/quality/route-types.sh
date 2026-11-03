#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
for app in admin space; do
  (cd "$ROOT/apps/$app" && bun run react-router typegen)
done
(cd "$ROOT/apps/web" && bun run tsr generate)
