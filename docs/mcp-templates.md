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
Rollback refuses to discard template data.

Proof must cover cross-workspace IDs, current membership and governance on replay,
project visibility in workspace listings, all three kinds at valid scopes, merge
and null behavior, summary/body separation, CAS and atomic audit/receipt writes.
The workspace project summary intentionally follows the tool's documented intent;
the pinned Python helper passes short_description to a model that ignores it.
[Upstream issue #253](https://github.com/makeplane/plane-mcp-server/issues/253)
records the confirmed helper defect. The compatibility exception can be removed
when the pinned reference preserves the project summary. A focused native
regression remains part of consolidated verification.

Template JSON retains numeric precision, including integers above the JavaScript safe-integer range. Timestamp projection applies to catalogue records and receipts, not keys inside template_data. These preservation paths are included in the consolidated regression batch.
