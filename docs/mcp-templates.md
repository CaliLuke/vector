# Template contracts

Templates are reusable JSON objects for work items, pages and projects. The
native template catalogue owns storage, authorization and mutation semantics;
MCP is its generated projection. A template stores data and never executes it,
resolves references, or grants permission to create a resource from it. Resource
creation must still pass its own current validation and authorization.

A project template exists only at workspace scope. Work-item and page templates
can belong to a workspace or project. Workspace listing includes visible project
templates; a project selector restricts the list to that project's templates.
Current active workspace membership is required. Project rows additionally require
current active project membership and an unarchived project. Guests can read.
Workspace administrators write workspace templates; project administrators write
project templates. Workspace template governance defaults to project writes allowed;
when enabled by the owning governance migration it blocks project writes, including
receipt replay, without hiding existing project templates. The MCP feature update
contract does not change governance.

Create requires a nonblank name and an object, including an empty object. Update
requires a name, summary or nonempty object. Object updates merge top-level keys;
provided nested values replace that key, and JSON null remains a stored value.
The summary is separate from the templated resource body. Project summaries use
short_description; workspace work-item/page summaries use description_html.
Workspace project summaries use description. HTML summaries are sanitized; the
opaque template object is retained without evaluating its contents.

Writes and deletions record immutable before/after audit events, optional retry
receipts and revisions in one transaction. Current scope and governance are checked
before replay. Every catalogue writer advances the workspace template revision.
Deletion retains a tombstone and removes the template from listing. Parent resource
deletion cascades its templates; no copied work items, pages or projects are touched.
The retention worker purges old tombstones while retaining template audit events
and unexpired deletion receipts. Physical project deletion removes its definitions
but retains their audit events; physical workspace deletion removes workspace-owned
storage. Rollback refuses to discard records, audit events or configured workspace
governance.

The [SDK record tests](../apps/go-api/internal/mcpactions/template_contract_integration_test.go)
cover all five valid kind/scope combinations, summary/body separation, top-level
merge and null preservation, record fields, storage, audit events and deletion.
[Visibility checks](../apps/go-api/internal/mcpactions/template_visibility_integration_test.go)
cover workspace/kind/project selection, foreign IDs, revoked project membership
and archived projects. [Validation checks](../apps/go-api/internal/mcpactions/template_validation_integration_test.go)
verify rejected writes leave records, audits, revisions and retry keys unchanged.

[Role and governance checks](../apps/go-api/internal/mcpactions/template_authorization_integration_test.go)
cover new writes and receipt replay. [Authorization after waiting](../apps/go-api/internal/mcpactions/template_recovery_integration_test.go)
and [project scope changes](../apps/go-api/internal/mcpactions/template_scope_wait_integration_test.go)
exercise revocation after admission under both credentials on protocol revision `2026-07-28`.
[Competing writes](../apps/go-api/internal/mcpactions/template_concurrency_integration_test.go)
verify same-key replay, changed-input conflicts and one winner for a shared revision.
The recovery tests also verify cancellation and commit rejection preserve state and
leave the retry identity usable.

[Receipt timestamp checks](../apps/go-api/internal/mcpactions/template_receipt_integration_test.go)
verify create and update receipts retain the accepted user-timezone representation
after a preference change. Template JSON retains its original timestamp-looking
values and numeric precision. Older receipts did not retain the accepted timezone;
they preserve their stored timestamps, and the original display offset cannot be
reconstructed from those records.

[Native writer checks](../apps/go-api/internal/mcpactions/template_native_revision_integration_test.go)
cover inserts, edits, soft deletion and physical deletion in all five scopes,
cross-workspace moves and governance changes. Each invalidates stale catalogue
revisions while leaving the retry key usable after a refused write.
[Project deletion](../apps/go-api/internal/mcpactions/template_parent_lifecycle_integration_test.go)
and [workspace deletion](../apps/go-api/internal/mcpactions/template_workspace_lifecycle_integration_test.go)
exercise durable parent cascades and preserve independent definitions and resources.
[Retention checks](../apps/go-api/internal/mcpactions/template_retention_integration_test.go)
verify deletion age, audit preservation and receipt replay after tombstone removal.

The native template model is appended to the deletion registry so persisted model
and relation indexes keep their meaning. The migration repairs missing tombstones
under already deleted parents. [Migration checks](../apps/go-api/internal/bootstrap/template_parent_migration_integration_test.go)
verify preserved definitions, audits, prior tombstones, governance and unrelated
workspaces, along with revision changes, idempotence and safe rollback.
[Storage rollback checks](../apps/go-api/internal/bootstrap/template_rollback_integration_test.go)
verify data-loss refusal and empty-storage rollback/upgrade.

The workspace project summary intentionally follows the tool's documented intent;
the pinned Python helper passes short_description to a model that ignores it.
[Upstream issue #253](https://github.com/makeplane/plane-mcp-server/issues/253)
records the confirmed helper defect. The compatibility exception can be removed
when the pinned reference preserves the project summary. The SDK record tests verify the workspace project summary.

Template JSON retains numeric precision, including integers above the JavaScript safe-integer range. Timestamp projection applies to catalogue records and receipts, not keys inside template_data. The SDK record and receipt timestamp tests exercise these preservation paths.
