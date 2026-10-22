# ADMIN-004: Refresh configuration after disabling email

Disabling email must make subsequent configuration reads reflect the disabled values immediately. Preserve the source value-only database update, including unchanged timestamps, and invalidate both `/api/instances/` and `/api/instances/configurations/` after authorization and before the update.

Django caches configuration GET for two hours but invalidates only the root on disable-email. A real source sequence (GET, DELETE disable-email, GET) proves that the second GET still returns the prefetched SMTP password. The corrected source overlay removes the configuration cache entry and the second GET returns the disabled value. Sanitized evidence is in `../evidence/instance-management/configuration-cache-sequence-pristine.json` and `configuration-cache-sequence-corrected.json`; the recorder asserts actual response content and cache state without recording a secret.

The pristine disable-email fixture remains unchanged. The corrected fixture `instance_management_corrected/config-disable-email.json` requires both invalidations. The Go service regression checks authorization and invalidation before writes; the PostgreSQL regression checks value changes and unchanged timestamps.

The live comparison enumerates only the first configuration GET after the matrix disables email, using exact source and corrected response digests. All status codes, headers, unrelated fields, and subsequent cases remain strict. This correction follows the user's instruction to fix source bugs during migration.
