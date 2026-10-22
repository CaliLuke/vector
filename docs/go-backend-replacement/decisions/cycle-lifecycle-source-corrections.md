# Cycle lifecycle source corrections

The pristine source baseline is pinned to Plane revision `e10b8aa6963c64ae99ae67a175d2e588a7fc7fab`. Its effective module hashes are in [cycle-lifecycle-source-hashes-pristine.json](cycle-lifecycle-source-hashes-pristine.json). The corrected capture uses the isolated snapshot used during the migration; its hashes are in [cycle-lifecycle-source-hashes-corrected.json](cycle-lifecycle-source-hashes-corrected.json). The historical Python correction patch was removed after the Go cutover.

The correction set makes these observable contracts explicit:

- A missing, cross-project, or soft-deleted legacy PATCH target returns 404.
- Draft archive attempts return 400 with `Only completed cycles can be archived` for both transports.
- Archived legacy item aliases bind the route's `pk`; archived collection routes without an ID return 400 with `Cycle ID is required` after authorization.
- Completed PATCH input is reduced to `sort_order` before serializer validation, duplicate external-ID checks, and model-activity recording. The v1 update serializer accepts `sort_order`.
- A v1 PATCH that omits `owned_by` keeps the existing owner. Explicit `owned_by: null` retains the source reset-to-requester behavior.
- Legacy cycle writes treat `created_by`, `updated_by`, and `deleted_at` as read-only. An attempted `deleted_at` update cannot bypass normal soft deletion and its queue effects.
- The corrected oracle uses the Python standard-library `zoneinfo.ZoneInfo` rules for cycle date parsing and response conversion. Pinned pytz 2024.1 stops carrying `America/Los_Angeles` transitions after 2037, which shifted a 2099 DST-boundary end date by one hour. This oracle-only correction preserves the existing start-time, end-of-day, and current-day boundary rules; the shipped shared Django helper remains unchanged.

The pristine fixtures are in `apps/go-api/internal/parity/testdata/fixtures/cycle_lifecycle/`; corrected fixtures are in the sibling `cycle_lifecycle_corrected/` directory. Both directories contain 112 fixtures: 22 corrected cases and 90 byte-identical cases. Three canonical missing-row cases are unchanged across source modes. Two archived-item alias missing-row cases preserve the pristine route-parameter 500 and record the corrected 404 under the accepted `pk` binding correction. The four focused `America/Los_Angeles` actor-timezone cases, five authenticated OPTIONS protocol cases, and four invalid-user-timezone OPTIONS cases are unchanged across source modes; their captured outputs and commands are recorded in `evidence/cycle-lifecycle/source/recording-metadata.json`. The following case IDs are the corrected fixture projection. The 14 focused review-repair cases are byte-identical across projections. Every other pristine case is expected to match the corrected capture unchanged.

```json
{
  "changed_fixture_cases": [
    "app-delete-archived-collection-missing-cycle-id",
    "app-delete-archived-item-missing",
    "app-delete-archived-item-pk-alias",
    "app-patch-archived-project",
    "app-patch-cross-project-cycle",
    "app-patch-dates-non-utc",
    "app-patch-deleted-at-bypass",
    "app-patch-detail-completed-sort-only",
    "app-patch-detail-missing",
    "app-patch-protected-audit-fields",
    "app-patch-soft-deleted-cycle",
    "app-post-archive-draft",
    "app-post-archived-collection-missing-cycle-id",
    "app-post-archived-item-missing",
    "app-post-archived-item-pk-alias",
    "app-post-create-date-non-utc",
    "v1-patch-dates-non-utc",
    "v1-patch-detail-completed-conflicting-external-id",
    "v1-patch-detail-completed-sort-only",
    "v1-patch-detail-current",
    "v1-patch-detail-preserves-owner",
    "v1-post-archive-draft"
  ],
  "unchanged_decision_cases": ["v1-patch-detail-explicit-null-owner", "app-delete-detail-original-creator"]
}
```

The pristine fixtures preserve the original failure responses and effects as source evidence. The corrected fixtures are recorded from the patched snapshot and are the only projection used for Go parity replay.
