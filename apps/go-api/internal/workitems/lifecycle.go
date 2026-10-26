package workitems

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
)

func (r *Repository) lifecycle(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	return r.projectRequest(ctx, q, body, contentType, (*Repository).authorizeLifecycle, (*Repository).lifecycleTransaction)
}

func (r *Repository) lifecycleTransaction(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	op := q.Operation
	if strings.HasSuffix(op, "_deleted_issues") {
		return r.deletedIssues(ctx, q)
	}
	if strings.HasSuffix(op, "_issue_dates") {
		return r.issueDates(ctx, q, body, contentType)
	}
	if strings.HasSuffix(op, "_bulk_delete_issues") || strings.HasSuffix(op, "_bulk_archive_issues") {
		return r.bulkLifecycle(ctx, q, body, contentType)
	}
	if q.Method == "GET" {
		row, err := r.row(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+`||jsonb_build_object('parent_id',`+djangoapi.VisibleParentSQL("i", "$4::uuid")+`,'is_subscribed',
 EXISTS(SELECT 1 FROM issue_subscribers s WHERE s.issue_id=i.id AND s.workspace_id=i.workspace_id AND s.project_id=i.project_id AND s.subscriber_id=$4 AND s.deleted_at IS NULL))
 FROM issues i JOIN workspaces w ON w.id=i.workspace_id WHERE i.id=$1 AND i.project_id=$2 AND w.slug=$3
 AND i.deleted_at IS NULL AND i.archived_at IS NOT NULL AND NOT EXISTS(SELECT 1 FROM issue_types t WHERE t.id=i.type_id AND t.is_epic)`, q.IssueID, q.ProjectID, q.Slug, q.ActorID)
		if err != nil {
			return Result{}, err
		}
		// Archive retrieval annotates only subscription. Missing list annotations
		// are omitted by IssueDetailSerializer, rather than rendered as empty/null.
		v, e := r.expandAppIssue(ctx, row, q, projectFields(row, detailFields))
		return Result{200, v}, e
	}
	row, err := r.issue(ctx, q, q.Method == "POST")
	if err != nil {
		return Result{}, err
	}
	if q.Method == "DELETE" && row["archived_at"] == nil {
		return Result{}, requiredObject()
	}
	if q.Method == "POST" {
		ok, e := r.archivable(ctx, row)
		if e != nil {
			return Result{}, e
		}
		if !ok {
			return Result{}, failure(400, "error", "Can only archive completed or cancelled state group issue")
		}
	}
	old := maps.Clone(row)
	date := any(nil)
	if q.Method == "POST" {
		date = r.now().UTC().Format("2006-01-02")
	}
	if err = r.archiveActivity(ctx, q, row, date); err != nil {
		return Result{}, err
	}
	row["archived_at"] = date
	if _, err = r.saveIssue(ctx, q.ActorID, row, old); err != nil {
		return Result{}, err
	}
	if q.Method == "DELETE" {
		return Result{Status: 204}, nil
	}
	return Result{200, map[string]any{"archived_at": date}}, nil
}
func (r *Repository) archivable(ctx context.Context, row map[string]any) (bool, error) {
	if row["state_id"] == nil {
		return false, errors.New("issue state missing")
	}
	var allowed bool
	err := r.db.QueryRow(ctx, `SELECT "group" IN ('completed','cancelled') FROM states WHERE id=$1`, row["state_id"]).Scan(&allowed)
	return allowed, err
}
func (r *Repository) archiveActivity(ctx context.Context, q Request, row map[string]any, date any) error {
	q.IssueID = row["id"].(string)
	requested := map[string]any{"archived_at": date}
	keys := []string{"archived_at"}
	if date != nil {
		requested["automation"] = false
		keys = append(keys, "automation")
	}
	data, err := orderedDump(requested, keys)
	if err != nil {
		return err
	}
	previous, err := orderedDump(timestampProjection(projectFields(row, compactFields), q.Timezone).(map[string]any), compactFields)
	if err != nil {
		return err
	}
	return r.activity(ctx, q, "issue.activity.updated", data, previous, false)
}
func (r *Repository) bulkLifecycle(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	input, err := parseObject(body, contentType)
	if err != nil {
		return Result{}, err
	}
	raw := input.Fields["issue_ids"]
	if !truthy(raw) {
		return Result{}, failure(400, "error", "Issue IDs are required")
	}
	ids, err := rawIDs(raw)
	if err != nil {
		return Result{}, err
	}
	predicate := "i.deleted_at IS NULL"
	if q.Method == "DELETE" {
		predicate = liveIssue
	}
	rows, err := r.rows(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+` FROM issues i JOIN workspaces w ON w.id=i.workspace_id WHERE w.slug=$1 AND i.project_id=$2 AND i.id=ANY($3::uuid[]) AND `+predicate+` ORDER BY i.created_at DESC`, q.Slug, q.ProjectID, ids)
	if err != nil {
		return Result{}, err
	}
	selected := []string{}
	for _, row := range rows {
		selected = append(selected, row["id"].(string))
	}
	if q.Method == "DELETE" {
		for _, table := range []string{"cycle_issues", "module_issues"} {
			if _, err = r.db.Exec(ctx, `UPDATE `+table+` SET deleted_at=$2 WHERE issue_id=ANY($1::uuid[]) AND deleted_at IS NULL`, selected, timestamp(r.now())); err != nil {
				return Result{}, err
			}
		}
		if _, err = r.db.Exec(ctx, `UPDATE issues SET deleted_at=$2 WHERE id=ANY($1::uuid[])`, selected, timestamp(r.now())); err != nil {
			return Result{}, err
		}
		return Result{200, map[string]any{"message": fmt.Sprintf("%d issues were deleted", len(selected))}}, nil
	}
	date := r.now().UTC().Format("2006-01-02")
	for _, row := range rows {
		ok, e := r.archivable(ctx, row)
		if e != nil {
			return Result{}, e
		}
		if !ok {
			return Result{}, invalid(map[string]any{"error_code": 4091, "error_message": "INVALID_ARCHIVE_STATE_GROUP"})
		}
		if err = r.archiveActivity(ctx, q, row, date); err != nil {
			return Result{}, err
		}
	}
	_, err = r.db.Exec(ctx, `UPDATE issues SET archived_at=$2 WHERE id=ANY($1::uuid[])`, selected, date)
	return Result{200, map[string]any{"archived_at": date}}, err
}
func (r *Repository) deletedIssues(ctx context.Context, q Request) (Result, error) {
	args := []any{q.Slug, q.ProjectID}
	extra := ""
	if q.Present["updated_at__gt"] {
		value, err := parseQueryDateTime(q.Params["updated_at__gt"], q.Timezone)
		if err != nil {
			return Result{}, err
		}
		args = append(args, value)
		extra = " AND i.updated_at>$3"
	}
	rows, err := r.rows(ctx, `SELECT jsonb_build_object('id',i.id) FROM issues i JOIN workspaces w ON w.id=i.workspace_id WHERE w.slug=$1 AND i.project_id=$2 AND (i.archived_at IS NOT NULL OR i.deleted_at IS NOT NULL)`+extra+` ORDER BY i.created_at DESC`, args...)
	if err != nil {
		return Result{}, err
	}
	ids := []any{}
	for _, row := range rows {
		ids = append(ids, row["id"])
	}
	return Result{200, ids}, nil
}
func parseQueryDateTime(raw, zone string) (string, error) {
	location, _ := time.LoadLocation(zone)
	if location == nil {
		location = time.UTC
	}
	if value, err := pythoncompat.DjangoDateTime(raw, location); err == nil {
		if local := value.In(location); local.Year() >= 1 && local.Year() <= 9999 {
			return local.Format(time.RFC3339Nano), nil
		}
	}
	return "", failure(400, "error", "Please provide valid detail")
}
func (r *Repository) issueDates(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	input, err := parseObject(body, contentType)
	if err != nil {
		return Result{}, err
	}
	var updates []any
	if raw, ok := input.Fields["updates"]; ok {
		updates, ok = raw.([]any)
		if !ok {
			return Result{}, errors.New("updates must be iterable objects")
		}
	}
	ids := []string{}
	for _, raw := range updates {
		item, ok := raw.(map[string]any)
		if !ok {
			return Result{}, errors.New("invalid update object")
		}
		id, ok := item["id"]
		if !ok {
			return Result{}, failure(400, "error", "The required key does not exist.")
		}
		value, e := djangoinput.UUID(id)
		if e != nil {
			return Result{}, failure(400, "error", "Please provide valid detail")
		}
		ids = append(ids, value.String())
	}
	rows, err := r.rows(ctx, `SELECT to_jsonb(i) FROM issues i JOIN workspaces w ON w.id=i.workspace_id WHERE w.slug=$1 AND i.project_id=$2 AND i.id=ANY($3::uuid[]) AND i.deleted_at IS NULL`, q.Slug, q.ProjectID, ids)
	if err != nil {
		return Result{}, err
	}
	byID := map[string]map[string]any{}
	for _, row := range rows {
		byID[row["id"].(string)] = row
	}
	changed := map[string]bool{}
	for _, raw := range updates {
		item := raw.(map[string]any)
		id, ok := item["id"].(string)
		if !ok {
			continue
		}
		row := byID[id]
		if row == nil {
			continue
		}
		start, target := row["start_date"], row["target_date"]
		if truthy(item["start_date"]) {
			start = item["start_date"]
		}
		if truthy(item["target_date"]) {
			target = item["target_date"]
		}
		valid, e := validateBulkDates(start, target)
		if e != nil {
			return Result{}, e
		}
		if !valid {
			return Result{}, failure(400, "message", "Start date cannot exceed target date")
		}
		for _, field := range []string{"start_date", "target_date"} {
			if !truthy(item[field]) {
				continue
			}
			local := q
			local.IssueID = id
			data, _ := pythoncompat.Dumps(map[string]any{field: item[field]}, nil)
			previous, _ := pythoncompat.Dumps(map[string]any{field: djangoinput.ScalarString(row[field])}, nil)
			if err = r.activity(ctx, local, "issue.activity.updated", data, previous, true); err != nil {
				return Result{}, err
			}
			row[field] = item[field]
			changed[id] = true
		}
	}
	// Model date preparation happens only after every activity has been queued.
	for id := range changed {
		for _, field := range []string{"start_date", "target_date"} {
			value, err := prepareBulkDateForPersistence(byID[id][field])
			if err != nil {
				return Result{}, err
			}
			byID[id][field] = value
		}
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback(ctx)
	for id := range changed {
		row := byID[id]
		if _, err = tx.Exec(ctx, `UPDATE issues SET start_date=$2,target_date=$3 WHERE id=$1`, id, row["start_date"], row["target_date"]); err != nil {
			return Result{}, failure(400, "error", "Please provide valid detail")
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Result{}, err
	}
	return Result{200, map[string]any{"message": "Issues updated successfully"}}, nil
}

func prepareBulkDate(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	text, ok := value.(string)
	if !ok {
		return nil, errors.New("invalid bulk update date type")
	}
	return parseDate(text)
}

func prepareBulkDateForPersistence(value any) (any, error) {
	prepared, err := prepareBulkDate(value)
	if err != nil {
		if _, ok := value.(string); !ok && value != nil {
			// Preserve issueDates' existing internal error for non-string input.
			return nil, err
		}
		// Django's model field raises a client validation error after activity
		// tasks have been published. Preserve those effects while returning the
		// serializer's 400 response instead of projecting the error as a 500.
		return nil, failure(400, "error", "Please provide valid detail")
	}
	return prepared, nil
}

func validateBulkDates(start, target any) (bool, error) {
	parsed := make([]time.Time, 2)
	for i, value := range []any{start, target} {
		if text, ok := value.(string); ok {
			// Python strptime's %d also accepts one space before a single
			// digit. DateField rejects that spelling later, after activities.
			if dash := strings.LastIndexByte(text, '-'); dash >= 0 && len(text[dash+1:]) == 2 && text[dash+1] == ' ' {
				text = text[:dash+1] + text[dash+2:]
			}
			date, err := time.Parse("2006-1-2", text)
			if err != nil {
				return false, err
			}
			parsed[i] = date
		}
	}
	if !truthy(start) || !truthy(target) {
		return true, nil
	}
	if parsed[0].IsZero() || parsed[1].IsZero() {
		return false, errors.New("invalid comparison date")
	}
	return !parsed[0].After(parsed[1]), nil
}

func (r *Repository) authorizeLifecycle(ctx context.Context, q Request) error {
	op := q.Operation
	roles := []int{15, 20}
	if strings.HasSuffix(op, "_bulk_delete_issues") {
		roles = []int{20}
	} else if strings.HasSuffix(op, "_deleted_issues") {
		roles = []int{5, 15, 20}
	}
	if err := r.allowProject(ctx, q, roles, false); err != nil {
		return err
	}
	return nil
}
