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
semantics, both authentication modes and MCP revision `2026-07-28`. Rollback
must refuse workspace release data that cannot fit the old project-only schema.

The shared generated-server content batch passes all 20 actions under both
authentication modes on protocol revision `2026-07-28`. It exercises release/tag/label CRUD,
receipt replay, latest-release selection, escaped changelog text, label assignment,
item linking and remove-wins behavior, and preservation of linked work items.
The additional acceptance evidence is recorded below.

## Acceptance

All twenty release, tag and label actions meet the pinned handler and
`plane-sdk==0.3.1` contracts. The inspected handlers are `release.py`,
`release_tag.py` and `release_label.py` at the revision pinned in the parity tracker;
SDK projections come from `plane/models/releases.py`.

[Field checks](../apps/go-api/internal/mcpactions/release_contract_integration_test.go)
cover false versus omitted updates, latest-release selection, HTML authority and
atomic rejection. [Visibility checks](../apps/go-api/internal/mcpactions/release_visibility_integration_test.go)
exclude foreign and project-owned records and filter private/guest item counts.
[Native picker checks](../apps/go-api/internal/mcpactions/release_picker_integration_test.go)
require workspace releases to be selectable, preserve referenced releases, and
reject retired choices.

[Authorization checks](../apps/go-api/internal/mcpactions/release_authorization_integration_test.go)
cover every mutation under both credentials, including revoked writes/replay,
request conflicts and credential changes during waits.
[Reference waits](../apps/go-api/internal/mcpactions/release_reference_wait_integration_test.go)
and [label replay checks](../apps/go-api/internal/mcpactions/release_reference_replay_integration_test.go)
require current lead, tag, label, project and item access before receipt replay.

[Recovery checks](../apps/go-api/internal/mcpactions/release_recovery_integration_test.go)
exercise cancelled waits, rejected commits, retry and competing revisions.
[Native-writer checks](../apps/go-api/internal/mcpactions/release_native_revision_integration_test.go)
cover all six revision-bearing storage boundaries.
[Worker checks](../apps/go-api/internal/mcpactions/initiative_worker_integration_test.go)
require immutable association history after rename/deletion and duplicate delivery.
[Rollback checks](../apps/go-api/internal/bootstrap/release_rollback_integration_test.go)
protect standalone tags/labels and retained audit, while allowing empty rollback.
Existing content workflows, native picker preservation/concurrency and receipt
timezone/opaque-description checks remain in force.
