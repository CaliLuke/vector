# Workspace search: reject malformed entity-search counts

The pristine Django fixture `entity-search-invalid-count` records HTTP 500 for
`count=many` because integer conversion is uncaught. This is an input-validation
bug, not a behavior to preserve. The corrected source oracle returns HTTP 400
with a count validation error before invoking the search repository.

The canonical differential projects only the exact malformed-count case from
the pristine source response (status 500, body SHA-256
`d06853a8ad551a5a3795f75fc78bcac6c8606bb49f6fe78a4e39573259c241e2`) to the
Go response (status 400, body SHA-256
`ad04adee6bf5958ece48c6b192079d21f610c43cbdfbf4a6beb98b7807633a28`). The
recorded headers are identical and remain unprojected. The corrected source
fixture remains separate from the pristine source evidence.

The recorded source decision and pristine/corrected fixture hashes are in
[`decision-notes.md`](../evidence/workspace-search/decision-notes.md) and
[`corrected/index.json`](../../../apps/go-api/internal/parity/testdata/fixtures/workspace_search/corrected/index.json).
