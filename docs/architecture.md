# Vector architecture

## Runtime

The Go service owns the application API, authentication, database bootstrap, and durable background jobs.
Caddy routes API requests to Go and serves the browser applications through the same origin.
PostgreSQL stores application data and durable work. Redis provides cache services.
S3-compatible storage holds uploaded files. The Node.js collaboration service supports live editing.

The runtime does not require Django, Celery, or RabbitMQ.
Startup requires native background tasks. Legacy broker settings remain accepted
for configuration compatibility, but no AMQP publisher is constructed.
Static response fixtures remain for contract regression tests. No Python code or interpreter is required.

The initial [remote MCP transport](mcp.md) lives in `apps/go-api/mcp` and
`internal/mcptransport`. The proxy sends `/mcp` and `/mcp/*` to Go. The generated
catalogue contains 208 actions across 30 tools; MCP is disabled by default. All action contracts are wired. Current generated-server workflow evidence covers all 208 actions; full acceptance checks remain incomplete. The [parity tracker](mcp-parity.md) separates wiring from verified behavior. Native project creation, editing and deletion, cycle/module work-item assignment and transfer, and work-item link/comment writes use transactional receipts and the existing effect outbox. Association, link and comment revisions include changes from native HTTP writers. Workspace retrieval reads current membership and returns the selected workspace metadata. Workspace listing returns all active memberships. Both authentication modes support per-tool workspace selection, with an optional fixed header default.
`internal/mcpauth` supplies personal-token authentication using primary token and
workspace membership reads. `internal/mcpoauth` owns fresh external-issuer
introspection, verified account linking and expiring proof storage. `mcpoauthlinks`
defines the generated browser-link API. `internal/mcpactions` projects credential-neutral
application operations shared with HTTP. `internal/projectmembershipreads.DirectoryReader`
owns current workspace-directory authorization, filtering, paging and the existing-role catalogue.
Project reads use `internal/workspaceprojectreads` on the primary. Cycle, module, and label reads use primary `internal/cyclereads`, `internal/modulereads`, and `internal/projectlabelreads` instances and their shared API serializers. A shared `internal/djangoapi.ProjectMemberReadScopeSQL` binds authorization and resource data to one database snapshot; immutable workspace IDs prevent slug reassignment from changing the target. Comment and link reads reuse `internal/workitems`; activity reads reuse `internal/activityreads`. Their bound queries also recheck parent-item visibility and relationship ownership. `internal/djangoapi.Timestamps` supplies the shared HTTP/MCP timezone projection. All scoped MCP
reads retain the verified workspace UUID through the data query, preventing slug
reassignment from changing the selected tenant. Project archive and restore share a transactional writer with HTTP. Optional receipts live in `internal/mutationreceipts`; a database trigger advances project revisions for all project-row writers. Discovery filters generated action schemas using current scopes and workspace/project permissions without changing the shared catalogue.
OAuth is available through an explicitly configured external issuer; remaining resource actions
and full parity are incomplete.

`internal/projectstatereads` also supplies the [state contract](mcp-states.md),
with project reads and workspace aggregation over accessible project states in
one authorization snapshot. HTTP and MCP mutations share native helpers, all-writer
revision triggers, receipts, and a transactional deletion outbox. The generated
state tool exposes five actions; workspace writes return workspace_not_managed.

The [PQL query boundary](pql.md) lives in `internal/issuequery` and executes through the shared `internal/workitems` list path before counting and pagination. It supports scalar, relationship (including native milestones), member, body and effective custom-property predicates. MCP binds immutable workspace identity and current access inside the read snapshot; property queries use the authorized property-discovery snapshot. Native compiler and PostgreSQL tests cover language semantics. The [planning query acceptance](mcp-cycles.md#planning-work-item-query-acceptance) records generated-server evidence and its publication status. The action tracker retains the remaining acceptance requirements; teamspaces remain deferred.

`internal/milestones` owns the [native milestone operations](mcp-milestones.md): consistent scoped reads and transactional record/association changes, receipts, revisions and outbox insertion. The webhook worker supports explicit milestone subscriptions and immutable workspace binding. Item-activity and notification delivery preserve milestone snapshots and recheck queued email scope. Existing history, profile, export, activity counts and notification readers share current milestone visibility checks; notification writes recheck the same scope. The generated native HTTP routes use primary API-key validation, immutable workspace admission and the shared operations. The generated MCP tool mounts all seven operations through the same repository, with project-scoped discovery, pinned output shapes and optional receipt/revision extensions.

## Source layout

Draft creation, editing, deletion and publication belong to `draftLifecycle` and
`draftOperation` in `internal/workitems`. The lifecycle owns standalone transactions;
reviewed changes bind the operation to their existing savepoint and review state.
Drafts use database-bound authorization helpers and narrow item-validation and
item-creation capabilities. They do not copy repository query state or unrelated
mutation flags. Activity and cleanup effects go directly to the transaction's
outbox, including on publication and deletion.

HTTP composition lives in `apps/go-api/cmd/system/http.go`. It registers domains
in order through their `*_routes.go` owners, then applies the shared request
pipeline in `http_middleware.go`. `http_server.go` owns listening and shutdown.
Protocol matching, headers, CORS, encoding, and session helpers have separate
files; test-only endpoint constructors live in `http_test_helpers_test.go`.
Public metadata session effects remain request-local: audit logging captures the
inner response before cookie refresh failures can replace the final response.

Membership invitation operations live in `internal/projectmembershipreads`.
`invitations_commands.go` owns dispatch and shared request context; invitation
reads, creation, acceptance/join, and updates each have a separate file.
`workspace_member_mutations.go` owns workspace member changes, while
`membership_insert.go` shares membership persistence and `invitations_fields.go`
shares field validation. Each operation retains its authorization, transaction,
and effect boundaries.

| Path                                   | Purpose                                       |
| -------------------------------------- | --------------------------------------------- |
| `apps/go-api`                          | Go API, workers, bootstrap, schema, and tests |
| `apps/web`                             | Main React application                        |
| `apps/admin`                           | Instance administration                       |
| `apps/space`                           | Public project pages                          |
| `apps/live`                            | Collaboration service                         |
| `apps/proxy`                           | Caddy configuration                           |
| `packages`                             | Shared frontend code                          |
| `apps/go-api/internal/parity/testdata` | Historical contract fixtures                  |

The main web application uses React, Vite, and TanStack Router. `apps/web/routes.ts` defines the
virtual route tree; route modules own their components, redirects, metadata, and route parameters.
TanStack generates `apps/web/app/routeTree.gen.ts` and splits route components during builds.
Components use TanStack navigation APIs directly. Admin and public project pages use React Router APIs directly;
there are no Next.js routing adapters.
Shared searchable pickers use Headless UI's native Combobox open state. `ComboSearchInput` owns search-key
behavior: Escape clears a nonempty search before closing the picker, and an empty-search Escape restores
trigger focus. Calendar pickers use the existing Base UI Popover with controlled visibility. Creation-form
Escape shortcuts are scoped to their form element and ignore already-handled events, so nested pickers do
not dismiss the surrounding draft.

TanStack Query owns request scheduling, cache identity, retries, and query invalidation in all three browser
applications. Each mounted application provider creates its own QueryClient. Sign-out clears private query data
and remounts private React state. Separate provider instances never share private query clients.
Query keys are structured arrays that include request scope and parameters. Queries use `skipToken` until their
required inputs are available. Shared keys live in `packages/constants/src/query-keys.ts`. Axios is the transport;
new query functions pass the cancellation signal to it.

TanStack DB owns normalized views of complete record collections through the shared
`useRecordCollection` hook. One Query-backed collection and ordered live view belong to
each exact Query instance; consumers share its canonical Query cache. Native Query observers retain per-consumer
freshness and retry policy; the DB observer synchronizes that cache without fetching. Replacing a query
retires that collection, even when the replacement has the same key. On-demand sync
subscribes while a live view is mounted and preserves the pre-existing Query cache on
teardown, so an accepted write can still reconcile after navigation. Full-list endpoints
keep their existing keys and apply ordering locally. Records are wrapped internally so
DB metadata cannot leak into forms or API payloads. Failed reads hide current lookups;
recovery restarts a failed initial live graph after Query accepts a successful result.

This covers workspace/project catalogues, member rosters, states, labels, estimates,
cycles, modules, favorites, views, pages and page versions, activity, comments, reactions, attachments,
links, home widgets, recents, quick links, invitations, integrations, API tokens, webhooks,
public lookups and admin configurations. Existing scoped mutation options remain the
write boundary. Cursor pages, singleton details, permissions, searches, and atomic
schema/statistics envelopes stay in Query; splitting their rows would lose pagination,
revision, or authorization context. A collection is a view of the canonical transport
cache, not a second mutable server-data owner.

TanStack Store owns the web session generation, item-editor ownership/status and sidebar
UI preferences. Session stores belong to each QueryClient, editor stores also include the
session and item scope, and sidebar stores belong to the mounted session provider.
React selectors subscribe to the relevant state; listener sets and the per-field sidebar
context tree are gone. Form/editor drafts and server mutation receipts keep their existing
owners. Other local component state stays in React.

TanStack Virtual measures the accumulating notification and intake rows through one
shared list component. Record IDs preserve identity through reorder and deletion; the
focused row stays mounted when it leaves the visible range, including its portaled menus.
Existing scroll containers and pagination controls remain in charge of navigation and
loading. Lists expose native list markup and each row's position and total count. The
virtual list opts out of React Compiler memoization while
[Virtual issue #736](https://github.com/TanStack/virtual/issues/736) remains unresolved;
[issue #1119](https://github.com/TanStack/virtual/issues/1119) tracks the diagnostic that persists
despite this opt-out. Remove the directive and its local diagnostic exception when a
released adapter supports the compiler.

Admin no longer uses MobX. Instance information, configurations, authenticated users, and paginated workspaces
live in Query. React owns sidebar collapse and dialog visibility; next-themes owns the theme. OAuth providers
share one form and settings page, with provider-specific fields and callback instructions supplied as data.
TanStack Form owns admin field values, validation, submission state, and saved-value comparisons. Shared field
components use its native form contexts. Configuration mutations preserve failed drafts and reset defaults only
after successful saves. SMTP security is one form choice mapped to mutually exclusive API flags; test emails use
saved configurations. Admin has no React Hook Form dependency.

The static browser apps render next-themes' bootstrap script through React, which triggers the client-rendered
script warning tracked in [next-themes issue #385](https://github.com/pacocoursey/next-themes/issues/385).
Theme changes still apply through the provider's client effects. Remove this note when an upstream release avoids
rendering the bootstrap script on the client.

Web instance information, notifications and stickies use Query directly; home widgets and quick links use
Query-backed DB collections. Query remains their canonical server-data cache. Instance consumers share a cancellable query through the existing instance service;
there is no instance store or store hook. The instance gate renders loading, maintenance, or setup from
the query state, and cached configuration updates reach authentication, onboarding, uploads, and integrations.
Notification filters, selected notifications, widget settings visibility, and sticky searches live in scoped React
providers. Mutations update or invalidate their workspace's cache; sticky autosaves serialize per sticky.

Current-user identity, profile, account settings and email notification preferences live in native Query caches.
Profile and settings keys include the account ID, and canonical responses are checked against that account.
Account and profile writes serialize within their respective mutation scopes. Sign-out confirms server session
revocation before clearing private caches. A session generation remounts private
React state, including unsaved Form drafts; late mutations cannot restore a previous session's data. Authentication
waits for profile, settings and workspace bootstrap before choosing the destination and exposes failed loads for retry.
Page permissions and member ordering receive the current-user ID explicitly; calendar views receive the native
profile's week-start preference.

Command-palette dialogs, Power K navigation, and command registration live in React contexts beneath the
session boundary. Shortcut registration belongs to its mounted provider; cleanup cannot remove a newer
registration. Signing out clears open dialogs and command context with the other private UI state.

Workspace membership and project roles live in native Query caches scoped to the account and workspace. Project-role
keys also include the canonical workspace membership revision. Pending, inactive, and failed permission reads deny
access; project metadata and member rosters do not grant current-user permissions. Page policies receive these roles
explicitly. Joined-project selectors combine native membership with native project catalogues for ordering and display.
Join and leave actions use native mutations. Accepted writes reconcile captured queries only while their account,
session, and query identities still match; joining waits for canonical roles instead of inferring access. Project
metadata comes from separate native catalogue and detail queries and never grants permission.

New scoped reads and writes require a successfully verified current-user query. An accepted
work-item receipt can still reconcile after a transient verification failure when its account,
session, permission query instances, membership revision and roles remain unchanged. Explicit
authorization failures reject reconciliation even when Query retains older permission data.
Canonical receipt application precedes ancillary refreshes; those refreshes retain their own
read guards and errors, so their failure does not turn an accepted save into a retryable write.

Member settings, selection controls, issue grouping, and migrated summary displays use native workspace roster queries. Their keys include the account,
workspace, membership revision, and guest/full response projection, so a restricted response cannot inherit
private fields from an earlier full roster. Project roster queries additionally capture the project role and
require current canonical access. Both reads forward cancellation and reject results after the session,
permission revision, or query instance changes. Hooks expose roster records only after successful reads;
disabling a project roster hides its derived members without discarding its cached response. Generic member
summaries derive from these authorized roster projections; no legacy member store remains.
Workspace invitations use a separate scoped query. Roster and invitation mutations capture their original session and
query identities, reconcile only still-current data, and preserve accepted writes when a follow-up refresh fails.
Project roster writes serialize per project; workspace member writes serialize per member. Role changes enforce
canonical workspace and project permissions, including actor rank and guest role limits. Member-table filters,
searches, and unsent invitation dialogs live in React state scoped to the workspace or project. Failed prerequisite
permission reads expose retry controls instead of appearing as empty member lists.

Workspace catalogue records use Query-backed DB collections, keyed by the current account. Screens select workspace
metadata by route slug or record ID from the shared list; a refresh replaces the full catalogue so removed
memberships do not linger. Create, update and delete mutations preserve canonical server records and guard
cache writes against replaced queries and account sessions. Project, label, view and favorite selectors receive
workspace identity explicitly.

Workspace navigation preferences use account- and workspace-scoped Query data. Native mutation scopes serialize
writes within each session; pending mutation variables provide the shared optimistic display. Each action sends
only changed fields, so a failed reorder cannot leak into a later pin update. Sidebar acknowledgements merge
accepted fields before refreshing; workspace-property responses replace the canonical record. Session and query
identity checks prevent late saves from restoring cleared data. Immediate failures show scoped feedback, while
the project-count Form keeps its draft for retry. App-rail display preferences remain browser-local.

Issue notification subscriptions use user- and work-item-scoped Query data and serialized native mutations.
Changing the work item discards pending feedback, and late saves cannot overwrite or cancel a replacement query.

Personal API-token records live in Query, with cancellable list requests and native create/delete mutations.
Mutations update the current session's list only; responses from cleared sessions cannot restore private data.
The create dialog owns the one-time secret, and list cache entries omit it. Closing the dialog prevents a late
response from displaying or downloading its secret. Failed token submissions retain their form drafts.

Workspace webhook records also live in Query, with cancellable list/detail reads and serialized
update, secret-regeneration, and delete mutations per webhook. Cache writes apply only to captured
queries that still belong to the current session. One-time secrets stay in the active dialog/page,
outside query and mutation results; changing scope or closing the view invalidates late feedback
and secret downloads. Failed webhook edits retain their native Form drafts. Workspace membership
and permission checks use the shared native permission queries.

Web editor uploads and asset duplication use native mutations. Each editor owns its upload progress and
abort controllers; changing workspace, project, or entity cancels that editor's pending operations. Upload
cancellation propagates through signing, object storage, and finalization. Mutations do not automatically
retry asset creation. Editors declare their asset scope and receive completed asset IDs for form attachment
finalization. Image actions read the current file handlers from editor storage, and concurrent image uploads
keep the editor busy until all active upload controls finish. There is no root editor asset store.

Editor link creation and editing use native TanStack Form drafts, validation and submission errors.
Drafts reset when the editor or edited link changes. Link edits preserve unrelated marks and selection;
stale link targets are rejected without changing the document. Failed commands retain the editable draft.

Issue titles use native Form drafts and serialized Query autosaves in detail, peek and intake views.
Title and description writes share an issue-scoped mutation queue. Each editor reports its own save
status; the header and navigation guard remain unsaved until both editors succeed. Failed drafts can
be retried, and leaving an editor flushes an unsent revision without duplicating a pending retry.
Native issue queries own the server records. The text-save boundary propagates primary write
failures and keeps ancillary activity refresh failures separate.

Avatar, workspace-logo and cover selection use TanStack Form with native upload/deletion mutations.
Avatar and logo dialogs share one form component and receive their owner callbacks explicitly; the workspace
modal has no router or store dependency. Successful asset operations are retained while a failed owner update
is retried, so retrying does not create or delete the same asset again. User uploads share the cancellable
signing/storage/finalization transport. Preview components release their object URLs on replacement or unmount.
Avatar changes save independently of unrelated profile drafts.
Asset finalization is idempotent for avatars, covers and workspace logos: confirming the current asset does not
retire it. Deletion clears an owner reference only when it still points to the deleted asset. Database regressions
cover replacement, repeated confirmation, deleting unused/current uploads and rejecting another owner.

The existing-work-item selection dialog uses a cancellable project search query and native Form submission.
Selections persist across searches and failed submissions, and selecting all affects only the visible results.
Changing the workspace/project or relation scope resets the dialog and cancels the old search; an old submission
cannot close the replacement dialog. The unused project multi-selection dialog has been removed.

Public pages no longer use MobX. Publishing settings, authenticated users, lookups, grouped issues, paginated
state lists, issue details, votes, reactions, and comments live in scoped Query caches. The URL owns layout and
filters; React owns peek mode and editor drafts. Failed mutations retain published data and editable drafts.
Comment attachment finalization can retry independently without posting the comment again.

Generic member identity now comes from workspace-scoped roster queries. Page authors,
activity actors, attachment/link attribution, editor mentions, and assignee-name ordering
use those query projections; there is no generic member loader or member root store.
Historical attribution retains inactive members returned by the authorized roster response.
Editor plugins read current committed identity callbacks without recreating the editor or
its undo history. Page exports wait for roster readiness and preserve failed Form submissions
for retry.

Project tab preferences use account-, workspace- and project-scoped Query data. The
active project wrapper owns loading; sidebar rows observe cached preferences without
starting per-row requests. Native mutation scopes serialize writes, and pending
variables provide the visible overlay. Hide/show actions compose against the latest
accepted preferences so failed changes cannot leak into later saves. Late responses
cannot restore replaced queries or cleared sessions. Project issue layouts also read these
native preferences and pass explicit filter snapshots to list requests. Mutations reconcile
the active project list only while its route, session, permission queries, preference query,
and request generation remain current. Off-view writes update their entities without altering
the active list. Failed preference reads invalidate captured list inputs even when older data
remains cached. Layouts reload from changed Query preferences; completing a save does not
imperatively refresh a list from the previous route. Project column collapse uses scoped
React state and the existing account/project local-storage format. Profile and archive filters
also use account-scoped React preferences, with stable request inputs. Cycle, module and
workspace-view preferences have their own explicit React scopes.

Favorites use an account- and workspace-scoped Query collection, including folder children.
Canonical membership and project-role revisions gate reads and writes; pending native mutations
provide the optimistic display without a second record store. Accepted entity deletions invalidate
favorites without turning a failed follow-up read into a failed deletion. The API validates entity,
project and folder ownership. Folder moves serialize on the actor's workspace membership before
checking ancestry, preventing concurrent reciprocal moves from creating cycles.

Workspace-view metadata uses a native Query collection scoped to the account, session,
workspace and canonical membership revision. Screens select exact view IDs from the
authorized list; create, update and delete mutations reconcile only captured current
queries. Retry recovers failed identity or membership reads before loading views.
Dialog completion checks its route, session and lifetime before navigating, closing or
updating local filters. The old global-view store, hook and workspace issue-filter
store are removed. Saved custom-view filters come from the native metadata query;
scoped React state owns working filter and display drafts. Static-view display
preferences persist under explicit account, workspace and view identities. Legacy
unowned local preferences are ignored without rewriting other domains' records.
Saving clears only the submitted, accepted draft; metadata-only edits preserve
unrelated working filters. Canonical editor resets do not mark pristine filters dirty.

Project-view metadata also uses native Query collections and detail records. Its keys
include the session, membership revision, and effective and actual project roles.
Accepted writes reconcile only the captured query instances; a replacement detail query
cannot receive an older write's response. Scoped React drafts own view filters and display
preferences. Calendar, group collapse, and Gantt consumers read those same preferences.
Pagination carries the immutable view input and preserves the accepted full-read generation.
Favorite rows request project-view details only for view entities.

Workspace issue reads receive explicit immutable filter snapshots. Read guards track
account, route, permissions and current inputs. Each accepted full read owns a pagination
generation; its groups have independent native infinite queries. Mutations capture exact
query instances before transport, cancel older reads, and apply accepted canonical records
before ancillary invalidation. The generic rich-filter editor owns its immutable expression
and unsaved draft in React.

Sidebar preferences and transient open/peek state live in a provider-scoped TanStack Store. Persisted choices retain
their browser-local storage keys across session remounts; transient values reset. Hydration preserves
viewport defaults already chosen by mounted consumers, so a saved desktop preference cannot reopen
the mobile sidebar. There is no sidebar ThemeStore.

Analytics reads and project counts live in native Query caches. Keys and response guards include
the account, session, workspace membership and project-role snapshot. Failed refreshes hide
cached totals and exports. React owns separate dashboard and modal filter drafts; scope changes
and closing a modal discard its draft. The modal's close control remains available while access
loads or fails. Project metadata and chart state names come from their native catalogues.

Project and workspace estimate catalogues live in native Query caches. Keys include the account,
session, membership revision, and both effective and actual project roles. The estimate models
and their legacy hooks are removed. Settings use native Form drafts and serialized mutations;
the native project detail query owns its selected-estimate pointer.
Deletion clears only a matching pointer and records durable cleanup in the same transaction.
Issue ordering and parent statistics receive explicit immutable point projections. Point-order
changes reorder loaded rows without reloading the issue list; numeric-value changes refresh
visible parent statistics through guarded reads. Failed catalogue reads expose retry controls
instead of presenting stale order or totals. Accepted mutations retain their captured access
lineage, while new and ancillary reads require verified access.

Cycle and module numeric totals, distributions, and burndown reads resolve only live estimate
points and live numeric systems in the issue's project and workspace. Soft deletion takes effect
in these reads immediately, before durable cleanup removes dependent rows. A still-live inactive
system continues to contribute points assigned to existing issues.

Project and workspace label catalogues live in native Query caches. Permission revisions and
session identity scope each catalogue; serialized mutations apply accepted canonical responses
only to their captured caches. Label names, tree order and group columns derive from those
catalogues. A changed label-ID set requires a guarded full
issue-list read before grouped rows display; pagination and mutation projections do not count
as that full-read receipt. Current issue labels require both a live relationship and a live
label in the same project and workspace across lists, details, activity expansion, analytics
and current-row exports. Draft labels use the same rule with nullable project ownership;
a projectless draft can retain only projectless labels in its workspace. Label writers serialize on the project row
and validate live, same-project ancestry before assigning a parent. Deletion marks the root and its descendants
and records one durable cleanup effect per changed label atomically. A tree with a foreign child reference,
including an already-deleted child, is rejected before any deletion or cleanup is admitted. Accepted deletes
remove the known branch from the native Query cache and refetch the canonical catalogue to reconcile concurrent moves.

Project, intake, and workspace states use native Query reads. The state store and its
global hook are removed. Keys include account/session identity, membership revision,
and project-role scope; failed access verification hides cached rows and exposes Retry.
State-name ordering and parent statistics consume explicit current projections instead
of a second state map. Serialized writes invalidate only the query instances captured
before submission, so an older completion cannot refresh replacement queries.

Project catalogues and details live in scoped Query caches, including partial navigation records and full
settings records. Native mutations reconcile captured cache instances and refresh canonical counts after
intake changes. React owns project-list filters; workspace changes clear search and temporary display flags
while preserving each workspace's saved sort order. Project classes and the old project hooks are removed.

Project pages use native list, detail and version queries. React contexts own the mounted editor reference,
assets and sync status. Optimistic writes preserve newer collaboration changes and roll back only their
captured value. Collaborative title persistence belongs to the collaboration service; editor events
update the visible Query records without issuing duplicate REST writes. Page mutations expose one operation
per hook; no page store or generic mutation facade remains.

Intake uses scoped infinite-list and detail queries, with React filter/sort preferences. Accepted status changes
retain their canonical detail response while list counts refresh. An acceptance editor remains open until its
following work-item edit succeeds; failed ancillary list reads do not discard that editor. The old inbox stores
and hook are removed.

Cycles and modules use native catalogues, details, links and analytics queries. Grouped issue layouts use
complete authorized active and archived catalogues. React owns catalogue and issue-list preferences. Parent
statistics use explicit captured Query owners and guarded rollback receipts. Successful removal writes put an
explicit null receipt in the captured detail query; the surviving route owner handles navigation after a row or
modal unmounts. Missing or denied reads do not imply accepted removal. Issue creation keeps its accepted ID
while failed cycle/module assignment steps retry, including inline creation.

Work-item details, identifier lookups, sub-items and every list family use native TanStack Query.
There is no RootStore, mutable issue entity map, or store-loader query facade. Query owns
records, pagination, cancellation and invalidation; React providers own transient layout,
peek and preference state. Native infinite-query observers retain each group's cursor and
loading state, and their React subscriptions end with the mounted list. Fresh queries reuse
the application freshness policy instead of refetching to initialize removed stores.
Primary write failures propagate to Form or mutation callers for retry. Issue PATCH
accepts the API's HTTP 204 response and invalidates its captured list and detail queries;
an empty success body is not a failed save. Responses containing a canonical row reconcile
that row before ancillary refreshes. A failed refresh does not repeat an accepted write.

The browser applications retain substantial Plane-derived code.
Existing API paths, database names, internal `@plane/*` imports, and configuration keys remain compatible.
The backend Go module retains its existing import path to preserve generated code.
These identifiers do not identify the product as an official Plane distribution.

## Contracts and jobs

Generated HTTP transports derive from checked-in designs. Domain services enforce access control and database constraints.
Use Loom's design, generated transport, validation, security hooks, and request
metadata facilities wherever they cover the requirement. Shared application
helpers run inside those service boundaries; they must not introduce parallel
HTTP routing, credential extraction, or transport validation paths.
Request transactions record durable work. Workers execute that work with retries and persisted status.
Do not replace durable effects with detached goroutines.

Draft publication records the item, relationships, asset ownership, consumed-draft
identity and effect outbox in one transaction. A lifecycle-owned worker admits
committed effects to the existing native workflows with stable IDs. Admission
status does not imply execution success; native workflow status owns completion.
Startup requires `--go-background-tasks` so effects cannot be stranded in the
retired legacy-only mode. Both deployment Compose files set this flag.

The route inventory records 639 retired Python application operations. Supplemental authentication redirects also run in Go.
Historical fixtures document compatibility and intentional corrections. They are not a second production backend.

All browser apps now use TanStack Form for the migrated editing and authentication
flows, and React Hook Form has been removed from dependencies. Native fields own
validation and draft values. Submission state locks edits and dismissal; failed
requests retain drafts. Creation workflows keep completed durable steps so retries
can finish attachment finalization or setup without creating the entity twice.
Estimate values use a native array field with stable row identities. Invitation
selection shares one form between onboarding and the invitations page. Shared UI
renders form errors and native form/modal frames without emulating a form library.

Project work-item searches include the account, session, workspace membership, project role and workspace role catalogue in their Query keys. Reads validate the same permission-query instances before and after transport; stale or cancelled results cannot repopulate replaced caches. Failed access verification hides cached choices and disables submission while retaining the selection for recovery. Parent and duplicate selection share a native Form and a scoped, cancellable Query.
They retain rejected selections and close after the save completes. Cycle transfer
uses a Query mutation; retries after a successful transfer refresh the affected
cycle data without repeating the transfer.

The view form owns metadata and filter drafts in native TanStack Form fields. Filter
nodes are stored in a flat, typed array with child indexes, then reconstructed into
the existing nested API expression at the editor and submission boundaries. This
preserves logical nesting, child order, operator keys and literal values while
allowing Form to track dirty state, reset defaults and retain failed submissions.
There is no separate React filter draft. The projection avoids the recursive field
path expansion reported in [TanStack Form #1830](https://github.com/TanStack/form/issues/1830);
it can be removed when a released native field API handles the nested expression
without TS2589. No dependency patches or compiler suppressions are used.

Calendar navigation dates are React state scoped to the identity session. Calendar
weeks derive from the current profile preference. Each mounted Kanban board owns
its drag state; per-card ownership releases a drag on teardown without clearing
another card's active drag.

Application attachment collection reads apply the shared item visibility predicate
in the SQL statement that returns metadata. An earlier project-membership check
cannot authorize a read after item visibility or membership changes. Raw attachment
snapshots used inside write workflows remain separate from authorized collection
reads.

Sub-item filter and display preferences are React state scoped to the identity
session, workspace, project, parent item, and service family. Filtering and grouping
derive from current item rows with pure helpers; preference state contains no
server records. Reset restores the default display properties for that scope.

Rich-filter editing no longer uses a global instance registry. Mounted React editors own
expressions, conversion errors, visibility and accepted save baselines. Pure shared
operations preserve nested logical groups, committed empty values and wire representations.
The route provider shares filter visibility/status and a stable clear action between the
header and list. Shared-state no longer depends on MobX.

Workspace drafts use scoped native list/detail queries and serialized mutations. Reviewed
edits capture account, session, membership and exact query owners before submitting;
late receipts cannot alter replacement queries. Accepted rows reconcile idempotently;
canonical list and membership refreshes own counts and cannot reject an accepted write.
Accepted deletion and publication keep a null detail receipt in the captured query, preserving
its identity while list and membership counts refresh. Reads started during a write are
cancelled before its accepted receipt is applied. Draft publication keeps the existing reviewed-edit identity and attachment retry flow.

Private attachment reads and mutations enforce issue, comment-parent, and page visibility
in their SQL statements. Page assets retain their exact project-link context. Attachment
activity and metadata effects commit with the primary database write. Irreversible legacy
attachment removal records the captured storage key for durable deletion; restorable asset
soft-deletes retain their storage objects.

MCP comment mutations validate mention targets against current active project membership, render mention chips only after receipt replay is ruled out, and enqueue notifications through the native activity workflow. Link title previews use the same transaction outbox and a stable workflow identity for admission retries. The native HTTP comment path retains its existing notification policy.

The SDK-shaped estimate API lives in `internal/projectconfiguration` beside the existing app editor operations. It resolves the selected estimate inside the mutation transaction, batches point creation atomically, and reuses native deletion and outbox cleanup. `project_estimate_revisions` covers all estimate/point writers and project selection changes. The MCP projection delegates estimate linking to the existing project update boundary. All native estimate writers acquire the workspace schema lock and then a shared workspace row lock before changing catalogue rows or revisions. The row lock orders workspace deletion after in-flight estimate writes. Aggregate edits and point creation/edits capture their response snapshots in the same transaction and recheck the active parent chain after waiting for that lock. Estimate activity resolves project-bound retained catalogue rows, including soft-deleted labels, and appends new history rows after existing activity fields to preserve replay identities.

`internal/workitemtypes` also exposes SDK-shaped type reads and mutations for MCP. Reads bind the active actor and immutable workspace in a repeatable-read snapshot. Mutations hold the existing workspace schema lock, current membership locks, and optional retry receipts while calling native save, removal, and project-import helpers. `work_item_type_api_revisions` tracks all type, project-link, and settings writers independently of the native editor revision. API mutations advance the native editor revision when the catalogue changes. Type resolution determines the currently required permissions before receipt lookup, and explicit project links require current project-admin authority. Type response timestamps are serialized before receipts are saved, preserving the accepted representation on replay. Native schema writers also retain workspace/project and membership rows after the schema lock and authorization recheck, so deletion or revocation waits for an admitted write to finish.

Core MCP work-item mutations use `internal/workitems.MutateWorkitem` to run native issue writes and archive operations inside the shared project receipt transaction. `issue_edit_revisions` tracks issue rows and assignment/label relationships. Assignment set changes read the current set while holding the item and workspace schema locks, then apply removals followed by additions through the native PATCH path. Activities, webhooks, and deletion effects remain transactional.

Core MCP work-item reads use `internal/workitems.ReadWorkitem` for workspace/project lists, archived items, direct and identifier lookup, search, and cycle/module item pages. Current identity, immutable workspace scope, candidate visibility, PQL, page counts and expanded relations share one primary-data snapshot. The transport projects sparse pages and returns correctable PQL failures with the reference. Grouped count reuses the same filtered query and aggregates distinct item identities across native dimensions and workspace release associations. The generated-server core workflow batch covers these reads under both authentication modes and protocol revisions.

MCP attachments use `internal/assets.AttachmentApplication` with the shared FileAsset repository and object store. Reads bind current identity, known workspace/project membership and item visibility to a repeatable-read snapshot. Mutations use project receipts and all-writer asset revisions. URL import commits bounded bytes and an immutable key in `file_asset_upload_intents`, with an `asset_storage_upload` effect, before object storage I/O. Request completion and the effect worker share idempotent upload completion; publication activity and metadata effects are enqueued after storage succeeds. Pending uploads block object-key replacement and physical asset purge. The existing pinned HTTP transport handles public URL fetching. The attachment workflow batch covers real object storage, receipt replay, inline content and private-address rejection. Local browser checks also cover downloaded bytes.

`internal/worklogs` owns time-entry reads and mutations. Composite item ownership follows native item moves; all row writers advance per-log revisions. Current scope and author/administrator authorization run before receipt replay. Changes enqueue immutable `work_log.activity.*` snapshots for the existing activity worker; no-op edits enqueue nothing. The generated-server core workflow batch covers all four work-log actions and project duration summaries.

`internal/workitemrelations` owns dependency edges and workspace custom relation definitions. Current endpoint permissions precede receipt lookup. Mutations lock all endpoint projects and items, validate complete batches, and persist per-endpoint activity events. A workspace relation revision covers all dependency, definition and custom-edge writers. The migration permits multiple dependency kinds per pair; rollback rejects data that cannot fit the older unique pair constraint. The generated-server core workflow batch covers these operations under both authentication modes and protocol revisions.

Property catalogue MCP operations live beside native property mutations in
`internal/workitemtypes`. They reuse save, attachment and preservation validation,
keep API-only metadata in a separate column, and advance workspace property tokens
through all definition/association writers. Item value API actions reuse native
item mutation and history. A companion identity table stores stable API value IDs
and external metadata; canonical values remain in issues.custom_properties and
all item writers synchronize those identities. Work-item references validate
current target visibility. The property configuration batch covers formulas, fixed content, cascading options and preservation rejection. The parity tracker records remaining compatibility checks. Email and File share native bounded-text storage;
Email adds address validation and both use text query operators.

Intake MCP operations use `internal/workitems.ReadIntake` and `MutateIntake`. They bind queue visibility to current scope, reuse native submission and triage transitions, and authorize current action rights before receipt replay. Intake association writers share the item revision counter. Item HTML uses native asset admission, and activity is persisted through the existing transactional effect adapter. The generated-server core workflow batch covers these operations under both authentication modes and protocol revisions.

`internal/releases` owns workspace releases, version tags, labels, changelogs and item associations. Workspace releases extend `work_item_releases` with nullable project ownership; existing project catalogue APIs keep their original scope. Native property choices, validation, filtering and preservation recognize live workspace releases. Catalogue mutations persist audit snapshots and association mutations reuse the native durable relation-activity worker. All release, tag, label, changelog and association writers advance a workspace revision; item associations also advance item revisions. The generated-server content workflow batch covers release, tag, label, changelog and association lifecycles under both authentication modes and protocol revisions.

Workspace pages share the native pages table and body-history workflows. Collections own workspace-only hierarchy filing and view/comment/edit memberships; private page ownership remains authoritative. Page transactions and versions enter the existing effect outbox, while item-page associations require both item and page visibility. See [pages and collections](mcp-pages-collections.md).

Native templates own reusable JSON definitions, top-level merge updates and workspace/project governance. Feature settings map existing project switches to their canonical columns and own workspace switches. Initiatives enforce their workspace switch, current target visibility, atomic associations and durable item history. Their contracts are recorded in [templates](mcp-templates.md) and [features/initiatives](mcp-features-initiatives.md).

Customers, requests, property definitions/options and values share a workspace catalogue in `internal/customers`. Atomic upserts preserve external identity; request associations remain customer-bound. Current workspace features, membership and reference visibility govern reads and receipt replay. Item links enqueue native activity. The catalogue workflow and receipt batches cover these operations. The [customer contracts](mcp-customers.md) and [parity tracker](mcp-parity.md) record compatibility decisions and remaining acceptance checks.
