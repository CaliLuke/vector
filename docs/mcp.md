# Remote MCP transport

VEC-2 is **in progress**. The Go API contains a generated, stateless MCP server
and ingress routing. No resource actions are registered. Personal-token
authentication is wired; OAuth remains unavailable. MCP is disabled by default.
Enabling `oauth` or `both` fails before database startup.

The source declaration is `apps/go-api/mcp/design/design.go`. Regenerate only
this projection with:

```sh
source scripts/dev-env.sh
go -C apps/go-api generate ./mcp/design
```

The integration pins `loom-mcp/v2 v2.1.0-alpha.47` with Vector's existing
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

The authentication package provides fresh revalidation for future action calls
and receipt replay, and binds optional tool workspace selectors to the header.
Each action must still enforce current project and domain permissions. No action
or receipt handler is registered yet, so those integrations remain unproven.

The supported revisions are `2026-07-28` and `2025-11-25`. The former uses
stateless `server/discover` and per-request protocol metadata; the latter uses
the legacy initialize handshake. A legacy initialize request cannot negotiate
the new lifecycle. The server does not issue session IDs or advertise
unimplemented tools, prompts, or resources.

## Remaining requirements

- **Generated action contracts:** [loom-mcp #381](https://github.com/CaliLuke/loom-mcp/issues/381)
  tracks two defects reproduced with a standalone, two-branch object union:
  generated transport conversion calls missing union accessors, and the
  generated input schema lacks the root object type required by the SDK.
  The same compile failure occurs with Loom alpha.16 and alpha.17. No workaround
  or patched dependency is retained. A released fix must compile the projection,
  register it with the SDK, and reject action-specific invalid inputs before
  resource actions can be added.
- **Discovery authorization:** prove per-principal permitted-action schemas and
  invocation checks through the generated handler after the action contract is
  usable. There is no handwritten replacement registry.
- **VEC-35 and VEC-3:** complete personal-token invocation integration and supply
  the OAuth provider, including scopes and account linking. Personal-token
  transport authentication is implemented; action-level authorization, discovery
  filtering and receipt replay still need end-to-end proof.
- **Reliability and observability:** connect the shared action authorization,
  throttling, durable write receipts and sanitized error/trace requirements
  before enabling covered actions. The transport accepts the generated observer
  hook, but this does not establish the complete observability contract.
- **Resource tickets and VEC-36:** implement and verify the 30 resource tools and
  207 actions. The empty initial catalogue is not a parity claim.

Focused verification exercises the real generated HTTP handler, both route
modes, current and legacy discovery, invalid protocol headers, host/origin
rejection, request limits, cancellation, unavailable authentication, and startup
configuration. It does not yet prove typed action results/errors or permission
filtering; those remain blocked above.
