package modulereads

// Keep legacy HTTP metrics while binding MCP relationships to the module.
func apiModuleMetricJoins(scoped bool) string {
	issueScope, stateScope, bridgeScope := "", "", ""
	if scoped {
		issueScope = " AND i.project_id=m.project_id AND i.workspace_id=m.workspace_id"
		stateScope = " AND s.project_id=m.project_id AND s.workspace_id=m.workspace_id"
		bridgeScope = " AND mi.project_id=m.project_id AND mi.workspace_id=m.workspace_id"
	}
	return ` LEFT JOIN LATERAL (SELECT jsonb_build_object(
   'total_issues',count(DISTINCT mi.id) FILTER(WHERE i.archived_at IS NULL AND NOT i.is_draft AND mi.deleted_at IS NULL),
   'completed_issues',count(DISTINCT s."group") FILTER(WHERE s."group"='completed' AND i.archived_at IS NULL AND NOT i.is_draft AND mi.deleted_at IS NULL),
   'cancelled_issues',count(DISTINCT s."group") FILTER(WHERE s."group"='cancelled' AND i.archived_at IS NULL AND NOT i.is_draft AND mi.deleted_at IS NULL),
   'started_issues',count(DISTINCT s."group") FILTER(WHERE s."group"='started' AND i.archived_at IS NULL AND NOT i.is_draft AND mi.deleted_at IS NULL),
   'unstarted_issues',count(DISTINCT s."group") FILTER(WHERE s."group"='unstarted' AND i.archived_at IS NULL AND NOT i.is_draft AND mi.deleted_at IS NULL),
   'backlog_issues',count(DISTINCT s."group") FILTER(WHERE s."group"='backlog' AND i.archived_at IS NULL AND NOT i.is_draft AND mi.deleted_at IS NULL)) AS counts
   FROM module_issues mi JOIN issues i ON i.id=mi.issue_id` + issueScope + ` LEFT JOIN states s ON s.id=i.state_id` + stateScope + ` WHERE mi.module_id=m.id` + bridgeScope + `) stats ON true`
}
