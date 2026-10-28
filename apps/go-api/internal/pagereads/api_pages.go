package pagereads

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	stduuid "uuid"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) applyPageAPI(ctx context.Context, tx pgx.Tx, in PageAPICommand) (uuid.UUID, any, error) {
	if in.Action == "attach_to_workitem" || in.Action == "detach_from_workitem" {
		return mutateItemPage(ctx, tx, in)
	}
	id := in.PageID
	if in.Action == "set_collection" {
		link, err := filePageBranch(ctx, tx, in.PageAPIQuery, id, in.CollectionID)
		return id, map[string]any{"page_id": id, "collection_id": in.CollectionID, "page_collection_id": link}, err
	}
	bound := *r
	bound.database = tx
	bound.nativeTasks = nil
	bound.publisher = projectmutations.Effects{Tx: tx}
	bound.recentVisits = nil
	var before map[string]any
	var err error
	if in.Action != "create" {
		before, err = pageScope(ctx, tx, in.PageAPIQuery, id, true, false, false)
		if err != nil {
			return id, nil, err
		}
	}
	if in.Action == "update" && (before["is_locked"] == true || before["archived_at"] != nil) {
		return id, nil, projectmutations.ErrInvalid
	}
	if in.ProjectID != uuid.Nil {
		fields := maps.Clone(in.Fields)
		if in.Action == "update" {
			fields["collaboration_revision"] = before["collaboration_revision"]
		}
		delete(fields, "external_source")
		delete(fields, "external_id")
		delete(fields, "archive")
		if in.ParentID != uuid.Nil {
			fields["parent"] = in.ParentID.String()
		}
		raw, marshalErr := json.Marshal(fields)
		if marshalErr != nil {
			return id, nil, marshalErr
		}
		input, marshalErr := djangoinput.Parse(bytes.NewReader(raw), "application/json")
		if marshalErr != nil {
			return id, nil, marshalErr
		}
		operation := in.Action
		if operation == "archive" && in.Fields["archive"] == false {
			operation = "unarchive"
		}
		_, response, marshalErr := bound.Lifecycle(ctx, operation, stduuid.UUID(in.ActorID), in.WorkspaceSlug, stduuid.UUID(in.ProjectID), stduuid.UUID(id), "UTC", input)
		if marshalErr != nil {
			return id, nil, marshalErr
		}
		if in.Action == "create" {
			var row struct {
				ID uuid.UUID `json:"id"`
			}
			if marshalErr = json.Unmarshal(response, &row); marshalErr != nil {
				return id, nil, marshalErr
			}
			id = row.ID
		}
	} else {
		id, err = bound.mutateWorkspacePage(ctx, tx, in, before)
		if err != nil {
			return id, nil, err
		}
	}
	if in.Action == "create" {
		_, err = tx.Exec(ctx, `UPDATE pages SET external_source=$3,external_id=$4 WHERE id=$1 AND workspace_id=$2`, id, in.WorkspaceID, in.Fields["external_source"], in.Fields["external_id"])
		if err != nil {
			return id, nil, err
		}
	}
	if in.Action == "update" && (in.Fields["description_html"] != nil || in.Fields["name"] != nil) {
		if _, err = tx.Exec(ctx, `UPDATE pages SET collaboration_revision=collaboration_revision+1,description_json='{}',description_binary=NULL WHERE id=$1 AND workspace_id=$2`, id, in.WorkspaceID); err != nil {
			return id, nil, err
		}
		if err = bound.admitPageVersion(ctx, id.String(), before, in.ActorID.String()); err != nil {
			return id, nil, err
		}
	}
	if in.Action == "delete" {
		return id, nil, nil
	}
	if in.Action == "archive" {
		return id, map[string]any{"page_id": id, "archived": in.Fields["archive"] != false}, nil
	}
	row, err := pageScope(ctx, tx, in.PageAPIQuery, id, false, false, false)
	if err != nil {
		return id, nil, err
	}
	value, err := pageAPIProjection(ctx, tx, in.PageAPIQuery, row)
	return id, value, err
}
func (r *Repository) mutateWorkspacePage(ctx context.Context, tx pgx.Tx, in PageAPICommand, before map[string]any) (uuid.UUID, error) {
	id := in.PageID
	var err error
	switch in.Action {
	case "create":
		id = uuid.New()
		access := 0
		if value, ok := in.Fields["access"].(int); ok {
			access = value
		}
		color := ""
		if value, ok := in.Fields["color"].(string); ok {
			color = value
		}
		locked := in.Fields["is_locked"] == true
		html := in.Fields["description_html"].(string)
		_, err = tx.Exec(ctx, `INSERT INTO pages(id,workspace_id,name,description_json,description_html,description_stripped,owned_by_id,access,color,parent_id,view_props,logo_props,is_global,sort_order,is_locked,created_by_id,updated_by_id,created_at,updated_at) VALUES($1,$2,$3,'{}',$4,$5,$6,$7,$8,$9,'{"full_width":false}','{}',true,65535,$10,$6,$6,now(),now())`, id, in.WorkspaceID, in.Fields["name"], html, content.StripTags(html), in.ActorID, access, color, nullablePageUUID(in.ParentID), locked)
		if err != nil {
			return id, err
		}
		collection := in.CollectionID
		if in.ParentID != uuid.Nil {
			var parentCollection *uuid.UUID
			if err = tx.QueryRow(ctx, `SELECT (SELECT collection_id FROM page_collection_pages WHERE page_id=$1 AND workspace_id=$2)`, in.ParentID, in.WorkspaceID).Scan(&parentCollection); err != nil {
				return id, err
			}
			if parentCollection != nil {
				collection = *parentCollection
			}
		}
		if collection != uuid.Nil {
			_, err = tx.Exec(ctx, `INSERT INTO page_collection_pages(workspace_id,collection_id,page_id,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$4)`, in.WorkspaceID, collection, id, in.ActorID)
			if err != nil {
				return id, err
			}
		}
		err = r.admitPageTransaction(ctx, html, nil, id.String())
	case "update":
		var stripped any
		if html, ok := in.Fields["description_html"].(string); ok {
			stripped = content.StripTags(html)
		}
		_, err = tx.Exec(ctx, `UPDATE pages SET name=COALESCE($3,name),description_html=COALESCE($4,description_html),description_stripped=COALESCE($5,description_stripped),updated_at=now(),updated_by_id=$6 WHERE id=$1 AND workspace_id=$2`, id, in.WorkspaceID, in.Fields["name"], in.Fields["description_html"], stripped, in.ActorID)
		if err == nil && in.Fields["description_html"] != nil {
			err = r.admitPageTransaction(ctx, in.Fields["description_html"], before["description_html"], id.String())
		}
	case "archive":
		err = archiveWorkspacePage(ctx, tx, in)
	case "delete":
		if before["archived_at"] == nil {
			return id, projectmutations.ErrInvalid
		}
		_, err = tx.Exec(ctx, `UPDATE pages SET parent_id=NULL,updated_at=now(),updated_by_id=$3 WHERE parent_id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND is_global`, id, in.WorkspaceID, in.ActorID)
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE pages SET deleted_at=now(),updated_at=now(),updated_by_id=$3 WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL`, id, in.WorkspaceID, in.ActorID)
		}
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE user_favorites SET deleted_at=now() WHERE entity_type='page' AND entity_identifier=$1 AND workspace_id=$2 AND deleted_at IS NULL`, id, in.WorkspaceID)
		}
		if err == nil {
			_, err = tx.Exec(ctx, `DELETE FROM user_recent_visits WHERE entity_name='page' AND entity_identifier=$1 AND workspace_id=$2`, id, in.WorkspaceID)
		}
		if err == nil {
			err = r.publisher.SoftDelete(ctx, "db", "page", id.String())
		}
	default:
		err = projectmutations.ErrInvalid
	}
	return id, err
}
func archiveWorkspacePage(ctx context.Context, tx pgx.Tx, in PageAPICommand) error {
	archive := in.Fields["archive"] != false
	if !archive {
		if _, err := tx.Exec(ctx, `UPDATE pages p SET parent_id=NULL WHERE p.id=$1 AND p.workspace_id=$2 AND EXISTS(SELECT 1 FROM pages parent WHERE parent.id=p.parent_id AND parent.workspace_id=$2 AND parent.archived_at IS NOT NULL)`, in.PageID, in.WorkspaceID); err != nil {
			return err
		}
	}
	_, err := tx.Exec(ctx, `WITH RECURSIVE tree(id) AS (SELECT $1::uuid UNION SELECT p.id FROM pages p JOIN tree t ON t.id=p.parent_id WHERE p.workspace_id=$2 AND p.is_global AND p.deleted_at IS NULL) UPDATE pages SET archived_at=CASE WHEN $3 THEN CURRENT_DATE ELSE NULL END,updated_at=now(),updated_by_id=$4 WHERE id IN(SELECT id FROM tree) AND workspace_id=$2`, in.PageID, in.WorkspaceID, archive, in.ActorID)
	if err == nil && archive {
		_, err = tx.Exec(ctx, `UPDATE user_favorites SET deleted_at=now() WHERE workspace_id=$2 AND entity_type='page' AND entity_identifier=$1 AND deleted_at IS NULL`, in.PageID, in.WorkspaceID)
	}
	return err
}
func mutateItemPage(ctx context.Context, tx pgx.Tx, in PageAPICommand) (uuid.UUID, any, error) {
	id := in.LinkID
	var err error
	if in.Action == "detach_from_workitem" {
		tag, execErr := tx.Exec(ctx, `UPDATE work_item_pages SET deleted_at=now(),updated_at=now(),updated_by_id=$5 WHERE id=$1 AND issue_id=$2 AND project_id=$3 AND workspace_id=$4 AND deleted_at IS NULL`, id, in.ItemID, in.ProjectID, in.WorkspaceID, in.ActorID)
		if execErr == nil && tag.RowsAffected() == 0 {
			execErr = projectmutations.ErrNotFound
		}
		return id, nil, execErr
	}
	err = tx.QueryRow(ctx, `INSERT INTO work_item_pages(workspace_id,project_id,issue_id,page_id,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$5,$5) ON CONFLICT(issue_id,page_id) WHERE deleted_at IS NULL DO UPDATE SET updated_at=now(),updated_by_id=EXCLUDED.updated_by_id RETURNING id`, in.WorkspaceID, in.ProjectID, in.ItemID, in.PageID, in.ActorID).Scan(&id)
	if err != nil {
		return id, nil, err
	}
	rows, err := itemPageRows(ctx, tx, in.PageAPIQuery, id)
	if err != nil {
		return id, nil, err
	}
	if len(rows) != 1 {
		return id, nil, projectmutations.ErrNotFound
	}
	return id, rows[0], nil
}
