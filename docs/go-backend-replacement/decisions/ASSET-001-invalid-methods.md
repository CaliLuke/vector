# ASSET-001: Reject unsupported asset methods without dispatch crashes

Status: Implementation decision; each source failure and the replacement still require executable evidence before activation.

The asset URL configuration exposes methods whose handlers require an identifier missing from the path, or cannot accept an identifier supplied by the path. These requests cannot reach a successful handler. Django normally reports a generic 500 after its authentication and permission checks. The two attachment collection DELETE routes instead raise `KeyError` in the creator-permission wrapper because the URL has no `pk`; the exception handler maps that failure to 400 with `{"error":"The required key does not exist."}`. These two exact responses also become 405. Their existing active-workspace-membership denial still takes precedence. This is a routing defect, not an intended business response.

Go will return the normal JSON 405 response for these method/path combinations: `{"detail":"Method \"DELETE\" not allowed."}`, substituting the actual method. HEAD has no response body. Existing authentication, permission, request-size and middleware precedence continue to apply. No database, cache, object-store or worker effect is introduced by this rejection.

The Allow header will advertise the methods that work on that resource, with HEAD only where GET works and OPTIONS where the source supports it. Valid methods keep their behavior. The exact source and target method sets are in [the decision matrix](ASSET-001-invalid-methods.json). The matrix includes 23 operations in the original inventory and four additional discovery-only methods already excluded from the retirement denominator. The latter do not count as extra retired operations.

The source fixtures must retain the actual failure responses. Replay may substitute only the exact, named response differences in this decision after checking the expected source values. It must still compare authentication failures, database deltas, cache changes, durable tasks and S3 effects. A broad exemption for 500 responses is prohibited. Normal 405 control requests establish the expected response headers and body; production proxy tests must cover the corrected methods and retained successful neighbors.

This change does not invent collection retrieval or bulk mutation APIs. Upload uses the existing collection POST; retrieval and modification use the existing identifier-bearing paths. The family caller audit must verify these supported paths before activation.
