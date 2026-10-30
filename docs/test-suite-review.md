# Test suite review

Reviewed on 2026-10-22 during the Loom dependency update and quality-gate setup.
The review sampled Go services, repositories, HTTP contracts, frontend regression tests,
Live, and codemods. It is not a proof that every test or product behavior is correct.

## Findings to address

| Area                   | Evidence                                                                                                                                                                          | Recommended change                                                                                                                                                                                                                                                                                                               |
| ---------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Restricted page reads  | `internal/pagereads/restricted_search_test.go` and `detail_visibility_integration_test.go` cover denied searches and page visibility.                                             | Fixed: denied searches return the same typed response without reading page content or recording visits. Child pages and pages without a visible project now return not-found for restricted guests too. Owner and explicit guest-feature access remain covered.                                                                  |
| Cycle grouping         | `internal/cyclereads/dynamic_grouping_integration_test.go` exercises real PostgreSQL grouping, pagination, and tenant/project filtering.                                          | Fixed: `created_by`, `start_date`, and `target_date` grouping works; null-state rows remain in primary and nested groups. Nested page totals use the largest result bucket. Invalid grouping fields retain typed validation errors.                                                                                              |
| Intake validation      | `internal/intakeinboxreads/pagination_validation_test.go`, `draft_filters_integration_test.go`, and `intake_validation_http_test.go` cover parser, PostgreSQL, and HTTP behavior. | Fixed: malformed pagination/status values and unsupported draft filters return typed 400 responses. Native draft label, assignee, cycle, and module filters work. Mixed empty/specific selections use OR and ignore deleted relation links. Large valid offsets use bigint arithmetic; supported numeric formats remain covered. |
| Label write failure    | `internal/projectlabelreads/finite_sort_order_integration_test.go` and the label numeric schema tests in `internal/bootstrap` exercise real storage.                              | Fixed: app and API label writes reject non-finite order values before mutation. A migration repairs historical invalid label orders and adds a finite-value constraint.                                                                                                                                                          |
| Source-text assertions | `design/user_identity_contract_test.go`, `design/workspace_project_reads_contract_test.go`, and `cmd/system/loom_error_status_contract_test.go`.                                  | Fixed: these checks now inspect generated OpenAPI routes, operation IDs, fixture statuses, parameter formats, response schemas, and headers. Existing generated-client tests continue to verify status-based error decoding through HTTP.                                                                                        |
| PDF recovery logging   | `apps/live/tests/services/pdf-export/effect-utils.test.ts` claimed to verify warning logging but observed recovery.                                                               | Fixed: renamed the test to describe continuation after recovery. It makes no logging assertion.                                                                                                                                                                                                                                  |

Go paths in the table are relative to `apps/go-api`.

Cycle issue pagination rejections now return typed JSON 400 responses for zero or
negative ungrouped page sizes, invalid previous-page slices, and offset overflow.
Generated-server tests cover direct and wrapped repository errors and verify that
unexpected failures still return a sanitized 500 response.
Adjacent cursor arithmetic also rejects page values that cannot represent both
neighbors, and next-page detection cannot overflow at the last valid offset.
PostgreSQL tests retain supported infinite first-page cursors and large grouped
bounds without changing their existing results.

Page list and detail requests now return typed JSON 400 responses for invalid or
empty ordering fields. The source design declares the list response, and the
obsolete HTML 500 adapter is removed. Tests exercise GET and HEAD through the
production middleware and prove that rejection stops after authorization without
reading page content or recording a visit. Page-read validation responses retain
the generated error-name header so clients can distinguish ordering and search
errors that share status 400. Unexpected failures remain sanitized 500 responses.

Malformed project-member writes now return JSON 400 responses for non-object
bodies, malformed bulk entries and invalid role values. Permission checks retain
their existing order. Generated-server tests check response bodies and prove that
rejected requests perform no writes or notifications. PostgreSQL regressions
compare memberships and member properties before and after rejection. Existing
permission, identity and successful-write tests continue to cover accepted inputs.

The audited clipboard actions now report success only after the browser accepts
the copy. Failures produce feedback and permit retry. The shared fallback checks
`execCommand` results and removes its temporary input even on failure. Public
pages use that same helper. Copying remains available without a Permissions API
probe, since the `clipboard-write` permission is not portable across browsers
([MDN](https://developer.mozilla.org/en-US/docs/Web/API/Clipboard_API)). Tests
cover rejection, retry, focus restoration, event handling, timer replacement,
and completion after a view or account changes. Scoped feedback in the editor,
work-item toast, inbox, cycle, module and view menus stays with its initiating
request. Rich-text clipboard formats retain their separate checked copy path.

All floating-point ordering columns now have finite-value constraints. Migration
43 repairs historical NaN and infinity positions to a fallback order of 65535.
PostgreSQL tests cover all 20 additional columns alongside the existing label
checks, including both tenants, soft-deleted rows, unchanged finite values and
metadata, and the existing revision triggers.

Activity timestamps now reject non-finite values before worker effects and at the
database boundary. Migration 44 repairs historical non-finite epochs to NULL, the
existing unknown-timestamp representation. Migration tests preserve all other
activity metadata across two tenants, including soft-deleted records, and keep
finite and missing timestamps unchanged.

Asset sizes also have a finite-value constraint. Migration 45 repairs historical
non-finite sizes to the existing missing-size fallback of zero. Tests cover
fractional legacy values, global and workspace assets, soft deletion, preserved
object metadata, and revision updates only for repaired rows. Attachment reads
use metadata sizes only when they contain finite nonzero numbers. Generated MCP
tests cover numeric list results through both credentials and protocol versions;
legacy upload tests verify invalid sizes cause no storage or activity writes.

Type levels now enforce the hierarchy editor's integer range of 0 through 20.
Migration 46 restores invalid levels from consistent project copies, or uses the
unassigned level of zero when no copy exists. It validates affected work-item and
private-draft relationships, preserves metadata, and advances native and MCP
revisions. Ambiguous copies or invalid final links abort the transaction without
unlinking records. Tests cover legacy adoption, soft deletion, both tenants,
concurrent repair, and the documented journal recovery and retry procedure.

The remaining numeric writer audit now covers cycle and sticky HTTP create/update
paths, native state and member ordering, and native work-item writes. Rejected
values return 400 before persistence; PostgreSQL checks preserve stored rows,
revisions, sequence allocation and durable effects. Existing custom-view and
asset tests cover the other recorded parser consumers. Finite fractional ordering
remains accepted. The solar-coordinate columns belong to the retained Celery
schema and have no active application writer; this review makes no claim about
historical values in those tables.

## Coverage and execution gaps

The previous CI omitted Live and codemod tests. The default Go command omitted
tests behind the `integration` build tag. Live's coverage configuration included
`.ts` files but omitted `.tsx` PDF renderers. The gate setup addresses test
discovery and coverage reporting; passing those gates does not resolve the
product-contract questions above.

The final workspace inventory also found the constants endpoint URL suite outside
the full gate. It now runs in the same TypeScript gate; its source coverage remains
explicitly unmeasured in the coverage-scope inventory.

Many repository tests capture SQL text and return canned rows. These are useful
for query construction but do not prove that SQL executes correctly or enforces
tenant isolation. Prefer PostgreSQL tests with two tenants, membership changes,
soft deletion, and persisted outcomes for these requirements.

Codemod output snapshots exercise real transformations and protect useful edge
cases. Some also depend on printer whitespace. Use semantic AST assertions where
formatting and comment preservation are not the behavior under test.

## Valid compatibility coverage

Keep tests for supported routes, response schemas, accepted input formats,
session signatures, authorization, and durable effects. The presence of “Django”
in a test name does not make it obsolete: existing clients and stored sessions
still rely on some of these contracts. Corrected behavior must have its own
explicit assertion rather than inheriting expected failures from old fixtures.

The frontend loading/race tests, Live persistence tests, PDF byte/content tests,
and generated-client response-contract tests exercise meaningful behavior and
should be retained.

The Tiptap 3 migration adds real-editor tests for formatting preservation through
a JSON round trip, suppressed update events during content replacement, and a
read-only ProseMirror view. These complement the existing stub-based no-write
tests; they do not establish full collaborative hydration correctness.

## Test repair during verification

`TestPostgreSQLReaderUsesDatabaseExpiryAndDoesNotMutate` refreshed a session using
a fixed September 2026 timestamp, then validated it against the database's live
clock. This failed after that date. It now reads PostgreSQL's clock before
refreshing, retaining the persistence and expiry assertions.

Local UI setup also exposed an untested initial-admin signup path. Successful
account creation returns both the new user and `SetupDone: true`; the redirect
service treated that flag as a concurrent signup and returned an error after
persisting the account. It now distinguishes successful creation from another
request completing setup. Regression tests check the successful session and
redirect, and the concurrent-signup rejection without a session.

The development launcher started shared-package builds alongside watchers that
write the same output directories. It now completes those builds first. A test
executes the launcher with fake commands and checks that watchers start only
after a successful build, and never after a failed build.

The React Hook Form update exposed another incorrect dependency: the cover-image
picker registered its image-search field in the parent project/profile form,
although search belongs to the picker. The input now uses its local state. A
mounted-component regression test types a query, submits it through the actual
controls, and verifies the request and retained text without a parent form or a
cover-image mutation.

Headless UI 2 requires a combobox trigger to be a single element that can receive
its ref and handlers. Eight dropdowns passed a Fragment containing their button;
the wrappers are now removed. The priority-dropdown regression test exercises
opening, Escape, and focus restoration with the real Headless UI components.

Tooltip tests now verify keyboard access for both historical `renderByDefault`
settings and object/callback-ref cleanup. The target handlers always mount, while
Blueprint still lazily renders the overlay. Sticky-layout tests verify card
reload/layout, resize notifications, cleanup, and cancellation when an async
layout-library import completes after unmount.

Headless UI tests now create the DOM before importing browser-sensitive modules
and flush animation-frame focus/registration work inside React's `act`. Closing
assertions wait for the actual transition to finish rather than requiring
immediate DOM removal. The duplicate-fetch test puts StrictMode at the root, as
required for React 19's effect replay, and retains both overlapping requests and
all rendered-child/store assertions. These repairs preserve the behavior under
test instead of accepting framework warnings or weakening expected outcomes.

Public metadata routes previously selected the admin session whenever an anchor
or workspace slug contained `instances`. The middleware and session inspector
now reserve admin cookies for the `/api/instances` namespace. Public-route
middleware no longer accepts an admin-cookie configuration. Regressions verify
that public requests inspect and refresh only the ordinary session, leave admin
cookies untouched on successful and slashless routes, and do not authenticate
using an admin cookie alone. Oversized public requests retain the ordinary cookie
and its expiry policy. Instance routes still use the admin cookie and expiry.
The new assertions failed against the old substring rule before the fix.

Read-only fallback hydration now runs the production HTML, Yjs, and document-format
converters. The regression asserts explicit paragraph text and a bold mark after
applying the binary update, and verifies that recovery makes no persistence call.
The existing locked, archived, and pending-fetch no-write tests remain in place.

The gate review also found two discovery exclusions: editor Vitest did not match
`.test.tsx` files, and Oxlint ignored owned Storybook configuration. Editor test
discovery now includes both TS and TSX files, and the Storybook configuration is
included in the shared lint gate.

The Celery publisher and its RabbitMQ test fixtures have been retired. Startup
requires native background tasks before publisher construction, so the old
fallback was unreachable. Native background-task tests cover the active durable
workers and admissions. The server still accepts legacy broker configuration
for compatibility, but does not connect to a broker.

The Redis upgrade now has a disposable-Valkey integration test that executes the
application's key operations and sends an admin command between two collaboration
instances. It exposed JSON admin messages being passed to Hocuspocus's binary
message decoder on the shared subscriber. Admin messages now use their own
subscriber, with explicit listener and connection cleanup; the regression fails
on unhandled decoder errors as well as failed delivery.

The Effect 4 migration is covered through the actual PDF controller and service
layer: unauthenticated and invalid requests return 401/400, and successful output
preserves page settings, PDF bytes, and safe Unicode download filenames. These
checks complement the timeout/retry tests and schema-boundary cases.

The local stack restart exposed a gap between bundler-backed tests and the built
Node application: Live imported the editor's browser entry point, which pulled
UI drag-and-drop packages into native Node. Runtime imports now use the editor's
server-safe entry point. The Redis integration suite also starts the built Live
server in native Node, checks its health response, and verifies graceful shutdown.

## Transition and transport regressions from browser review

The sidebar regression now waits for both its rows and loading skeleton to
unmount after a disclosure transition. Comparing a mounted React DOM node with
null in a failing Node assertion caused runaway diagnostic formatting; DOM
identity/absence checks use boolean assertions, and warning capture is bounded.
An external process supervisor and serialized workers contain failures that
block the test runner's own event loop. A dropdown test also now waits for its
scheduled focus restoration before tearing down jsdom.

Priority selection tests check that real options have no aria-hidden ancestor
and that selecting an option emits the correct value. The analytics test mounts
the actual DataTable, types a filter, clears it, and checks rendered rows and
React warnings. Collaboration tests use two real WebSocket clients through the
application router, assert bidirectional document edits, reject unauthorized
clients before document loading, and verify connection cleanup. These cover
observable behavior rather than merely successful imports or route registration.
The same transport behavior runs with Buffer, ArrayBuffer, and buffer-fragment
messages. This closes the uncovered fragmented-message branch without lowering
the coverage floor.
