# Vector Go backend

This module serves the Vector API, authentication, native background jobs, and database bootstrap.
It uses Go 1.27.3. See `go.mod` for pinned dependencies.

Run from the repository root:

```sh
source scripts/dev-env.sh
go -C apps/go-api test ./...
go -C apps/go-api vet ./...
```

Use [the development guide](../../docs/development.md) to run the complete application.
Use [the deployment guide](../../docs/self-hosting.md) for the source-built production stack.

`cmd/system` runs the API. `cmd/bootstrap` manages database bootstrap and installation.
The production process enables native workers with `--go-background-tasks`.
Liveness is available at `/health/live`, and database readiness at `/health/ready`.

Edit source designs before regenerating transports. Keep generated files unchanged by hand.
Generate only affected designs.
The module import path remains `github.com/CaliLuke/go-plane` for generated-code compatibility.

Static contract fixtures live in `internal/parity/testdata/fixtures`.
Historical Django replay tools have been removed; current tests run against Go.
