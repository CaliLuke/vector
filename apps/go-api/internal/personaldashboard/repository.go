// Package personaldashboard implements the PostgreSQL reads used by the
// authenticated user's workspace dashboard endpoints.
package personaldashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/jackc/pgx/v5"
)

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, errors.New("personal dashboard database is required")
	}
	return &Repository{db: db}, nil
}

func (r *Repository) Dashboard(ctx context.Context, userID, slug, month string) (json.RawMessage, error) {
	return r.json(ctx, `WITH ws AS (SELECT id,slug FROM workspaces WHERE slug=$2),
activities AS (
 SELECT COALESCE(json_agg(json_build_object('created_date',x.day,'activity_count',x.n) ORDER BY x.day),'[]'::json) value
 FROM (SELECT a.created_at::date AS day,count(a.id)::int n FROM issue_activities a JOIN ws ON ws.id=a.workspace_id WHERE a.actor_id=$1 AND a.deleted_at IS NULL AND `+issuequery.MilestoneActivityVisibleSQL("a.field", "a.issue_id", "a.project_id", "a.workspace_id", "$1::uuid")+` AND a.created_at::date>=CURRENT_DATE-INTERVAL '3 months' GROUP BY 1) x),
completed AS (
 SELECT COALESCE(json_agg(json_build_object('week_in_month',x.week,'completed_count',x.n) ORDER BY x.week),'[]'::json) value
 FROM (SELECT floor((extract(day FROM i.completed_at)-1)/7)+1 week,count(i.id)::int n FROM issues i JOIN ws ON ws.id=i.workspace_id JOIN projects p ON p.id=i.project_id LEFT JOIN states s ON s.id=i.state_id JOIN issue_assignees ia ON ia.issue_id=i.id AND ia.assignee_id=$1 WHERE i.deleted_at IS NULL AND s."group" IS DISTINCT FROM 'triage' AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE AND i.completed_at IS NOT NULL AND p.archived_at IS NULL AND extract(month FROM i.completed_at)=$3::numeric GROUP BY 1) x),
base_issues AS (
 SELECT i.id,i.state_id,i.target_date,i.start_date,i.completed_at,s."group" state_group,p.id project_id,ws.slug
 FROM issues i JOIN ws ON ws.id=i.workspace_id JOIN projects p ON p.id=i.project_id
 JOIN issue_assignees ia ON ia.issue_id=i.id AND ia.assignee_id=$1
 LEFT JOIN states s ON s.id=i.state_id
 WHERE i.deleted_at IS NULL AND s."group" IS DISTINCT FROM 'triage' AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE AND p.archived_at IS NULL),
state_counts AS (SELECT COALESCE(json_agg(json_build_object('state_group',state_group,'state_count',n) ORDER BY state_group),'[]'::json) value FROM (SELECT state_group,count(state_group)::int n FROM base_issues GROUP BY state_group) c),
overdue AS (SELECT COALESCE(json_agg(json_build_object('id',i.id,'name',i.name,'workspace__slug',ws.slug,'project_id',p.id,'target_date',i.target_date) ORDER BY i.created_at DESC),'[]'::json) value FROM issues i JOIN ws ON ws.id=i.workspace_id JOIN projects p ON p.id=i.project_id JOIN issue_assignees ia ON ia.issue_id=i.id AND ia.assignee_id=$1 LEFT JOIN states s ON s.id=i.state_id WHERE i.deleted_at IS NULL AND s."group" IS DISTINCT FROM 'triage' AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE AND p.archived_at IS NULL AND (s."group" IS NULL OR s."group" NOT IN ('completed','cancelled')) AND i.target_date<CURRENT_TIMESTAMP AND i.completed_at IS NULL),
upcoming AS (SELECT COALESCE(json_agg(json_build_object('id',i.id,'name',i.name,'workspace__slug',ws.slug,'project_id',p.id,'start_date',i.start_date) ORDER BY i.created_at DESC),'[]'::json) value FROM issues i JOIN ws ON ws.id=i.workspace_id JOIN projects p ON p.id=i.project_id JOIN issue_assignees ia ON ia.issue_id=i.id AND ia.assignee_id=$1 LEFT JOIN states s ON s.id=i.state_id WHERE i.deleted_at IS NULL AND s."group" IS DISTINCT FROM 'triage' AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE AND p.archived_at IS NULL AND (s."group" IS NULL OR s."group" NOT IN ('completed','cancelled')) AND i.start_date>=CURRENT_TIMESTAMP AND i.completed_at IS NULL)
SELECT json_build_object(
 'issue_activities',(SELECT value FROM activities),
 'completed_issues',(SELECT value FROM completed),
 'assigned_issues_count',(SELECT count(*) FROM base_issues),
 'pending_issues_count',(SELECT count(*) FROM base_issues WHERE (state_group IS NULL OR state_group NOT IN ('completed','cancelled'))),
 'completed_issues_count',(SELECT count(*) FROM base_issues WHERE state_group='completed'),
 'issues_due_week_count',(SELECT count(*) FROM base_issues WHERE extract(week FROM target_date)=extract(week FROM CURRENT_DATE)),
 'state_distribution',(SELECT value FROM state_counts),
 'overdue_issues',(SELECT value FROM overdue),
 'upcoming_issues',(SELECT value FROM upcoming))`, userID, slug, month)
}

func (r *Repository) ActivityGraph(ctx context.Context, userID, slug string) (json.RawMessage, error) {
	return r.json(ctx, `SELECT COALESCE(json_agg(json_build_object('created_date',day,'activity_count',n) ORDER BY day),'[]'::json) FROM (SELECT a.created_at::date AS day,count(a.id)::int n FROM issue_activities a JOIN workspaces w ON w.id=a.workspace_id WHERE a.actor_id=$1 AND a.deleted_at IS NULL AND `+issuequery.MilestoneActivityVisibleSQL("a.field", "a.issue_id", "a.project_id", "a.workspace_id", "$1::uuid")+` AND w.slug=$2 AND a.created_at::date>=CURRENT_DATE-INTERVAL '6 months' GROUP BY 1) x`, userID, slug)
}

func (r *Repository) CompletedGraph(ctx context.Context, userID, slug, month string) (json.RawMessage, error) {
	return r.json(ctx, `SELECT COALESCE(json_agg(json_build_object('week',week,'completed_count',n) ORDER BY week),'[]'::json) FROM (SELECT extract(week FROM i.completed_at)::int%4 week,count(extract(week FROM i.completed_at))::int n FROM issues i JOIN workspaces w ON w.id=i.workspace_id JOIN projects p ON p.id=i.project_id LEFT JOIN states s ON s.id=i.state_id JOIN issue_assignees ia ON ia.issue_id=i.id AND ia.assignee_id=$1 WHERE w.slug=$2 AND i.deleted_at IS NULL AND s."group" IS DISTINCT FROM 'triage' AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE AND i.completed_at IS NOT NULL AND p.archived_at IS NULL AND extract(month FROM i.completed_at)=$3::numeric GROUP BY 1) x`, userID, slug, month)
}

func (r *Repository) json(ctx context.Context, query string, args ...any) (json.RawMessage, error) {
	var raw []byte
	if err := r.db.QueryRow(ctx, query, args...).Scan(&raw); err != nil {
		return nil, fmt.Errorf("personal dashboard query: %w", err)
	}
	if !json.Valid(raw) {
		return nil, errors.New("personal dashboard returned invalid JSON")
	}
	return json.RawMessage(raw), nil
}
