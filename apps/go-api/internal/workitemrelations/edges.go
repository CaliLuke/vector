package workitemrelations

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func inverse(kind string) string {
	switch kind {
	case "blocking":
		return "blocked_by"
	case "blocked_by":
		return "blocking"
	case "start_before":
		return "start_after"
	case "start_after":
		return "start_before"
	case "finish_before":
		return "finish_after"
	case "finish_after":
		return "finish_before"
	}
	return kind
}
func mutateEdges(ctx context.Context, tx pgx.Tx, in Command) (json.RawMessage, error) {
	if in.Action == "delete" {
		return deleteEdges(ctx, tx, in)
	}
	custom := in.RelationType == ""
	table := "issue_relations"
	label, opposite := in.RelationType, inverse(in.RelationType)
	reverse := in.RelationType == "blocking" || in.RelationType == "start_after" || in.RelationType == "finish_after"
	symmetric := false
	if custom {
		table = "work_item_custom_relations"
		var outward, inward string
		err := tx.QueryRow(ctx, `SELECT outward,inward FROM work_item_relation_definitions WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND is_active FOR SHARE`, in.DefinitionID, in.WorkspaceID).Scan(&outward, &inward)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, projectmutations.ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		if in.DefinitionLabel != outward && in.DefinitionLabel != inward {
			return nil, projectmutations.ErrInvalid
		}
		label = in.DefinitionLabel
		reverse = label != outward
		opposite = inward
		if reverse {
			opposite = outward
		}
		symmetric = outward == inward
	}
	out := []json.RawMessage{}
	for _, target := range in.Targets {
		left, right := in.IssueID, target
		if reverse || symmetric && strings.Compare(left.String(), right.String()) > 0 {
			left, right = right, left
		}
		id := uuid.New()
		var existing uuid.UUID
		var err error
		if custom {
			err = tx.QueryRow(ctx, `SELECT id FROM work_item_custom_relations WHERE definition_id=$1 AND issue_id=$2 AND related_issue_id=$3 AND deleted_at IS NULL`, in.DefinitionID, left, right).Scan(&existing)
		} else {
			kind := label
			if reverse {
				kind = opposite
			}
			err = tx.QueryRow(ctx, `SELECT id FROM issue_relations WHERE issue_id=$1 AND related_issue_id=$2 AND relation_type=$3 AND workspace_id=$4 AND deleted_at IS NULL`, left, right, kind, in.WorkspaceID).Scan(&existing)
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		created := errors.Is(err, pgx.ErrNoRows)
		if !created {
			id = existing
		} else if custom {
			_, err = tx.Exec(ctx, `INSERT INTO work_item_custom_relations(id,workspace_id,definition_id,issue_id,related_issue_id,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$5,$6,$6)`, id, in.WorkspaceID, in.DefinitionID, left, right, in.ActorID)
		} else {
			kind := label
			if reverse {
				kind = opposite
			}
			_, err = tx.Exec(ctx, `INSERT INTO issue_relations(id,workspace_id,project_id,issue_id,related_issue_id,relation_type,created_at,updated_at,created_by_id,updated_by_id) SELECT $1,$2,i.project_id,$3,$4,$5,now(),now(),$6,$6 FROM issues i WHERE i.id=$3`, id, in.WorkspaceID, left, right, kind, in.ActorID)
		}
		if err != nil {
			return nil, err
		}
		if created {
			if err = edgeActivity(ctx, tx, in, target, label, opposite, "created"); err != nil {
				return nil, err
			}
		}
		var record json.RawMessage
		if err = tx.QueryRow(ctx, `SELECT `+targetProjection+`||jsonb_build_object('relation_type',$3::text) FROM `+table+` r JOIN issues i ON i.id=$2 WHERE r.id=$1`, id, target, label).Scan(&record); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return json.Marshal(out)
}
func deleteEdges(ctx context.Context, tx pgx.Tx, in Command) (json.RawMessage, error) {
	table := "work_item_custom_relations"
	query := `SELECT r.id,r.issue_id,d.outward,d.inward FROM work_item_custom_relations r JOIN work_item_relation_definitions d ON d.id=r.definition_id AND d.workspace_id=r.workspace_id WHERE r.workspace_id=$1 AND ((r.issue_id=$2 AND r.related_issue_id=$3) OR (r.issue_id=$3 AND r.related_issue_id=$2)) AND r.deleted_at IS NULL ORDER BY r.id FOR UPDATE OF r`
	if in.IsDependency {
		table = "issue_relations"
		query = `SELECT r.id,r.issue_id,r.relation_type,'' FROM issue_relations r WHERE r.workspace_id=$1 AND ((r.issue_id=$2 AND r.related_issue_id=$3) OR (r.issue_id=$3 AND r.related_issue_id=$2)) AND r.deleted_at IS NULL AND r.relation_type IN('blocking','blocked_by','start_before','start_after','finish_before','finish_after') ORDER BY r.id FOR UPDATE OF r`
	}
	rows, err := tx.Query(ctx, query, in.WorkspaceID, in.IssueID, in.RelatedID)
	if err != nil {
		return nil, err
	}
	type edge struct {
		id, left        uuid.UUID
		outward, inward string
	}
	selected := []edge{}
	for rows.Next() {
		var e edge
		if err = rows.Scan(&e.id, &e.left, &e.outward, &e.inward); err != nil {
			rows.Close()
			return nil, err
		}
		selected = append(selected, e)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	for _, e := range selected {
		if _, err = tx.Exec(ctx, `UPDATE `+table+` SET deleted_at=now(),updated_at=now(),updated_by_id=$2 WHERE id=$1`, e.id, in.ActorID); err != nil {
			return nil, err
		}
		if in.IsDependency {
			e.inward = inverse(e.outward)
		}
		if e.left != in.IssueID {
			e.outward, e.inward = e.inward, e.outward
		}
		if err = edgeActivity(ctx, tx, in, in.RelatedID, e.outward, e.inward, "deleted"); err != nil {
			return nil, err
		}
	}
	return json.RawMessage("null"), nil
}
func edgeActivity(ctx context.Context, tx pgx.Tx, in Command, target uuid.UUID, label, opposite, verb string) error {
	for _, side := range []struct {
		item, other uuid.UUID
		label       string
	}{{in.IssueID, target, label}, {target, in.IssueID, opposite}} {
		var project uuid.UUID
		var identifier string
		if err := tx.QueryRow(ctx, `SELECT i.project_id FROM issues i WHERE i.id=$1 AND i.workspace_id=$2`, side.item, in.WorkspaceID).Scan(&project); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `SELECT p.identifier||'-'||i.sequence_id FROM issues i JOIN projects p ON p.id=i.project_id AND p.workspace_id=i.workspace_id WHERE i.id=$1 AND i.workspace_id=$2`, side.other, in.WorkspaceID).Scan(&identifier); err != nil {
			return err
		}
		data := map[string]any{"type": "workitem_relation.activity." + verb, "workspace_id": in.WorkspaceID.String(), "project_id": project.String(), "issue_id": side.item.String(), "actor_id": in.ActorID.String(), "epoch": time.Now().Unix(), "notification": false, "requested_data": map[string]any{"related_id": side.other.String(), "label": side.label, "identifier": identifier}}
		if err := (projectmutations.Effects{Tx: tx}).Task(ctx, "plane.bgtasks.issue_activities_task.issue_activity", nil, data); err != nil {
			return err
		}
	}
	return nil
}
