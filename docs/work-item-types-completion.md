# Work item types completion

Goal: implement and verify every in-scope requirement in
[the feature specification](work-item-types-spec.md). A passing intermediate
checkpoint does not complete this goal. Prototype screens remain available for
comparison, but fixture behavior does not count as production integration.

## Requirement tracker

- [x] Workspace activation, protected Task, type creation/editing and rollout.
- [x] Project imports, pause/resume/removal and availability enforcement.
- [x] Property definitions, attachments/order, all eleven configuration editors.
- [x] Context resolution, scope validation and explicit override inheritance.
- [x] Formula parsing, type checking, evaluation and authoring preview.
- [ ] Formula results throughout item detail and discovery surfaces.
- [x] Cascading configuration, path validation and confirmed preservation.
- [x] Type selection and custom fields in full creation/editing forms.
- [x] Type selection and custom fields in inline detail and peeks.
- [x] Custom values through ordinary writers and UI duplication.
- [ ] Audit remaining import/move and secondary writer integrations.
- [x] Private draft values and atomic publication with required-field validation.
- [ ] Hierarchy persistence, previews, cross-project parentage and all writers.
- [x] Project default preference and hierarchy-aware creation defaults.
- [x] Used-type deletion with valid reassignment/preservation.
- [x] Type switching from full detail and peek entry points.
- [x] Property detach/delete/context changes with revision-checked impact review.
- [x] Member eligibility and subscriptions.
- [x] Real project release catalog and management.
- [x] Custom-value activity and rich-text history/restore.
- [x] Rich-text search and search snippets.
- [x] Independent type display in list, board and table, using a shared scoped catalog.
- [x] Text, number and numeric-range filter controls.
- [x] Authorized effective-property catalog and typed filters on project/workspace/profile/archive pages.
- [x] Authorized type catalogs and filter integration across project/workspace/module/cycle lists and counts.
- [ ] Type grouping/subgrouping and parent-type grouping.
- [x] Custom filters on module/cycle/view endpoints and filtered counts.
- [ ] Custom categorical grouping and grouped counts.
- [x] Custom table columns, including formatted values and separate cascading levels.
- [ ] Formula results throughout the remaining discovery/export surfaces.
- [x] Saved-view round trips and repair without silently broadening results.
- [ ] Explicit authorized custom exports; unchanged public-field visibility.
- [ ] Keyboard/narrow viewport checks, full gates and final independent reviews.

## Review checkpoints

1. Workspace/project production slice: independent backend/UI reviews completed;
   findings fixed and rechecked. Full combined quality gate passed.
2. Completeness review: ordinary writers discard custom values; draft publication
   is not atomic; the original trigger lacks kind validation and restricts parents
   to the same project; workspace mutation completion can repopulate a cleared
   session cache. These are open integration obligations, not acceptable exclusions.
3. Formula engine: focused tests and race checks pass. Fresh review found decimal
   rounding and empty/error precedence defects; fixes and regressions were rechecked.
   Production authoring and item projection have not been connected yet.
4. Property contract: all eleven value kinds, context precedence and cascading
   structure have focused tests. Fresh review found image-only rich-text loss,
   reorderable levels, unreferenceable internal names and unbounded scope work;
   these were fixed. The scope-limit regression uses valid unique IDs on both sides
   of the boundary. Transactional reference checks and persistence remain next.
5. Session cache: workspace saves now invalidate existing scoped queries instead
   of restoring a response after logout. A fresh reviewer verified the successful
   save path and confirmed the regression fails with the previous implementation.

6. Property persistence: definitions, contexts, formula dependency validation,
   attachment order and unused removal now use authenticated, revision-checked
   transactions. The workspace editor reads one coherent catalog snapshot. It no
   longer uses fixture properties, projects or members. Ever-used locks survive
   clearing values and deleting private drafts; a concurrent deleted-item write
   cannot race a structural property edit. Fresh backend review findings were fixed
   and rechecked. Repository integration tests pass with race detection (80.9%
   statement coverage, above the unchanged 74.83% package floor).
7. Editor persistence: create/edit/attach/context/cascade saves await the server;
   errors retain the native Form draft. Fresh review found invisible context errors,
   invalid defaults serialized as null, missing cascading defaults after kind changes,
   and independently loaded catalog revisions. All four were fixed and independently
   rechecked. The 21 focused frontend tests pass. Project/workspace schema mutations
   share cache invalidation without repopulating cleared sessions.
8. Draft publication: owner checks and current project access now apply to edits,
   deletion and publication. Publication merges saved fields and relationships with
   explicit overrides, validates current selections, and commits the item, links,
   assets, consumed-draft identity and queued effects together. Retries reuse the
   original item even after draft cleanup. Concurrent publication, ordinary creation
   and draft edits serialize without duplication or resurrection. The concurrency
   regression passed five consecutive race-enabled runs.
9. Durable effects: a database outbox feeds the existing native workflows with
   stable identities. A lifecycle-owned worker resumes pending rows after restart;
   failed rows do not starve other work. Admission acknowledgement failures reuse
   the original identity. Fresh reviews corrected form normalization, retry response
   fields, lock order, worker lifecycle and unsupported-runtime startup behavior.
   Both deployment Compose files already enable the required native runtime.

10. Item values: private draft and ordinary item writers now merge submitted
    custom-property keys into locked saved values. Creation defaults respect
    explicit clears; zero and false survive round trips. Publication validates
    required fields, type availability and project-scoped references. Inactive
    values are retained. Invalid changes roll back item data, relationships,
    subscriptions and effects. Member choices never grant access, and eligibility
    uses the resulting creator when an API write changes authorship. An unrelated
    edit does not recreate a manually removed subscription.
11. Read boundaries: authenticated compact/detail/API projections return type and
    custom values and honor selected fields. Public serializers retain their field
    allowlist. Draft detail authorization now checks the owner's draft, fixing its
    old published-item table lookup. API guest reads and sub-item reads enforce
    actual item visibility; sub-item results and counts exclude inaccessible child
    projects. Three fresh integration/privacy reviews found defects, which were
    fixed and rechecked. Work-item-types integration tests pass under the race
    detector with 80.3% coverage, above the unchanged 74.83% floor.
12. The new webhook outbox admission makes ordinary creation/editing atomic with
    its durable follow-up work. A late-effect failure regression verifies rollback
    of items, sequences, subscriptions and queued effects. Go lint now has 1,355
    existing diagnostics: the obsolete constant-task-name warning was removed
    after the shared task publisher began handling both activity and webhook jobs.

The combined run passed every underlying Go, frontend production build and
TypeScript check on 2026-10-24 (`/tmp/vector-identity-full-clean.log`). Prek exited
1 because files changed during the hook; a stable-checkout exit-0 run remains
required. The corrections required no baseline increases.
Draft custom-value validation now runs on private saves and publication; the
ordinary frontend forms still need integration. Backend publication alone does
not complete the draft requirement above.

Destructive property edits currently reject when values require preservation; their
review/commit workflow is still outstanding. These safe rejections are temporary,
not a completed deletion/preservation feature. Ordinary authenticated create/update writers now validate and retain values;
frontend forms, duplication/import/move integrations and other writer paths remain
outstanding. The full feature remains incomplete.

Initial coverage floors for the two new packages were measured from focused tests:
formula 94.56% (331 statements), property 82.15% (297 statements). Existing package
floors were not changed. Focused Go lint reports zero issues; the five relevant
frontend tests and focused type-aware frontend lint pass. The complete gate will
run after these contracts are integrated; this checkpoint is not feature completion.

## Explicit implementation decisions

- Formula resources: 4,096 UTF-8 source bytes, 1,024 tokens, 32 parser nesting
  levels, finite IEEE-754 numbers and 65,536-byte text results. ROUND precision
  is an integer from -15 to 15. Date years are 1–9999. Failures are reported;
  no coercion or arbitrary execution is introduced.
- Empty operands propagate independently of operand order when evaluation succeeds.
  A runtime failure remains visible even beside an empty operand. IF evaluates only
  its selected branch; compilation validates references/types in both branches.
- ROUND applies half-away-from-zero to the number's shortest decimal representation
  so decimal ties such as 1.005 round to 1.01 at two places.
- The prototype's resolved UI contracts remain the implementation target. No
  additional warning banners or development controls belong in the product UI.

Unresolved production decisions in the specification will be recorded explicitly
as they are resolved; they must not become silently omitted capabilities.

### Checkpoint 13 — scoped item editor schema

- Added the generated, session-protected workspace `work-item-schema` read endpoint
  with explicit create/edit/draft/publish modes and presence-aware type selection.
  It returns ordered active properties with only their effective configuration;
  unrelated contexts and project catalogs are not returned.
- Private drafts retain owner access after membership removal. New target projects
  and publication require current project access. Inaccessible saved targets expose
  only generic labels for already selected releases and no membership eligibility.
- Member choices use the writer's access predicate with disabled reasons; unavailable
  stored members have generic labels. Draft schemas omit Cascading; publication
  schemas include it and check current type availability.
- Fresh independent review found and fixed draft-access disagreement, UUID casing
  in context resolution, and a differing-denial item-existence leak. Scope matching
  and overlap/duplicate validation now treat UUID case consistently.
- HTTP tests prove omitted `type_id` differs from `type_id=` and reject malformed
  IDs/modes before repository dispatch. Focused database regressions cover contexts,
  paused types, private drafts, guest access and reference privacy. Ordinary move
  destination previews remain part of the unfinished move integration.
- The direct effect-writer integration test restores workitems coverage to 18.7%,
  above the unchanged 18.56% floor. Its cleanup lint findings were fixed; no baseline
  was increased. A stable full quality-gate run is still required. This checkpoint
  does not claim the ordinary frontend forms or the complete feature are finished.

### Checkpoint 14 — read-only item formula projection and preview

The scoped item schema now computes Formula fields from the effective, attached
property definitions and one transaction timestamp. Estimates resolve through the
current project's estimate catalog instead of treating an estimate-point UUID as
a number. Missing/inactive values, zero, false, date-only values, malformed inputs,
and lazy conditional evaluation have focused regressions.

A session-authenticated, CSRF-protected preview endpoint applies unsaved editor
inputs inside the same read-only authorized snapshot. It uses saving's title
normalization and property validation. Preview does not update an item, create
activity, or add subscriptions. Revoked-project draft access suppresses restricted
estimate/release data. Scoped reference choices retain unavailable saved selections
without disclosing foreign names.

Fresh reviews covered the adapter and endpoint independently. Findings about
estimate resolution, default priority, normalization, and access checks were fixed.
The full Go quality gate passed, including lint, dead code, duplicate checks,
integration tests, and the 140-package coverage ratchet. No baseline increased.

### Checkpoint 15 — ordinary editor integration (in progress)

The ordinary create/update modal now loads the authorized item schema, places the
type selector beside the project selector, and renders its effective custom fields
in order. Values remain local when changing types; only applicable edits are sent.
Number typing retains incomplete decimals until submission, and explicit clears,
zero, and false remain distinct. Unchanged Member values are omitted from ordinary
updates so unrelated edits do not re-add a removed subscription.

Draft publication has a separate preparation step that displays Cascading fields.
The Save to Drafts exit path obtains a fresh draft schema before serialization,
retaining input even if the creation schema is reloading and excluding Cascading.
Formula previews use debounced unsaved inputs and refresh while visible. An
unchanged estimate is omitted so the server controls access to its saved value.

The shared searchable picker now exposes a closed-state Tab stop and returns focus
after keyboard selection. It preserves focus when dismissal follows an outside
click. Boolean controls forward their field accessibility attributes and refs.
Fresh review identified and verified these keyboard fixes.

Further fresh reviews found and fixed draft-exit serialization, rejected-save
focus timing, repeated focus during corrections, stale type hydration, and an
untouched-field overwrite during source refresh. Modal-source queries now include
user identity and fetch custom values alongside descriptions. Dirty custom values
use a stable comparison baseline; refreshed untouched type values are hydrated.
Focused frontend regressions cover these paths. The existing modal complexity
ratchet was explicitly tightened from 37 to 36 after extracting its action footer;
no allowance or source exclusion increased. Cascading input rendering and the form
test harness are shared to avoid new duplicate fragments.

Still incomplete: projectless draft UI, changing a saved draft's target project,
full-detail/peek editing, preservation confirmations, custom-field asset/history
and search integration, all remaining writers, hierarchy, and filters/layouts.
The new backend must be rebuilt into the local stack before browser verification.
This checkpoint does not mark S9 or the complete feature finished.

### Checkpoint 16 — editor lifecycle and schema transport

The real schema service now sends its project, mode, draft and type inputs through
Axios query parameters and forwards cancellation. A real Axios-adapter regression
checks serialization, including explicit type clearing.

Transient submission locks use the parent inert fieldset without changing the
editor's editable configuration. A regression mounts the ordinary form with the
real TipTap lifecycle: a rejected save retains the editor, undo history and value,
then focuses the field with its validation error. Fresh review found the original
editor recreation and verified the correction. Focused frontend checks passed.

### Checkpoint 17 — private draft assets and publication

Drafts can associate uploaded assets through a session-authenticated, CSRF-protected
workspace endpoint. Association validates the entire set and locks the owned draft
before the assets. Upload creation, confirmation and publication use that same
lock order, including legacy draft uploads. Publication cannot be followed by a
late attachment or upload confirmation that changes the published item.

Draft ownership is checked across workspace/project reads, duplication and legacy
aliases. Public and ordinary attachment routes exclude private draft assets,
including older rows whose entity kind disagrees with their draft binding. Legacy
creation rejects conflicting bindings and verifies the owner, workspace and active
draft under lock. Publication transfers only completed, active assets belonging to
the draft owner and workspace, without another entity binding.

Unfinished uploads referenced by submitted description/custom-field content block
publication without consuming the draft. Abandoned pending uploads do not block a
successful retry and remain private. Real database tests cover these cases, revoked
membership, guest ownership, mixed-owner atomic rejection, injected legacy bindings,
and concurrent publication. HTTP tests cover session/CSRF, body validation and
authorization. Fresh independent reviews identified the alias and legacy-writer
defects; fixes were reviewed again with no remaining blocking findings.

The full TypeScript gate passed at this checkpoint. Go formatting, build, vet,
lint, dead-code and duplicate checks passed; the integration run exposed a stale
route-count assertion, now corrected and verified in a focused transport run.
The final combined gate is pending. Three obsolete lint allowances were removed;
no baseline was increased. S9 frontend projectless drafts and publication orchestration
remain next; the overall spec is still incomplete.

The combined checkpoint passed frontend production builds and the full TypeScript
gate. All 9,050 Go tests passed, but workitems coverage was 18.52% against its
unchanged 18.56% floor. Fresh review of the added database regression found that
literal markup in a plain Text property could be mistaken for a pending upload.
Publication now inspects image elements in the description and schema-identified
Rich text properties only. Focused public-publication and database-boundary tests
pass, including retained drafts on failure, literal text preservation, abandoned
uploads and owner/workspace/draft isolation. The corrected combined run passed all
three hooks (Go, frontend builds and TypeScript); the coverage floor is unchanged.
Evidence: `/tmp/vector-work-item-types-combined-corrected.log`. The single local
development stack was restarted and is running; startup evidence is in
`/tmp/vector-work-item-types-dev.log`. This verifies the backend checkpoint, not S9
as a whole or the full feature.

### Fresh completeness review — remaining production work

The independent audit confirms the unchecked requirements above remain open.
After S9, prioritize authoritative preservation preview/commit (production impact
dialogs currently receive empty item lists), used-type reassignment and switching,
detail/peek fields and field history/restore, then durable hierarchy with
cross-project parentage and shared filter/projection/layout support. The current
hierarchy settings route still uses fixture state. These are implementation gaps,
not completed features.

Configuration gaps include formula field insertion/signatures/context/sample
preview, context default pickers and effective-resolution previews, cascading
inheritance and destructive-change review, property/option icon editing, drag
ordering, project-default precedence and a real release catalog provider. S9
frontend review additionally requires reference-aware upload finalization, retained
identical publication retries after ambiguous responses, adjacent standard-field
errors, explicit null clears, canonical draft reloads and count reconciliation.

### Checkpoint 18 — draft frontend and publication recovery

Projectless drafts now render, save incomplete titles, reopen authoritative fields,
and retain saved selections when changing destination. Standard field errors stay
beside their controls. Publication keeps an immutable request and native mutation
receipt across modal remounts; uncertain retries bypass consumed-draft/schema reads
and never repeat accepted publication. Creation retries reuse their original input.
Account changes stop subsequent writes and completion UI effects. Accepted draft
mutations refresh the canonical membership count through native permission queries.

Copied private images are marked for duplication in descriptions and rich-text
properties. Pending/failed copies block submission. Replacing a failed copy cannot
delete its source; late duplication results cannot overwrite replacement uploads.
Fresh reviews found these races and verified their fixes. Twelve mounted editor
regressions and the editor typecheck pass; the expanded form/source suites pass
36 tests, the publication suite passed six, and pagination/query tests pass eight.

A real browser check found that the older database draft trigger contradicted API
validation for optional types without a project. Migration 000008 aligns the guard
while retaining active-type, tenant and destination-import restrictions. Both new
integration regressions pass; a fresh independent migration review found no issues.
The migration is applied locally. Browser checks saved/reloaded an untitled draft
without a project, then successfully changed its type to Bug. The verification
record is `1221a77c-2090-4e0a-bd58-9fd23ad74856` in the local test account.

The current whole-web typecheck remains blocked by concurrent permission migration
edits outside this slice; the prior combined exit-0 gate predates this checkpoint.
No baseline increased. The development stack remains running. Preservation,
hierarchy, full detail/history/search and filters/layouts are still incomplete.

### Checkpoint 19 — property removal review and writer transactions

Production property detach/delete now has an actor/workspace-bound review receipt,
configuration revision and affected-row fingerprint. Preview exposes aggregate
counts only, including archived/trashed items and private drafts. Commit rechecks
impact, atomically changes schema/values/descriptions/activity, and replays committed
receipts without duplicating preserved content. Oversized descriptions are rejected
before mutation; paragraph line breaks and former selection labels are preserved.
The retention decision in the spec now explicitly removes current values while
retaining existing history, without creating new copies of discarded values.

Fresh reviews found stale full-row intake/archive writes, non-transactional cleanup,
an intake edit-plus-accept overwrite, and reversed worker/subissue lock ordering.
Intake/archive/public-intake/subissue writers now use schema-first transactions;
activity and cleanup use the existing outbox. Activity and automation workers acquire
schema locks before item rows; automation revalidates candidates under those locks.
Focused database tests pass for intake interleaving, edit-plus-accept and cleanup
rollback, plus preservation counts, retry, stale impact, size and history retention.
An injected late-failure regression also passes: schema, item values, descriptions,
activity and the receipt all roll back together, and the same review can retry.

The production dialog uses server counts, requires exact confirmation for current-
value removal, and retains the original receipt after an ambiguous response. Fresh
UI review caught a Refresh-review path that discarded that receipt; the fix also
moves preview queries outside blanket catalogue invalidation. The expanded focused frontend run passes all 49 tests, with 23 passing again after
the final error-fallback and lint cleanup. The whole-web typecheck passes. Fourteen
editor regressions pass after the final readiness change. Obsolete permission fixture
remnants were removed; native membership invalidation remains covered.

Migration 000009 and the new API are active locally. A real browser check attached
a temporary text property to Bug, saved a value in the projectless private draft,
and detached the property. The preview reported one item, one type and zero projects
without exposing private content. Reopening the draft confirmed the original
description and appended human-readable value, with the detached editor absent.
Evidence: `/tmp/vector-property-preservation-preview.png` and
`/tmp/vector-property-preservation-result.png`. The property remains unattached in
the test library. The full combined gate subsequently passed, as recorded below.
The owned modal/editor lint findings are resolved without baseline increases.
Estimate-point deletion now participates in schema locking and transactional effects;
three database regressions cover preservation races, rollback and immediate clearing
without a replacement. Workitems integration tests pass at 19.0% coverage, above
the unchanged 18.56% floor. Both affected background worker integration tests pass
against disposable PostgreSQL and Redis. A fresh independent worker review found
no further locking, cursor, revalidation or effect-replay defects. Evidence:
`/tmp/vector-schema-worker-tests.log`. The final fixture cleanup passes 23 frontend
tests. Go fast checks found cleanup-error handling and a redundant constant
argument; these are fixed. Fresh review found no issues in the consolidated writer
wrappers. The errcheck allowance for the shared project-configuration writer was
tightened from eight to seven. The combined Go run passed 9,085 tests but failed
two package coverage floors. Direct preservation and estimate-deletion regressions
now pass at 84.3% and 25.5% respectively, above the unchanged floors. Fresh review
strengthened the foreign-project fixture and exact transactional effect assertions;
the final focused rerun passes. The corrected frozen snapshot passes all three full
Prek hooks (Go, frontend production builds and TypeScript), exit 0, recorded at
`/tmp/vector-schema-permissions-confirmed.log`. The development stack was restarted
and the authenticated drafts screen loaded successfully. This gate covers this
checkpoint and the concurrent native permission migration, not subsequent slices.
Context/cascading preservation, used-type reassignment, hierarchy, detail/history,
search, and filter/layout/export integration remain unfinished.

### Checkpoint 20 — context and cascading edits (in progress)

Context and cascading changes now use the reviewed, revision-bound schema writer.
The complete proposed definition is bound to the receipt. Only selections whose
catalog ownership or selected path becomes invalid are moved to descriptions;
renaming/reordering options and unrelated option changes retain valid selections.
Context fallback uses former labels. Schema, values, description, activity and
receipt roll back together, including an injected late failure, and retries append
preserved content once. Nullable project/type scopes include private projectless
drafts. Required-level changes apply on next edit, without retroactive preservation.

Native context/cascading forms retain their drafts when review is canceled. Used
selections require exact `Confirm`; updates preserve values and do not offer discard.
An outdated complete-definition edit cannot adopt a newer revision and overwrite
concurrent changes. Ambiguous commit retries retain the immutable reviewed input.
The confirmation includes former option paths from configuration, never private
item content, and labels conservative scope-change paths as potential candidates.

Context rows show scopes and overrides; All controls restore unbounded dimensions.
A project/type preview shows the winning context, effective settings and inheritance.
Default controls use numeric inputs and named member/option choices. Inherited
cascading options remain read-only until explicitly overridden; switching inheritance
clears obsolete branch navigation. Descendant removal is independent of array order.

Fresh independent reviews covered catalog identity, the preservation predicate,
reviewed schema writer, editor lifecycle and S4–S6 completion. All reported findings
are resolved. Actual repository checks pass: six targeted database regressions,
31 expanded mounted/pure frontend tests, 12 final review regressions, whole-web
TypeScript, Go fast checks and full type-aware lint without a diagnostic increase.
The Go full run exposed catalog-slice aliasing during removal dry runs. Cloning
properties and attachments before mutation fixed the rollback leak. Existing
preservation regressions and a new reviewed-detachment regression pass; the full
work-item-types package passes at 80.4% coverage above its unchanged 80% floor.
Evidence: `/tmp/vector-context-preservation-rerun.log`,
`/tmp/vector-context-detach.log`, `/tmp/vector-context-workitemtypes-full.log`.
The previous combined full gate predates this slice. No quality floor or baseline
was increased.

The rebuilt API is active while the frontend remains running. Browser verification
created the unattached `Cascade verification` property, saved `Platform → API`
options and a `Platform scope` context, then reopened it and verified scope and
base inheritance. Screenshot: `/tmp/vector-context-effective-settings.png`.
This verification configuration remains unattached, so ordinary work items are
unaffected. Formula authoring, used-type reassignment, hierarchy, detail/history,
search and discovery integration remain unfinished.

### Checkpoint 21 — formula authoring

The property editor now provides stable field insertion, eligible references across
all attached types, typed examples, function/operator help, and an explicit Evaluate
action. The read-only authoring endpoint compiles the unsaved full definition with
the writer's catalog validation. It evaluates either creation defaults or an
accessible existing item, showing validation errors separately from runtime errors,
empty results, inactive contexts and actual values. Responses show the evaluation
time and effective contexts. Editing an input hides and cancels obsolete results;
project/type changes clear the sample.

Three independent reviewers checked authoring UI, backend and end-to-end contracts.
Findings fixed: duplicate internal names, unattached formula dependencies, missing
search type IDs, identical-token caret replacement, and type filtering after the
search limit. Search now applies the selected type before LIMIT. A database
regression covers an older matching Task behind 101 newer Bug matches.

Endpoint tests cover session/CSRF/origin protection, scope and definition propagation,
zero serialization, malformed/missing input, authorization/conflict mapping and
private server-error redaction. Formula database tests cover authorized samples,
private drafts, revoked access, context/default semantics, stale revisions and
preview/save validation parity. Sixteen mounted frontend regressions pass across
authoring and reviewed context/cascade edits. Knip reports no increased findings.
The full Go gate and browser verification now pass, as recorded below. This does
not complete the full feature specification.

Browser verification evaluated `LEN({{name}})` against the saved Platform App
sample N86D3C863E2-338 and returned 57. The `Formula verification` definition
was then created and remained present after reload. It is unattached to any type.
Evidence: `/tmp/vector-formula-authoring-preview.png` and
`/tmp/vector-formula-authoring-saved.png`. The app and rebuilt API remain running.
Search filtering also breaks an inherited source-design clone; independent review
approved retiring its exact count-two baseline entry (505 → 503 directed pairs),
with no replacement allowance. Remaining web lint/clone findings at this point
belong to the concurrent roster migration, not this slice.

The final focused backend run passes all work-item-types packages: repository
81.2%, formula 94.6%, property 86.3%, above unchanged floors. The latest whole-web
typecheck also passes. Logs: `/tmp/vector-formula-workitemtypes-full.log` and
`/tmp/vector-formula-types-final.log`.

Next review boundary: hierarchy must become durable before used-type deletion can
rely on its final-graph validation. Fresh read-only review identified fixture-only
S8 state, missing cycle protection before activation, same-project-only parent
validation, and related-item expansion visibility as the required integration
points. A hierarchy migration needs revision-bound reviewed configuration, private
conflict accounting, atomic unlink/activity, and exact retry. Used-type reassignment
then requires destination validation without defaults or required-field bypass,
source attachment cleanup, and context cleanup that never turns a sole-type scope
into an all-types scope. These remain open requirements, not exclusions.

The final full Go gate passed with exit 0: 9,126 passing tests, 36 explicitly
allowlisted skips, and all 140 production package coverage floors unchanged.
Evidence: `/tmp/vector-formula-go-full.log`. All 56 work-item frontend regressions
pass at `/tmp/vector-formula-all-work-item-frontend.log`; whole-web types and owned
formatting pass. Log triage found reload cancellations and the already documented
React dependency warning, with no formula runtime failure. The app remains running.
The combined full Prek gate, including production frontend builds, is still due
for this slice after the concurrent roster changes reach a stable checkpoint.

The coordinated frozen snapshot subsequently passed the complete Prek gate with
exit 0: Go, frontend production builds and TypeScript full checks. Evidence:
`/tmp/vector-formula-roster-full.log`. The frontend watchers were stopped for the
gate and restarted afterward. This gate covers checkpoints 20/21 and the concurrent
roster changes; it predates the following hierarchy implementation.

### Checkpoint 22 — durable hierarchy (in progress)

The production hierarchy settings route now reads persisted types, stages level/default
changes locally, previews their exact impact, and commits with a revision-bound,
actor-bound receipt. Retry does not duplicate unlinking or activity. Database
constraints validate final parent graphs, including private/trashed items and
private drafts, cycles, cross-workspace links, and configured adjacency. Activation
is irreversible; previews and discarded drafts do not activate it.

Related-item projections now mask inaccessible parent IDs, expansions and child
counts in detail/list/drafts, modules, cycles, exports, public boards and history.
Canonical stored links remain intact. Three independent review boundaries covered
transactions/constraints, settings UI, and read privacy; reported defects were fixed.
The migration and API were deployed. Browser verification previewed the actual
workspace's 347 incompatible links, including eight restricted links, then discarded
the draft and left the hierarchy inactive. Evidence:
`/tmp/vector-hierarchy-impact-preview.png`.

Parent assignment and search now permit accessible cross-project parents within
the workspace. Search checks recursive cycle prevention, prospective child type,
hierarchy adjacency and visibility before pagination. Child selection additionally
requires permission to edit the candidate. Ordinary writes, drafts and bulk child
attachment reauthorize after schema-lock waits. Bulk attachment rejects unavailable
requested IDs atomically. Cross-project UI refreshes retain the parent's project
identity while writes use the child's project.

Fresh review found and fixed bulk workspace-membership/deleted-project omissions,
wrong-project UI refreshes, and a guest child-selection/write permission mismatch.
Focused race regressions pass for search, parent writes, private graph traversal,
queued revocation and bulk atomicity. Mounted UI regressions cover cross-project
select/unlink and search scoping. The preceding full Go run passed its tests but
failed the visibility helper package's coverage floor. Direct serializer/SQL tests
raise that package to 60.5%, above the unchanged 40.53% floor. Evidence:
`/tmp/vector-hierarchy-parent-review-regressions.log`,
`/tmp/vector-hierarchy-parent-refresh-test.log`,
`/tmp/vector-hierarchy-visibility-coverage.log`.

The full Go gate passed 9,177 tests across 475 packages and all 140 coverage floors
(`/tmp/vector-hierarchy-parent-go-full.log`). The combined full Prek gate passed all
three hooks (`/tmp/vector-roster-consumers-full-gate.log`) on the frozen hierarchy/roster
snapshot. The stack was rebuilt and restarted, and the hierarchy route returns 200.
Creation-default changes started only after this passing snapshot; they require
separate validation. Creation default precedence is the next implementation boundary. Used-type reassignment,
reviewed item type switching, detail/peek custom fields, custom-value activity and
rich-text versions, discovery filters/layouts, release catalog management, search
snippets and custom exports remain open. Durable private draft/custom-value writes,
atomic publication, contexts and formula authoring are implemented; their existence
does not imply those remaining integration surfaces are complete.

### Checkpoint 23 — creation defaults

A fresh independent review established the production precedence and lifecycle now
recorded in the specification. The UI needs a distinct project-default action and
an explicit no-selection state. The same resolver must drive schema discovery and
writes, with presence-aware parent inputs and no silent retyping of saved items.
A follow-up migration must remove both legacy Task-injection triggers; incomplete
private drafts stay valid while published items require an explicit/resolved type.
Project preference changes must reauthorize after waiting for the schema lock.

The first implementation review caught a selector recovery dead end and intake
creation paths that relied on database Task injection. The schema now keeps valid
replacement choices when the selected type is unavailable; intake and public
feedback creation use the same default/value resolver. Explicit null selections
remain distinct from omitted selections. Saved drafts never silently acquire a
new default. Unselected private drafts can retain unfinished parent intent, while
published writes enforce the hierarchy and an available type.

The backend migration and API are deployed. The project settings control and
creation schema hook are implemented; fresh frontend review caught and fixed
cached suggestions becoming sticky explicit choices when reopening a form.
Browser checks confirmed the Automatic project preference and Task selected in a
new Platform App item, with unavailable Bug disabled. The form was discarded.
Evidence: `/tmp/vector-project-default-selector.png`. Both fresh reviews closed.

The full Go gate passed, including all 140 unchanged coverage floors
(`/tmp/vector-creation-default-go-full.log`). Focused race, HTTP/transport and 22
mounted creation tests passed; whole-web typecheck passed. The combined gate passed Go and production builds; its TypeScript hook exposed two
outdated frontend fixtures. The fixtures now include the required eligible-type
list and default-selector export. All six affected tests pass, and the complete
TypeScript hook subsequently passed with production source unchanged. Evidence:
`/tmp/vector-picker-full-gate.log`, `/tmp/vector-full-gate-fixture-repairs.log`,
`/tmp/vector-picker-typescript-gate-recheck.log`. This is a combined exit-1 run
followed by an exit-0 TypeScript repair run, not a claimed exit-0 combined run.
The single frontend watcher was restarted and browser picker checks passed.

### Checkpoint 24 — reviewed type changes (in progress)

A fresh boundary review identified two prerequisite defects. Workspace schema
writers and property-removal reviews now reauthorize after lock waits and reject
workspace identity changes. Unused type deletion now cleans type attachments and
property context references, removes a context when its last explicit type is
deleted, and preserves shared definitions, other types' values, and history.
Independent fresh reviews of both fixes closed without remaining findings.
Focused race tests passed (`/tmp/vector-schema-cleanup-final-tests.log`).

The transition planner preserves old-only values and incompatible cascading
selections using their former configuration, appends them to the proposed
edited description, and then runs normal destination validation. It retains zero,
false and shared values. Fresh review found and fixed label loss for canonical
member/release slices. Planner race tests passed
(`/tmp/vector-item-transition-race.log`), and the full Go gate passed all 140
unchanged coverage floors (`/tmp/vector-type-transition-go-full.log`).
The planner is now connected to session/CSRF-protected item preview/commit endpoints.
The modal has not yet adopted them. The preceding picker checkpoint predates these
new endpoint and atomic-edit changes.

The reviewed item transaction binds the complete proposed edit, schema revision,
source state, resolved preservation and assets to an actor-bound receipt. A preview
executes the actual writer under a rolled-back savepoint. Commit revalidates the
snapshot and saves fields, cycle/module links, assets, rich-text preservation
versions and queued effects atomically. Exact retries return a freshly authorized
projection. Publication retry still works after draft consumption.

Fresh independent reviews caught and verified fixes for draft destination state
and relation scope, concurrent selection removal, unreviewed asset publication and
old project bindings on existing images. A regression also caught normalization
mutating the caller's map; it now copies inputs and preserves JSON number precision.
The asset snapshot includes existing bindings; publication transfers only final
inline references and retained attachments. Removed inline images stay private.

Seven database regressions plus the normalization regression pass with race
detection. HTTP tests cover session/CSRF/origin, zero/false/null, integers above
2^53, structured field errors and private error redaction. All three fresh review
boundaries closed without remaining findings. Applied-code verification passes:
`/tmp/vector-item-edit-applied-tests.log`, `/tmp/vector-item-edit-go-fast.log`.
The full Go gate is running at `/tmp/vector-item-edit-go-full.log`.

The modal now uses the reviewed transaction for saved edits, private draft saves
and publication. Preservation requires confirmation; cancelling retains the form.
Uncertain responses retain the exact receipt across reopening, and a server-proven
uncommitted conflict restores the editable form for a fresh review. Canonical saved
data stays separate from recovered form values, including parent changes and clears.
The current custom-value patch replaces stale recovered overrides, so undoing an
edit cannot silently reapply it. Accepted results reconcile the active list and
cycle/module membership without issuing another write.

Fresh frontend and backend reviews caught and verified fixes for recovery lifetime,
CSRF-time account changes, canonical response projection, list insertion/removal,
unsaved former-field edits, inaccessible source selection labels and required
cascading clears. All review rechecks closed without actionable findings. The 47
focused frontend tests pass, including mounted form recovery and schema retry;
focused database race tests pass. Evidence:
`/tmp/vector-reviewed-edit-canonical-frontend.log`,
`/tmp/vector-reviewed-transition-final-tests.log`.

The API was rebuilt and deployed. A browser check created a verification draft and
successfully updated it through the reviewed-save flow; evidence:
`/tmp/vector-reviewed-draft-save.png`. The full Go gate passed all 140 unchanged
coverage floors at `/tmp/vector-reviewed-edit-go-full-final.log`, before the next
type-removal batch. The TypeScript gate at
`/tmp/vector-reviewed-edit-typescript-full.log` failed on editor integration work,
static diagnostics and its frontend deadline. This is not a passing combined gate.
Subsequent hook/fixture cleanup passes 48 focused tests and direct web typechecking
(`/tmp/vector-reviewed-refactor-tests.log`, `/tmp/vector-reviewed-refactor-types.log`).
The modal complexity ratchet tightens from 22 to 21; no allowance increased.
The final lifecycle refactor review closed without actionable findings.

Still open: used-type replacement, remaining inline/detail/discovery/history/export
integrations, and final combined quality gates. This checkpoint does not complete
the entire spec. The next reviewed boundary is atomic used-type replacement:
aggregate-only impact accounting over published, archived, trashed and private
records; preservation inside each owning record; validation of required values and
the final hierarchy; exact-receipt retries; and type configuration cleanup only
after every affected record migrates successfully.

### Checkpoint 25 — used-type replacement

The backend now previews and commits replacement plus deletion atomically. It
counts published, archived, trashed and all private draft records, including
consumed drafts retained for recovery. Only aggregate counts leave the preview.
Internal preservation keeps readable former member/release labels without
granting the workspace administrator access to the owning items. Required values,
destination availability, final parent/child links and configuration cleanup are
validated before a receipt is created. Commit binds source values, preservation
and all graph neighbours; exact retries cannot repeat history or activity.

A fresh backend review caught missing hierarchy compaction. Deletion now removes
empty levels, carries the default with a surviving level or resets it to zero,
and rejects invalid final links without unlinking them. The recheck closed.
Focused migration, stale-review, authorization, historical/private preservation,
retry and hierarchy tests pass (`/tmp/vector-type-removal-hierarchy-tests.log`).
The session/CSRF-protected preview/commit endpoints are generated from source
designs; HTTP protection and safe-retry-marker tests pass
(`/tmp/vector-type-removal-http-tests.log`). Go fast gates pass
(`/tmp/vector-type-removal-go-fast.log`). The endpoints were subsequently deployed as recorded below.

The completed dialog and verification are recorded below. The complete feature
goal remains active.

Checkpoint 25 completion: the production replacement dialog now requires review,
shows aggregate impact and blocked-state guidance, and retains the exact uncertain
request across close/reopen. A safe server rejection allows a fresh review;
an acknowledged result is reconciled without another write. Independent backend
and frontend review findings were fixed and rechecked, including legacy unused-type
hierarchy compaction, stale peek/draft caches, and form defaults during recovery.

The API is deployed. Browser review and a local API check confirmed draft
reassignment, preserved description content, and idempotent retry; the isolated
verification draft was cleaned up. The latest 13 focused frontend tests, web
typecheck, type-aware lint, duplicate ratchet, Go fast gate and focused race tests
pass. Evidence: `/tmp/vector-type-removal-final-tests.log`,
`/tmp/vector-type-removal-final-types.log`,
`/tmp/vector-type-removal-final-lint.json`,
`/tmp/vector-type-removal-clones.log`,
`/tmp/vector-type-removal-final-go-fast.log`,
`/tmp/vector-unused-type-after.log`, `/tmp/vector-type-removal-review.png`.

The shared full Prek run exited 0 at `/tmp/vector-identity-full-gate.log`, covering
the preceding reviewed item-edit batch and initial deletion backend. It predates
the final unused-type hierarchy correction and deletion frontend; those changes
have the affected checks above, not a subsequent full-gate claim. Detail, discovery,
history, release catalog and export work remain unfinished.

### Checkpoint 26 — saved fields and formulas in detail and peeks

Full detail, side peek and modal peek share one read-only renderer for all eleven
field families. It preserves attachment order, option paths/icons, date formats,
reference labels, zero/false/empty distinctions, read-only text configuration and
formula empty/error states. Rich text uses the existing editor with editing
disabled. Calculated results display their evaluation time; clock formulas refresh
while visible, and accepted ordinary edits invalidate earlier optimistic reads.

The schema endpoint now returns scoped saved values, the server item timestamp
and computed formulas from one authorized repeatable-read snapshot. It excludes
inactive, unattached and formula storage values. The detail query captures account,
permission and query identities; a denied refresh or changed access hides cached
private fields. Effective-context and deterministic concurrent-write regressions
verify that field values, built-in formula inputs and timestamps cannot mix versions.

Fresh backend and frontend reviews closed without actionable findings. Thirteen
focused frontend tests, direct web typechecking, global type-aware lint, Go fast
checks, schema HTTP tests and focused database race tests pass. Browser checks
confirmed the same values in full/side/modal views, read-only rich text, and immediate
formula refresh after changing Priority. Evidence: `/tmp/vector-detail-final-tests2.log`,
`/tmp/vector-detail-types2.log`, `/tmp/vector-detail-global-lint2.log`,
`/tmp/vector-detail-go-fast2.log`, `/tmp/vector-detail-http-tests.log`,
`/tmp/vector-detail-snapshot-race.log`, `/tmp/vector-custom-fields-detail.png`.
The API is deployed. The isolated verification type/properties were removed and
its verification item moved to Trash.

This completes the saved-value display slice, not all S10. Next: accessible type
switch/update entry points and a canonical edit-source precondition. Fresh review
identified the form-open-to-preview stale-write gap and an incomplete modal source
projection; a server snapshot token and explicit conflict recovery must be implemented
before considering those entry points complete. Field activity/history/restore,
release catalog, discovery, exports and final full-spec verification remain open.

### Checkpoint 27 — canonical edit source and stale-form protection

Saved-item editors load one authorized, repeatable-read source snapshot containing
all editable fields and relationships. An opaque source version travels with the
frozen Form baseline through preservation preview and commit. Both write stages
check that version under the existing locks. Concurrent writes without timestamp
changes are detected; exact retries of accepted receipts remain idempotent.
Private draft ownership, project access and revoked sessions retain their guards.

A source conflict retains the local draft and offers an explicit discard-and-reload
action. Failed reloads retain the draft. Permission loss cancels pending reviews
without discarding uncertain write receipts. Fresh independent backend and frontend
reviews closed without actionable findings after fixes. Focused race/database and
HTTP tests, Go fast checks, web types and global type-aware lint passed. Browser
verification exercised a concurrent title write, rejected stale save, explicit
reload and successful subsequent save. Evidence: `/tmp/vector-source-stage-go.log`,
`/tmp/vector-source-stage-http.log`, `/tmp/vector-source-go-fast.log`,
`/tmp/vector-source-types4.log`, `/tmp/vector-item-source-conflict.png`.

### Checkpoint 28 — type editing from detail and peeks

The type identity beside the item identifier now opens the canonical update form
from full detail, side peek and modal peek. Inactive types retain their identity;
read-only and archived items omit the action. Scoped catalog reads reject stale
permission/session responses. Successful saves refresh schema and activity without
losing the activity method's receiver.

All published-item modal entry points coordinate with native title/description
autosaves. A modal waits for pending, unsaved or failed text edits and exclusively
holds its item editor while open. Distinct editor owners retain their own status;
old sessions cannot flush unsent text or unlock a replacement session. Creation,
copies and private drafts retain their separate workflow. Fresh reviews corrected
early status reporting, session teardown and aggregate save-state notifications.

The shared searchable picker uses the dialog-aware portal and focuses its search
field when that field mounts. This preserves anchoring while keeping options in
the modal accessibility boundary. Native pointer opening from a title field, Escape,
outside clicks and multiple selection were independently reviewed and tested.
Browser checks verified both peek entry points, nested Escape dismissal, successful
type changes and preservation of an old-only numeric value in the description.

The affected frontend suite passed 112 tests, followed by 32 picker regressions
after the focus correction. Web types, UI types and global type-aware lint passed
before that final focus correction; final UI types and the next combined full gate
will verify the final source. Evidence: `/tmp/vector-item-entry-final-tests.log`,
`/tmp/vector-type-menu-focus2.log`, `/tmp/vector-type-entry-types3.log`,
`/tmp/vector-type-entry-lint3.log`, `/tmp/vector-type-switch-preservation.png`,
`/tmp/vector-type-switch-modal-peek.png`. All fresh review findings are closed.
The isolated verification type/property were removed and its item moved to Trash.

These checkpoints do not complete the spec. Inline custom-field editing, activity
and rich-text history/restore, release management, discovery/layouts/exports,
remaining writer audits and final full-spec verification remain open.

### Checkpoint 29 — atomic property history and retained-field reads

Published model saves now capture canonical before/after state under schema/item
locks and append field activity plus rich-text revisions inside the same transaction.
Creation, edits, clears, private-draft publication and reviewed type changes use this
boundary. No-op values do not add revisions. A missing pre-edit version is captured
as an unknown-author baseline; preservation snapshots retain the historical field
name and explicit provenance. Reviewed former-field edits preserve the accepted
new value without losing the previous canonical value. Exact receipt retries do
not duplicate activity, versions or preserved descriptions. Type changes produce
one type activity. Schema removal writers share version metadata; permanent removal
continues to retain old history without copying discarded current content.

All existing item-history reads now enforce current owning-item visibility in the
actual data query, including guest restrictions, private drafts, deleted scopes
and revoked workspace membership. New field-history catalog, version metadata and
sanitized detail routes remain usable after a definition is deleted. Metadata pages
use immutable revision cursors so concurrent appends cannot shift older pages.

Fresh recorder and integration reviews closed their findings. Tests cover canonical
create/edit/clear/no-op behavior, late transaction rollback, publication, edited
former values, explicit former clears, receipt retry, historical labels/provenance,
permanent removal, denied history reads, deleted definitions and stable pagination.
The four affected Go package suites passed with the race detector (including the
full work-item types suite); writer Go fast checks passed without new diagnostics.
Read API database and generated HTTP tests passed. Evidence:
`/tmp/vector-history-all-current.log`, `/tmp/vector-history-writer-focused.log`,
`/tmp/vector-history-publication.log`, `/tmp/vector-history-go-fast2.log`,
`/tmp/vector-history-reads-db.log`, `/tmp/vector-history-http-focused3.log`.
The fresh read-API review closed without findings, and read-API Go fast checks
passed (`/tmp/vector-history-reads-go-fast.log`). Migration12 and the backend
were deployed without restarting frontend watchers. Local authenticated smoke
checks passed for creation/edit/no-op, revision detail, deleted-field discovery,
preservation and activity (`/tmp/vector-history-live.log`). The disposable type
and property were removed and their verification item moved to Trash.

This is the durable history foundation. The production history dialog, activity
renderers, reviewed restore command and search integration remain open, along with
the other unchecked spec requirements above.

### Checkpoint 30 — reviewed revision restore and visible property activity

Saved rich-text revisions can now be restored through the canonical reviewed edit
command. The receipt binds immutable property/revision IDs and the current item
source version; the server resolves and sanitizes the historical body. Current
field attachment, effective context, required rules, item access and asset ownership
are revalidated under the existing locks. A canonical-equivalent restore is rejected
explicitly. Successful restores append a new revision with restore provenance and
one activity event; exact accepted retries do not duplicate either. A late failure
rolls back the item, revisions, activity and durable effects together.

Item activity and profile activity now render custom-property edits, restores, type
changes and preservation/removal notices using their stored historical labels and
readable values. Values remain text, never executable HTML. Detail entries wrap
long values. Shared activity links retain their existing behavior.

Fresh reviews found and closed three authorization gaps: committed edit receipt
responses now check current item visibility; profile activity does the same inside
the result query; activity exports authorize the requester instead of the activity
author. These queries also exclude private/deleted items and revoked membership.
Regression tests cover permission loss between preflight and the result query.

Restore/context/scope/retry/rollback tests passed with the race detector, as did
the activity access and HTTP contract tests. Go formatting/build/vet/lint passed
without new diagnostics. Twenty focused frontend tests passed. Web types passed
before the final profile-map extraction; subsequent global types/lint are blocked
by the concurrent project-filter migration. Evidence:
`/tmp/vector-restore-activity-db.log`, `/tmp/vector-restore-final-http-access.log`,
`/tmp/vector-restore-final-go-fast2.log`, `/tmp/vector-property-activity-types6.log`,
`/tmp/vector-restore-final-frontend-tests2.log`. Full workspace lint is still affected
by the concurrent project-filter migration; no baseline was increased.

The backend was deployed without stopping frontend watchers. An authenticated
localhost check restored a prior value, verified exactly one new revision/event
after retry, and verified retained history after field deletion. The browser showed
the saved value and restore entry (`/tmp/vector-property-restore-activity.png`).
Disposable configuration was deleted and its item moved to Trash. Recent API error
records were canceled project-metadata requests, not restore failures.

The restore backend and activity rendering are complete for this slice. The
production property-history dialog, inline custom-property editing, rich-text search,
release management, discovery/layout/export work and final full-spec gates remain
open. The overall goal is active.

## Checkpoint 31 — production property-history browser and restore UI

Item detail and peek property panels now expose Property history independently of
current property definitions. The dialog lists retained fields, pages through
immutable revisions, shows attribution and read-only rich text, and preserves the
selected revision during refresh. Removed fields remain browsable; only attached,
active rich-text fields expose restore. Browsing does not acquire an editor lease
or call the edit-only source endpoint.

Restore acquires the shared item editor lease, loads the canonical source
fingerprint and schema, and confirms the selected immutable revision. Conflicts
require an explicit reload. Successful saves refresh schema, history and activity
before applying the item and acknowledging its receipt. A recovered move removes
the item from the captured original project list and reconciles the authorized
destination. Replacement list/session/access scopes suppress stale effects.

Fresh reviews found and resolved three defects: successful but unreconciled saves
previously reused stale response data; moved receipts could leave an ID in the old
project list; item-level access denial could leave cached revision content visible.
The shared reviewed-edit hook now replays the exact accepted receipt to obtain a
fresh authorized projection and returns the actual accepted command for attribution.
Definitive history-source or save denials purge scoped history and replace the
dialog with an unavailable/retry state; uncertain receipts remain separately
recoverable. Activity refreshes reject stale content and clear only their own loader.

Independent reviews: `review_property_history_ui_design`,
`review_property_history_ui_implementation`, and `review_history_ui_final`; the last
re-review reported no remaining findings. Verification:

- 49 focused regressions passed in `/tmp/vector-history-ui-final-all-tests.log`.
- Eleven affected production files have zero type-aware lint diagnostics in
  `/tmp/vector-history-ui-final-owned-lint.json`; formatting passed for 16 files.
- Web types passed before the final denial fix. The subsequent check reports only
  a concurrent Kanban migration error in `use-issue-kanban-collapse.ts` (missing
  `filters` on `TIssueKanbanFilters`), recorded in `/tmp/vector-history-ui-types4.log`.
- A real localhost restore updated the displayed value, appended its revision and
  activity, and retained the selected history entry. Screenshot:
  `/tmp/vector-property-history-restored.png`. The temporary live harness's fixed
  revision-count assertion became stale after this extra browser restore; its
  cleanup still removed configuration and moved the disposable item to Trash.
- The single existing frontend watcher stayed running; no backend changes were
  needed for this slice. Combined full gates remain pending shared-checkout work.

History/restore and custom-property activity are implemented. Rich-text search,
inline custom-property editing, release management, discovery/layout/export audits,
and the final full-spec verification remain open. The overall goal is active.

## Checkpoint 32 — current rich-text search

Authenticated workspace/entity/project relationship search now matches sanitized
current rich-text values. One bounded plain-text field snippet accompanies a match.
The projection is separate from canonical items and historical revisions, refreshed
in the same transaction as ordinary writes, reviewed changes, type replacement and
property removal. Read gates enforce current value hashes, attachment, effective
context activation, rollout and canonical item visibility before LIMIT. API-token
v1 search and public projections retain their prior contracts.

Migration 13 backfills without rewriting item timestamps, source fingerprints,
activity or history. Workspace transactions and completion records make interrupted
upgrades retryable. `BOOTSTRAP_TIMEOUT` is configurable (default one hour); the
upgrade instructions require stopping older writers. This migration and backfill
ran successfully locally with the old API stopped, followed by a new API startup.
The shared frontend watcher was not stopped.

Power K uses cancellable query reads scoped to account/session generation,
membership, roles and term. Late content is rejected on scope replacement. Cached
snippets are hidden during a new term/refetch or permission failure. Server results
bypass the palette's title-only filter; inactive membership has an explicit access
state. Snippets render as text nodes. Selecting a field-only match was verified in
the browser and opens its item. Screenshot: `/tmp/vector-richtext-search.png`.
Disposable verification type/field/item were cleaned up (item moved to Trash).

Fresh read-only backend and UI reviews found and resolved the migration deadline,
client filtering and inactive-membership loading defects. Both final reviews are
clear. Their test-strengthening suggestions were implemented: deterministic
failure after a full backfill batch and explicit navigation/closure assertions.

Validation: 40 frontend tests pass (`/tmp/vector-search-final-frontend2.log`);
focused Go race/integration tests pass (`/tmp/vector-search-final-race2.log`),
covering literal/Unicode matching, current values, restore, removed/shared fields,
context activation, full guest/membership visibility, >100-item backfill,
rollback/repair, concurrent backfills and cancellation/retry. Affected Go unit tests
pass (`/tmp/vector-search-units2.log`). New search Go modules/tests and eight owned
frontend files have no lint findings. Formatting passes. Whole-web typecheck has
one concurrent migration error: missing `useParams` in `cycle-issue.tsx`; no search
diagnostic. Full repository gate remains required after all feature work.

The local disk filled during verification. Six GiB of disposable host Go build
cache older than two hours was removed; no project data or shared package output
was removed. No quality baseline was increased.

Next: inline custom-property editing, real release management, remaining
filter/layout/export and writer/hierarchy coverage, then final full-spec gates.

## Checkpoint 33 — integrated inline editing, release catalog and type display

The reviewed inline-property stage is now in production source. Full detail and
peek panels share the published-item editor lease, canonical source fingerprint,
reviewed save/reconciliation flow and access-denial state. Explicit reload retains
local changes; uploads block discard/save until ready. History restore uses the
same source and receipt handling.

Project settings now contain a real Releases catalog and authenticated API.
Project members read it; project administrators create, edit and delete entries.
Mutations use the workspace schema lock/revision and recheck authorization after
locking. Stable client UUIDs support uncertain-create recovery. Deletion rejects
retained references, including private drafts, without exposing their contents.
Names are trimmed, unique per project and limited to 255 characters; descriptions
are limited to 10,000 characters and target dates are optional strict dates. This
supplies Release picker references; shipping, scope and changelog remain separate.

Type badges now follow the Type display toggle independently of the work-item key
across ordinary layouts. One scoped catalog query serves multiple rows, with
retained inactive identities and explicit unavailable/untyped states. Table
loading columns use the actual header visibility rules. The old unused column
counter was removed. Its replacement reduced IssueRowDetails complexity from 27
to 25; both exact baseline entries were tightened, with no increased allowance.

Intake edits now use the ordinary custom-value validation, preservation,
history/search and subscription path. API intake retains the workspace identity
needed to validate cross-project parents. A separate creation-default defect is
fixed: project role reads exclude deleted/orphaned/foreign-workspace projects,
and the observed modal provider selects its default from the visible authorized
picker IDs. Existing item/route/draft identities and archived-project permissions
remain intact. Tests cover metadata arriving after permissions.

S11 foundations were integrated, but S11 is not complete: text, numeric and numeric
range filter editors and an explicitly authorized type-filter SQL compiler exist.
The compiler has no authenticated list/catalog integration yet. Custom-property
catalogs, filters/grouping/counts/saved views, custom columns and exports remain
open; these foundations do not satisfy those requirements by themselves.

Verification so far:

- Integrated focused backend tests passed for issuequery, projectmembershipreads
  and workitemtypes, including actual SQL type-filter/null/count semantics, role
  scope, release locking/retries/references and intake atomicity/parent rules.
  Evidence: `tmp/work-item-spec-integrated-backend-tests.log`.
- Release service mapping and HTTP protection tests passed before integration;
  affected transport was then regenerated from the actual source design.
- Focused UI suites passed 51 tests, with one test-module loader failure. Removing
  its obsolete staging-only constant import allowed all seven type-display tests
  to pass against the real package. Creation/source/access tests passed 14 cases.
  Logs: `tmp/work-item-spec-integrated-frontend-tests.log`,
  `tmp/work-item-spec-type-display-tests.log`, and
  `tmp/work-item-spec-creation-tests.log`.
- Fresh reviews cleared the staged features and integrated batch. The creation
  reviewer caught a shared-hook reactivity regression; the corrected fix is
  confined to the observed modal provider and passed a fresh follow-up review.
- Fast lint ratchet passed with no increased diagnostics. Whole-web types passed
  after restarting the frontend watcher to load the new Releases virtual route.
  Evidence: `tmp/work-item-spec-web-types4.log`.
- Go formatting, build and vet passed. Lint required retiring one obsolete intake
  shadow diagnostic (count 11 to 10); the exact reduced baseline now passes its
  ratchet. No new allowance was added.
- The host disk filled during verification. Six GiB of generated Go cache files
  older than two hours were removed. Frontend restarted successfully; Docker
  reported storage I/O failures. Colima recovered after restart, but its cached
  API binary had a zeroed executable header. A clean image build with a fresh
  disposable Go cache restored the API. No source workaround or data deletion
  was required. The Releases route now loads its actual empty catalog and its
  creation modal enforces required-name validation. Further live UI verification
  and final full gates remain required.

The earlier shared Go full gate passed before this integration. It is not evidence
of a full-gate pass for checkpoint 33. The complete feature goal remains active.

### Checkpoint 34 — live inline editor and integration lint

A live rich-text property edit exposed a render/transaction feedback loop. The
inline form incremented a render counter for every editor transaction, while the
editor updated its file handler on render and emitted another transaction. The
form now stores only upload readiness; unchanged readiness does not rerender it.
Save and reload still consult live editor readiness before proceeding.

The regression uses the real editor hook with a fresh file-handler object on each
render. It failed before the fix, then passed with typing, busy-to-ready upload
transitions, preserved editor identity and the saved custom-value payload. Eight
focused tests passed, and a fresh reviewer independently reran both real-editor
suites (two tests) with no findings. In localhost, the actual rich-text property
saved successfully, its activity entry appeared, and both survived reload. The
verification type/property were removed and the disposable work item moved to
Trash. Screenshot: `tmp/work-item-inline-saved.png`.

The full type-aware lint check then exposed integration issues missed by fast
lint. FilterValueInput now accepts the full value union used by its actual caller,
removing unsafe callback narrowing; release drafts use a typed copy, access
initialization has an explicit return type, and permission checks use the scope's
actual enum. Reload returns are consistent, and new test registrations are
awaited. A fresh review found no actionable issues in these repairs. The first
focused batch passed 43 tests; the corrected literal/source/release batch passed
18 tests. These overlap and must not be reported as 61 distinct tests.

The next authenticated discovery review also identified the cycle repository's
separate filter parser and grouped query path. Its existing relation multiplicity
must remain intact when adding type predicates. Saved-view adapter parse failures
currently turn into null/empty expressions, so repair cannot be treated as done.
The usage-bearing workspace type catalog is unsuitable for member filter options;
add a bounded, usage-free authorized identity catalog. Snapshot, public/API
exclusion and result/count agreement remain explicit implementation requirements.

Full gates and remaining S11 work are still open. This checkpoint is progress,
not completion of the feature goal.

Release catalog creation and editing were verified through the actual project
settings UI. The updated record survived reload; the disposable release was then
removed and the empty catalog confirmed through the production API. Evidence:
`tmp/work-item-release-saved.png`. Together with the revision/reference/permission
backend tests and draft/access frontend suites, this completes the catalog slice.

After repairs, whole-web types passed (`tmp/work-item-spec-web-types-lint2.log`)
and the full type-aware lint ratchet passed (`tmp/work-item-spec-full-lint-final.log`).
The four obsolete unsafe callback-cast baseline diagnostics were retired exactly;
no allowance was added. The full repository gate remains pending.

### Checkpoint 35 — consolidation required by the full quality gate

The concurrent duplication scan found new copies in inline preview authorization
and feature test setup. `captureItemPropertyRead` now owns the identical account,
session, membership, query identity, cancellation and definitive-denial checks
used by published edit reads and inline formula previews. Query/permission test
fixtures and real-editor lifecycle setup are shared without removing scenario
assertions or changing production behavior.

The affected run exposed an older detail-property fixture that lacked the item
project identity now required by the production guard, then expected pre-denial
error copy. Supplying that identity and asserting the current definitive-denial
message restored the test; its private-content, readonly editor, retry and
membership-revocation assertions remain intact.

Verification: all 40 tests across eleven affected suites passed
(`tmp/work-item-spec-clone-tests2.log`). Whole-web types passed
(`tmp/work-item-spec-clone-types.log`). The complete current duplicate scan found
zero new clones (`tmp/work-item-spec-clones-final/jscpd-report.json`), and full
type-aware lint passed with no increased diagnostics
(`tmp/work-item-spec-clone-full-lint.log`). No baseline increase or exclusion was
used. A fresh review cleared the extraction and fixtures, and independently
reran the repaired detail test (one pass). The frontend watcher was not restarted.

A separate fresh review confirmed the next type-discovery transaction design:
usage-free bounded identities, authorization inside the repeatable-read snapshot,
repository-specific error mapping, preserved visit ordering, and explicit v2
filter handling. The catalog/compiler/list integration is not implemented yet.
This checkpoint advances required gates; S11 remains open.

### Checkpoint 36 — authenticated type discovery and list filtering

Added a bounded, usage-free type identity endpoint for authenticated workspace and
project members. Catalog authorization, retained identities, filter compilation,
counts and list reads share a repeatable-read transaction. Missing identities and
catalog overflow reject the filter instead of broadening the result. Inactive and
unimported workspace types remain recognizable. Public and API-v1 filter contracts
remain unchanged.

Project, workspace, profile and archived lists now use the authorized type compiler.
The v2 route validates every explicit structured filter; malformed input cannot
skip compilation. Workspace/profile lists also exclude deleted projects and
workspaces. Real database regressions cover pagination, guest/archive visibility,
foreign scope, missing identities and concurrent changes between count and rows.

Focused discovery integration, all work-item repository tests, generated service
and HTTP tests pass. Go build, vet and formatting pass; lint passes with 1,346
inherited diagnostics and duplicate checks with 503 inherited pairs. A fresh
review found and verified the v2/deleted-project fixes, then cleared the final
slice. Shared project/release design declarations were consolidated to remove a
clone; regeneration produced byte-identical transport artifacts, independently
reviewed. No diagnostic baseline increased.

Evidence: `tmp/type-discovery-integration-final.log`,
`tmp/type-discovery-workitems-regressions.log`,
`tmp/type-discovery-transport-tests.log`,
`tmp/type-discovery-lint-check.log`, `tmp/type-discovery-dupl-final.log`.
The running API has not yet been rebuilt with the discovery endpoint. Client
catalog/filter wiring and the rest of S11 remain open; this is not completion.

### Checkpoint 37 — module and cycle type discovery

Module and cycle app lists now resolve their type catalog and execute scoped
ordinary/grouped queries inside the same authorized snapshot. Their database
interfaces require transactions. Existing elevated project access checks still
run inside that snapshot. Private per-request cycle options carry catalog IDs;
shared type-leaf compilation leaves the legacy cycle relation joins intact.

Cycle logical operators now render case-insensitively, matching their validator;
previously an accepted uppercase operator could be ignored. Shared compiler errors
are converted to the cycle HTTP-error contract. Module and cycle app list payloads
now include type_id for the existing type-display UI. Public and API-v1 serializers
are separate and unchanged.

Real PostgreSQL tests cover pagination, grouped counts, mixed-case AND/OR/NOT,
relation multiplicity, empty selections, retained paused/unimported types,
malformed/unknown filters, disabled configuration and guest/outsider denial.
They also assert type identities in ordinary and grouped response rows. The
production schema correctly rejects clearing a type while types are enabled;
null matching remains covered by the shared compiler's independent SQL tests.

Verification: focused module/cycle regressions pass
(`tmp/module-cycle-integration-final.log`), the existing cycle SQL integration
and shared compiler tests pass (`tmp/module-cycle-integration3.log`), and a fresh
reviewer independently passed all three affected unit packages. It found no
remaining issues in snapshot authorization, error conversion, legacy joins or
projection boundaries. Go fast checks passed with 1,346 inherited diagnostics;
the latest duplicate check passes with 503 inherited pairs. No baselines changed.
Localhost responds HTTP 200; the frontend watcher remains running. Full lint
passed again after the final projection assertions (1,346 inherited diagnostics;
`tmp/module-cycle-full-lint-final.log`). Client catalog/filter wiring,
saved-filter repair, other S11 requirements and the complete Prek gate remain open.

### Checkpoint 38 — preserve saved predicates through conversion and async edits

Filter conversion now retains the original external expression and exposes a
visible error when the editor cannot represent it. Loading/resetting unsupported
input does not notify an empty replacement, and edit/save actions are blocked
until it is resolved. An explicit clear is available and remains dirty until the
updated view is successfully saved. This error UI describes an actual saved-data
problem, not prototype status.

Loaded null and empty selections remain committed predicates; unfinished new
controls are still omitted. Loaded homogeneous arrays retain their array wire
format through edits and saves. Converting those arrays to comma strings would
change the legacy backend's matching behavior. Unsupported mixed/null arrays and
unimplemented OR structures retain their raw data for repair rather than changing
semantics. Full grouped-expression editing remains next; this checkpoint does not
claim the entire saved-view requirement complete.

Clearing waits for its callback to succeed and checks whether newer data arrived
before applying the result. View updates acknowledge only the submitted snapshot;
newer edits stay dirty, and a reset during the save cannot receive a stale
acknowledgement. Workspace preference equivalence preserves unsupported and
null/empty predicates without changing the native revision and query guards.

The fresh reviewer found and verified fixes for rejected clears, delayed saves,
array rewrites and raw-baseline dirtiness. It independently ran 15 preservation,
operator and actual error-UI tests with no remaining findings. The combined
regression run passed 40 tests, including pending-save/view-transition protection,
existing workspace preferences and filter lifecycle behavior
(`tmp/filter-preservation-complete-tests.log`). Shared-state types pass;
whole-web types and full type-aware lint were checked, and the duplicate scan
reported 572 inherited pairs with no new copies. No baselines were increased.
Localhost remains HTTP 200; the frontend watcher was not restarted.

### Checkpoint 39 — grouped predicates and sidebar restrictions

The filter model and editor now retain AND, OR and nested NOT boundaries, including
mixed-case logical keys and bare equality fields. Group controls switch all/any
without flattening predicates. Saved scalar, array, null and empty values retain
their wire semantics. CSV tokens are preserved verbatim; whitespace and empty tokens
cannot silently disappear. New incomplete controls remain distinct from committed
empty predicates. Edits beyond the API's five-level limit are rejected atomically
with an editable error state.

Sidebar selections now find only conjunctive conditions outside OR and excluded
groups. Changing an assignee restriction cannot rewrite a branch of an existing
OR predicate. Both module and cycle sidebar reads use the same lookup as writes.
A fresh review found and verified the CSV, depth and sidebar fixes; independently
ran 29 tests without remaining findings. The combined run passed 53 tests
(`tmp/group-filter-final-tests.log`). Later catalog tests also cover this slice.
The normalization helper reuses the sanitizer with an explicit completeness
predicate, removing a new duplicate rather than expanding the baseline.

### Checkpoint 40 — authorized type filter catalogs and live verification

Type filters and badges now share the bounded, usage-free identity endpoint.
Queries support workspace or project scope and reject responses after membership,
query identity or session-generation changes. Project-role request failures do not
block workspace-only identity reads. Disabled or failed catalogs remove cached
choices and labels while retaining the original predicate; failures expose retry.
Config registration removes omitted definitions, and disabled operators render a
repairable raw-ID chip. Missing selected IDs stay visible rather than disappearing.
Async select options belong to the exact config that requested them; late previous
catalog responses cannot restore old labels.

The type selector includes icons/colors and distinct equality, inequality,
inclusion and exclusion operators. Paused and unimported identities are retained.
The fresh reviewer found and verified the workspace retry and duplicate-label
fixes. Independent catalog/badge/filter UI tests passed. Browser testing then
caught a nested-chip rendering defect: recursive calls resolved to the named inner
component instead of its observable wrapper. Renaming the inner function restores
observable recursive rendering; a focused actual-UI regression and fresh review
verify in-place value changes.

The local API binary was rebuilt and deployed, including checkpoints 36–37's type
identity and module/cycle query support. The frontend stayed running. Live Platform
App checks confirmed Bug equality returns no matches, Bug inequality returns 48
items and survives reload, Task inclusion returns 48, and excluding Task plus Bug
returns no matches. Original project filters were restored afterward. No items or
configuration definitions were changed for verification.

Before the final nested-chip regression, 68 combined tests passed, whole-web types
passed, full typed lint passed (3,684 files, no new diagnostics), and the duplicate
scan reported 571 inherited pairs with no new copies. Final expanded validation
is recorded in `tmp/type-filter-complete-final-tests.log`,
`tmp/type-filter-final-types.log`, `tmp/type-filter-final-lint.log`, and
`tmp/type-filter-final-clones.log`. Independent evidence:
`tmp/type-filter-independent-review.log`, `tmp/type-filter-recursive-review.log`.

Full S11 is still open: effective custom fields/formulas before filtering/counting,
custom columns/grouping/export, type/parent-type groups and remaining writer audits.
The complete stable Prek gate and final live acceptance audit remain required.

Checkpoint 40 final checks passed: 69 combined tests, whole-web types, full typed
lint (3,684 files; 456 inherited errors and 3,294 warnings; no increases), and 571
inherited clone pairs with zero new pairs. Final local errors were the already
tracked next-themes client-script warning and request-cancellation messages during
navigation, not a failed type catalog or filter request.

### Checkpoint 41 — shared effective-value projection and reusable formula plans

The item-schema path now resolves its presentation and formula data through a
request-scoped projection plan. The plan accepts only attached definitions and
captures effective project/type contexts. It distinguishes not applicable, empty,
value and error states, keeps option-catalog ownership separate from the winning
presentation context, and projects cascading values by stable level identity.
Formula values also carry their result kind. Draft cascading remains unavailable.

Read normalization shares scalar parsing with writers but does not impose creation
defaults, requiredness or current member/release eligibility. Missing Boolean is
false; zero remains a value; read-only Text uses configured content. Retained
selection IDs remain readable. Unattached saved fields are not projected. Malformed
values produce an error without changing stored data. The reviewer caught an
overlong cascading path whose full value disagreed with its level projection; it
now produces an error, with the original saved path retained for repair.

Formula compilation is separated from row evaluation. Each resolved scope compiles
its expressions once, then evaluates isolated row inputs using the supplied common
instant. Existing lazy-branch, typed-input, empty and error semantics are retained.
This avoids introducing a second formula implementation for discovery queries.

Fresh review found no remaining issues after the cascading/result-kind fixes and
independently passed all three affected unit packages
(`tmp/effective-projection-independent-review-final.log`). New behavioral coverage
checks all stored kinds, inactive/unattached fields, missing required values, false
and zero, retained references, context/catalog identities, copied row inputs,
shared timestamps, lazy/error formulas and immutable compiled expressions.

Affected unit packages pass (`tmp/effective-projection-tests-final.log`), as do
expanded item-schema, formula authoring, item-value and preservation integration
checks (`tmp/effective-projection-integration-final.log`, 32 seconds). Go formatting,
build, vet and lint pass with 1,346 inherited diagnostics
(`tmp/effective-projection-go-final.log`); duplicate checks pass with 503 inherited
pairs (`tmp/effective-projection-dupl-final-check.log`). Repeated formula test
fixtures were consolidated instead of increasing a duplicate allowance.

This is a shared evaluation foundation, not completed custom filtering. Batched
authorized discovery inputs, custom-aware Boolean compilation, counting/pagination,
columns/grouping/export integration and the remaining full-spec gates are still
open. These Go changes are not deployed yet; the existing local app remains HTTP
200 and the frontend watcher was not restarted.

### Checkpoint 42 — coherent discovery snapshots and list value projection

Discovery now bulk-loads attached property definitions in the authorized repeatable
read transaction. It caches effective projection plans by project/type and uses one
transaction timestamp for every formula result. Definitions and other contexts stay
private to the snapshot; rows still have to pass the route's complete item visibility
predicate. Missing types, inaccessible projects and catalog overflow fail explicitly.
Concurrent configuration changes become visible on the next request, without mixing
old and new definitions in one response.

Authenticated work-item lists accept `include_properties=true` to return
`property_values` and `properties_evaluated_at`. Values include applicability,
empty/value/error state, effective context, option-catalog owner, formula result
kind and cascading level identities. Ordinary project, workspace, user, detail,
v2, grouped and ID-selected response shapes retain this projection. Numeric
estimate values are loaded in one scoped query for admitted rows. Existing raw
query forwarding carries the opt-in without modifying generated transport output.
Legacy lists and API-token responses keep their previous representation.

This only projects admitted list rows. It does not implement custom filtering:
custom predicates must evaluate all authorized candidates before counts and page
windows. Module/cycle projections, custom columns, grouping and exports also remain
open. No production data was changed and these Go changes are not deployed yet.

Eight focused database tests pass, including scope denial, paused types, catalog
overflow, concurrent schema changes, common formula time, guest visibility,
legacy/API isolation, numeric estimate/detail parity, grouped and ID-selected
responses (`tmp/property-list-integration-final.log`). A fresh reviewer found no
remaining defects and independently passed the five list tests
(`tmp/property-list-independent-review.log`). Snapshot loading was also independently
reviewed and tested (`tmp/property-discovery-independent-review.log`).

The broader Knip audit found the identity catalog return type was unnecessarily
exported. It is now module-local; the service behavior and runtime contract are
unchanged. No baseline allowance was added.

Checkpoint 42 final Go checks pass: three affected unit packages, formatting,
build, vet and lint (1,346 inherited diagnostics), and duplication (503 inherited
pairs). Evidence is in `tmp/property-list-unit.log`,
`tmp/property-list-go-fast.log` and `tmp/property-list-dupl-check.log`. Localhost
still returns HTTP 200. Full Prek and end-to-end feature acceptance remain open.

Fresh review of the next custom-filter slice identified two required compatibility
seams: relation-name ordering reuses the structured filter, and cycle reads have
different legacy negation and row-multiplicity semantics. The implementation must
preserve those route contracts while carrying separate true/false eligibility
through mixed expressions. The review is saved locally in
`tmp/custom-predicate-design-review.md`; this compiler is not implemented yet.

### Checkpoint 43 — prepared custom-filter Boolean compiler

The query compiler can now prepare a validated expression while retaining the
legacy AST and relation scopes. Authorized callers supply additional property-field
validation, then evaluate each enumerated leaf over their complete candidate set.
Binding requires a result for every leaf, rejects invalid or contradictory item
identities, and copies input values. Unknown membership is distinct from false.
Existing public and legacy entry points keep their field allowlist and behavior.

Custom predicates compose separate true and false results through AND, OR and
NOT. Mixed relation scopes retain the same positive aliases and quantify definite
false separately from absence of a true match. This prevents nested exclusions
from accepting unavailable custom values. Relation-name ordering consumes the same
prepared expression with its existing ordering alias. Cycle-specific compilation
is still separate; this does not replace its distinct negation or join semantics.

Real PostgreSQL tests exposed an eager-binding defect: unused true/false parameters
could not be typed when only one branch was emitted. Each leaf now binds one typed
JSONB map from admitted item identity to Boolean result. An absent identity remains
unknown in either branch. No dummy predicates are used to consume parameters.

Coverage includes complete/disjoint materialization, copied inputs, legacy no-op
and validation compatibility, positive/negative/double-negative results, mixed
legacy null behavior, decisive Boolean branches, missing/deleted relations,
shared positive aliases and relation-name ordering. Fresh review independently
passed the full integration-tagged query package after the binding fix
(`tmp/property-filter-independent-review.log`). Four affected unit packages pass
(`tmp/property-filter-affected-units.log`).

This compiler is not yet connected to production custom-property predicates.
Authorized property/operator/catalog validation, all-candidate evaluation before
counting, route adapters, UI controls and final acceptance remain required.

Checkpoint 43 final Go formatting, build, vet and lint pass (1,346 inherited
diagnostics) after fixing a new shadowed-variable warning rather than changing its
allowance. Duplicate checks retain 503 inherited pairs. Final evidence:
`tmp/property-filter-go-final.log`, `tmp/property-filter-dupl-check.log`, and
`tmp/property-filter-integration-final.log`. Full-spec acceptance remains open.

### Checkpoint 44 — effective custom predicates before counts and pagination

Authenticated work-item lists now recognize custom-property expressions, authorize
field identities and compile typed comparisons within the discovery snapshot.
Supported families cover Text, Number, Boolean, Date, URL, Dropdown, Member,
Release, cascading levels and formula result kinds. Rich text is explicitly
rejected. Invalid operators, malformed bounds, unknown definitions, foreign
references and unavailable catalogs fail rather than broadening the query.

Text and Formula Text use Unicode case folding, as required by the spec. Fresh
review caught the initial case-sensitive implementation; equality and contains
regressions now include Unicode cases. URL comparisons retain literal case and
Date comparisons use UTC calendar days, including timestamp-valued formulas;
these decisions are recorded in the specification. False and zero remain values.
Applicable empty fields match explicit empty tests but are unknown for nonempty
comparisons. Inactive, unattached, erroneous and wrong-catalog values remain
unknown through negation. Applicability belongs to each leaf, allowing another
branch of a mixed expression to decide the result.

Property keys use `customproperty_<property UUID>`, optionally followed by
`_level_<level UUID>` and `_catalog_<context UUID or base>`, then the operator
suffix. Cascading predicates require both level and catalog. Validation resolves
the option owner separately from presentation context. Member identities are
workspace-scoped; release identities are limited to authorized projects. Historical
identities can remain valid for discovery without granting new write eligibility.
The internal caches and effective configurations are not returned as a public
catalog by this change.

The list repository applies every route visibility rule and independent flat
filter to the candidate query, omits the entire structured expression, projects
all authorized candidates, then binds the complete mixed expression before counts,
relation ordering and windows. It never filters a fetched first page or extracts
terms from mixed OR/NOT. Ordinary, workspace, user, detail, v2, grouped and ID lists
share this path. Value projection remains separately opt-in. API-token structured
filters remain rejected; public projections gain no custom fields.

Tests verify matches beyond the initial unfiltered page, exact totals, formula
parity, legacy disjunction, catalog owners with identical option IDs, invalid levels,
foreign references, inactive contexts, guest restrictions, grouping and ID-selected
lists. The combined predicate/projection/snapshot database suite passes in 29.261s
(`tmp/property-predicate-integration-final.log`). Fresh follow-up review found no
remaining defects and independently passed the predicate suite in 14.599s
(`tmp/property-predicate-independent-review-final.log`).

Module/cycle adapters, the client filter catalog and controls, custom columns and
grouping, exports and final end-to-end acceptance remain open. These backend changes
are not deployed yet; the existing local application remains running.

Checkpoint 44 final affected unit packages, Go formatting/build/vet/lint and
independent duplicate checks pass. The inherited counts remain 1,346 diagnostics
and 503 clone pairs; the new ineffective assignment was fixed without increasing
an allowance. Logs: `tmp/property-predicate-affected-units.log`,
`tmp/property-predicate-go-final.log`, `tmp/property-predicate-dupl-check.log`.
Localhost still returns HTTP 200. Full Prek and final feature acceptance remain open.

### Checkpoint 45 — effective property catalog and typed filter controls

The authenticated `work-item-property-identities` endpoint now exposes only the
active, effective field identities available to the current project/workspace
scope. It includes the supported operators and authorized selection choices, with
separate identities for cascading levels and option-catalog owners. It omits
usage counts, defaults, expressions and private context configuration. Reference
choices are scoped; rich text remains display-only. Backend validation and catalog
operators share one definition. The two discovery transports share their route
contract rather than duplicating it.

Project, workspace, profile and archive filters now consume that catalog through
account/session/membership/project-scoped Query reads. These pages use the
repository predicate compiler from checkpoint 44. Module, cycle and saved-view
endpoints are not enabled until their repository adapters support the same
contract. Failed or disabled catalogs remove unavailable labels/choices without
rewriting saved predicates; failed reads expose retry. Configuration mutations
invalidate both discovery catalogs.

The actual filter picker supports numeric comparisons/ranges, text containment,
dates, selections, Boolean values and explicit empty checks. Formula fields use
their result type. Cascading choices retain both level and catalog identity.
Extended negative operators now convert to the underlying operator plus NOT.
Independent review found and verified fixes for incompatible operands surviving
operator changes, date ranges shifting across time zones, `false` opening a
selector automatically, and ambiguous empty-check labels. Live logging also found
a controlled/uncontrolled selector transition; absent values now normalize to
`null` at the control boundary without coercing `false` or `0`.

Fresh backend catalog review passed the database and HTTP tests
(`tmp/property-catalog-independent-review.log`). Independent frontend reviews
passed the catalog, routing, session isolation and final control regressions
(`tmp/property-filter-wiring-review-fixed.log`,
`tmp/property-filter-control-independent-review.log`). Web, types, utilities and
shared-state typechecks pass. Go formatting/build/vet/lint passes with 1,335
inherited diagnostics; the label slice's separately reviewed baseline retirement
accounts for the decrease. Go duplicate checking passes at 503 pairs. Knip remains
398; frontend duplicates are 569. Type-aware lint passes with no increase (456
inherited errors, 3,273 warnings). No allowance was added for this checkpoint.

The local API now includes checkpoints 41–45. Only the API container was replaced;
the existing frontend watcher stayed running. Live browser checks verified numeric
zero, greater-than exclusion, dropdown equality, switching to an empty check and
saved-filter reload. The temporary properties/type were removed, the disposable
item moved to Trash, and the original filters restored. The app remains available
at `http://localhost:8000/`. Verification screenshot:
`tmp/property-filter-live/verified-presence.png`.

Module/cycle/view custom adapters, custom table columns, categorical and type/parent
regrouping, authorized exports, remaining writer audits and full Prek/end-to-end
acceptance are still open. This checkpoint does not complete S11 or the full spec.

### Checkpoint 46 — shared discovery projection for list adapters

Effective item projection now has a typed batch boundary in `workitemtypes`.
`PropertyDiscoverySnapshot.ProjectItems` uses its own callback transaction, validates
all project/type scopes before reading estimates, and resolves numeric estimate
values through actual issue relations in the snapshot workspace/projects. The
supplied estimate identifier is only a lookup hint. Removed points/catalogs and
missing estimates clear copied formula inputs; caller objects remain unchanged.
Results retain input order and duplicate rows, and failures return no partial
projection. The existing work-item list adapter now uses this boundary.

This extraction supports module and cycle integration without importing one
repository into another or duplicating formula/estimate behavior. Fresh review
confirmed that cycles must retain their own plain SQL negation and direct relation
joins: the general prepared filter renderer has different legacy NULL semantics.
The cycle extension should render custom leaves as nullable Boolean lookups while
keeping its existing relation multiplicity and pagination.

The focused batch/list/custom-filter database suite passes in 29.675s
(`tmp/property-batch-integration.log`). Independent batch tests pass in 7.893s,
and the reviewer found no remaining production defects
(`tmp/property-batch-independent-review.log`). A test initially tried to simulate
concurrency through the active read-only connection; it now uses a separate writer
and verifies the estimate value stays frozen within the read snapshot. Affected
unit packages, Go formatting/build/vet/lint (1,335 inherited diagnostics), and
owned-source duplication (503 pairs) pass without baseline increases.

Module/cycle query adapters are still open. This checkpoint changes no frontend
controls or deployed API; localhost remains HTTP 200 and the existing stack stays
running. The other integration's label query changes were preserved.

### Checkpoint 47 — module property filters and query-scope regression

Module lists now evaluate scoped properties and formulas over every authorized
candidate before counts, groups, ordering, and pagination. The adapter shares the
snapshot batch projector and preserves module route joins, flat predicates, role
checks and relation ordering. Module pages expose the existing typed controls.

The removed-module-item regression and fresh review found a shared SQL precedence
bug: a top-level custom/legacy OR could escape surrounding route and access
predicates. The complete expression is now parenthesized centrally. Database
regressions cover both sides of a surrounding conjunction, guest visibility,
deleted items, and removed module membership. The issuequery and workitemtypes
suites pass (6.396s and 22.599s); independent affected units and review are clear.
Go fast gates pass with 1,335 inherited diagnostics and duplication with 503 pairs;
no baseline increased. Web typechecking and seven focused frontend tests pass.

The local API includes checkpoints 46–47, and the frontend watcher stayed running.
Live module checks verified numeric equality with zero, saved-filter reload, and
greater-than exclusion. A development refresh temporarily dropped project metadata;
reload recovered it, with only the previously tracked script-rendering warning in
the error log. The test filter, temporary module/type/properties and item were
removed. Evidence: `tmp/property-filter-live/module-greater-than.png`.

Cycle/view custom filtering, table columns, categorical/type/parent grouping,
exports, remaining writer audits and full acceptance gates remain open. A fresh
cycle design review confirmed its separate compiler needs nullable custom leaves
and preservation of direct relation multiplicity, and found the equivalent
existing top-level OR scope bug there. Cycle implementation is now in progress.

### Checkpoint 48 — cycle property filters with native relation semantics

Cycle lists now evaluate custom properties and formulas within the authorized
repeatable-read snapshot before counts, grouping and pagination. Candidate and
final queries share route joins and flat predicates. The existing cycle compiler
retains plain SQL negation and direct relation joins: custom leaves use nullable
Boolean lookups, so unavailable properties remain unknown and duplicate relation
rows keep their established pagination behavior. Bindings are frozen after
validation; final rendering rejects an unprepared operand.

The cycle compiler also now groups its complete top-level OR before it is combined
with route predicates. Regressions verify both custom/legacy and legacy-only OR
against removed cycle links and independent foreign-workspace items, including
grouped queries. Additional checks cover same-field predicates with different
operands, nullable nested NOT, formulas, matches beyond the first page, counts,
role checks and disabled definitions. Test fixtures were corrected to use legacy
CSV relation operands and fresh tenant records rather than forbidden scope moves.

Focused cycle/work-item database tests pass in 19.432s; independent cycle property,
type-discovery and live-label suites pass, and the reviewer found no remaining
production issues. Go fast gates pass (1,335 inherited diagnostics), duplication
passes (503 pairs), and web typechecking passes without baseline increases.
The local API includes this checkpoint; CYCLE controls are enabled. Browser checks
verified False matches, True excludes, and reload preserves the saved expression.
Evidence: `tmp/property-filter-live/cycle-boolean.png`. The temporary cycle was
removed; its disposable item/properties remain temporarily for saved-view checks.

Saved-view query support already uses the general work-item repository. Fresh
review found save-acknowledgement and premature canonical-cache mutation defects;
those are the next fixes before PROJECT_VIEW controls are enabled. S11 and the
full feature acceptance remain incomplete.

### Checkpoint 49 — saved custom filters and acknowledged view updates

Project saved views now expose the same typed property predicates as project,
module, cycle and workspace lists. Their existing query paths already preserve
structured expressions and evaluate them before pagination. The shared view
controls now return and await the actual save operation. Rejected writes retain
the dirty filter expression; accepted writes acknowledge only the submitted
revision, preserving edits made while the request was pending.

Project views adopt the canonical response only after success, with current
identity, permission, store and latest-write guards. The common access checks were
extracted from work-item read scope without weakening its resource-query guard.
The common creation payload and save feedback live in shared helpers, preserving
workspace preference ownership and preventing late feedback after navigation.

The combined focused frontend suite passes 58 tests; a fresh independent review
passes 31 affected tests without findings. Web typechecking, Knip and duplication
pass (397 unused-code findings and 559 existing clone pairs, zero new clones).
The latest full typed lint identified only seven diagnostics in the concurrent
pagination test work; this feature's test-loop diagnostic was fixed and its seven
tests rerun successfully. No baseline was increased.

Browser and independent API checks verified numeric zero plus Boolean false in
both project and workspace saved views, changing false to true, saving, and
reloading the persisted expression with no remaining dirty update action.
Evidence: `tmp/property-filter-live/project-view-reload.png` and
`tmp/property-filter-live/workspace-view-reload.png`. The workspace fixture was
corrected from an unsupported list layout to its supported spreadsheet layout.

Cleanup exposed a separate deletion defect: an unused SQL parameter caused HTTP
500 after the view had already been soft-deleted. Its atomic deletion correction
and database regression are in progress. Temporary properties/type have been
removed and the test item moved to Trash; saved-view cleanup will finish after
that correction is deployed. Custom table columns, grouping, exports and the
remaining full-spec acceptance checks are still open.

### Checkpoint 50 — atomic saved-view removal

View deletion and its favorite/recent-visit cleanup now run in one PostgreSQL
statement. This removes the untyped unused parameter and prevents a failed
cleanup from leaving a deleted view behind. Project and workspace permission and
reference scoping remain unchanged. Database tests cover both scopes, denial,
unrelated references, and cleanup failure under autocommit; the latter would
catch the previous partial deletion without a test transaction masking it.

The affected integration suite passes (8.792s); fresh independent review and its
suite pass (7.023s). Go fast gates pass with 1,334 inherited diagnostics, and
owned-source duplication passes with 503 pairs. No baseline increased. The local
API was rebuilt and restarted with this fix and the concurrent label changes;
the frontend watcher stayed running. The remaining disposable project view now
deletes successfully through the API, completing saved-view fixture cleanup.
The browser is back on Platform App. The full spec remains incomplete.

### Checkpoint 51 — module/cycle table projection boundary

Authenticated module and cycle lists now accept `include_properties=true` and
project their admitted page rows through the existing effective-property and
formula evaluator. A shared snapshot method bulk-loads canonical inputs, validates
every item/project pair, preserves row order and duplicates, and changes no row
until the whole batch succeeds. Projection-only requests do not evaluate the
entire candidate set. Grouped cycle responses retain their bucket metadata and
use one projection batch across their leaf rows. Public/API-token routes remain
on their existing serializers.

Fresh review found a permission mismatch between strict item visibility and the
module/cycle workspace-admin exception. Projection now selects an explicit
visibility policy: strict callers retain guest item restrictions, while elevated
project routes verify active workspace-admin and project membership in SQL.
Regression cases cover both routes with ordinary guests, admins with project
guest access, inactive memberships, and strict admin reads of hidden items.

Focused database checks pass in 57.948s; independent final projection checks pass
in 34.238s and adapter checks in 0.428s, with no remaining review findings. Tests
also cover snapshot consistency, estimates/formulas, missing/draft/deleted items,
conflicting and accessible-but-mismatched project claims, no partial row mutation,
and unchanged paginated/grouped envelopes. JSON number decoding is consistent
across the new projection paths. Final Go gates and local deployment verification
are running. The table frontend is not wired yet; full-spec work remains open.

Checkpoint 51 verification: Go fast gates pass (1,334 inherited diagnostics), with
no baseline increase. The local API now includes this checkpoint; a read-only
live check on a populated module confirms identical item IDs/counts and projected
values on each returned row. Platform App remains available without restarting
the frontend. Final duplication verification is being repeated after the access
regression additions.

### Checkpoint 52 — table presentation contract

Projected Date values now carry the effective context's display format; catalog
choices carry optional option icons and parent IDs for path labels. This exposes
only presentation data from authorized catalogs, without copying configuration,
defaults, formulas or private contexts into list responses. The frontend now has
a typed projection DTO on work items and shares its property-kind union with the
identity catalog service.

The focused projection/catalog tests pass (14.143s), including effective Date
contexts, option presentation and private-context exclusion. Fresh independent
review is clear and its suite passes in 9.257s. Shared types and web typechecking
pass. Final lint/Go checks are running; this contract is not yet deployed and the
table frontend remains the next integration step.

Checkpoint 52 final checks: Go fast gates pass (1,334 inherited diagnostics), and
full typed lint passes across 3,716 files with no diagnostic increase. No baseline
was changed. The local API remains at checkpoint 51 pending the next combined
backend deployment.

### Checkpoint 53 — disabled-feature projection reads

Presentation-only projection reads now keep ordinary table data available when
work-item types are disabled. They retain the authorized snapshot, project scope
and timestamp, skip definitions and estimate evaluation, and return explicit
empty property maps. This also clears previously projected values during entity
merges. Requests containing type or custom-property predicates still use the
strict discovery boundary and fail while disabled; malformed legacy filters and
authorization errors are never downgraded to successful ordinary reads.

Project, V2, workspace, module and cycle regression checks pass. The broader
focused integration suite passes in 60.580s, and fresh independent review passes
in 22.533s with no findings. Affected units, Go fast gates (1,334 inherited
diagnostics) and duplication (503 pairs) pass without baseline changes. This
checkpoint and the presentation contract await combined local deployment after
the shared frontend test run; the local API is still checkpoint 51.

### Checkpoint 54 — custom table columns and guarded refresh

Project, workspace, saved-view, module and cycle tables expose the authorized
property catalog in Display. Column preferences retain stable property and
cascading-level identities across navigation and saved-view serialization. Static
columns keep their existing sorting behavior; custom columns do not advertise
unsupported sorting. Table-only preferences do not affect board card sizing.

Cells render zero, false, effective Date formats, option paths/icons, references,
formula errors and safe rich-text previews. Unattached or foreign-context columns
show an explicit not-applicable value. Enter and Space open editable cells through
the existing coordinated property editor. Modal dismissal respects save, reload
and rich-text upload readiness; transient permission verification failures retain
the local draft, while actual revocation removes it.

A native Query overlay reads selected columns in batches of up to 100 items per
project, including loaded descendants. It never seeds from the shared item map,
which lacks reader provenance. Account, session, raw project membership role,
effective role, membership revision, item versions and configuration revision
identify the request. Responses cannot cross those boundaries, restore missing
items or pair values with another catalog revision. Projection responses now
include `properties_revision`. Ordinary item receipts trigger fresh reads through
the item-version key instead of retaining stale computed values.

Sub-item reads now capture the owning session, project/whole-workspace role
revision, parent location, parent-list contents and existing item versions.
Superseded requests are aborted per parent. Delayed responses cannot overwrite
newer saves, resurrect deleted children, or restore an in-place list removal.
Failures expose a retry affordance on the expansion control.

Fresh request/preference, rendering and query/privacy reviews found and corrected
optional metadata typing, nonapplicable cell presentation, keyboard activation,
modal dismissal, admin-to-project-guest changes and stale child-read rollback.
Focused final checks pass: 27 value/query/editor/sub-item tests, 18 UI/request/real
rich-text tests, and independent 15 frontend tests plus 8.893s database checks.
The revision/grouped-page database checks also pass locally in 9.420s. Full static
checks are being completed; this checkpoint does not complete S11 or the goal.

Local browser verification displayed zero/false, the configured Date format and
rich text; the preview opened the editor, saved 7/true refreshed immediately, and
columns/values survived reload. Evidence is in
`tmp/property-filter-live/table-edited-values.png` and `table-reload.png`.
All temporary definitions were removed, the test item moved to Trash, and original
project display/filter preferences restored. API checkpoint 54 is deployed; the
frontend watcher remained running throughout.
