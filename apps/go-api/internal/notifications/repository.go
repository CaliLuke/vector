// Package notifications owns scoped notification reads and pagination.
package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/jackc/pgx/v5"
)

type Database interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}
type Repository struct {
	db Database
}
type Principal struct {
	UserID   string
	Timezone string
}
type HTTPError = apicontract.HTTPError

func Failure(status int, key, message string) error {
	return &HTTPError{Status: status, Body: map[string]any{key: message}}
}
func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, errors.New("notification database is required")
	}
	return &Repository{db: db}, nil
}
func (r *Repository) Authorize(ctx context.Context, p Principal, slug string) error {
	var allowed bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE w.slug=$2 AND wm.member_id=$1 AND wm.deleted_at IS NULL AND wm.is_active AND wm.role IN(5,15,20))`, p.UserID, slug).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return Failure(403, "error", "You don't have the required permissions.")
	}
	return nil
}

var scope = `n.deleted_at IS NULL AND n.receiver_id=$1 AND w.slug=$2 AND ` + issuequery.MilestoneActivityVisibleSQL("n.data#>>'{issue_activity,field}'", "n.entity_identifier", "n.project_id", "n.workspace_id", "$1::uuid")

const joins = ` FROM notifications n JOIN workspaces w ON w.id=n.workspace_id
 LEFT JOIN users u ON u.id=n.triggered_by_id
 LEFT JOIN file_assets a ON a.id=u.avatar_asset_id
 LEFT JOIN workspaces aw ON aw.id=a.workspace_id `
const actorJSON = `CASE WHEN u.id IS NULL THEN 'null'::jsonb ELSE jsonb_build_object(
 'id',u.id,'first_name',u.first_name,'last_name',u.last_name,'avatar',u.avatar,
 'display_name',u.display_name,'is_bot',u.is_bot,
 'avatar_url',CASE
 WHEN u.avatar_asset_id IS NULL THEN NULLIF(u.avatar,'')
 WHEN a.entity_type IN('WORKSPACE_LOGO','USER_AVATAR','USER_COVER','PROJECT_COVER') THEN '/api/assets/v2/static/'||a.id::text||'/'
 WHEN a.entity_type='ISSUE_ATTACHMENT' THEN '/api/assets/v2/workspaces/'||aw.slug||'/projects/'||COALESCE(a.project_id::text,'None')||'/issues/'||COALESCE(a.issue_id::text,'None')||'/attachments/'||a.id::text||'/'
 WHEN a.entity_type IN('ISSUE_DESCRIPTION','COMMENT_DESCRIPTION','PAGE_DESCRIPTION','DRAFT_ISSUE_DESCRIPTION') THEN '/api/assets/v2/workspaces/'||aw.slug||'/projects/'||COALESCE(a.project_id::text,'None')||'/'||a.id::text||'/'
 ELSE NULL END
 ) - CASE WHEN a.entity_type IN('ISSUE_ATTACHMENT','ISSUE_DESCRIPTION','COMMENT_DESCRIPTION','PAGE_DESCRIPTION','DRAFT_ISSUE_DESCRIPTION') AND a.workspace_id IS NULL THEN ARRAY['avatar_url'] ELSE ARRAY[]::text[] END END`
const notificationJSON = `jsonb_build_object(
 'id',n.id,'created_at',n.created_at,'updated_at',n.updated_at,'created_by',n.created_by_id,'updated_by',n.updated_by_id,
 'deleted_at',n.deleted_at,'workspace',n.workspace_id,'project',n.project_id,'data',n.data,
 'entity_identifier',n.entity_identifier,'entity_name',n.entity_name,'title',n.title,'message',n.message,
 'message_html',n.message_html,'message_stripped',n.message_stripped,'sender',n.sender,
 'triggered_by',n.triggered_by_id,'receiver',n.receiver_id,'read_at',n.read_at,'snoozed_till',n.snoozed_till,'archived_at',n.archived_at,
 'triggered_by_details',` + actorJSON + `)`
const intakeExists = `EXISTS(SELECT 1 FROM issues i JOIN intake_issues ii ON ii.issue_id=i.id WHERE i.deleted_at IS NULL AND i.id=n.entity_identifier AND i.workspace_id=n.workspace_id AND ii.status IN(0,2,-2))`
const annotationJSON = `jsonb_build_object('is_inbox_issue',` + intakeExists + `,'is_intake_issue',` + intakeExists + `,'is_mentioned_notification',UPPER(n.sender) LIKE '%MENTIONED%')`

func decodeRecord(raw []byte, timezone string) (map[string]any, error) {
	result := map[string]any{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}
	for _, key := range []string{"created_at", "updated_at", "deleted_at", "read_at", "snoozed_till", "archived_at"} {
		value, ok := result[key].(string)
		if !ok {
			continue
		}
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			return nil, err
		}
		parsed = parsed.In(location)
		layout := "2006-01-02T15:04:05"
		if parsed.Nanosecond() != 0 {
			layout += ".000000"
		}
		layout += "Z07:00"
		result[key] = parsed.Format(layout)
	}
	return result, nil
}
func (r *Repository) Detail(ctx context.Context, p Principal, slug, id string, inherited bool) (map[string]any, error) {
	var raw []byte
	err := r.db.QueryRow(ctx, `SELECT `+notificationJSON+joins+` WHERE `+scope+` AND n.id=$3`, p.UserID, slug, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		if inherited {
			return nil, Failure(404, "detail", "No Notification matches the given query.")
		}
		return nil, Failure(404, "error", "The required object does not exist.")
	}
	if err != nil {
		return nil, err
	}
	return decodeRecord(raw, p.Timezone)
}
func last(q url.Values, key, fallback string) string {
	v, ok := q[key]
	if !ok || len(v) == 0 {
		return fallback
	}
	return v[len(v)-1]
}
func (r *Repository) List(ctx context.Context, p Principal, slug string, q url.Values) (any, error) {
	selection, err := listSelection(q)
	if err != nil {
		return nil, err
	}
	page, err := parsePage(q)
	if err != nil {
		return nil, err
	}
	order := "n.snoozed_till ASC NULLS LAST,n.created_at DESC"
	if page != nil {
		column := last(q, "order_by", "-created_at")
		if !slices.Contains([]string{"created_at", "-created_at", "updated_at", "-updated_at"}, column) {
			column = "-created_at"
		}
		direction := "ASC"
		if strings.HasPrefix(column, "-") {
			direction = "DESC"
			column = column[1:]
		}
		order = "n." + column + " " + direction + " NULLS LAST,n.created_at DESC"
	}
	where := selection.where()
	args := []any{p.UserID, slug}
	query := `SELECT ` + notificationJSON + ` || ` + annotationJSON + joins + ` WHERE ` + where + ` ORDER BY ` + order
	var total int64
	if page != nil {
		if scanErr := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications n JOIN workspaces w ON w.id=n.workspace_id WHERE `+where, args...).Scan(&total); scanErr != nil {
			return nil, scanErr
		}
		query += ` LIMIT $3 OFFSET $4`
		args = append(args, page.limit+1, page.offset)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []any{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		record, err := decodeRecord(raw, p.Timezone)
		if err != nil {
			return nil, err
		}
		results = append(results, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if page == nil {
		return results, nil
	}
	next := len(results) > page.limit
	if next {
		results = results[:page.limit]
	}
	envelope := apicontract.Envelope(results, int(total), page.limit, page.number)
	envelope["next_page_results"] = next
	return envelope, nil
}

type pagination struct{ limit, number, offset int }

func parsePage(q url.Values) (*pagination, error) {
	if !q.Has("per_page") && !q.Has("cursor") {
		return nil, nil
	}
	limit, page, offset, err := apicontract.PageWindow(apicontract.Query{
		Params:  map[string]string{"per_page": last(q, "per_page", ""), "cursor": last(q, "cursor", "")},
		Present: map[string]bool{"per_page": q.Has("per_page"), "cursor": q.Has("cursor")},
	})
	if err != nil {
		return nil, err
	}
	return &pagination{limit: limit, number: page, offset: offset}, nil
}

func (r *Repository) Counts(ctx context.Context, p Principal, slug string) (map[string]int64, error) {
	var total, mentions int64
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FILTER(WHERE UPPER(n.sender) NOT LIKE '%MENTIONED%'),COUNT(*) FILTER(WHERE UPPER(n.sender) LIKE '%MENTIONED%') FROM notifications n JOIN workspaces w ON w.id=n.workspace_id WHERE `+scope+` AND n.read_at IS NULL AND n.archived_at IS NULL AND n.snoozed_till IS NULL`, p.UserID, slug).Scan(&total, &mentions)
	return map[string]int64{"total_unread_notifications_count": total, "mention_unread_notifications_count": mentions}, err
}
