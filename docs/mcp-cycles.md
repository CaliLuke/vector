# MCP cycles

The `cycle` tool implements all eleven actions from the pinned Plane MCP handler
and `plane-sdk==0.3.1`: list, retrieve, create, update, delete, list_workitems,
manage_workitems, transfer_workitems, complete, archive and unarchive.
Implementation does not close the remaining mutation acceptance requirements.

All actions require `project_id`. Create also requires `name` and `owned_by`; actions on an existing cycle require `cycle_id`. Transfer additionally requires `new_cycle_id`. Workspace selection follows the shared PAT/OAuth contract. Discovery advertises cycle reads only when the principal has an active membership in a live project in the selected workspace, or in any eligible workspace for context-free OAuth. Invocation requires access to the specific project.

List defaults to active cycles. `archived: true` selects archived cycles and ignores `status`, including otherwise invalid values. Active status values are `current`, `upcoming`, `completed`, `draft`, and `incomplete`. Active lists always return a page, including `status=current`. Zero or omitted `per_page` uses 1000; the accepted range otherwise is 1–1000. Empty cursors and ordering strings use server defaults. The existing API cursor format is retained. Active ordering supports created/updated timestamps, name, start/end dates, and sort order; archived lists preserve the API's fixed ordering.

Active list entries contain the 22 SDK `CycleLite` fields and omit metrics. Detail and archived entries contain the 31 SDK `Cycle` fields, including nullable issue counts and estimates. Missing detail estimates are null. Integral API estimates such as `3.0` become integer `3` without a floating-point conversion. Retrieve does not return archived cycles. An authorized empty collection is an empty page; lost authorization is an error.

Typed generated records and pages own the wire contract. A concrete entry carrier avoids the nested untagged-union limitation tracked in [Loom #1036](https://github.com/CaliLuke/loom/issues/1036). The projection enforces the action-specific fields and non-null lite ownership fields. Replace that carrier with the corresponding nested union when a released generator supports the composition and the empty-page/field-set tests pass. No dependency source is patched.

`internal/cyclereads` retains the existing HTTP projections. MCP binds an immutable workspace ID and checks the active account, live workspace/project, current workspace role, and active project membership in the primary query that returns data. Metric relationships must belong to that project and workspace. The HTTP design now accepts the canonical `status` parameter; full lists retain `cycle_view` as a fallback and their existing bare-array shape for current cycles. Lite lists do not use that alias.

HTTP and MCP share timezone conversion. Modeled timestamps use the user's current timezone and preserve microsecond precision; arbitrary JSON such as `progress_snapshot` is unchanged.

Generated-server PostgreSQL tests cover both credentials and protocol revisions,
exact SDK field sets, filters and pagination, archived estimates, hidden/absent
targets, empty pages, late authorization changes, cross-workspace relationships,
and timezone behavior. Acceptance status is tracked in
[MCP parity](mcp-parity.md) and the [debt catalog](tech-debt-catalog.md).
Production OAuth issuer selection remains an accepted deferral.

## Mutation acceptance

Cycle mutations use the native lifecycle and planning writers within the shared
project transaction. Current active project membership with member or administrator
role is required. Archived or deleted projects reject writes and receipt replay.
Optional receipts and expected revisions follow the [write contract](mcp-writes.md).
Database triggers advance cycle revisions for row changes and issue associations,
including writes outside MCP.

The [core workflow batch](../apps/go-api/internal/mcpactions/core_batch_integration_test.go)
covers create replay, update, association, transfer, completion, archive, restore
and deletion under both credentials and protocols. Deleting planning resources
preserves their work items. Those workflows passed in the full gate at `1be8fcd5`.
The [planning receipt regressions](../apps/go-api/internal/mcpactions/planning_receipt_integration_test.go)
add changed-input conflicts, direct-writer revision invalidation, preservation of
later edits on replay, and atomic association/effect rollback with retry.
[Relation-writer cases](../apps/go-api/internal/mcpactions/planning_revision_integration_test.go)
cover insert, move and removal of work-item links and module memberships.
[Lifecycle cases](../apps/go-api/internal/mcpactions/planning_lifecycle_receipt_integration_test.go)
prove replay preserves later restoration and deletion.
[Delete authorization](../apps/go-api/internal/mcpactions/planning_delete_authorization_integration_test.go)
checks current owner/admin authority before replay, including a removed target.
Their combined cycle/module run passes 52 focused MCP cases.
[Role and lifecycle cases](../apps/go-api/internal/mcpactions/planning_permissions_integration_test.go)
add 72 cases for denied scope, allowed member/admin writes and owned-delete replay.
[Lock-wait cases](../apps/go-api/internal/mcpactions/planning_authorization_wait_integration_test.go)
add 40 cases that revoke credentials, OAuth write scope or project membership after
the request waits on a project lock, before a new write or receipt replay.
These transport cases cover both credentials and protocol revisions. Both full gates passed at `97660842`, including all 13,814 Go test events.
Further isolation and contention checks and completed browser acceptance are
described below.

The acceptance batch published at `a2e44d6f` adds
[resource and receipt scope checks](../apps/go-api/internal/mcpactions/planning_scope_integration_test.go),
[foreign work-item preservation](../apps/go-api/internal/mcpactions/planning_association_scope_integration_test.go),
[competing requests and revisions](../apps/go-api/internal/mcpactions/planning_concurrency_integration_test.go),
and [cancellation with safe retry](../apps/go-api/internal/mcpactions/planning_cancellation_integration_test.go).
The combined cycle/module run passes 24 new cases against the applied source.
Both full gates passed at `a2e44d6f`, including all 13,894 Go test events.

The fixes published at `a2e44d6f` cover calendar and transfer snapshot boundaries.
[Calendar regressions](../apps/go-api/internal/cyclereads/mutation_calendar_integration_test.go)
verify that completion uses the current actor's calendar day, archive ends cycles
whose end timestamp is still in the future (including later today), and a later
receipt replay preserves the saved end date after the actor's timezone changes.
An already-completed cycle retains its end date during archive.
The date-only running check in the pinned handler is reported in
[upstream #266](https://github.com/makeplane/plane-mcp-server/issues/266); retain
instant comparison when reconciling future handler revisions until that source
and the local calendar regressions agree.
[Transfer scope regressions](../apps/go-api/internal/mcpactions/planning_transfer_scope_integration_test.go)
verify stored and returned snapshot totals, label/assignee distributions and
completion charts, including estimate distributions, exclude malformed foreign
associations. Three unfinished local items move while completed/cancelled and
foreign associations remain in place; receipt replay leaves state and effects
unchanged. These fixes are published at `a2e44d6f`. The combined run passes 34 focused cases:
seven calendar, seven existing native date, four transfer-scope and sixteen
existing core workflow cases. The fast and full publication gates pass at `a2e44d6f`.

## Planning work-item query acceptance

The shared cycle/module `list_workitems` path returns the seven-field toolkit
page: `results`, `count`, `total_count`, `next_cursor`, `prev_cursor`,
`next_page_results` and `prev_page_results`. PQL applies before counting and
pagination; selected `fields` apply to each returned work item.

[Generated-server query cases](../apps/go-api/internal/mcpactions/planning_query_integration_test.go)
exercise scalar and actor-relative relationship predicates together, two-page
traversal, sparse projection, matching grouped counts and unchanged stored state.
The fixture separates matching rows from unlinked, archived, draft, deleted,
foreign-item and incorrectly scoped association rows. Invalid predicates return
the failed query and full PQL reference without partial results; a corrected
request succeeds.

[Scope cases](../apps/go-api/internal/mcpactions/planning_query_scope_integration_test.go)
reject parents from another accessible project or workspace, reject a mismatched
workspace/project pair, and distinguish an authorized empty page from revoked
project membership. Access to the other authorized workspace remains available.
All three behaviors cover both credentials and both supported protocol revisions,
with explicit per-call workspace selection in the query and scope cases.

[Custom-property cases](../apps/go-api/internal/mcpactions/planning_property_query_integration_test.go)
also exercise the separate property-discovery snapshot through the generated
server. Numeric zero matches before pagination, the unlinked item is excluded,
counts agree with the parent predicate, and sparse projection and stored state
remain intact. Both credentials and protocols are covered.

The applied-source run passes 32 planning leaf cases (52 test events), with no
failed or skipped cases. The Go fast gate and duplication ratchet pass with no added allowances.
Publication remains required. This
closes no broader PQL grammar or unrelated action acceptance requirement.

The [PQL reference transport cases](../apps/go-api/internal/mcpactions/pql_reference_integration_test.go)
verify exact default/full/brief responses and invalid-detail rejection without a
workspace selector, for both credentials and protocols. Four additional applied-source
cases pass, giving 36 leaf cases and 57 test events in the combined run. These supplement the existing language semantics tests; they do not
change the accepted teamspace deferral or unsupported property mappings.

## Native browser acceptance

Local checks on 2026-10-30 used `http://localhost:8000/`, the API image
`sha256:5e8a6fdad64a33ba93297c24595a2d8adab7556730705da611f89a2355ff4b0c`,
and the application source at `f47720fd` (production code unchanged from
`a2e44d6f`). Dedicated cycle and module fixtures were created through MCP and
appeared in the native lists. Native rename forms advanced each revision from 1
to 2. MCP rejected updates with revision 1 without changing either record. Updates
with revision 2 advanced to 3, and the new names appeared after browser reload.
Replaying each update preserved the saved payload and revision; only the
`replayed` indicator changed.

Archive, unarchive and delete each passed receipt replay and active/archived-list
checks. The native lists hid archived fixtures, showed restored fixtures, and
returned to empty after deletion. Archiving the running cycle completed it; the
restored cycle appeared in the completed group. A planned module rejected archive
without changing its status or archive timestamp; completing it allowed archive.
Cycle detail rejected an archived cycle, and both detail endpoints rejected the
deleted fixtures. Both final list responses exactly matched their saved empty
baselines. All original project feature settings were restored and the dedicated
project was retained.

Local evidence is in `/tmp/vector-planning-browser-20261030/`: redacted MCP
requests/responses, `edit-proof.json`, `lifecycle-proof.json`, `cleanup-proof.json`,
and native screenshots. The browser check uses a personal token and the current
protocol; the generated-server tests provide both credential/protocol matrices.
This closes the recorded browser gap for these planning actions. Publication of
the 36 query/reference cases remains pending the full gate.
