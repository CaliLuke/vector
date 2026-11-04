# Technical debt catalog

This is the authoritative queue of outstanding technical debt. Git retains
completed work. Reviewed analyzer findings are recorded separately below;
they are not a count of confirmed defects or independent refactors.

## Remaining backend work

**EXT-01 — Dependency limitations.** Keep the upstream reports and removal
conditions in [dependency-refresh.md](dependency-refresh.md#dependency-limitations)
and the [Docker Desktop setup](development.md#docker-desktop-on-macos). The latest tagged
Testcontainers release still lacks the pinned HostPort fix. PostgreSQL cleanup,
Go overlay coverage, bodyclose, nilerr and errorlint limitations remain.
Dependency development is outside this repository's scope. A passing consumer
run alone does not establish that a workaround can be removed.

**Runtime investigations.** These observations have no confirmed cause or proof
that they are obsolete. Current passing runs do not close them:

- **NEW-04:** `TestModuleLinkEditsPreserveOmittedMetadataAndAdmitScopedDeletion`
  exhausted its one-minute fixture deadline at the initial parent lookup.
- **NEW-07:** consecutive two-second `ps` samples timed out in Go/frontend gates;
  the native-allocation regression also exceeded its seven-second harness limit.
  The original harness did not retain inspector output; current failures do.
- **NEW-08:** attachment import returned `internal_error` for the personal-token
  `empty_path` case. Current assertions retain the native mutation error.
- **NEW-09:** template reauthorization setup returned HTTP 503. A later page
  credential case failed during PostgreSQL SASL with a broken pipe; cleanup then
  waited on an authenticating backend's `ProcSignalBarrier`. The barrier explains
  the cleanup wait, not the preceding transport failure.
- **NEW-10:** milestone setup exhausted two minutes at initial creation and at
  database cloning for the archived-item association case. The latter failed
  before domain assertions. The independently reproduced Colima direct-address
  limitation is documented; its contribution to these failures is unconfirmed.
- **NEW-11:** OAuth proof cleanup missed its ten-second polling deadline. The
  original run omitted the attempt result and row count; current assertions
  capture both while checking expired-proof deletion and active-flow retention.
- **NEW-12:** MCP completed 320 top-level tests without case failures, then
  exceeded its 20-minute package deadline during template cleanup under elevated
  host load. Shared credential fixtures reduce measured setup cost, but the
  historical contention mechanism remains unconfirmed.

Close an observation only after reproducing and resolving its cause or proving
that the affected behavior is obsolete. Preserve current deadlines, memory
limits, fail-closed monitoring, tenant isolation, authorization, replay,
concurrency, durable effects and coverage scope. Do not repeat broad suites
merely to accumulate passing reruns of an unconfirmed failure.

## Retained review decisions

- **Tests:** credential-neutral domain checks run once. Credential/action
  matrices remain for authorization, replay and generated action smoke coverage.
  Compatible lock-wait scenarios share a database after restoring each credential;
  domain-reference mutations and fixtures requiring a fresh named catalogue keep
  isolation. Work-item type tests clone a migrated, seeded template; their writes
  and concurrent connections remain in independent databases.
- **Go diagnostics:** the [lint baseline](../configs/quality/go-lint-baseline.json)
  retains 36 findings: 22 validation/CLI text diagnostics, seven bodyclose/nilerr
  limitations, six exact sentinel comparisons and one intentionally non-wrapped
  logging regression. These preserve contracts; upstream analyzer limitations
  remain in EXT-01. No new allowances or suppressions were added.
- **Unchecked errors:** the [reviewed inventory](unchecked-error-inventory.json)
  retains 150 source-level ownership decisions, with no overlap with the lint
  baseline and no deferred policy decisions. These cover validated input,
  infallible typed encoding, best-effort bookkeeping and writes after response
  commitment. They are not 150 unhandled operational failures.
- **Duplication:** the [ratchet](../configs/quality/go-dupl-baseline.json) retains
  378 directed reports in 279 fingerprints: 303 authored-design reports, 17
  regression-fixture reports and 58 typed adapter/workflow reports. Distinct
  schemas, payload bindings, authorization policies, durable task identities and
  assertions remain explicit. Shared execution policy has one owner; do not add
  generic abstractions solely to erase similar declarations. New or increased
  duplication still fails the gate.
- **Backend coverage:** [168 package floors and 12 file floors](../configs/quality/go-coverage-baseline.json)
  remain enforced. The only zero floor is `internal/testsupport/mcpstub`, a
  transport test double whose unconfigured methods reject unexpected calls.
  It remains measured; asserting each canned rejection would add test quantity
  without product protection. All runtime packages and authored designs execute
  in the suite. Coverage is evidence of execution, not exhaustive behavior proof;
  new gaps should name the missing behavior rather than a percentage alone.

## Frontend work outside this batch

- **COV-03:** the 18 [coverage baselines](../.quality/) retain 1,903 files below
  the existing 80% line reference, including 1,510 zero floors, out of 3,184 files.
  Review meaningful missing behavior before adding tests; preserve per-file
  ratchets, source scope and new-file minimums.
- **NEW-02:** intermittent color-picker pending-save timeout at five seconds.
- **NEW-13:** command palette lifecycle exceeded 15 seconds during source
  coverage; the failed phase was not captured. Preserve initial focus, Escape
  handling and focus return.
- **NEW-14:** issue source-version refresh exceeded 15 seconds during source
  coverage; it passes alone in about 3.4 seconds. Preserve retained local edits
  and the submitted source-version assertion.

## Deferred product work

Teamspaces, production OAuth issuer selection and sophisticated MCP permissions
management remain deferred. Do not reactivate them as debt cleanup.
[Issue #6](https://github.com/CaliLuke/vector/issues/6), MCP management and client
setup UI, is a separate product follow-up.

## Working rules

Name an observable problem, its owner and closure evidence before starting a
batch. Do not increase baselines, suppress new findings or exclude owned source
to make a gate pass. Dependency development and new product scope require
separate authorization. Keep the checkout unchanged during publication hooks.
