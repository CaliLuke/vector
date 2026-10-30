# MCP projects

The `project` tool provides `list`, `retrieve`, `archive`, and `unarchive`. Its action and result contracts come from the pinned Plane MCP handler and `plane-sdk==0.3.1`. Other project mutations and feature settings remain outside this slice. See [write recovery](mcp-writes.md) for archive permissions, receipt and revision extensions, discovery filtering, and remaining evidence.

## Inputs and results

`list` accepts `cursor`, `order_by`, and `per_page`. Omitted or zero `per_page`
uses 100; the accepted range is 1–1000. An empty cursor selects the first page.
The existing API cursor parser and ordering rules remain authoritative. Lists
include archived projects, matching the pinned handler's `exclude_archived=False`.
They return the SDK's lightweight project fields and complete pagination envelope.

`retrieve` requires a project UUID and returns the SDK's full project model.
The shared HTTP reader omits the member-specific `sort_order`; the MCP projection
supplies its SDK default, null. Existing JSON-valued icon, logo and description
fields retain their values. The enclosing project and page records are typed;
there is no opaque whole-result map.

All four actions accept the optional MCP `workspace_slug` selector. Personal tokens
and OAuth can select a workspace through that argument or an optional fixed
workspace header default. A scoped action without a workspace fails with
`invalid_params`; conflicting selectors fail at the existing HTTP boundary.
Other-action fields and unimplemented actions fail generated validation before
application dispatch.

## Authorization and data ownership

The generated tool calls `internal/mcpactions`, which revalidates the credential,
current OAuth read scope, workspace selection and timezone. The application
reader is `internal/workspaceprojectreads`, wired to the primary database.

The data query checks the active actor, live workspace and current active,
undeleted membership with role 5, 15 or 20 in the same PostgreSQL statement as
the returned projects. Earlier admission cannot authorize a later read after
revocation. MCP also passes the verified immutable workspace ID. Reassigning its
slug to a different workspace cannot change the scope of an admitted operation,
even if the actor also belongs to that other workspace.

Workspace members can see public projects. A private project additionally
requires current project membership. A workspace administrator does not bypass
private-project membership. Collection counts and pages apply visibility before
pagination. A hidden, missing or other-workspace detail returns the same safe
`not_found` error. Database failures remain failures, not empty successful pages.

These checks also repair stale workspace admission in the existing lightweight,
full-list and detail HTTP readers. HTTP callers retain their wire projections and
can omit the optional expected-workspace binding. The full collection's supported
`network` ordering now selects its sort column, matching the existing lightweight
collection's behavior.

## Verification

Generated-server tests cover both supported protocol revisions with personal
tokens and OAuth, using PostgreSQL and a TLS issuer. They verify exact SDK detail
projection, paging, archive inclusion, private and foreign-project isolation,
fresh primary data, selectors, narrowed scopes, and revocation after HTTP
authentication. Repository regressions change workspace, account and membership
state between initial admission and the data statement, including slug
reassignment and private-project membership removal.

The same immutable-workspace binding is used by the four existing scoped member
reads. Project-member reads additionally recheck current workspace membership
in the roster statement. Their regressions mutate scope after action admission
and before the domain reader executes, so another credential check alone cannot
make these tests pass.

All four project actions are integrated and tested through the generated server with both credentials and protocol revisions. The full Go gate and focused race tests pass; package coverage floors remain unchanged. Frontend builds and checks pass after the parity-table formatting correction. Read-action discovery now reflects current workspace and project membership. Browser conflict proof remains open.
Complete OAuth browser/proxy proof and 208-action parity remain separate open work.
