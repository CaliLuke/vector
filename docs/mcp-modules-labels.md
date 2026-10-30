# MCP modules and labels

The `module` and `label` read actions are `list` and `retrieve`. Both require
`project_id`; retrieve also requires `module_id` or `label_id`. An optional
`workspace_slug` selects an authorized workspace when the credential has no bound
selector. Personal tokens and OAuth use the same generated action contracts.

The pinned handler and SDK versions are recorded in [the parity tracker](mcp-parity.md).
Both lists accept `cursor` and `per_page`; zero or omitted page size uses the API's
1000-entry default, and 1000 is the maximum. Empty cursors use the first page.
Module lists also accept `archived` and `order_by`. Active module lists support the
existing seven ordering fields; archived lists retain the API's fixed ordering.
Retrieve returns active modules only.

Module pages retain all twelve SDK pagination fields. Active entries contain the
23 `ModuleLite` fields. Detail and archived entries contain 29 fields: the SDK's
28 `Module` fields plus the API's `members` array, retained by the SDK's extra-field
policy. Six issue metrics are nullable; active lists omit them. Existing metric
semantics are preserved, including distinct issue-bridge counts and distinct state
groups. The concrete record carrier follows the documented
[Loom #1036 workaround](mcp-cycles.md); remove it when released nested-union support
passes the field-set and empty-page tests.

Label entries contain the SDK's fifteen fields. Label lists use the pinned
handler's seven-field toolkit envelope: `results`, `total_count`, `count`,
`next_cursor`, `prev_cursor`, `next_page_results`, and `prev_page_results`.
Label reads exclude archived projects, matching the shared API policy. Module
reads retain the API's different policy and allow archived projects.

Both repositories use the primary database. A shared scope CTE checks the live
account, immutable workspace ID and slug, current workspace role, live project,
and active, undeleted project membership in the query that returns resource data.
Denied scope is distinct from an authorized empty page. Module membership and
metric relationships must belong to the selected project and workspace. A label
parent from another tenant or project is hidden; an owned deleted parent keeps its
existing identifier. Existing HTTP calls retain their prior policy through the
optional workspace binding. HTTP and MCP share timestamp conversion.

[PostgreSQL transport tests](../apps/go-api/internal/mcpactions/resource_read_integration_test.go)
cover both credentials and protocol revisions, exact response fields, pagination,
archive policies, scope changes after admission, every query projection, empty
pages, and foreign relationships. [Discovery tests](../apps/go-api/internal/mcpactions/discovery_integration_test.go)
verify current project-membership filtering. [Label repository regressions](../apps/go-api/internal/projectlabelreads/scoped_read_integration_test.go)
cover denied versus empty queries and parent ownership directly. OAuth remains generic, with production issuer selection deferred. Local browser checks verified module creation and an MCP update in the native UI.
Module mutations have generated-server workflow coverage; their combined domain
acceptance remains open. Label mutation acceptance is recorded below.

## Label mutation acceptance

Label mutations share the native API v1 writer and its ProjectMemberPermission:
active project guests, members and administrators can write. The app editor's
stricter role threshold does not change this API contract. MCP additionally binds
current workspace/project membership, recognized roles and immutable workspace
identity. Archived or deleted projects and inactive/deleted project memberships
reject fresh writes and receipt replay.

Native creation appends at the highest live project sort order plus 10000, or
65535 for the first label. A supplied creation sort order does not override that
native ordering rule. Updates apply supplied positions, including zero.

The [receipt regressions](../apps/go-api/internal/mcpactions/label_receipt_integration_test.go)
cover normalized input, conflicts, native-writer revisions, preservation of native
edits on replay, and atomic cascade deletion with revisions/effects/receipts.
The [permission cases](../apps/go-api/internal/mcpactions/label_permissions_integration_test.go)
verify the role and lifecycle policy without mutation effects on rejection.
The [lock-wait cases](../apps/go-api/internal/mcpactions/label_authorization_wait_integration_test.go)
hold the project lock, observe the blocked request, commit credential/scope or
project-membership revocation, then release it. Fresh updates and receipt replay
reject the changed authorization and leave owned state unchanged.
The [scope cases](../apps/go-api/internal/mcpactions/label_scope_integration_test.go)
cover workspace, project, resource and parent binding, including receipt replay.
The [concurrency cases](../apps/go-api/internal/mcpactions/label_concurrency_integration_test.go)
prove one commit for identical requests, conflicts for changed inputs sharing a
request ID, and one successful update for competing revisions.
The combined run passes 75 label cases and four related feature cases, with no
failures or skips. Transport cases cover both credentials and protocols;
concurrency cases exercise the repository directly.

Local browser/proxy checks on 2026-10-30 used the rebuilt application at
`87a791d9`, a personal token and protocol `2026-07-28`. The initial MCP/native
label pages matched. An MCP-created label appeared in settings after reload; a
native browser rename advanced the MCP revision and retained its description.
A stale-revision write was rejected without changing the record. A current-revision
MCP update stored position zero and displayed its new name/color in the browser
after reload. Deletion/replay succeeded, detail rejected the deleted record, and
both lists exactly matched their original empty baselines. The browser displayed
its empty state afterward. These checks close the recorded label read browser gap;
both full publication gates passed at `1be8fcd5`, including 13,567 Go test events
with no failures or skips. All five label actions meet the recorded acceptance
requirements. Other catalogue domains remain open.

## Module mutation acceptance

Module mutations use the native lifecycle and planning writers within the shared
project transaction. Current active project membership with member or administrator
role is required. Archived or deleted projects reject writes and receipt replay.
Row, issue-association and module-member triggers advance the module revision for
all writers. Optional receipt and revision extensions follow the
[write contract](mcp-writes.md).

The [core workflow batch](../apps/go-api/internal/mcpactions/core_batch_integration_test.go)
covers create replay, update, association, archive, restore and deletion under
both credentials and protocols. It passed in the full gate at `1be8fcd5`.
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
These transport cases cover both credentials and protocol revisions.
Eight [native module-delete authorization cases](../apps/go-api/internal/mcpactions/module_native_delete_authorization_integration_test.go)
also preserve the distinct app and API rules for creators, members and project or
workspace administrators. Both full gates passed at `97660842`, including all 13,814 Go test events.
Further isolation and contention checks are described below. The
[native browser acceptance](mcp-cycles.md#native-browser-acceptance) is complete.

The acceptance batch published at `a2e44d6f` adds
[resource and receipt scope checks](../apps/go-api/internal/mcpactions/planning_scope_integration_test.go),
[foreign work-item preservation](../apps/go-api/internal/mcpactions/planning_association_scope_integration_test.go),
[competing requests and revisions](../apps/go-api/internal/mcpactions/planning_concurrency_integration_test.go),
and [cancellation with safe retry](../apps/go-api/internal/mcpactions/planning_cancellation_integration_test.go).
The combined cycle/module run passes 24 new cases against the applied source.
Both full gates passed at `a2e44d6f`, including all 13,894 Go test events.

The [planning work-item query acceptance](mcp-cycles.md#planning-work-item-query-acceptance)
records the shared cycle/module query and reference checks. Native browser
acceptance is complete. The batch is published at `81009902`, with both full gates
and all 13,951 Go test events passing without failures or skipped tests. All nine
module actions meet their recorded acceptance requirements.
