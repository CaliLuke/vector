package workitems

import (
	"context"
	"encoding/json"
	"maps"
	"reflect"
	"strings"

	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// These closed column lists are the model boundary, never request-derived SQL.
var collaborationColumns = map[string]string{
	"issue_comments":         "created_at updated_at deleted_at created_by_id updated_by_id project_id workspace_id comment_stripped comment_json comment_html attachments access external_source external_id edited_at description_id issue_id actor_id parent_id",
	"issue_links":            "created_at updated_at deleted_at created_by_id updated_by_id project_id workspace_id title url metadata issue_id",
	"issue_reactions":        "created_at updated_at deleted_at created_by_id updated_by_id project_id workspace_id actor_id issue_id reaction",
	"comment_reactions":      "created_at updated_at deleted_at created_by_id updated_by_id project_id workspace_id actor_id comment_id reaction",
	"issue_subscribers":      "created_at updated_at deleted_at created_by_id updated_by_id project_id workspace_id issue_id subscriber_id",
	"descriptions":           "created_at updated_at deleted_at created_by_id updated_by_id project_id workspace_id description_html description_json description_stripped description_binary",
	"project_public_members": "created_at updated_at deleted_at created_by_id updated_by_id project_id workspace_id member_id",
}

func collaborationDefault(table string, q Request, workspace any) map[string]any {
	row := map[string]any{"id": uuid.NewString(), "project_id": q.ProjectID, "workspace_id": workspace, "created_at": nil, "updated_at": nil, "created_by_id": nil, "updated_by_id": nil, "deleted_at": nil}
	switch table {
	case "issue_comments":
		maps.Copy(row, map[string]any{"comment_stripped": "", "comment_json": map[string]any{}, "comment_html": "<p></p>", "attachments": []any{}, "access": "INTERNAL", "external_source": nil, "external_id": nil, "edited_at": nil, "description_id": nil, "issue_id": q.IssueID, "actor_id": q.ActorID, "parent_id": nil})
	case "issue_links":
		maps.Copy(row, map[string]any{"title": nil, "url": "", "metadata": map[string]any{}, "issue_id": q.IssueID})
	case "issue_reactions":
		row["actor_id"], row["issue_id"], row["reaction"] = q.ActorID, q.IssueID, ""
	case "comment_reactions":
		row["actor_id"], row["comment_id"], row["reaction"] = q.ActorID, q.Params["comment_id"], ""
	case "issue_subscribers":
		row["issue_id"], row["subscriber_id"] = q.IssueID, q.ActorID
	case "project_public_members":
		row["member_id"] = q.ActorID
	}
	return row
}

func writeCollaborationRecord(ctx context.Context, tx pgx.Tx, table string, row map[string]any, create bool) error {
	columns, ok := collaborationColumns[table]
	if !ok {
		panic("unknown collaboration model")
	}
	record := map[string]any{"id": row["id"]}
	for _, col := range strings.Fields(columns) {
		record[col] = row[col]
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return err
	}
	statement := `INSERT INTO ` + table + ` SELECT (jsonb_populate_record(NULL::` + table + `,$1::jsonb)).*`
	if !create {
		cols := strings.Fields(columns)
		values := make([]string, len(cols))
		for i, col := range cols {
			values[i] = "v." + col
		}
		statement = `UPDATE ` + table + ` t SET (` + strings.Join(cols, ",") + `)=(` + strings.Join(values, ",") + `) FROM jsonb_populate_record(NULL::` + table + `,$1::jsonb) v WHERE t.id=v.id`
	}
	_, err = tx.Exec(ctx, statement, encoded)
	return err
}

// Comment and Description persist in the same transaction as IssueComment.save.
// The second description-only save changes the serializer instance's updater,
// while leaving the first save's persisted updater unchanged.
func (r *Repository) saveCollaboration(ctx context.Context, q Request, table string, row, old map[string]any) (map[string]any, error) {
	now := timestamp(r.now())
	row["updated_at"] = now
	if old == nil {
		row["created_at"], row["created_by_id"], row["updated_by_id"] = now, q.ActorID, nil
	} else {
		row["updated_by_id"] = q.ActorID
	}
	if table == "issue_comments" {
		row["comment_stripped"] = content.StripTags(row["comment_html"].(string))
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err = writeCollaborationRecord(ctx, tx, table, row, old == nil); err != nil {
		return nil, err
	}
	if table == "issue_comments" {
		if old == nil || row["description_id"] == nil {
			description := collaborationDefault("descriptions", q, row["workspace_id"])
			maps.Copy(description, map[string]any{"description_html": row["comment_html"], "description_json": row["comment_json"], "description_binary": nil, "description_stripped": stripTags(row["comment_html"].(string)), "created_at": now, "updated_at": now, "created_by_id": q.ActorID, "updated_by_id": nil})
			if err = writeCollaborationRecord(ctx, tx, "descriptions", description, true); err != nil {
				return nil, err
			}
			row["description_id"] = description["id"]
			if _, err = tx.Exec(ctx, `UPDATE issue_comments SET description_id=$2 WHERE id=$1`, row["id"], description["id"]); err != nil {
				return nil, err
			}
			row["updated_by_id"] = q.ActorID
		} else {
			changes := map[string]any{}
			for _, pair := range [][2]string{{"comment_html", "description_html"}, {"comment_json", "description_json"}, {"comment_stripped", "description_stripped"}} {
				if !reflect.DeepEqual(row[pair[0]], old[pair[0]]) {
					changes[pair[1]] = row[pair[0]]
				}
			}
			if len(changes) > 0 {
				changes["updated_at"], changes["updated_by_id"] = now, q.ActorID
				encoded, e := json.Marshal(changes)
				if e != nil {
					return nil, e
				}
				// Update only tracked fields, preserving unrelated description content.
				if _, err = tx.Exec(ctx, `UPDATE descriptions d SET description_html=CASE WHEN $2::jsonb ? 'description_html' THEN $2::jsonb->>'description_html' ELSE d.description_html END, description_json=CASE WHEN $2::jsonb ? 'description_json' THEN $2::jsonb->'description_json' ELSE d.description_json END, description_stripped=CASE WHEN $2::jsonb ? 'description_stripped' THEN $2::jsonb->>'description_stripped' ELSE d.description_stripped END, updated_at=$3,updated_by_id=$4 WHERE id=$1`, row["description_id"], encoded, now, q.ActorID); err != nil {
					return nil, err
				}
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return row, nil
}
func (r *Repository) deleteCollaboration(ctx context.Context, q Request, table string, row map[string]any) error {
	old := maps.Clone(row)
	row["deleted_at"] = timestamp(r.now())
	if _, err := r.saveCollaboration(ctx, q, table, row, old); err != nil {
		return err
	}
	model := map[string]string{"issue_comments": "issuecomment", "issue_links": "issuelink", "issue_reactions": "issuereaction", "comment_reactions": "commentreaction", "issue_subscribers": "issuesubscriber"}[table]
	return r.publisher.SoftDelete(ctx, "db", model, row["id"].(string))
}
