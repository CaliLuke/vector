# ADMIN-003: Serve estimate-point request metadata

Status: Implementation decision; corrected source capture passes, final target replay remains required before activation.

The estimate-point collection's source OPTIONS request fails because the view omits `serializer_class`, which inherited DRF `SimpleMetadata` needs. Supplying the existing `EstimatePointSerializer` on the isolated corrected source produces HTTP 200 with the endpoint metadata and `actions.POST` field descriptions, required/read-only flags, and numeric/string limits. Go must return that complete source-generated metadata.

The active corrected oracle is [the corrected `options_estimate_points` fixture](../../../apps/go-api/internal/parity/testdata/fixtures/project_configuration_corrected/options_estimate_points.json), captured in `source-20261022T015849Z-86588.log` with a 1/1 pass counter. The earlier [corrected `options_estimate_points_valid_metadata` fixture](../../../apps/go-api/internal/parity/testdata/fixtures/project_configuration_corrected/options_estimate_points_valid_metadata.json) remains as a separate capture of the same metadata behavior. The pristine source HTTP 500 remains in its original fixture. Preserve `Allow: POST, OPTIONS`. The larger valid metadata body also adds `Accept-Encoding` to `Vary` through the existing gzip middleware; the exact header change is included in the differential projection.

Authentication and middleware checks precede metadata, and the request performs no database writes or durable effects. The live differential projection applies only to the named estimate-point collection OPTIONS case and exact source status/body/header values. Unexpected errors or metadata changes remain failures.
