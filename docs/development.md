# Develop Vector

## Requirements

- Go 1.27.3, as pinned in `apps/go-api/go.mod`.
- Node.js 26.12.1 and Corepack 0.36.0 (`npm install --global corepack@0.36.0`). `package.json` pins pnpm 12.15.0 with its registry integrity hash.
- Docker with Compose and Buildx. On macOS, the helper supports the default Colima profile.

Workspace typechecks use the native TypeScript 7 compiler (`tsc`, currently 7.0.2).
The `@typescript/native` catalog alias installs that compiler. The separate
`typescript` alias provides Microsoft's TypeScript 6 compatibility API for tools
and test loaders that import it; it does not supply the `tsc` command. This follows
[Microsoft's side-by-side installation guidance](https://devblogs.microsoft.com/typescript/announcing-typescript-7-0/#running-side-by-side-with-typescript-60).
Verify command resolution with `pnpm exec tsc --version` and
`pnpm -r exec tsc --version`.

Dependency updates are recorded in the workspace catalog and lockfile. The Tiptap
HTML server currently has a scoped Happy DOM 21.0.4 override to include the fix for
[GHSA-834v-8h66-3h79](https://github.com/advisories/GHSA-834v-8h66-3h79).
[Tiptap issue #8540](https://github.com/ueberdosis/tiptap/issues/8540) tracks its
outdated peer range. Remove the override when Tiptap accepts the fixed release;
the editor serialization regression covers the consumer integration.

Compose uses PostgreSQL 18.6 and keeps the existing `/var/lib/postgresql/data`
mount and explicit `PGDATA` for compatibility with the current physical volume.
The default named volume remains `<Compose project>_pgdata`; set
`POSTGRES_DATA_VOLUME` in the ignored `.env` file to select a separate physical
volume for a verified major-version restore. A new image tag does not migrate
the database. Before switching an existing installation, make and verify a
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

The [remote MCP transport](mcp.md) is disabled by default. Personal-token mode can
authenticate requests and call five `member` reads and two `project` reads. OAuth mode requires
the [external issuer profile and account linking](mcp-oauth.md). The remaining
resource action contracts are still pending.
The proxy serves all browser entry points. Frontend ports 3000–3100 are internal development upstreams.

`PLANE_ADMIN_PORT` and `PLANE_SPACE_PORT` can override the admin and Sites upstream ports.
These compatibility names remain supported in Vector.

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

For the default Colima profile, `scripts/dev-env.sh` sets
`VECTOR_TEST_HOST_GATEWAY=host.docker.internal`. Proxy fixtures use Colima's native
host gateway to reach their local HTTP test servers. Other Docker environments
keep Testcontainers' managed host-port tunnel. An explicitly empty
`VECTOR_TEST_HOST_GATEWAY` also selects that portable default.

This Colima default avoids the SSH-sidecar readiness defect tracked in
[Testcontainers issue #4054](https://github.com/testcontainers/testcontainers-go/issues/4054).
Remove the default once a released HostPort wait strategy handles transient TCP
timeouts and the SSH path passes targeted Colima verification. Test assertions and
readiness deadlines are the same for both network paths.

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

The full Go gate and frontend pipeline run concurrently. Within the frontend pipeline,
production builds run first, followed by TypeScript checks; builds and checks remain
sequential so generated package output is ready before typechecking. Frontend tests
use two isolated workers, a 384 MiB JavaScript heap limit, a 1 GiB sampled
process-tree RSS ceiling, and an external five-minute deadline. Each TypeScript gate
has a ten-minute deadline and 3 GiB RSS ceiling;
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

The TanStack Query migration tightens eight existing complexity identities by
one to seventeen branches and removes obsolete lint findings. Five duplicate
fingerprints replace reviewed pre-existing instance-store declarations, analytics
request arguments, and issue-filter contracts and assignments whose boundaries
changed when query functions began returning their data. The duplicate baseline
falls from 922 to 914 pairs after consolidating cycle-service methods and removing routing shims. One more
fingerprint replaces an unchanged cycle-store fragment beside a renamed service
call. The independent audit fixes replace seventeen more pre-existing clone
fingerprints: native router imports and link props, removed aliases, and scoped
store fields change their boundaries. Twenty-two stale fingerprints were retired,
and duplicated test setup was consolidated. No threshold, source exclusion, or debt
allowance increased.

The first MobX replacement stage removes admin's stores and the web notification,
home, quick-link, and sticky stores. The quality baselines retire their obsolete
findings. NotificationItem's reviewed complexity identity tightens from 37 to 21;
no diagnostic allowance increases. Shared OAuth forms, notification controls, and
drag handlers reduce the duplicate baseline from 914 to 871 pairs without adding
fingerprints. The lint scan covers 3,415 files after the reviewed deletions and
new query modules and tests. No threshold or source exclusion changed.

The public-page MobX removal and admin TanStack Form adoption reduce TypeScript
clone pairs from 871 to 801. Five replacement fingerprints refer to unchanged web
filter fragments previously paired with deleted public-page copies; both sides
were checked against HEAD. Seventy-five obsolete fingerprints were retired.
Shared filter sections, reactions, peek content, authentication email fields, and
admin form components remove repeated implementations. Public password-form
complexity tightens from 27 to 26 and peek-property complexity from 28 to 27.
No diagnostic allowance, clone threshold, or owned-source exclusion increased.

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

The native-state cleanup retires 15 Go lint allowances: three unchecked
cleanup calls disappeared with the shared attachment fixture, eleven shadowed
errors were renamed in the asset repository, and one shadow disappeared from
the consolidated asset service. New visibility tests check transaction rollback
errors. No diagnostic allowance increased.

The native issue-layout cleanup tightens 15 existing complexity identities in
both Oxlint baselines. These retain the same file and function responsibility;
the two anonymous findings belong to the existing Kanban rendering callback and
group drag handler. The list block falls from 42 to 29 and the drag handler from
26 to 20. Shared settings breadcrumbs and applied date-filter rendering remove
copied UI. No new lint allowance or source exclusion was added.
Eight duplicate fingerprints were also replaced after comparing both reported
source ranges byte for byte against the pre-cutover snapshot. Each retired
fingerprint was absent from the current scan; token and pair counts are unchanged.
Eighteen further replacements retain the same reported fragments after removing
only the MobX observer wrapper and formatting its function signature. Both source
ranges were reviewed for every pair, and all old fingerprints were absent from
the new scan. No duplicate allowance increased.
The same cleanup retires 52 obsolete Knip allowances for deleted stores,
unused wrappers and exports, and corrected test imports. The fresh scan has 374
findings and no new identities; Knip's source configuration is unchanged.
Shared parent issue display controls remove repeated cycle/module header handlers.
The cycle header no longer exceeds the complexity threshold; the module header's
existing allowance tightens from 24 to 17 in both lint ratchets. Display options
now derive exclusions without mutating the caller's array, with a mounted
regression for disabling and re-enabling cycle and module grouping.

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
migrator, and proxy output; it does not forward the API's stdout a second time.
`dev:check-logs` exercises a read-only HTTP 500 and requires both its error log
and matching server span, in addition to checking proxy ingestion.
This check also retires five existing Go lint allowances by handling response
cleanup and output errors, removing a shadowed variable, and normalizing errors.

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

The broader TanStack Form migration reduces the TypeScript duplicate baseline from
801 to 719 pairs. Five reviewed identities replace existing code-resend callers,
issue-search empty states, cycle peek navigation, and test resolver fragments.
Shared validation rendering, password fields, invitation forms, and modal frames
remove repeated code. Eighty-seven obsolete identities were retired; no duplicate
threshold or owned-source exclusion changed. Deleted estimate point/input
components, their obsolete hook, and the unused export modal reduce the lint scan;
these deletions do not exclude remaining source. Issue-modal complexity identities
also tighten from 30 to 22 and from 40 to 37.

The editor asset migration removes its root MobX store and consolidates file and
mention handlers across the editor wrappers. LiteTextEditor complexity tightens
from 25 to 23 in both Oxlint ratchets. The duplicate scan reports 707 pairs with
no new fingerprints; no duplicate allowance or source exclusion increased.

The identity/profile/settings migration reduces the duplicate baseline from 719 to
673 pairs. Independent review confirmed 20 changed fingerprints as inherited
fragments and five as unchanged filter fragments re-anchored after copied member
filters were deleted. Seventy-one obsolete fingerprints were retired; the 25
replacements introduce no additional duplication allowance. Shared member filters,
authentication fixtures and onboarding account-data setup remove actual copies.
IssueRootStore complexity tightens from 29 to 26 and WorkspaceMembersList from
20 to 19 in both Oxlint ratchets. No threshold, source exclusion or suppression
was added.

The workspace catalogue migration tightens the activity component's line-count identity
from 746 to 729 after extracting its label pill, and global-view header complexity
from 24 to 21 after removing redundant route guards and duplicate default-view checks.
The extracted label request also handles rejected loads, retiring an unhandled-promise finding.
One independently reviewed clone fingerprint replaces the inherited root-store
constructor/reset fragment after removing its obsolete workspace constructor argument.
Two retired fingerprints reduce the duplicate baseline from 673 to 672 pairs; no
threshold, source exclusion, or diagnostic allowance increased.

The workspace access gate now retries failed catalogue loads and stops rendering a
workspace after membership is revoked, even if older permission data is cached.
Using the catalogue's accessible slug removes redundant request guards and type
conversions; its complexity ratchet tightens from 33 to 27. A semantic sign-out
button also retires two accessibility findings and its promise-handler finding.

The navigation preference migration removes the remaining workspace preference store
and its hook. Removing the wrapper's obsolete preference fetches tightens its
complexity ratchet from 27 to 25, and retires two findings in the deleted store.
Independent clone review confirmed that removing its constructor/reset assignments
shortened the inherited root-store pair from 15 lines/133 tokens to 14 lines/124 tokens.
The exact fingerprint replacement and one already-retired allowance tighten the
duplicate baseline from 672 to 671 pairs. No diagnostic allowance or source exclusion increased.

The permission migration removes the legacy permission store and its obsolete lint
findings. Reviewed complexity reductions tighten both Oxlint ratchets: issue deletion
17 to 16, workspace dropdown 23 to 22, project sidebar item 23 to 21, and workspace
access wrapper 25 to 18. Two unreachable legacy sidebar-menu components are also
removed, retiring their two Knip entries and four full-lint findings. These changes
do not increase allowances or exclude source.

The permission migration reduces TypeScript duplicate pairs from 671 to 655. Twelve
independently reviewed fingerprints replace inherited page-create, cycle-layout,
quick-action, module-card and root-store fragments after native permission hooks
changed their boundaries. Twenty-eight stale occurrences are retired. Shared
project layouts, settings pages, Kanban prop contracts and test fixtures remove
actual copies; no duplicate threshold or owned-source exclusion changed.

Formula sample search adds an optional type filter before its result limit. An
independent review confirmed that its source design no longer duplicates the
deleted-item listing design. The Go duplicate baseline retires exactly that
obsolete count-two entry, reducing directed pairs from 505 to 503; no replacement
fingerprint, diagnostic allowance, or source exclusion was added.

The member-roster consumer migration removes the obsolete workspace/project membership
maps, write methods, and member filter stores. Native queries now supply settings,
selectors, and issue grouping. Eight resolved full-lint allowances and two obsolete
Knip entries are retired; IssueRootStore complexity tightens from 26 to 19. Reviewed
clone fingerprints retain only inherited fragments after scope-prop forwarding and
constructor cleanup. Shared filter rendering, default-member controls, and test
fixtures remove actual copies. Duplicate occurrences decrease from 650 to 644
(647 to 641 unique fingerprints); no threshold or source exclusion changed.

The native picker cleanup removes the unused web dropdown state/keyboard hooks and consolidates search
Escape handling in the shared input. Its reviewed lint baseline retires 44 fast and 92 full diagnostic counts;
six existing complexity diagnostics decrease. The independently reviewed clone baseline shrinks from 644
to 628 occurrences (641 to 625 unique fingerprints); its 24 changed fingerprints are inherited fragment
reanchors, including the existing Popper setup. No rule, threshold, or source exclusion changes accompany
these reductions. Local evidence is recorded in `tmp/tanstack-audit/picker-lint-baseline-review.json` and
`tmp/tanstack-audit/picker-clone-baseline-review.json`.

Canonical item-source extraction tightens IssueFormRoot's existing complexity
identity from 30 to 28 in both Oxlint ratchets. Source loading and conflict recovery
now have dedicated helpers; no diagnostic allowance or source exclusion increased.

## Current-value search migration

Schema migration 13 adds a separate rich-text search projection. Before upgrading,
stop older API and worker writers, then run the new `bootstrap schema` or
`bootstrap install` command to completion before starting the new application.
Do not perform a rolling upgrade across this boundary: older writers do not
maintain the projection. Frontend watchers may remain running during this schema
step; API requests will be unavailable while the backend is stopped.

Bootstrap sanitizes existing saved values without editing items, timestamps,
history, or activity. It processes bounded batches within each workspace transaction
and records that workspace's completion atomically. Failure or interruption rolls
back that workspace; a retry skips completed workspaces. Invalid legacy rich text
reports item and field IDs without logging their content. Repair the identified
value before retrying. `BOOTSTRAP_TIMEOUT` accepts a positive Go duration (default
`1h`); use a longer duration, for example `4h`, for large installations.

The command/sidebar/favorites/project-preference integration tightens seven reviewed
complexity identities: list root 22 to 19, Kanban root 28 to 25, calendar day tile
20 to 17, calendar issue block 29 to 28, workspace access wrapper 18 to 17,
module card 22 to 16, and project card 28 to 24. The issue store's line-count
identity tightens from 1241 to 1232 after moving its interface and pure property
maps into dedicated modules, removing a runtime store/helper dependency cycle.
These are lower diagnostic identities, not increased allowances. Inherited clone
fingerprint replacements are reviewed against captured pre-integration sources;
scanner runs without the configured baseline do not establish a passing ratchet.

The final quick-action cleanup shares modal state, copy payloads and dialog rendering,
and removes the unused copy-menu adapter. Seven retired clone identities reduce the
baseline from 627 to 620 occurrences. One replacement identifies the same shorter
module/project menu setup after that extraction; the configured scan reports 595
pairs with no new identities. Favorites test cleanup also retires four Go lint
occurrences for shadowed errors and direct error assertions. No allowance, threshold
or source exclusion increases.

Workspace-view metadata now uses session- and membership-scoped native Query reads
and mutations. GlobalIssuesHeader's reviewed complexity identity tightens from 21
to 20. Two reviewed clone fingerprint changes reanchor existing RootStore
constructor/reset and workspace/project filter HOC fragments; the clone baseline
remains at 620 occurrences, with no allowance added. The shared quick-action menu
renderer removes the copied menu implementation; the configured scan reports 590
clones.

The frontend test loaders use direct esbuild transforms for application TypeScript
and TSX. The standard limits remain two workers, a 384 MiB heap limit per Node
process, a 1 GiB sampled process-tree RSS ceiling, and a five-minute deadline. The
full frontend suite passed 953 tests in 279.5 seconds at 835 MiB peak sampled RSS.

The native analytics migration removes its MobX store and three unused components.
Both lint ratchets retire the dashboard's effect-driven tab state and one removed
project-store log statement. Header complexity decreases by one branch in both
cycle and module headers. Two independently reviewed clone fingerprints replace
existing constructor/reset and breadcrumb fragments; the allowance count stays
at 620. Three deleted-file Knip entries are retired. No threshold, source exclusion,
or diagnostic allowance increased.

The native label catalogue migration retires 44 syntax and 115 full typed-lint diagnostic
counts from the affected source. Three existing identities tighten: base issue-store length
1225 to 1211, sorting-population complexity 36 to 30, and ordering complexity 31 to 29.
Removing the LabelStore assignment shortens the inherited root constructor/reset clone
from 149 to 139 tokens; its exact fingerprint replaces the old one. The current TypeScript
scan reports 562 clone pairs with no new pair. No diagnostic allowance, source exclusion,
or duplication threshold increased.

The native state and project-view migration retires 11 syntax-lint occurrences and
31 full-lint occurrences belonging to removed stores and hooks. Ten existing
complexity or file-length identities tighten in each ratchet after native preference
selection and state projection cleanup. No diagnostic allowance, source exclusion,
or tool limit increases. Combined repository checks remain separate from these
reviewed reductions.

The calendar/Kanban migration deletes the two UI stores and their hooks. Its lint
baseline retires their obsolete findings and two removed calendar type assertions.
The existing Kanban complexity finding retains its value of 24 under the renamed
provider child component; its body is unchanged. No allowance or exclusion increases.

The collaborative editor resolves `@tiptap/y-tiptap` to release 3.0.9 because
3.0.10 drops the first text inserted into an empty block when a block-ID plugin
appends a transaction. The reproduction and release comparison are recorded in
[upstream issue #8584](https://github.com/ueberdosis/tiptap/issues/8584#issuecomment-6413802381).
No dependency source is patched. Remove the override when an upstream release
passes `packages/editor/src/core/extensions/unique-id/collaboration.test.ts` and
the page typing/reload integration check.

The utility subpath cleanup reanchors four existing clone fingerprints after
reviewing both fragments: two Vite configuration pairs, the editor table drag
handle imports, and the UI select imports. Only their import sources changed.
One further replacement covers the existing cycle/module issue-select setup,
where the observer and store read became a scoped native detail query. Each
replacement removes its old fingerprint and retains one allowance; the total
remains 617. New duplicated behavior is not covered by these replacements.

The final source inventory retires lint allowances only for files that no longer
exist: 122 syntax and 405 full-lint occurrences. The shared layout action and
presentation cleanup also tightens Kanban complexity from 22 to 21 and list
complexity from 17 to 16 in both ratchets. No diagnostic allowance increases.

Form-core 1.33.5 resolves `@tanstack/devtools-event-client` to its released 0.5.0
version. Version 0.4.4 keeps a Node process alive for about five seconds after a
form is unmounted while it retries a devtools connection. The narrow override
uses the upstream release's environment-aware entry point; no dependency source
or test runtime is patched. Remove it after a Form release incorporates
[the upstream dependency update](https://github.com/TanStack/form/pull/2410).
The process-lifetime reproduction is recorded on
[the existing upstream issue](https://github.com/TanStack/form/issues/2132).

The final native-state cleanup consolidates duplicated filter panels, dropdown
options, Gantt pagination, and confirmation behavior. The duplicate baseline now
contains 291 fingerprint identities (292 current reported pairs). Retired
identities were removed; reviewed import-only and unchanged dropdown fragments
retain one-for-one replacements after source extraction. No threshold or owned
source exclusion changed, and the final scan has no new duplicate identities.

The same shared-presentation cleanup tightens two existing complexity identities:
Gantt issue rows decrease from 17 to 16, and calendar issue blocks from 23 to 21.
Both syntax and type-aware ratchets retain those lower limits.
