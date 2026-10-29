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

The [remote MCP transport](mcp.md) is disabled by default. Its 30 tools expose
208 actions through personal tokens or generic external-issuer OAuth. OAuth
requires the [issuer profile and account linking](mcp-oauth.md). Generated-server
workflow tests cover all actions under both authentication modes and protocol
revisions; the [parity tracker](mcp-parity.md) records the remaining acceptance
checks. Mutations support optional [receipt and revision extensions](mcp-writes.md).
Teamspaces are deferred until native support exists.
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
duplication, enforced function complexity, and integration-tagged tests with coverage. Docker is required for
disposable PostgreSQL, proxy, and broker tests. Package concurrency is bounded.
The test-result guard rejects failures, incomplete reports, and unreviewed skips.
`.quality/go-test-skips.json` records tests that need explicitly provisioned
fixtures; each entry states its prerequisite. Never point these tests at a database
with valuable data.

Both fast and full Go gates enforce a maintainability index floor of 20 for new
functions. Higher scores are better. `configs/quality/go-maintainability-baseline.json`
records numeric floors for existing functions; any score decline fails even if
that function was already flagged. Identity uses file, package and receiver/function
name, not the source line. The pinned `maintidx` analyzer runs under the same
integration tags and generated-source exclusions as lint. The existing AST-based
function inventory resolves receiver identity; its complexity values are not gates.
Missing tools, analyzer failures, malformed reports and unknown functions fail.

Project-field validation now separates field parsing, reference existence,
cover-asset authorization, name validation, relationship scope, and HTML cleanup.
The coordinator's cyclomatic complexity falls from 106 to 12; each extracted
helper is at most 11. The refactor retires its maintainability allowance of 2,
and all extracted functions meet the floor of 20. Characterization tests preserve
field-error precedence and the session/API distinction in relationship errors.

The work-item refactor retires nine more production maintainability allowances,
including the zero-score list assembler. Field decoding, reference checks,
write preparation, persistence, effects, and response projection now have explicit
boundaries. JSON field decoding is shared by issue and collaboration validation.
Planning selection uses one assignment lookup while preserving target order and
repeated IDs. Existing authorization, transaction, error-precedence, and effect
contracts remain covered by the work-item suites.

| Work-item function      | Cyclomatic complexity before | After |
| ----------------------- | ---------------------------: | ----: |
| `listIssueResults`      |                          125 |    29 |
| `validateIssue`         |                          102 |    12 |
| `writeIssueTransaction` |                          102 |     6 |
| `writeCollaboration`    |                           83 |    27 |
| `validateCollaboration` |                           76 |     8 |
| `patchIntake`           |                           63 |    27 |
| `writeDraft`            |                           62 |    17 |
| `createIntake`          |                           49 |    30 |
| `planningTransaction`   |                           49 |    29 |

All extracted functions meet the maintainability floor. The lint baseline also
retires four shadowing occurrences and one redundant assignment. The duplicate
baseline remains at 499 directed pairs, without replacement fingerprints, added
allowances, or source exclusions.

The dead-code cleanup removes all 44 unreachable functions and leaves an empty,
enforced dead-code baseline. It also retires 27 lint occurrences, leaving 1,277.
Duplicate reports fall from 499 to 495: deleting an unused HTTP wrapper removes
one pair, and deleting an unused sync constructor joins two existing clone ranges.
Both retained sync ranges were compared with their original source and are unchanged.
The two zero-score assembly functions are also byte-for-byte unchanged; their
file fingerprints were reviewed after removing unrelated unused helpers.
Maintainability still has 44 allowances. The separate inventory includes 35 allowed
test skips and three inline suppressions in owned source; coverage retains 166
package floors (45 at zero) and five file floors. These remain subject to the
debt-removal work, not evidence that all quality debt is resolved.

The HTML compatibility refactor retires all three of its maintainability
allowances, leaving 41 across Go. Body-token dispatch falls from complexity 123
to 6; rendering separates node dispatch, doctypes, and elements; script tokenization
separates double-escaped states. Every extracted function meets the existing floor.
The unchanged parser corpus and added short-read script cases verify tree and token
behavior. One resolved lint occurrence lowers the Go lint baseline to 1,276.

Activity-row construction now dispatches to event-family handlers. Issue updates
keep their ordered stages visible: milestone, scalar fields and description
coalescing, references, archival, then collection changes. Pure handlers return
rows without database dependencies or error plumbing. The transaction owner and
index-derived activity IDs are unchanged; a database regression test verifies row
order when a description edit refreshes existing history. This retires one more
maintainability allowance, leaving 40, with lint and duplication counts unchanged.

Refactoring acceptance requires coherent responsibilities, narrow interfaces,
visible effect ordering, and less work to understand or change the behavior.
Passing a function-size or complexity metric alone is insufficient; inline or
regroup helpers that merely redistribute branches or introduce forwarding layers.

The legacy background-task database tests now clone an owned, migrated PostgreSQL
template. All 21 of their skip allowances are removed, leaving 14 elsewhere.
Running these proofs exposed missing workspace membership in an export fixture,
an outdated metrics service-name assertion, and a workspace-seeding defect:
whole-record inserts supplied NULL instead of new-column database defaults.
Seed inserts now name the captured columns, preserving defaults for later columns;
the existing rollback and replay proof verifies empty custom properties on all
seven seeded issues. Removing obsolete fixture setup also retires two lint
occurrences and one maintainability allowance, leaving 1,274 and 39 respectively.

Email preparation decodes positional arguments once before recipient lookups or
message construction. Task metadata owns the expected argument counts; the
decoder preserves count-before-type errors, argument order, and null-to-empty
values. Template-specific subjects, URLs, and database effects remain together.
Characterization tests and the database/SMTP proofs pass. This removes one lint
occurrence and one maintainability allowance, leaving 1,273 and 38 respectively;
the duplicate baseline remains at 495 directed reports.

Analytics export separates filter/SQL preparation from distribution reading and
table formatting with one query-plan value. Segment totals, null estimates,
missing dimensions, array filters, and priority ordering are characterized against
PostgreSQL before and after the extraction. The analytics allowance is retired,
leaving 37; lint and duplication baselines are unchanged.

Issue notification fanout uses one event context shared by recipient loading and
subscriber/mention delivery. The coordinator retains the transaction and ordering
of subscriptions, mention synchronization, delivery, and commit. The producer's
optional workspace binding remains distinct from the stored workspace; SQL,
recipient permissions, and replay keys are unchanged. Database proofs cover
completion preferences, membership revocation, recipient policy, and replay.
This retires one maintainability allowance and six lint occurrences, leaving 36
and 1,267 respectively; the duplicate baseline remains at 495 directed reports.

Unused-code cleanup removes three unreachable design helpers, eight unreferenced
schema declarations, four obsolete constants, and two unused test declarations.
Page-lifecycle helpers also lose two redundant parameters. Fresh generation before
and after preserves OpenAPI and transport behavior; repository regeneration changes
only the three affected manifests. The blanket test-file `unused` exclusion is
removed, and all owned Go source now has zero `unused` findings. Seventeen lint
allowances are retired, leaving 1,250. The remaining test-file `unparam` exclusion
hides 24 findings and still requires removal; it is not covered by that lint count.

The test-file `unparam` exclusion is now removed too. Fixture helpers accept only
values their cases vary, and session doubles explicitly satisfy the optional
production interface that middleware discovers at runtime. New failure cases
cover cycle/module metadata lookups, state authorization, invitation reads, and
expired-session cookies. Shared proxy assertions and table-driven catalogue and
work-item precedence proofs remove seven duplicate reports, leaving 488; checked
response closure retires three lint occurrences, leaving 1,247. The proxy suite
passes 847 tests/subtests; affected database workflows and unit suites also pass.
Neither test-file exclusion is replaced by a baseline or inline suppression.

Shadow cleanup gives local errors operation-specific names and distinguishes
request contexts, compatibility options, and validated values from their outer
bindings. It also removes one obsolete loop-variable capture. Binding-aware
renames preserve every identifier reference; no helper extraction or control-flow
change is needed. All 593 baselined shadow diagnostics are retired, and the 20
additional declarations exposed while resolving nested shadows are fixed too.
This leaves 654 lint occurrences and no shadow findings. Maintainability remains at 36
allowances with identical scores. The two zero-score assembly-file fingerprints
are reviewed identifier-only updates. All 488 duplicate report ranges are
identical; one fingerprint pair is reanchored for the renamed cache-invalidation
errors in instance management. No threshold, exclusion, or suppression changes.
The untagged Go suite passes 9,117 tests/subtests across 105 packages; its seven
existing opt-in skips remain tracked separately for removal.

Ten more opt-in skips are removed by giving database proofs disposable fixtures.
UUID and three sticky proofs already owned their containers; their redundant
flags are deleted. Bootstrap and public-asset proofs share empty-database
ownership, sticky differential comparison uses two clones of the migrated schema,
and parity keeps one repeatability proof instead of two copies. Adoption tests
start from the historical Django schema and cover missing/empty ledgers and
partial-schema rejection independently of fresh installation. The asset fixture
includes the draft-issue column now read by the repository. All 24 affected
tests/subtests pass without skips. Four service-backed skip allowances remain.
Maintainability drops to 35 allowances, three checked connection closures reduce
lint to 651 occurrences, and the 488 duplicate reports need no baseline changes.

The final four service-backed skips now use owned Valkey, RustFS, and PostgreSQL
fixtures. Existing notification and attachment tests share those service owners.
The metadata crash proof still kills its subprocess after the committed write,
then requires a successful replay, identical metadata, and at least two attempts.
Bootstrap installation runs twice against an empty owned database and bucket.
The combined service/topology/HTML run passes 30 tests/subtests, and shared
notification/attachment checks pass six more. One initial container-reaper startup
timeout was followed by successful isolated and combined installation runs;
no timeout or cleanup policy was relaxed. The skip allowlist is empty, and owned
Go tests contain no skip calls. Missing Docker fails the topology check, and HTML
fuzzing now exercises inputs above the old 64 KiB cutoff under the production
size limit. Checked Redis-client cleanup reduces lint to 650 occurrences;
maintainability and duplication remain at 35 and 488 respectively.

Customer and release command normalization now separates command shape from
field conversion. Each domain has one contract validator and one field normalizer
that owns its map copy; receipt preparation and rich-text processing retain their
existing order. Required identities, allowed fields, canonical property IDs,
collection ownership, and JSON encodability keep their existing behavior. Twenty
public mutation-contract tests and 38 catalogue tests pass before and after the
change, including retry replay and changelog escaping. The two production
maintainability allowances are retired, leaving 33; lint and duplication remain
unchanged at 650 occurrences and 488 directed reports.

All owned Go inline lint suppressions are now removed. Bootstrap transaction
cleanup checks rollback failures, preserves the original operation error, and
uses a bounded cleanup context even after caller cancellation. All five Redis
cache clients now use standard TLS certificate verification for `rediss` URLs.
Private deployments must install their CA in the host/container trust store and
use a certificate matching the Redis hostname. This intentionally rejects
previously accepted untrusted certificates. The bootstrap and public-metadata
packages pass 104 tests/subtests, and application-level probes verify certificate
rejection through each cache client (six tests/subtests). Lint, maintainability,
and duplication remain at 650, 33, and 488 respectively without baseline changes.

Cycle updates now separate date normalization from persistence and webhook
admission. Create and patch share the same range validation and timezone
conversion. Patch retains the distinction between omitted, null, single, and
paired dates: only paired dates are range-checked and expanded to project-day
boundaries. PostgreSQL proofs pass before and after the extraction, including
unchanged state and no webhook on invalid dates; the complete cycle suite passes
129 tests/subtests. This retires the cycle patch maintainability allowance,
leaving 32. Removing an overwritten assignment, an empty branch, and an unused
timestamp success flag retires three lint findings, leaving 647. No duplicate
report or source exclusion was added.

The full integration checkpoint after the cycle date extraction passes 12,215
tests/subtests across 167 packages with zero test skips. Fresh cross-package
coverage passes all existing floors. Review of all 166 measured packages raises
67 percentage floors without lowering any; the five file floors stay unchanged.
Four former zero floors now enforce measured coverage: bootstrap CLI 31%, password
strength 66.67%, personal dashboard 66.67%, and workspace search 61%. There are
still 41 zero package floors. Statement counts are refreshed from this complete
run, including owned designs and CLI packages.

A separate strict errcheck audit with `check-blank: true` identifies 1,294
additional discarded-error findings (467 production, 827 test) beyond the 303
errcheck findings in the configured gate. The current configuration does not
enforce these yet. They remain outstanding work for the zero-exceptions goal;
no baseline allowance was added for them. The configured lint baseline remains
647 occurrences, distinct from this additional audit inventory.

MCP discovery now separates current permission queries from the action catalogue.
A capability value contains the existing domain grants; a pure projection applies
token scope and builds action visibility. Query order, conditional reads, and
failure behavior are preserved. Repeated write predicates are named once.
The fresh applied-source build passes all 831 MCP tests/subtests, covers the
extracted functions through their current source ranges, and raises the MCP
coverage floor from 83.73% to 83.82%. Its maintainability allowance is retired,
leaving 31; configured lint and duplication remain at 647 and 488.

Do not combine source overlays with `-coverpkg` for verification until
[golang/go#73802](https://github.com/golang/go/issues/73802) is fixed in the pinned
toolchain and its reproduction passes. The issue can ignore overlaid code and
produce a false passing test. A standalone Go 1.27.3 reproduction was
[added upstream](https://github.com/golang/go/issues/73802#issuecomment-6480328772).
This batch used a forced fresh build of the applied MCP source and checked the
coverage ranges; no compiler flag or dependency patch is retained in the gates.

The test cleanup pass checks discarded errors while preserving the caller's
close/defer timing. The shared `testsupport.Close` and `CloseContext` helpers
report errors and allow other cleanup to continue; context cleanup preserves the
caller-supplied context. The audit also removed duplicate listener ownership from
metrics fixtures, made lost-commit proxy shutdown close each resource once, and
made restart-test cleanup follow the currently open journal.

This retires 131 configured lint occurrences (647 to 516), leaving 319 lint
fingerprints. The strict errcheck inventory falls from 1,597 to 1,316 findings;
1,144 remain beyond the configured gate (467 production, 677 test). These extra
findings have no baseline allowance and remain part of the zero-exceptions goal.
Maintainability remains at 31 allowances. Duplication remains at 488 directed
reports / 376 fingerprints, with the existing fingerprints unchanged. All clone
bodies match the original report after accounting for the reviewed cleanup edits.
No new clones were admitted, and coverage floors were not lowered.

The Go fast gate passes with the reduced baseline. The cleanup helpers cover all
six new statements, the mutation-receipt suite passes 39 tests/subtests with the
race detector, and 73 affected HTTP and metrics tests/subtests pass. The Notion
suite also passes after the journal ownership fix. The integration sweep and
corrected-package reruns exercise 12,219 tests/subtests with zero test skips;
the five initial cleanup failures pass after their ownership fixes. The complete
repository gate remains required before the zero-exceptions goal can be complete.

Runtime construction and shutdown now share one resource owner for the activity
store, email store, and database pool. It removes 27 discarded cleanup errors,
covers partial construction, and preserves all shutdown failures. The stricter
audit now has 1,117 findings beyond the configured gate (440 production, 677 test).
The constructor allowance tightens from score 7 to 8; it remains one of 31
allowances. The background-task suite passes 187 tests/subtests and its command
and focused cross-package consumers pass another 2,226, with zero test skips.
Combined fresh coverage raises the package floor from 66.10% to 66.69%; configured
lint stays at 516 and duplication at 488, with no new exceptions.

Workflow assembly now follows five responsibilities: asset storage, content
history and synchronization, email delivery, exports, and work-item effect
delivery. The runtime object stays private until every registration succeeds;
its public fields and outer resource ownership are unchanged. A source audit
confirmed all 28 field bindings and 86 factory/call expressions retain their
order and inputs after expanding the methods. The constructor and every extracted
method meet the existing maintainability floor, leaving 30 allowances.
The lifecycle integration test now starts the assembled runtime and verifies its
durable periodic schedules before shutdown. Fresh combined coverage is 66.79%;
the 66.69% floor is retained, with its statement inventory updated for the new
boundaries. Configured lint, duplication, and the stricter error inventory remain
unchanged; no new exceptions were introduced.

Cycle date filters now own their temporal grammar and argument binding in one
helper. The scalar/relation dispatcher keeps its existing responsibilities;
requester-calendar filters and legacy exact/range database casts retain their
separate semantics. Characterization tests pass before and after the extraction,
and all 4,405 affected tests/subtests pass without skips. Both functions meet the
maintainability floor, retiring one allowance and leaving 29. Configured lint and
duplication remain at 516 and 488. Fresh combined cycle coverage is 75.96%; the
75.40% floor is retained and its statement inventory is updated to 1,805.

Page-command normalization now delegates permitted field values and HTML
sanitization to one helper over the cloned field map and allowed names. Command
routing, required identifiers, authorization, and transactions keep their existing
owners and ordering. Boundary tests cover strict value types, Unicode limits,
sanitization, and caller-map isolation before and after the move. All 274 affected
tests/subtests pass without skips. This retires another maintainability allowance,
leaving 28; lint and duplication remain unchanged. Fresh combined page coverage
is 64.12%, retaining the 63.48% floor with 1,572 owned statements.

API label mutations now separate serializer validation and whole-tree deletion
from create/update persistence. The serializer has no database access; parent
scope checks and hierarchy locks remain inside the existing transaction. Source
comparisons confirm both extracted bodies preserve the original operations and
error mapping. All three functions meet the maintainability floor, leaving 27
allowances (10 production, 17 test). Final affected runs pass 1,042 tests/subtests
without skips; one PostgreSQL fixture connection failure passed on a complete
label-read package rerun. Including MCP discovery preserves the visibility
coverage omitted by the focused selection: combined coverage is 55.43%, retaining
the 55.16% floor with 857 statements. Lint, duplication, and strict error findings
remain at 516, 488, and 1,289 respectively.

Workspace invitation eligibility now has one operation owning the actor lookup,
role ceiling, live-workspace lookup, and existing-member check. A source comparison
confirms the query/check order and inputs are unchanged. Shared email validation,
whole-batch insertion, commit-before-publish behavior, and project invitation rules
retain their existing boundaries. Both functions meet the maintainability floor,
retiring another allowance and leaving 26. Final affected runs pass 643
tests/subtests without skips. A consumer run stopped after PostgreSQL startup
timeouts under host pressure; serial reruns passed. Including cache TLS and
project/workspace creation consumers yields 69.33% fresh coverage, retaining the
69.24% floor with 1,575 statements. Lint and duplication remain at 516 and 488.

Fourteen redundant type conversions are removed across twelve owned source/test
files. The cleanup preserves the conversions that turn bytes into raw JSON and
leaves response encoding, interfaces, and transport generation unchanged. The six
affected package suites pass 3,549 tests/subtests without the integration build
tag or any skips, including the asset bindings generator check. The reviewed lint
baseline removes only twelve `unconvert` fingerprints (fourteen occurrences),
leaving 307 fingerprints and 502 occurrences. The linter remains enabled with no
allowances; maintainability stays at 26 and duplication at 488.

Session test fakes use `Request.CookiesNamed` when missing cookies are ordinary
absence. The pinned standard library uses the same parser and ordering as
`Request.Cookie`; the admin fake still selects the first matching cookie before
validating its value. This removes seven error-shaped presence checks without a
new helper or suppression. All 2,196 HTTP assembly tests/subtests pass without
skips in the default suite. The reviewed baseline removes seven `nilerr`
occurrences, leaving 495 lint findings; maintainability and duplication stay at
26 and 488. Production coverage floors are unchanged.

Private helper results now match their consumers: route matching returns a
predicate, mounting and stack popping expose only their effects, and cycle
creation, page JSON encoding, list paging, and asset-column mapping drop unused
results. Cycle create and patch reuse the existing API throttle operation,
preserving permission-before-throttle order and rate response effects. No helper
functions are added. The reviewed lint baseline retires seven `unparam` findings,
leaving 488 occurrences and 295 fingerprints. Duplication remains at 488 and
maintainability at 26. The HTTP assembly allowance only refreshes its file hash;
the assembly function itself is byte-for-byte unchanged. The full integration
suite passes 12,255 tests/subtests with no skips. Its coverage check exposed a
0.01-point HTML compatibility shortfall after statement removal and a 0.12-point
shortfall in unchanged Notion sync code. Focused tests now protect HTML tree
ownership on rejected mutations and reject truncated Notion page responses; both
package suites pass another 1,838 tests/subtests. Combined current-source coverage
passes all 166 package floors and five file floors (HTML 93.85%, Notion sync
80.90%). All percentage floors are retained. Six statement inventories are
refreshed, including the six cleanup-helper statements added earlier in
`1f460c6e` that were absent from the test-support inventory. The strict error audit
still reports 1,289 findings, including 1,117 beyond configured errcheck coverage.

Private parameter cleanup retires all 29 remaining `unparam` allowances while
keeping that linter enabled. Narrow constructors expose their actual feature
inputs, the sticky text-field forwarding wrapper is removed, and asset response
headers no longer capture unused mount data. SQL scope predicates, authorization
order, public session tuples, and actual runtime debug controls are preserved.
An AST call audit checks removed arguments for evaluation effects. Two identical
instance-read failure tests now share a table with their original routes, cookies,
and response assertions. Final affected runs pass 5,441 tests/subtests without
skips, including the 144-case integration-tagged cycle suite. Regenerating only
the two changed designs leaves transport code unchanged; their manifests refresh
the design digests and record the already-pinned generator version. Lint now has
459 occurrences and 266 fingerprints; duplication stays at 488 and maintainability
at 26. The assembly-file hash review confirms only the unused middleware closure
call changed inside the assembly function. Coverage policy is unchanged; the
complete repository gate remains required before this cleanup goal is complete.

Module issue listing now separates page SQL construction from query execution and
result decoding. The two extracted operations own complete responsibilities;
filtering, authorization, and property binding retain their order. Source
comparison confirms the SQL and decoding bodies are unchanged. This retires the
module listing allowance, leaving 25 maintainability allowances; lint and
duplication remain at 459 and 488. All 4,330 tests/subtests across the five affected
packages pass with no skips. Fresh module coverage is 65.59%, above its retained
65.51% floor; the statement inventory grows from 1,447 to 1,450 for the extracted
calls and return. The complete repository gate remains pending.

Removing redundant state retires the last ten `ineffassign` findings, the last
configured `govet` finding, and three `SA4006` findings. Workspace-insights mounting
shares its existing middleware factory; exhaustive branches assign values without
unused initializers. Attachment creation retains permission checks and assigns
its workspace ID at the active-workspace lookup, preserving API-specific errors.
No helper functions are added. The baseline removes fourteen findings and eleven
fingerprints, leaving 445 occurrences and 255 fingerprints; all analyzers remain
enabled. Maintainability stays at 25 and duplication at 488. The zero-score HTTP
assembly fingerprint is refreshed after reviewing its three local changes.
Final affected runs pass 5,066 tests/subtests across nine packages without skips.
New cases protect quoted Accept parameters and reject unsafe export ordering
before a database query. Fresh coverage passes all six affected package floors;
five statement inventories are refreshed, including earlier parameter cleanup.
All percentage floors are retained. The complete repository gate remains pending.

HTTP response bodies retain explicit checked closes where the current bodyclose
analyzer cannot follow a cross-package cleanup helper. This limitation is tracked
in [timakin/bodyclose#60](https://github.com/timakin/bodyclose/issues/60), with a
[current standalone reproduction](https://github.com/timakin/bodyclose/issues/60#issuecomment-6480673025).
Those call sites can use the shared helper when the pinned analyzer accepts that
reproduction. No linter suppression or dependency patch was added.

The initial inventory at `4d7deab2` contains 54 allowances below 20, including
35 production functions and the current MCP delivery. Five functions already
score zero: startup assembly, HTTP assembly, activity-row construction, HTML
parsing, and work-item listing. A zero cannot decline further, so changes to
those files require explicit review through a source fingerprint; this is a
conservative file-level review trigger, not a claim that every edit worsens the code.

These allowances expose existing debt rather than establish structural health.
[Draft lifecycle extraction](https://github.com/CaliLuke/vector/issues/2),
[HTTP registration separation](https://github.com/CaliLuke/vector/issues/3), and
[invitation responsibility separation](https://github.com/CaliLuke/vector/issues/4)
track confirmed structural refactors. Function scores do not replace those reviews;
automated structural analysis remains blocked by
[Lagotto's Go 1.27 loader incompatibility](https://github.com/CaliLuke/lagotto/issues/58).

Run `bash scripts/quality/go.sh --maintainability` for the focused gate. An explicit
`--write-maintainability-baseline` regenerates allowances for review; normal checks
never change the baseline. Improvements are reported so their floors can be
raised. Do not refresh the baseline to accept a newly failing implementation.

Coverage instruments the same owned Go packages that the ratchet measures, so
integration tests also count calls into other packages. Repeated coverage blocks
from separate test binaries are merged by source location; a hit in either binary
covers the block once. Generated-source exclusions and package floors still apply.
The parser streams large profiles instead of loading their complete text into memory.
Optional file floors protect established contracts independently of the package average.
The baseline refresh command preserves those file floors.

The initial MCP floor covered 30 identity statements at 100%.
The expanded package contains 3,547 statements, with a reviewed aggregate floor of 82.94%.
Identity, member directory, project reads, project members and service construction retain explicit 100% file floors.
New domain, design and test-support packages have explicit initial floors, including visible zero-coverage entries.
Other existing package floors remain unchanged.

The work-item type tests, full-schema MCP workflows, and MCP identity tests migrate
disposable templates per package and clone a separate database for each test. Test data and
connections are never shared between those clones. Migration-specific tests still
exercise fresh database installation directly.
PostgreSQL fixtures verify a SQL query through the published port before running
seed migrations. Only readiness is retried; a failed migration is returned once
and its container is reclaimed.
The macOS environment keeps Testcontainers' resource reaper enabled so containers
are reclaimed when a test process exits before its normal cleanup. Explicitly
disabling it requires an external runner that owns cleanup; see the
[Testcontainers cleanup guidance](https://golang.testcontainers.org/features/garbage_collector/).

The full Go gate and frontend pipeline run concurrently. Within the frontend pipeline,
production builds run first, followed by TypeScript checks; builds and checks remain
sequential so generated package output is ready before typechecking. Frontend tests
run in two sequential shards, each with two isolated workers, a 384 MiB JavaScript
heap limit, a 1 GiB sampled process-tree RSS ceiling, and an external five-minute
deadline. Every frontend file runs once; either shard failing fails the gate.
Individual tests retain their 15-second timeout. Each TypeScript gate
has a ten-minute deadline and 3 GiB RSS ceiling;
production builds use one Turbo task at a time with a 4 GiB RSS ceiling. The Go
gate has a 6 GiB RSS ceiling and two Go runtime threads. These are process-group
watchdogs sampled every 100 ms, not OS-enforced total-memory reservations. They
retry one timed-out process sample; consecutive sample timeouts or other
monitoring errors fail the gate. The external command deadline remains active.
They track descendant processes and their groups, including Turbo task groups, but
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

The frontend behavior suite uses Node's compilation cache and a shared TypeScript transform cache.
Transform keys include source content, compiler version and every transform option.
Workers publish cache entries atomically under ignored `node_modules/.cache` directories.
These caches preserve worker isolation and do not cache test results.
Node's compilation cache is scoped to behavior tests, because it can reduce V8 coverage precision.
Coverage suites use separate commands without that cache setting.

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

Adding canonical cycle status filtering removes two obsolete duplicate pairs between the cycle and module API declarations. The Go duplicate baseline falls from 501 to 499 pairs; no allowance or exclusion was added.

The milestone activity visibility checks add the active-user column to two existing
history/profile SQL fixtures. Their duplicated schema spans were compared with
HEAD; only that column changes. One duplicate fingerprint (two directed reports)
was replaced, with the total unchanged at 499. No new allowance or exclusion was
added.

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

The seven [work-item relation actions](mcp-work-item-relations.md) are wired through the native relation domain. Custom definitions require a workspace administrator; edge mutations require writable endpoint projects. The core workflow batch covers these actions across both authentication modes and protocol revisions. Durable worker tests cover relation history and replay; remaining acceptance is tracked in [MCP parity](mcp-parity.md).

The [property API](mcp-work-item-properties.md) wires eleven catalogue actions.
Migration 000032 adds API metadata and revision triggers. Native preservation
checks still govern destructive edits. Item-value and configuration workflows cover
formula, read-only content and cascading paths. The consolidated quality and
acceptance checks remain tracked in [MCP parity](mcp-parity.md).
