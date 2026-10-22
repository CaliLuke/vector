# ASSET-005: bind API attachments to the authorized issue

Status: accepted security correction; focused handler and PostgreSQL regressions pass.
Final composed proof and fresh review passed on repair commit `52f40b1738ab1c69fdb47826a246afe3c1f81591`; see the [final acceptance](../evidence/post-retirement-review/final-acceptance/acceptance.md).

The post-retirement audit reproduced unsafe behavior preserved from Django:
API-key nonmembers can list attachment metadata, detail GET ignores the issue ID,
and PATCH/DELETE authorize one issue then mutate another issue's project-scoped
asset. Both API v1 aliases share these handlers. Existing source fixtures remain
immutable evidence of the defect, not the required product contract.

Every API attachment operation must authorize the requested issue within the
workspace/project. Detail lookup, upload confirmation and deletion must bind the
asset to that same issue. Rejected requests must not sign downloads, publish jobs
or mutate rows. Valid attachment response shapes and native durable effects are
preserved. Reuse the existing issue-scoped repository lookup and writes.

See [audit](../evidence/post-retirement-review/auth-effects.md) and
[campaign ledger](../post-retirement-review-ledger.md) for callers, regressions and
final acceptance. No operation ownership changes and no historical Django replay
are required for this post-retirement repair.
