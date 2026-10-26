# Sync a Notion roadmap to Vector

## Team pilot

Use `--team '<Notion team URL or UUID>' --snapshot /absolute/path/pages.private.json`
to import one team as one Vector project. Its roadmap items become modules,
linked Notion projects become parent work items, and their tasks become sub-items.
Each roadmap's `👷 Team` relation determines selection. A roadmap shared by teams
is included when the selected team appears in that relation.

The snapshot must contain the team page, all selected roadmaps, and every linked
project and task, including explicit empty relation arrays. Team mode currently
requires a fresh connector snapshot; it cannot poll. It cannot combine with
`--roadmap`, `--roadmap-only`, or `--include-parents`. Use a new journal for the
team mapping; a previous roadmap journal is deliberately incompatible.

A task linked to multiple projects is imported once, belongs to every applicable
roadmap module, and retains links to its source projects in its description. It
has no parent because Vector permits only one. Other tasks are placed directly
under their source project, replacing the source task-parent hierarchy in this
mapping. No source records are changed.

## Single-roadmap mapping

The `notion-sync` command imports one roadmap through Vector's public API. It can
run once or poll for changes. It does not write to Notion.

| Notion object                               | Vector object    |
| ------------------------------------------- | ---------------- |
| Selected roadmap                            | Project          |
| Pages in its `🏗️ Projects` relation         | Modules          |
| Pages in each project's `🧑‍💻 Tasks` relation | Work items       |
| Task `Parent` relation within the selection | Parent work item |

A task linked to several selected projects becomes one work item with several
module memberships. `--include-parents` also imports ancestors, without importing
their other children. Ancestors outside the selection are otherwise reported.

## Access and configuration

Give a Notion integration read access to the Roadmaps, Projects, and Tasks
databases. Sharing only Roadmaps does not grant access to related databases.
On each database, use its connections menu to add the integration. The public
Notion API and CLI cannot grant that access themselves.

Store `NOTION_TOKEN` in the ignored `.env.notion.local` file and `VECTOR_API_KEY`
in the ignored `.env.test.local` file. Restrict both files to mode `0600`. The
Vector key's owner needs access to the destination workspace and permission to
create projects, modules, states, and work items.

From the repository root:

```sh
source scripts/dev-env.sh
set -a
source .env.notion.local
source .env.test.local
set +a
go -C apps/go-api run ./cmd/notion-sync \
  --roadmap '<Notion roadmap URL or UUID>' \
  --workspace '<Vector workspace slug>' \
  --state "$PWD/tmp/notion-sync/state.private.json"
```

The default is a preview. Add `--apply` to import, and add `--interval 5m` or a
longer interval to poll. `--vector` defaults to `http://localhost:8000`; remote
origins require HTTPS. Polling runs a fresh traversal after each interval, so it
also detects changes to linked tasks and relations. The command exits on an
error or conflict; investigate it before restarting with the same journal.

Vector requests default to 1.1 seconds apart (below 60 requests per minute).
`--request-interval` can match a different server rate limit. For example, use
`--request-interval 110ms` only when the server permits at least 600 requests per
minute. Restore any temporary local server limit after a bulk import.

If only Roadmaps access is available, add `--roadmap-only`. This imports the
selected roadmap as a Vector project without reading linked Projects or Tasks.
After those databases become accessible, omit this flag and reuse the same
journal to add modules and work items to the existing Vector project.

For an initial import, an authorized connected Notion account can supply a
snapshot when the standalone integration cannot read the related databases.
Use `--snapshot /absolute/path/pages.private.json` instead of `NOTION_TOKEN`.
The file is a JSON array of normalized `notionsync.Page` records with canonical
page IDs, titles, sanitized HTML, status, priority, dates, people, and named
relations. Export every page reachable in the selection; missing pages stop
the import before writes. Explicitly include empty relation arrays. Preserve
source links and report any omitted content in each page's `warnings`.
Keep snapshots ignored and mode `0600`; they contain private source content.
Reuse the same journal so later API-based sync updates the imported objects.
Snapshots cannot use `--interval`: export fresh source data for each run.

An optional `--users /absolute/path/users.private.json` accepts a JSON object
mapping Notion user UUIDs to Vector member UUIDs. If any task owner is unmapped,
the command reports it and leaves that task's assignees unchanged.

## Imported fields

The command imports names, descriptions, source links, task status and priority,
dates, and mapped task assignees. Task status names become Vector states. Common
statuses map to Vector's backlog, unstarted, started, completed, or cancelled
groups; unfamiliar names use backlog. Module statuses use the closest supported
status, with the original status retained in the description. Roadmap status is
retained in its project description. Project and module owners are not imported.

Task priorities map P0 to urgent, P1 to high, P2 to medium, and P3/P4 to low.
Other priorities use none. Dates use `Timeline`, then `Date`, then `ETA`; ETA's
start date becomes the target date. Time-of-day information is not imported.

Task descriptions retain supported rich text and nested blocks. Project and
module descriptions are plain text. Unsupported blocks, including attachments,
produce warnings; their contents remain available through the Notion source
link. Comments, attachments, database views, and arbitrary custom properties are
not synchronized.

## Identity, conflicts, and recovery

Keep the private journal and reuse its path on every run. It is bound to the
roadmap, Vector origin, and workspace, and locked against concurrent use. Back it
up as private data: it contains source content and last-synchronized field values.
Use one journal per selection and avoid overlapping runners for that selection.

Destination records carry `external_source=notion` and the Notion page UUID as
`external_id`. Repeated runs find existing records. Pending writes are saved
before HTTP mutations so an interrupted run can reconcile committed changes.
Do not delete the journal to resolve an error.

If a managed Vector field differs from the last synchronized value, the command
reports a conflict and preserves that object. Other objects may still update.
Resolve the field deliberately in Notion or Vector, then rerun. Matching the
current Notion value resolves a field conflict. Restoring a locally removed
module membership allows that relationship to synchronize again.

Archived or deselected source records are retained in Vector and reported.
Removed task-module links are removed only if previously managed by this sync;
unrelated manual memberships remain. A deleted mapped destination produces an
error instead of silently recreating it. This first version provides no automatic
deletion or bidirectional conflict resolution.
