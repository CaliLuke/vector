# Work-item relations

The native relation application owns six dependency directions and workspace-defined
custom relations. Dependencies keep the existing issue_relations storage. Custom
relations have their own definitions and edges; deleting one kind never deletes the
other kind or legacy duplicate/relates_to links.

Current known workspace members can read definitions. Item reads require visibility
of the source and each returned target. Workspace administrators manage definitions.
Edge writes require current member or administrator access to every endpoint's
project; all targets are validated before any edge changes. Archived projects,
drafts, foreign workspace items and self-links are rejected. Direction is selected
by the dependency type or an exact outward/inward definition label. Equal custom
labels mean a symmetric relation. A repeated create has no additional effect.

Definition names are unique within a workspace, ignoring case. Omitted labels default
to the name; new definitions are active and non-default. Inactive definitions cannot
receive new edges and are excluded from edge reads. Deletion retains definitions
and edges for history while removing them from reads. Deleting an edge removes all
matching edges of the selected kind between the two endpoints, in either direction.

Mutations support optional retry receipts and a workspace relation revision. The
revision covers dependency rows, definitions and custom edges from every writer.
include_revision on either list returns that token. Authorization precedes receipt
replay, and mutation transactions retain current memberships and endpoint locks.
Edge changes enqueue immutable per-endpoint activity snapshots in the same transaction.

This contract and implementation await the consolidated integration pass. No new
browser editor or independent HTTP route is introduced in this MCP batch.

The generated custom-label dictionary temporarily uses a JSON carrier for [Loom #1041](https://github.com/CaliLuke/loom/issues/1041); the receipt result uses a JSON carrier for the nested-union limitation in [Loom #1036](https://github.com/CaliLuke/loom/issues/1036). Restore the typed map and nested union when released generator support is available and the combined wire-contract checks pass. Domain projections retain the pinned record shapes.
