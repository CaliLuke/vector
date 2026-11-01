package workitems

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/datetime"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/CaliLuke/go-plane/internal/requestinput"
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
		row, err := r.row(ctx, `SELECT `+apicontract.IssueRowSQL("i")+`||jsonb_build_object('parent_id',`+apicontract.VisibleParentSQL("i", "$4::uuid")+`,'is_subscribed',
 EXISTS(SELECT 1 FROM issue_subscribers s WHERE s.issue_id=i.id AND s.workspace_id=i.workspace_id AND s.project_id=i.project_id AND s.subscriber_id=$4 AND s.deleted_at IS NULL))
 FROM issues i JOIN workspaces w ON w.id=i.workspace_id WHERE i.id=$1 AND i.project_id=$2 AND w.slug=$3
 AND i.deleted_at IS NULL AND i.archived_at IS NOT NULL AND NOT EXISTS(SELECT 1 FROM issue_types t WHERE t.id=i.type_id AND t.is_epic)`, q.IssueID, q.ProjectID, q.Slug, q.ActorID)
		if err != nil {
			return Result{}, err
		}
		// Archive retrieval annotates only subscription. Missing list annotations
		// are omitted by IssueDetailSerializer, rather than rendered as empty/null.
		projected, projectionErr := projectFields(row, detailFields)
		if projectionErr != nil {
			return Result{}, projectionErr
		}
		v, e := r.expandAppIssue(ctx, row, q, projected)
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
	projected, projectionErr := projectFields(row, compactFields)
	if projectionErr != nil {
		return projectionErr
	}
	previous, err := orderedDump(timestampProjection(projected, q.Timezone).(map[string]any), compactFields)
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
	rows, err := r.rows(ctx, `SELECT `+apicontract.IssueRowSQL("i")+` FROM issues i JOIN workspaces w ON w.id=i.workspace_id WHERE w.slug=$1 AND i.project_id=$2 AND i.id=ANY($3::uuid[]) AND `+predicate+` ORDER BY i.created_at DESC`, q.Slug, q.ProjectID, ids)
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
	if value, err := datetime.Parse(raw, location); err == nil {
		if local := value.In(location); local.Year() >= 1 && local.Year() <= 9999 {
			return local.Format(time.RFC3339Nano), nil
		}
	}
	return "", failure(400, "error", "Please provide valid detail")
}
func (r *Repository) issueDates(ctx context.Context, q Request, body io.Reader, contentType string) (out Result, resultErr error) {
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
		value, e := requestinput.UUID(id)
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
	changed, activities, err := prepareBulkDateChanges(updates, byID)
	if err != nil {
		return Result{}, err
	}
	for _, change := range activities {
		local := q
		local.IssueID = change.issueID
		data, dumpErr := pythoncompat.Dumps(map[string]any{change.field: change.value}, nil)
		if dumpErr != nil {
			return Result{}, dumpErr
		}
		previous, dumpErr := pythoncompat.Dumps(map[string]any{change.field: requestinput.ScalarString(change.previous)}, nil)
		if dumpErr != nil {
			return Result{}, dumpErr
		}
		if activityErr := r.activity(ctx, local, "issue.activity.updated", data, previous, true); activityErr != nil {
			return Result{}, activityErr
		}
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	for id, row := range changed {
		if _, err = tx.Exec(ctx, `UPDATE issues SET start_date=$2,target_date=$3 WHERE id=$1`, id, row["start_date"], row["target_date"]); err != nil {
			return Result{}, failure(400, "error", "Please provide valid detail")
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Result{}, err
	}
	return Result{200, map[string]any{"message": "Issues updated successfully"}}, nil
}

type bulkDateChange struct {
	issueID, field  string
	value, previous any
}

// prepareBulkDateChanges validates the whole batch before activity admission.
// The returned rows are copies; a rejected batch cannot alter its source snapshot.
func prepareBulkDateChanges(updates []any, byID map[string]map[string]any) (map[string]map[string]any, []bulkDateChange, error) {
	changed := map[string]map[string]any{}
	activities := []bulkDateChange{}
	for _, raw := range updates {
		item := raw.(map[string]any) // issueDates already validates the object and ID.
		id, err := requestinput.UUID(item["id"])
		if err != nil {
			return nil, nil, failure(400, "error", "Please provide valid detail")
		}
		row := changed[id.String()]
		if row == nil {
			row = maps.Clone(byID[id.String()])
		}
		if row == nil {
			continue
		}
		for _, field := range []string{"start_date", "target_date"} {
			value, present := item[field]
			if !present {
				continue
			}
			prepared, dateErr := prepareBulkDateForPersistence(value)
			if dateErr != nil {
				return nil, nil, dateErr
			}
			activities = append(activities, bulkDateChange{id.String(), field, prepared, row[field]})
			row[field] = prepared
			changed[id.String()] = row
		}
		valid, err := validateBulkDates(row["start_date"], row["target_date"])
		if err != nil {
			return nil, nil, err
		}
		if !valid {
			return nil, nil, failure(400, "message", "Start date cannot exceed target date")
		}
	}
	return changed, activities, nil
}

func prepareBulkDateForPersistence(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	text, ok := value.(string)
	prepared, err := datetime.Date(text)
	if !ok || err != nil {
		return nil, failure(400, "error", "Please provide valid detail")
	}
	return prepared, nil
}

func validateBulkDates(start, target any) (bool, error) {
	startDate, err := prepareBulkDateForPersistence(start)
	if err != nil {
		return false, err
	}
	targetDate, err := prepareBulkDateForPersistence(target)
	if err != nil {
		return false, err
	}
	if startDate == nil || targetDate == nil {
		return true, nil
	}
	return startDate.(string) <= targetDate.(string), nil
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
