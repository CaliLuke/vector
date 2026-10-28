# Work-item property API

Vector's workspace property catalogue remains the source of truth for MCP definitions,
options and type associations. The API maps SDK property names and settings to native
kinds and effective configurations. It keeps native rules for required values,
read-only properties, type availability and preservation of existing data.

Current known workspace members can read the workspace catalogue. Project reads
require current project membership and return effective properties of available
project types. Type-scoped lists fall back to project-flat and then workspace type
properties when empty, as in the pinned handler. A supplied project never grants
access to another project's names, references or configuration contexts.

Workspace administrators own definition, option and association changes. Creation
with a type can fall back from project scope to the workspace catalogue, after
checking project access. Other project-scoped catalogue mutations report workspace
ownership; clients can repeat them without project_id. Attach/detach batches are
atomic; detach wins. Deletion and destructive option/type changes retain the native
preservation requirements instead of discarding stored values or history.

Definitions and options keep API metadata alongside the native definition. Native
editor writes preserve that metadata. Optional request receipts authorize current
scope before replay. Catalogue mutations use the workspace property revision;
include_revision on reads provides the token. Every definition and association
writer contributes to the revision.

Item value reads, writes and clearing use native item validation and history,
with current item visibility and permissions. Rich text reuses asset ownership
checks. Values remain in issues.custom_properties; a companion table supplies
stable API identities and external metadata, synchronized by all item writers.
Clearing removes stored presence after required-field and read-only validation.
Formula and read-only text reads resolve the current effective configuration in
the same authorized snapshot. They return stable derived value IDs, with null
storage timestamps and external metadata. Empty formulas return a null value;
evaluation failures return a validation error. Their optional revision fingerprints
the computed response, since time or configuration can change a value without an
item write. These records remain read-only and do not create stored value rows.
Optional receipts reauthorize the current property scope before replay, and item
revisions cover both canonical values and companion metadata. All new paths
remain subject to consolidated integration and quality checks.

All fourteen property actions are wired. Email and File
are native bounded text kinds; Email validates an address. Existing native number,
boolean, text, date, dropdown, member, release, rich-text, formula and cascading
configuration rules remain in force. ISSUE relations validate newly submitted work-item references against current
actor visibility. Retained IDs follow native reference preservation without
revealing private target labels. Date values accept YYYY-MM-DD, YYYY-MM-DD HH:MM:SS,
and RFC3339, preserving the supplied time component in storage and value reads.
Formula inputs retain date-time precision. Native date filters compare UTC calendar
days, including values carrying a time. Email and File use text PQL operators.
Project-only and type-only catalogue reads resolve their effective context;
an existing property with no options returns an empty array, while a missing
property returns not_found. These behaviors have focused native regression proof.
Date display-format updates resolve the existing property kind when it is omitted.
Text formats remain immutable under the native configuration rules.
The optional Vector `configuration` extension supplies fields absent from the
pinned tool. Formula creation requires `expression` and `result` (`Number`,
`Text`, `Boolean`, or `Date`); expressions use the existing bounded native formula
language. Cascading creation requires two to four `levels` and can supply an
`options` tree. Levels have UUID `id`, `name`, and `required`; options have UUID
`id`, `name`, zero-based `level`, and optional `parent`, `icon`, and `next_label`.
Caller-supplied identities let children reference parents in the same request.
Read-only text accepts `configuration.content`, including an empty string.
Updates preserve omitted configuration fields and replace supplied arrays.
Retrieve/list expose these settings in `settings`, so clients can edit them
without losing identities. The generated-server configuration batch verifies these additions with both authentication modes and protocol revisions, including empty versus omitted inputs and preservation of stored cascading paths.

Native defaults remain limited to optional number, dropdown and member properties.
Used dropdown structures and destructive cascading edits retain native preservation
review requirements. MCP returns a validation/conflict error for those changes;
it does not discard existing values or bypass review. These restrictions are
intentional compatibility differences.

A single JSON-item array carries property rows, option rows or attached IDs, so an
empty result remains an unambiguous array. The receipt result also uses a JSON
carrier for [Loom #1036](https://github.com/CaliLuke/loom/issues/1036). Restore the
typed nested unions after released support and passing wire-contract checks.
Standalone records retain typed fields and required SDK nulls.

Create and update use separate generated configuration type names until
[Loom-MCP #385](https://github.com/CaliLuke/loom-mcp/issues/385) deduplicates shared
nested validators. The native cascading validator enforces the two-level minimum;
the optional transport array omits `MinLength` until
[Loom #1042](https://github.com/CaliLuke/loom/issues/1042) preserves absent arrays.
Remove these workarounds after released fixes pass the affected contract checks.
