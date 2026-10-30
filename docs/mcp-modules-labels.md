# MCP module and label reads

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
The remaining catalogue actions have generated-server workflow coverage; final
combined quality-gate review remains open.

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
the complete publication gates are still required for the new regression batch.
