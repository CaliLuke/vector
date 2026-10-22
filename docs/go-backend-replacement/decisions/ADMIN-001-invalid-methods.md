# ADMIN-001: Reject invalid administration method/path combinations

Status: Implementation decision; activation requires the source/corrected fixture replay and live differential proof.

Pinned Django views expose handlers with incompatible URL arguments. Workspace webhook collection PATCH and DELETE require an absent `pk`; detail POST receives an unexpected `pk`. Instance administrator collection DELETE requires an absent `pk`; detail GET, HEAD and POST receive an unexpected `pk`. After their existing authentication and permission checks these requests fail with HTTP 500. They cannot perform the requested operation.

Return HTTP 405 with the standard DRF-shaped JSON body `{"detail":"Method \"DELETE\" not allowed."}`, substituting the actual method; HEAD remains bodyless. Preserve authentication, permission, timezone and middleware precedence, existing Allow headers, and absence of database or durable effects. No collection update/delete or detail read/create operation is introduced.

Keep pristine failures and corrected source oracle fixtures separate. The family contracts enumerate only the exact canonical cases and exact source/target status and body digests. Headers, authentication failures, state changes and effects remain compared normally. An unexpected source body or status cannot use this decision. The source 405 controls establish the response envelope; the final focused regressions and family replay must establish the corrected behavior before activation.
