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

The native reads, mutations, receipts, revisions and generated five-action MCP
projection are implemented. Real PostgreSQL tests exercise both credential types
and protocol revisions. The full gates passed at `87a791d9`, and local
browser/proxy verification passed on 2026-10-30. The
[state acceptance evidence](mcp-parity.md#state-acceptance-mcp-01-mcp-02) records
the reviewed tests, native/MCP edit sequence, stale-revision rejection and deletion
checks. Other MCP domains retain their separate acceptance requirements.

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

The MCP projection preserves the SDK's seventeen declared fields, nullable
metadata, required name/color, and the known native extra field `slug`. The SDK
State model uses `extra=allow`; its model dump retains extra input fields, as
[documented by Pydantic](https://pydantic.dev/docs/validation/2.12/concepts/models/#extra-data).
Thus the ordinary native record has eighteen fields through the SDK too. The
toolkit's seven-field page envelope remains separate. That projection is separate from the native
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

## Project mutations

The three project mutations use `internal/projectstatereads.WriteRepository`.
A new credential-neutral typed command carries immutable actor/workspace/project
UUIDs, workspace slug, action, optional state UUID, typed field presence, optional
request UUID/revision, and credential revalidation. Its transaction coordinator
uses `mutationreceipts.Execute`; existing HTTP operations and this command share
`createState`, `updateState` and an extracted transaction-scoped deletion helper.
No transport has a second SQL writer. Existing HTTP authorization policies and
ordinary response/status shapes remain unchanged; MCP selects the API-v1 member
write policy (project role 15 or 20), never the app PATCH guest exception.

Workspace writes follow the ungoverned SDK behavior: reject sequence/default at
workspace scope, require group on workspace creation, then require fresh actor
and workspace admission before returning `workspace_not_managed`. They create
neither a receipt nor a state. This is an explicit supported error contract, not
an independent workspace catalogue. Project creation rejects a target UUID and
`expected_revision`; update/delete require a nonzero target UUID. Delete rejects
field changes. Empty updates are valid and retain native timestamp behavior.
Malformed IDs/actions and empty revision strings are invalid input.

### Fields and identity

Typed changes contain optional name, color, description, group, sequence,
default, external_source and external_id. They do not expose `is_triage`.
The MCP adapter drops exact empty optional strings, retains false and zero, and
rejects nonfinite sequence values. Native validation uses the existing shared
text/boolean/number rules: name/color required for creation, maximum 255 characters
for name/color/external identity, and the five non-triage lifecycle groups. Group
is `backlog`, description is empty and default is false when omitted on project
creation. Omitted update fields retain their values. Native HTTP still supports
its existing explicit-null external identity clearing; the pinned MCP interface
cannot express that clearing and does not reinterpret an empty string as null.

Creation retains the native sequence rule: append at `MAX(sequence)+15000`, or
65535 if there is no ordinary live state. Supplied create sequence is accepted
but does not override that rule. Update sequence is applied, including zero.
Setting default true clears other live non-triage project defaults; setting it
false does not choose a replacement. The separate `projects.default_state_id`
reference remains separate from the state default flag.

Live names are case-sensitive and unique within the project through the existing
partial index. Names use the shared native text normalization and slug function;
there are no aliases. After normalization, case-insensitive `Triage` is reserved
for system creation and is rejected by both HTTP and MCP mutation validation.
Existing system-created triage rows remain usable by their existing internal
workflows, but MCP cannot retrieve, update or delete either a triage-group or
is_triage-flagged row. Legacy app/API treatment of is_triage flags is otherwise
retained; this command's scope excludes those rows before calling shared helpers.

A nonempty external-source/id pair is unique among live states in its project.
The shared API update helper checks the final pair when either component changes,
excluding the target itself. This closes the current source-only collision gap.
Explicit HTTP null/empty identity fields store the supplied null/empty value
and are outside the nonempty-pair collision rule. Native project-row locking
serializes these checks with other API state mutations.

### Authorization and transaction order

The command has the following order, including unkeyed requests:

1. Validate and normalize typed input; preserve supplied versus omitted update
   fields and canonical UUID/numeric values. Creation fills static defaults.
   Keep supplied create sequence in the request hash even though its placement
   is server-derived. Exclude credentials and request_id; include expected_revision.
2. Acquire the receipt lock when keyed. Lock current active user, live workspace
   and active known-role workspace membership for share, binding UUID and slug.
3. For project scope, lock the live, unarchived, correctly owned project for
   update, then its active role-15/20 membership for share. For workspace scope,
   revalidate the credential and return workspace_not_managed.
4. Revalidate the credential after all lock waits. Only then inspect a receipt.
   Replay requires current scope permissions but not the target's continued
   existence; changed normalized input conflicts.
5. For update/delete, lock the correctly owned live non-triage target before
   reading its revision in a subsequent statement. Compare expected_revision if
   supplied. A target outside this scope is not found; denied project admission
   is forbidden. Revision conflict changes nothing.
6. Apply the shared native mutation, read its resulting revision, persist its
   safe canonical result and any required outbox entry, and commit once.

Authorization locks survive through commit. Missing accounts/memberships,
workspace slug reassignment, project deletion/archival and role revocation fail
closed. The domain exposes classified errors; transport errors do not echo SQL,
raw errors, credentials or request bodies. Commit uncertainty uses the shared
indeterminate outcome. Successful commit is not relabelled as failure by a late
cache error or request cancellation.

### Revisions and receipts

A new migration adds `state_edit_revisions(state_id, revision)` outside the
existing state row projection, seeded for all states and maintained by an
AFTER INSERT OR UPDATE trigger. Every state-row writer advances the counter,
including native HTTP, default changes, workers, deletion and direct SQL. An
update that retains the same business values may still advance it; this matches
the native timestamp write. The opaque token is `state:<uuid>:<counter>`. Physical
state deletion removes its revision row, but never its receipt.

Create/update ordinarily return the pinned state record; delete ordinarily
returns no content. With request_id or expected_revision, MCP returns exactly
`{state_id, revision, result}`, where result is the ordinary record or null for
delete. Keyed results additionally contain request_id, replay_expires_at and
replayed. Ordinary reads never add revision. Bound detail with include_revision
adds only revision, from the same SQL snapshot as authorization and the record;
list does not accept that extension. Existing native HTTP payloads/statuses are
not changed to expose these extensions in this work.

### Deletion and durable cleanup

Shared native deletion locks the project and target and rejects the default-flag
state and any state referenced by a live issue **or live draft**. Adding the draft
check intentionally corrects the previous possibility of cascading deletion of
an otherwise hidden draft. Soft-deleted issue/draft history does not block
cleanup. Deletion soft-deletes the state and inserts one `work_item_effects` row
of kind `soft_delete`, with the existing `{app:"db",model:"state",id}` payload,
in the same transaction. HTTP uses that same durable helper; its success no
longer depends on a publisher after commit. Outbox failure rolls back deletion.
Replay inserts no second effect. The existing deletion workflow admits the
immutable outbox ID, retries delivery and clears applicable live references,
including the separate project default, under the existing deletion policy.
Already-soft-deleted issue/draft dependents remain during this soft cleanup.
Later retention cleanup physically deletes the state and its historical
dependents through the existing database CASCADE foreign keys.

A new deferred constraint trigger on issue/draft insertion and updates of
state_id/deleted_at checks the final live row at commit and locks its selected
state FOR SHARE. A selected state must exist and be live. Deferral preserves
transactional insert ordering and permits later corrections/deletions within
the same transaction. The trigger does not scan or rewrite historical data and
does not reject unrelated edits to historical rows. It covers direct SQL and
legacy writers as well as the modern item-edit selection locks. Together with
the deletion helper's FOR UPDATE lock and subsequent live-reference query,
this prevents a concurrent assignment from committing to a state admitted for
deletion. A new reference waits and then fails if deletion won; if assignment
won, deletion observes the committed reference and fails. Whole-project/workspace
cascade deletion remains available through its existing worker policy; this
constraint validates new live references, not every state soft deletion.

State create/update have no existing activity or webhook publisher, so this work
adds none. The existing workspace-state Redis cache remains best effort. The
bound command attempts invalidation after acknowledged success, including
replay, with a bounded context; it cannot turn a committed mutation into a
reported failure. MCP reads use primary state data, never this cache.

## Mutation proof obligations

Real PostgreSQL evidence must cover ordinary create/update/delete, zero/false
presence, static defaults and native sequence behavior, name/triage/identity
rules, empty updates and null/empty HTTP compatibility. Every external identity
component change must receive the same collision check. Revisions advance for
non-MCP writers and for every state whose default changes. Receipt replay must
precede CAS and target lookup, require fresh authorization after waits, reject
changed input, survive deletion, and commit atomically with state and outbox.

Concurrent state assignment and deletion must have only the two outcomes above,
for issues and drafts and for direct SQL, with evidence that a call actually
waited on the relevant lock. Deferred checks must accept later correction,
deferred parent creation and final row deletion in the same transaction. The
native deletion workflow must process the committed state outbox and reconcile
repeated admission without losing live-reference cleanup. Soft cleanup must
retain historical issue/draft rows; a later hard deletion must remove them
through the existing foreign keys. Existing HTTP fixtures retain their
ordinary shapes and policies, with explicit regressions for the reserved name,
source-only identity collision and live-draft deletion corrections. Full MCP
completion additionally requires generated action contracts, permission-filtered
discovery, sanitized telemetry, both authentication modes/protocol revisions,
full quality gates and live proof; native mutation success alone is insufficient.

## MCP contract and projection

The authored `mcp/design/state.go` contract registers one `state` method through
`mcp/design/design.go`, with five closed action variants: `list`, `retrieve`,
`create`, `update`, `delete`. There are no legacy aliases. Generated service
interfaces, request/result unions, JSON-RPC endpoints, MCP tool schemas and
streamable HTTP projections under `mcp/gen` derive from that design. The existing
runtime enablement, credential verifier and permission-filtered transport remain
the only public entry point; registration does not enable MCP in deployment.

| Action   | Required inputs     | Optional inputs                                                                                                             | Ordinary output                           | Native route           |
| -------- | ------------------- | --------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------- | ---------------------- |
| list     | action              | project_id, cursor, per_page                                                                                                | seven-field toolkit page of state records | bound ReadAPIV1 list   |
| retrieve | action, state_id    | project_id, include_revision                                                                                                | state record                              | bound ReadAPIV1 detail |
| create   | action, name, color | project_id, description, group, sequence, default, external_source, external_id, request_id                                 | state record                              | MutateState create     |
| update   | action, state_id    | project_id, name, color, description, group, sequence, default, external_source, external_id, request_id, expected_revision | state record                              | MutateState update     |
| delete   | action, state_id    | project_id, request_id, expected_revision                                                                                   | null / empty MCP content                  | MutateState delete     |

Every action also accepts the existing optional workspace_slug selector, which
must match the authenticated binding. Project omission or exact empty string
selects workspace scope. Other IDs must be nonzero UUIDs. Unknown properties,
wrong-action fields and unknown actions fail validation. Create deliberately does
not offer expected_revision. Per-page zero/omission uses 1000; 1–1000 is valid;
cursor omission/empty starts at the beginning. Exact empty optional text fields
are omitted; sequence/default also accept JSON null as omission, while false/zero
are retained. Update external_source/external_id follow
the pinned handler's actual payload(UpdateState), despite omission from its
short ACTIONS description. Workspace write limitations are defined above.

The typed record declares all seventeen SDK fields (name/color required strings,
the remaining fields required nullable values), optional typed slug, and optional
revision. The native projection always includes slug; revision appears only for
an explicitly requested detail. A page has results, count, total_count,
next_cursor, prev_cursor, next_page_results, prev_page_results and no native
extra envelope keys. Mutations without extensions return the ordinary record or
null; extensions select the receipt shape already specified above. No untyped
result map or parallel SQL path is exposed. Timestamps use the same principal
timezone conversion as other MCP API resources, including replayed results.

Discovery offers list/retrieve when the current active user has a live known-role
workspace membership, even with zero accessible projects; project mutations
require at least one active role-15/20 membership in a live unarchived project.
OAuth read/write scopes further restrict those actions. Each invocation still
checks its actual target and current credential; advertisement is not authority.
Both supported MCP protocol revisions and PAT/OAuth use identical schemas and
result semantics after filtering. No second discovery catalogue is introduced.

Errors map to the existing unauthorized, forbidden, invalid_params, not_found,
conflict, revision_conflict, request_conflict, indeterminate and internal_error
contracts. Workspace mutation rejection has the explicit workspace_not_managed
error with a safe message. Native write status 400/403/404/409 maps to
invalid_params/forbidden/not_found/conflict, without exposing raw bodies or SQL.
Mutation telemetry uses the shared action recorder, reporting action, outcome,
latency and committed IDs/replay status only; no name, color, external identity,
request ID, revision, credentials or request body is recorded.

Transport proof must compare actual JSON records/pages with the native API
projection and pinned model behavior; exercise all five actions through real
PAT and OAuth authentication on both supported protocol revisions; and verify
closed schemas, empty content, receipt replay, CAS conflicts, workspace rejection,
current permission-filtered discovery and secret-free mutation telemetry.
