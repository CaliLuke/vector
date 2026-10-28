package workitems

import (
	"context"
	"strings"

	"github.com/CaliLuke/go-plane/internal/issuequery"
)

// countIssueResults aggregates the same authorized, PQL-filtered item set as
// native API lists. Distinct IDs prevent multi-valued dimensions multiplying
// either parent-group counts or the overall total.
func (r *Repository) countIssueResults(ctx context.Context, q Request, from string, args []any) (Result, error) {
	group, sub := q.Params["group_by"], q.Params["sub_group_by"]
	if sub != "" && (group == "" || group == sub) {
		return Result{}, failure(400, "error", "Choose distinct group and sub-group fields.")
	}
	g, ok := countDimension(group)
	if !ok {
		return Result{}, failure(400, "error", "Invalid group_by field.")
	}
	s, ok := countDimension(sub)
	if !ok {
		return Result{}, failure(400, "error", "Invalid sub_group_by field.")
	}
	sql := `WITH base AS (SELECT issue.*` + from + `), expanded AS (SELECT issue.id,COALESCE(g.value,'None') AS g,COALESCE(s.value,'None') AS s FROM base issue LEFT JOIN LATERAL (` + g + `) g ON true LEFT JOIN LATERAL (` + s + `) s ON true) SELECT 'total' AS kind,NULL::text AS g,NULL::text AS s,COUNT(*) AS count FROM base`
	if group != "" {
		sql += ` UNION ALL SELECT 'group',g,NULL::text,COUNT(DISTINCT id) FROM expanded GROUP BY g`
	}
	if sub != "" {
		sql += ` UNION ALL SELECT 'sub',g,s,COUNT(DISTINCT id) FROM expanded GROUP BY g,s`
	}
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return Result{}, err
	}
	defer rows.Close()
	total := int64(0)
	groups := map[string]any{}
	for rows.Next() {
		var kind string
		var key, subkey *string
		var count int64
		if err = rows.Scan(&kind, &key, &subkey, &count); err != nil {
			return Result{}, err
		}
		if kind == "total" {
			total = count
			continue
		}
		entry, exists := groups[*key].(map[string]any)
		if !exists {
			entry = map[string]any{"count": int64(0), "sub_grouped_counts": nil}
			if sub != "" {
				entry["sub_grouped_counts"] = map[string]any{}
			}
			groups[*key] = entry
		}
		if kind == "group" {
			entry["count"] = count
		} else {
			entry["sub_grouped_counts"].(map[string]any)[*subkey] = map[string]any{"count": count}
		}
	}
	if err = rows.Err(); err != nil {
		return Result{}, err
	}
	var grouped, subgrouped any
	if group != "" {
		grouped = group
	}
	if sub != "" {
		subgrouped = sub
	}
	return Result{Status: 200, Body: map[string]any{"grouped_by": grouped, "sub_grouped_by": subgrouped, "total_count": total, "grouped_counts": groups}}, nil
}
func countDimension(key string) (string, bool) {
	switch key {
	case "":
		return `SELECT NULL::text AS value`, true
	case "priority", "project_id", "created_by", "target_date", "start_date":
		column := key
		if key == "created_by" {
			column = "created_by_id"
		}
		return `SELECT issue.` + column + `::text AS value`, true
	case "state_id", "state__group":
		column := "id"
		if key == "state__group" {
			column = `"group"`
		}
		return `SELECT s.` + column + `::text AS value FROM states s WHERE s.id=issue.state_id AND s.project_id=issue.project_id AND s.workspace_id=issue.workspace_id AND s.deleted_at IS NULL`, true
	case "type_id":
		return `SELECT t.id::text AS value FROM issue_types t WHERE t.id=issue.type_id AND t.workspace_id=issue.workspace_id AND t.deleted_at IS NULL`, true
	case "labels__id":
		return `SELECT DISTINCT rel.label_id::text AS value FROM issue_labels rel WHERE rel.project_id=issue.project_id AND rel.workspace_id=issue.workspace_id AND ` + issuequery.LiveIssueLabelPredicate("issue", "rel"), true
	case "assignees__id":
		return `SELECT DISTINCT rel.assignee_id::text AS value FROM issue_assignees rel JOIN users u ON u.id=rel.assignee_id AND u.is_active JOIN project_members pm ON pm.member_id=u.id AND pm.project_id=issue.project_id AND pm.workspace_id=issue.workspace_id AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20) JOIN workspace_members wm ON wm.workspace_id=issue.workspace_id AND wm.member_id=u.id AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20) WHERE rel.issue_id=issue.id AND rel.project_id=issue.project_id AND rel.workspace_id=issue.workspace_id AND rel.deleted_at IS NULL`, true
	case "release_work_items__release_id":
		return `SELECT DISTINCT rel.release_id::text AS value FROM release_work_items rel JOIN work_item_releases target ON target.id=rel.release_id AND target.workspace_id=issue.workspace_id AND target.project_id IS NULL AND target.deleted_at IS NULL WHERE rel.issue_id=issue.id AND rel.project_id=issue.project_id AND rel.workspace_id=issue.workspace_id`, true
	case "cycle_id", "issue_module__module_id", "milestone_id":
		kind := strings.TrimSuffix(key, "_id")
		if key == "issue_module__module_id" {
			kind = "module"
		}
		extra := ""
		if kind == "milestone" {
			extra = " AND " + issuequery.MilestoneReadScopeSQL("issue.project_id", "issue.workspace_id", "$1::uuid")
		}
		return `SELECT DISTINCT rel.` + kind + `_id::text AS value FROM ` + kind + `_issues rel JOIN ` + kind + `s target ON target.id=rel.` + kind + `_id AND target.project_id=issue.project_id AND target.workspace_id=issue.workspace_id AND target.deleted_at IS NULL WHERE rel.issue_id=issue.id AND rel.project_id=issue.project_id AND rel.workspace_id=issue.workspace_id AND rel.deleted_at IS NULL` + extra, true
	}
	return "", false
}
