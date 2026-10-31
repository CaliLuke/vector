# MCP work-item types

The seven `workitem_type` actions follow the pinned
[MCP handler](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/workitem_type.py)
and `plane-sdk==0.3.1` fields. Vector retains its native workspace-owned type
catalogue and project imports. Project-scoped definition changes return
`workspace_owned`. Enable types in workspace settings before changing them.

Definitions start inactive unless `is_active: true` is explicit. Empty optional
strings are omitted, so an empty description does not clear the current value.
An omitted or empty `project_ids` string keeps existing links; `"[]"` explicitly
removes them. Import lists accept comma-separated IDs or JSON arrays and
normalize duplicate IDs. Batch failures roll back definitions, links, revisions,
receipts, and effects together.

`list` returns an array. Workspace lists include the whole catalogue and ignore
pagination arguments; project lists use `cursor` and `per_page`. `retrieve`
requires a type in the selected workspace and, when supplied, project scope.
The response includes the SDK's nullable fields. Its `project_ids` includes
only active projects visible through the caller's active membership. Modeled
timestamps use the caller's timezone; arbitrary `logo_props` JSON stays intact.

Workspace administrators create, update, and delete definitions. Every project
explicitly named in `project_ids` also requires current project-admin rights.
Project administrators import active types or resume paused links. `resolve`
matches trimmed names exactly: any current project member can resolve an
already available type; importing or resuming requires project-admin rights;
creating or activating also requires workspace-admin rights. Native name
uniqueness still rejects a differently cased duplicate.

The default type stays active and cannot be deleted. Types used by published,
archived, deleted, or draft items remain protected by native preservation rules.
Unused-type deletion uses the native configuration cleanup.

Optional receipts and revisions follow [write recovery](mcp-writes.md).
The revision covers workspace type definitions, project links, and type settings,
including native writers. API changes also invalidate the native editor revision.
Current identity, workspace, project, and action authorization run before replay.
For `resolve`, authorization considers current availability before returning a
receipt; replay itself does not repeat creation, activation, or import.
Receipts retain their accepted representation when the user changes timezone.

[Contract tests](../apps/go-api/internal/mcpactions/workitem_type_contract_integration_test.go),
[scope tests](../apps/go-api/internal/mcpactions/workitem_type_scope_integration_test.go),
[receipt tests](../apps/go-api/internal/mcpactions/workitem_type_receipts_integration_test.go),
[preservation tests](../apps/go-api/internal/mcpactions/workitem_type_preservation_integration_test.go),
and [timezone tests](../apps/go-api/internal/mcpactions/workitem_type_timezone_integration_test.go)
exercise the generated server with personal tokens and OAuth on both protocol
revisions. The native
[configuration deletion test](../apps/go-api/internal/workitemtypes/type_configuration_integration_test.go)
checks cleanup without broadening property scope.

Mutation acceptance still needs domain-specific native/MCP contention,
cancellation, commit-failure, and authorization-wait checks. Shared receipt-engine
tests establish the common transaction behavior but do not replace these checks.
