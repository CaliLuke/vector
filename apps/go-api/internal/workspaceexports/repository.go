// Package workspaceexports contains durable workspace export history writes and reads.
package workspaceexports

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrForbidden = errors.New("workspace export requires active member or admin")
var ErrWorkspaceMissing = errors.New("workspace export workspace missing")

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, errors.New("workspace export database required")
	}
	return &Repository{db: db}, nil
}

func (r *Repository) Authorized(ctx context.Context, userID, slug string) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE wm.member_id=$1::uuid AND w.slug=$2 AND wm.role IN (15,20) AND wm.is_active=true AND wm.deleted_at IS NULL)`, userID, slug).Scan(&allowed)
	return allowed, err
}

func (r *Repository) UserEmail(ctx context.Context, userID string) (string, error) {
	var email string
	err := r.db.QueryRow(ctx, `SELECT email FROM users WHERE id=$1::uuid`, userID).Scan(&email)
	return email, err
}

// CreateIssueExport stores values using Django UUIDField coercion while
// returning the original request values for the background task payload.
func (r *Repository) CreateIssueExport(ctx context.Context, userID, slug, provider string, projects []any, includeCustom bool) (json.RawMessage, string, string, []any, error) {
	var workspaceID string
	if err := r.db.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug=$1 AND deleted_at IS NULL`, slug).Scan(&workspaceID); errors.Is(err, pgx.ErrNoRows) {
		return nil, "", "", nil, ErrWorkspaceMissing
	} else if err != nil {
		return nil, "", "", nil, err
	}
	if len(projects) == 0 {
		var data []byte
		if err := r.db.QueryRow(ctx, `SELECT COALESCE(json_agg(p.id::text ORDER BY p.id),'[]'::json) FROM projects p JOIN project_members pm ON pm.project_id=p.id WHERE p.workspace_id=$1::uuid AND pm.member_id=$2::uuid AND pm.is_active=true AND p.archived_at IS NULL AND p.deleted_at IS NULL`, workspaceID, userID).Scan(&data); err != nil {
			return nil, "", "", nil, err
		}
		var defaults []string
		if err := json.Unmarshal(data, &defaults); err != nil {
			return nil, "", "", nil, err
		}
		projects = make([]any, len(defaults))
		for i, project := range defaults {
			projects[i] = project
		}
	}
	return r.createHistory(ctx, userID, workspaceID, provider, projects, includeCustom)
}

func (r *Repository) createHistory(ctx context.Context, userID, workspaceID, provider string, projects []any, includeCustom bool) (json.RawMessage, string, string, []any, error) {
	parsedProjects := make([]pgtype.UUID, 0, len(projects))
	for _, value := range projects {
		id, err := djangoUUIDArrayValue(value)
		if err != nil {
			return nil, "", "", nil, err
		}
		parsedProjects = append(parsedProjects, id)
	}
	id, token := uuid.New(), strings.ReplaceAll(uuid.NewString(), "-", "")
	var response []byte
	err := r.db.QueryRow(ctx, `INSERT INTO exporters (id,created_at,updated_at,type,workspace_id,project,provider,status,reason,key,token,initiated_by_id,created_by_id,updated_by_id,rich_filters) VALUES ($1,clock_timestamp(),clock_timestamp(),'issue_exports',$2,$3,$4,'queued','','',$5,$6,$6,NULL,CASE WHEN $7::boolean THEN '{"include_custom_properties":true}'::jsonb ELSE '{}'::jsonb END) RETURNING json_build_object('id',id,'created_at',to_json(created_at),'updated_at',to_json(updated_at),'project',project,'provider',provider,'status',status,'url',url,'initiated_by',initiated_by_id,'initiated_by_detail',(SELECT json_build_object('id',u.id,'first_name',u.first_name,'last_name',u.last_name,'avatar',u.avatar,'avatar_url',CASE WHEN u.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||u.avatar_asset_id::text||'/' ELSE u.avatar END,'is_bot',u.is_bot,'display_name',u.display_name) FROM users u WHERE u.id=initiated_by_id),'token',token,'created_by',created_by_id,'updated_by',updated_by_id)`, id, workspaceID, parsedProjects, provider, token, userID, includeCustom).Scan(&response)
	if err != nil {
		return nil, "", "", nil, err
	}
	if !json.Valid(response) {
		return nil, "", "", nil, errors.New("export history returned invalid json")
	}
	return json.RawMessage(response), workspaceID, token, projects, nil
}

func djangoUUIDArrayValue(value any) (pgtype.UUID, error) {
	if value == nil {
		return pgtype.UUID{}, nil
	}
	if boolean, ok := value.(bool); ok {
		if boolean {
			value = json.Number("1")
		} else {
			value = json.Number("0")
		}
	}
	parsed, err := djangoinput.UUID(value)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: [16]byte(parsed), Valid: true}, nil
}

func (r *Repository) History(ctx context.Context, userID, slug string, query djangoapi.Query) (json.RawMessage, error) {
	limit, page, offset, err := djangoapi.PageWindow(query)
	if err != nil {
		return nil, err
	}
	order := "created_at DESC"
	switch strings.TrimSpace(query.Params["order_by"]) {
	case "", "-created_at":
		order = "created_at DESC"
	case "created_at":
		order = "created_at ASC"
	case "-updated_at":
		order = "updated_at DESC"
	case "updated_at":
		order = "updated_at ASC"
	case "-id":
		order = "id DESC"
	case "id":
		order = "id ASC"
	default:
		return nil, errors.New("unsupported exporter order_by")
	}
	_ = page
	var response []byte
	err = r.db.QueryRow(ctx, `WITH filtered AS (SELECT e.*,u.first_name,u.last_name,u.display_name,u.avatar,u.avatar_asset_id,u.is_bot FROM exporters e JOIN workspaces w ON w.id=e.workspace_id LEFT JOIN users u ON u.id=e.initiated_by_id WHERE w.slug=$1 AND e.type='issue_exports' AND e.deleted_at IS NULL AND `+customExportHistoryVisibleSQL+`), totals AS (SELECT count(*)::int total FROM filtered), rows AS (SELECT * FROM filtered ORDER BY `+order+` LIMIT $2 OFFSET $3), result AS (SELECT COALESCE(json_agg(json_build_object('id',id,'created_at',to_json(created_at),'updated_at',to_json(updated_at),'project',project,'provider',provider,'status',status,'url',url,'initiated_by',initiated_by_id,'initiated_by_detail',json_build_object('id',initiated_by_id,'first_name',first_name,'last_name',last_name,'avatar',avatar,'avatar_url',CASE WHEN avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||avatar_asset_id::text||'/' ELSE avatar END,'is_bot',is_bot,'display_name',display_name),'token',token,'created_by',created_by_id,'updated_by',updated_by_id) ORDER BY `+order+`),'[]'::json) results FROM rows) SELECT json_build_object('grouped_by',NULL,'sub_grouped_by',NULL,'total_count',totals.total,'next_cursor',concat($2,':',($3/$2)+1,':0'),'prev_cursor',concat($2,':',($3/$2)-1,':1'),'next_page_results',($3+$2)<totals.total,'prev_page_results',$3>0,'count',json_array_length(result.results),'total_pages',CASE WHEN totals.total=0 THEN 0 ELSE CEIL(totals.total::numeric/$2)::int END,'total_results',totals.total,'extra_stats',NULL,'results',result.results) FROM totals CROSS JOIN result`, slug, limit, offset, userID).Scan(&response)
	if err != nil {
		return nil, err
	}
	if !json.Valid(response) {
		return nil, errors.New("export history returned invalid json")
	}
	return json.RawMessage(response), nil
}
