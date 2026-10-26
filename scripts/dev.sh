#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

# Keep the Docker and Go settings consistent with backend verification.
source scripts/dev-env.sh

if [[ "${1:-}" == "stop" ]]; then
    docker compose -p plane -f docker-compose-local.yml stop
    exit 0
fi

PLANE_ADMIN_PORT="${PLANE_ADMIN_PORT:-3001}"
PLANE_SPACE_PORT="${PLANE_SPACE_PORT:-3002}"
if [[ "$PLANE_ADMIN_PORT" == "$PLANE_SPACE_PORT" ]]; then
    echo "PLANE_ADMIN_PORT and PLANE_SPACE_PORT must be different." >&2
    exit 1
fi
export PLANE_ADMIN_PORT PLANE_SPACE_PORT

for env_file in .env apps/go-api/.env apps/web/.env apps/admin/.env apps/space/.env apps/live/.env; do
    if [[ ! -f "$env_file" ]]; then
        echo "Missing $env_file. Run ./setup.sh first." >&2
        exit 1
    fi
done

if ! docker info >/dev/null 2>&1; then
    if [[ "$(uname -s)" == "Darwin" ]] && command -v colima >/dev/null 2>&1; then
        colima start
    else
        echo "Start Docker, then run pnpm dev again." >&2
        exit 1
    fi
fi

docker compose -p plane -f docker-compose-local.yml up -d --build
# Recreate Caddy so it reads the selected upstream ports and current route policy.
docker compose -p plane -f docker-compose-local.yml up -d --no-deps --force-recreate proxy

# Override older .env files too: browser requests and auth redirects must share
# the proxy origin. Ports 3000–3100 are upstreams, not browser entry points.
export VITE_API_BASE_URL=http://localhost:8000
export VITE_WEB_BASE_URL=http://localhost:8000
export VITE_ADMIN_BASE_URL=http://localhost:8000
export VITE_SPACE_BASE_URL=http://localhost:8000
export VITE_LIVE_BASE_URL=http://localhost:8000
export API_BASE_URL=http://localhost:8000
export CORS_ALLOWED_ORIGINS=http://localhost:8000
export WEB_BASE_URL=http://localhost:8000
export LIVE_BASE_URL=http://localhost:8000
export PLANE_DEV_HOST=0.0.0.0
export VITE_LOCAL_LOGGING=1

echo "Vector: http://localhost:8000/"
echo "Instance administration: http://localhost:8000/god-mode/"
echo "Admin and space upstreams: localhost:${PLANE_ADMIN_PORT}, localhost:${PLANE_SPACE_PORT}"
echo "Ctrl-C stops the frontend servers. pnpm dev:stop stops the containers without deleting data."
echo "Logal: Docker volume plane_logaldata (48-hour retention). Query with pnpm dev:logs."
# Populate the development build cache serially before app servers start.
# Turbo watches this same dependency graph; rebuilds preserve live package outputs.
node scripts/quality/run-bounded.mjs --seconds 600 --memory-mib 4096 --heap-mib 2048 -- pnpm exec turbo run dev:build --concurrency=1 --filter='./packages/*'
exec go -C apps/go-api run ./cmd/devtools start
