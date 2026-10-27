# MCP action parity tracker

The accepted scope is 30 tools and 207 operations. The pinned [catalogue](../apps/go-api/mcp/testdata/plane-mcp/catalogue.json) supplies the expected names and input declarations. This tracker records implementation evidence; catalogue presence alone is not implementation.

Five member reads, four project actions, two cycle reads, two module reads, two label reads, six work-item comment/link/activity reads, and all seven milestone actions are implemented with personal tokens and OAuth and tested through the generated server under both MCP revisions. OAuth tests use an isolated TLS issuer and real PostgreSQL; OAuth remains generic, production issuer selection is deferred, and complete local browser verification remains pending. Archive and restore include optional receipts and revision guards, transactional favorite cleanup, and correlated telemetry. These are 28 of 207 actions. The other 179 actions are not implemented. Read-action discovery reflects current workspace and project membership; milestone discovery also applies its guest feature and archived-project policy, plus OAuth write scopes. Browser conflict proof remains open. No full-parity claim is made.

For each action, completion requires inspected pinned handler/SDK behavior, typed generated contracts, current authorization and tenant checks, both authentication modes and protocol revisions, and real PostgreSQL proof. Mutations additionally require durable receipt/replay, request-hash conflicts, current authorization on replay, CAS on all relevant writers, and durable effects. Asset operations require real object-storage and SSRF proof. Missing domain features require an accepted domain design before transport implementation.

Member proof: [identity and project members](../apps/go-api/internal/mcpactions/member_integration_test.go), [workspace directory and roles](../apps/go-api/internal/mcpactions/member_directory_integration_test.go), [OAuth invocation and revocation](../apps/go-api/internal/mcpactions/oauth_integration_test.go), and [domain authorization](../apps/go-api/internal/projectmembershipreads/directory_integration_test.go). [Member contract notes](mcp-members.md) record limits and compatibility decisions.

Project proof: [project contracts and primary data](../apps/go-api/internal/mcpactions/project_integration_test.go), [scope changes after admission](../apps/go-api/internal/mcpactions/scope_integration_test.go), and [repository revocation regressions](../apps/go-api/internal/workspaceprojectreads/mcp_authorization_integration_test.go). [Project contract notes](mcp-projects.md) record pagination, SDK projection and workspace binding. [Archive/restore proof](../apps/go-api/internal/mcpactions/project_archive_integration_test.go), [transaction and CAS regressions](../apps/go-api/internal/workspaceprojectreads/archive_integration_test.go), and [telemetry assertions](../apps/go-api/internal/mcpactions/mutation_observability_test.go) cover the first two mutations; [write recovery](mcp-writes.md) records the extension contract and remaining limits.

Cycle proof: [generated-server contracts, scope revocation, tenant isolation, and timezone behavior](../apps/go-api/internal/mcpactions/cycle_integration_test.go). [Cycle contract notes](mcp-cycles.md) record active/archived shapes and shared HTTP filtering.

Module and label proof: [generated-server contracts, scope revocation, and relationship isolation](../apps/go-api/internal/mcpactions/resource_read_integration_test.go). [Contract notes](mcp-modules-labels.md) record their different page shapes and archive policies.
Work-item child reads: [contract and scope notes](mcp-workitem-reads.md) and [generated-server regression tests](../apps/go-api/internal/mcpactions/workitem_read_integration_test.go).

Milestone proof: [native domain and contract](mcp-milestones.md), [generated-server lifecycle and receipts](../apps/go-api/internal/mcpactions/milestone_integration_test.go), [discovery and late scope changes](../apps/go-api/internal/mcpactions/milestone_scope_integration_test.go), and [sanitized telemetry](../apps/go-api/internal/mcpactions/milestone_observability_test.go). Native HTTP and MCP share the same seven operations, revision triggers, transactional outbox and worker delivery paths.

The [PQL foundation](pql.md) adds shared scalar, relationship (including native milestones), member, body, and effective custom-property filtering to HTTP work-item lists. It does not add an MCP action: full query grammar, domain resolvers, immutable workspace binding, and generated core contracts remain pending.

| Operation                                  | Kind     | Required upstream inputs                                 | Vector implementation | Remaining proof                                     |
| ------------------------------------------ | -------- | -------------------------------------------------------- | --------------------- | --------------------------------------------------- |
| `customer.list`                            | Read     | None                                                     | Not implemented       | Domain mapping, contract and authorization          |
| `customer.retrieve`                        | Read     | `customer_id`                                            | Not implemented       | Domain mapping, contract and authorization          |
| `customer.create`                          | Mutation | `name`                                                   | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `customer.update`                          | Mutation | `customer_id`                                            | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `customer.delete`                          | Mutation | None                                                     | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `customer.list_workitems`                  | Read     | `customer_id`                                            | Not implemented       | Domain mapping, contract and authorization          |
| `customer.manage_workitems`                | Mutation | `customer_id`                                            | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `customer_property.list`                   | Read     | None                                                     | Not implemented       | Domain mapping, contract and authorization          |
| `customer_property.retrieve`               | Read     | `property_id`                                            | Not implemented       | Domain mapping, contract and authorization          |
| `customer_property.create`                 | Mutation | `display_name`, `property_type`                          | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `customer_property.update`                 | Mutation | `property_id`                                            | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `customer_property.delete`                 | Mutation | `property_id`                                            | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `customer_property.get_values`             | Read     | `customer_id`                                            | Not implemented       | Domain mapping, contract and authorization          |
| `customer_property.set_values`             | Mutation | `customer_id`, `values`                                  | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `customer_request.list`                    | Read     | `customer_id`                                            | Not implemented       | Domain mapping, contract and authorization          |
| `customer_request.retrieve`                | Read     | `customer_id`, `request_id`                              | Not implemented       | Domain mapping, contract and authorization          |
| `customer_request.create`                  | Mutation | `customer_id`, `name`                                    | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `customer_request.update`                  | Mutation | `customer_id`, `request_id`                              | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `customer_request.delete`                  | Mutation | `customer_id`, `request_id`                              | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `cycle.list`                               | Read     | `project_id`                                             | Implemented           | Local browser/proxy verification                    |
| `cycle.retrieve`                           | Read     | `project_id`, `cycle_id`                                 | Implemented           | Local browser/proxy verification                    |
| `cycle.create`                             | Mutation | `project_id`, `name`, `owned_by`                         | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `cycle.update`                             | Mutation | `project_id`, `cycle_id`                                 | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `cycle.delete`                             | Mutation | `project_id`, `cycle_id`                                 | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `cycle.list_workitems`                     | Read     | `project_id`, `cycle_id`                                 | Not implemented       | Domain mapping, contract and authorization          |
| `cycle.manage_workitems`                   | Mutation | `project_id`, `cycle_id`                                 | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `cycle.transfer_workitems`                 | Mutation | `project_id`, `cycle_id`, `new_cycle_id`                 | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `cycle.complete`                           | Mutation | `project_id`, `cycle_id`                                 | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `cycle.archive`                            | Mutation | `project_id`, `cycle_id`                                 | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `cycle.unarchive`                          | Mutation | `project_id`, `cycle_id`                                 | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `get_pql_reference`                        | Read     | None                                                     | Not implemented       | Domain mapping, contract and authorization          |
| `initiative.list`                          | Read     | None                                                     | Not implemented       | Domain mapping, contract and authorization          |
| `initiative.retrieve`                      | Read     | `initiative_id`                                          | Not implemented       | Domain mapping, contract and authorization          |
| `initiative.create`                        | Mutation | `name`                                                   | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `initiative.update`                        | Mutation | `initiative_id`                                          | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `initiative.delete`                        | Mutation | `initiative_id`                                          | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `initiative.list_projects`                 | Read     | `initiative_id`                                          | Not implemented       | Domain mapping, contract and authorization          |
| `initiative.add_projects`                  | Mutation | `initiative_id`, `project_ids`                           | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `initiative.remove_projects`               | Mutation | `initiative_id`, `project_ids`                           | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `initiative.list_workitems`                | Read     | `initiative_id`                                          | Not implemented       | Domain mapping, contract and authorization          |
| `initiative.manage_workitems`              | Mutation | `initiative_id`                                          | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `intake.list`                              | Read     | `project_id`                                             | Not implemented       | Domain mapping, contract and authorization          |
| `intake.retrieve`                          | Read     | `project_id`, `workitem_id`                              | Not implemented       | Domain mapping, contract and authorization          |
| `intake.create`                            | Mutation | `project_id`, `name`                                     | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `intake.update`                            | Mutation | `project_id`, `workitem_id`                              | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `intake.delete`                            | Mutation | `project_id`, `workitem_id`                              | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `label.list`                               | Read     | `project_id`                                             | Implemented           | Local browser/proxy verification                    |
| `label.retrieve`                           | Read     | `project_id`, `label_id`                                 | Implemented           | Local browser/proxy verification                    |
| `label.create`                             | Mutation | `project_id`, `name`                                     | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `label.update`                             | Mutation | `project_id`, `label_id`                                 | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `label.delete`                             | Mutation | `project_id`, `label_id`                                 | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `member.me`                                | Read     | None                                                     | PAT and OAuth         | Local browser/proxy verification                    |
| `member.list_workspace`                    | Read     | None                                                     | PAT and OAuth         | Local browser/proxy verification                    |
| `member.list_project`                      | Read     | `project_id`                                             | PAT and OAuth         | Local browser/proxy verification                    |
| `member.list_roles`                        | Read     | None                                                     | PAT and OAuth         | Local browser/proxy verification                    |
| `member.retrieve_role`                     | Read     | `role_id`                                                | PAT and OAuth         | Local browser/proxy verification                    |
| `milestone.list`                           | Read     | `project_id`                                             | Implemented           | Current-worktree live proof                         |
| `milestone.retrieve`                       | Read     | `project_id`, `milestone_id`                             | Implemented           | Current-worktree live proof                         |
| `milestone.create`                         | Mutation | `project_id`, `title`                                    | Implemented           | Current-worktree live proof                         |
| `milestone.update`                         | Mutation | `project_id`, `milestone_id`                             | Implemented           | Current-worktree live proof                         |
| `milestone.delete`                         | Mutation | `project_id`, `milestone_id`                             | Implemented           | Current-worktree live proof                         |
| `milestone.list_workitems`                 | Read     | `project_id`, `milestone_id`                             | Implemented           | Current-worktree live proof                         |
| `milestone.manage_workitems`               | Mutation | `project_id`, `milestone_id`                             | Implemented           | Current-worktree live proof                         |
| `module.list`                              | Read     | `project_id`                                             | Implemented           | Local browser/proxy verification                    |
| `module.retrieve`                          | Read     | `project_id`, `module_id`                                | Implemented           | Local browser/proxy verification                    |
| `module.create`                            | Mutation | `project_id`, `name`                                     | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `module.update`                            | Mutation | `project_id`, `module_id`                                | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `module.delete`                            | Mutation | `project_id`, `module_id`                                | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `module.list_workitems`                    | Read     | `project_id`, `module_id`                                | Not implemented       | Domain mapping, contract and authorization          |
| `module.manage_workitems`                  | Mutation | `project_id`, `module_id`                                | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `module.archive`                           | Mutation | `project_id`, `module_id`                                | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `module.unarchive`                         | Mutation | `project_id`, `module_id`                                | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `page.list`                                | Read     | None                                                     | Not implemented       | Domain mapping, contract and authorization          |
| `page.retrieve`                            | Read     | `page_id`                                                | Not implemented       | Domain mapping, contract and authorization          |
| `page.create`                              | Mutation | `name`, `description_html`                               | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `page.update`                              | Mutation | `page_id`                                                | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `page.archive`                             | Mutation | `page_id`                                                | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `page.delete`                              | Mutation | `page_id`                                                | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `page.set_collection`                      | Mutation | `page_id`, `collection_id`                               | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `page.list_workitem_pages`                 | Read     | `project_id`, `workitem_id`                              | Not implemented       | Domain mapping, contract and authorization          |
| `page.attach_to_workitem`                  | Mutation | `project_id`, `workitem_id`, `page_id`                   | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `page.detach_from_workitem`                | Mutation | `project_id`, `workitem_id`, `workitem_page_id`          | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `project.list`                             | Read     | None                                                     | PAT and OAuth         | Local browser/proxy verification                    |
| `project.retrieve`                         | Read     | `project_id`                                             | PAT and OAuth         | Local browser/proxy verification                    |
| `project.create`                           | Mutation | `name`, `identifier`                                     | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `project.update`                           | Mutation | `project_id`                                             | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `project.delete`                           | Mutation | `project_id`                                             | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `project.archive`                          | Mutation | `project_id`                                             | PAT and OAuth         | Browser conflict and deployed issuer verification   |
| `project.unarchive`                        | Mutation | `project_id`                                             | PAT and OAuth         | Browser conflict and deployed issuer verification   |
| `project.worklog_summary`                  | Read     | `project_id`                                             | Not implemented       | Domain mapping, contract and authorization          |
| `project.get_features`                     | Read     | `project_id`                                             | Not implemented       | Domain mapping, contract and authorization          |
| `project.update_features`                  | Mutation | `project_id`                                             | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `project_estimate.retrieve`                | Read     | `project_id`                                             | Not implemented       | Domain mapping, contract and authorization          |
| `project_estimate.create`                  | Mutation | `project_id`, `name`                                     | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `project_estimate.update`                  | Mutation | `project_id`                                             | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `project_estimate.delete`                  | Mutation | `project_id`                                             | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `project_estimate.link`                    | Mutation | `project_id`, `estimate_id`                              | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `project_estimate.list_points`             | Read     | `project_id`, `estimate_id`                              | Not implemented       | Domain mapping, contract and authorization          |
| `project_estimate.create_points`           | Mutation | `project_id`, `estimate_id`, `points`                    | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `project_estimate.update_point`            | Mutation | `project_id`, `estimate_id`, `estimate_point_id`         | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `project_estimate.delete_point`            | Mutation | `project_id`, `estimate_id`, `estimate_point_id`         | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `release.list`                             | Read     | None                                                     | Not implemented       | Domain mapping, contract and authorization          |
| `release.retrieve`                         | Read     | `release_id`                                             | Not implemented       | Domain mapping, contract and authorization          |
| `release.create`                           | Mutation | `name`                                                   | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `release.update`                           | Mutation | `release_id`                                             | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `release.delete`                           | Mutation | `release_id`                                             | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `release.get_changelog`                    | Read     | `release_id`                                             | Not implemented       | Domain mapping, contract and authorization          |
| `release.update_changelog`                 | Mutation | `release_id`                                             | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `release.list_workitems`                   | Read     | `release_id`                                             | Not implemented       | Domain mapping, contract and authorization          |
| `release.manage_workitems`                 | Mutation | `release_id`                                             | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `release_label.list`                       | Read     | None                                                     | Not implemented       | Domain mapping, contract and authorization          |
| `release_label.create`                     | Mutation | `name`                                                   | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `release_label.update`                     | Mutation | `label_id`                                               | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `release_label.delete`                     | Mutation | `label_id`                                               | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `release_label.attach`                     | Mutation | `release_id`, `label_ids`                                | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `release_label.detach`                     | Mutation | `release_id`, `label_ids`                                | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `release_tag.list`                         | Read     | None                                                     | Not implemented       | Domain mapping, contract and authorization          |
| `release_tag.retrieve`                     | Read     | `tag_id`                                                 | Not implemented       | Domain mapping, contract and authorization          |
| `release_tag.create`                       | Mutation | `version`                                                | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `release_tag.update`                       | Mutation | `tag_id`                                                 | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `release_tag.delete`                       | Mutation | `tag_id`                                                 | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `state.list`                               | Read     | None                                                     | Not implemented       | Domain mapping, contract and authorization          |
| `state.retrieve`                           | Read     | `state_id`                                               | Not implemented       | Domain mapping, contract and authorization          |
| `state.create`                             | Mutation | `name`, `color`                                          | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `state.update`                             | Mutation | `state_id`                                               | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `state.delete`                             | Mutation | `state_id`                                               | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `work_log.list`                            | Read     | `project_id`, `workitem_id`                              | Not implemented       | Domain mapping, contract and authorization          |
| `work_log.create`                          | Mutation | `project_id`, `workitem_id`, `duration`                  | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `work_log.update`                          | Mutation | `project_id`, `workitem_id`, `work_log_id`               | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `work_log.delete`                          | Mutation | `project_id`, `workitem_id`, `work_log_id`               | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem.list`                            | Read     | None                                                     | Not implemented       | Domain mapping, contract and authorization          |
| `workitem.list_archived`                   | Read     | `project_id`                                             | Not implemented       | Domain mapping, contract and authorization          |
| `workitem.retrieve`                        | Read     | `project_id`, `workitem_id`                              | Not implemented       | Domain mapping, contract and authorization          |
| `workitem.retrieve_by_identifier`          | Read     | `workitem_identifier`                                    | Not implemented       | Domain mapping, contract and authorization          |
| `workitem.search`                          | Read     | `query`                                                  | Not implemented       | Domain mapping, contract and authorization          |
| `workitem.count`                           | Read     | None                                                     | Not implemented       | Domain mapping, contract and authorization          |
| `workitem.create`                          | Mutation | `project_id`, `name`                                     | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem.update`                          | Mutation | `project_id`, `workitem_id`                              | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem.delete`                          | Mutation | `project_id`, `workitem_id`                              | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem.archive`                         | Mutation | `project_id`, `workitem_id`                              | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem.manage_assignee`                 | Mutation | `project_id`, `workitem_id`                              | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem.manage_label`                    | Mutation | `project_id`, `workitem_id`                              | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_activity.list`                   | Read     | `project_id`, `workitem_id`                              | Implemented           | Local browser/proxy verification                    |
| `workitem_activity.retrieve`               | Read     | `project_id`, `workitem_id`, `activity_id`               | Implemented           | Local browser/proxy verification                    |
| `workitem_attachment.list`                 | Read     | `project_id`, `workitem_id`                              | Not implemented       | Domain mapping, contract and authorization          |
| `workitem_attachment.read`                 | Read     | `project_id`, `workitem_id`, `attachment_id`             | Not implemented       | Domain mapping, contract and authorization          |
| `workitem_attachment.download_url`         | Read     | `project_id`, `workitem_id`, `attachment_id`             | Not implemented       | Domain mapping, contract and authorization          |
| `workitem_attachment.upload_from_url`      | Mutation | `project_id`, `workitem_id`, `url`                       | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_attachment.delete`               | Mutation | `project_id`, `workitem_id`, `attachment_id`             | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_comment.list`                    | Read     | `project_id`, `workitem_id`                              | Implemented           | Local browser/proxy verification                    |
| `workitem_comment.retrieve`                | Read     | `project_id`, `workitem_id`, `comment_id`                | Implemented           | Local browser/proxy verification                    |
| `workitem_comment.create`                  | Mutation | `project_id`, `workitem_id`, `comment_html`              | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_comment.update`                  | Mutation | `project_id`, `workitem_id`, `comment_id`                | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_comment.delete`                  | Mutation | `project_id`, `workitem_id`, `comment_id`                | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_link.list`                       | Read     | `project_id`, `workitem_id`                              | Implemented           | Local browser/proxy verification                    |
| `workitem_link.retrieve`                   | Read     | `project_id`, `workitem_id`, `link_id`                   | Implemented           | Local browser/proxy verification                    |
| `workitem_link.create`                     | Mutation | `project_id`, `workitem_id`, `url`                       | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_link.update`                     | Mutation | `project_id`, `workitem_id`, `link_id`                   | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_link.delete`                     | Mutation | `project_id`, `workitem_id`, `link_id`                   | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_property.list`                   | Read     | None                                                     | Not implemented       | Domain mapping, contract and authorization          |
| `workitem_property.retrieve`               | Read     | `workitem_property_id`                                   | Not implemented       | Domain mapping, contract and authorization          |
| `workitem_property.create`                 | Mutation | `display_name`, `property_type`                          | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_property.update`                 | Mutation | `workitem_property_id`                                   | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_property.delete`                 | Mutation | `workitem_property_id`                                   | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_property.manage_type_properties` | Mutation | `workitem_type_id`                                       | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_property.list_options`           | Read     | `property_id`                                            | Not implemented       | Domain mapping, contract and authorization          |
| `workitem_property.retrieve_option`        | Read     | `property_id`, `option_id`                               | Not implemented       | Domain mapping, contract and authorization          |
| `workitem_property.create_option`          | Mutation | `property_id`, `name`                                    | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_property.update_option`          | Mutation | `property_id`, `option_id`                               | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_property.delete_option`          | Mutation | `property_id`, `option_id`                               | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_property.get_value`              | Read     | `project_id`, `workitem_id`, `property_id`               | Not implemented       | Domain mapping, contract and authorization          |
| `workitem_property.set_value`              | Mutation | `project_id`, `workitem_id`, `property_id`               | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_property.delete_value`           | Mutation | `project_id`, `workitem_id`, `property_id`               | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_relation.list`                   | Read     | `project_id`, `workitem_id`                              | Not implemented       | Domain mapping, contract and authorization          |
| `workitem_relation.create`                 | Mutation | `project_id`, `workitem_id`, `workitem_ids`              | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_relation.delete`                 | Mutation | `project_id`, `workitem_id`, `related_workitem_id`       | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_relation.list_definitions`       | Read     | None                                                     | Not implemented       | Domain mapping, contract and authorization          |
| `workitem_relation.create_definition`      | Mutation | `name`                                                   | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_relation.update_definition`      | Mutation | `definition_id`                                          | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_relation.delete_definition`      | Mutation | `definition_id`                                          | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_type.list`                       | Read     | None                                                     | Not implemented       | Domain mapping, contract and authorization          |
| `workitem_type.retrieve`                   | Read     | `workitem_type_id`                                       | Not implemented       | Domain mapping, contract and authorization          |
| `workitem_type.resolve`                    | Mutation | `project_id`, `name`                                     | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_type.create`                     | Mutation | `name`                                                   | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_type.update`                     | Mutation | `workitem_type_id`                                       | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_type.delete`                     | Mutation | `workitem_type_id`                                       | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workitem_type.import_to_project`          | Mutation | `project_id`, `workitem_type_ids`                        | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `workspace.retrieve`                       | Read     | None                                                     | Not implemented       | Domain mapping, contract and authorization          |
| `workspace.get_features`                   | Read     | None                                                     | Not implemented       | Domain mapping, contract and authorization          |
| `workspace.update_features`                | Mutation | None                                                     | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `template.list`                            | Read     | `kind`                                                   | Not implemented       | Domain mapping, contract and authorization          |
| `template.create`                          | Mutation | `kind`, `name`, `template_data`                          | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `template.update`                          | Mutation | `kind`, `template_id`                                    | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `template.delete`                          | Mutation | `kind`, `template_id`                                    | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `collection.list`                          | Read     | None                                                     | Not implemented       | Domain mapping, contract and authorization          |
| `collection.retrieve`                      | Read     | `collection_id`                                          | Not implemented       | Domain mapping, contract and authorization          |
| `collection.create`                        | Mutation | `name`                                                   | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `collection.update`                        | Mutation | `collection_id`                                          | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `collection.delete`                        | Mutation | `collection_id`                                          | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `collection.list_pages`                    | Read     | `collection_id`                                          | Not implemented       | Domain mapping, contract and authorization          |
| `collection.search_pages`                  | Read     | `collection_id`                                          | Not implemented       | Domain mapping, contract and authorization          |
| `collection.add_pages`                     | Mutation | `collection_id`, `page_ids`                              | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `collection.remove_page`                   | Mutation | `collection_id`, `page_collection_id`                    | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `collection.list_members`                  | Read     | `collection_id`                                          | Not implemented       | Domain mapping, contract and authorization          |
| `collection.add_member`                    | Mutation | `collection_id`, `user_id`, `member_access`              | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `collection.update_member`                 | Mutation | `collection_id`, `collection_member_id`, `member_access` | Not implemented       | Domain mapping, contract, receipts, CAS and effects |
| `collection.remove_member`                 | Mutation | `collection_id`, `collection_member_id`                  | Not implemented       | Domain mapping, contract, receipts, CAS and effects |

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
