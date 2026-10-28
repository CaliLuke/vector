package pagereads

import (
	"context"
	"errors"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Filing a root moves its entire hierarchy without changing page visibility.
func pageBranchForMove(ctx context.Context, tx pgx.Tx, q PageAPIQuery, id uuid.UUID) ([]uuid.UUID, error) {
	root, err := pageScope(ctx, tx, q, id, true, false, false)
	if err != nil {
		return nil, err
	}
	if root["parent_id"] != nil || root["is_global"] != true {
		return nil, projectmutations.ErrInvalid
	}
	rows, err := tx.Query(ctx, `WITH RECURSIVE tree(id) AS (SELECT $1::uuid UNION SELECT p.id FROM pages p JOIN tree t ON p.parent_id=t.id WHERE p.workspace_id=$2 AND p.deleted_at IS NULL) SELECT p.id FROM pages p JOIN tree t ON t.id=p.id ORDER BY p.id FOR UPDATE OF p`, id, q.WorkspaceID)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, err
	}
	for _, child := range ids {
		row, err := pageScope(ctx, tx, q, child, true, false, false)
		if err != nil {
			return nil, err
		}
		if row["archived_at"] != nil || row["is_locked"] == true {
			return nil, projectmutations.ErrInvalid
		}
	}
	return ids, nil
}
func filePageBranch(ctx context.Context, tx pgx.Tx, q PageAPIQuery, root uuid.UUID, collection uuid.UUID) (uuid.UUID, error) {
	ids, err := pageBranchForMove(ctx, tx, q, root)
	if err != nil {
		return uuid.Nil, err
	}
	var rootLink uuid.UUID
	for _, page := range ids {
		if collection == uuid.Nil {
			_, err = tx.Exec(ctx, `DELETE FROM page_collection_pages WHERE page_id=$1 AND workspace_id=$2`, page, q.WorkspaceID)
		} else {
			var link uuid.UUID
			err = tx.QueryRow(ctx, `INSERT INTO page_collection_pages(workspace_id,collection_id,page_id,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$4) ON CONFLICT(page_id) DO UPDATE SET collection_id=EXCLUDED.collection_id,updated_at=now(),updated_by_id=EXCLUDED.updated_by_id RETURNING id`, q.WorkspaceID, collection, page, q.ActorID).Scan(&link)
			if page == root {
				rootLink = link
			}
		}
		if err != nil {
			return uuid.Nil, err
		}
	}
	return rootLink, nil
}
func collectionPageRows(ctx context.Context, tx pgx.Tx, q PageAPIQuery, ids []uuid.UUID) ([]map[string]any, error) {
	return pageAPIRows(ctx, tx, `SELECT jsonb_build_object('id',id,'collection',collection_id,'page',page_id,'workspace',workspace_id,'sort_order',sort_order,'created_at',created_at,'updated_at',updated_at,'created_by',created_by_id,'updated_by',updated_by_id) FROM page_collection_pages WHERE collection_id=$1 AND workspace_id=$2 AND page_id=ANY($3::uuid[]) ORDER BY sort_order,id`, q.CollectionID, q.WorkspaceID, ids)
}
func applyCollectionAPI(ctx context.Context, tx pgx.Tx, in PageAPICommand) (uuid.UUID, any, error) {
	id := in.CollectionID
	var err error
	switch in.Action {
	case "create":
		access := 0
		if value, ok := in.Fields["access"].(int); ok {
			access = value
		}
		id = uuid.New()
		in.CollectionID = id
		_, err = tx.Exec(ctx, `INSERT INTO page_collections(id,workspace_id,name,owned_by_id,access,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$5,$4,$4)`, id, in.WorkspaceID, in.Fields["name"], in.ActorID, access)
		if err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO page_collection_members(workspace_id,collection_id,member_id,access,created_by_id,updated_by_id) VALUES($1,$2,$3,2,$3,$3)`, in.WorkspaceID, id, in.ActorID)
		}
	case "update":
		_, err = tx.Exec(ctx, `UPDATE page_collections SET name=COALESCE($3,name),sort_order=COALESCE($4,sort_order),updated_at=now(),updated_by_id=$5 WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL`, id, in.WorkspaceID, in.Fields["name"], in.Fields["sort_order"], in.ActorID)
	case "delete":
		archive := true
		if value, ok := in.Fields["archive_pages"].(bool); ok {
			archive = value
		}
		if archive {
			_, err = tx.Exec(ctx, `UPDATE pages SET archived_at=CURRENT_DATE,updated_at=now(),updated_by_id=$3 WHERE workspace_id=$2 AND deleted_at IS NULL AND id IN(SELECT page_id FROM page_collection_pages WHERE collection_id=$1 AND workspace_id=$2)`, id, in.WorkspaceID, in.ActorID)
			if err == nil {
				_, err = tx.Exec(ctx, `UPDATE user_favorites SET deleted_at=now() WHERE workspace_id=$2 AND entity_type='page' AND deleted_at IS NULL AND entity_identifier IN(SELECT page_id FROM page_collection_pages WHERE collection_id=$1 AND workspace_id=$2)`, id, in.WorkspaceID)
			}
		}
		if err == nil {
			_, err = tx.Exec(ctx, `DELETE FROM page_collection_pages WHERE collection_id=$1 AND workspace_id=$2`, id, in.WorkspaceID)
		}
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE page_collections SET deleted_at=now(),updated_at=now(),updated_by_id=$3 WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL`, id, in.WorkspaceID, in.ActorID)
		}
		return id, nil, err
	case "add_pages":
		for _, page := range in.PageIDs {
			if _, err = filePageBranch(ctx, tx, in.PageAPIQuery, page, id); err != nil {
				return id, nil, err
			}
		}
		rows, collectionPageRowsErr := collectionPageRows(ctx, tx, in.PageAPIQuery, in.PageIDs)
		return id, rows, collectionPageRowsErr
	case "remove_page":
		var page uuid.UUID
		err = tx.QueryRow(ctx, `SELECT page_id FROM page_collection_pages WHERE id=$1 AND collection_id=$2 AND workspace_id=$3`, in.LinkID, id, in.WorkspaceID).Scan(&page)
		if errors.Is(err, pgx.ErrNoRows) {
			err = projectmutations.ErrNotFound
		}
		if err == nil {
			_, err = filePageBranch(ctx, tx, in.PageAPIQuery, page, uuid.Nil)
		}
		return in.LinkID, nil, err
	case "add_member", "update_member", "remove_member":
		return mutateCollectionMember(ctx, tx, in)
	default:
		return id, nil, projectmutations.ErrInvalid
	}
	if err != nil {
		return id, nil, err
	}
	row, err := pageAPIRow(ctx, tx, `SELECT to_jsonb(c) FROM page_collections c WHERE id=$1 AND workspace_id=$2`, id, in.WorkspaceID)
	if err != nil {
		return id, nil, err
	}
	value, err := collectionProjection(ctx, tx, in.PageAPIQuery, row)
	return id, value, err
}
func mutateCollectionMember(ctx context.Context, tx pgx.Tx, in PageAPICommand) (uuid.UUID, any, error) {
	id := in.MemberID
	var err error
	if in.Action == "add_member" {
		var owner uuid.UUID
		if err = tx.QueryRow(ctx, `SELECT owned_by_id FROM page_collections WHERE id=$1 AND workspace_id=$2`, in.CollectionID, in.WorkspaceID).Scan(&owner); err != nil {
			return id, nil, err
		}
		if owner == in.UserID && in.Fields["member_access"] != 2 {
			return id, nil, projectmutations.ErrInvalid
		}
		err = tx.QueryRow(ctx, `INSERT INTO page_collection_members(workspace_id,collection_id,member_id,access,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$5,$5) ON CONFLICT(collection_id,member_id) WHERE deleted_at IS NULL DO UPDATE SET access=EXCLUDED.access,updated_at=now(),updated_by_id=EXCLUDED.updated_by_id RETURNING id`, in.WorkspaceID, in.CollectionID, in.UserID, in.Fields["member_access"], in.ActorID).Scan(&id)
	} else {
		var owner bool
		err = tx.QueryRow(ctx, `SELECT m.member_id=c.owned_by_id FROM page_collection_members m JOIN page_collections c ON c.id=m.collection_id AND c.workspace_id=m.workspace_id WHERE m.id=$1 AND m.collection_id=$2 AND m.workspace_id=$3 AND m.deleted_at IS NULL FOR UPDATE OF m`, id, in.CollectionID, in.WorkspaceID).Scan(&owner)
		if errors.Is(err, pgx.ErrNoRows) {
			err = projectmutations.ErrNotFound
		}
		if err != nil {
			return id, nil, err
		}
		if owner && (in.Action == "remove_member" || in.Fields["member_access"] != 2) {
			return id, nil, projectmutations.ErrInvalid
		}
		if in.Action == "remove_member" {
			_, err = tx.Exec(ctx, `UPDATE page_collection_members SET deleted_at=now(),updated_at=now(),updated_by_id=$2 WHERE id=$1`, id, in.ActorID)
			return id, nil, err
		}
		_, err = tx.Exec(ctx, `UPDATE page_collection_members SET access=$2,updated_at=now(),updated_by_id=$3 WHERE id=$1`, id, in.Fields["member_access"], in.ActorID)
	}
	if err != nil {
		return id, nil, err
	}
	rows, err := collectionMemberRows(ctx, tx, in.PageAPIQuery, id)
	if err != nil {
		return id, nil, err
	}
	if len(rows) != 1 {
		return id, nil, projectmutations.ErrNotFound
	}
	return id, rows[0], nil
}
