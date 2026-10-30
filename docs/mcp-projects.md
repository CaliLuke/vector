# MCP projects

This page documents the `project` tool’s `list`, `retrieve`, `create`, `update`, `delete`, `archive`, and `unarchive` actions. Its action and result contracts come from the pinned Plane MCP handler and `plane-sdk==0.3.1`. Feature settings and work-log summaries have separate contracts in the parity tracker. See [write recovery](mcp-writes.md) for archive permissions, receipt and revision extensions, discovery filtering, and remaining evidence.

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

All seven actions accept the optional MCP `workspace_slug` selector. Personal tokens
and OAuth can select a workspace through that argument or an optional fixed
workspace header default. A scoped action without a workspace fails with
`invalid_params`; conflicting selectors fail at the existing HTTP boundary.
Fields from another action and unknown actions fail generated validation before
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

Project lifecycle mutations share the native repository and receipt transaction.
[Receipt safety](../apps/go-api/internal/mcpactions/catalogue_safety_integration_test.go)
covers changed requests, native-writer revisions and current replay permissions.
[Storage rollback](../apps/go-api/internal/mcpactions/project_mutation_rollback_integration_test.go)
covers receipt and effect failures without retaining partial project defaults.
[Contention and cancellation](../apps/go-api/internal/mcpactions/project_record_concurrency_integration_test.go)
prove one winning write, stable same-key recovery, competing revisions, unchanged
state after cancellation and request-key reuse. The
[real-worker workflow](../apps/go-api/internal/mcpactions/project_delete_workflow_integration_test.go)
proves delete recovery after child cascades, including integer project identifiers.

Native browser acceptance through `localhost:8000` verifies the MCP-created
project's name, identifier and defaults; a browser rename advances the MCP
revision and rejects an old-revision write. MCP update replay returns the original
result and appears after browser refresh. Archive removes the project from the
active list and shows it in Archives. Native restoration advances the revision;
a stale archive request fails. MCP archive/restore receipts replay without
repeating the transition. Deletion hides the project from native and MCP lists
and detail reads. After the running worker completes the cascade, the original
delete receipt remains recoverable and there is one root deletion workflow.

The native restore dialog uses the project supplied by the archived card. It
must not fetch active-project detail to display its confirmation; archived
projects are absent from that endpoint. The
[dialog regression](../tests/frontend/project-archive-dialog.test.mjs) verifies
restoration without that read.

Production OAuth issuer selection remains deferred. Overall 208-action parity
remains in the outstanding debt catalog.

## Credential changes during archive lock waits

Archive and restore repeat credential
validation after domain authorization has acquired its locks and before receipt
lookup or mutation. Native HTTP callers retain their existing permissions and
have no MCP credential callback.

The [generated-server regression](../apps/go-api/internal/mcpactions/project_archive_wait_integration_test.go)
holds the project lock, waits for the request to block, changes the credential,
and releases the lock. Its 24 cases cover PAT revocation, OAuth revocation and
OAuth write-scope removal, both protocols, archive/restore, and writes/replays.
Each returns a safe error without structured content and preserves the exact
project, favorite, revision and receipt snapshot. The
[update/delete lock-wait tests](../apps/go-api/internal/mcpactions/project_record_authorization_wait_integration_test.go)
apply the same requirement to project record writes and receipt replay,
including explicit project membership revocation. These checks do not promise
continuous issuer validation through commit.
