# MCP cycles

The `cycle` tool implements all eleven actions from the pinned Plane MCP handler
and `plane-sdk==0.3.1`: list, retrieve, create, update, delete, list_workitems,
manage_workitems, transfer_workitems, complete, archive and unarchive.
Implementation does not close the remaining mutation acceptance requirements.

Both actions require `project_id`. Retrieve also requires `cycle_id`. Workspace selection follows the shared PAT/OAuth contract. Discovery advertises cycle reads only when the principal has an active membership in a live project in the selected workspace, or in any eligible workspace for context-free OAuth. Invocation requires access to the specific project.

List defaults to active cycles. `archived: true` selects archived cycles and ignores `status`, including otherwise invalid values. Active status values are `current`, `upcoming`, `completed`, `draft`, and `incomplete`. Active lists always return a page, including `status=current`. Zero or omitted `per_page` uses 1000; the accepted range otherwise is 1–1000. Empty cursors and ordering strings use server defaults. The existing API cursor format is retained. Active ordering supports created/updated timestamps, name, start/end dates, and sort order; archived lists preserve the API's fixed ordering.

Active list entries contain the 22 SDK `CycleLite` fields and omit metrics. Detail and archived entries contain the 31 SDK `Cycle` fields, including nullable issue counts and estimates. Missing detail estimates are null. Integral API estimates such as `3.0` become integer `3` without a floating-point conversion. Retrieve does not return archived cycles. An authorized empty collection is an empty page; lost authorization is an error.

Typed generated records and pages own the wire contract. A concrete entry carrier avoids the nested untagged-union limitation tracked in [Loom #1036](https://github.com/CaliLuke/loom/issues/1036). The projection enforces the action-specific fields and non-null lite ownership fields. Replace that carrier with the corresponding nested union when a released generator supports the composition and the empty-page/field-set tests pass. No dependency source is patched.

`internal/cyclereads` retains the existing HTTP projections. MCP binds an immutable workspace ID and checks the active account, live workspace/project, current workspace role, and active project membership in the primary query that returns data. Metric relationships must belong to that project and workspace. The HTTP design now accepts the canonical `status` parameter; full lists retain `cycle_view` as a fallback and their existing bare-array shape for current cycles. Lite lists do not use that alias.

HTTP and MCP share timezone conversion. Modeled timestamps use the user's current timezone and preserve microsecond precision; arbitrary JSON such as `progress_snapshot` is unchanged.

Generated-server PostgreSQL tests cover both credentials and protocol revisions,
exact SDK field sets, filters and pagination, archived estimates, hidden/absent
targets, empty pages, late authorization changes, cross-workspace relationships,
and timezone behavior. Remaining browser and mutation acceptance is tracked in
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
Further isolation and contention checks are described below; browser acceptance
remains open.

The next locally applied acceptance batch adds
[resource and receipt scope checks](../apps/go-api/internal/mcpactions/planning_scope_integration_test.go),
[foreign work-item preservation](../apps/go-api/internal/mcpactions/planning_association_scope_integration_test.go),
[competing requests and revisions](../apps/go-api/internal/mcpactions/planning_concurrency_integration_test.go),
and [cancellation with safe retry](../apps/go-api/internal/mcpactions/planning_cancellation_integration_test.go).
The combined cycle/module run passes 24 new cases against the applied source.
Complete publication gates remain pending.
