# Customer contracts

Customers, customer requests, property definitions/options and property values
belong to one native workspace catalogue. Current active workspace membership is
required. Guests read; workspace administrators write. Work-item associations also
require current writable project membership and visibility for every target. The
customers workspace feature gates these operations and replay; disabling it keeps
the data. Reads and related projections use a repeatable-read snapshot.

Customer create is an atomic upsert following the [pinned handler's identity contract](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/customer.py). A complete external_source/external_id pair
selects that identity; without the pair, an exact name match selects the customer.
Only supplied fields change. Names are not globally unique when external identities
differ. The catalogue lock serializes matching; external identity pairs are unique
among live customers. An ambiguous legacy name match is rejected. Delete by ID
requires the row; deleting an unmatched external pair succeeds. Stage and contract
status retain free-form strings. Domain is industry, not a hostname. Revenue is text.
Zero employees and empty optional strings retain the pinned tool's omitted sentinel.
Updates with no supplied changes preserve the saved fields. Deleted identities can
be recreated with a new customer ID. An earlier create receipt retains its original
result; replaying an earlier external-identity deletion cannot delete the replacement.

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

[Identity acceptance](../apps/go-api/internal/mcpactions/customer_identity_integration_test.go)
covers exact names, partial and complete external identities, ambiguity, collision
rollback, workspace binding and receipt replay across identity replacement under
both credentials and protocols. [Concurrent mutation tests](../apps/go-api/internal/mcpactions/customer_concurrency_integration_test.go)
verify one identity with disjoint field preservation, receipt conflicts and competing
revisions. [Recovery and authorization tests](../apps/go-api/internal/mcpactions/customer_identity_recovery_integration_test.go)
cover customer create, upsert, update and delete; request create, update and delete;
and property create, update, delete and value replacement. They exercise cancelled
waits, rejected commits, reusable receipts, and current token, scope and membership
after lock waits before
new writes or replay. The latter generated-server matrix covers both credentials
and protocols; concurrency and rollback use the same native repository. The
[current role and feature checks](../apps/go-api/internal/mcpactions/customer_identity_access_integration_test.go)
reject fresh customer, request and property writes and receipt replay for guests,
members and a disabled customers feature. Restoring administrator access and the feature replays the original result
without another mutation.

[Request ownership checks](../apps/go-api/internal/mcpactions/customer_request_ownership_integration_test.go)
reject another customer's request for reads, writes and association filters, and
reject redirected retry receipts without changing stored state.
[Linked-item visibility checks](../apps/go-api/internal/mcpactions/customer_request_visibility_integration_test.go)
hide draft and deleted items, archived projects and private projects after membership
revocation. Mixed-workspace request creation leaves no request, association, effect
or receipt; its mutation key remains reusable. These generated-server checks cover
both credentials and protocol revisions. Request projections omit linked IDs;
clients read current visible links through `customer.list_workitems`.
[Request-link authorization checks](../apps/go-api/internal/mcpactions/customer_request_target_authorization_integration_test.go)
reject new writes and receipt replay after project membership loss, a guest-role
change, project archival or deletion, and item draft or deletion. The same changes
committed by a lock-holding writer reject waiting requests without customer data,
receipt, revision or activity changes. Restoring target access replays the original
result without duplicating effects. These checks cover both credentials and protocols.
[Workspace binding checks](../apps/go-api/internal/mcpactions/customer_request_workspace_integration_test.go)
reject foreign customer and request IDs for every request action and association
filters, even when the actor has administrator membership in both workspaces.

[Request SDK projection checks](../apps/go-api/internal/mcpactions/customer_request_contract_integration_test.go)
verify the eight request fields and twelve pagination fields in
[plane-sdk 0.3.1](https://pypi.org/project/plane-sdk/0.3.1/) under both credentials
and protocols. They cover create/update/retrieve equivalence,
retained associations after metadata edits, next/previous cursor navigation,
default and empty pages, and search that excludes deleted and other customers'
requests.

[Property format checks](../apps/go-api/internal/mcpactions/customer_property_formats_integration_test.go)
cover all three text and four date display formats declared by the pinned SDK,
including omitted defaults, multiline values and readonly protection. Type and
format changes are rejected without stored effects, even when the replacement is
valid for creation. Metadata edits preserve the format. Unsupported formats are
rejected for every property kind. These checks use both credentials and protocols.

[Native revision checks](../apps/go-api/internal/mcpactions/customer_native_revision_integration_test.go)
cover direct inserts, updates and deletes in customer, request, association,
property, option and value rows. Every change invalidates an earlier catalogue
revision. A rejected stale mutation preserves the native result, audit and receipts;
refreshing its revision leaves the same retry key usable, with stable replay.

[Value-reference authorization checks](../apps/go-api/internal/mcpactions/customer_value_reference_authorization_integration_test.go)
cover private project membership loss, archived/deleted projects, draft/deleted
items, inactive users, inactive/deleted workspace memberships, unsupported roles,
and inactive/deleted options. Reads hide unavailable references. Fresh writes and
receipt replay reject them without changing customer state; restoring access
replays the original result without another mutation. The same changes committed
by a lock holder reject queued writes and replay. Expected state is read inside
that transaction so legitimate option and revision changes remain accounted for.
These checks use both credentials and protocol revisions.

[Definition retry checks](../apps/go-api/internal/mcpactions/customer_value_definition_authorization_integration_test.go)
use native property updates and deletion to reject fresh value writes and accepted
receipt replay after a definition is inactive or deleted. Reactivation restores
visible values and permits the original receipt to replay without another write.

[Reference workspace checks](../apps/go-api/internal/mcpactions/customer_reference_workspace_integration_test.go)
reject foreign items, members and options on value writes and default selection,
even when the actor administers both workspaces. Rejected writes leave their retry
keys usable. Reads mask imported foreign values and relation defaults; replay and
read masking preserve the imported stored state. Both definition and workspace
checks cover both credentials and protocol revisions.

[Default-reference wait checks](../apps/go-api/internal/mcpactions/customer_default_reference_wait_integration_test.go)
cover item/member selection on property creation and updates, and existing option
selection on updates. Access changes committed by the lock holder reject queued
new writes and receipt replay without changing customer state. Inline options on
creation have no pre-existing reference IDs. The checks use both credentials and
protocol revisions.

[Association transaction checks](../apps/go-api/internal/mcpactions/customer_association_atomicity_integration_test.go)
verify that a failed unlink rolls back earlier additions, activity, receipts and
both item and catalogue revisions. The rejected key remains reusable. Duplicate
adds preserve item revisions and enqueue no additional activity. Direct and
request-specific links project each item once; filtered removals and request
or customer deletion preserve other owners' links. Unfiltered unlink removes all
links for that customer, including links added in the same call. These checks
cover both credentials and protocol revisions.

[Durable customer activity checks](../apps/go-api/internal/mcpactions/customer_association_worker_integration_test.go)
queue direct and request-associated additions and removals while the worker is
stopped. Starting the real worker delivers each committed name snapshot even after
customer rename and deletion. Replaying deletion receipts and re-admitting completed
outbox identities adds no history or effects. Linked work items remain live.

[Customer and property SDK record checks](../apps/go-api/internal/mcpactions/customer_sdk_record_integration_test.go)
verify all 26 customer, 23 property and 17 option fields from the pinned SDK.
They cover create/retrieve equivalence, partial update preservation, actor and
workspace metadata, timestamps, option identity and live request counts excluding
soft-deleted requests.
[SDK page checks](../apps/go-api/internal/mcpactions/customer_sdk_pagination_integration_test.go)
verify all twelve envelope fields, forward/backward cursor navigation, default and
empty pages, case-insensitive customer search and exclusion of deleted or foreign
workspace rows. Both record and page checks use both credentials and protocols.

[Association mutation recovery](../apps/go-api/internal/mcpactions/customer_association_mutation_fixture_integration_test.go)
extends the shared contention, cancellation and rejected-commit tests to a mutation
that adds one item and removes another. Rollback preserves the complete customer
aggregate, linked item rows and native item revisions. Competing receipts and
revision guards admit one mutation; failed keys remain reusable. Guest/member
roles and a disabled feature reject fresh association writes and replay.
[Association target authorization](../apps/go-api/internal/mcpactions/customer_association_authorization_integration_test.go)
checks both added and removed items on fresh calls, replay and after lock waits,
including current project membership, guest roles, archival/deletion and item
visibility. The shared credential wait matrix also covers association writes.

[Read-access checks](../apps/go-api/internal/mcpactions/customer_read_access_integration_test.go)
exercise all eight read actions: guests and members retain reads, disabled features
reject reads, and inactive workspace membership is rejected before tool dispatch.
Restoring scope restores the same results without rewriting customer data.
[Linked-item SDK checks](../apps/go-api/internal/mcpactions/customer_linked_item_contract_integration_test.go)
verify the nine-field array projection and search by name, sequence number and
project identifier. Both sets cover both credentials and protocol revisions.

[Receipt timezone checks](../apps/go-api/internal/mcpactions/customer_timezone_integration_test.go)
cover customer create/upsert/update, request create/update and property
create/update, including nested options. Modeled timestamps are serialized in the
accepted timezone before saving the receipt. Changing the user's timezone does
not change that receipt's result or metadata on replay. Reads continue to use the
current preference. The timezone is excluded from mutation identity.

Receipts created before accepted-timezone serialization retain their stored
representation. Their timestamp instants are unchanged, but an earlier response's
user-specific offset cannot be reconstructed from those legacy receipts.
