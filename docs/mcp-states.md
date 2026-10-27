# State application contract

## Scope and evidence

The pinned `state` handler has five actions: list, retrieve, create, update and
delete. Its optional `project_id` selects the project API; omission selects the
workspace API. The compatibility sources are the [pinned handler](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/state.py)
and Plane SDK 0.3.1 `plane/api/states.py`, `plane/api/workspace_states.py` and
`plane/models/states.py`.

The SDK explicitly distinguishes governed and ungoverned workspaces. In an
ungoverned workspace, workspace reads aggregate accessible project states and
workspace writes fail with `workspace_not_managed`. Vector currently has only
project-owned states. Its workspace metadata state list already aggregates them.
Adding an independent catalogue without workflows would not implement governance.
Vector retains the ungoverned contract; this work does not enable governance or
change the existing workspace metadata endpoint. Governed workflows require their
own native domain before they can be advertised as supported.

This document currently settles the read boundary. Project mutation integration,
receipts, revisions and durable deletion remain required before the complete
five-action tool can be claimed. A read implementation alone is not full state
parity and does not complete the full MCP catalogue.

## Canonical read boundary

`internal/projectstatereads.Repository.ReadAPIV1` remains the canonical API state
projection. Its `APIV1Request` adopts the shared
`djangoapi.ProjectReadRequest`. A non-nil `ExpectedWorkspaceID` selects a bound
read that requires a fresh primary-database authorization snapshot. An absent
binding retains the existing HTTP request contract, including serialization,
field selection, expansion, ordering and pagination. No new public HTTP route is
introduced by this change.

| Bound request                                     | Meaning                                         | Result                         |
| ------------------------------------------------- | ----------------------------------------------- | ------------------------------ |
| `list`, nonempty project UUID                     | Project states                                  | Native API page                |
| `detail`, nonempty project UUID and state UUID    | Project state                                   | Native API record or not found |
| `list`, omitted project                           | Accessible states across the bound workspace    | Native API page                |
| `detail`, omitted project and supplied state UUID | Accessible project state in the bound workspace | Native API record or not found |
| Any other operation                               | Unsupported read                                | Forbidden                      |

Bound reads reject zero or malformed actor, workspace, project or target UUIDs,
an empty workspace slug, and field/expansion requests. The MCP adapter does not
offer arbitrary serializer expansion, which could otherwise perform additional
queries outside the admitted snapshot. Exact empty optional project strings mean
omission. UUIDs that are well formed but unknown never broaden the scope.

All of these invalid bound requests return `ErrForbidden`, including a missing
`state_id` on detail and an explicitly present empty `fields` or `expand` value.
Presence flags or nonempty parameter values each suffice to reject those two
serializer options. This internal boundary fails closed; the later MCP adapter
can return invalid-parameters errors for malformed public inputs before calling
it. List paging retains `djangoapi.PageWindow` and `Envelope`, including their
defaults and invalid-query errors. Detail ignores paging parameters as before.

The bound workspace must match both immutable UUID and current slug, be live,
and have an active known-role membership (5, 15 or 20) for an active user. Project
rows must belong to that workspace, be live and unarchived, and have an active,
live, correctly owned membership for that user with a known role. All three
existing roles can read states, irrespective of guest feature flags. A workspace
administrator does not implicitly gain access to projects without membership.

Project-scoped denial returns forbidden, including archived or unavailable
projects. Workspace-scoped reads require workspace admission but may legitimately
return an empty page when no projects are accessible. Workspace detail returns
the same not-found error for unknown, foreign, inaccessible, archived and deleted
targets. A caller cannot use detail to discover a hidden project's existence.

The state itself must have matching project and workspace ownership, be live,
have `group != triage` and `is_triage = false`. Filtering occurs before counting,
ordering and pagination. Bound ordering is `(sequence, id)` for project reads
and `(project_id, sequence, id)` for workspace reads. The UUID tie breaker makes
equal sequence values deterministic. No new catalogue state or project lifecycle
state is inferred from these rows.

Authorization, rows and count use one SQL statement and one snapshot. Scope
admission is independent of whether matching states exist. Counts cannot expose
inaccessible rows. Concurrent revocation committed before the statement is
observed; reads already executing retain normal PostgreSQL statement-snapshot
semantics.

## Projection and compatibility

The canonical API serializer returns the existing eighteen fields: `id`,
`created_at`, `updated_at`, `deleted_at`, `created_by`, `updated_by`, `workspace`,
`project`, `name`, `description`, `color`, `slug`, `sequence`, `group`, `is_triage`,
`default`, `external_source` and `external_id`. It retains numeric sequence
normalization and the shared API pagination envelope. Bound reads select the
same record projection as unbound reads with no field/expansion options.

The later MCP projection must follow the SDK's seventeen declared fields (the
same fields without `slug`), nullable metadata, required name/color and the
toolkit's seven-field page envelope. That projection is separate from the native
HTTP envelope. The generated authored MCP design will own registration; this
read-boundary change registers no action and changes no discovery result.

OAuth remains provider agnostic. The adapter must bind both personal-token and
OAuth calls to the same verified workspace identity; scope filtering does not
replace database membership checks. Runtime MCP enablement remains independent
of this native read capability.

## Proof obligations

Real PostgreSQL tests must show that bound project and workspace reads agree on
the same visible record; equal sequence values paginate deterministically; and
counts exclude foreign ownership, deleted/archived resources, inactive users,
revoked memberships and triage rows. Valid empty workspace admission differs
from forbidden scope. Detail masks inaccessible targets. Slug reassignment
cannot move a call to another workspace. Zero or malformed IDs and unsupported
operations fail before querying. Bound serializer expansion is rejected.

Existing native state tests must retain their HTTP representations and
pagination behavior. Later generated transport proof must cover the pinned
record/page shapes, discovery, both authentication modes and both protocol
revisions before the state tool is counted as implemented.
