# MCP module and label reads

The `module` and `label` tools each implement `list` and `retrieve`. Both require
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
cover denied versus empty queries and parent ownership directly. OAuth remains generic, with production issuer selection deferred. Complete local browser
verification and the remaining catalogue actions are still open.
