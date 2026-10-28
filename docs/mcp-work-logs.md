# Work-log contract

Vector owns time entries in `internal/worklogs`. The MCP `work_log` actions use that application boundary. Time is stored as positive integer minutes; the pinned handler omits zero durations and empty descriptions on update. The list result is a flat array, with native cursor/per-page selection, as in the pinned SDK.

A work log belongs to exactly one work item, project and workspace. Composite foreign keys enforce that ownership, including native item moves. The server sets the logged-by and audit identities. Current project members can read logs on visible published items. Project members and administrators can create logs; a log's author or a current project administrator can update or delete it. Unknown or inactive membership roles grant no access. Archived projects are unavailable. Item visibility and all selected records use the same read snapshot.

Mutations take the workspace schema lock and current project/item scope before changing a record. Optional request receipts and expected revisions use the shared mutation coordinator. Authorization runs again before a receipt replay, including the author/administrator rule against retained soft-deleted records. Every row writer advances its revision. Listing with `include_revision` returns current tokens; deletion retains the row until its parent is purged.

Creation, changes and deletion enqueue immutable work-log activity snapshots in the native work-item effect queue. The native activity worker renders those events with stable event identities. No-op updates produce no extra activity or revision. Time-entry changes do not send subscriber notifications.

The project `worklog_summary` action groups live time entries by visible work item in the same authorized snapshot. It returns seconds as specified by the pinned SDK (`sum(duration) * 60`), while individual entries use minutes. Items with no live entries have no summary row.

This contract adds the four MCP work-log actions and the project summary and a reusable native domain. Dedicated HTTP routes and UI editors are outside this MCP task. Implementation, SQL migrations, both transport revisions and authentication modes, permission/replay races, and native worker delivery still require combined validation.
