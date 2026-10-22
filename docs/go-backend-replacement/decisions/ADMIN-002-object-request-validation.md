# ADMIN-002: Reject non-object administration request bodies

Status: Implementation decision; new pristine/corrected source fixtures and final replay are required before activation.

The estimate create/update, estimate-point create, and bulk label-create operations consume an object request body. A top-level JSON array reaches object-only access in Django and produces HTTP 500. Empty arrays must not be accepted as successful bulk creation either.

After the source authentication, authorization, and any preceding object lookups, reject a non-object body with HTTP 400 and `{"error":"Please provide valid detail"}` for these project operations. This uses their existing validation-error envelope.

Instance administrator create, configuration update, workspace create, and email-credential checks also crash on a top-level array. Return HTTP 400 with `{"non_field_errors":["Invalid data. Expected a dictionary, but got list."]}` for these requests, matching the source-proven DRF serializer response of instance root PATCH. Root PATCH already returns this response in Django and requires no intentional difference.

Perform no database writes or durable effects for these rejections. Preserve any cache invalidation that the source performs before body validation; the corrected recorder must assert that effect rather than erase it. Object payloads retain their existing field validation and behavior; the parser's malformed JSON and UTF-8 checks remain separate.

Pristine fixtures retain the source failure. Corrected source fixtures must demonstrate the chosen response, unchanged database and durable effects, and any preceding cache invalidation. Live differential projections enumerate exact named cases, source status/body digest, and replacement status/body digest. Other response headers and authorization outcomes remain strict; this is not a general exception for source HTTP 500 responses.
