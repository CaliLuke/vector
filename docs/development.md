# Develop Vector

## Requirements

- Go 1.27.3, as pinned in `apps/go-api/go.mod`.
- Node.js 22.18 or later and Corepack. `package.json` pins pnpm.
- Docker with Compose and Buildx. On macOS, the helper supports the default Colima profile.

## Start

Run from the repository root:

```sh
./setup.sh
pnpm dev
```

Setup copies `.env.example` files, creates a signing secret, and installs frontend dependencies.
Keep local environment files private. The example database and storage credentials are for local development only.

Open `http://localhost:8000/`. Instance administration uses `/god-mode/`, and public projects use `/spaces/`.
The proxy serves all browser entry points. Frontend ports 3000–3100 are internal development upstreams.

`PLANE_ADMIN_PORT` and `PLANE_SPACE_PORT` can override the admin and Sites upstream ports.
These compatibility names remain supported in Vector.

Stop frontend watchers with Ctrl-C. Stop containers without deleting data:

```sh
pnpm dev:stop
```

## Tests

```sh
pnpm test:frontend
pnpm check:lint
pnpm --filter web check:types
```

While the development stack runs, use package typecheck commands directly.
A Turbo typecheck can rebuild shared packages and interrupt browser requests.

Prepare the Docker environment before Go or container tests:

```sh
source scripts/dev-env.sh
go -C apps/go-api test ./...
go -C apps/go-api vet ./...
```

Development orchestration and login/log checks use `cmd/devtools`, written in Go.
Go unit tests and focused database tests run without Python or historical Git objects.

## Logs

```sh
pnpm dev:check-logs
pnpm dev:errors
pnpm dev:logs
pnpm dev:traces
```

The local collector stores logs inside its Docker volume. Use these commands instead of opening its SQLite database from the host.
Redact cookies, tokens, credentials, and personal data before sharing logs.

## Test account

Store machine-specific test credentials in `.env.test.local`. This file is ignored by Git.
Use the keys `PLANE_TEST_EMAIL` and `PLANE_TEST_PASSWORD`. Restrict access with `chmod 600 .env.test.local`.

```sh
set -a
source .env.test.local
set +a
pnpm dev:check-login
```

Do not copy another operator's credentials into this file or into test evidence.
