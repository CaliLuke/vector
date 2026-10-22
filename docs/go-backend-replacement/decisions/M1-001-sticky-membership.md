# M1-001: Require Active Membership for Sticky Access

Status: Review ready for the M1 Go cutover.

## Source behavior

The Django app retrieve route restricts a Sticky by its workspace and owner. It does not require an active workspace membership.

The `retrieve-owner-without-membership.json` fixture records a `200` response for this case. The other app operations require an active membership.

## Go behavior

Every Go Sticky operation requires an active, non-deleted `workspace_members` row. This rule applies to list, create, retrieve, update, sort, and delete.

Each repository scope includes the workspace and authenticated user. Existing Sticky queries also restrict rows to the authenticated owner.

Update, sort, and delete operations also require the authenticated user to be the creator. Create assigns this user as owner and creator.

The future retrieve handler returns `404` when the membership is absent. This response does not reveal whether the Sticky exists.

## Reason for the difference

The Django retrieve behavior permits access after membership removal. The Go behavior closes this authorization gap before the route cutover.

## Regression evidence

`internal/stickies/access_test.go` requires membership for all six operations. It also records the owner and creator predicates for each operation.

`internal/stickies/access_integration_test.go` proves the PostgreSQL predicate with active, inactive, deleted, missing, and cross-workspace memberships.

## Frontend effect

No frontend change is required. A removed member can no longer retrieve an old Sticky through a saved URL.

The Django route remains active while `GO_STICKIES_ENABLED=false`. The security change starts only when the Go Sticky route is enabled.
