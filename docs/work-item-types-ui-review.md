# Work item types: UI implementation status

The reference contract is [work-item-types-spec.md](work-item-types-spec.md).

## Interactive prototype

Open <http://localhost:8000/notion-sync/settings/work-item-types/>.
The workspace library and hierarchy use their settings destinations. The prototype
screens and interactions remain, but the preview warning banner, disclaimer copy,
and screen/state toolbar are removed from the rendered interface.

Other prototype screens remain available through direct links for review:

- [Project settings](http://localhost:8000/notion-sync/settings/work-item-types/?screen=Project%20settings)
- [Work items](http://localhost:8000/notion-sync/settings/work-item-types/?screen=Work%20item)

The optional `state` search parameter can select existing fixture states without
adding development controls to the page. These links support prototype review;
production navigation still needs integration into the corresponding surfaces.

## Implementation still required

Workspace type definitions now persist through the API described below. The remaining
prototype screens use temporary in-memory state. Formula execution, real release/member
catalogs, property and hierarchy persistence, type selection, and
custom-property search/filtering still need implementation. These limitations are tracked
here rather than rendered as warnings in the interface.

The full acceptance criteria in the spec remain open, including integration into
production project/work-item surfaces, custom-property filters, grouping, board
layouts, alternate peek modes, and production version history.

The unconnected backend draft is outside active source. Migration 000003 had
already been applied locally (version 3, clean), so its files remain to preserve
migration history. Transaction-local schema selection allows fresh installations
to succeed after the baseline clears the search path. The initial UI pass did not register an API. The implementation update below supersedes that limitation for workspace type definitions.

## Verification

Focused regression tests cover context-form initialization and unsaved edits,
context resolution and overlap, defaults, hierarchy conflicts, scoped cascading
removal, and readable preservation of rich text, false, and zero values.

## Workspace type logic — October 24, 2026

The default workspace library now reads and writes PostgreSQL through the generated
`/api/workspaces/{slug}/work-item-types/` API. Workspace admins can enable types,
create and edit definitions, activate or deactivate optional types, change rollout,
and delete unused types. Ordinary members can read the catalog. All mutations
validate workspace membership, use the workspace schema lock, and reject stale
catalog revisions. Mutation failures retain editor values and display the error.
The browser uses scoped TanStack Query data and native mutation state.

Enabling assigns the default Task to existing untyped issues and drafts without
replacing existing type assignments. Default types cannot be deactivated or deleted.
New types begin inactive, at level zero, with Import rollout. Mandatory rollout
activates the type and adds it to every existing and future project; switching
back to Import keeps project assignments. Project creation and draft type changes
share the schema lock with library edits (migration 000004).

Deletion currently rejects types referenced by any issue or draft, including
archived or deleted records. Bulk replacement and custom-property preservation on
type deletion remain separate work. Work-item type
selection, custom properties, contexts, formulas, and hierarchy persistence are
not connected by this slice; their existing interactive screens remain available.
Property attachments made in the UI are still local state. The normal library
contains real definitions; explicit scenario URLs still render their UI fixtures.

Verification covers the generated HTTP route's authentication, CSRF, input validation
and request scope, plus PostgreSQL authorization, stale revisions, duplicate names,
default protection, mandatory rollout to existing/new projects, rollout reversal,
draft safety, and durable deletion. Browser verification covered enabling, creating,
activating, editing, and reloading the local workspace library. The compact type editor now opens a searchable icon sheet from the tile beside
the name. The shared icon catalog and color swatches replace the five-name list
and standalone color input. The sheet uses Headless UI's anchored popover; icon
and color selection stay open, and Escape closes only the sheet. Saved catalog
icons and existing custom colors survive reload; failed saves retain appearance
choices with the rest of the draft.

Coverage review: the two new Go packages have explicit initial floors (74.83% for
the repository and 0% for the declarative API design, matching how existing design
packages are measured). Existing coverage floors and lint/duplication allowances
were not raised. The type route is mounted separately so adding it does not expand
the existing duplicated service-selection condition.

## Project availability — October 24, 2026

Project settings now has a real **Work item types** destination under Work structure.
For the local pilot, open
<http://localhost:8000/notion-sync/settings/projects/15a482d9-bde1-4820-b67e-0b2f3bc81b12/work-item-types/>.
This implements S7 imports and availability, independently of the retained project
preview screen. A project administrator can import active optional workspace types,
pause/resume imports, and remove an import after confirmation. Members can read
availability. Task and Mandatory types stay protected; inactive workspace types
remain recognizable but cannot be imported or resumed until activated.

Imports reference workspace definitions, so name/icon/color and rollout changes
appear without copying definitions. Removal preserves existing work items and
drafts and permits reimport. The API uses project membership/admin checks, the
workspace schema lock, and catalog revisions; rejected batches write nothing.
Project catalogs do not expose workspace-wide work-item or private draft counts.

Import failures retain selections and errors inside the dialog. If another admin
changes availability, stale selections remain visible and removable. Route scope
changes remount drafts; query requests are cancellable, and mutation completion
invalidates existing caches without restoring cleared session data.

Fresh independent backend and UI reviews identified and prompted fixes for
relationship edits persisting after a rejected issue type change, inaccessible
stale selections, removal errors behind the modal, and incorrect native Form
success state after a rejected import. Existing-item fields and relationship
updates now share one transaction. Creation initializes the custom-properties
column when inserting a whole record.

Focused database regressions cover permissions, protected types, batch rollback,
stale revisions, shared definition changes, pause/resume/removal/reimport, and
ordinary edits to existing items with unavailable types. Generated HTTP tests
cover session/CSRF and payload validation. Native Form/Query tests cover failed
save retries, stale choices, pending controls, and removal errors. Browser verification
covered import, reload persistence, pause/resume, and removal; the project was
returned to its original imports afterward.

Validation checkpoint: all 438 frontend behavior tests and production builds
passed. The web package typecheck, focused formatting, and unused-code ratchet
also passed. The full Prek invocation was not clean: frontend lint/format findings
came from the concurrent Form/Query migration, and its build hook observed files
changing during the run. The new Go lint findings were fixed; the model-layer
rollback/retry regression restores work-item coverage above its unchanged floor.
The subsequent complete Go gate passed, including lint, integration tests,
coverage, and static checks.
Both fresh reviewers rechecked their fixes with no remaining findings.

The next production integration is work-item type selection in creation and detail.
Custom-property persistence, contexts, formulas, hierarchy, migration of used
types, and discovery/display integration remain open in the main specification.
