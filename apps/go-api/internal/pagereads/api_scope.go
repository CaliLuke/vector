package pagereads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PageAPIQuery struct {
	ActorID, WorkspaceID, ProjectID, PageID, CollectionID, ItemID, LinkID, MemberID, UserID, ParentID uuid.UUID
	WorkspaceSlug, Tool, Action, Search                                                               string
	Params                                                                                            map[string]string
	Present                                                                                           map[string]bool
	IncludeRevision                                                                                   bool
}
type PageAPICommand struct {
	PageAPIQuery
	Fields           map[string]any
	PageIDs          []uuid.UUID
	RequestID        *uuid.UUID
	ExpectedRevision *string
	Revalidate       func(context.Context) error
}

func NewPageAPIRepository(pool *pgxpool.Pool) *Repository { return &Repository{database: pool} }
func pageAPIToken(workspace uuid.UUID, revision int64) string {
	return fmt.Sprintf("page:%s:%d", workspace, revision)
}
func pageAPIWorkspace(ctx context.Context, tx pgx.Tx, q PageAPIQuery, write bool) (int, error) {
	if q.ActorID == uuid.Nil || q.WorkspaceID == uuid.Nil || q.WorkspaceSlug == "" || q.Tool != "page" && q.Tool != "collection" {
		return 0, projectmutations.ErrInvalid
	}
	suffix := ""
	if write {
		suffix = " FOR SHARE OF u,wm,w"
	}
	var role int
	err := tx.QueryRow(ctx, `SELECT wm.role FROM users u JOIN workspace_members wm ON wm.member_id=u.id JOIN workspaces w ON w.id=wm.workspace_id WHERE u.id=$1 AND u.is_active AND w.id=$2 AND w.slug=$3 AND w.deleted_at IS NULL AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20)`+suffix, q.ActorID, q.WorkspaceID, q.WorkspaceSlug).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		err = projectmutations.ErrForbidden
	}
	return role, err
}
func pageAPIProject(ctx context.Context, tx pgx.Tx, q PageAPIQuery, write bool) (int, error) {
	suffix := ""
	if write {
		suffix = " FOR SHARE OF p,pm"
	}
	var role int
	err := tx.QueryRow(ctx, `SELECT pm.role FROM projects p JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=p.workspace_id WHERE p.id=$1 AND p.workspace_id=$2 AND p.deleted_at IS NULL AND p.archived_at IS NULL AND pm.member_id=$3 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20)`+suffix, q.ProjectID, q.WorkspaceID, q.ActorID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		err = projectmutations.ErrForbidden
	}
	return role, err
}
func pageAPIRow(ctx context.Context, tx pgx.Tx, query string, args ...any) (map[string]any, error) {
	var raw []byte
	err := tx.QueryRow(ctx, query, args...).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, projectmutations.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var row map[string]any
	err = json.Unmarshal(raw, &row)
	return row, err
}
func collectionScope(ctx context.Context, tx pgx.Tx, q PageAPIQuery, id uuid.UUID, write, deleted bool) (map[string]any, int, error) {
	suffix := ""
	if write {
		suffix = " FOR UPDATE"
	}
	row, err := pageAPIRow(ctx, tx, `SELECT to_jsonb(c) FROM page_collections c WHERE id=$1 AND workspace_id=$2`+suffix, id, q.WorkspaceID)
	if err != nil {
		return nil, 0, err
	}
	if row["deleted_at"] != nil && !deleted {
		return nil, 0, projectmutations.ErrNotFound
	}
	var access *int
	if row["owned_by_id"] == q.ActorID.String() {
		value := 2
		access = &value
	} else if row["deleted_at"] == nil {
		err = tx.QueryRow(ctx, `SELECT collection_access($1,$2,$3)`, id, q.ActorID, q.WorkspaceID).Scan(&access)
	}
	if err != nil {
		return nil, 0, err
	}
	if access == nil || write && *access != 2 {
		return nil, 0, projectmutations.ErrForbidden
	}
	if write {
		if _, err = tx.Exec(ctx, `SELECT id FROM page_collection_members WHERE collection_id=$1 AND workspace_id=$2 AND member_id=$3 AND deleted_at IS NULL FOR SHARE`, id, q.WorkspaceID, q.ActorID); err != nil {
			return nil, 0, err
		}
	}
	return row, *access, nil
}
func pageScope(ctx context.Context, tx pgx.Tx, q PageAPIQuery, id uuid.UUID, write, deleted, linked bool) (map[string]any, error) {
	suffix := ""
	if write {
		suffix = " FOR UPDATE"
	}
	row, err := pageAPIRow(ctx, tx, `SELECT to_jsonb(p) FROM pages p WHERE id=$1 AND workspace_id=$2`+suffix, id, q.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if row["deleted_at"] != nil && !deleted {
		return nil, projectmutations.ErrNotFound
	}
	global := row["is_global"] == true
	if global && q.ProjectID != uuid.Nil && !linked || !global && q.ProjectID == uuid.Nil {
		return nil, projectmutations.ErrNotFound
	}
	role, err := pageAPIWorkspace(ctx, tx, q, write)
	if err != nil {
		return nil, err
	}
	owner := row["owned_by_id"] == q.ActorID.String()
	if global {
		var visible bool
		if row["deleted_at"] == nil {
			err = tx.QueryRow(ctx, `SELECT workspace_page_visible($1,$2,$3)`, id, q.ActorID, q.WorkspaceID).Scan(&visible)
		} else {
			visible = owner || row["access"] == float64(0)
			if parent, ok := row["parent_id"].(string); ok && visible {
				err = tx.QueryRow(ctx, `SELECT workspace_page_visible($1,$2,$3)`, parent, q.ActorID, q.WorkspaceID).Scan(&visible)
			}
		}
		if err != nil {
			return nil, err
		}
		if !visible {
			return nil, projectmutations.ErrForbidden
		}
		var collection *string
		if err = tx.QueryRow(ctx, `SELECT (SELECT collection_id::text FROM page_collection_pages WHERE page_id=$1 AND workspace_id=$2)`, id, q.WorkspaceID).Scan(&collection); err != nil {
			return nil, err
		}
		if collection != nil {
			_, access, collectionScopeErr := collectionScope(ctx, tx, q, uuid.MustParse(*collection), write, false)
			if collectionScopeErr != nil {
				return nil, collectionScopeErr
			}
			if write {
				if access == 2 {
					role = max(role, 15)
				}
			}
		}
	} else {
		role, err = pageAPIProject(ctx, tx, q, write)
		if err != nil {
			return nil, err
		}
		var visible bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_pages pp JOIN projects pr ON pr.id=pp.project_id AND pr.workspace_id=pp.workspace_id WHERE pp.page_id=$1 AND pp.project_id=$2 AND pp.workspace_id=$3 AND pp.deleted_at IS NULL AND ($4::boolean OR $5::int=0) AND ($6::int<>5 OR pr.guest_view_all_features OR $4::boolean))`, id, q.ProjectID, q.WorkspaceID, owner, int(row["access"].(float64)), role).Scan(&visible)
		if err != nil {
			return nil, err
		}
		if !visible {
			return nil, projectmutations.ErrForbidden
		}
	}
	if write {
		minimum := 15
		if q.Action == "delete" || q.Action == "archive" {
			minimum = 20
		}
		if !owner && role < minimum {
			return nil, projectmutations.ErrForbidden
		}
	}
	return row, nil
}
func pageAPIItem(ctx context.Context, tx pgx.Tx, q PageAPIQuery, write bool) error {
	role, err := pageAPIProject(ctx, tx, q, write)
	if err != nil {
		return err
	}
	if write && role < 15 {
		return projectmutations.ErrForbidden
	}
	suffix := ""
	if write {
		suffix = " FOR UPDATE"
	}
	var id uuid.UUID
	err = tx.QueryRow(ctx, `SELECT i.id FROM issues i WHERE i.id=$1 AND i.project_id=$2 AND `+djangoapi.IssueVisibleSQL("i", "$3::uuid", "$4::uuid")+suffix, q.ItemID, q.ProjectID, q.ActorID, q.WorkspaceID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = projectmutations.ErrNotFound
	}
	return err
}
