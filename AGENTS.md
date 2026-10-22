# Vector development instructions

Vector is an independent derivative of Plane. Preserve upstream copyright notices and the AGPL license.
Use Vector in product copy. Keep compatibility identifiers unless a change explicitly migrates their consumers.

Read `docs/development.md` for commands and `docs/architecture.md` for source boundaries.
Store local test-account credentials in ignored `.env.test.local` with mode 0600.
Check that file before asking for credentials. Never print or commit credentials.

Use existing libraries and shared helpers. Keep authorization, tenant isolation, data integrity, and durable effects intact.
Add focused regression tests for defects. Run affected tests, formatting, lint, and typechecks.
Do not edit generated Go transport output. Regenerate only affected source designs.

Use `http://localhost:8000/` for local browser checks. Keep the proxy and application origins consistent.
While the development stack runs, use package typecheck scripts directly. Do not clean shared package output.
Read local logs through `pnpm dev:logs`, `pnpm dev:errors`, and `pnpm dev:traces`.
Never read the collector's live SQLite database from the host.

Before Docker or Go checks, run:

```sh
source scripts/dev-env.sh
```

Keep dependency work within this repository unless explicitly authorized elsewhere.
Do not publish private notes, credentials, customer data, or internal information from other projects.
Do not send messages to third parties without authorization.
