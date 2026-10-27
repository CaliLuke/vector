# Remote MCP transport

VEC-2 is **in progress**. The Go API contains a generated, stateless MCP server
and ingress routing. Available member actions are `me`, `list_workspace`, `list_project`, `list_roles`, and `retrieve_role`. The `project` tool provides `list`, `retrieve`, `archive`, and `unarchive`. The `cycle`, `module`, `label`, `workitem_comment`, `workitem_link`, and `workitem_activity` tools provide `list` and `retrieve`; cycles and modules also support archived lists. Personal-token and external-issuer OAuth authentication are wired. MCP is disabled
by default. OAuth requires an explicitly configured issuer and deliberate account
linking; invalid configuration or discovery fails before database startup.

The source declaration is `apps/go-api/mcp/design/design.go`. Regenerate only
this projection with:

```sh
source scripts/dev-env.sh
go -C apps/go-api generate ./mcp/design
```

The integration pins `loom-mcp/v2 v2.1.0-alpha.48` with Vector's existing
`loom v1.10.0-alpha.17`. Generated files are not edited by hand. The dependency
adds its SDK and schema/runtime dependencies; no local module replacement is used.

## Transport configuration

| Environment variable    | Default   | Meaning                                                                          |
| ----------------------- | --------- | -------------------------------------------------------------------------------- |
| `MCP_ENABLED`           | `false`   | Enable configured routes only when their authentication providers are available. |
| `MCP_AUTH_MODE`         | empty     | `header`, `oauth`, or `both`; required when enabled.                             |
| `MCP_RESOURCE_URL`      | empty     | Canonical HTTPS URL ending in `/mcp`; required when enabled.                     |
| `MCP_TRUSTED_ORIGINS`   | empty     | Additional comma-separated HTTPS origins, without paths.                         |
| `MCP_MAX_REQUEST_BYTES` | `1048576` | Positive request body limit, at most 16 MiB.                                     |
| `MCP_REQUEST_TIMEOUT`   | `30s`     | Positive request timeout, at most five minutes.                                  |

`/mcp` selects only OAuth; `/mcp/api-key` selects only personal-token
authentication. A missing provider is a startup error. A rejected credential
never falls back to the other provider. Disabled routes, including `/mcp/*`,
return 404 instead of the frontend page. All three proxy configurations route
these paths to the Go API.

The `mcp` workspace slug is now reserved. **Before deploying these proxy routes,
rename any existing workspace with that slug through workspace settings.**
The route reservation also applies while MCP is disabled. This change does not
rename existing workspaces or modify their data automatically.

Each request must use the configured host. Origins must match the canonical
HTTPS origin or the explicit trusted-origin list. Query strings are rejected.
The HTTP transport limits body size and execution time. Responses are not stored
by caches. MCP result metadata also declares a private cache scope and zero
lifetime. Authentication runs for every request, with cancellation and safe
400/401/403/429/503 errors. The generated server owns the MCP wire protocol.

## Personal-token authentication

Header mode requires exactly one `Authorization: Bearer <personal-api-token>`
and one `X-Workspace-Slug` header on every request. Duplicate headers, combined
credentials, malformed workspace selectors and any `X-Api-Key` header return 400.
Missing or invalid credentials return 401; a current nonmember receives 403.
Cookies cannot authenticate this route, and it never issues an OAuth challenge.

The existing API-token validator reads the primary database and checks the
token's active, expiry and revocation state and its owner's active status.
Workspace membership is also read from the primary. A token's optional workspace
metadata does not limit or grant access. Personal tokens retain the owner's
domain permissions and do not acquire synthetic OAuth scopes.

Requests use the shared API-key throttle implementation and `API_KEY_RATE_LIMIT`,
with MCP keys for both the actor and credential. Credential cache keys contain a
SHA-256 digest. A denied request returns 429 and `Retry-After`; cache or database
failures return a sanitized 503. Mounted MCP routes use the shared request-ID
middleware, including authentication failures. The response returns a validated
or generated UUID in `X-Request-ID`. Successful authentication updates token
`last_used` through the existing validator.

The authentication package revalidates before each member action invocation and
binds optional tool workspace selectors to the header. Each future action must
also enforce its current project and domain permissions. Archive/restore receipt replay and revision checks are described in [write recovery](mcp-writes.md).

The supported revisions are `2026-07-28` and `2025-11-25`. The former uses
stateless `server/discover` and per-request protocol metadata; the latter uses
the legacy initialize handshake. A legacy initialize request cannot negotiate
the new lifecycle. The server does not issue session IDs or advertise
unimplemented tools, prompts, or resources.

## OAuth authentication and account linking

OAuth uses the [issuer and account-link contract](mcp-oauth.md). Configure the
following settings in addition to the shared transport settings:

| Variable                                | Purpose                                              |
| --------------------------------------- | ---------------------------------------------------- |
| `MCP_OAUTH_ISSUER_URL`                  | Exact external OIDC issuer URL.                      |
| `MCP_OAUTH_INTROSPECTION_URL`           | HTTPS endpoint for fresh access-token introspection. |
| `MCP_OAUTH_INTROSPECTION_CLIENT_ID`     | Confidential introspection client.                   |
| `MCP_OAUTH_INTROSPECTION_CLIENT_SECRET` | Introspection client secret.                         |
| `MCP_OAUTH_LINK_CLIENT_ID`              | Separate confidential OIDC account-linking client.   |
| `MCP_OAUTH_LINK_CLIENT_SECRET`          | Account-linking client secret.                       |

The issuer must support the documented code/S256/form-post profile, RS256 ID
tokens, stable subjects shared across clients, and the required introspection
claims. Register the resource origin plus `/auth/mcp/callback/` as the linking
client's callback. Vector neither creates OAuth clients nor matches accounts by
email. The production issuer has not been selected or deployed.

Signed-in users connect their identity under profile **API tokens**, return from
the issuer, then explicitly confirm the verified subject. The original session
and a protected callback-browser cookie must both match. Disconnect removes the
link and pending flows; it does not revoke grants at the external issuer.

Each OAuth HTTP request and action invocation rechecks introspection, the live
account link and account activity. Reads require `vector:read`; mutations
require both read and write scopes. Workspace selection uses the header or tool
argument. `member.me` can run without a workspace selector. Scoped actions still
check current membership and domain permissions against the primary database.

Only OAuth-enabled modes expose `/.well-known/oauth-protected-resource/mcp`.
OAuth credential and scope errors carry the corresponding Bearer challenge;
personal-token errors never do. The observer emits sanitized failure logs with
request/trace correlation. These routes reject query strings and redact their
values before HTTP telemetry. Authorization codes, bearer tokens and browser
proofs are not logged.

## Available actions

`member` accepts the five read actions below, each with an optional
`workspace_slug` selector. Discovery advertises only implemented actions.

| Action           | Inputs                                                                   |
| ---------------- | ------------------------------------------------------------------------ |
| `me`             | No resource identifier.                                                  |
| `list_workspace` | Optional text, role and boolean filters, ordering, cursor and page size. |
| `list_project`   | Required `project_id`.                                                   |
| `list_roles`     | Optional namespace, cursor and page size.                                |
| `retrieve_role`  | Required `role_id`.                                                      |

`project` provides `list` with optional cursor, page size and ordering, and
`retrieve` with a required project UUID. Archive and restore require a project UUID and accept the optional [receipt and revision extensions](mcp-writes.md). All four accept `workspace_slug` and use
current primary-backed authorization. The [project contract](mcp-projects.md)
defines SDK projections, archive inclusion and immutable workspace binding.

The [member directory contract](mcp-members.md) defines filters, role mappings,
IDs, permissions, defaults and compatibility behavior. The
generated union schema retains a top-level object and typed properties, with
closed, action-specific branches. Unknown actions and fields belonging to other
actions fail with `invalid_params`. A supplied workspace selector that conflicts
with the header returns HTTP 400 `invalid_request` before any application read.

The `me` action uses the same credential-neutral identity reader as `GET /api/v1/users/me/`.
That reader selects only the authenticated actor and checks the account's current
active state. The MCP projection normalizes absent UserLite fields to null, as the
pinned Plane SDK does; the HTTP response shape remains unchanged. Both designs
reference the same API identity field declaration.

The generated contract declares stable member and project request/result unions, typed identities, pages, role and project records, and safe named errors.
Tests use the generated HTTP server and real PostgreSQL-backed token, membership,
and identity readers. They cover both protocol revisions, actor isolation, fresh
profile data, and revocation between request authentication and action execution.
Separate OAuth integration tests exercise all twenty-one actions with a real TLS
issuer and PostgreSQL, including scope reduction and revocation between HTTP
authentication and execution. Complete browser verification and VEC-36 proof
remain pending.

The `list_project` action and the project-member HTTP endpoint use the same credential-neutral `MemberReader`.
It reads the primary database and checks project access in the same SQL statement
as the roster read. MCP also binds the selected workspace ID and rechecks current workspace membership in that statement. All scoped member reads reject a reassigned workspace slug. PostgreSQL tests cover revocation after initial authorization,
deleted projects and workspaces, and tenant isolation. Its response remains the
existing UserLite array. The MCP projection fills the SDK's absent role fields
with null; it does not infer role metadata absent from the HTTP contract. The
operation is unpaginated, as in the pinned tool. Its nonempty result is a JSON
text block without `structuredContent`; an empty result has no content blocks.
This matches the pinned FastMCP list response on both protocol revisions.

`list_workspace`, `list_roles` and `retrieve_role` use the same primary-backed
`DirectoryReader` as the members-lite and new role HTTP endpoints. Directory
access requires a current workspace member or admin; guests can read role
definitions. Filters run before counting and paging. Required identity strings
preserve the HTTP serializer's empty-string projection for database nulls.

The catalogue contains four workspace definitions and three project definitions.
It maps existing permissions without adding a Commenter or custom-role model.
Role IDs are stable within a workspace and cannot resolve in another workspace.
The HTTP role routes are `/api/v1/workspaces/{slug}/roles/` and
`/api/v1/workspaces/{slug}/roles/{role_id}/`. Workspace member filters and
`role_slug` are additive to the existing members-lite endpoint. Its default page
size and legacy cursor/errors remain unchanged.

## Remaining requirements

- **Generated action contracts:** the upstream union defects in
  [loom-mcp #381](https://github.com/CaliLuke/loom-mcp/issues/381) are fixed in
  alpha.48. Vector consumes the released module without a replacement or retained
  workaround for #381. Vector's generated-server tests verify closed multi-action
  inputs and object/array result-union dispatch through both protocol revisions.
  Expand the generated catalogue as application actions become available.
- **Array result compatibility:** [loom-mcp #383](https://github.com/CaliLuke/loom-mcp/issues/383)
  tracks generated non-object `structuredContent` being sent to 2025-11-25 clients,
  whose protocol requires an object. The generated interceptor hook now applies
  the pinned member-list content contract without editing generated code. Remove
  this interceptor when the minimum upstream version can produce that content-only
  list shape, including empty lists, for both revisions. Identity results retain
  their object-valued structured content.
- **Nested result unions:** [Loom #1036](https://github.com/CaliLuke/loom/issues/1036)
  tracks the unsupported nested untagged-union composition. Member directory and cycle
  results use one page with a concrete typed entry carrier, preserving exact
  action-specific fields and empty pages. The member contract records its removal
  condition; no dependency source is patched.
- **Discovery authorization:** action schemas follow current scopes and domain permissions, using copies of the generated catalogue. Workspace guests do not see `member.list_workspace`; `member.list_project` requires an active membership in a live project. Role and project reads require an eligible workspace membership. Context-free OAuth discovery considers all eligible workspaces; without one, only `member.me` remains. A selected workspace binds both its immutable ID and slug. Invocation rechecks access to the selected resource. There is no handwritten replacement registry.
- **VEC-35 and VEC-3:** personal-token and generic OAuth invocation are implemented
  for twenty-one member/project/cycle/module/label/comment/link/activity actions, including current scopes, linking and revocation.
  Production issuer selection and complete browser/proxy evidence remain open.
  Mixed-action discovery filtering and archive/restore replay now have focused
  generated-server PostgreSQL proof and pass the full Go gate.
- **Reliability and observability:** the [receipt transaction foundation](mcp-writes.md)
  has real PostgreSQL race, rollback, and lost-acknowledgement proof. Archive and restore now connect shared action authorization, revision checks,
  favorite cleanup, and receipts in one transaction. Their mutation logs and traces
  have correlated safe-outcome tests and pass the full Go gate. Remaining mutations and browser conflict proof are still open. Authentication and generated-validation
  failures now have sanitized log/trace proof; this does not establish the
  complete mutation or asset observability contract.
- **Resource tickets and VEC-36:** implement and verify the 30 resource tools and
  207 actions. The pinned expected catalogue lives under
  `apps/go-api/mcp/testdata/plane-mcp`; the [207-action tracker](mcp-parity.md)
  records remaining work. Neither is a runtime registry or a parity claim.

Focused verification exercises the real generated HTTP handler, both route
modes, current and legacy discovery, invalid protocol headers, host/origin
rejection, request limits, cancellation, unavailable authentication, and startup
configuration. Typed results and errors are exercised for all five member actions.
Generated-server discovery tests cover both credentials and revisions, membership
changes, context-free OAuth, guest permissions, and removed project membership.

See [work-item child read contracts](mcp-workitem-reads.md) for page shapes, mention normalization, relationship isolation, and the nullable-enum workaround.
