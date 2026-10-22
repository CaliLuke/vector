# ADMIN-005: Validate scalar workspace name and slug inputs

## Decision

Treat truthy non-string `name` or `slug` values as serializer validation errors and return HTTP 400 with the field error `Not a valid string.`. Preserve the existing required-field response for zero, false, null, and omitted values. The user explicitly approved this correction: return HTTP 400 for invalid types and create nothing.

## Source evidence

The pinned Django workspace view evaluates `len(name)` and `len(slug)` before serializer validation. Four source-recorded requests with numeric or boolean values therefore fail with the generic HTTP 500 response and no database writes. The pristine fixtures remain unchanged:

- `instance_management/workspaces-post-number-name.json`
- `instance_management/workspaces-post-bool-name.json`
- `instance_management/workspaces-post-number-slug.json`
- `instance_management/workspaces-post-bool-slug.json`

The corrected source overlay returns a field-specific HTTP 400, with no database changes or durable effects. Its four fixtures are under `instance_management_corrected/`. Capture evidence is `../evidence/instance-management/source-20261022T033215Z.log` and `source-20261022T033838Z.log`.

## Scope

This applies only to the selected workspace-create operation. String values, including strings containing only digits, continue through existing validation. Existing name/slug rules, raw-length precedence, and required-field handling remain unchanged. The parity corpus keeps each pristine 500 fixture and pairs it with a corrected 400 fixture for the same named input. Source replay checks exact field errors, unchanged database state, and no durable effects. The HTTP method matrix does not synthesize these scalar request bodies, so the decision does not widen a route-level live-comparison projection.
