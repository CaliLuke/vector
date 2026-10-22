# M1-002: Lock the Workspace for Sticky Sort Order

Status: Review ready for the M1 Go cutover.

## Source behavior

Django reads the largest active Sticky sort order without a lock. The first Sticky uses `65535`. Each later Sticky adds `10000`.

Two concurrent creates can read the same value. Both creates can then store the same sort order.

## Go behavior

The Go repository starts one database transaction. It locks the active workspace row before it reads the largest active Sticky sort order.

The repository checks active membership inside the same transaction. Then it creates the Sticky before it releases the workspace lock.

The first Sticky still uses `65535`. Each later Sticky still adds `10000`.

## Reason for the difference

The workspace lock prevents duplicate sort orders during concurrent creates. It also keeps the source sequence for each workspace.

## Regression evidence

`internal/stickies/sort_repository_test.go` records the source allocation rule and the required scope.

`internal/stickies/sort_repository_integration_test.go` creates Stickies concurrently in disposable PostgreSQL. The result contains distinct and increasing sort orders.

The database proof also covers workspace isolation, inactive membership, soft-deleted Stickies, and transaction rollback.

## Frontend effect

No frontend change is necessary. New Stickies keep the source sort-order spacing and list order.

The Django route remains active while `GO_STICKIES_ENABLED=false`. This difference starts when the Go Sticky route is active.
