# Develop Vector

## Requirements

- Go 1.27.3, as pinned in `apps/go-api/go.mod`.
- Node.js 22.22.2+, 24.15+, or 26+ and Corepack. `package.json` pins pnpm.
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

The frontend launcher has a 6 GiB sampled process-tree RSS ceiling and a 1 GiB
heap limit per Node process. It stops all frontend workers if the ceiling is
exceeded or after 24 hours. Shared-package startup builds run serially under a
4 GiB ceiling. Docker containers are outside these host-process limits.

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

## Quality gates

Install Prek 0.5.5 and the pinned Go analysis tools once per machine or checkout:

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

The full gate includes frontend production builds. Stop frontend watchers before
running it, then restart `pnpm dev` for browser checks. To work while watchers run,
use the package typecheck scripts directly. Do not clean shared package output.

Husky remains the Git hook manager. Its pre-commit hook invokes the non-mutating
Prek `fast` group; pre-push invokes `full`. `pnpm install` installs Husky's hooks.
Do not run `prek install` over the same hooks.

CI selects the same full hooks with `--group ci`, using `--require-group go` or
`--require-group typescript` to split the jobs. The Go tools are pinned in
`scripts/quality/go-tools.sh`; frontend tools are pinned in the pnpm lockfile.

Go checks cover build, vet, formatting/imports, module-tidy drift, lint, dead code,
duplication, and integration-tagged tests with coverage. Docker is required for
disposable PostgreSQL, proxy, and broker tests. Package concurrency is bounded.
The test-result guard rejects failures, incomplete reports, and unreviewed skips.
`.quality/go-test-skips.json` records tests that need explicitly provisioned
fixtures; each entry states its prerequisite. Never point these tests at a database
with valuable data.

Checks run sequentially. Frontend tests use one worker, a 384 MiB JavaScript heap
limit, a 1 GiB sampled process-tree RSS ceiling, and an external five-minute
deadline. Each TypeScript gate has a ten-minute deadline and 3 GiB RSS ceiling;
production builds use one Turbo task at a time with a 4 GiB RSS ceiling. The Go
gate has a 6 GiB RSS ceiling and two Go runtime threads. These are process-group
watchdogs sampled every 100 ms, not OS-enforced total-memory reservations. They
track descendant processes and their groups, including Turbo task groups, but
do not measure Docker containers. A process that fully daemonizes between
samples can escape ancestry tracking; these guards are for build/test workers,
not an isolation boundary for untrusted programs.
The supervisor terminates the process group on failure or deadline and kills
leftover workers when their parent exits. See `scripts/quality/run-bounded.mjs`
for running focused commands under the same protection.

Frontend checks cover Oxlint, type diagnostics, Oxfmt, Knip, duplication, and the
Node frontend, Live, codemod, editor, and UI suites. Full builds also compile both
component Storybooks. Coverage reports must be freshly generated
before their ratchets run. Existing debt is recorded as explicit baselines rather
than hidden by excluding application source. Review and tighten those baselines
as findings are fixed; do not regenerate them simply to accept a failing change.

Initial static-analysis debt includes 2,395 syntax-lint findings and 5,681 full
TypeScript-aware lint findings (the full count includes syntax findings), plus
517 Knip findings. Go starts with 1,365 lint occurrences, 52 dead-code findings,
and 525 directed duplicate pairs. These baselines identify individual findings;
they are not global warning allowances. TypeScript compiler errors cannot be
baselined.

The TypeScript duplicate baseline was reviewed after the Vite and Recharts
migrations. It now records 981 pairs, nine fewer than the original 990. Twenty-
two fingerprint changes replace retired entries: 16 dropdown pairs became
identical when ref-only empty fragments were removed, five chart pairs retain
pre-existing Recharts scaffolding whose identities changed during the Recharts 3
typing updates, and one Storybook pair became identical when both configurations
moved to Vite. A drag-and-drop import-only match was later retired after the two
import lists diverged. The later Headless UI nonmodal fix changed eight more
fingerprints: all eight match existing file-pair fragments after removing only
`modal={false}` and whitespace. Their retired identities were replaced; the
baseline remains at 981 pairs. No new allowance was added for these identity changes.
Eight real duplicates were removed by sharing Go baseline-counting logic, a UI
dropdown keyboard handler, frontend test module-stub helpers, read-only hydration
test assertions, repeated page-persistence test setup, and the project-issues
test resolver.

The subsequent service/onboarding consolidation reduces the TypeScript baseline
to 974 pairs. One retained onboarding row-prop fingerprint replaces its longer
pre-existing version after removing an unused prop; eight old fingerprints were
retired. Go still reports 525 directed pairs, but seven fingerprints now identify
shorter adapter shells around a shared read pipeline (112 fewer reported duplicate
lines, counting overlapping reports). Four Go shadowing findings were also retired.
No duplicate thresholds or source exclusions changed.

The TanStack Router migration tightens four existing Oxlint identities in both
ratchets: authentication complexity falls from 54 to 36, issue-modal complexity
from 31 to 30 and 48 to 43, and issue-action file length from 724 to 703 lines.
Nine more complexity identities decrease after removing redundant route-parameter
conversions (one to six branches per function).
Only generated `routeTree.gen.ts` is excluded from lint, formatting, and duplication checks;
owned route modules remain checked. No new debt allowance was added.

The router migration and frontend consolidation reduce TypeScript clone pairs from
974 to 922 (981 at the original baseline). All 63 changed fingerprints were reviewed:
47 match pre-existing clone ranges, nine follow extracted filter/menu code or
shorter callers, four cover unchanged JSX/imports beside native route declarations,
and three cover unchanged service code or shorter issue-menu callers. The review
also removed redundant parameter aliases and string conversions. Stale fingerprints
were retired; no clone threshold or owned-source exclusion changed.

The Go service consolidation lowers the duplicate baseline from 525 to 505
directed reports, with 590 fewer reported duplicate lines than the original
baseline (overlapping reports counted separately). Shared session handling,
user-property conversion, attachment listing, throttle serialization, cookie
parsing, public metadata decoding, and background-task admission preserve each
service's
authorization and effect ordering. Eight changed report identities replace
pre-existing fragments in smaller helpers' callers; no file-pair allowance
increased. Two more shadowing findings and an unused conversion helper were
retired from the lint baseline; the conversion helper was also retired from the
dead-code baseline. Generated contracts and workflow-specific durable identities
remain explicit.

The initial measured coverage is 30.20% lines across 44 Live source files,
79.82% across 2 codemod files, 0.05% across 224 editor files, and 2.16% across
116 UI files. These are explicit starting floors, not coverage targets. The
ratchet checks each file and requires 80% coverage for new files in measured
suites. `.quality/coverage-scope.json` lists unmeasured workspaces and explains
why the Node frontend harness cannot yet provide reliable source coverage.
The shared UI icon registry has a 100% floor for lines, statements, functions,
and branches.

See [the test-suite review](test-suite-review.md) for legacy failure expectations
and test-quality gaps that require separate behavior changes.

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
