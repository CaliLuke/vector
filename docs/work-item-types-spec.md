# Work item types: UI and behavior specification

Status: UI-first design for review. Researched 2026-10-24.

This specifies Vector's workspace-managed work item types, including all eleven
property kinds, formulas, contexts, and hierarchy. The first implementation
deliverable is an interactive UI with representative placeholder content. Its
screens and states must be comparable with the documentation before persistence
is designed around them. This document deliberately does not prescribe tables,
migrations, or endpoint payloads.

## Reading this specification

- **Documented** identifies behavior described by the linked Plane documentation.
- **Observed** identifies layout visible in a documentation screenshot.
- **Vector decision** supplies a concrete proposed contract where the documentation
  is incomplete or where integration with Vector requires a choice.
- **Unresolved** identifies a decision that must remain visible in the prototype;
  it is not permission to silently omit a capability.

This is an independently authored product contract, not a claim that the paid
implementation or its private API has been inspected. Sources are listed at the
end. Source references such as [W] resolve there.

## Product scope and intended experience

A workspace administrator defines a reusable vocabulary of work: Task, Bug,
Story, Incident, and other types. Each type exposes an ordered set of properties.
A project administrator chooses which workspace types the project uses. Members
encounter those choices in ordinary creation, editing, viewing, and filtering.
Hierarchy optionally constrains which types may be parents and children. [W]

The selected model is the workspace library from the user's linked article.
The separately documented project-owned model has different ownership and
defaults; it is reference material, not a second mode to build. Vector does not
reproduce Plane's commercial plan gates. Existing authorization still applies.

Included: library management, rollout, project imports, property configuration
and attachment, contexts, every property editor, formula authoring and results,
cascading management, hierarchy, value preservation, activity, rich-text history
and search, and work-item discovery/display integration.

Outside this feature: billing, a new permissions framework, workflow automation,
the complete releases product, and redesigning unrelated work-item screens.
The Release picker still needs a real release catalog to be functional; that
integration dependency is not satisfied by a permanently fake dropdown.

## Screen map

All navigation below describes product locations, not committed URL schemas.
Existing workspace and project settings shells remain the surrounding UI.

```text
Workspace settings
  Work item Types
    Enable state
    Work item Types tab
      Create / edit type
      Expand type → attached properties → Add properties
      Rollout, activation, deletion
    Properties tab
      Create / edit property
      Manage contexts
      Manage cascading levels / options
      Delete property and impact preview
  Hierarchy
    Enable state
    Level builder → Save → conflict review

Project settings
  Work item Types
    Imported and mandatory types
    Import from workspace

Work items
  Create / update modal → project, type, applicable properties
  Full detail / side peek / modal peek → type, values, history
  Switch type → edit → preservation confirmation
  Parent / sub-work-item selection → allowed types
  Lists / board / table / filters / display → type and property projections
```

### Documentation screenshot inventory

These originals were visually inspected. Links intentionally retain attribution
and avoid copying upstream media into the repository. Screenshots establish
composition; they do not establish unseen behavior or exact pixel dimensions.

| Reference                                                                                   | Observed composition                                                                                                                                             | Vector comparison target           |
| ------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------- |
| [R1: Workspace library](https://media.docs.plane.so/workspaces/work-item-types.webp)        | Settings sidebar; heading and subtitle; two tabs; section count and right-aligned add action; bordered expandable rows with icon, description, badge/menu/toggle | S1 library shell and expanded row  |
| [R2: Create type](https://media.docs.plane.so/workspaces/create-work-item-type.webp)        | Compact centered modal; icon tile beside name; description below; Cancel and primary action at bottom right                                                      | S2 type editor                     |
| [R3: Create property](https://media.docs.plane.so/workspaces/create-custom-property.webp)   | Taller modal; title, description, kind selector, conditional controls; mandatory checkbox in footer left; actions right                                          | S4 property editor with every kind |
| [R4: Attach properties](https://media.docs.plane.so/workspaces/add-properties-to-type.webp) | Checkbox rows; property title left, kind badge right; Cancel/Add footer                                                                                          | S3 attachment dialog               |
| [R5: Import types](https://media.docs.plane.so/workspaces/import-from-workspace.webp)       | Project settings behind modal; checkbox, icon, title/description rows; counted Add action                                                                        | S7 import dialog                   |
| [R6: Choose type](https://media.docs.plane.so/workspaces/choose-work-item-type.webp)        | Project then type selector at modal top; searchable icon/name menu with selection check; title/editor/property area below                                        | S9 create/update form              |
| [R7: Switch affordance](https://media.docs.plane.so/work-item-types/switch-issue-type.webp) | Switch control beside item identifier above title                                                                                                                | S10 switch entry point             |
| [R8: Item detail](https://media.docs.plane.so/work-items/work-item-detail.webp)             | Type badge above title; frequent properties below title; right sidebar with Details, Project structure, Custom properties; activity below content                | S10 full and peek detail           |

No dedicated context, formula-editor, or hierarchy screenshot was embedded in
the reviewed type articles. Those layouts below are Vector proposals derived
from described controls and the surrounding UI conventions, not observed copies.

## UI contracts

### S1. Workspace type library

Before activation, show the feature purpose, an enable control, and an explicit
irreversibility explanation. Activation exposes the two tabs and built-in Task.
There is no feature-off control after activation. [W]

Match R1's settings layout. Rows contain an expand control, colored icon, name,
description, attached-property count, activation control, and overflow menu.
Task has a distinct protected/default indication. Row expansion reveals ordered
properties and **Add properties**; the toggle and overflow menu must not also
expand the row. Long descriptions wrap without displacing the row actions.

Vector decision: show Import or Mandatory as a readable rollout badge. Explain
that Mandatory means availability across projects, not a required field. Provide
the rollout editor through the row menu; never reuse the property-required toggle.

States: disabled feature; Task only; populated; expanded type with no properties;
long names; inactive type; mandatory type; loading; load failure/retry; mutation
failure with prior saved state restored; permission denied.

### S2. Create and edit type

Match R2: icon/color control adjacent to the name field, description beneath,
Cancel and Add work item type in the footer. Edit uses the same structure and
an Update action. Keep visible inputs minimal: the icon tile beside the name and
an optional description, with accessible field names. Clicking the tile opens an
anchored sheet with a searchable, scrollable icon grid. Show the shared color
palette as tiles below it, with a centered check on the selected color. Selecting
an icon or color updates the draft and keeps the sheet open; Escape, the trigger,
and outside clicks close the sheet. Existing custom colors remain selectable when
editing. Use the shared icon catalog and retain legacy saved icon identifiers.
This expanded picker is a Vector design decision; R2 shows only its trigger.
Require a name; show duplicate-name errors beside that field.
Creation leaves the type inactive and rollout set to Import. [W]

Vector decision: description is optional; names compare after trimming and
case-folding. Saving a new type keeps it visible in the library and labels it
Inactive. Task's protected operations are unavailable with an explanation.

### S3. Attach and arrange properties

Match R4. The modal selects existing workspace properties, rather than creating
copies. Already attached properties do not appear as new choices. Add is disabled
for zero selections. Distinguish an empty library from all properties already
being attached, with a route to the Properties tab. [W]

The expanded type lists kind, title, required/optional state, and active state.
Reordering changes only this type's display order. Provide keyboard movement as
well as drag handles. Attachment rejects conflicting titles/internal references.

Vector decision: detaching a used property is a preservation operation scoped
to this type. Preview affected items and append their removed values to their
descriptions, leaving the reusable definition and other attachments intact.

### S4. Property library and editor

The Properties tab lists title, kind, active state, attached types, and an overflow
menu for Edit, Manage contexts, and Delete. Cascading rows also expose Manage
levels and Manage options. Vector decision: show context count and usage summary;
the library must remain useful when a property is not yet attached anywhere.

Match R3's form structure. Common fields are title, description, icon, property
kind, active state, and applicable default/required controls. The kind selector
reveals the specific editor in the table below. Unsupported controls are absent
or disabled with an explanation; they must not silently accept ignored settings.

Keep the header and action footer available while long option lists scroll. A
used property's locked structural fields display why they are locked. A context
override must not disguise a forbidden change to the property's underlying kind.

| Kind           | Configuration UI                                                                   | Work-item editor / display                     | Required and default behavior                                                             |
| -------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------- | ----------------------------------------------------------------------------------------- |
| Text           | Single line, Paragraph, Read-only; fixed content for Read-only                     | Input, textarea, or non-editable text          | Editable formats may be required; no default; format immutable after creation             |
| Number         | Decimal value control; optional default                                            | Numeric editor with finite-number validation   | Required or default, never both                                                           |
| Dropdown       | Single/multi control; named/icon options; parent option nesting; default selection | Searchable one/many picker                     | At least one configured option; defaults respect cardinality; no default when required    |
| Boolean        | Fixed two-state behavior                                                           | Toggle with explicit true/false                | Initially false; cannot be required                                                       |
| Date           | Format selector with sample output                                                 | Calendar/input; consistent display format      | May be required; no default                                                               |
| Member picker  | Single/multi; optional default members                                             | Searchable workspace people picker             | May be required; optional defaults; selected members become subscribers subject to access |
| Release picker | Single/multi                                                                       | Searchable project release references          | May be required; no default; does not change release scope                                |
| Rich text      | Document-field configuration                                                       | Rich editor, saved/read state, version history | May be required; no default; searchable; no filtering/grouping                            |
| URL            | Link-field configuration                                                           | One URL input and clickable saved value        | May be required; no default                                                               |
| Formula        | Expression editor, field/function reference, preview                               | Read-only result, empty state, or error state  | Cannot be required or manually set                                                        |
| Cascading      | Two to four levels, per-level required state, then options manager                 | Ordered dependent selectors                    | Required per level; no whole-property required/default control; unavailable on drafts     |

This table states the target behavior drawn from [W], with the release reference
distinction established by [R]. Formula-editor presentation and preview are Vector
decisions. Date display choices are `Jan 15, 2025`, `15/01/2025`, `01/15/2025`, and
`2025/01/15`; changing presentation must not change stored dates.

### S5. Context management

Entry: property's overflow menu on Properties. The default context covers all
projects/types; more specific contexts customize behavior. [W]

Vector proposed layout: a contexts list beside an editor, or a full-width list
leading to an editor on narrow screens. Each row summarizes name, project scope,
type scope, and overridden settings. Default is protected. An editor contains
project and type multi-selects, an All choice for each dimension, and individual
**Use base setting / Override** controls. Show the effective result for a selected
project/type so the administrator can verify precedence before saving.

The screen must demonstrate inherited, overridden, and conflicting configurations.
Cascading options are edited inside the selected context. Deleting a context
previews fallback effects; it must not quietly leave values referring to options
that no longer exist. Proposed resolution rules are specified below.

### S6. Cascading levels and options

Creation starts with two named level fields. Add level stops at four. Manage
levels allows renaming, appending, or deleting a level and its descendants; it
does not offer reorder. Labels allow up to 100 characters. [W]

Vector proposed options layout: one column per level, with the selected branch
highlighted. Add/edit options within the current parent. Show the branch-specific
next-field label next to the parent option. Empty branches get an Add option
action, not a blank unexplained column. Creation can skip options and return later.

Required settings propagate upward. A selected leaf with no further options ends
the path even if a deeper level is marked required. Changing an ancestor clears
incompatible descendant draft choices and immediately shows what remains required.
Persist the whole path together; do not save an invalid partial required path.

Removing in-use options/levels shows affected counts and the values that will be
preserved in descriptions. Require the documented case-sensitive `Confirm` text.

### S7. Project availability and import

The project settings page shows Task, imported types, and mandatory workspace
types. Match R5 for import: select rows with checkbox/icon/name/description and
use an Add action showing the selected count. Imports are links to the workspace
definition; projects do not get editable copies. [W]

Vector decisions: project admins can pause/remove optional imports, but existing
items remain readable/editable with their original type. Mandatory and Task rows
cannot be paused or removed here. Inactive workspace types remain recognizable in
settings but cannot become selectable through a project override. Removing an
import does not remove properties or values from existing items.

Show an empty-import state, all-available-types-already-added state, and inactive
library rows. Default-type selection is a separately labeled decision below; do
not conflate a project default with the workspace's protected Task.

### S8. Hierarchy

Use a separate workspace settings destination, consistent with R1's sidebar.
Before activation explain the dependency on work item types and irreversibility.
The builder places numbered levels vertically, broader types above leaf types,
with connecting arrows. Each level contains draggable type chips, a default
badge/menu, and a drop target. A top drop zone creates a level when a type is
placed there; the keyboard move menu also offers New level. There is no action
that creates an empty level. Link to the type library for creating additional
types. Save changes and Discard act on both placement and the default level. [W]

Vector proposed states: all types at zero; a three-level configuration; multiple
types at one level; drag target; keyboard move menu; unsaved changes; saving;
invalid configuration; stale configuration; conflict preview.

The conflict dialog shows the number of invalid parent links and an inspectable
list of affected pairs. **Go back and fix** retains the draft. **Save anyway**
requires the documented lowercase `confirm`. It unlinks invalid relationships,
never deletes work items, and records activity. Preview counts must reflect the
configuration actually being committed, not a stale earlier request.

### S9. Create, update, and draft work items

Match R6: type selection next to project selection at the top. Display type icon
and name; the menu has search and a selected check. Changing type updates the
custom-property area in its configured order. Preserve existing title,
description, and unaffected standard fields. Field errors remain adjacent to
their editor and Save focuses the first invalid field. [W]

Vector decisions: unsaved values for another type remain in the local form draft
when switching back before submission. Only the selected type's values are sent.
Changing project reruns availability/context validation and visibly identifies
values that no longer fit; do not silently replace a selected type.

Drafts remain private. Their Save action permits incomplete required fields; the
publish action validates the current project/type/context. Cascading editors are
not available in draft editing; publishing must present them if required. Do not
send subscriber notifications until publication. [D] establishes draft privacy
and publication; incomplete-field handling is a Vector decision.

### S10. Detail, switch type, and value history

Match R7/R8: type indication near the identifier/title; switch action accessible
on hover and keyboard focus; custom fields below the existing property groups.
Apply the same field order and renderers to full detail, side peek, and modal peek.
Read-only viewers see values without edit affordances. Inactive types retain
their identity on existing items. [W], [I]

Switching opens the update form. After choosing a type and satisfying its required
fields, Update opens a confirmation explaining which old-only values move into
the description. Shared property values remain attached by identity. A rejected
hierarchy or validation change leaves the original item untouched and the form
draft intact. Bulk type switching is not supported. [W]
Switching back later does not automatically extract preserved text from the
description and restore old field values; the confirmation must make this clear.

Custom-property changes appear in activity with property label, old/new value,
actor, and time. Rich-text fields expose earlier versions independently from the
main description. Vector decision: version restore creates a new version and
requires item-edit permission; neither field histories nor search snippets may
escape item visibility restrictions.

### S11. Filters, layouts, and discovery

Use the existing filter row: property, operator, value. New properties must use
the repaired anchored popovers, toggle closed on the second trigger click, and
close sibling menus. Preserve saved expressions across navigation/reload.

Type is a display property and a grouping/subgrouping choice. Parent-type
grouping chooses a type and groups beneath its parents; it conflicts with the
separate Show sub-work items expansion option. [V]

| Field family      | Filter controls required by [F]                                                                        |
| ----------------- | ------------------------------------------------------------------------------------------------------ |
| Type              | Equality/inequality against one type; inclusion/exclusion across several                               |
| Text              | Equality, inequality, contains, does not contain, empty                                                |
| Number            | Equality/inequality; `<`, `<=`, `>`, `>=` and their negations; inclusive range and its negation; empty |
| Dropdown / Member | One or many selected values; inclusion/exclusion; empty                                                |
| Boolean           | Is / is not, true or false                                                                             |
| Date              | Equality/inequality; before/after with inclusive forms and negations; range and negation; empty        |
| URL               | Equality/inequality; contains / does not contain                                                       |

Vector decisions: formulas expose operators for their result kind; releases use
selection operators; cascading levels use selection operators with the stable
level name, not branch aliases. Rich text has no filter or grouping control.
Options are identified by property/context/option identity, never label alone.

List, board, and table expose type consistently. Table supports custom-property
columns, including separate cascading levels; long rich text uses a preview that
opens the editor. Custom grouping is initially defined for scalar categorical
fields (single Dropdown, Boolean, single Member, single Release, cascading level).
Other custom grouping is explicitly unsupported; do not present inert controls.
The docs do not specify a complete custom grouping matrix, so this is a Vector
decision, not a claim of paid-version equivalence.

## Placeholder content and UI comparison contract

The first build makes the screens above navigable using deterministic local
fixtures. Keep the interactive prototype and its screens. The user requested removal
of the warning/disclaimer UI and the preview screen/state toolbar. Keep development
controls out of the rendered interface; direct review links can select fixture
screens and states. Do not delete prototype functionality as part of that cleanup.

Do not write fixture edits into the user's imported project or describe them as
durable production changes. Track missing integration in the implementation status
document. Use fictional content, not customer data. A useful fixture contains:

- Task, Story, Bug, and Incident, plus one inactive type and one long type name.
- Two projects with different imports and one Mandatory type.
- A property reused by Bug and Incident; a property attached nowhere.
- At least one instance of every property kind, including all Text formats and
  both single/multi selectors.
- Default and project/type-specific contexts with visible inheritance differences.
- A cascading tree with uneven branch depths and a branch-specific label.
- A formula returning a number, a date, text, and a boolean; empty and error results.
- A three-level hierarchy with a deliberately conflicting parent/child pair.
- Used property/type deletion previews, rich-text versions, and activity records.
- Editable, read-only, loading, empty, invalid, failed-save, and stale-save states.

Comparison is structural: navigation placement, tab hierarchy, row anatomy, modal
field order, footer actions, selector placement, custom-property sidebar placement,
and content density. It is not a demand to recreate screenshot annotations or
unrelated paid controls. At narrow widths dialogs stay in the viewport, long
lists scroll internally, and the property sidebar becomes an accessible stacked
section. At 200% zoom no essential action is clipped.

Keyboard-only operation must reach every action, trap focus in dialogs, restore
focus on close, and provide a non-drag way to reorder. Tooltips cannot be the only
source of a field name or validation error. Esc closes the active overlay first.

## Behavioral rules behind the screens

### Type lifecycle and availability

| Operation          | Observable result                           | Rejection / preservation                     |
| ------------------ | ------------------------------------------- | -------------------------------------------- |
| Enable types       | Task becomes available throughout workspace | Cannot disable the feature afterward         |
| Create type        | Inactive library row; Import rollout        | Duplicate workspace name rejected            |
| Activate           | Eligible for project use                    | Does not automatically import optional type  |
| Set Mandatory      | Added to all existing and future projects   | Project cannot remove/deactivate             |
| Mandatory → Import | Existing assignments remain                 | No automatic project cleanup                 |
| Deactivate         | Excluded from new selections                | Existing type and values survive             |
| Rename/change icon | Shared presentation updates everywhere      | Does not change type identity or values      |
| Delete type        | Requires an explicit migration outcome      | Task protected; no orphaned or deleted items |

Vector decision: deletion presents a replacement type and an impact preview.
Every affected item must be valid after reassignment; if one replacement cannot
satisfy existing hierarchy/required fields across projects, deletion is blocked
until items are migrated individually. No invisible fallback conversion to Task.
The docs require reassignment but do not specify its transaction or dialog.

### Property identity, inheritance, and required values

Property labels are unique within each attached type. Reserved field names include
state, due date, cycle, and modules. Internal references are derived from labels.
Titles can change; values and formulas must survive a label-only rename. [W]

Vector decisions:

- Keep immutable property identity and a stable internal name after creation.
  Attachment validates normalized reference collisions, including built-in names.
- Normalize reference lookup by case and spaces/underscores; reject ambiguity,
  rather than choosing the first matching field.
- An inactive property is hidden, retains values, and is not required on edits.
  Re-enabling restores the values and normal validation.
- Defaults apply to creation, not to later reads or clearing an existing field.
  Setting required removes an incompatible default in the same save.
- Zero and false are values. Empty string/whitespace-only editable text, an empty
  selection, null, and rich text without meaningful sanitized content are empty.
- Newly required fields block the next item mutation, including inline edits;
  present an editor for the missing fields rather than an unexplained save failure.
  Comments/subscriptions are not item-property mutations.
- A used property cannot change kind or selection cardinality, even after its last
  value is cleared. Draft usage also counts. Text format is immutable from creation.

The generic structural-lock rule in [W] appears to conflict with its editable Date
format and cascading management sections. Vector treats date presentation and the
explicit cascading operations as exceptions. Ordinary Dropdown option structure
locks after use; a future mutable-option migration is not implied.

### Context resolution

The documentation establishes a default and the most-specific matching context,
but does not define tie-breaking. Proposed Vector contract:

```text
matching context = project is in scope AND type is in scope
All is a wildcard, not an empty selection
specificity = number of constrained dimensions (0, 1, or 2)
winner = matching context with greatest specificity
effective settings = base settings + explicitly overridden winner settings
equal-specificity contexts whose scopes overlap are rejected at save time
```

This deliberately prevents an order-dependent answer for project-only versus
type-only contexts. A winner's non-overridden fields inherit from base, not from
an unrelated lower-priority context. Narrower lists of projects do not break ties.

Vector override allowlist: active, required, applicable default, date format, and
cascading option catalog. Type/kind, Text format, internal name, and selection
cardinality remain shared. Other overrides are unresolved until explicitly added
to this contract. Changing scope or deleting a context previews incompatible
values and preserves removed values in descriptions; required-field policy still
applies to subsequent edits. Invalid references never silently become valid by
matching another context's option label.

### Formula authoring and evaluation

The editor must distinguish invalid expressions from valid expressions whose
result is empty or fails for a particular item. Source [W] defines the following
language surface; detailed evaluation choices below are Vector decisions.

| Surface         | Contract                                                                                     |
| --------------- | -------------------------------------------------------------------------------------------- |
| References      | `{{field_name}}`; same-type Text, Number, Date, Boolean; no self-reference                   |
| Results         | Number, Text, Date, Boolean                                                                  |
| Operators       | Numeric `+ - * /`; date subtraction in days; date ± numeric days; text `&`; `= != < <= > >=` |
| Functions       | `IF`, `ROUND`, `ABS`, `UPPER`, `LOWER`, `LEN`, `CONCAT`, `TODAY`, `NOW`                      |
| Empty inputs    | Propagate empty; an unchosen IF branch is not evaluated                                      |
| Runtime failure | Visible error state, not zero or an empty success value                                      |

Vector proposed editor has an expression area, insert-field menu, function help,
result-kind selector, and sample-item preview. Preview includes the effective
context and evaluation time. Invalid syntax/references block saving the definition;
data-specific errors do not block saving unrelated item fields.

Proposed deterministic semantics:

- Parentheses, unary signs, then multiply/divide, add/subtract, concatenation,
  comparison. Chained comparisons are rejected; use explicit IF expressions.
- Double-quoted strings with escaped quotes/backslashes; finite decimal numbers;
  boolean literals; no implicit string-to-number coercion. Concatenation uses
  locale-independent output and ISO dates. Function names ignore case.
- `IF` takes three arguments and a Boolean condition; only its selected branch
  executes. `ROUND(number, digits=0)` rounds half away from zero; ABS takes one
  number. UPPER/LOWER/LEN take one text value; LEN counts Unicode code points.
  CONCAT takes one or more scalar arguments. NOW/TODAY take none.
- NOW uses a single UTC evaluation instant. TODAY uses its UTC calendar date.
  Date-only arithmetic uses whole calendar days; date plus fractional days is
  rejected. NOW preserves time; subtraction returns elapsed days.
- Formula-to-formula references and arbitrary execution are unsupported. Field
  suggestions contain only eligible fields attached to the type. A reusable
  formula must validate against every type to which it is attached.
- Proposed built-in references are `name`, `priority`, `estimate_point`,
  `start_date`, and `due_date`. Priority is normalized to uppercase text; missing
  dates/estimates are empty. [W] examples demonstrate built-ins but do not provide
  an exhaustive catalog. These five are the proposed initial allowlist.
- Server results are authoritative. The same values/time/configuration must give
  the same preview and persisted-item display. Read-only values cannot be patched.
- Time-sensitive formulas refresh on access and at a visible clock boundary;
  cached results carry their evaluation time. Filter/page operations evaluate
  all rows against one instant. No per-second activity spam for clock changes.

Numeric bounds, resource limits, and exact timestamp presentation remain technical
decisions; the evaluator must be bounded and report limits as validation errors.
The UI must not expose function signatures or coercions that differ from execution.

### Hierarchy rules

Configured hierarchy requires `parent.level = child.level + 1`; a zero-level type
is a leaf. Types may share a level. Cross-project parentage is allowed within the
same workspace when both items are accessible. A type change must satisfy both
the item's parent and all its children. [W], [I]

Vector decisions: before the first saved hierarchy configuration, activation
opens a draft builder but does not sever existing links. The first Save performs
the same impact review as subsequent saves. Unassigned types resolve to zero;
empty nonzero levels are removed and numbering stays contiguous, with Level 0
retained. Renumbering carries the default marker with its level; removing the
default level returns the marker to Level 0. These are draft changes until saved.
Parent cycles and self-parenting remain invalid even when hierarchy is unconfigured.

The documented level menu includes Set as default but does not define its effect.
Vector's production decision: it suggests the level for new standalone work items.
An explicit available type may create a standalone item at any level. For a child,
only the immediately lower level is eligible, including cross-project parents;
a Level 0 parent cannot have children. Explicit choices are never replaced to
make a save succeed.

For an unselected new item, choose among active, project-available eligible types:
prefer the project's saved default, then protected Task, then a sole remaining
candidate. If none or multiple remain, keep the selection empty and ask the user
to select a type. The schema response must remain usable in this state; published
creation returns a field error until a type is selected. Unconfigured hierarchy
does not restrict suggested candidates by level. Projectless/incomplete private
drafts may remain untyped; publication requires a valid selection. An untyped
private draft may retain any same-workspace, readable parent as unfinished intent,
including a leaf. Only type-level adjacency is deferred until a type is chosen or
the draft is published; visibility, workspace and cycle rules still apply.
Hierarchy reviews retain these unfinished links and include them in review
fingerprints. Changing a parent type does not silently type or unlink such drafts.

The project preference is distinct from protected Task and the hierarchy's default
level. Pause, deactivation, or removal of an import preserves the preference but
makes it ineligible until restored/reimported. Fallback suggestions never overwrite
the preference. Deleting the type clears the preference. Legacy conflicting
preferences require an explicit choice, never an arbitrary first row.

Automatic suggestions apply only to new items. Changing their project or parent
keeps an existing suggestion selected and validates it in the new context; the user
can choose another type. Existing items and saved drafts never change type through
a refresh, including a deliberately cleared draft type. Parent omission retains
the canonical relationship, while explicit clearing removes it. Unchanged hidden
parents remain usable for internal validation without exposing their identity.
The server and form use the same resolver and revalidate current configuration on
save. These decisions require the creation-default implementation checkpoint below;
the parent-selection checkpoint alone does not implement them.

### Preservation and destructive changes

For type switches, used property detachment, preserving deletion, context fallback,
and cascading removal, show a review before committing. Preserved content includes
the property label and human-readable former value, not only raw IDs. Rich text
retains readable content and permitted links; unsafe markup is never reintroduced.
Append to existing description without overwriting it. Record the reason in
activity. Shared values survive a type switch without duplication.

Property deletion offers preserve-to-description by default or explicit permanent
value removal. Show affected work-item, type, and project counts. [W] Deleting a
definition does not delete its work items. Vector decision: include archived items
and private drafts in impact accounting without exposing private draft content.
Draft preservation stays inside each private draft. Vector retention decision:
“Remove current values without preserving” removes current values and their
configuration, but retains existing versions and audit records. It does not create
new copies of the discarded values. The dialog explicitly states that history
remains available; historical purge is a separate operation and is not offered here.

Every operation is all-or-nothing from the user's perspective. A retry cannot
append the same preserved block twice. If configuration or item versions change
after preview, require a refreshed preview. Cancellation or failure preserves the
saved item/configuration and the user's editable draft.

### Permissions and visibility

| Actor / capability      | Allowed scope                                                                     |
| ----------------------- | --------------------------------------------------------------------------------- |
| Workspace administrator | Enable and manage library, properties, contexts, rollout, hierarchy               |
| Project administrator   | Import and manage optional project availability; no workspace-definition mutation |
| Item creator/editor     | Select allowed types and edit applicable values within existing item permissions  |
| Item viewer             | Read type and visible values; no configuration or value mutation                  |
| Draft owner             | Read/edit private drafts; current project permissions required on publication     |

Use Vector's existing role/access checks. Workspace configuration access must not
accidentally grant access to private item content in impact previews. Catalogs
returned to an item editor contain only the information needed for that scope.
Cross-workspace type/property/context/member/release identifiers are rejected.

The permissions matrix and project-type article disagree about contributors
managing project definitions. Workspace governance from [W] is the controlling
model here; do not infer workspace-admin rights from contributor permissions. [P]

Vector decision: selecting a workspace member never grants project access.
Ineligible members are disabled with a reason; automatic subscription is only
created where existing item access permits it. Removing a field selection does
not silently unsubscribe a person who may have subscribed for another reason.

### Filtering and projection semantics

Vector decisions: a property not attached/applicable is **not applicable**, distinct
from an applicable field with an empty value. Custom-property filters operate on
applicable items only, including negative predicates. Empty matches applicable
missing values. Non-empty comparison predicates exclude empty/error values.
Multi-select inclusion means any intersection; exclusion means no intersection.
Ranges include both endpoints; Text and Formula Text matching use Unicode case
folding. URL comparisons retain literal case, since paths can be case-sensitive.
Date filters compare UTC calendar days, including timestamp-valued formulas. Read-only Text
filters use its configured content. Formula errors are visibly marked and do not
compare as values. Applicability is evaluated per predicate: an unknown custom
value can coexist with a decisive branch (`unknown OR true` is true;
`NOT(unknown AND false)` is true). Only definite true results match; direct
negation never turns unknown into a match. A saved filter referencing a deleted definition displays an
invalid-filter warning and requires repair, rather than broadening results.

Type/field values must agree across modal, detail, list, board, table, filter counts,
saved views, and authorized exports. Search includes sanitized rich-text content
under item permissions. Public project projections must not start exposing custom
fields implicitly; external/public visibility requires its own explicit contract.

Search decision: authenticated workspace, entity, and project relationship search
match case-insensitive literal substrings in sanitized **current** rich-text field
values. `%`, `_`, and backslashes are literal input. A result may include one
matching field's ID, current title, and up to 240 characters of plain text around
the match. The UI renders this as text beneath the item title. Historical revisions,
detached fields, inactive effective contexts, and disabled rollout values do not
participate. An inactive item type alone does not hide an existing applicable value.
The existing API-token v1 search contract remains title/identifier-only; public
project search and projections do not gain custom-property content.

## Existing Vector integration boundaries

The current settings catalog has no workspace feature entries, and the existing
`IssueTypeSwitcher` only renders the item identifier. These are reusable entry
points, not a hidden complete implementation. An earlier backend draft exists
locally but is unintegrated and provisional; it does not determine this UI design.

Source owners inspected for this specification:

- `apps/web/routes.ts`: authored route tree; generated route output is not edited.
- `packages/constants/src/settings/workspace.ts` and
  `packages/constants/src/settings/project.ts`: settings destinations.
- `apps/web/core/components/issues/issue-type-switcher.tsx`: current switch stub.
- `apps/web/core/components/issues/issue-modal` and
  `apps/web/core/components/issues/peek-overview/properties.tsx`: existing editing
  and property presentation boundaries.
- `apps/go-api/internal/workitems`: existing work-item mutation/projection owner.
- [Architecture](architecture.md): Query owns server data, Form owns editable
  drafts, generated transports derive from source designs, Go owns authorization
  and durable effects.

For the prototype, an explicit preview-state provider supplies screen-shaped
fixtures. It must be replaceable without changing the visible interaction
contract. Avoid making a fake backend API or selecting storage layout to drive
the first UI. After UI comparison, the final contract must be shared by normal
creation, inline edits, type switching, draft publication, duplication, moves,
imports, bulk parent changes, and existing API writers. Reject unsupported
metadata rather than silently dropping it. No surface can bypass hierarchy or
required-value validation.

Existing items, IDs, state, dates, memberships, parent links, and descriptions
remain unchanged merely by visiting the prototype. Enabling the eventual feature
must preserve existing types/properties and their identities; conflicting legacy
names require an explicit resolution preview, not silent merging.

## Decisions still requiring resolution

These gaps do not block placeholder screens. They do block claiming complete
production parity. Proposed controls and states must stay visible during review.

| Question                          | Evidence / uncertainty                                                                                     | Prototype and proposed direction                                                                                                                                                                                                                                            |
| --------------------------------- | ---------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Default type versus default level | [T] supports a project default; [W] protects Task and exposes a default-level action without its semantics | Resolved above: explicit choice; otherwise eligible project preference, Task, or sole candidate. Ambiguous/empty remains unselected; saved items never auto-retype                                                                                                          |
| Workspace type deletion           | [W] requires reassignment; [T] describes retained deleted-type labels                                      | Use explicit replacement/impact UI and workspace rules; do not adopt project tombstone behavior accidentally                                                                                                                                                                |
| Context override surface          | [W] gives examples, not a complete per-kind override matrix                                                | Use the explicit allowlist above; mark other controls inherited/locked                                                                                                                                                                                                      |
| Context conflicts                 | No documented tie rule                                                                                     | Reject overlapping equal-specificity contexts; show conflicting scopes before save                                                                                                                                                                                          |
| Formula precision and time        | No complete grammar, time zone, refresh, rounding, or coercion specification                               | Use the proposed deterministic rules above; verify fixture results before persistence design                                                                                                                                                                                |
| Release catalog integration       | Release picker is included, but a complete releases subsystem was not found in the inspected Vector paths  | Use a real project catalog: members read; project admins manage names, descriptions and optional target dates. Schema revisions and locks protect changes; retained references block deletion. Stable creation IDs support retries. Full release lifecycle remains separate |
| Rich-text version UX              | [W] promises versioning/search but does not show a custom-field history screen                             | Reuse detail-history conventions with field-specific history and restore; prove access isolation                                                                                                                                                                            |
| Permanent deletion and history    | [W] describes deleting values but does not specify retention of historical versions and audit payloads     | Retain existing versions/audit records, create no new copies of removed values, and explicitly disclose that current-value removal does not purge history                                                                                                                   |
| Used-property exceptions          | Generic structural lock conflicts with date/cascading edit instructions                                    | Explicit date-presentation and cascading-migration exceptions; ordinary dropdown structure locked                                                                                                                                                                           |
| Workspace member eligibility      | [W] says workspace members; [T] says project members                                                       | Workspace model lists workspace members; existing item access limits subscription; show unavailable recipients clearly                                                                                                                                                      |
| Custom group/filter coverage      | [F] lists common kinds, not Formula/Release/Cascading exhaustively                                         | Show proposed matrix above and label unsupported grouping; no fake controls                                                                                                                                                                                                 |

## Acceptance and proof obligations

### UI completeness

Every S1–S11 destination and dialog is reachable in the prototype. Each has populated,
empty, loading, error, and restricted-access states where meaningful. Every field
kind has both its configuration editor and work-item renderer. Formula, context,
cascading, and hierarchy screens have their intended product interactions, not explanatory
text standing in for controls. R1–R8 each have a matching comparison view.

The UI review must establish that the information hierarchy and essential controls
match the references; unscreenshoted screens are explicitly reviewed as Vector
proposals. Narrow viewport and keyboard operation must preserve the same actions.

### Behavior completeness

| Capability         | Default / supported behavior                                        | Must reject                                                       | Required evidence                                                                 |
| ------------------ | ------------------------------------------------------------------- | ----------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| Activation/library | Protected Task; new optional types inactive                         | Disable activated feature, delete/deactivate Task, name collision | Lifecycle transitions and preserved existing items                                |
| Imports/rollout    | Shared identities; optional import; Mandatory reaches all projects  | Project removal of Mandatory; importing across workspace          | Definition edits propagate without copies; rollback retains assignments           |
| Attach/order       | Explicit attachments; new field appended                            | Duplicate attachment, normalized field collision                  | Same definition reused; order independently follows each type                     |
| Values             | Eleven kinds; effective defaults/required state                     | Wrong kind, invalid IDs, readonly writes, missing required value  | Configuration and item editor agree; errors retain input                          |
| Contexts           | Base fallback plus most-specific override                           | Ambiguous overlaps, cross-tenant scope                            | Same project/type resolves identically in preview, save, filter, and detail       |
| Formula            | Safe typed evaluation, lazy IF, empty/error states                  | Unknown/self/ineligible reference, arbitrary execution            | Every function/operator, Unicode, time boundary, null branch, division by zero    |
| Cascading          | Two–four ordered levels; per-context trees                          | Invalid parent path, duplicate sibling, draft field, reorder      | Required ancestry, leaf exception, ancestor reset, typed-confirm preservation     |
| Hierarchy          | Adjacent levels; level-zero leaf; cross-project within workspace    | Cycle, illegal type switch, invalid import/API parent             | Every writer enforces same relation; conflict save unlinks only confirmed pairs   |
| Type change        | Shared values retained; exclusive values preserved                  | Bulk switch, inaccessible/disabled target, hierarchy violation    | Confirmation describes exact result; retry does not duplicate description content |
| Delete/detach      | Preserve by default; impact review                                  | Stale impact, protected definition, hidden partial migration      | Counts match affected records; failures leave all prior values intact             |
| Drafts             | Private, incomplete allowed; validate on publish                    | Cascade on draft; unauthorized publication                        | No pre-publication notifications or private-content exposure                      |
| History/search     | Field-level changes and rich-text versions                          | Unsanitized content, unauthorized snippets/history                | Restores are new revisions; authorized results agree with saved values            |
| Layout/filter      | Stable identities, documented operators, explicit unsupported cases | Rich-text grouping/filtering, silent invalid saved filter         | Same matching IDs/counts across layout, saved view, reload, and API               |
| Concurrency        | Failed/stale saves retain editable input                            | Overwriting newer configuration/item versions                     | Impact preview revalidated; preservation and activity are atomic                  |
| Compatibility      | Existing standard properties and access behavior retained           | New accidental public exposure or destructive feature activation  | Existing work-item workflows remain contract-equivalent                           |

### Contract ownership for eventual integration

The authored UI/behavior contract is this specification. Runtime type/property
definitions belong to the workspace; project assignments select availability;
work-item values belong to the item and its access boundary. The same effective
configuration must be used by every projection and writer. Shared query caches
are scoped by workspace/project/item and cleared on sign-out. Form drafts remain
local until a successful mutation. Generated transport designs remain the API
source of truth; preview fixtures are never production authority.

Deferred: a separate project-owned configuration mode, full release management,
new public-field sharing policy, and custom grouping outside the explicit matrix.
Rejected rather than deferred: bulk type switching, arbitrary formula execution,
cross-workspace references, and unsafe/partial preservation operations.

## Research sources

Reviewed on 2026-10-24. The workspace and project type articles were read in full;
the linked articles below were reviewed for the relevant integration behavior.
Descriptions and screenshots can change independently, so unresolved details are
recorded above rather than treated as observed implementation facts.

- [W]: [Workspace work item types](https://docs.plane.so/work-items/workspace-work-item-types) — primary scope, properties, formulas, contexts, hierarchy, lifecycle.
- [T]: [Project work item types](https://docs.plane.so/work-items/project-work-item-types) — contrasting ownership/default rules and switch affordance.
- [B]: [Work item properties](https://docs.plane.so/work-items/work-item-properties) — built-in versus custom fields and editing surfaces.
- [I]: [Work item details](https://docs.plane.so/work-items/work-item-details) — property placement, peek modes, cross-project parentage, history.
- [F]: [Work item filters](https://docs.plane.so/core-concepts/issues/visualise_filter) — custom-property operator matrix and saved views.
- [V]: [Display options](https://docs.plane.so/core-concepts/issues/display-options) — type display/grouping and parent-type grouping.
- [D]: [Draft work items](https://docs.plane.so/core-concepts/drafts) — privacy, saving, and publication.
- [R]: [Project releases](https://docs.plane.so/projects/project-releases) — custom reference versus built-in release membership.
- [WR]: [Workspace releases](https://docs.plane.so/releases) — distinction from project release ownership.
- [P]: [Permissions matrix](https://docs.plane.so/roles-and-permissions/permissions-matrix) — administrative and item-level access distinctions.
- [S]: [Subscribers](https://docs.plane.so/communication-and-collaboration/subscribers) — subscription access and independent unsubscribe behavior.

[W]: https://docs.plane.so/work-items/workspace-work-item-types
[T]: https://docs.plane.so/work-items/project-work-item-types
[B]: https://docs.plane.so/work-items/work-item-properties
[I]: https://docs.plane.so/work-items/work-item-details
[F]: https://docs.plane.so/core-concepts/issues/visualise_filter
[V]: https://docs.plane.so/core-concepts/issues/display-options
[D]: https://docs.plane.so/core-concepts/drafts
[R]: https://docs.plane.so/projects/project-releases
[WR]: https://docs.plane.so/releases
[P]: https://docs.plane.so/roles-and-permissions/permissions-matrix
[S]: https://docs.plane.so/communication-and-collaboration/subscribers
