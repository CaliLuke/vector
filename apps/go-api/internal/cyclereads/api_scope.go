package cyclereads

func apiCycleStatus(request APIV1Request) string {
	if request.Present["status"] || request.Params["status"] != "" {
		return request.Params["status"]
	}
	if request.Operation == "list" {
		return request.Params["cycle_view"]
	}
	return ""
}

// Preserve legacy HTTP metrics while binding scoped relationships to the cycle.
func apiCycleMetricJoins(scoped bool) string {
	memberScope, issueScope, stateScope, estimateScope, bridgeScope := "", "", "", "", ""
	if scoped {
		memberScope = " AND pm.deleted_at IS NULL AND pm.workspace_id=c.workspace_id"
		issueScope = " AND i.project_id=c.project_id AND i.workspace_id=c.workspace_id"
		stateScope = " AND s.project_id=c.project_id AND s.workspace_id=c.workspace_id"
		estimateScope = " AND ep.project_id=c.project_id AND ep.workspace_id=c.workspace_id"
		bridgeScope = " AND ci.project_id=c.project_id AND ci.workspace_id=c.workspace_id"
	}
	return ` JOIN LATERAL (SELECT count(*) AS n FROM project_members pm WHERE pm.project_id=c.project_id AND pm.member_id=$3::uuid AND pm.is_active=true` + memberScope + `) membership ON membership.n>0
LEFT JOIN LATERAL (SELECT jsonb_build_object(
 'total_issues',count(ci.id) FILTER (WHERE i.archived_at IS NULL AND NOT i.is_draft AND ci.deleted_at IS NULL)*membership.n,
 'completed_issues',count(ci.id) FILTER (WHERE s."group"='completed' AND i.archived_at IS NULL AND NOT i.is_draft AND ci.deleted_at IS NULL)*membership.n,
 'cancelled_issues',count(ci.id) FILTER (WHERE s."group"='cancelled' AND i.archived_at IS NULL AND NOT i.is_draft AND ci.deleted_at IS NULL)*membership.n,
 'started_issues',count(ci.id) FILTER (WHERE s."group"='started' AND i.archived_at IS NULL AND NOT i.is_draft AND ci.deleted_at IS NULL)*membership.n,
 'unstarted_issues',count(ci.id) FILTER (WHERE s."group"='unstarted' AND i.archived_at IS NULL AND NOT i.is_draft AND ci.deleted_at IS NULL)*membership.n,
 'backlog_issues',count(ci.id) FILTER (WHERE s."group"='backlog' AND i.archived_at IS NULL AND NOT i.is_draft AND ci.deleted_at IS NULL)*membership.n) AS counts,
 sum(ep.key)*membership.n AS total_estimates,
 sum(ep.key) FILTER (WHERE s."group"='completed' AND i.archived_at IS NULL AND NOT i.is_draft AND ci.deleted_at IS NULL)*membership.n AS completed_estimates,
 sum(ep.key) FILTER (WHERE s."group"='started' AND i.archived_at IS NULL AND NOT i.is_draft AND ci.deleted_at IS NULL)*membership.n AS started_estimates
 FROM cycle_issues ci JOIN issues i ON i.id=ci.issue_id` + issueScope + ` LEFT JOIN states s ON s.id=i.state_id` + stateScope + ` LEFT JOIN estimate_points ep ON ep.id=i.estimate_point_id` + estimateScope + ` WHERE ci.cycle_id=c.id` + bridgeScope + `) stats ON true`
}
