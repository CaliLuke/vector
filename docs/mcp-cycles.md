# MCP cycle reads

The `cycle` tool implements `list` and `retrieve` from the pinned Plane MCP handler and `plane-sdk==0.3.1`. The other nine cycle actions remain unimplemented.

Both actions require `project_id`. Retrieve also requires `cycle_id`. Workspace selection follows the shared PAT/OAuth contract. Discovery advertises cycle reads only when the principal has an active membership in a live project in the selected workspace, or in any eligible workspace for context-free OAuth. Invocation requires access to the specific project.

List defaults to active cycles. `archived: true` selects archived cycles and ignores `status`, including otherwise invalid values. Active status values are `current`, `upcoming`, `completed`, `draft`, and `incomplete`. Active lists always return a page, including `status=current`. Zero or omitted `per_page` uses 1000; the accepted range otherwise is 1–1000. Empty cursors and ordering strings use server defaults. The existing API cursor format is retained. Active ordering supports created/updated timestamps, name, start/end dates, and sort order; archived lists preserve the API's fixed ordering.

Active list entries contain the 22 SDK `CycleLite` fields and omit metrics. Detail and archived entries contain the 31 SDK `Cycle` fields, including nullable issue counts and estimates. Missing detail estimates are null. Integral API estimates such as `3.0` become integer `3` without a floating-point conversion. Retrieve does not return archived cycles. An authorized empty collection is an empty page; lost authorization is an error.

Typed generated records and pages own the wire contract. A concrete entry carrier avoids the nested untagged-union limitation tracked in [Loom #1036](https://github.com/CaliLuke/loom/issues/1036). The projection enforces the action-specific fields and non-null lite ownership fields. Replace that carrier with the corresponding nested union when a released generator supports the composition and the empty-page/field-set tests pass. No dependency source is patched.

`internal/cyclereads` retains the existing HTTP projections. MCP binds an immutable workspace ID and checks the active account, live workspace/project, current workspace role, and active project membership in the primary query that returns data. Metric relationships must belong to that project and workspace. The HTTP design now accepts the canonical `status` parameter; full lists retain `cycle_view` as a fallback and their existing bare-array shape for current cycles. Lite lists do not use that alias.

HTTP and MCP share timezone conversion. Modeled timestamps use the user's current timezone and preserve microsecond precision; arbitrary JSON such as `progress_snapshot` is unchanged.

Generated-server PostgreSQL tests cover both credentials and protocol revisions, exact SDK field sets, filters and pagination, archived estimates, hidden/absent targets, empty pages, late authorization changes, cross-workspace relationships, and timezone behavior. Complete deployed issuer/browser proof and the broader 207-action catalogue remain open.
