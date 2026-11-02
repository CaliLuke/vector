# Vector development instructions

Vector is an independent derivative of Plane. Preserve upstream copyright notices and the AGPL license.
Use Vector in product copy. Keep compatibility identifiers unless a change explicitly migrates their consumers.
Django compatibility is no longer required. MCP supports only revision `2026-07-28`; do not add legacy handshakes or revision matrices.
Run credential-neutral domain checks once. Use credential matrices for authentication and authorization interactions, plus generated action smoke coverage.

Read `docs/development.md` for commands and `docs/architecture.md` for source boundaries.
Store local test-account credentials in ignored `.env.test.local` with mode 0600.
Check that file before asking for credentials. Never print or commit credentials.

Use existing libraries and shared helpers. Keep authorization, tenant isolation, data integrity, and durable effects intact.
Add focused regression tests for defects. Run affected tests, formatting, lint, and typechecks.
Do not edit generated Go transport output. Regenerate only affected source designs.

TanStack packages bundle skills that may not appear in the agent's skill list.
Before TanStack work, read the installed package's `skills/**/SKILL.md` and its declared `requires`:

- Router: `node_modules/@tanstack/react-router/skills/react-router/SKILL.md`; also see `skills/compositions/router-query/SKILL.md` in that package and the `@tanstack/router-core` skills.
- React DB: `packages/hooks/node_modules/@tanstack/react-db/skills/react-db/SKILL.md`; follow its `@tanstack/db` core, collection, live-query, mutation, and persistence skills.
- Table: `apps/web/node_modules/@tanstack/react-table/skills/getting-started/SKILL.md`, plus its table-state and migration skills and the `@tanstack/table-core` skills.
- Route tooling: the `skills/` directories in `apps/web/node_modules/@tanstack/router-plugin` and `apps/web/node_modules/@tanstack/virtual-file-routes`.

Find bundled skills, including transitive core packages, with:

```sh
rg --files --hidden --no-ignore node_modules/.pnpm -g SKILL.md | rg '/@tanstack/'
```

Use the version linked by the consuming workspace; `.pnpm` can retain older package copies.

Run `prek run --all-files --group full` for the complete quality gate.
Use `./check.sh --fast` for non-mutating fast checks and `./check.sh --fix` for formatting/lint fixes.
Husky owns Git hooks and invokes Prek; do not install a second hook manager.
Full checks include frontend builds: stop frontend watchers first, then restart the stack for UI checks.
Review baseline changes explicitly. Do not increase a baseline to hide a new defect or exclude owned source to make a gate pass.

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
