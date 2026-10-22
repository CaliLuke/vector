package workspacerecentvisits

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"uuid"
)

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("workspace recent visits database is required")
	}
	return &Repository{db: db}, nil
}

// recentVisitEntityData mirrors the three serializers used by Django's
// WorkspaceRecentVisitSerializer. The correlated subqueries deliberately
// return NULL for deleted or missing entities, matching DoesNotExist handling.
const recentVisitEntityData = `CASE v.entity_name
WHEN 'issue' THEN (SELECT json_build_object('id',i.id,'name',i.name,'state',i.state_id,'priority',i.priority,'assignees',(SELECT COALESCE(json_agg(ia.assignee_id ORDER BY ia.created_at DESC),'[]'::json) FROM issue_assignees ia WHERE ia.issue_id=i.id AND ia.deleted_at IS NULL),'type',i.type_id,'sequence_id',i.sequence_id,'project_id',i.project_id,'project_identifier',(SELECT p.identifier FROM projects p WHERE p.id=i.project_id AND p.deleted_at IS NULL)) FROM issues i WHERE i.id=v.entity_identifier AND i.deleted_at IS NULL)
WHEN 'page' THEN (SELECT json_build_object('id',p.id,'name',p.name,'logo_props',p.logo_props,'project_id',(SELECT pp.project_id FROM project_pages pp JOIN projects ppj ON ppj.id=pp.project_id AND ppj.deleted_at IS NULL WHERE pp.page_id=p.id AND pp.deleted_at IS NULL ORDER BY ppj.created_at DESC LIMIT 1),'owned_by',p.owned_by_id,'project_identifier',(SELECT ppj.identifier FROM project_pages pp JOIN projects ppj ON ppj.id=pp.project_id AND ppj.deleted_at IS NULL WHERE pp.page_id=p.id AND pp.deleted_at IS NULL ORDER BY ppj.created_at DESC LIMIT 1)) FROM pages p WHERE p.id=v.entity_identifier AND p.deleted_at IS NULL)
WHEN 'project' THEN (SELECT json_build_object('id',p.id,'name',p.name,'logo_props',p.logo_props,'project_members',(SELECT COALESCE(json_agg(pm.member_id ORDER BY pm.created_at DESC),'[]'::json) FROM project_members pm JOIN users u ON u.id=pm.member_id WHERE pm.project_id=p.id AND pm.is_active=true AND pm.deleted_at IS NULL AND u.is_bot=false),'identifier',p.identifier) FROM projects p WHERE p.id=v.entity_identifier AND p.deleted_at IS NULL)
ELSE NULL END`

const recentVisitJSON = `json_build_object('id',v.id,'entity_name',v.entity_name,'entity_identifier',v.entity_identifier,'entity_data',` + recentVisitEntityData + `,'visited_at',v.visited_at)`

func (r *Repository) Authorized(ctx context.Context, user uuid.UUID, slug string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE wm.member_id=$1 AND w.slug=$2 AND wm.role IN (5,15,20) AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL)`, user, slug).Scan(&ok)
	return ok, err
}

func (r *Repository) List(ctx context.Context, user uuid.UUID, slug string, entityName *string) (json.RawMessage, error) {
	var body []byte
	var filter any
	if entityName != nil {
		filter = *entityName
	}
	err := r.db.QueryRow(ctx, `SELECT COALESCE(json_agg(visit ORDER BY created_at DESC),'[]'::json) FROM (SELECT `+recentVisitJSON+` AS visit,v.created_at FROM user_recent_visits v JOIN workspaces w ON w.id=v.workspace_id WHERE v.user_id=$1 AND w.slug=$2 AND v.entity_name=COALESCE($3,v.entity_name) AND v.entity_name IN ('issue','page','project') AND v.deleted_at IS NULL AND w.deleted_at IS NULL ORDER BY v.created_at DESC LIMIT 20) visits`, user, slug, filter).Scan(&body)
	return body, err
}

// Record updates an existing visit or creates one, then retains the newest 20
// active visits for that user and workspace. The workspace row lock serializes
// concurrent records that would otherwise race past the cap.
func (r *Repository) Record(
	ctx context.Context,
	user uuid.UUID,
	slug string,
	entityName string,
	entityID uuid.UUID,
	projectID uuid.UUID,
) error {
	var recorded bool
	var pruned int
	err := r.db.QueryRow(ctx, `
WITH target_workspace AS MATERIALIZED (
  SELECT workspace.id
  FROM workspaces workspace
  WHERE workspace.slug=$2
    AND workspace.deleted_at IS NULL
  FOR UPDATE
),
target AS (
  SELECT workspace.id AS workspace_id
  FROM target_workspace workspace
),
updated AS (
  UPDATE user_recent_visits visit
  SET visited_at=NOW()
  FROM target
  WHERE visit.id=(
    SELECT candidate.id
    FROM user_recent_visits candidate
    WHERE candidate.user_id=$1
      AND candidate.workspace_id=target.workspace_id
      AND candidate.entity_name=$3
      AND candidate.entity_identifier=$4
      AND candidate.project_id=$5
      AND candidate.deleted_at IS NULL
    ORDER BY candidate.created_at DESC
    LIMIT 1
  )
  RETURNING visit.id, visit.created_at
),
inserted AS (
  INSERT INTO user_recent_visits (
    id, created_at, updated_at, deleted_at, created_by_id, updated_by_id,
    entity_identifier, entity_name, user_id, workspace_id, project_id, visited_at
  )
  SELECT $6, NOW(), NOW(), NULL, $1, $1, $4, $3, $1, target.workspace_id, $5, NOW()
  FROM target
  WHERE NOT EXISTS (SELECT 1 FROM updated)
  RETURNING id, created_at
),
recorded AS (
  SELECT id, created_at FROM updated
  UNION ALL
  SELECT id, created_at FROM inserted
),
active_candidates AS (
  SELECT visit.id, visit.created_at
  FROM user_recent_visits visit
  JOIN target ON target.workspace_id=visit.workspace_id
  WHERE visit.user_id=$1
    AND visit.deleted_at IS NULL
  UNION ALL
  SELECT recorded.id, recorded.created_at
  FROM recorded
  WHERE NOT EXISTS (SELECT 1 FROM user_recent_visits existing WHERE existing.id=recorded.id)
),
ranked AS (
  SELECT candidate.id,
    row_number() OVER (ORDER BY candidate.created_at DESC, candidate.id DESC) AS position
  FROM active_candidates candidate
),
pruned AS (
  UPDATE user_recent_visits visit
  SET deleted_at=NOW()
  WHERE visit.id IN (SELECT ranked.id FROM ranked WHERE ranked.position>20)
  RETURNING visit.id
)
SELECT EXISTS(SELECT 1 FROM recorded), (SELECT count(*)::int FROM pruned)
`, user, slug, entityName, entityID, projectID, uuid.New()).Scan(&recorded, &pruned)
	if err != nil {
		return fmt.Errorf("record workspace recent visit: %w", err)
	}
	if !recorded {
		return fmt.Errorf("record workspace recent visit: workspace %q not found", slug)
	}
	return nil
}
