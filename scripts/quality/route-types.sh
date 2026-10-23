#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
for app in admin space; do
  (cd "$ROOT/apps/$app" && ./node_modules/.bin/react-router typegen)
done
(cd "$ROOT/apps/web" && ./node_modules/.bin/tsr generate)
