#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
for app in admin space web; do
  (cd "$ROOT/apps/$app" && ./node_modules/.bin/react-router typegen)
done
