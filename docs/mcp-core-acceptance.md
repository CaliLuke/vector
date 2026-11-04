# Work items, collaboration, intake and relations acceptance

The remaining 30 actions in these five groups have contract acceptance. The
reviewed sources are the pinned `plane_mcp/tools/workitem*.py` and `intake.py`
handlers at `dc7cfd9f3c305891909e2372f0cc507291353801`, plus the work-item,
intake and relation models in `plane-sdk==0.3.1`. Existing generated-server
workflow and projection tests remain part of the evidence. MCP supports only
revision `2026-07-28`.

The checks compose domain, transport and worker evidence instead of repeating
every domain assertion for each credential. Credential-neutral cases run once;
authentication and authorization interactions use personal tokens and OAuth.

- [Read acceptance](../apps/go-api/internal/mcpactions/core_read_acceptance_integration_test.go)
  covers the six work-item queries, guest visibility, drafts and archives,
  identifier lookup, PQL, pagination and distinct counts across multiplying
  label/release joins. Existing comment/link projection tests retain nullable
  fields, mention conversion, timezone and parent ownership checks.
- [Mutation recovery](../apps/go-api/internal/mcpactions/core_mutation_recovery_integration_test.go)
  covers all 15 work-item, comment, link and intake mutations: current credentials
  after lock waits, replay, request conflicts, cancellation and rejected commits.
  [Reference replay](../apps/go-api/internal/mcpactions/core_reference_replay_integration_test.go)
  rejects retired labels, inactive assignees and mentions, deleted states and
  inactive custom definitions before returning a receipt. Mention markup and
  notification admission are checked at the generated-server boundary.
- [Intake contracts](../apps/go-api/internal/mcpactions/intake_contract_integration_test.go)
  cover queue visibility, administrator triage, decline/snooze/accept/duplicate
  transitions, invalid-input atomicity, feature changes and the distinct deletion
  rules for accepted and pending submissions.
- [Relation contracts](../apps/go-api/internal/mcpactions/relation_contract_integration_test.go)
  cover all six directions, inverse reads, symmetric custom edges, inactive
  definitions, mixed-tenant rejection, revoked endpoints and deletion by kind.
  Duplicate creation leaves revisions and effects unchanged; creation commits
  immutable activity for both endpoints. [Relation recovery](../apps/go-api/internal/mcpactions/relation_acceptance_integration_test.go)
  covers all five mutations with current authorization, receipts and atomic retry.
- [Native revisions](../apps/go-api/internal/mcpactions/core_native_revision_integration_test.go)
  cover competing reviews, native work-item/comment/link/intake edits and all
  three relation storage tables. [Rollback checks](../apps/go-api/internal/bootstrap/relation_rollback_integration_test.go)
  preserve live and tombstoned custom definitions; the
  [shared empty-storage check](../apps/go-api/internal/bootstrap/catalogue_rollback_integration_test.go)
  allows rollback and upgrade when no domain data would be lost.

Durable delivery stays at its owning worker boundary:
[activity workflows](../apps/go-api/internal/backgroundtasks/issue_activities_integration_test.go),
[relation activity snapshots and idempotency](../apps/go-api/internal/backgroundtasks/catalogue_activities_integration_test.go),
and [link-preview retries and stale-row guards](../apps/go-api/internal/backgroundtasks/link_preview_integration_test.go).
The mutation recovery checks assert that domain data, receipts and effects commit
together. This completes MCP-01/MCP-02's recorded acceptance requirements; it does
not close unrelated runtime incidents, coverage gaps or dependency limitations.
