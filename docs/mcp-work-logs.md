# Work-log contract

Vector owns time entries in `internal/worklogs`. The MCP `work_log` actions use that application boundary. Time is stored as positive integer minutes; the pinned handler omits zero durations and empty descriptions on update. The list result is a flat array, with native cursor/per-page selection, as in the pinned SDK.

A work log belongs to exactly one work item, project and workspace. Composite foreign keys enforce that ownership, including native item moves between projects in the same workspace. A work item cannot change its workspace. The server sets the logged-by and audit identities. Current project members can read logs on visible published items. Project members and administrators can create logs; a log's author or a current project administrator can update or delete it. Unknown or inactive membership roles grant no access. Archived projects are unavailable. Item visibility and all selected records use the same read snapshot.

Mutations take the workspace schema lock and current project/item scope before changing a record. Optional request receipts and expected revisions use the shared mutation coordinator. Authorization runs again before a receipt replay, including the author/administrator rule against retained soft-deleted records. Every row writer advances its revision. Listing with `include_revision` returns current tokens; deletion retains the row until its parent is purged.

Creation, changes and deletion enqueue immutable work-log activity snapshots in the native work-item effect queue. The native activity worker renders those events with stable event identities. No-op updates produce no extra activity or revision. Time-entry changes do not send subscriber notifications.

The project `worklog_summary` action groups live time entries by visible work item in the same authorized snapshot. It returns seconds as specified by the pinned SDK (`sum(duration) * 60`), while individual entries use minutes. Items with no live entries have no summary row.

This contract adds the four MCP work-log actions and the project summary and a reusable native domain. Dedicated HTTP routes and UI editors are outside this MCP task. Acceptance tests cover both transport revisions and authentication modes with real PostgreSQL:

- [Receipts and native revisions](../apps/go-api/internal/mcpactions/work_log_receipts_integration_test.go) cover replay, changed inputs, no-op edits, native writes, stale revisions and deletion.
- [Current membership and author/admin rules](../apps/go-api/internal/mcpactions/work_log_permissions_integration_test.go) and [authorization after lock waits](../apps/go-api/internal/mcpactions/work_log_authorization_wait_integration_test.go) cover fresh writes and replay.
- [Visibility, ownership and summary units](../apps/go-api/internal/mcpactions/work_log_visibility_integration_test.go) cover pagination, tenant boundaries, parent moves and aggregate seconds beyond the individual duration range.
- [Contention, cancellation and invalid inputs](../apps/go-api/internal/mcpactions/work_log_concurrency_integration_test.go) and [storage rollback](../apps/go-api/internal/mcpactions/work_log_rollback_integration_test.go) verify domain rows, revisions, receipts and effects together. The shared receipt coordinator separately covers uncertain commit outcomes.
- [Native worker delivery](../apps/go-api/internal/mcpactions/work_log_worker_integration_test.go) consumes committed activity snapshots after deletion, checks stable event identities and verifies that delivery creates no notifications or email.
