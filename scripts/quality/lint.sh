#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

bash scripts/quality/route-types.sh
node scripts/quality/check-oxlint-baseline.mjs full
