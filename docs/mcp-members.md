# Member directories and role catalogue

## Summary

VEC-7 adds a read-only catalogue of Vector's existing membership roles and the
filters required by `member.list_workspace`. Both HTTP and MCP use authorized
application operations in `internal/projectmembershipreads`. Catalogue entries
describe existing roles; they do not create permissions or custom roles.

This contract is implemented for personal-token MCP and the HTTP member reads.
OAuth verification and overall MCP parity remain incomplete.

## Problem statement and current state

`Repository.Roles` returns the caller's project-to-numeric-role assignments. It
is not a catalogue and cannot implement `list_roles` or `retrieve_role`.
Workspace and project memberships use 20, 15 and 5. Workspaces also store an
`owner_id`. The product labels project role 15 Member; the pinned SDK calls it
Contributor. There is no operational project Commenter or custom-role domain.

The workspace members-lite HTTP endpoint uses `InvitationsRepository` and
`WorkspaceMemberLitePage`, which previously read a replica, had no actor in its
query, and only supported pagination. The new shared directory reader replaces
that unscoped page read. Its HTTP authorization allows active
workspace members with roles 15 or 20. The pinned member tool additionally
requires text, role and boolean filters and ordering.

## Goals and boundaries

Provide real, tenant-scoped member and role reads with filters applied before
counts and pagination. Preserve existing numeric authorization rules and current
HTTP field values. No membership writes, permission engine, custom-role editing,
or deployment enablement is included. Project Commenter cannot be advertised as
an available Vector role until its actual permissions are designed and enforced.

## Proposed design

### Role mapping

One domain declaration in `internal/projectmembershipreads` defines the mapping
used by catalogue results and member `role_slug` projections:

| Namespace | Slug        | Name        | Existing meaning                                | Sort order |
| --------- | ----------- | ----------- | ----------------------------------------------- | ---------- |
| workspace | owner       | Owner       | role 20 and member_id equals workspace.owner_id | 0          |
| workspace | admin       | Admin       | role 20, excluding the owner mapping            | 1          |
| workspace | member      | Member      | role 15                                         | 2          |
| workspace | guest       | Guest       | role 5                                          | 3          |
| project   | admin       | Admin       | role 20                                         | 0          |
| project   | contributor | Contributor | role 15                                         | 1          |
| project   | guest       | Guest       | role 5                                          | 2          |

The owner mapping does not grant permissions and requires an existing membership
row. An owner pointer alone never authenticates or authorizes anyone. Unknown
numeric membership roles retain their numeric value and project `role_slug:null`;
they never produce invented catalogue definitions. A role 15 or 5 owner is
projected according to its numeric membership, not as Owner.

Every authorized workspace catalogue contains all seven definitions, regardless
of whether any current member holds them. The existing-row requirement above
only governs member projection, not catalogue contents.

Each definition has exactly `id`, `name`, `slug`, `namespace`. IDs are UUIDv5 using
the immutable workspace UUID as namespace and the UTF-8 name
`vector.role/v1/<namespace>/<slug>`. IDs survive workspace renames and restarts,
differ across workspaces and namespaces, and need no persistence or seed rows.
These are identifiers, never bearer capabilities. No synthetic permission list,
member counts, timestamps, or editable role records are returned. The pinned SDK
requires only the three nullable text fields; `id` is a supported extra field.

### Authorization and ownership

A credential-neutral application reader accepts the authenticated actor UUID and
workspace slug. The shared application repository reads only the primary. Retained preliminary
HTTP authorization/existence checks still use their current replica selection;
a stale denial after a recent grant remains possible there. They cannot bypass
the final primary authorization or grant a stale read. Its scope query joins an
undeleted workspace to an active, undeleted actor membership and active actor
account. The member directory requires numeric role 15 or 20, matching HTTP.
The new role catalogue requires any of the supported workspace roles 5, 15, 20;
its definitions contain no member data or private project information.

For directory reads, authorization and the filtered roster/count come from one
SQL statement. For catalogue reads, the same primary scope statement authorizes
and resolves the immutable workspace ID; the immutable declaration is projected
only after that result succeeds. Revocation committed before that statement
prevents access. Neither path caches scope or treats an earlier HTTP/MCP
credential check as current domain authorization. Absent, deleted and inaccessible
workspaces return forbidden at the shared application boundary without disclosing
existence. The existing members-lite HTTP adapter retains its preliminary checks:
a successful legacy authorization followed by a failed WorkspaceExists check
returns the existing 404 before entering the application. If those checks pass
but final primary scope authorization fails, return 403. New role endpoints have
no legacy existence-check exception. After authorization,
an unknown or other-workspace role ID returns not_found.

### Projection surfaces and source of truth

The HTTP source contracts live in the existing
`apps/go-api/invitationsmembership/design` projection. New GET endpoints are
`/api/v1/workspaces/{slug}/roles/` and `/api/v1/workspaces/{slug}/roles/{role_id}/`.
They use the existing API-key protocol authentication and throttling, then the
shared application reader. MCP's authored action declarations stay in
`apps/go-api/mcp/design`; generated transport code is never edited. Shared HTTP
result type declarations are referenced by the MCP projection, with only pinned
SDK null/default normalization applied by MCP. The MCP result union has one `MemberDirectoryPage` branch for both directory
actions, with one concrete, fully typed entry carrier. Workspace rows contain exactly
the 11 pinned identity/membership fields, with explicit nulls retained; role rows
contain exactly id/name/slug/namespace. Action-specific typed application values
supply these projections. Unset carrier fields remain omitted, so role rows do
not acquire null identity fields and member rows do not acquire role-definition
fields. The carrier uses no Any or raw-map result fields.
[Loom #1036](https://github.com/CaliLuke/loom/issues/1036) tracks the generator's
nested untagged-union restriction. Replace the carrier with an entry union when
the minimum released generator supports that composition and generated tests
preserve exact empty and nonempty page output on both protocol revisions. It must not contain two page
branches distinguished only by their item type: their empty arrays have the same
wire shape and would make union serialization ambiguous. Wire authentication is not passed
into the domain reader. Neither transport calls an unscoped repository directly.

The members-lite endpoint gains optional filters and `role_slug` in its result.
Existing response fields and values remain unchanged. Required identity strings
(first_name, last_name, email, display_name) project a database null as an empty
string, matching the current HTTP typed serializer. The shared application
applies that rule so MCP sees the same values; nullable avatar fields stay null.
Filtering and ordering still use the original database values. New filter arguments are
additive. Its default page size remains 1000, and existing cursor/error parsing
remains in the HTTP adapter. The shared application accepts normalized page index
and size, not transport contexts or credentials. Existing HTTP authentication and
throttle ordering remains intact; a final application authorization follows it.
The unpaginated workspace-member route and project-member wire shape are unchanged.

## API and detailed behavior

| Capability     | Supported and default                                                                                                                                           | Rejected                                                          | Routing and equivalence                                                                                                      | Proof                                                                                             |
| -------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| list_workspace | Current workspace; default size 100 when MCP per_page is omitted or 0; size 1..1000; cursor defaults to first page                                              | Invalid size, cursor, ordering or missing workspace               | HTTP and MCP use the same filtered application page; HTTP retains default 1000; MCP fills absent SDK properties with null    | Filters and authorization precede counts; boolean false differs from absent; no cross-tenant rows |
| list_roles     | namespace omitted/empty means both; otherwise workspace/project; default size 20 when MCP per_page is omitted or 0; size 1..1000; first page when cursor absent | Other namespace, invalid size/cursor or missing workspace         | Both transports project the same authorized definitions; order namespace ascending, sort order ascending, then name, then id | Every listed id retrieves the identical definition; no unsupported roles                          |
| retrieve_role  | UUID identifier and workspace required                                                                                                                          | Malformed UUID; missing scope; authorized unknown id is not_found | Both transports use the same authorized workspace catalogue lookup                                                           | Cross-workspace IDs never resolve; inactive/deleted actors cannot read                            |

For MCP, empty filter/cursor/namespace/order strings mean absent, as in the pinned
tool. `first_name`, `last_name`, `email`, `display_name` use literal case-insensitive
substring matching and combine with AND. SQL wildcard characters in input are
literal characters. `role_slug` is an exact, case-sensitive match to the workspace
mapping; an unknown slug yields an empty page. `is_active` filters the target
membership flag; `is_bot` filters the target user flag. Both are tri-state and do
not remove inactive target memberships when absent. Soft-deleted memberships are
always excluded. The same joined relation supplies results and total counts.

Directory `order_by` accepts comma-separated fields, each optionally prefixed
with `-`: `created_at`, `updated_at`, `first_name`, `last_name`, `email`,
`display_name`, `role`, `is_active`, `is_bot`. Timestamps and role/is_active refer
to the membership; names, email and is_bot refer to the user. Default is
`-created_at`. Unknown fields, empty components and repeated fields are rejected.
SQL identifiers come only from this allowlist. Append membership id ascending as
a stable final tie-breaker. Text ordering uses the database's normal collation.

Workspace-directory cursors preserve the existing parser on both surfaces. Split
exactly three components. The size component is trimmed and parsed as a float
when it contains a dot, otherwise as an integer. Page and direction are untrimmed
integers. Page must be nonnegative; any nonzero direction means previous. Forward
requests ignore the cursor size; previous requests require it to equal the
requested size. Thus `100.0:1:2` with size 100 is accepted. Malformed cursors or
negative pages are invalid_params in MCP and retain the current HTTP 400 shape.
A previous-size mismatch retains HTTP 500 and becomes a sanitized MCP
internal_error, matching an upstream request failure. Integer offset overflow
fails before querying (HTTP 400 / MCP invalid_params).

New role endpoints use the same cursor acceptance and arithmetic. Malformed,
negative, mismatched previous-size or overflow cases return HTTP 400 / MCP
invalid_params, since there is no legacy Vector role endpoint to preserve.

| Surface                    | Absent size | Explicit zero     | Invalid filters and types                                                                                                                                                                                                               |
| -------------------------- | ----------- | ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Existing HTTP members-lite | 1000        | Existing HTTP 500 | New boolean filters accept only literal `true` or `false`; empty booleans, other strings, invalid ordering or duplicate new query parameters return HTTP 400; existing pagination parsing and repeated-parameter behavior are preserved |
| MCP list_workspace         | 100         | 100               | JSON boolean or null only; malformed filters/order/size return invalid_params; closed generated branches reject duplicate or cross-action arguments                                                                                     |
| New HTTP roles list        | 20          | HTTP 400          | namespace absent/empty means both; only workspace/project otherwise; size 1..1000; duplicate query parameters, malformed integer size and invalid namespace return HTTP 400                                                             |
| MCP list_roles             | 20          | 20                | namespace and size validation failures return invalid_params; size 1..1000 after defaulting                                                                                                                                             |

Empty HTTP text filters mean absent, matching MCP normalization. Whitespace is
literal filter text. For new HTTP filters, type validation occurs after existing
authentication/authorization/throttling; safe errors contain no SQL or raw input.
The new role endpoints likewise authenticate and throttle before application
validation and lookup. HTTP retrieve requires a UUID and returns 400 for malformed
IDs, 403 for inaccessible scope, and 404 for an unknown ID after scope authorization.

Paging computes offset = page * requested size, reads size+1, and trims the extra
row. The response contains count, total_count, total_results, total_pages,
next_cursor, prev_cursor, next_page_results, prev_page_results, results, and null
extra_stats/grouped_by/sub_grouped_by. Count is the current page length; totals
are the filtered count; total_pages is ceil(total/size). Next cursor is
`size:page+1:0`; previous is `size:page-1:1`, including -1 on the first page where
prev_page_results is false. A flagged-out cursor is not a usable request. Empty
pages and pages past the end still report accurate totals. Concurrent mutation
between separate page calls does not promise snapshot pagination.

MCP advertises only implemented action branches; catalogue and directory actions
are read actions available to the same credential classes as other member reads.
Workspace-specific authorization is rechecked at invocation. OAuth will additionally
require vector:read under its separate contract. No role or action aliases are
added. The generated root input remains a typed object with closed action
branches. Unknown actions, cross-action fields, invalid types and nonadvertised
branches fail invalid_params. Object results preserve the pinned object text and
structured-content shape on both supported protocol revisions.

## Proof obligations

Real PostgreSQL and generated-transport evidence must show primary reads,
revocation after request authentication, actor/account/workspace deletion,
cross-tenant isolation, all role mappings, stable IDs after workspace rename,
exact list/retrieve equivalence, and both MCP protocol revisions. Directory
fixtures must distinguish each filter, literal wildcard input, false from absent,
ties, empty pages and filtered totals. HTTP regressions must preserve existing
unfiltered field values, auth/throttle order, default pagination and legacy errors.
No passing role-read test constitutes proof of Commenter, custom roles, OAuth,
or overall MCP parity.
