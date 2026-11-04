# Pages and collections

Pages retain the native pages table, page lifecycle and body history. Project
pages use project_pages and the existing lifecycle. Workspace pages are global
pages without a live project association. Collection membership never turns a
project page into a workspace page. MCP selects scope explicitly with project_id.

Current workspace membership is always required. Project reads additionally
require current project membership and native guest/owner visibility. Project
writes preserve native owner/member/admin rules. Workspace public pages are
readable by current workspace members; private pages remain owner-only. Workspace ancestor
visibility also applies. A private collection requires explicit membership or
ownership, even for workspace administrators. Filing a private page does not make
it public. Collection access is fixed at creation.

Collection owners have edit access and cannot remove or downgrade their own
membership. Explicit view/comment/edit membership overrides public defaults;
public collections otherwise give workspace members edit access and guests view
access. Only the owner manages membership or deletes a collection. Editors may
rename, reorder and file pages. A page writer must also have edit access to its
collection. Moving pages requires edit access in both collections and write access
to the page; changing membership never grants access to unrelated private pages.

A page's parent is chosen at creation in the same scope. Children inherit their
parent's collection. Filing, moving or unfiling a root moves its whole subtree;
a nested page cannot be filed independently of its parent. List-pages defaults
to roots and can select a parent. Search returns visible, editable workspace roots
that can be filed into the destination. Collection deletion retains pages and
unfiles them, archiving them by default. Private page access survives unfiling.

Updates replace supplied name/body fields and reject locked or archived pages.
Each MCP name or body replacement advances the page's collaboration revision and
clears its cached binary document. The editor uses that revision in its live and
offline document identity. Native title/body saves must match the revision they
opened; stale sessions receive HTTP 409 and must reload. Receipt replays do not
advance the revision again. Page IDs and links remain stable.
Creation sanitizes HTML and retains native owner and access defaults. Archive and
restore apply to descendants; native restoration can detach a root whose parent
is still archived. Deletion requires prior archive, retains body history and uses
native durable cleanup. Item-page links require current item visibility and page
visibility; adding/removing links requires project member/admin access. A link
never grants page visibility. Link IDs address detachment; page IDs address attach.

Read filtering, counts and projections share a repeatable-read snapshot. Optional
receipts check current action rights before replay. A workspace page revision
covers pages, project links, collections, collection memberships and item links
from every writer. Private resource deletion keeps authorization tombstones until
receipts expire through native retention. Page transaction/version work is enqueued
with mutations and admitted by the native effect worker after commit. Catalogue
and membership changes retain immutable audit snapshots.

Consolidated proof must cover private ancestor/collection visibility, guest scope,
owner protection, whole-subtree moves, archive/restore, atomic name/body updates,
current permissions on replay, every-writer revisions, durable body history and
asset behavior, both authentication modes and protocol revision `2026-07-28`. Workspace page
and collection data must prevent a lossy rollback to the older project-only model.

The shared generated-server content batch passes all 23 actions under both
authentication modes on protocol revision `2026-07-28`. It exercises workspace page CRUD,
receipt replay, subtree filing/removal, collection candidate search, membership
changes, item linking, archive/restore and transactional body/version admission.
The acceptance evidence below covers the remaining privacy, recovery, revision,
worker and asset boundaries.

## Acceptance evidence

The pinned page and collection handlers at `dc7cfd9f3c305891909e2372f0cc507291353801`
and SDK 0.3.1 page, collection, membership and branch models were inspected.
The handler exposes name/body replacement, tri-state archive, numeric page access,
string collection/member access and whole-branch filing. Broader SDK editor fields
are not additional MCP inputs.

The [content batch](../apps/go-api/internal/mcpactions/content_batch_integration_test.go)
covers every generated action with PAT and OAuth on revision `2026-07-28`.
[Privacy and hierarchy](../apps/go-api/internal/mcpactions/page_privacy_integration_test.go)
cover private ancestors, owner protections, collection access, atomic subtree moves
and archive/restore. [Scope checks](../apps/go-api/internal/mcpactions/page_scope_integration_test.go)
cover immutable workspace binding, project pages and restricted guests.
[Recovery](../apps/go-api/internal/mcpactions/page_recovery_integration_test.go)
covers all 15 mutation paths through cancellation, commit rejection, retry and
replay, queued credential revocation, request conflicts and competing revisions.
[Reference waits](../apps/go-api/internal/mcpactions/page_reference_wait_integration_test.go)
and the [membership-row race](../apps/go-api/internal/mcpactions/collection_membership_wait_integration_test.go)
verify fresh permissions after waiting. [Native writers](../apps/go-api/internal/mcpactions/page_native_revision_integration_test.go)
cover insert/update/delete at each revision boundary.

[Worker delivery](../apps/go-api/internal/mcpactions/page_worker_integration_test.go)
checks body versions, copied-image component ownership and duplicate admission;
removing an image log is scoped to its page and workspace. Existing
[asset visibility](../apps/go-api/internal/assets/asset_parent_visibility_integration_test.go)
and [collaboration replacement](../apps/go-api/internal/mcpactions/page_collaboration_integration_test.go)
checks retain ownership and stale-editor protection. [Rollback checks](../apps/go-api/internal/bootstrap/page_rollback_integration_test.go)
preserve workspace pages, collections, project-item links and retained audit,
while empty storage can roll back and upgrade.
