# MCP OAuth resource access and account linking

## Summary

Vector accepts access tokens from one configured external OIDC issuer on `/mcp`.
The issuer owns OAuth clients, authorization, consent, refresh and revocation.
Vector owns a deliberate link from the issuer's verified subject to an existing
signed-in Vector user. No email-based matching or account creation occurs.
This contract implements the OAuth part of the accepted MCP master design;
implementation does not enable remote access or establish 207-action parity.

## Implementation status

The backend, generated linking routes and profile controls are implemented.
Focused checks cover both MCP revisions, real PostgreSQL, the TLS issuer, session
revocation, account linking, scopes and sanitized correlated logs/traces.
The frontend and Go full quality gates pass. Complete browser/proxy evidence remains open. OAuth stays generic and provider-agnostic;
production issuer selection and deployment verification are deferred. The verification section below states completion criteria,
not a claim that every criterion has already passed.

## Current state and ownership

`internal/mcptransport` mounts the generated stateless MCP server for both
supported revisions. `internal/mcpauth` verifies personal tokens or introspected OAuth identities and
workspace membership against the primary database, then revalidates each action.
`internal/mcpactions` calls credential-neutral application readers.
`cmd/system/mcp_config.go` validates selected authentication configuration before
startup; OAuth discovery also completes before databases or workers start.
These boundaries remain. Auth class and scopes are private request context,
never tool arguments or personal-token scope grants.

The existing `oauth_redirects.go` login service uses provider profile/email
identity and can create accounts. MCP linking cannot use that login operation.
The existing Django session validator, CSRF checks, shared API-key throttle,
instance cipher and embedded bootstrap SQL migration mechanism remain owners
of their respective primitives. New account-link HTTP contracts are authored
in a dedicated Loom design and generated; their application service uses the
same session validator and primary database. There is no handwritten alternate
router or handwritten generated transport.

## Provider and startup contract

OAuth configuration is required only when MCP is enabled in `oauth` or `both`.
Header mode serves personal-token clients directly at `/mcp`, with `/mcp/api-key` retained for existing clients. In `both` mode, `/mcp` remains OAuth-only and personal tokens use `/mcp/api-key`; there is no authentication fallback. Header mode remains independent of issuer/linking configuration. Disabled
OAuth metadata and linking routes return 404. Defaults remain MCP disabled.

Required settings are issuer URL, an explicit introspection endpoint URL,
introspection client ID and secret, and a separate OIDC linking client ID and
secret. Resource URL remains the exact canonical HTTPS URL ending in `/mcp`.
The linking callback is its origin plus `/auth/mcp/callback/`; operators register
that exact URI. URLs require HTTPS, a host, no credentials, query or fragment.
Issuer paths and trailing slashes are preserved for exact issuer comparison.
There is no public insecure-development bypass. TLS test servers supply tests.

OIDC discovery must return exactly the configured issuer, HTTPS authorization,
token and JWKS endpoints, code flow, S256 PKCE, `form_post`, `openid`, and
`client_secret_basic` token authentication, and RS256 ID-token signing. Other
ID-token signing algorithms are outside the initial supported issuer profile. Introspection uses only the
explicit configured endpoint and `client_secret_basic`. Startup discovers and
validates this profile; failure prevents starting the selected mode. Runtime
outages fail closed. Endpoint values come only from trusted operator
configuration or that configured issuer's discovery, never request input.
HTTP clients have a ten-second timeout, reject redirects and cap response bodies
at 1 MiB. They do not include upstream errors or bodies in logs or responses.

The issuer must issue stable end-user subjects shared by its linking client and
MCP access-token clients. Pairwise subjects that differ between those clients
cannot be silently reconciled by email. The issuer must report `grant_type`
as `authorization_code` or `refresh_token` for supported user access tokens.
This is a required Vector issuer profile, not a claim that RFC 7662 requires
those fields. Missing grant classification and machine/client-credentials grants
are rejected. No provider-specific guessed claim aliases are accepted.

## Token validation, scopes and workspace selection

Every OAuth HTTP request and every action/replay independently introspects the
presented bearer token. There is no active-token cache or JWT-only fallback.
An authenticated successful JSON introspection response must have:

- `active: true`, nonempty `sub` and `client_id`, exact configured `iss`;
- `aud` string or string array containing the exact canonical MCP resource;
- integer `exp` strictly greater than now, and, if present, `nbf` no later than now;
- `token_type` equal to Bearer ignoring ASCII case;
- a supported end-user `grant_type` and a valid space-delimited OAuth `scope`.

Unknown additional response fields are ignored. Invalid credential claims or
`active:false` yield 401. Network, timeout, non-200, malformed/oversized JSON and
issuer authentication failures yield sanitized 503. Inactive responses need
no other claims. An ID token, personal token or unsupported machine token does
not trigger another validator. No token is sent to any endpoint except the
configured introspection endpoint; linking exchange uses its own code/verifier.

After claims validation, the primary database must resolve the exact
`(issuer,subject)` link and an active non-bot Vector account. Missing link or
inactive/deleted/bot account yields 401. Revalidation retains the original
actor and issuer/subject/client identity: changing those for the same bearer
fails closed. Scopes may narrow and are checked afresh. All reads require
`vector:read`; mutations require both `vector:read` and `vector:write`. An
otherwise valid credential lacking read gets 403 `insufficient_scope` with
the read challenge. Unknown scope tokens grant no additional rights.

OAuth accepts exactly one bearer header and zero or one workspace header.
Personal tokens use the same optional workspace header and selector rules. X-Api-Key, duplicate selectors,
duplicate Authorization headers, comma-combined bearer values, malformed
workspace slugs or conflicting header/tool selectors yield HTTP 400.
Missing or invalid bearer credentials yield 401. Cookies never authenticate MCP.

Workspace selection uses header when present, otherwise optional tool argument.
The selected slug is validated and resolved to a current primary-backed active
membership before any scoped application read. Missing scope on a scoped action
yields generated `invalid_params`. `workspace.list`, `member.me` and the PQL reference need
no selector; an explicitly provided selector still requires membership. Unknown
or inaccessible workspace yields the same safe forbidden error. Shared
application operations retain their own authorization inside data reads/writes.
The private verified context owns a workspace resolver, rather than accepting a
caller-fabricated actor. Both credential classes use this resolver, while token
validation and OAuth scopes remain separate.

The shared throttle applies actor and SHA-256 credential keys to each HTTP
request. Failure is sanitized 503; denial is 429 with Retry-After. Introspection
also occurs at invocation, without double-charging the same HTTP request.

## Metadata and generated discovery

Only enabled OAuth publishes `/.well-known/oauth-protected-resource/mcp`.
The SDK's metadata type/handler projects the configured resource, one issuer,
`vector:read` and `vector:write`, and bearer method `header`. No resource JWKS,
registration, token, refresh or authorization endpoint is hosted by Vector.
The proxy routes this exact well-known path to Go. GET/OPTIONS is public with
canonical host validation and CORS `*` without credentials, no-store. Query
strings, unknown subpaths and unsupported methods fail without SPA fallback.

OAuth 401 responses include a Bearer challenge with canonical resource_metadata
and scope `vector:read`; invalid tokens also include `error="invalid_token"`.
The initial 403 insufficient_scope challenge requests vector:read. Ordinary domain denials do not cause a scope challenge. PAT never
emits OAuth challenges or metadata. Authentication/validation failure outputs
remain safe and correlated by the shared request ID.

Generated action declarations remain the catalogue source of truth. Discovery
filters the generated actions using current OAuth scopes and primary-database
permissions. Workspace directory reads require a
member or admin role; project roster reads require an active project membership.
Role and project reads require an eligible workspace membership. Cycle, module, label, comment, link, and activity reads require an active project membership. Without a
selected workspace, discovery considers all eligible memberships; accounts with
none can still use `workspace.list`, `member.me`, and the PQL reference. Archive and restore follow their domain permissions.
The interceptor filters fresh copies of generated schemas without a second action
registry. Invocation always rechecks current scopes and access to the selected
resource. Cache metadata stays private, TTL zero.

### Failure precedence and wire stage

The HTTP boundary and generated action boundary have distinct wire contracts:

| Stage                                                             | Rejection                                                      | Response                                          |
| ----------------------------------------------------------------- | -------------------------------------------------------------- | ------------------------------------------------- |
| Initial request target/header/selector validation                 | malformed or conflicting input                                 | HTTP 400 invalid_request                          |
| Initial OAuth introspection/account validation                    | missing/invalid credential                                     | HTTP 401 OAuth challenge                          |
| Initial validation                                                | issuer/database/cache outage                                   | HTTP 503 sanitized failure                        |
| Initial validation                                                | valid token lacks vector:read                                  | HTTP 403 insufficient_scope challenge             |
| Generated action-specific schema and eligible branch validation   | unknown, malformed or non-advertised action                    | generated invalid_params, no application dispatch |
| Revalidation inside an eligible action after initial HTTP success | revoked token, changed subject/client, account or link removed | existing generated unauthorized tool error        |
| Revalidation inside an eligible action                            | issuer/database unavailable                                    | existing generated internal_error tool error      |
| Revalidation inside an eligible action                            | newly narrowed scopes or resource access revoked               | existing generated forbidden tool error           |

Generated tool errors retain the SDK's normal HTTP 200 JSON-RPC tools/call
response with isError=true and safe text, matching the current PAT action path.
No late attempt rewrites committed transport headers or adds a challenge to a
tool result. A new HTTP request with that revoked or narrowed token receives the
initial-authentication status above. In particular read present at HTTP auth
but lost at action revalidation yields a generated forbidden tool error; a
read-only caller's write branch is excluded and rejected as invalid_params
before action dispatch. Authentication challenges belong only to the outer HTTP
boundary in internal/mcptransport. This distinction preserves existing PAT wire
behavior and does not cache or reuse an earlier authorization decision.

## Signed-in account linking

New generated HTTP methods under `/api/users/me/mcp-link/` expose status (GET),
start (POST `/start/`), confirm (POST `/confirm/`), and disconnect (DELETE base).
These require the existing active browser session. All mutations require the
existing CSRF token/origin checks; bearer tokens cannot substitute for the cookie.
Body limit is 16 KiB. No user-selected redirect or issuer is accepted.

Status returns enabled issuer, connected boolean, linked subject when connected,
and nullable pending confirmation `{id,subject,expires_at}` only when the user,
original session AND callback proof cookie all match the verified flow. These are private/no-store. Start returns the authorization URL;
its only sensitive random values are protocol state/nonce and S256 challenge,
never bearer tokens, verifier, client secret or authorization code. The browser
navigates to it directly; no persistence or request logging of that URL.

Start generates independent 256-bit state/nonce and an oauth2 PKCE verifier,
stores only state/session hashes and nonce hash, encrypts the verifier using the
existing instance cipher, and sets a non-extendable ten-minute expiry. It replaces
all older pending flows for that user under the same row lock used by disconnect
and confirm. A current link returns 409; replacing links requires disconnect.
The authorization request is code + S256, scope openid, response_mode form_post,
and prompt select_account. No offline access or MCP scopes are requested by the
linking client. The issuer owns separate MCP-client grants.

POST `/auth/mcp/callback/` consumes a bounded form response, not query arguments.
It is the only cross-site callback exempt from ordinary session CSRF checks;
state, nonce and PKCE protect it. It does not require cookies, which SameSite
can suppress on a cross-site POST. Duplicate form keys, mixed code/error, missing
state, query parameters or non-form content fail 400. Unknown extra provider
form keys are ignored, except an `iss` value if present must match the issuer.
The callback never creates a link or a login session.

The state transition is atomic pending -> exchanging before any code exchange;
unknown, expired or already-consumed state fails 400. It validates the returned
ID token's signature, exact issuer, linking-client audience, expiry, nonce and
nonempty subject using released go-oidc; multiple audiences require matching azp,
and any azp must equal the linking client. Access/refresh/ID tokens are discarded
after verification. The callback generates a fresh 256-bit browser proof and records its hash with
the verified subject only if the same unexpired exchanging row still exists. It
sets the raw proof only in a Secure, HttpOnly, SameSite=Strict cookie named
`__Host-vector-mcp-link` with Path=/, no Domain and expiry equal to the flow
expiry. Neither start nor status returns this proof. It then redirects 303 to the fixed existing
profile API-token settings page. No callback value appears in the redirect URL.
Provider denial consumes the flow and returns the same fixed page; invalid or
failed exchanges yield a safe error and require starting again. No automatic
exchange retry occurs after a lost response. All callback responses use no-store
and no-referrer; input form bodies, codes, tokens, state and nonce are redacted
from application/proxy traces and logs.

The profile page displays the configured issuer and verified subject and asks
the signed-in user to confirm. Confirm accepts only the pending UUID; it checks
the same active user and session hash, a constant-time match of the callback
proof cookie hash, and unexpired verified phase, then inserts
the unique `(issuer,subject)->user` link and consumes the pending row in one
transaction. The database also limits a user to one subject per issuer. Another
user's subject or any existing conflicting link returns safe 409 without
disclosing its owner. A successful confirm deletes the pending row and clears its proof cookie.
There is no confirmation receipt or automatic retry. Reusing a consumed pending
UUID returns safe 409 without mutation, even if the caller still has its old
proof cookie. After a lost acknowledgement the UI reloads status; connected
means success, disconnected means a fresh flow is required. Subsequent
disconnect cannot be reversed by a delayed confirm retry. Status, rather than
a retained success response, remains authoritative.

Disconnect deletes the user's current issuer link and every pending flow under
the same user lock. It is idempotent and returns disconnected status. An earlier
callback cannot recreate a deleted flow and an earlier confirm serializes before
or after disconnect. A fresh link always requires fresh explicit consent.
An in-flight action whose authorization already completed is not rolled back;
every subsequent request/call/replay rechecks the link. Start, successful confirm
and disconnect clear the callback proof cookie. Its expiry also limits abandoned
flows. A second browser cannot complete an attacker-started flow for the attacker:
the attacker has the original session but lacks the callback proof, while the
callback browser has the proof but lacks that original session. Status cannot
reveal the pending subject to either mismatched context. Issuer grants are not
revoked by unlink; Vector immediately stops accepting them for that mapping.

Expired pending records are pruned on start and by bounded periodic cleanup;
expiry is enforced on every state transition even if cleanup has not run.
Verifier ciphertext is removed upon successful exchange or final failure.
No external-token storage, email field, arbitrary account reassignment, or
admin bypass exists. Database ownership follows the application migrations.

## Proof obligations

- Header-only behavior, generated member schemas/results and HTTP roster
  behavior remain unchanged on both supported protocol revisions.
- OAuth metadata/route registration matches disabled/header/oauth/both exactly;
  wrong token classes never fall back. Strict headers, scope loss, missing scope,
  canonical audiences, current membership and tenant isolation are exercised.
- A real TLS issuer fixture provides discovery, JWKS, signed ID tokens, code
  exchange and introspection. Wrong signature/nonce/state/audience/client,
  expiry, duplicate callback, cancellation, oversize and issuer outage fail closed.
- Real PostgreSQL proves unique ownership, session-bound consent, no email
  linking, account/membership/token revocation and unlink on the next call.
  Controlled callback/confirm/unlink races prove no resurrection. Two separate
  browser contexts prove that A starting, B authenticating at the issuer, and A
  trying to confirm cannot bind B's external identity to A's Vector account.
- Browser proof covers start, callback, explicit confirm, connected state and
  disconnect with actual generated HTTP and isolated test issuer/database.
  Captured failed-request logs/traces contain correlation IDs and no credentials
  or callback bodies; verification includes the proxy.

## Tradeoffs and provider selection

Fresh introspection adds issuer latency and makes issuer availability part of
every call. Form-post plus final same-session confirmation adds one deliberate
step, keeps authorization codes out of browser URLs, and works with SameSite
cookies. The required introspection claim profile and stable subject constrain
eligible providers. Production issuer selection is deferred. The current implementation remains generic;
implementation and isolated issuer proof do not require deploying an issuer.

References: [MCP authorization](https://modelcontextprotocol.io/specification/2026-07-28/basic/authorization),
[RFC 7662](https://www.rfc-editor.org/rfc/rfc7662.html),
[RFC 9728](https://www.rfc-editor.org/rfc/rfc9728.html),
[OIDC Core](https://openid.net/specs/openid-connect-core-1_0.html),
[form-post response mode](https://openid.net/specs/oauth-v2-form-post-response-mode-1_0.html).
