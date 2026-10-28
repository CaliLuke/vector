# Feature settings and initiatives

The native feature settings application owns the workspace flags and projects'
existing module/cycle/view/page/intake/type settings. Project feature writes update
the canonical project columns, so browser and MCP reads agree. Additional project
flags and workspace flags are durable configuration, initially disabled. They do
not assert that Vector implements the unrelated teams, AI, workflows or project
updates products. Work-item types and releases report enabled because their native
catalogues exist; states_owned_by_workspace reports false because states remain
project-owned. These read-only facts cannot be changed by feature updates.

The inherited teams table contains workspace metadata, but Vector has no native
teamspace membership or project-permission model. Teamspace support is deferred.

Discovery without a selected workspace checks feature availability across current
memberships. A write action is advertised only when the actor administers a
workspace where that feature is enabled. A selected workspace constrains discovery
to its immutable ID and slug. Revoked membership removes its capabilities.

Only supplied booleans change; false is a real value. Workspace administrators
manage workspace settings. Current project administrators manage project settings;
workspace administration alone does not grant access to a private project. Reads
require current active workspace membership and, for project settings, project
membership. Changes are audited, revisioned across all writers, and support optional
receipts whose replay rechecks current scope.

Initiatives are a native workspace catalogue, gated by the initiatives flag.
Disabling the flag preserves data and associations but blocks initiative operations,
including replay, with guidance to re-enable it or use Initiative work-item types.
The catalogue starts disabled. Its ten MCP actions use one native application.
Workspace members can read; workspace administrators create, update and delete.
Lead references must be current workspace members. Names are nonblank; HTML is
sanitized, dates are ISO dates, start must not exceed end, and state defaults to
DRAFT with the five SDK states accepted. Optional fields are changed only when sent.

Initiative projects and work items use workspace-bound foreign keys. Linking or
unlinking additionally requires writable current project membership for every
selected project/item. Lists filter private or inaccessible targets before counting
and pagination. Project and item links are independent: linking an item does not
implicitly link its project. Removals apply before additions, so addition wins for
an ID in both lists. Deleting an initiative removes its links, keeps all projects
and work items, and records item association history through the durable outbox.

The catalogue and links retain audit events. All catalogue/link writers advance a
workspace initiative revision; item links also advance native item revisions.
Current workspace, project, lead and feature access are checked before receipt
replay. Multi-target changes and their audit/effects commit atomically. Rollback
must refuse to lose initiative data.

Proof must establish each field's native storage mapping, false-versus-omitted
updates, feature disable/re-enable and replay behavior, all-writer revisions,
reference ownership, private-target filtering, atomic link changes, durable history,
and the pinned SDK projections under both authentication modes and MCP revisions.
