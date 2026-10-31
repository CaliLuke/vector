# MCP action parity tracker

The scope is 30 tools and 208 operations, including the Vector `workspace.list` extension. The pinned [catalogue](../apps/go-api/mcp/testdata/plane-mcp/catalogue.json) supplies the expected names and input declarations. This tracker records implementation evidence; catalogue presence alone is not implementation. The user explicitly deferred teamspaces until native support exists. Teamspace query fields/functions are outside this delivery and return explicit errors; the reference lists that limitation.

All 208 actions are wired across 30 tools. The original 33 actions have detailed generated-server integration evidence, including current authorization, receipt and revision cases. Five subsequent workflow batches exercise another 174 actions with personal tokens and generic OAuth under both MCP protocol revisions. The attachment batch uses real disposable S3 storage. These counts describe workflow coverage, not full parity: compatibility, authorization/concurrency, and durable-effect acceptance remain open. Current-worktree browser evidence is recorded below.

OAuth tests use an isolated TLS issuer and real PostgreSQL. Production issuer selection is deferred. Discovery reflects current workspace and project membership; milestones also apply guest-feature and archived-project policy, while writes require the OAuth write scope. State discovery allows workspace aggregation and limits mutations to writable projects.

Local browser checks on 2026-10-27 verified native project, module, work-item and
page creation/editing against the MCP transport, native comments, a downloaded
attachment from real object storage, and custom-property edits in both directions.
A browser work-item edit defeated a stale MCP revision as intended. Page checks
exposed and fixed stale collaboration caches: an MCP replacement now opens with
only its replacement content, rejects old title/body saves with HTTP 409, and
accepts subsequent browser edits. The
[page collaboration regression](../apps/go-api/internal/mcpactions/page_collaboration_integration_test.go)
covers both authentication modes and protocol versions.

The local collector recorded the revision conflict with its request ID and trace.
A controlled outage of the isolated Redis service returned a sanitized MCP 503
and produced an error-level record with a matching error trace, without the token
or authorization header. Full frontend and Go gates passed at published revision
`1be8fcd5` on 2026-10-30, including 13,567 passing Go test events with no failures or skips.
The [debt catalog](tech-debt-catalog.md) records the remaining coverage and
acceptance work; a passing gate does not close those requirements.

The [consolidated catalogue workflow batch](../apps/go-api/internal/mcpactions/catalogue_batch_integration_test.go) now exercises the 37 customer, property, request, template, initiative and feature actions against the real generated server and PostgreSQL under both authentication modes and protocol revisions. It covers the primary lifecycles, selected receipt/authorization cases, atomic value rejection, association preservation, and opaque template JSON. This is preliminary integration evidence; exhaustive domain compatibility and permission/concurrency cases remain open; completed browser evidence is recorded above.

The [content workflow batch](../apps/go-api/internal/mcpactions/content_batch_integration_test.go) also passes the 43 page, collection and release actions across both authentication modes and protocol revisions, sharing one PostgreSQL fixture.

The [core workflow batch](../apps/go-api/internal/mcpactions/core_batch_integration_test.go) and [configuration workflow batch](../apps/go-api/internal/mcpactions/configuration_batch_integration_test.go) add passing coverage for 89 actions across both authentication modes and protocol revisions. These cover work-item queries and lifecycle, planning, comments, links, work logs, estimates, types, properties, relations and intake. Enabling intake now creates its default queue through the shared native helper. The [attachment workflow batch](../apps/go-api/internal/mcpactions/attachment_batch_integration_test.go) adds all five remaining actions with real RustFS object storage, including signed downloads, native text/image content, receipt replay without another source download, current-role rejection and private-address blocking. Successful source downloads use a controlled fetch boundary. All 207 pinned actions have generated-server workflow evidence; `workspace.list` has separate multi-workspace regressions. Passing these workflows does not replace the outstanding authorization, concurrency, durability and compatibility checks. The separate [property configuration batch](../apps/go-api/internal/mcpactions/property_configuration_batch_integration_test.go) passes formula creation/evaluation/update, fixed read-only content, cascading option paths, omitted versus empty fields, and preservation rejection under all four authentication/protocol combinations.

The [catalogue receipt batch](../apps/go-api/internal/mcpactions/catalogue_safety_integration_test.go) verifies customers, initiatives, releases, templates, pages and collections under all four authentication/protocol combinations. Direct database edits invalidate prior revisions; changed input conflicts with an existing request ID; revoked write access blocks update replay; deletion receipts replay after the target is deleted. Shared receipt-engine tests separately cover contention, cancellation and uncertain commits.

For each action, completion requires inspected pinned handler/SDK behavior, typed generated contracts, current authorization and tenant checks, both authentication modes and protocol revisions, and real PostgreSQL proof. Mutations additionally require durable receipt/replay, request-hash conflicts, current authorization on replay, CAS on all relevant writers, and durable effects. Asset operations require real object-storage and SSRF proof. Missing domain features require an accepted domain design before transport implementation.

Member proof: [identity and project members](../apps/go-api/internal/mcpactions/member_integration_test.go), [workspace directory and roles](../apps/go-api/internal/mcpactions/member_directory_integration_test.go), [OAuth invocation and revocation](../apps/go-api/internal/mcpactions/oauth_integration_test.go), and [domain authorization](../apps/go-api/internal/projectmembershipreads/directory_integration_test.go). [Member contract notes](mcp-members.md) record limits and compatibility decisions.

Project proof: [project contracts and primary data](../apps/go-api/internal/mcpactions/project_integration_test.go), [scope changes after admission](../apps/go-api/internal/mcpactions/scope_integration_test.go), and [repository revocation regressions](../apps/go-api/internal/workspaceprojectreads/mcp_authorization_integration_test.go). [Project contract notes](mcp-projects.md) record pagination, SDK projection and workspace binding. [Archive/restore proof](../apps/go-api/internal/mcpactions/project_archive_integration_test.go), [transaction and CAS regressions](../apps/go-api/internal/workspaceprojectreads/archive_integration_test.go), and [telemetry assertions](../apps/go-api/internal/mcpactions/mutation_observability_test.go) cover the first two mutations; [write recovery](mcp-writes.md) records the extension contract and remaining limits.

Cycle proof: [generated-server contracts, scope revocation, tenant isolation, and timezone behavior](../apps/go-api/internal/mcpactions/cycle_integration_test.go). [Cycle contract notes](mcp-cycles.md) record active/archived shapes and shared HTTP filtering.

Module and label proof: [generated-server contracts, scope revocation, and relationship isolation](../apps/go-api/internal/mcpactions/resource_read_integration_test.go). [Contract notes](mcp-modules-labels.md) record their different page shapes and archive policies.
The [planning query acceptance](mcp-cycles.md#planning-work-item-query-acceptance)
adds 32 applied-source cases for ordinary/property query snapshots, parent and
tenant scope, current access, filtered pagination, projection and recovery.
Four reference cases verify default/full/brief output and invalid-detail rejection
without a workspace selector. Both credentials and protocols are covered.
[Native browser acceptance](mcp-cycles.md#native-browser-acceptance) now proves
creation visibility, bidirectional edits and revision conflicts, lifecycle replay,
active-list visibility and fixture cleanup. Publication of the query test batch
completed at `81009902`: both full gates pass, with 13,951 Go test events and no failures or skipped tests. The 20 cycle/module actions and `get_pql_reference` meet their recorded acceptance requirements. Other action domains remain open.

Work-item child reads: [contract and scope notes](mcp-workitem-reads.md) and [generated-server regression tests](../apps/go-api/internal/mcpactions/workitem_read_integration_test.go).

Milestone proof: [native domain and contract](mcp-milestones.md), [generated-server lifecycle and receipts](../apps/go-api/internal/mcpactions/milestone_integration_test.go), [discovery and late scope changes](../apps/go-api/internal/mcpactions/milestone_scope_integration_test.go), and [sanitized telemetry](../apps/go-api/internal/mcpactions/milestone_observability_test.go). Native HTTP and MCP share the same seven operations, revision triggers, transactional outbox and worker delivery paths.

State proof: [contract](mcp-states.md), [generated-server lifecycle and receipts](../apps/go-api/internal/mcpactions/state_integration_test.go), [discovery and scope](../apps/go-api/internal/mcpactions/state_scope_integration_test.go), [telemetry](../apps/go-api/internal/mcpactions/state_observability_test.go), and [durable deletion](../apps/go-api/internal/backgroundtasks/state_deletion_integration_test.go). Native HTTP and MCP share mutation rules; revisions cover all state-row writers.

The [PQL foundation](pql.md) adds shared scalar, relationship (including native milestones), member, body, and effective custom-property filtering to HTTP work-item lists. The new core reads bind immutable workspace identity, current memberships, filters, pagination and expansions to one repeatable-read snapshot. Full grammar parity and integration proof remain pending. Grouped counts now use the same authorized item query and support all fourteen SDK dimensions, including workspace releases, with distinct-item totals and nested groups. The core workflow batch exercises query and transport behavior; broader query acceptance remains open. Expanded relation fields retain JSON carriers linked to Loom #1036, and property objects retain the map workaround linked to Loom #1041.

Attachment imports stage up to 5 MiB of bytes durably, then complete the immutable object upload through the request or effect worker. Inline reads use the configured object store directly and preserve native MCP image/text content. The attachment workflow batch verifies real storage, receipt replay and upload-intent completion. The [recovery tests](../apps/go-api/internal/mcpactions/attachment_recovery_integration_test.go) cover saved source bytes, native-worker upload and deletion activity, and deletion before upload. The [completion rollback test](../apps/go-api/internal/mcpactions/attachment_completion_rollback_integration_test.go) checks recovery after a successful object PUT and failed database completion. [Receipt tests](../apps/go-api/internal/mcpactions/attachment_receipts_integration_test.go) cover changed-input conflicts and native metadata revisions; [lock-wait tests](../apps/go-api/internal/mcpactions/attachment_authorization_wait_integration_test.go) cover current token, scope and project membership before new writes and replay. These generated-server cases run with personal tokens and OAuth under both protocol revisions. [Contention tests](../apps/go-api/internal/mcpactions/attachment_concurrency_integration_test.go) cover concurrent import/delete receipts and cancellation with reusable receipt keys. [Scope tests](../apps/go-api/internal/mcpactions/attachment_visibility_integration_test.go) reject cross-workspace and mismatched-parent access, including reads, writes and receipt replay after a parent becomes hidden. [Fetch-time authorization tests](../apps/go-api/internal/mcpactions/attachment_fetch_authorization_integration_test.go) reject revoked credentials or lost write scope before staging bytes. [Staging rollback tests](../apps/go-api/internal/mcpactions/attachment_staging_rollback_integration_test.go) verify unchanged database and object-store state after failed writes, followed by successful retry and replay. [UTF-8 compatibility tests](../apps/go-api/internal/mcpactions/attachment_text_compatibility_integration_test.go) preserve the pinned handler’s replacement boundaries; [storage-read tests](../apps/go-api/internal/assets/attachment_storage_close_test.go) preserve read, size-limit and response-close failures. [Metadata compatibility tests](../apps/go-api/internal/mcpactions/attachment_metadata_compatibility_integration_test.go) verify the SDK fields, timezone serialization, opaque JSON and download shape. [Import contracts](../apps/go-api/internal/mcpactions/attachment_import_contract_integration_test.go) preserve escaped and Unicode filenames, declared MIME priority and the inclusive 5 MiB upload limit. [Inline contracts](../apps/go-api/internal/mcpactions/attachment_inline_contract_integration_test.go) cover all advertised text/image MIME types, host-independent extension inference, inclusive text/image limits and native upload restrictions. [Native-parent lifecycle tests](../apps/go-api/internal/mcpactions/attachment_parent_lifecycle_integration_test.go) cover issue/project archive and deletion, receipt visibility and worker cascades with retained objects. [Public-fetch tests](../apps/go-api/internal/backgroundtasks/public_fetch_test.go) exercise the production pinned transport with controlled DNS and sockets: every redirect is revalidated, DNS rebinding and mixed private/public answers are rejected, webhook allowlists and proxy settings cannot bypass the policy, and response size, truncation and cancellation are bounded. All five attachment actions meet the recorded contract requirements. Native upload restrictions and completed-upload receipts remain intentional Vector behavior.

The [work-log contract](mcp-work-logs.md) defines minute-based entries, author/admin editing, all-writer revisions, optional receipts and native activity history. Its four actions and project summary have combined receipt, authorization, ownership, concurrency, rollback and native-worker acceptance linked from that contract.

| Operation                                  | Kind     | Required upstream inputs                                 | Vector implementation | Remaining proof                                                                       |
| ------------------------------------------ | -------- | -------------------------------------------------------- | --------------------- | ------------------------------------------------------------------------------------- |
| `customer.list`                            | Read     | None                                                     | Workflow verified     | Combined SDK, scope, feature and query checks                                         |
| `customer.retrieve`                        | Read     | `customer_id`                                            | Workflow verified     | Combined SDK, scope, feature and query checks                                         |
| `customer.create`                          | Mutation | `name`                                                   | Workflow verified     | Combined SDK, scope, receipts, CAS and effect checks                                  |
| `customer.update`                          | Mutation | `customer_id`                                            | Workflow verified     | Combined SDK, scope, receipts, CAS and effect checks                                  |
| `customer.delete`                          | Mutation | None                                                     | Workflow verified     | Combined SDK, scope, receipts, CAS and effect checks                                  |
| `customer.list_workitems`                  | Read     | `customer_id`                                            | Workflow verified     | Combined SDK, scope, feature and query checks                                         |
| `customer.manage_workitems`                | Mutation | `customer_id`                                            | Workflow verified     | Combined SDK, scope, receipts, CAS and effect checks                                  |
| `customer_property.list`                   | Read     | None                                                     | Workflow verified     | Combined SDK, scope, feature and query checks                                         |
| `customer_property.retrieve`               | Read     | `property_id`                                            | Workflow verified     | Combined SDK, scope, feature and query checks                                         |
| `customer_property.create`                 | Mutation | `display_name`, `property_type`                          | Workflow verified     | Combined SDK, scope, receipts, CAS and effect checks                                  |
| `customer_property.update`                 | Mutation | `property_id`                                            | Workflow verified     | Combined SDK, scope, receipts, CAS and effect checks                                  |
| `customer_property.delete`                 | Mutation | `property_id`                                            | Workflow verified     | Combined SDK, scope, receipts, CAS and effect checks                                  |
| `customer_property.get_values`             | Read     | `customer_id`                                            | Workflow verified     | Combined SDK, scope, feature and query checks                                         |
| `customer_property.set_values`             | Mutation | `customer_id`, `values`                                  | Workflow verified     | Combined SDK, scope, receipts, CAS and effect checks                                  |
| `customer_request.list`                    | Read     | `customer_id`                                            | Workflow verified     | Combined SDK, scope, feature and query checks                                         |
| `customer_request.retrieve`                | Read     | `customer_id`, `request_id`                              | Workflow verified     | Combined SDK, scope, feature and query checks                                         |
| `customer_request.create`                  | Mutation | `customer_id`, `name`                                    | Workflow verified     | Combined SDK, scope, receipts, CAS and effect checks                                  |
| `customer_request.update`                  | Mutation | `customer_id`, `request_id`                              | Workflow verified     | Combined SDK, scope, receipts, CAS and effect checks                                  |
| `customer_request.delete`                  | Mutation | `customer_id`, `request_id`                              | Workflow verified     | Combined SDK, scope, receipts, CAS and effect checks                                  |
| `cycle.list`                               | Read     | `project_id`                                             | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `cycle.retrieve`                           | Read     | `project_id`, `cycle_id`                                 | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `cycle.create`                             | Mutation | `project_id`, `name`, `owned_by`                         | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `cycle.update`                             | Mutation | `project_id`, `cycle_id`                                 | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `cycle.delete`                             | Mutation | `project_id`, `cycle_id`                                 | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `cycle.list_workitems`                     | Read     | `project_id`, `cycle_id`                                 | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `cycle.manage_workitems`                   | Mutation | `project_id`, `cycle_id`                                 | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `cycle.transfer_workitems`                 | Mutation | `project_id`, `cycle_id`, `new_cycle_id`                 | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `cycle.complete`                           | Mutation | `project_id`, `cycle_id`                                 | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `cycle.archive`                            | Mutation | `project_id`, `cycle_id`                                 | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `cycle.unarchive`                          | Mutation | `project_id`, `cycle_id`                                 | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `get_pql_reference`                        | Read     | None                                                     | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `initiative.list`                          | Read     | None                                                     | Workflow verified     | Combined scope, feature/governance, SDK, receipt, revision and effect checks          |
| `initiative.retrieve`                      | Read     | `initiative_id`                                          | Workflow verified     | Combined scope, feature/governance, SDK, receipt, revision and effect checks          |
| `initiative.create`                        | Mutation | `name`                                                   | Workflow verified     | Combined scope, feature/governance, SDK, receipt, revision and effect checks          |
| `initiative.update`                        | Mutation | `initiative_id`                                          | Workflow verified     | Combined scope, feature/governance, SDK, receipt, revision and effect checks          |
| `initiative.delete`                        | Mutation | `initiative_id`                                          | Workflow verified     | Combined scope, feature/governance, SDK, receipt, revision and effect checks          |
| `initiative.list_projects`                 | Read     | `initiative_id`                                          | Workflow verified     | Combined scope, feature/governance, SDK, receipt, revision and effect checks          |
| `initiative.add_projects`                  | Mutation | `initiative_id`, `project_ids`                           | Workflow verified     | Combined scope, feature/governance, SDK, receipt, revision and effect checks          |
| `initiative.remove_projects`               | Mutation | `initiative_id`, `project_ids`                           | Workflow verified     | Combined scope, feature/governance, SDK, receipt, revision and effect checks          |
| `initiative.list_workitems`                | Read     | `initiative_id`                                          | Workflow verified     | Combined scope, feature/governance, SDK, receipt, revision and effect checks          |
| `initiative.manage_workitems`              | Mutation | `initiative_id`                                          | Workflow verified     | Combined scope, feature/governance, SDK, receipt, revision and effect checks          |
| `intake.list`                              | Read     | `project_id`                                             | Workflow verified     | Combined queue, triage, visibility, revision and effects checks                       |
| `intake.retrieve`                          | Read     | `project_id`, `workitem_id`                              | Workflow verified     | Combined queue, triage, visibility, revision and effects checks                       |
| `intake.create`                            | Mutation | `project_id`, `name`                                     | Workflow verified     | Combined queue, triage, visibility, revision and effects checks                       |
| `intake.update`                            | Mutation | `project_id`, `workitem_id`                              | Workflow verified     | Combined queue, triage, visibility, revision and effects checks                       |
| `intake.delete`                            | Mutation | `project_id`, `workitem_id`                              | Workflow verified     | Combined queue, triage, visibility, revision and effects checks                       |
| `label.list`                               | Read     | `project_id`                                             | Contract verified     | Verified; see module/label contract and browser evidence                              |
| `label.retrieve`                           | Read     | `project_id`, `label_id`                                 | Contract verified     | Verified; see module/label contract and browser evidence                              |
| `label.create`                             | Mutation | `project_id`, `name`                                     | Contract verified     | Verified; see module/label acceptance evidence                                        |
| `label.update`                             | Mutation | `project_id`, `label_id`                                 | Contract verified     | Verified; see module/label acceptance evidence                                        |
| `label.delete`                             | Mutation | `project_id`, `label_id`                                 | Contract verified     | Verified; see module/label acceptance evidence                                        |
| `member.me`                                | Read     | None                                                     | Contract verified     | Verified; see member/native HTTP and workspace read acceptance                        |
| `member.list_workspace`                    | Read     | None                                                     | Contract verified     | Verified; see member/native HTTP and workspace read acceptance                        |
| `member.list_project`                      | Read     | `project_id`                                             | Contract verified     | Verified; see member/native HTTP and workspace read acceptance                        |
| `member.list_roles`                        | Read     | None                                                     | Contract verified     | Verified; see member/native HTTP and workspace read acceptance                        |
| `member.retrieve_role`                     | Read     | `role_id`                                                | Contract verified     | Verified; see member/native HTTP and workspace read acceptance                        |
| `milestone.list`                           | Read     | `project_id`                                             | Implemented           | Current-worktree live proof                                                           |
| `milestone.retrieve`                       | Read     | `project_id`, `milestone_id`                             | Implemented           | Current-worktree live proof                                                           |
| `milestone.create`                         | Mutation | `project_id`, `title`                                    | Implemented           | Current-worktree live proof                                                           |
| `milestone.update`                         | Mutation | `project_id`, `milestone_id`                             | Implemented           | Current-worktree live proof                                                           |
| `milestone.delete`                         | Mutation | `project_id`, `milestone_id`                             | Implemented           | Current-worktree live proof                                                           |
| `milestone.list_workitems`                 | Read     | `project_id`, `milestone_id`                             | Implemented           | Current-worktree live proof                                                           |
| `milestone.manage_workitems`               | Mutation | `project_id`, `milestone_id`                             | Implemented           | Current-worktree live proof                                                           |
| `module.list`                              | Read     | `project_id`                                             | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `module.retrieve`                          | Read     | `project_id`, `module_id`                                | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `module.create`                            | Mutation | `project_id`, `name`                                     | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `module.update`                            | Mutation | `project_id`, `module_id`                                | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `module.delete`                            | Mutation | `project_id`, `module_id`                                | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `module.list_workitems`                    | Read     | `project_id`, `module_id`                                | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `module.manage_workitems`                  | Mutation | `project_id`, `module_id`                                | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `module.archive`                           | Mutation | `project_id`, `module_id`                                | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `module.unarchive`                         | Mutation | `project_id`, `module_id`                                | Contract verified     | Verified; see planning contract, query and browser evidence at `81009902`             |
| `page.list`                                | Read     | None                                                     | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `page.retrieve`                            | Read     | `page_id`                                                | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `page.create`                              | Mutation | `name`, `description_html`                               | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `page.update`                              | Mutation | `page_id`                                                | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `page.archive`                             | Mutation | `page_id`                                                | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `page.delete`                              | Mutation | `page_id`                                                | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `page.set_collection`                      | Mutation | `page_id`, `collection_id`                               | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `page.list_workitem_pages`                 | Read     | `project_id`, `workitem_id`                              | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `page.attach_to_workitem`                  | Mutation | `project_id`, `workitem_id`, `page_id`                   | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `page.detach_from_workitem`                | Mutation | `project_id`, `workitem_id`, `workitem_page_id`          | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `project.list`                             | Read     | None                                                     | Read verified         | None                                                                                  |
| `project.retrieve`                         | Read     | `project_id`                                             | Read verified         | None                                                                                  |
| `project.create`                           | Mutation | `name`, `identifier`                                     | Write verified        | None                                                                                  |
| `project.update`                           | Mutation | `project_id`                                             | Write verified        | None                                                                                  |
| `project.delete`                           | Mutation | `project_id`                                             | Write verified        | None                                                                                  |
| `project.archive`                          | Mutation | `project_id`                                             | Write verified        | None                                                                                  |
| `project.unarchive`                        | Mutation | `project_id`                                             | Write verified        | None                                                                                  |
| `project.worklog_summary`                  | Read     | `project_id`                                             | Read verified         | None; see work-log contract and acceptance tests                                      |
| `project.get_features`                     | Read     | `project_id`                                             | Read verified         | None for the documented project feature read contract                                 |
| `project.update_features`                  | Mutation | `project_id`                                             | Write verified        | None for the documented project feature write contract                                |
| `project_estimate.retrieve`                | Read     | `project_id`                                             | Workflow verified     | Combined regression, authorization, receipt and effect checks                         |
| `project_estimate.create`                  | Mutation | `project_id`, `name`                                     | Workflow verified     | Combined regression, authorization, receipt and effect checks                         |
| `project_estimate.update`                  | Mutation | `project_id`                                             | Workflow verified     | Combined regression, authorization, receipt and effect checks                         |
| `project_estimate.delete`                  | Mutation | `project_id`                                             | Workflow verified     | Combined regression, authorization, receipt and effect checks                         |
| `project_estimate.link`                    | Mutation | `project_id`, `estimate_id`                              | Workflow verified     | Combined regression, authorization, receipt and effect checks                         |
| `project_estimate.list_points`             | Read     | `project_id`, `estimate_id`                              | Workflow verified     | Combined regression, authorization, receipt and effect checks                         |
| `project_estimate.create_points`           | Mutation | `project_id`, `estimate_id`, `points`                    | Workflow verified     | Combined regression, authorization, receipt and effect checks                         |
| `project_estimate.update_point`            | Mutation | `project_id`, `estimate_id`, `estimate_point_id`         | Workflow verified     | Combined regression, authorization, receipt and effect checks                         |
| `project_estimate.delete_point`            | Mutation | `project_id`, `estimate_id`, `estimate_point_id`         | Workflow verified     | Combined regression, authorization, receipt and effect checks                         |
| `release.list`                             | Read     | None                                                     | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release.retrieve`                         | Read     | `release_id`                                             | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release.create`                           | Mutation | `name`                                                   | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release.update`                           | Mutation | `release_id`                                             | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release.delete`                           | Mutation | `release_id`                                             | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release.get_changelog`                    | Read     | `release_id`                                             | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release.update_changelog`                 | Mutation | `release_id`                                             | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release.list_workitems`                   | Read     | `release_id`                                             | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release.manage_workitems`                 | Mutation | `release_id`                                             | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release_label.list`                       | Read     | None                                                     | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release_label.create`                     | Mutation | `name`                                                   | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release_label.update`                     | Mutation | `label_id`                                               | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release_label.delete`                     | Mutation | `label_id`                                               | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release_label.attach`                     | Mutation | `release_id`, `label_ids`                                | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release_label.detach`                     | Mutation | `release_id`, `label_ids`                                | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release_tag.list`                         | Read     | None                                                     | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release_tag.retrieve`                     | Read     | `tag_id`                                                 | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release_tag.create`                       | Mutation | `version`                                                | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release_tag.update`                       | Mutation | `tag_id`                                                 | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `release_tag.delete`                       | Mutation | `tag_id`                                                 | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `state.list`                               | Read     | None                                                     | Contract verified     | Verified; see state acceptance evidence below                                         |
| `state.retrieve`                           | Read     | `state_id`                                               | Contract verified     | Verified; see state acceptance evidence below                                         |
| `state.create`                             | Mutation | `name`, `color`                                          | Contract verified     | Verified; see state acceptance evidence below                                         |
| `state.update`                             | Mutation | `state_id`                                               | Contract verified     | Verified; see state acceptance evidence below                                         |
| `state.delete`                             | Mutation | `state_id`                                               | Contract verified     | Verified; see state acceptance evidence below                                         |
| `work_log.list`                            | Read     | `project_id`, `workitem_id`                              | Read verified         | None; see work-log contract and acceptance tests                                      |
| `work_log.create`                          | Mutation | `project_id`, `workitem_id`, `duration`                  | Write verified        | None; see work-log contract and acceptance tests                                      |
| `work_log.update`                          | Mutation | `project_id`, `workitem_id`, `work_log_id`               | Write verified        | None; see work-log contract and acceptance tests                                      |
| `work_log.delete`                          | Mutation | `project_id`, `workitem_id`, `work_log_id`               | Write verified        | None; see work-log contract and acceptance tests                                      |
| `workitem.list`                            | Read     | None                                                     | Workflow verified     | Combined query, projection, scope and protocol checks                                 |
| `workitem.list_archived`                   | Read     | `project_id`                                             | Workflow verified     | Combined query, projection, scope and protocol checks                                 |
| `workitem.retrieve`                        | Read     | `project_id`, `workitem_id`                              | Workflow verified     | Combined query, projection, scope and protocol checks                                 |
| `workitem.retrieve_by_identifier`          | Read     | `workitem_identifier`                                    | Workflow verified     | Combined query, projection, scope and protocol checks                                 |
| `workitem.search`                          | Read     | `query`                                                  | Workflow verified     | Combined query, projection, scope and protocol checks                                 |
| `workitem.count`                           | Read     | None                                                     | Workflow verified     | Combined catalogue, ownership, association, counts, revision and effects checks       |
| `workitem.create`                          | Mutation | `project_id`, `name`                                     | Workflow verified     | Combined regression, authorization, receipt and native effect checks                  |
| `workitem.update`                          | Mutation | `project_id`, `workitem_id`                              | Workflow verified     | Combined regression, authorization, receipt and native effect checks                  |
| `workitem.delete`                          | Mutation | `project_id`, `workitem_id`                              | Workflow verified     | Combined regression, authorization, receipt and native effect checks                  |
| `workitem.archive`                         | Mutation | `project_id`, `workitem_id`                              | Workflow verified     | Combined regression, authorization, receipt and native effect checks                  |
| `workitem.manage_assignee`                 | Mutation | `project_id`, `workitem_id`                              | Workflow verified     | Combined regression, authorization, receipt and native effect checks                  |
| `workitem.manage_label`                    | Mutation | `project_id`, `workitem_id`                              | Workflow verified     | Combined regression, authorization, receipt and native effect checks                  |
| `workitem_activity.list`                   | Read     | `project_id`, `workitem_id`                              | Implemented           | Local browser/proxy verification                                                      |
| `workitem_activity.retrieve`               | Read     | `project_id`, `workitem_id`, `activity_id`               | Implemented           | Local browser/proxy verification                                                      |
| `workitem_attachment.list`                 | Read     | `project_id`, `workitem_id`                              | Contract verified     | None                                                                                  |
| `workitem_attachment.read`                 | Read     | `project_id`, `workitem_id`, `attachment_id`             | Contract verified     | None                                                                                  |
| `workitem_attachment.download_url`         | Read     | `project_id`, `workitem_id`, `attachment_id`             | Contract verified     | None                                                                                  |
| `workitem_attachment.upload_from_url`      | Mutation | `project_id`, `workitem_id`, `url`                       | Contract verified     | None                                                                                  |
| `workitem_attachment.delete`               | Mutation | `project_id`, `workitem_id`, `attachment_id`             | Contract verified     | None                                                                                  |
| `workitem_comment.list`                    | Read     | `project_id`, `workitem_id`                              | Implemented           | Local browser/proxy verification                                                      |
| `workitem_comment.retrieve`                | Read     | `project_id`, `workitem_id`, `comment_id`                | Implemented           | Local browser/proxy verification                                                      |
| `workitem_comment.create`                  | Mutation | `project_id`, `workitem_id`, `comment_html`              | Workflow verified     | Combined mention, authorization, receipt and delivery checks                          |
| `workitem_comment.update`                  | Mutation | `project_id`, `workitem_id`, `comment_id`                | Workflow verified     | Combined mention, authorization, receipt and delivery checks                          |
| `workitem_comment.delete`                  | Mutation | `project_id`, `workitem_id`, `comment_id`                | Workflow verified     | Combined mention, authorization, receipt and delivery checks                          |
| `workitem_link.list`                       | Read     | `project_id`, `workitem_id`                              | Implemented           | Local browser/proxy verification                                                      |
| `workitem_link.retrieve`                   | Read     | `project_id`, `workitem_id`, `link_id`                   | Implemented           | Local browser/proxy verification                                                      |
| `workitem_link.create`                     | Mutation | `project_id`, `workitem_id`, `url`                       | Workflow verified     | Combined regression, authorization, receipt and effect checks                         |
| `workitem_link.update`                     | Mutation | `project_id`, `workitem_id`, `link_id`                   | Workflow verified     | Combined regression, authorization, receipt and effect checks                         |
| `workitem_link.delete`                     | Mutation | `project_id`, `workitem_id`, `link_id`                   | Workflow verified     | Combined regression, authorization, receipt and effect checks                         |
| `workitem_property.list`                   | Read     | None                                                     | Workflow verified     | Native mapping compatibility and combined scope, revision, preservation checks        |
| `workitem_property.retrieve`               | Read     | `workitem_property_id`                                   | Workflow verified     | Native mapping compatibility and combined scope, revision, preservation checks        |
| `workitem_property.create`                 | Mutation | `display_name`, `property_type`                          | Workflow verified     | Native mapping compatibility and combined scope, revision, preservation checks        |
| `workitem_property.update`                 | Mutation | `workitem_property_id`                                   | Workflow verified     | Native mapping compatibility and combined scope, revision, preservation checks        |
| `workitem_property.delete`                 | Mutation | `workitem_property_id`                                   | Workflow verified     | Native mapping compatibility and combined scope, revision, preservation checks        |
| `workitem_property.manage_type_properties` | Mutation | `workitem_type_id`                                       | Workflow verified     | Native mapping compatibility and combined scope, revision, preservation checks        |
| `workitem_property.list_options`           | Read     | `property_id`                                            | Workflow verified     | Native mapping compatibility and combined scope, revision, preservation checks        |
| `workitem_property.retrieve_option`        | Read     | `property_id`, `option_id`                               | Workflow verified     | Native mapping compatibility and combined scope, revision, preservation checks        |
| `workitem_property.create_option`          | Mutation | `property_id`, `name`                                    | Workflow verified     | Native mapping compatibility and combined scope, revision, preservation checks        |
| `workitem_property.update_option`          | Mutation | `property_id`, `option_id`                               | Workflow verified     | Native mapping compatibility and combined scope, revision, preservation checks        |
| `workitem_property.delete_option`          | Mutation | `property_id`, `option_id`                               | Workflow verified     | Native mapping compatibility and combined scope, revision, preservation checks        |
| `workitem_property.get_value`              | Read     | `project_id`, `workitem_id`, `property_id`               | Workflow verified     | Combined value, asset, visibility, history and revision checks                        |
| `workitem_property.set_value`              | Mutation | `project_id`, `workitem_id`, `property_id`               | Workflow verified     | Combined value, asset, visibility, history and revision checks                        |
| `workitem_property.delete_value`           | Mutation | `project_id`, `workitem_id`, `property_id`               | Workflow verified     | Combined value, asset, visibility, history and revision checks                        |
| `workitem_relation.list`                   | Read     | `project_id`, `workitem_id`                              | Workflow verified     | Combined direction, endpoint scope, definition, receipt and activity checks           |
| `workitem_relation.create`                 | Mutation | `project_id`, `workitem_id`, `workitem_ids`              | Workflow verified     | Combined direction, endpoint scope, definition, receipt and activity checks           |
| `workitem_relation.delete`                 | Mutation | `project_id`, `workitem_id`, `related_workitem_id`       | Workflow verified     | Combined direction, endpoint scope, definition, receipt and activity checks           |
| `workitem_relation.list_definitions`       | Read     | None                                                     | Workflow verified     | Combined direction, endpoint scope, definition, receipt and activity checks           |
| `workitem_relation.create_definition`      | Mutation | `name`                                                   | Workflow verified     | Combined direction, endpoint scope, definition, receipt and activity checks           |
| `workitem_relation.update_definition`      | Mutation | `definition_id`                                          | Workflow verified     | Combined direction, endpoint scope, definition, receipt and activity checks           |
| `workitem_relation.delete_definition`      | Mutation | `definition_id`                                          | Workflow verified     | Combined direction, endpoint scope, definition, receipt and activity checks           |
| `workitem_type.list`                       | Read     | None                                                     | Workflow verified     | Combined regression, authorization, receipt and native policy checks                  |
| `workitem_type.retrieve`                   | Read     | `workitem_type_id`                                       | Workflow verified     | Combined regression, authorization, receipt and native policy checks                  |
| `workitem_type.resolve`                    | Mutation | `project_id`, `name`                                     | Workflow verified     | Combined regression, authorization, receipt and native policy checks                  |
| `workitem_type.create`                     | Mutation | `name`                                                   | Workflow verified     | Combined regression, authorization, receipt and native policy checks                  |
| `workitem_type.update`                     | Mutation | `workitem_type_id`                                       | Workflow verified     | Combined regression, authorization, receipt and native policy checks                  |
| `workitem_type.delete`                     | Mutation | `workitem_type_id`                                       | Workflow verified     | Combined regression, authorization, receipt and native policy checks                  |
| `workitem_type.import_to_project`          | Mutation | `project_id`, `workitem_type_ids`                        | Workflow verified     | Combined regression, authorization, receipt and native policy checks                  |
| `workspace.list`                           | Read     | None                                                     | Contract verified     | Verified; see member/native HTTP and workspace read acceptance                        |
| `workspace.retrieve`                       | Read     | None                                                     | Read verified         | None for the documented identity contract; see workspace evidence                     |
| `workspace.get_features`                   | Read     | None                                                     | Read verified         | None for the documented workspace feature read contract                               |
| `workspace.update_features`                | Mutation | None                                                     | Write verified        | None for the documented workspace feature write contract                              |
| `template.list`                            | Read     | `kind`                                                   | Workflow verified     | Combined scope, feature/governance, SDK, receipt, revision and effect checks          |
| `template.create`                          | Mutation | `kind`, `name`, `template_data`                          | Workflow verified     | Combined scope, feature/governance, SDK, receipt, revision and effect checks          |
| `template.update`                          | Mutation | `kind`, `template_id`                                    | Workflow verified     | Combined scope, feature/governance, SDK, receipt, revision and effect checks          |
| `template.delete`                          | Mutation | `kind`, `template_id`                                    | Workflow verified     | Combined scope, feature/governance, SDK, receipt, revision and effect checks          |
| `collection.list`                          | Read     | None                                                     | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `collection.retrieve`                      | Read     | `collection_id`                                          | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `collection.create`                        | Mutation | `name`                                                   | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `collection.update`                        | Mutation | `collection_id`                                          | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `collection.delete`                        | Mutation | `collection_id`                                          | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `collection.list_pages`                    | Read     | `collection_id`                                          | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `collection.search_pages`                  | Read     | `collection_id`                                          | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `collection.add_pages`                     | Mutation | `collection_id`, `page_ids`                              | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `collection.remove_page`                   | Mutation | `collection_id`, `page_collection_id`                    | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `collection.list_members`                  | Read     | `collection_id`                                          | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `collection.add_member`                    | Mutation | `collection_id`, `user_id`, `member_access`              | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `collection.update_member`                 | Mutation | `collection_id`, `collection_member_id`, `member_access` | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |
| `collection.remove_member`                 | Mutation | `collection_id`, `collection_member_id`                  | Workflow verified     | Combined privacy, hierarchy, membership, receipt, revision and durable history checks |

## Contract sources

The fixture pins Plane MCP Server revision `dc7cfd9f3c305891909e2372f0cc507291353801` and `plane-sdk==0.3.1`. Its adjacent MIT license applies to the derived catalogue; Vector retains its AGPL license. Each source below must be read together with the pinned SDK models before its action is marked complete.

- `customer`: [plane_mcp/tools/customer.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/customer.py)
- `customer_property`: [plane_mcp/tools/customer_property.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/customer_property.py)
- `customer_request`: [plane_mcp/tools/customer_request.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/customer_request.py)
- `cycle`: [plane_mcp/tools/cycle.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/cycle.py)
- `get_pql_reference`: [plane_mcp/tools/get_pql_reference.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/get_pql_reference.py)
- `initiative`: [plane_mcp/tools/initiative.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/initiative.py)
- `intake`: [plane_mcp/tools/intake.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/intake.py)
- `label`: [plane_mcp/tools/label.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/label.py)
- `member`: [plane_mcp/tools/member.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/member.py)
- `milestone`: [plane_mcp/tools/milestone.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/milestone.py)
- `module`: [plane_mcp/tools/module.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/module.py)
- `page`: [plane_mcp/tools/page.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/page.py)
- `project`: [plane_mcp/tools/project.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/project.py)
- `project_estimate`: [plane_mcp/tools/project_estimate.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/project_estimate.py)
- `release`: [plane_mcp/tools/release.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/release.py)
- `release_label`: [plane_mcp/tools/release_label.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/release_label.py)
- `release_tag`: [plane_mcp/tools/release_tag.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/release_tag.py)
- `state`: [plane_mcp/tools/state.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/state.py)
- `work_log`: [plane_mcp/tools/work_log.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/work_log.py)
- `workitem`: [plane_mcp/tools/workitem.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/workitem.py)
- `workitem_activity`: [plane_mcp/tools/workitem_activity.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/workitem_activity.py)
- `workitem_attachment`: [plane_mcp/tools/workitem_attachment.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/workitem_attachment.py)
- `workitem_comment`: [plane_mcp/tools/workitem_comment.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/workitem_comment.py)
- `workitem_link`: [plane_mcp/tools/workitem_link.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/workitem_link.py)
- `workitem_property`: [plane_mcp/tools/workitem_property.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/workitem_property.py)
- `workitem_relation`: [plane_mcp/tools/workitem_relation.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/workitem_relation.py)
- `workitem_type`: [plane_mcp/tools/workitem_type.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/workitem_type.py)
- `workspace`: [plane_mcp/tools/workspace.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/workspace.py)
- `template`: [plane_mcp/tools/template.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/template.py)
- `collection`: [plane_mcp/tools/collection.py](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/collection.py)

[Page and collection contracts](mcp-pages-collections.md) cover workspace pages, project lifecycle reuse, hierarchy, private collection membership, work-item links and durable body-history admission. The generated-server content batch covers all 23 actions across both authentication modes and protocol revisions.

[Templates](mcp-templates.md), [features/initiatives](mcp-features-initiatives.md), and [customers](mcp-customers.md) are wired and covered by the generated-server workspace catalogue batch. Full acceptance still requires the remaining domain checks and compatibility review.

The `workspace.list` extension returns the current user's active workspace memberships
with IDs, names and slugs. Generated-server regressions cover listing and per-call
workspace selection under personal tokens and OAuth on both protocol revisions.
Header defaults and conflicts remain supported. Unscoped discovery merges each
workspace's current capabilities; invocation and receipt replay retain current
credential, workspace and domain authorization.

### Workspace read acceptance (MCP-01)

The [workspace selection regressions](../apps/go-api/internal/mcpactions/workspace_selection_integration_test.go)
exercise the generated server and PostgreSQL with personal tokens and OAuth under
both protocol revisions. They prove exact ordered workspace lists, exclusion of
another user's workspace, per-call project isolation, missing selectors, and
legacy header defaults/conflicts. The live-access matrix covers inactive,
soft-deleted, removed, and invalid-role memberships and deleted workspaces.
Revocation hides only the affected workspace, denies its workspace/project reads,
and preserves access to the other workspace. Token workspace metadata and another
user's surviving membership do not grant access.

Credential changes between authentication and invocation are covered by the
personal-token tests in that file and the [OAuth read regressions](../apps/go-api/internal/mcpactions/oauth_integration_test.go).
OAuth cases cover token, scope, issuer availability, account linking, account
activity, client identity, and link reassignment. Rejected calls return no
structured data. Losing the last membership returns a successful empty list for
both credentials; it still rejects a scoped member read. The focused selection,
live-access, and late-revalidation run has 62 leaf cases across four tests.

This closes the recorded `workspace.list` read-contract proof gap. Workspace
retrieval now asserts the exact four-field connection metadata for each known
input credential. OAuth returns the selected workspace's ID and name (distinct
from its slug); personal tokens retain null ID/name and `api_key_header`. This
matches the [pinned handler and metadata projection](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/client.py),
with Vector's documented per-call selection and current authorization. Receipt, CAS and durable-effect
requirements apply to mutations, not these read actions. Feature reads and writes
remain separate rows; this evidence does not close their domain requirements.

### Workspace feature acceptance (MCP-01, MCP-02)

The [feature regressions](../apps/go-api/internal/mcpactions/workspace_features_integration_test.go)
cover exact workspace and project feature projections, false versus omitted/null
updates, the native project-column mapping, and null/omitted receipt equivalence
under both credentials and protocol revisions. The pinned
[workspace handler](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/workspace.py)
and [project handler](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/tools/project.py)
accept optional nullable booleans. The `WorkspaceFeature` and `ProjectFeature`
models and their `update_features` methods were inspected in the published
[plane-sdk 0.3.1 source release](https://pypi.org/project/plane-sdk/0.3.1/);
both SDK methods omit null fields from PATCH requests. The source archive SHA-256
is `e093f0d6cc83a155cc9e1670d26da14e8f57f4dc33fed9672aa785f70662ec64`.

The regression first reproduced null rejection in both tools. Making the source
contract nullable exposed conversion of null to false in the existing flag
normalizer. The fix now skips null flags before forming the mutation and its
receipt hash. Explicit false remains a change; nonboolean strings are rejected.
Only the affected MCP design was regenerated.

Workspace proof covers guest/member reads without write permission, foreign and
revoked workspace rejection without data or audit changes, preservation of another
workspace, durable audit before/after snapshots, changed-input request conflicts,
replay after native edits, and stale revisions after native workspace or project
feature writes. A forced audit constraint failure rolls back flags, revisions and
receipts; retrying the same key succeeds and replays without another audit row.
The existing catalogue workflow also checks revoked-role replay rejection.

The [concurrent CAS regression](../apps/go-api/internal/mcpactions/workspace_feature_concurrency_integration_test.go)
holds the feature row until two writes are waiting, for both workspace and project
settings. Exactly one flag change, audit row and receipt commit; the other call
returns a revision conflict. Shared receipt-engine tests cover cancellation and lost commit acknowledgements. Features
persist audit events in the same transaction and do not dispatch external effects.

The focused identity/feature run contains 41 leaf cases across nine tests. It
closes workspace identity and feature-read projection/scope proof.

The [waiting-write regressions](../apps/go-api/internal/mcpactions/workspace_feature_authorization_integration_test.go)
now hold admission before transaction authorization for both workspace and project
settings. Forty cases cover both protocols, fresh writes and receipt replay:
personal-token and OAuth revocation, removed OAuth write scope, and demotion of
the relevant administrator. Each rejected request returns no structured data and
leaves flags, project rows, intakes, audit events and receipts unchanged. This
closes the recorded workspace feature-write authorization gap. It proves checks
after admission waits, not continuous external-issuer revocation checks through commit.

The [project permission regressions](../apps/go-api/internal/mcpactions/project_feature_permissions_integration_test.go)
verify current project administrators can write even when their workspace role
is guest or member. Workspace administration alone cannot bypass private-project
membership or elevate a project guest/member into a feature administrator.
Inactive/deleted memberships and deleted projects reject reads and writes without
mutation effects. Forcing an audit failure after enabling intakes rolls back the
native project flag, newly created default intake, revision and receipt. Same-key
retry and replay then produce one default intake and one audit event.

The [project receipt regressions](../apps/go-api/internal/mcpactions/project_feature_receipts_integration_test.go)
reuse one request ID across two projects and verify separate results. Native
project-column writes invalidate earlier revisions; a current token from another
project is rejected. Changed-input conflicts and replay preserve native edits and
do not add effects, while a matching project revision permits a fresh update.
The [workspace-binding regression](../apps/go-api/internal/mcpactions/project_feature_workspace_integration_test.go)
uses an actor who administers both workspaces and both private projects. Selecting
one workspace cannot read or write the other's project, including when reusing a
successful receipt key. Both workspaces' flags, projects, intakes, audits and
receipts remain unchanged on rejection; correct-scope replay still succeeds.

The [archive regressions](../apps/go-api/internal/mcpactions/project_feature_archive_integration_test.go)
first reproduced a discrepancy: native project retrieval retained its feature
columns after archive, but the feature reader rejected the same current member.
The feature scope now applies its archive restriction only to writes. Reads retain
the native project's archived settings while still rejecting revoked membership.
Fresh writes and receipt replay remain forbidden until restore, with no mutation
effects; restoration permits replay and a new update. This is the explicit Vector
native-lifecycle policy; the pinned handler/SDK do not specify a separate archive
policy. The combined focused feature run passes 102 leaf cases, including project
CAS and existing current-authorization, revision, receipt and rollback regressions.

### State acceptance (MCP-01, MCP-02)

The five state actions meet the documented [state contract](mcp-states.md).
[Generated-server tests](../apps/go-api/internal/mcpactions/state_integration_test.go)
cover both credentials and protocols, exact native projections, workspace errors,
receipt conflicts/replay and revisions. [Scope tests](../apps/go-api/internal/mcpactions/state_scope_integration_test.go)
cover current permissions and discovery. The [bound-read cases](../apps/go-api/internal/projectstatereads/bound_reads_integration_test.go)
verify authorization before counts, tenant binding and deterministic pagination.
[Mutation atomicity](../apps/go-api/internal/projectstatereads/mutation_atomicity_integration_test.go)
and [reference races](../apps/go-api/internal/projectstatereads/reference_concurrency_integration_test.go)
prove rollback, current credential checks after a lock wait, concurrent receipts/CAS,
and assignment/deletion serialization. [Deletion recovery](../apps/go-api/internal/backgroundtasks/state_deletion_integration_test.go)
proves durable admission, lost-acknowledgement reconciliation, retained history,
purge and replay. All fifteen reviewed test roots passed in the complete gate at
`87a791d9`.

Local browser/proxy verification on 2026-10-30 used the rebuilt application at
`http://localhost:8000/`, an existing personal token, protocol `2026-07-28`, and
a dedicated acceptance project. MCP list matched its five native initial states.
An MCP-created nondefault state appeared in settings. A browser name/description
edit produced a new MCP revision; an old-revision update returned
`revision_conflict` and left the edited record unchanged. A current-revision MCP
update appeared in the browser. MCP deletion and replay succeeded, detail rejected
the deleted state, and both native and MCP lists exactly matched their original
five-state baselines. The browser showed only those original states afterward.
External changes were verified after explicit reloads; this does not claim
automatic cross-client refresh. OAuth and legacy-protocol coverage comes from
the generated-server integration tests, not this browser session.
