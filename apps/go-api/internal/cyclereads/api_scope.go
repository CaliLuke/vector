package cyclereads

// The scope and data share one primary-database snapshot. Selecting the final
// aggregate FROM scope distinguishes denied access from an allowed empty page.
const apiCycleScopeSQL = `scope AS MATERIALIZED (
 SELECT p.id,p.workspace_id FROM projects p JOIN workspaces w ON w.id=p.workspace_id
 WHERE p.id=$1::uuid AND w.slug=$2 AND w.id=$7::uuid AND p.deleted_at IS NULL AND w.deleted_at IS NULL
 AND EXISTS(SELECT 1 FROM users u WHERE u.id=$3::uuid AND u.is_active)
 AND EXISTS(SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=w.id AND wm.member_id=$3::uuid
  AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20))
 AND EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.workspace_id=w.id
  AND pm.member_id=$3::uuid AND pm.is_active AND pm.deleted_at IS NULL)
), `

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
