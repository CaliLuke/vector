package workspacefavorites

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"uuid"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrNotFound = errors.New("workspace favorite not found")
var ErrAlreadyExists = errors.New("workspace favorite already exists")
var ErrInvalidScope = errors.New("workspace favorite target is outside the user's accessible scope")

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}

type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("workspace favorites database is required")
	}
	return &Repository{db: db}, nil
}

// CreateInput mirrors the writable fields of UserFavorite. A nil pointer means
// that the corresponding nullable field should be NULL (or use its default).
type CreateInput struct {
	EntityType       string
	EntityIdentifier *uuid.UUID
	Name             *string
	IsFolder         bool
	Sequence         *float64
	Parent           *uuid.UUID
	ProjectID        *uuid.UUID
}

// UpdateInput carries PATCH presence separately from values. This is needed
// because nil is a valid value for entity_identifier, name, parent, and
// project_id.
type UpdateInput struct {
	EntityTypePresent       bool
	EntityType              *string
	EntityIdentifierPresent bool
	EntityIdentifier        *uuid.UUID
	NamePresent             bool
	Name                    *string
	IsFolderPresent         bool
	IsFolder                *bool
	SequencePresent         bool
	Sequence                *float64
	ParentPresent           bool
	Parent                  *uuid.UUID
	ProjectIDPresent        bool
	ProjectID               *uuid.UUID
}

const favoriteEntityData = `CASE f.entity_type
WHEN 'project' THEN (SELECT json_build_object('id',e.id,'name',e.name,'logo_props',e.logo_props) FROM projects e WHERE e.id=f.entity_identifier AND e.deleted_at IS NULL)
WHEN 'page' THEN (SELECT json_build_object('id',e.id,'name',e.name,'logo_props',e.logo_props,'project_id',f.project_id) FROM pages e WHERE e.id=f.entity_identifier AND e.deleted_at IS NULL)
WHEN 'cycle' THEN (SELECT json_build_object('id',e.id,'name',e.name,'logo_props',e.logo_props,'project_id',e.project_id) FROM cycles e WHERE e.id=f.entity_identifier AND e.deleted_at IS NULL)
WHEN 'module' THEN (SELECT json_build_object('id',e.id,'name',e.name,'logo_props',e.logo_props,'project_id',e.project_id) FROM modules e WHERE e.id=f.entity_identifier AND e.deleted_at IS NULL)
WHEN 'view' THEN (SELECT json_build_object('id',e.id,'name',e.name,'logo_props',e.logo_props,'project_id',e.project_id) FROM issue_views e WHERE e.id=f.entity_identifier AND e.deleted_at IS NULL)
ELSE NULL END`

const favoriteJSON = `json_build_object('id',f.id,'entity_type',f.entity_type,'entity_identifier',f.entity_identifier,'entity_data',` + favoriteEntityData + `,'name',f.name,'is_folder',f.is_folder,'sequence',f.sequence,'parent',f.parent_id,'workspace_id',f.workspace_id,'project_id',f.project_id)`

func favoriteScopePredicate(favoriteAlias, workspaceExpression string) string {
	const userExpression = "$1"
	projectMember := func(projectExpression string) string {
		return projectMembershipSQL(projectExpression, userExpression, workspaceExpression)
	}

	entity := `((` + favoriteAlias + `.entity_type='folder' AND ` + favoriteAlias + `.is_folder=true AND ` + favoriteAlias + `.entity_identifier IS NULL AND ` + favoriteAlias + `.project_id IS NULL) OR (` + favoriteAlias + `.is_folder=false AND CASE ` + favoriteAlias + `.entity_type
		WHEN 'project' THEN ` + favoriteAlias + `.entity_identifier IS NOT NULL AND ` + favoriteAlias + `.project_id=` + favoriteAlias + `.entity_identifier AND ` + projectMember(favoriteAlias+`.entity_identifier`) + `
		WHEN 'page' THEN ` + favoriteAlias + `.entity_identifier IS NOT NULL AND ` + favoritePageScopeSQL(favoriteAlias+`.entity_identifier`, favoriteAlias+`.project_id`, userExpression, workspaceExpression) + `
		WHEN 'cycle' THEN ` + favoriteAlias + `.entity_identifier IS NOT NULL AND ` + favoriteAlias + `.project_id IS NOT NULL AND EXISTS (SELECT 1 FROM cycles scoped_entity WHERE scoped_entity.id=` + favoriteAlias + `.entity_identifier AND scoped_entity.project_id=` + favoriteAlias + `.project_id AND scoped_entity.workspace_id=` + workspaceExpression + ` AND scoped_entity.deleted_at IS NULL AND ` + projectMember(favoriteAlias+`.project_id`) + `)
		WHEN 'module' THEN ` + favoriteAlias + `.entity_identifier IS NOT NULL AND ` + favoriteAlias + `.project_id IS NOT NULL AND EXISTS (SELECT 1 FROM modules scoped_entity WHERE scoped_entity.id=` + favoriteAlias + `.entity_identifier AND scoped_entity.project_id=` + favoriteAlias + `.project_id AND scoped_entity.workspace_id=` + workspaceExpression + ` AND scoped_entity.deleted_at IS NULL AND ` + projectMember(favoriteAlias+`.project_id`) + `)
		WHEN 'view' THEN ` + favoriteAlias + `.entity_identifier IS NOT NULL AND ` + favoriteAlias + `.project_id IS NOT NULL AND EXISTS (SELECT 1 FROM issue_views scoped_entity WHERE scoped_entity.id=` + favoriteAlias + `.entity_identifier AND scoped_entity.project_id=` + favoriteAlias + `.project_id AND scoped_entity.workspace_id=` + workspaceExpression + ` AND scoped_entity.deleted_at IS NULL AND ` + projectMember(favoriteAlias+`.project_id`) + `)
		ELSE false END))`
	parent := favoriteParentScopeSQL(favoriteAlias, userExpression, workspaceExpression)
	workspaceMember := `EXISTS (SELECT 1 FROM workspace_members scoped_workspace_member WHERE scoped_workspace_member.workspace_id=` + workspaceExpression + ` AND scoped_workspace_member.member_id=` + userExpression + ` AND scoped_workspace_member.role IN (15,20) AND scoped_workspace_member.is_active=true AND scoped_workspace_member.deleted_at IS NULL)`
	return entity + ` AND ` + parent + ` AND ` + workspaceMember
}

func favoriteParentScopeSQL(favoriteAlias, userExpression, workspaceExpression string) string {
	parent := `EXISTS (SELECT 1 FROM user_favorites scoped_parent WHERE scoped_parent.id=` + favoriteAlias + `.parent_id AND scoped_parent.workspace_id=` + workspaceExpression + ` AND scoped_parent.user_id=` + userExpression + ` AND scoped_parent.entity_type='folder' AND scoped_parent.is_folder=true AND scoped_parent.entity_identifier IS NULL AND scoped_parent.project_id IS NULL AND scoped_parent.deleted_at IS NULL)`
	acyclic := `NOT EXISTS (WITH RECURSIVE parent_chain(id,parent_id,path) AS (` +
		`SELECT scoped_parent.id,scoped_parent.parent_id,ARRAY[scoped_parent.id] FROM user_favorites scoped_parent WHERE scoped_parent.id=` + favoriteAlias + `.parent_id AND scoped_parent.workspace_id=` + workspaceExpression + ` AND scoped_parent.user_id=` + userExpression + ` AND scoped_parent.deleted_at IS NULL ` +
		`UNION ALL SELECT ancestor.id,ancestor.parent_id,parent_chain.path || ancestor.id FROM user_favorites ancestor JOIN parent_chain ON ancestor.id=parent_chain.parent_id WHERE ancestor.workspace_id=` + workspaceExpression + ` AND ancestor.user_id=` + userExpression + ` AND ancestor.deleted_at IS NULL AND NOT ancestor.id=ANY(parent_chain.path) ` +
		`) SELECT 1 FROM parent_chain WHERE parent_chain.id=` + favoriteAlias + `.id OR parent_chain.parent_id=ANY(parent_chain.path))`
	return `(` + favoriteAlias + `.parent_id IS NULL OR (` + parent + ` AND ` + acyclic + `))`
}

func projectMembershipSQL(projectExpression, userExpression, workspaceExpression string) string {
	return `EXISTS (SELECT 1 FROM projects scoped_project JOIN project_members scoped_member ON scoped_member.project_id=scoped_project.id WHERE scoped_project.id=` + projectExpression + ` AND scoped_project.workspace_id=` + workspaceExpression + ` AND scoped_project.deleted_at IS NULL AND scoped_member.workspace_id=` + workspaceExpression + ` AND scoped_member.member_id=` + userExpression + ` AND scoped_member.role IN (5,15,20) AND scoped_member.is_active=true AND scoped_member.deleted_at IS NULL)`
}

func favoritePageScopeSQL(entityExpression, projectExpression, userExpression, workspaceExpression string) string {
	return `((` + projectExpression + ` IS NOT NULL AND EXISTS (SELECT 1 FROM pages scoped_page_entity JOIN project_pages scoped_page_link ON scoped_page_link.page_id=scoped_page_entity.id AND scoped_page_link.project_id=` + projectExpression + ` AND scoped_page_link.workspace_id=` + workspaceExpression + ` AND scoped_page_link.deleted_at IS NULL JOIN projects scoped_page_project ON scoped_page_project.id=scoped_page_link.project_id AND scoped_page_project.workspace_id=` + workspaceExpression + ` AND scoped_page_project.deleted_at IS NULL JOIN project_members scoped_page_member ON scoped_page_member.project_id=scoped_page_project.id AND scoped_page_member.workspace_id=` + workspaceExpression + ` AND scoped_page_member.member_id=` + userExpression + ` AND scoped_page_member.role IN (5,15,20) AND scoped_page_member.is_active=true AND scoped_page_member.deleted_at IS NULL WHERE scoped_page_entity.id=` + entityExpression + ` AND scoped_page_entity.workspace_id=` + workspaceExpression + ` AND scoped_page_entity.deleted_at IS NULL AND (scoped_page_entity.owned_by_id=` + userExpression + ` OR scoped_page_entity.access=0) AND (scoped_page_member.role<>5 OR scoped_page_project.guest_view_all_features OR scoped_page_entity.owned_by_id=` + userExpression + `))) OR (` + projectExpression + ` IS NULL AND EXISTS (SELECT 1 FROM pages scoped_page_entity WHERE scoped_page_entity.id=` + entityExpression + ` AND scoped_page_entity.workspace_id=` + workspaceExpression + ` AND scoped_page_entity.is_global=true AND scoped_page_entity.deleted_at IS NULL AND (scoped_page_entity.owned_by_id=` + userExpression + ` OR scoped_page_entity.access=0))))`
}

func (r *Repository) Authorized(ctx context.Context, user uuid.UUID, slug string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE wm.member_id=$1 AND w.slug=$2 AND wm.role IN (15,20) AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL)`, user, slug).Scan(&ok)
	return ok, err
}

func (r *Repository) List(ctx context.Context, user uuid.UUID, slug string, all bool) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `SELECT COALESCE(json_agg(favorite ORDER BY created_at DESC),'[]'::json) FROM (SELECT `+favoriteJSON+` AS favorite,f.created_at FROM user_favorites f JOIN workspaces w ON w.id=f.workspace_id WHERE f.user_id=$1 AND w.slug=$2 AND ($3 OR f.parent_id IS NULL) AND f.deleted_at IS NULL AND w.deleted_at IS NULL AND `+favoriteScopePredicate("f", "w.id")+`) x`, user, slug, all).Scan(&body)
	return body, err
}

func (r *Repository) Group(ctx context.Context, user uuid.UUID, slug string, parent uuid.UUID) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `SELECT COALESCE(json_agg(favorite ORDER BY created_at DESC),'[]'::json) FROM (SELECT `+favoriteJSON+` AS favorite,f.created_at FROM user_favorites f JOIN workspaces w ON w.id=f.workspace_id WHERE f.user_id=$1 AND w.slug=$2 AND f.parent_id=$3 AND f.deleted_at IS NULL AND w.deleted_at IS NULL AND EXISTS (SELECT 1 FROM user_favorites folder WHERE folder.id=$3 AND folder.workspace_id=w.id AND folder.user_id=$1 AND folder.entity_type='folder' AND folder.is_folder=true AND folder.entity_identifier IS NULL AND folder.project_id IS NULL AND folder.deleted_at IS NULL) AND `+favoriteScopePredicate("f", "w.id")+`) x`, user, slug, parent).Scan(&body)
	return body, err
}

func (r *Repository) Get(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `SELECT `+favoriteJSON+` FROM user_favorites f JOIN workspaces w ON w.id=f.workspace_id WHERE f.id=$3 AND f.user_id=$1 AND w.slug=$2 AND f.deleted_at IS NULL AND w.deleted_at IS NULL AND `+favoriteScopePredicate("f", "w.id"), user, slug, id).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return body, err
}

func (r *Repository) Create(ctx context.Context, user uuid.UUID, slug string, input CreateInput) (json.RawMessage, error) {
	var body []byte
	// Django returns the existing favorite when an entity identifier is given.
	// The CTE validates the requested entity, project membership, and parent in
	// the same statement that inserts it, while the unique index handles races.
	query := `WITH candidate AS (
		SELECT requested.workspace_id
		FROM (
			SELECT w.id AS workspace_id,$10::uuid AS id,$3::text AS entity_type,$4::uuid AS entity_identifier,$6::boolean AS is_folder,$8::uuid AS parent_id,$9::uuid AS project_id
			FROM workspaces w WHERE w.slug=$2 AND w.deleted_at IS NULL
		) requested
		WHERE ` + favoriteScopePredicate("requested", "requested.workspace_id") + `
	), existing AS (
		SELECT ` + favoriteJSON + ` AS favorite FROM user_favorites f
		JOIN workspaces w ON w.id=f.workspace_id
		JOIN candidate c ON c.workspace_id=w.id
		WHERE $4 IS NOT NULL AND f.user_id=$1 AND w.slug=$2 AND f.entity_type=$3
			AND f.entity_identifier=$4 AND f.deleted_at IS NULL AND w.deleted_at IS NULL
			AND ` + favoriteScopePredicate("f", "w.id") + `
		ORDER BY f.created_at ASC LIMIT 1
	), inserted AS (
		INSERT INTO user_favorites AS f(workspace_id,user_id,entity_type,entity_identifier,name,is_folder,sequence,parent_id,project_id,created_by_id,id,created_at,updated_at)
		SELECT c.workspace_id,$1,$3,$4,$5,$6,
			CASE WHEN EXISTS (SELECT 1 FROM user_favorites previous WHERE previous.workspace_id=COALESCE(p.workspace_id,c.workspace_id) AND previous.deleted_at IS NULL)
			THEN COALESCE((SELECT MAX(previous.sequence) FROM user_favorites previous WHERE previous.workspace_id=COALESCE(p.workspace_id,c.workspace_id) AND previous.deleted_at IS NULL),65535)+10000
			ELSE COALESCE($7,65535) END,$8,$9,$1,$10,NOW(),NOW()
		FROM candidate c LEFT JOIN projects p ON p.id=$9 AND p.deleted_at IS NULL
		WHERE NOT EXISTS (SELECT 1 FROM existing)
		RETURNING ` + favoriteJSON + ` AS favorite
	), selected AS (
		SELECT favorite FROM existing UNION ALL SELECT favorite FROM inserted LIMIT 1
	)
	SELECT favorite FROM selected`
	err := r.db.QueryRow(ctx, query, user, slug, input.EntityType, input.EntityIdentifier, input.Name, input.IsFolder, input.Sequence, input.Parent, input.ProjectID, uuid.NewV7()).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidScope
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nil, ErrAlreadyExists
	}
	return body, err
}

func (r *Repository) Update(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID, input UpdateInput) (_ json.RawMessage, resultErr error) {
	query := `WITH candidate AS (SELECT f.workspace_id,f.user_id,f.created_at,f.id,CASE WHEN $4 THEN $5 ELSE f.entity_type END AS entity_type,CASE WHEN $6 THEN $7 ELSE f.entity_identifier END AS entity_identifier,CASE WHEN $8 THEN $9 ELSE f.name END AS name,CASE WHEN $10 THEN $11 ELSE f.is_folder END AS is_folder,CASE WHEN $12 THEN $13 ELSE f.sequence END AS sequence,CASE WHEN $14 THEN $15 ELSE f.parent_id END AS parent_id,CASE WHEN $16 THEN $17 ELSE f.project_id END AS project_id FROM user_favorites f JOIN workspaces w ON w.id=f.workspace_id WHERE f.id=$3 AND f.workspace_id=w.id AND f.user_id=$1 AND w.slug=$2 AND f.deleted_at IS NULL AND w.deleted_at IS NULL), updated AS (UPDATE user_favorites f SET entity_type=CASE WHEN $4 THEN $5 ELSE f.entity_type END,entity_identifier=CASE WHEN $6 THEN $7 ELSE f.entity_identifier END,name=CASE WHEN $8 THEN $9 ELSE f.name END,is_folder=CASE WHEN $10 THEN $11 ELSE f.is_folder END,sequence=CASE WHEN $12 THEN $13 ELSE f.sequence END,parent_id=CASE WHEN $14 THEN $15 ELSE f.parent_id END,project_id=CASE WHEN $16 THEN $17 ELSE f.project_id END,updated_at=NOW(),updated_by_id=$1 FROM candidate c WHERE f.id=c.id AND ` + favoriteScopePredicate("c", "c.workspace_id") + ` RETURNING f.*) SELECT ` + favoriteJSON + ` FROM updated f`
	args := []any{user, slug, id, input.EntityTypePresent, input.EntityType, input.EntityIdentifierPresent, input.EntityIdentifier, input.NamePresent, input.Name, input.IsFolderPresent, input.IsFolder, input.SequencePresent, input.Sequence, input.ParentPresent, input.Parent, input.ProjectIDPresent, input.ProjectID}
	var body []byte
	var err error
	if input.ParentPresent {
		tx, beginErr := r.db.Begin(ctx)
		if beginErr != nil {
			return nil, beginErr
		}
		defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
		var membershipID uuid.UUID
		lockQuery := `SELECT wm.id FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE wm.member_id=$1 AND w.slug=$2 AND wm.role IN (15,20) AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL FOR UPDATE OF wm`
		if err = tx.QueryRow(ctx, lockQuery, user, slug).Scan(&membershipID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("workspace favorite update membership lock: %w", ErrNotFound)
			}
			return nil, err
		}
		err = tx.QueryRow(ctx, query, args...).Scan(&body)
		if err == nil {
			err = tx.Commit(ctx)
		}
	} else {
		err = r.db.QueryRow(ctx, query, args...).Scan(&body)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nil, ErrAlreadyExists
	}
	return body, err
}

func (r *Repository) Delete(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM user_favorites f USING workspaces w WHERE f.id=$3 AND f.workspace_id=w.id AND f.user_id=$1 AND w.slug=$2 AND f.deleted_at IS NULL AND w.deleted_at IS NULL`, user, slug, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
