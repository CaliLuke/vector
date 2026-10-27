# Pinned upstream MCP contract

`catalogue.json` records the tool signatures and action declarations from
[Plane MCP Server at dc7cfd9f3c305891909e2372f0cc507291353801](https://github.com/makeplane/plane-mcp-server/tree/dc7cfd9f3c305891909e2372f0cc507291353801),
which pins `plane-sdk==0.3.1`. The adjacent MIT license applies to this derived
fixture. Vector retains its own AGPL license.

This is expected parity data, not a runtime registry or a claim that the actions
are implemented. It contains 30 tools and 207 operations. The `read` record for
`get_pql_reference` identifies its sole operation; that tool has no `action`
parameter. Its `detail` parameter defaults to `full` and accepts `brief` or `full`.

The fixture preserves the order in `plane_mcp/tools/registry.py`, parameter
annotations and defaults from each registered function, and the `Action`
declarations from each resource module. Per-file SHA-256 hashes and source lines
locate the evidence. Mixed-action tool annotations remain conservative.

The snapshot was extracted from Python syntax trees without importing or running
the source. `Action` positional fields must follow the dataclass declaration in
`plane_mcp/toolkit/spec.py`: name, requires, optional, note, read, destructive,
open_world. Extraction must resolve tuple expansions and slices such as
`*QUERY_FIELDS` and `WRITE_FIELDS[1:]`; unresolved values are errors, not empty
argument lists. The action literals, declared argument names, uniqueness, counts
and reference-tool exception were checked against the source.

These declarations do not describe every domain rule or response shape. Each
action's parity test must also inspect its handler and the pinned SDK models,
then verify actual behavior through the generated Vector HTTP handler with both
authentication modes. Vector adds workspace selection and durable-write fields
according to its accepted MCP design; those are not upstream parameters.
