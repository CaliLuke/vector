# Workspace releases

Workspace releases add a shared delivery catalogue, version tags, labels,
changelogs and explicit item associations. The existing project release picker
continues to use its project-owned records. Both scopes use work_item_releases;
workspace records have no project_id. MCP release actions expose workspace-owned
records, so a project-only release never becomes workspace-visible implicitly.

Current known workspace members can read the catalogue. Workspace administrators
manage releases, tags and labels. Managing item associations additionally requires
current member or administrator access to each item's live, unarchived project.
Reads of associated items filter current item visibility before counting and
pagination. Workspace release deletion can detach its associations without
revealing private item data to the caller. Private project catalogue records are
never accepted as workspace release targets.

Names and tag versions are unique within their live workspace catalogue. Releases
default to unreleased and not prerelease, with empty sanitized description and an
empty changelog created atomically. Release and target dates are independent
YYYY-MM-DD values. The latest release is the most recently dated released,
non-prerelease entry, with creation time and ID breaking ties. A lead must be a
current active workspace member; a tag must be live in this workspace. Retired
tags are omitted from current projections. Deleting a referenced tag clears the
release pointer atomically; deleting a label removes its bindings, not releases.

Changelogs accept sanitized HTML or plain text. Plain text produces matching HTML
and ProseMirror paragraphs. HTML is authoritative when both forms are supplied;
its JSON field is empty rather than retaining an unrelated old document. HTML
does not grant access to referenced assets or change their native ownership.

Item association batches are atomic, validate all item IDs, add before removing,
and make removal win for overlapping IDs. Duplicate additions and absent removals
are no-ops. Native item activity captures the release name at mutation time.
Workspace catalogue changes keep an immutable audit snapshot in the transaction.
Soft deletion retains those snapshots and item history. Release deletion rejects
remaining custom-property references, matching native preservation policy.

Workspace releases are eligible release-picker values in every project in that
workspace. Native validation, choice discovery, filtering and preservation use the
same rows and omit retired records. Existing project release settings and their
scope remain unchanged.

The native release application owns authorization, validation, storage and effects.
Generated MCP contracts project the pinned SDK record and full pagination shapes;
MCP remains disabled unless configured. Optional receipts reauthorize current
scope before replay. A workspace revision covers every release, tag, label,
changelog and association writer. Coarse revisions can conflict after an unrelated
catalogue edit; no current write can silently bypass the revision.

Consolidated proof must establish tenant isolation, guest item visibility,
cross-project batch atomicity, current rights on retry, all-writer CAS, durable
activity, catalogue audit retention, native picker compatibility, HTML/JSON
semantics, both authentication modes and both MCP protocol revisions. Rollback
must refuse workspace release data that cannot fit the old project-only schema.

The shared generated-server content batch passes all 20 actions under both
authentication modes and protocol revisions. It exercises release/tag/label CRUD,
receipt replay, latest-release selection, escaped changelog text, label assignment,
item linking and remove-wins behavior, and preservation of linked work items.
The broader permission, concurrency, worker-delivery and picker cases remain open.
