// Package notifications preserves the notification request contract owned by Django.
package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
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
type HTTPError struct {
	Status int
	Body   map[string]any
}

func (e *HTTPError) Error() string { return fmt.Sprintf("notification HTTP %d", e.Status) }
func Failure(status int, key, message string) error {
	return &HTTPError{status, map[string]any{key: message}}
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
func stateFilter(snoozed, archived bool) string {
	result := "n.snoozed_till IS NOT NULL"
	if !snoozed {
		result = "(n.snoozed_till IS NULL OR n.snoozed_till>=CURRENT_TIMESTAMP)"
	}
	if archived {
		return result + " AND n.archived_at IS NOT NULL"
	}
	return result + " AND n.archived_at IS NULL"
}

const assigned = `n.entity_identifier IN(SELECT ia.issue_id FROM issue_assignees ia WHERE ia.deleted_at IS NULL AND ia.workspace_id=n.workspace_id AND ia.assignee_id=$1)`
const subscribed = `n.entity_identifier IN(SELECT s.issue_id FROM issue_subscribers s WHERE s.deleted_at IS NULL AND s.workspace_id=n.workspace_id AND s.subscriber_id=$1 AND NOT EXISTS(SELECT 1 FROM issues i WHERE i.deleted_at IS NULL AND i.id=s.issue_id AND i.created_by_id=$1) AND NOT EXISTS(SELECT 1 FROM issue_assignees ia WHERE ia.deleted_at IS NULL AND ia.id=s.issue_id AND ia.assignee_id=$1))`
const created = `n.entity_identifier IN(SELECT i.id FROM issues i WHERE i.deleted_at IS NULL AND i.workspace_id=n.workspace_id AND i.created_by_id=$1)`

func (r *Repository) isGuest(ctx context.Context, p Principal, slug string) (bool, error) {
	var yes bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE w.slug=$2 AND wm.member_id=$1 AND wm.deleted_at IS NULL AND wm.is_active AND wm.role<15)`, p.UserID, slug).Scan(&yes)
	return yes, err
}
func (r *Repository) List(ctx context.Context, p Principal, slug string, q url.Values) (any, error) {
	snoozed, archived := last(q, "snoozed", "false"), last(q, "archived", "false")
	if (snoozed != "true" && snoozed != "false") || (archived != "true" && archived != "false") {
		return nil, Failure(400, "error", "The required key does not exist.")
	}
	predicates := []string{scope, "n.entity_name='issue'", stateFilter(snoozed == "true", archived == "true")}
	switch last(q, "read", "") {
	case "true":
		predicates = append(predicates, "n.read_at IS NOT NULL")
	case "false":
		predicates = append(predicates, "n.read_at IS NULL")
	}
	mentioned := "UPPER(n.sender) LIKE '%MENTIONED%'"
	if last(q, "mentioned", "") == "" {
		mentioned = "NOT (" + mentioned + ")"
	}
	predicates = append(predicates, mentioned)
	types := strings.Split(last(q, "type", "all"), ",")
	typePredicates := []string{}
	guestCreated := false
	if slices.Contains(types, "subscribed") {
		typePredicates = append(typePredicates, subscribed)
	}
	if slices.Contains(types, "assigned") {
		typePredicates = append(typePredicates, assigned)
	}
	if slices.Contains(types, "created") {
		guest, err := r.isGuest(ctx, p, slug)
		if err != nil {
			return nil, err
		}
		if guest {
			guestCreated = true
			predicates = append(predicates, "FALSE")
		} else {
			typePredicates = append(typePredicates, created)
		}
	}
	if len(typePredicates) > 0 {
		predicates = append(predicates, "("+strings.Join(typePredicates, " OR ")+")")
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
	where := strings.Join(predicates, " AND ")
	args := []any{p.UserID, slug}
	query := `SELECT ` + notificationJSON + ` || ` + annotationJSON + joins + ` WHERE ` + where + ` ORDER BY ` + order
	var total int64
	if page != nil {
		if scanErr := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications n JOIN workspaces w ON w.id=n.workspace_id WHERE `+where, args...).Scan(&total); scanErr != nil {
			return nil, scanErr
		}
		if page.offset.IsInt64() {
			query += ` LIMIT $3 OFFSET $4`
			args = append(args, page.limit+1, page.offset.Int64())
		} else if guestCreated {
			// Django's .none() skips SQL, including the overflowing OFFSET.
			query += ` LIMIT 0`
		} else {
			return nil, Failure(500, "error", "Something went wrong please try again later")
		}
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
	next := int64(len(results)) > page.limit
	if next {
		results = results[:page.limit]
	}
	return map[string]any{"count": len(results), "next_cursor": fmt.Sprintf("%d:%s:0", page.limit, new(big.Int).Add(page.number, big.NewInt(1))), "prev_cursor": fmt.Sprintf("%d:%s:1", page.limit, new(big.Int).Sub(page.number, big.NewInt(1))), "next_page_results": next, "prev_page_results": page.number.Sign() > 0, "total_count": total, "total_results": total, "total_pages": (total + page.limit - 1) / page.limit, "extra_stats": nil, "grouped_by": nil, "sub_grouped_by": nil, "results": results}, nil
}

type pagination struct {
	limit          int64
	number, offset *big.Int
}

func parsePage(q url.Values) (*pagination, error) {
	limitText, cursor := last(q, "per_page", ""), last(q, "cursor", "")
	if limitText == "" || cursor == "" {
		return nil, nil
	}
	limit, err := pythoncompat.DecimalInteger(limitText)
	if err != nil {
		return nil, Failure(400, "detail", "Invalid per_page parameter.")
	}
	if limit.Cmp(big.NewInt(1000)) > 0 {
		return nil, Failure(400, "detail", "Invalid per_page value. Cannot exceed 1000.")
	}
	bits := strings.Split(cursor, ":")
	bad := func() (*pagination, error) { return nil, Failure(400, "detail", "Invalid cursor parameter.") }
	if len(bits) != 3 {
		return bad()
	}
	equal := false
	if strings.Contains(bits[0], ".") {
		value, e := pythoncompat.DecimalFloat64(bits[0])
		if e != nil {
			return bad()
		}
		equal = limit.IsInt64() && value == float64(limit.Int64())
	} else {
		value, e := pythoncompat.DecimalInteger(bits[0])
		if e != nil {
			return bad()
		}
		equal = value.Cmp(limit) == 0
	}
	number, err := pythoncompat.DecimalInteger(bits[1])
	if err != nil {
		return bad()
	}
	previous, err := pythoncompat.DecimalInteger(bits[2])
	if err != nil {
		return bad()
	}
	if number.Sign() < 0 && limit.Sign() > 0 {
		return nil, Failure(400, "detail", "Error in parsing")
	}
	if limit.Sign() <= 0 || (!equal && previous.Sign() != 0) {
		return nil, Failure(500, "error", "Something went wrong please try again later")
	}
	return &pagination{limit: limit.Int64(), number: number, offset: new(big.Int).Mul(number, limit)}, nil
}
func (r *Repository) Counts(ctx context.Context, p Principal, slug string) (map[string]int64, error) {
	var total, mentions int64
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FILTER(WHERE UPPER(n.sender) NOT LIKE '%MENTIONED%'),COUNT(*) FILTER(WHERE UPPER(n.sender) LIKE '%MENTIONED%') FROM notifications n JOIN workspaces w ON w.id=n.workspace_id WHERE `+scope+` AND n.read_at IS NULL AND n.archived_at IS NULL AND n.snoozed_till IS NULL`, p.UserID, slug).Scan(&total, &mentions)
	return map[string]int64{"total_unread_notifications_count": total, "mention_unread_notifications_count": mentions}, err
}
