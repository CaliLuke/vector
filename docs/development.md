# Develop Vector

## Requirements

- Go 1.27.3, as pinned in `apps/go-api/go.mod`.
- Node.js 26.12.1 and Corepack 0.36.0 (`npm install --global corepack@0.36.0`). `package.json` pins pnpm 12.15.0 with its registry integrity hash.
- Docker with Compose and Buildx. On macOS, the helper supports the default Colima profile.

Workspace typechecks use the native TypeScript 7 compiler (`tsc`, currently 7.0.2).
The `@typescript/native` catalog alias installs that compiler. The separate
`typescript` alias provides Microsoft's TypeScript 6 compatibility API for tools
and test loaders that import it. It does not supply the `tsc` command. This follows
[Microsoft's side-by-side installation guidance](https://devblogs.microsoft.com/typescript/announcing-typescript-7-0/#running-side-by-side-with-typescript-60).
Check command resolution with `pnpm exec tsc --version` and
`pnpm -r exec tsc --version`.

Dependency updates are recorded in the workspace catalog and lockfile. The Tiptap
HTML server currently has a scoped Happy DOM 21.0.4 override to include the fix for
[GHSA-834v-8h66-3h79](https://github.com/advisories/GHSA-834v-8h66-3h79).
[Tiptap issue #8540](https://github.com/ueberdosis/tiptap/issues/8540) tracks its
outdated peer range. Remove the override when Tiptap accepts the fixed release.
The editor serialization regression covers the consumer integration.

Compose uses PostgreSQL 18.6 and keeps the existing `/var/lib/postgresql/data`
mount and explicit `PGDATA` for compatibility with the current physical volume.
The default named volume remains `<Compose project>_pgdata`.
For a major-version restore, set `POSTGRES_DATA_VOLUME` in the ignored `.env` file to select a separate physical volume. A new image tag does not migrate
the database. Before switching an existing installation, make and check a
backup, restore it into the selected volume, then start the upgraded service.
This prevents a new empty volume from silently replacing the database selected
by the default project-scoped name.

## Start

Run from the repository root:

```sh
./setup.sh
pnpm dev
```

Setup copies `.env.example` files, creates a signing secret, and installs frontend dependencies.
Keep local environment files private. The example database and storage credentials are for local development only.

Open `http://localhost:8000/`. Instance administration uses `/god-mode/`, and public projects use `/spaces/`.

The [remote MCP transport](mcp.md) is disabled by default. Its 30 tools expose
208 actions through personal tokens or generic external-issuer OAuth. OAuth
requires the [issuer profile and account linking](mcp-oauth.md). Generated-server
workflow tests cover all actions under both authentication modes and protocol
revisions. The [parity tracker](mcp-parity.md) records the remaining acceptance
checks. Mutations support optional [receipt and revision extensions](mcp-writes.md).
Teamspaces are deferred until native support exists.
The proxy serves all browser entry points. Frontend ports 3000–3100 are internal development upstreams.

`PLANE_WEB_PORT`, `PLANE_ADMIN_PORT`, and `PLANE_SPACE_PORT` can override the web,
admin, and Sites upstream ports. Choose distinct free ports. The servers fail if
a selected port is occupied. The proxy uses the same selected ports, and the
browser origin remains `http://localhost:8000/`. These compatibility names remain
supported in Vector.

The frontend launcher has a 6 GiB sampled process-tree RSS ceiling and a 1 GiB
heap limit per Node process. It stops all frontend workers if the ceiling is
exceeded or after 24 hours. Shared-package startup builds run serially under a
4 GiB ceiling. Turbo then watches the app dependency graph with five task slots:
four persistent app servers and one slot for shared-package rebuilds. Development
rebuilds preserve existing package output. Docker containers are outside these
host-process limits.

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

Testcontainers core and its PostgreSQL module are pinned to
`v0.44.1-0.20261029064637-ff84c26717f1`. This upstream commit includes the
[HostPort timeout fix](https://github.com/testcontainers/testcontainers-go/pull/4055)
for [issue #4054](https://github.com/testcontainers/testcontainers-go/issues/4054).
The proxy fixtures use Testcontainers' SSH host-port forwarding on all platforms.
The Colima gateway workaround is removed. The proxy integration suite passes
with the pinned version and the existing readiness deadlines. Replace both pins
with a tagged release that contains this commit when one is available.

Ryuk remains enabled. `scripts/dev-env.sh` defaults its reconnection grace to
30 minutes to cover idle gaps between package test binaries while
[Testcontainers #3867](https://github.com/testcontainers/testcontainers-go/issues/3867)
remains open. A stopped reaper can otherwise be selected for reuse, and its missing
logs consume the startup timeout. Normal fixtures explicitly remove their containers.
The grace controls fallback cleanup after the last client disconnects. Fixture
readiness and test deadlines are unchanged. Remove this default when the pinned
version includes the [reaper reuse fix](https://github.com/testcontainers/testcontainers-go/pull/3868).

Development orchestration and login/log checks use `cmd/devtools`, written in Go.
Go unit tests and focused database tests run without Python or historical Git objects.

## Quality gates

Install Prek 0.5.7 and the pinned Go analysis tools once per machine or checkout:

```sh
brew install prek
bash scripts/quality/go-tools.sh --install
```

Use the shared entry point locally and in CI:

```sh
prek run --all-files --group full
./check.sh --fast
./check.sh --fix
```

The full gate includes frontend production builds. Before the full gate, stop frontend watchers.
After the gate, restart `pnpm dev` for browser checks. To work while watchers run,
use the package typecheck scripts directly. Do not clean shared package output.

Husky remains the Git hook manager. Its pre-commit hook invokes the non-mutating
Prek `fast` group. Pre-push invokes `full`. `pnpm install` installs Husky's hooks.
Do not run `prek install` over the same hooks.

CI selects the same full hooks with `--group ci`, using `--require-group go` or
`--require-group typescript` to split the jobs. The Go tools are pinned in
`scripts/quality/go-tools.sh`. Frontend tools are pinned in the pnpm lockfile.

Go checks cover build, vet, formatting/imports, module-tidy drift, lint, dead code,
duplication, enforced function complexity, and integration-tagged tests with coverage. Docker is required for
disposable service fixtures. Package concurrency is bounded.
The test-result guard rejects failures, incomplete reports, and unreviewed skips.
`.quality/go-test-skips.json` is empty. Tests own their disposable fixtures.
Never point these tests at a database with valuable data.

### Baselines and review

Current Go allowances and coverage floors live in `configs/quality/`.
Current frontend allowances, coverage floors, and unmeasured scopes live in `.quality/`.
The gate reports current findings against these files. Historical cleanup counts remain in Git history.

Review each baseline change against its source finding. When evidence supports the change, remove resolved entries and raise floors.
Do not increase an allowance, relax a threshold, or exclude owned source to accept a failure.
TypeScript compiler errors cannot have baseline allowances.

Configured lint does not cover every discarded Go error. The stricter `errcheck` audit with `check-blank: true` found additional debt.
Those findings remain outstanding until a fresh audit proves their removal.
Zero coverage floors and unmeasured frontend workspaces also remain outstanding work.
A passing gate does not prove that all technical debt or MCP acceptance work is complete.

Both fast and full Go gates enforce a maintainability index floor of 20 for new
functions. Higher scores are better. `configs/quality/go-maintainability-baseline.json`
records numeric floors for existing functions. Any score decline fails even if
that function was already flagged. Identity uses file, package and receiver/function
name, not the source line. The pinned `maintidx` analyzer runs under the same
integration tags and generated-source exclusions as lint. The existing AST-based
function inventory resolves receiver identity. Its complexity values are not gates.
Missing tools, analyzer failures, malformed reports and unknown functions fail.

A zero maintainability score cannot decline further. Changes to files with zero-score allowances therefore require explicit source-fingerprint review.
This file-level check requires review. It does not mean that every edit worsens the implementation.

Refactoring acceptance requires coherent responsibilities, narrow interfaces,
visible effect ordering, and less work to understand or change the behavior.
A passing size or complexity metric alone is insufficient.

Inline or regroup helpers that only redistribute branches or add forwarding layers.

Run `bash scripts/quality/go.sh --maintainability` for the focused gate. An explicit
`--write-maintainability-baseline` regenerates allowances for review. Normal checks
never change the baseline. Improvements are reported so their floors can be
raised. Do not refresh the baseline to accept a newly failing implementation.

Automated structural analysis remains blocked by
[Lagotto’s Go 1.27 loader incompatibility](https://github.com/CaliLuke/lagotto/issues/58).
Function scores do not replace a review of responsibilities, interfaces, and effect ordering.

### Coverage and fixtures

Coverage instruments the same owned Go packages that the ratchet measures, so
integration tests also count calls into other packages. Repeated coverage blocks
from separate test binaries are merged by source location. A hit in either binary
covers the block once. Generated-source exclusions and package floors still apply.
The parser streams large profiles instead of loading their complete text into memory.
Optional file floors protect established contracts independently of the package average.
The baseline refresh command preserves those file floors.

Frontend coverage checks each measured file and requires 80% coverage for new files in measured suites.
`.quality/coverage-scope.json` records unmeasured workspaces and the limits of the Node frontend harness.
The shared UI icon registry has a 100% floor for lines, statements, functions, and branches.

The work-item type tests, full-schema MCP workflows, and MCP identity tests migrate
disposable templates per package and clone a separate database for each test. Test data and
connections are never shared between those clones. Migration-specific tests still
exercise fresh database installation directly.
PostgreSQL fixtures check a SQL query through the published port before running
seed migrations. Only readiness is retried. A failed migration is returned once
and its container is reclaimed.
The macOS environment keeps Testcontainers' resource reaper enabled so containers
are reclaimed when a test process exits before its normal cleanup. Explicitly
disabling it requires an external runner that owns cleanup. See the
[Testcontainers cleanup guidance](https://golang.testcontainers.org/features/garbage_collector/).

See [the test-suite review](test-suite-review.md) for legacy failure expectations
and test-quality gaps that require separate behavior changes.

### Gate execution and resource limits

The frontend pipeline completes before the full Go gate starts. Distinct Prek
priorities keep frontend builds from competing with Docker-backed tests for host
resources. Within the frontend pipeline,
production builds run first, followed by TypeScript checks. Builds and checks remain
sequential so generated package output is ready before typechecking. Frontend tests run in two sequential shards with two isolated workers each.
Each shard has a 384 MiB JavaScript heap limit, a 1 GiB sampled process-tree RSS ceiling, and an external five-minute deadline. Every frontend file runs once. Either shard failing fails the gate.
Individual tests retain their 15-second timeout. Each TypeScript gate
has a ten-minute deadline and 3 GiB RSS ceiling.
Production builds use one Turbo task at a time with a 4 GiB RSS ceiling. The Go
gate has a 6 GiB RSS ceiling and two Go runtime threads. These are process-group
watchdogs sampled every 100 ms, not OS-enforced total-memory reservations. They
retry one timed-out process sample. Consecutive sample timeouts or other
monitoring errors fail the gate. The external command deadline remains active.
They track descendant processes and their groups, including Turbo task groups, but
do not measure Docker containers. A process that fully daemonizes between
samples can escape ancestry tracking. These guards are for build/test workers,
not an isolation boundary for untrusted programs.
The supervisor terminates the process group on failure or deadline and kills
leftover workers when their parent exits. See `scripts/quality/run-bounded.mjs`
for running focused commands under the same protection.

Frontend checks cover Oxlint, type diagnostics, Oxfmt, Knip, duplication, and the
Node frontend, Live, codemod, editor, and UI suites. Full builds also compile both
component Storybooks. Coverage reports must be freshly generated
before their ratchets run. Existing debt is recorded as explicit baselines rather
than hidden by excluding application source. Review and tighten those baselines
as findings are fixed. Do not regenerate them simply to accept a failing change.

Knip has no accepted findings. Version 6.44.0 includes the
[package-extension peer fix](https://github.com/webpro-nl/knip/pull/2246), so the
editor’s declared `@tiptap/pm` peer no longer needs a root-workspace allowance.

The frontend behavior suite uses Node's compilation cache and a shared TypeScript transform cache.
Transform keys include source content, compiler version and every transform option.
Workers publish cache entries atomically under ignored `node_modules/.cache` directories.
These caches preserve worker isolation and do not cache test results.
Node's compilation cache is scoped to behavior tests, because it can reduce V8 coverage precision.
Coverage suites use separate commands without that cache setting.

### Dependency limitations

Multipart readers keep the `NextPart` error separate from body-read errors because
[errorlint issue #49](https://codeberg.org/polyfloyd/go-errorlint/issues/49) describes
incorrect EOF warnings when an error variable is reassigned. This separation can
be simplified when the analyzer tracks assignments at each comparison. Preserve
direct `io.EOF` comparisons: wrapped EOF errors must still report parse failures.

Do not combine source overlays with `-coverpkg` for coverage checks until
[golang/go#73802](https://github.com/golang/go/issues/73802) is fixed in the pinned
toolchain and its reproduction passes. The issue can ignore overlaid code and
produce a false passing test. A standalone Go 1.27.3 reproduction was
[added upstream](https://github.com/golang/go/issues/73802#issuecomment-6480328772).
Use the applied source and a fresh build for coverage checks. The gates retain no compiler flag or dependency patch for this limitation.

HTTP response bodies retain explicit checked closes where the current bodyclose
analyzer cannot follow a cross-package cleanup helper. This limitation is tracked
in [timakin/bodyclose#60](https://github.com/timakin/bodyclose/issues/60), with a
[current standalone reproduction](https://github.com/timakin/bodyclose/issues/60#issuecomment-6480673025).
Those call sites can use the shared helper when the pinned analyzer accepts that
reproduction. No linter suppression or dependency patch was added.

The legacy tooltip uses Floating UI's controlled element API with callback-ref
state. Oxlint currently mistakes a callback property passed to JSX `ref` for a
render-time ref read. [oxc#23753](https://github.com/oxc-project/oxc/issues/23753#issuecomment-6510863595)
contains a standalone reproduction on the pinned versions. The tooltip can use
Floating UI's returned ref setters directly when the pinned analyzer accepts
that callback-member form. No rule is suppressed.

The collaborative editor resolves `@tiptap/y-tiptap` to release 3.0.9.
Version 3.0.10 drops the first text inserted into an empty block if a block-ID plugin appends a transaction. The reproduction and release comparison are recorded in
[upstream issue #8584](https://github.com/ueberdosis/tiptap/issues/8584#issuecomment-6413802381).
No dependency source is patched. Remove the override when an upstream release
passes `packages/editor/src/core/extensions/unique-id/collaboration.test.ts` and
the page typing/reload integration check.

Form-core 1.33.5 resolves `@tanstack/devtools-event-client` to its released 0.5.0
version. Version 0.4.4 keeps a Node process alive for about five seconds after a
form is unmounted while it retries a devtools connection. The narrow override
uses the upstream release's environment-aware entry point. No dependency source
or test runtime is patched. Remove it after a Form release incorporates
[the upstream dependency update](https://github.com/TanStack/form/pull/2410).
The process-lifetime reproduction is recorded on
[the existing upstream issue](https://github.com/TanStack/form/issues/2132).

## Logs

```sh
pnpm dev:check-logs
pnpm dev:errors
pnpm dev:logs
pnpm dev:traces
```

The local collector stores logs inside its Docker volume. Use these commands instead of opening its SQLite database from the host.
Redact cookies, tokens, credentials, and personal data before sharing logs.

The Go API exports structured logs and traces directly to the configured OTLP
endpoint, including request trace and span IDs. This continues while frontend
watchers are stopped for checks. The development runner forwards frontend,
migrator, and proxy output. It does not forward the API's stdout a second time.
`dev:check-logs` exercises a read-only HTTP 500 and requires both its error log
and matching server span, in addition to checking proxy ingestion.

`next-themes@0.4.6` emits a React development warning about its inline script on
client rendering with React 19.3.1. A standalone reproduction is recorded in
[upstream issue #385](https://github.com/pacocoursey/next-themes/issues/385#issuecomment-6378110860).
The dependency remains unpatched and the warning is not suppressed. Recheck it
when adopting an upstream release that fixes client-rendered providers.

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

## Current-value search migration

Schema migration 13 adds a separate rich-text search projection. Before upgrading,
stop older API and worker writers, then run the new `bootstrap schema` or
`bootstrap install` command to completion before starting the new application.
Do not perform a rolling upgrade across this boundary: older writers do not
maintain the projection. Frontend watchers can remain running during this schema step.
API requests will be unavailable while the backend is stopped.

Bootstrap sanitizes existing saved values without editing items, timestamps,
history, or activity. It processes bounded batches within each workspace transaction
and records that workspace's completion atomically. Failure or interruption rolls
back that workspace. A retry skips completed workspaces. Invalid legacy rich text
reports item and field IDs without logging their content. Repair the identified
value before retrying. `BOOTSTRAP_TIMEOUT` accepts a positive Go duration (default
`1h`). Use a longer duration, for example `4h`, for large installations.

## Runtime and API contracts

Redis cache clients use standard TLS certificate checks for `rediss` URLs.
For private deployments, install the CA in the host or container trust store.
Use a certificate that matches the Redis hostname. Untrusted certificates fail the connection.

[Architecture](architecture.md) describes source ownership and runtime boundaries.
[MCP parity](mcp-parity.md) records acceptance work that remains after action wiring.
The [relation actions](mcp-work-item-relations.md) use native relation operations, permissions, and durable effects.
The [property API](mcp-work-item-properties.md) documents catalogue actions, revision triggers, and preservation checks.
