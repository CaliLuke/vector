package pagereads

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func pageAPIVisible(alias string, linked bool) string {
	project := djangoapi.PageVisibleSQL(alias, "$2::uuid", "$1::uuid", "$3::uuid") + ` AND EXISTS(SELECT 1 FROM project_members vm JOIN projects vp ON vp.id=vm.project_id AND vp.workspace_id=vm.workspace_id WHERE vm.member_id=$2 AND vm.project_id=$3 AND vm.workspace_id=$1 AND vm.is_active AND vm.deleted_at IS NULL AND vp.archived_at IS NULL AND (vm.role IN(15,20) OR vm.role=5 AND (vp.guest_view_all_features OR ` + alias + `.owned_by_id=$2)))`
	if linked {
		return `(` + alias + `.is_global AND workspace_page_visible(` + alias + `.id,$2,$1) OR NOT ` + alias + `.is_global AND ` + project + `)`
	}
	return `(CASE WHEN $3::uuid IS NULL THEN workspace_page_visible(` + alias + `.id,$2,$1) ELSE NOT ` + alias + `.is_global AND ` + project + ` END)`
}
func pageAPIArgs(q PageAPIQuery) []any {
	var project any
	if q.ProjectID != uuid.Nil {
		project = q.ProjectID
	}
	return []any{q.WorkspaceID, q.ActorID, project}
}
func pageAPIRows(ctx context.Context, tx pgx.Tx, query string, args ...any) ([]map[string]any, error) {
	return djangoapi.ReadJSONRows(ctx, tx, query, args...)
}
func (r *Repository) ReadPageAPI(ctx context.Context, q PageAPIQuery) (json.RawMessage, error) {
	db, ok := r.database.(mutationreceipts.Database)
	if !ok {
		return nil, projectmutations.ErrInvalid
	}
	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err = pageAPIWorkspace(ctx, tx, q, false); err != nil {
		return nil, err
	}
	var value any
	if q.Tool == "collection" {
		value, err = readCollectionAPI(ctx, tx, q)
	} else {
		switch q.Action {
		case "list":
			if q.ProjectID != uuid.Nil {
				if _, err = pageAPIProject(ctx, tx, q, false); err != nil {
					return nil, err
				}
			}
			value, err = listPagesAPI(ctx, tx, q)
		case "retrieve":
			var row map[string]any
			row, err = pageScope(ctx, tx, q, q.PageID, false, false, false)
			if err == nil {
				value, err = pageAPIProjection(ctx, tx, q, row)
			}
		case "list_workitem_pages":
			if err = pageAPIItem(ctx, tx, q, false); err == nil {
				value, err = itemPageRows(ctx, tx, q, uuid.Nil)
			}
		default:
			err = projectmutations.ErrInvalid
		}
	}
	if err != nil {
		return nil, err
	}
	if q.IncludeRevision {
		var rev int64
		if err = tx.QueryRow(ctx, `SELECT revision FROM page_api_revisions WHERE workspace_id=$1`, q.WorkspaceID).Scan(&rev); err != nil {
			return nil, err
		}
		revision := pageAPIToken(q.WorkspaceID, rev)
		switch v := value.(type) {
		case map[string]any:
			v["revision"] = revision
		case []map[string]any:
			for _, row := range v {
				row["revision"] = revision
			}
		}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return raw, tx.Commit(ctx)
}
func pageAPIProjection(ctx context.Context, tx pgx.Tx, q PageAPIQuery, row map[string]any) (map[string]any, error) {
	out := map[string]any{}
	for _, key := range []string{"id", "name", "description_stripped", "description_html", "description_binary", "created_at", "updated_at", "access", "is_locked", "archived_at", "parent_id"} {
		out[key] = row[key]
	}
	out["owned_by"], out["workspace"], out["description"], out["anchor"] = row["owned_by_id"], row["workspace_id"], row["description_json"], nil
	if binary, ok := row["description_binary"].(string); ok && strings.HasPrefix(binary, `\x`) {
		decoded, err := hex.DecodeString(binary[2:])
		if err != nil {
			return nil, err
		}
		out["description_binary"] = base64.StdEncoding.EncodeToString(decoded)
	}
	var projects []string
	var collection, link *string
	err := tx.QueryRow(ctx, `SELECT COALESCE(ARRAY(SELECT pp.project_id::text FROM project_pages pp JOIN projects p ON p.id=pp.project_id AND p.workspace_id=pp.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=p.workspace_id AND pm.member_id=$3 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20) WHERE pp.page_id=$1 AND pp.workspace_id=$2 AND pp.deleted_at IS NULL AND p.deleted_at IS NULL AND p.archived_at IS NULL ORDER BY pp.project_id),'{}'::text[]),
 (SELECT collection_id::text FROM page_collection_pages WHERE page_id=$1 AND workspace_id=$2),(SELECT id::text FROM page_collection_pages WHERE page_id=$1 AND workspace_id=$2)`, row["id"], q.WorkspaceID, q.ActorID).Scan(&projects, &collection, &link)
	if err != nil {
		return nil, err
	}
	out["projects"], out["collection_id"], out["page_collection_id"] = projects, collection, link
	if parent, ok := row["parent_id"].(string); ok {
		var visible bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pages p WHERE p.id=$4 AND p.workspace_id=$1 AND `+pageAPIVisible("p", false)+`)`, append(pageAPIArgs(q), parent)...).Scan(&visible)
		if err != nil {
			return nil, err
		}
		if !visible {
			out["parent_id"] = nil
		}
	}
	return out, nil
}
func listPagesAPI(ctx context.Context, tx pgx.Tx, q PageAPIQuery) (map[string]any, error) {
	args := pageAPIArgs(q)
	where := `p.workspace_id=$1 AND p.deleted_at IS NULL AND p.archived_at IS NULL AND ` + pageAPIVisible("p", false)
	if q.Tool == "collection" {
		args = append(args, q.CollectionID)
		where += ` AND EXISTS(SELECT 1 FROM page_collection_pages link WHERE link.page_id=p.id AND link.workspace_id=p.workspace_id AND link.collection_id=$4)`
		var parent any
		if q.ParentID != uuid.Nil {
			parent = q.ParentID
		}
		args = append(args, parent, q.Search)
		where += ` AND p.parent_id IS NOT DISTINCT FROM $5::uuid AND strpos(lower(p.name),lower($6))>0`
	}
	limit, page, offset, err := djangoapi.PageWindow(djangoapi.Query{Params: q.Params, Present: q.Present})
	if err != nil {
		return nil, err
	}
	var total int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM pages p WHERE `+where, args...).Scan(&total); err != nil {
		return nil, err
	}
	args = append(args, limit, offset)
	query := `SELECT to_jsonb(p) FROM pages p WHERE ` + where + ` ORDER BY p.sort_order,p.created_at DESC,p.id LIMIT $` + pageParam(len(args)-1) + ` OFFSET $` + pageParam(len(args))
	rows, err := pageAPIRows(ctx, tx, query, args...)
	if err != nil {
		return nil, err
	}
	values := []any{}
	for _, row := range rows {
		value, err := pageAPIProjection(ctx, tx, q, row)
		if err != nil {
			return nil, err
		}
		if q.Tool == "collection" {
			value = map[string]any{"page_collection_id": value["page_collection_id"], "collection_id": value["collection_id"], "parent_id": value["parent_id"], "sort_order": row["sort_order"], "page": value}
		}
		values = append(values, value)
	}
	return djangoapi.Envelope(values, total, limit, page), nil
}
func collectionProjection(ctx context.Context, tx pgx.Tx, q PageAPIQuery, row map[string]any) (map[string]any, error) {
	out := map[string]any{}
	for _, key := range []string{"id", "name", "owned_by_id", "access", "is_default", "is_global", "logo_props", "sort_order", "created_at", "updated_at"} {
		out[key] = row[key]
	}
	out["workspace"], out["created_by"], out["updated_by"] = row["workspace_id"], row["created_by_id"], row["updated_by_id"]
	var access *int
	var pages bool
	err := tx.QueryRow(ctx, `SELECT collection_access($1,$2,$3),EXISTS(SELECT 1 FROM page_collection_pages link JOIN pages p ON p.id=link.page_id AND p.workspace_id=link.workspace_id WHERE link.collection_id=$1 AND link.workspace_id=$3 AND p.archived_at IS NULL AND workspace_page_visible(p.id,$2,$3))`, row["id"], q.ActorID, q.WorkspaceID).Scan(&access, &pages)
	out["current_user_access"], out["has_pages"] = access, pages
	return out, err
}
func readCollectionAPI(ctx context.Context, tx pgx.Tx, q PageAPIQuery) (any, error) {
	if q.Action == "list" {
		rows, err := pageAPIRows(ctx, tx, `SELECT to_jsonb(c) FROM page_collections c WHERE c.workspace_id=$1 AND c.deleted_at IS NULL AND collection_access(c.id,$2,$1) IS NOT NULL ORDER BY c.sort_order,c.name,c.id`, q.WorkspaceID, q.ActorID)
		if err != nil {
			return nil, err
		}
		out := []map[string]any{}
		for _, row := range rows {
			value, err := collectionProjection(ctx, tx, q, row)
			if err != nil {
				return nil, err
			}
			out = append(out, value)
		}
		return out, nil
	}
	row, _, err := collectionScope(ctx, tx, q, q.CollectionID, false, false)
	if err != nil {
		return nil, err
	}
	switch q.Action {
	case "retrieve":
		return collectionProjection(ctx, tx, q, row)
	case "list_pages":
		return listPagesAPI(ctx, tx, q)
	case "list_members":
		return collectionMemberRows(ctx, tx, q, uuid.Nil)
	case "search_pages":
		args := append(pageAPIArgs(q), q.CollectionID, q.Search)
		return pageAPIRows(ctx, tx, `SELECT jsonb_build_object('id',p.id,'name',p.name,'logo_props',p.logo_props) FROM pages p WHERE p.workspace_id=$1 AND $3::uuid IS NULL AND p.parent_id IS NULL AND p.archived_at IS NULL AND workspace_page_visible(p.id,$2,$1) AND strpos(lower(p.name),lower($5))>0 AND NOT EXISTS(SELECT 1 FROM page_collection_pages link WHERE link.page_id=p.id AND link.collection_id=$4) AND (p.owned_by_id=$2 OR EXISTS(SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=$1 AND wm.member_id=$2 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(15,20))) AND NOT EXISTS(SELECT 1 FROM page_collection_pages link WHERE link.page_id=p.id AND collection_access(link.collection_id,$2,$1)<2) ORDER BY p.name,p.id`, args...)
	default:
		return nil, projectmutations.ErrInvalid
	}
}
func collectionMemberRows(ctx context.Context, tx pgx.Tx, q PageAPIQuery, id uuid.UUID) ([]map[string]any, error) {
	return pageAPIRows(ctx, tx, `SELECT jsonb_build_object('id',m.id,'collection',m.collection_id,'member',m.member_id,'access',m.access,'workspace',m.workspace_id,'created_at',m.created_at,'updated_at',m.updated_at,'created_by',m.created_by_id,'updated_by',m.updated_by_id) FROM page_collection_members m JOIN users u ON u.id=m.member_id AND u.is_active WHERE m.collection_id=$1 AND m.workspace_id=$2 AND m.deleted_at IS NULL AND ($3::uuid IS NULL OR m.id=$3) AND EXISTS(SELECT 1 FROM workspace_members wm WHERE wm.member_id=m.member_id AND wm.workspace_id=m.workspace_id AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20)) ORDER BY m.created_at,m.id`, q.CollectionID, q.WorkspaceID, nullablePageUUID(id))
}
func itemPageRows(ctx context.Context, tx pgx.Tx, q PageAPIQuery, id uuid.UUID) ([]map[string]any, error) {
	args := append(pageAPIArgs(q), q.ItemID, nullablePageUUID(id))
	return pageAPIRows(ctx, tx, `SELECT jsonb_build_object('id',link.id,'page',jsonb_build_object('id',p.id,'name',p.name,'created_at',p.created_at,'updated_at',p.updated_at,'created_by',p.created_by_id,'is_global',p.is_global,'logo_props',p.logo_props),'issue',link.issue_id,'project',link.project_id,'workspace',link.workspace_id,'created_at',link.created_at,'updated_at',link.updated_at,'created_by',link.created_by_id) FROM work_item_pages link JOIN pages p ON p.id=link.page_id AND p.workspace_id=link.workspace_id WHERE link.workspace_id=$1 AND link.project_id=$3 AND link.issue_id=$4 AND ($5::uuid IS NULL OR link.id=$5) AND link.deleted_at IS NULL AND p.deleted_at IS NULL AND `+pageAPIVisible("p", true)+` ORDER BY link.created_at,link.id`, args...)
}
func pageParam(n int) string { return strconv.Itoa(n) }
func nullablePageUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
