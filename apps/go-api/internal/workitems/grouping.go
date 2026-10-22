package workitems

import (
	"context"
	"math"
	"slices"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/issuequery"
)

func (r *Repository) groupUniverse(ctx context.Context, q Request, field string, rows []map[string]any) ([]any, error) {
	switch field {
	case "":
		return []any{}, nil
	case "priority":
		return []any{"low", "medium", "high", "urgent", "none"}, nil
	case "state__group":
		return []any{"backlog", "unstarted", "started", "completed", "cancelled"}, nil
	case "created_by", "target_date", "start_date":
		values := []any{}
		column := field
		if field == "created_by" {
			column = "created_by_id"
		}
		for _, row := range rows {
			if !slices.Contains(values, row[column]) {
				values = append(values, row[column])
			}
		}
		return values, nil
	}
	table, column, extra, none := "", "id", "", false
	switch field {
	case "state_id":
		table, extra = "states", ` AND NOT t.is_triage AND t."group" <> 'triage'`
	case "labels__id":
		table, none = "labels", true
	case "assignees__id":
		table, column, extra = "project_members", "member_id", " AND t.is_active"
		if q.ProjectID == "" {
			table = "workspace_members"
		}
	case "issue_module__module_id":
		table, none = "modules", true
	case "cycle_id":
		table, none = "cycles", true
	case "project_id":
		table = "projects"
	default:
		return []any{}, nil
	}
	args := []any{q.Slug}
	if q.ProjectID != "" && field != "project_id" {
		args = append(args, q.ProjectID)
		extra += " AND t.project_id=$2"
	}
	values, err := r.rows(ctx, `SELECT jsonb_build_object('value',t.`+column+`) FROM `+table+` t JOIN workspaces w ON w.id=t.workspace_id WHERE w.slug=$1 AND t.deleted_at IS NULL`+extra+` ORDER BY t.created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, value := range values {
		out = append(out, value["value"])
	}
	if none {
		out = append(out, "None")
	}
	return out, nil
}

func (r *Repository) groupedPage(ctx context.Context, q Request, results []any, rows []map[string]any, limit, page int) (Result, error) {
	return r.groupedPageWithVisibleRowCounts(ctx, q, results, rows, limit, page, false)
}

// groupedPageWithVisibleRowCounts keeps the established grouping, ordering and
// pagination behavior while allowing a source contract to count every row in
// its already-filtered queryset.
func (r *Repository) groupedPageWithVisibleRowCounts(ctx context.Context, q Request, results []any, rows []map[string]any, limit, page int, countEveryVisibleRow bool) (Result, error) {
	group, subgroup := q.Params["group_by"], q.Params["sub_group_by"]
	var groups, subgroups []any
	if len(rows) > 0 {
		var err error
		groups, err = r.groupUniverse(ctx, q, group, rows)
		if err != nil {
			return Result{}, err
		}
		subgroups, err = r.groupUniverse(ctx, q, subgroup, rows)
		if err != nil {
			return Result{}, err
		}
	}
	values := []map[string]any{}
	ids := []string{}
	for _, row := range rows {
		ids = append(ids, row["id"].(string))
	}
	counts := map[string]bool{}
	if countEveryVisibleRow {
		for _, row := range rows {
			counts[row["id"].(string)] = true
		}
	} else {
		countRows, err := r.rows(ctx, `SELECT jsonb_build_object('id',i.id,'counts',i.archived_at IS NULL AND NOT i.is_draft AND (NOT EXISTS(SELECT 1 FROM intake_issues t WHERE t.issue_id=i.id) OR EXISTS(SELECT 1 FROM intake_issues t WHERE t.issue_id=i.id AND t.status IN (-1,1,2)))) FROM issues i WHERE i.id=ANY($1::uuid[])`, ids)
		if err != nil {
			return Result{}, err
		}
		for _, row := range countRows {
			counts[row["id"].(string)], _ = row["counts"].(bool)
		}
	}
	groupCounts, subCounts := map[string]int{}, map[string]int{}
	total := 0
	for _, value := range results {
		row := value.(map[string]any)
		row["_count_for_pages"] = counts[row["id"].(string)]
		values = append(values, row)
		gvalues := issuequery.GroupValues(row, group)
		svalues := []string{""}
		if subgroup != "" {
			svalues = issuequery.GroupValues(row, subgroup)
		}
		total += len(gvalues) * len(svalues)
		increment := 0
		if counts[row["id"].(string)] {
			increment = 1
		}
		for _, g := range gvalues {
			// Preserve a present group with a zero filtered count. Django emits
			// total_results=1 for it but still uses zero for page counts.
			groupCounts[g] += increment
			for _, s := range svalues {
				subCounts[g+":::"+s] += increment
			}
		}
	}
	grouped, _, maximum, next := issuequery.Group(values, group, subgroup, groups, subgroups, groupCounts, subCounts, limit, page)
	count := 0
	for _, g := range grouped.(map[string]any) {
		record := g.(map[string]any)
		if subgroup == "" {
			count += len(record["results"].([]map[string]any))
		} else {
			for _, s := range record["results"].(map[string]any) {
				count += len(s.(map[string]any)["results"].([]map[string]any))
			}
		}
	}
	envelope := djangoapi.Envelope([]any{}, total, limit, page)
	envelope["results"] = grouped
	envelope["grouped_by"] = group
	if subgroup != "" {
		envelope["sub_grouped_by"] = subgroup
	}
	envelope["count"] = count
	envelope["total_pages"] = int(math.Ceil(float64(maximum) / float64(limit)))
	envelope["next_page_results"] = next
	return Result{200, envelope}, nil
}
