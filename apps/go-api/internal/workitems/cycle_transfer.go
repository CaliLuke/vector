package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
)

func (r *Repository) transferCycle(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	input, err := parseObject(body, contentType)
	if err != nil {
		return Result{}, err
	}
	raw := input.Fields["new_cycle_id"]
	if !truthy(raw) {
		return Result{}, failure(400, "error", "New Cycle Id is required")
	}
	id, err := djangoinput.UUID(raw)
	if err != nil {
		return Result{}, failure(400, "error", "Please provide valid detail")
	}
	load := func(id string) (map[string]any, error) {
		return r.row(ctx, `SELECT to_jsonb(c) FROM cycles c JOIN workspaces w ON w.id=c.workspace_id WHERE c.id=$1 AND c.project_id=$2 AND w.slug=$3 AND c.deleted_at IS NULL`, id, q.ProjectID, q.Slug)
	}
	var old map[string]any
	if q.API {
		old, err = load(q.Params["cycle_id"])
		if err != nil {
			return Result{}, err
		}
		if afterNow(old["end_date"], r.now()) {
			return Result{}, failure(400, "error", "The old cycle is not completed yet")
		}
	}
	target, err := load(id.String())
	if err != nil {
		return Result{}, errors.New("destination cycle missing before end_date lookup")
	}
	if beforeNow(target["end_date"], r.now()) {
		return Result{}, failure(400, "error", "The cycle where the issues are transferred is already completed")
	}
	if old == nil {
		old, err = load(q.Params["cycle_id"])
		if err != nil {
			return Result{}, failure(400, "error", "Source cycle not found")
		}
	}
	snapshot, err := r.transferSnapshot(ctx, q, old)
	if err != nil {
		return Result{}, err
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return Result{}, err
	}
	if _, err = r.db.Exec(ctx, `UPDATE cycles SET progress_snapshot=$2 WHERE id=$1`, old["id"], encoded); err != nil {
		return Result{}, err
	}
	rows, err := r.rows(ctx, `SELECT to_jsonb(ci) FROM cycle_issues ci JOIN issues i ON i.id=ci.issue_id JOIN states s ON s.id=i.state_id JOIN workspaces w ON w.id=ci.workspace_id WHERE ci.cycle_id=$1 AND ci.project_id=$2 AND w.slug=$3 AND ci.deleted_at IS NULL AND i.archived_at IS NULL AND NOT i.is_draft AND s."group" IN ('backlog','unstarted','started') ORDER BY ci.created_at DESC`, old["id"], q.ProjectID, q.Slug)
	if err != nil {
		return Result{}, err
	}
	updated := []any{}
	ids := []string{}
	for _, row := range rows {
		ids = append(ids, row["id"].(string))
		updated = append(updated, map[string]any{"old_cycle_id": old["id"], "new_cycle_id": id.String(), "issue_id": row["issue_id"]})
	}
	if _, err = r.db.Exec(ctx, `UPDATE cycle_issues SET cycle_id=$1 WHERE id=ANY($2::uuid[])`, id.String(), ids); err != nil {
		return Result{}, err
	}
	current, err := planningCurrent("cycle", updated, []any{})
	if err != nil {
		return Result{}, err
	}
	requested, _ := pythoncompat.Dumps(map[string]any{"cycles_list": []any{}}, nil)
	err = r.publishTask(ctx, "plane.bgtasks.issue_activities_task.issue_activity", []any{}, map[string]any{"type": "cycle.activity.created", "requested_data": requested, "actor_id": q.ActorID, "issue_id": nil, "project_id": q.ProjectID, "current_instance": current, "epoch": r.now().Unix(), "notification": true, "origin": r.origin})
	return Result{200, map[string]any{"message": "Success"}}, err
}
func afterNow(raw any, now time.Time) bool {
	s, ok := raw.(string)
	if !ok {
		return false
	}
	value, e := time.Parse(time.RFC3339Nano, s)
	return e == nil && value.After(now)
}
func (r *Repository) transferSnapshot(ctx context.Context, q Request, cycle map[string]any) (map[string]any, error) {
	snapshot, err := r.row(ctx, `SELECT jsonb_build_object('total_issues',COUNT(*),'completed_issues',COUNT(*) FILTER(WHERE s."group"='completed'),'cancelled_issues',COUNT(*) FILTER(WHERE s."group"='cancelled'),'started_issues',COUNT(*) FILTER(WHERE s."group"='started'),'unstarted_issues',COUNT(*) FILTER(WHERE s."group"='unstarted'),'backlog_issues',COUNT(*) FILTER(WHERE s."group"='backlog')) FROM cycle_issues ci JOIN issues i ON i.id=ci.issue_id LEFT JOIN states s ON s.id=i.state_id WHERE ci.cycle_id=$1 AND ci.deleted_at IS NULL AND i.deleted_at IS NULL AND i.archived_at IS NULL AND NOT i.is_draft`, cycle["id"])
	if err != nil {
		return nil, err
	}
	var estimates bool
	if err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects p JOIN workspaces w ON w.id=p.workspace_id JOIN estimates e ON e.id=p.estimate_id WHERE p.id=$1 AND w.slug=$2 AND p.deleted_at IS NULL AND e.type='points')`, q.ProjectID, q.Slug).Scan(&estimates); err != nil {
		return nil, err
	}
	base := ` FROM cycle_issues ci JOIN issues i ON i.id=ci.issue_id JOIN workspaces w ON w.id=i.workspace_id LEFT JOIN estimate_points ep ON ep.id=i.estimate_point_id `
	where := ` WHERE ci.cycle_id=$1 AND ci.deleted_at IS NULL AND i.project_id=$2 AND w.slug=$3 AND ` + liveIssue
	distribution := func(points bool) (map[string]any, error) {
		suffix, aggregate := "issues", "COUNT(i.id)"
		if points {
			suffix, aggregate = "estimates", "SUM(ep.value::float8)"
		}
		counts := `'total_` + suffix + `',` + aggregate + `,'completed_` + suffix + `',` + aggregate + ` FILTER(WHERE i.completed_at IS NOT NULL),'pending_` + suffix + `',` + aggregate + ` FILTER(WHERE i.completed_at IS NULL)`
		assignees, e := r.rows(ctx, `SELECT jsonb_build_object('display_name',u.display_name,'assignee_id',u.id,'avatar_url',CASE WHEN u.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||u.avatar_asset_id||'/' ELSE u.avatar END,`+counts+`)`+base+` LEFT JOIN issue_assignees ia ON ia.issue_id=i.id LEFT JOIN users u ON u.id=ia.assignee_id `+where+` GROUP BY u.id,u.display_name,u.avatar_asset_id,u.avatar ORDER BY u.display_name`, cycle["id"], q.ProjectID, q.Slug)
		if e != nil {
			return nil, e
		}
		labels, e := r.rows(ctx, `SELECT jsonb_build_object('label_name',l.name,'label_id',l.id,'color',l.color,`+counts+`)`+base+` LEFT JOIN issue_labels il ON il.issue_id=i.id LEFT JOIN labels l ON l.id=il.label_id `+where+` GROUP BY l.id,l.name,l.color ORDER BY l.name`, cycle["id"], q.ProjectID, q.Slug)
		if e != nil {
			return nil, e
		}
		chart, e := r.transferChart(ctx, q, cycle, snapshot["total_issues"], points)
		return map[string]any{"labels": labels, "assignees": assignees, "completion_chart": chart}, e
	}
	snapshot["distribution"], err = distribution(false)
	if err != nil {
		return nil, err
	}
	snapshot["estimate_distribution"] = map[string]any{}
	if estimates {
		snapshot["estimate_distribution"], err = distribution(true)
	}
	return snapshot, err
}
func (r *Repository) transferChart(ctx context.Context, q Request, cycle map[string]any, total any, points bool) (map[string]any, error) {
	chart := map[string]any{}
	start, ok := cycle["start_date"].(string)
	if !ok {
		return chart, nil
	}
	end, ok := cycle["end_date"].(string)
	if !ok {
		return chart, nil
	}
	startTime, e := time.Parse(time.RFC3339Nano, start)
	if e != nil {
		return nil, e
	}
	endTime, e := time.Parse(time.RFC3339Nano, end)
	if e != nil {
		return nil, e
	}
	zone := q.Timezone
	if zone == "" {
		zone = "UTC"
	}
	aggregate := "COUNT(i.id)"
	extra := ""
	if points {
		aggregate = "SUM(ep.value::float8)"
		extra = " AND i.estimate_point_id IS NOT NULL"
	}
	rows, err := r.rows(ctx, `SELECT jsonb_build_object('date',(i.completed_at AT TIME ZONE $4)::date,'total',`+aggregate+`) FROM issues i JOIN cycle_issues ci ON ci.issue_id=i.id LEFT JOIN estimate_points ep ON ep.id=i.estimate_point_id JOIN workspaces w ON w.id=i.workspace_id WHERE ci.cycle_id=$1 AND ci.deleted_at IS NULL AND i.project_id=$2 AND w.slug=$3 AND `+liveIssue+extra+` GROUP BY (i.completed_at AT TIME ZONE $4)::date ORDER BY (i.completed_at AT TIME ZONE $4)::date`, cycle["id"], q.ProjectID, q.Slug, zone)
	if err != nil {
		return nil, err
	}
	number := func(v any) float64 {
		switch n := v.(type) {
		case json.Number:
			f, _ := n.Float64()
			return f
		case int:
			return float64(n)
		}
		return 0
	}
	totalValue := number(total)
	if points {
		totalValue = 0
		for _, row := range rows {
			totalValue += number(row["total"])
		}
	}
	today := r.now().UTC().Format("2006-01-02")
	for day := startTime; !day.After(endTime); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		if date > today {
			chart[date] = nil
			continue
		}
		remaining := totalValue
		for _, row := range rows {
			if d, ok := row["date"].(string); ok && strings.Compare(d, date) <= 0 {
				remaining -= number(row["total"])
			}
		}
		if points {
			chart[date] = remaining
		} else {
			chart[date] = int64(remaining)
		}
	}
	return chart, nil
}
