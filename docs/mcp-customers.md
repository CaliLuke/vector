# Customer contracts

Customers, customer requests, property definitions/options and property values
belong to one native workspace catalogue. Current active workspace membership is
required. Guests read; workspace administrators write. Work-item associations also
require current writable project membership and visibility for every target. The
customers workspace feature gates these operations and replay; disabling it keeps
the data. Reads and related projections use a repeatable-read snapshot.

Customer create is an atomic upsert. A complete external_source/external_id pair
selects that identity; without the pair, an exact name match selects the customer.
Only supplied fields change. Names are not globally unique when external identities
differ. The catalogue lock serializes matching; external identity pairs are unique
among live customers. An ambiguous legacy name match is rejected. Delete by ID
requires the row; deleting an unmatched external pair succeeds. Stage and contract
status retain free-form strings. Domain is industry, not a hostname. Revenue is text.
Zero employees and empty optional strings retain the pinned tool's omitted sentinel.
Updates with no supplied changes preserve the saved fields.

Requests belong to exactly one customer. Creation can link work items atomically;
update changes metadata only. Request IDs are never usable under another customer.
Request deletion removes only its own associations. Customer deletion removes its
requests, values and associations but keeps the linked work items. Work-item lists
return each visible item once. Request-filtered unlink removes that request's link;
unfiltered unlink removes every link to the item for that customer. Unlink follows
link, so removal wins when both select an item. Association changes enqueue native
item activity in the same transaction and advance native item revisions.

Property display names are unique per workspace and generate a stable slug name.
Supported kinds are TEXT, DATETIME, DECIMAL, BOOLEAN, OPTION, RELATION, URL, EMAIL
and FILE. RELATION selects ISSUE or USER; values do not grant access to references.
TEXT/DATETIME formats have the pinned defaults. Type and display format remain fixed
after creation. Options have stable UUIDs; updates patch specified IDs or add new
options, retaining omitted options. Option-only updates keep the property definition
and option identities intact. Default flags respect single/multi cardinality.
Defaults accept a trimmed literal string, with commas preserved, or a JSON string
array. A literal beginning with `[` is still text when it is not valid JSON. Null
and non-string elements in JSON arrays are rejected. A blank default string omits the
field; `[]` removes the default.

Values map property UUIDs to string arrays. Setting a property replaces its array;
omitted properties are unchanged. Required properties cannot be cleared, single
properties accept at most one value, and inactive properties are excluded from reads.
Defaults apply to new customers, never to a clear operation. Required definitions
without defaults can be populated after customer creation. Option values use option
UUIDs; relation values use current visible item or workspace member UUIDs. Reads
mask references that are no longer visible. Dates accept YYYY-MM-DD, YYYY-MM-DD HH:MM:SS or RFC3339, booleans use
True/False, decimals are finite decimal strings. FILE retains opaque string metadata
like the existing native File property kind; it does not fetch or disclose assets.
Changing required/cardinality/relation rules validates existing saved values before
committing. Read-only text properties reject value writes. Property deletion removes
its values and hides its options while retaining immutable audit history.

Mutations support optional receipts and revision guards. Customer and property tools
use request_id for retries. Customer-request actions instead use mutation_id because
request_id already identifies the domain request; their receipts use mutation_id too.
Current scope, feature state and selected references are rechecked before replay.
Supplied property defaults require current item visibility, active workspace member
status or active option state on every retry. Retained property metadata binds those
checks after soft deletion. Receipts for property deletion remain replayable without
live targets.
Catalogue and association writers advance a workspace customer revision. Audit events
retain inputs and before/after state; request/body updates are not external effects.
Migrations refuse rollback that would discard customer data.

The pinned property update helper advertises is_multi but omits it from its payload.
The native contract applies the advertised field, validating retained values before
narrowing cardinality. [Upstream issue #254](https://github.com/makeplane/plane-mcp-server/issues/254)
tracks the omitted field. Remove this compatibility note when the pinned helper
forwards is_multi to its SDK update model.
Generated-server tests exercise all nineteen actions under both authentication modes
and protocol revisions. Customer property checks cover defaults, replacement values
and invalid-value preservation for all nine kinds; literal and JSON defaults;
option identity, omitted options and default changes; inactive and readonly values;
required-field clearing, cardinality narrowing and atomic bulk rejection. Clearing
an optional value keeps omitted properties and does not reapply defaults. Changing
option defaults leaves existing customer values unchanged. Default-reference retry
checks cover issue/member access revocation and restoration, inactive options,
soft-deleted definitions and cross-workspace property rejection.

Remaining acceptance includes upsert identity/concurrency, complete customer/request
binding and private-target filtering, the complete format and reference-visibility
matrix, atomic association changes, current authorization on replay, CAS and durable
activity, and full SDK page and record projections under both auth modes and MCP
revisions.
