package releases

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *Repository) Mutate(ctx context.Context, in Command) (projectmutations.Result, error) {
	if err := normalize(&in); err != nil {
		return projectmutations.Result{}, err
	}
	normalized, err := json.Marshal(map[string]any{"id": in.ID, "release_id": in.ReleaseID, "fields": in.Fields, "add": in.Add, "remove": in.Remove, "expected_revision": in.ExpectedRevision})
	if err != nil {
		return projectmutations.Result{}, err
	}
	receipt, err := mutationreceipts.Execute(ctx, r.db, mutationreceipts.Command{ActorID: in.ActorID, WorkspaceID: in.WorkspaceID, Tool: in.Tool, Action: in.Action, RequestID: in.RequestID, NormalizedInput: normalized}, mutationreceipts.Operation{
		Authorize: func(ctx context.Context, tx pgx.Tx) error { return authorize(ctx, tx, in) },
		Apply: func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
			var revision int64
			if scanErr := tx.QueryRow(ctx, `SELECT revision FROM release_revisions WHERE workspace_id=$1 FOR UPDATE`, in.WorkspaceID).Scan(&revision); scanErr != nil {
				return nil, scanErr
			}
			if in.ExpectedRevision != nil && *in.ExpectedRevision != token(in.WorkspaceID, revision) {
				return nil, projectmutations.ErrRevision
			}
			id := in.ID
			if in.Action == "create" {
				id = uuid.New()
			}
			if in.Action == "attach" || in.Action == "detach" {
				id = in.ReleaseID
			}
			var before, after any
			if in.Action != "create" {
				q := in.Query
				if in.Action == "attach" || in.Action == "detach" {
					q.Tool = "release"
				}
				row, recordErr := record(ctx, tx, q, id)
				if recordErr != nil {
					return nil, recordErr
				}
				before = row
			}
			var changed bool
			var operationErr error
			switch in.Action {
			case "manage_workitems":
				changed, operationErr = manageItems(ctx, tx, in, id, before.(map[string]any)["name"].(string))
			case "attach", "detach":
				changed, operationErr = manageLabels(ctx, tx, in)
			case "update_changelog":
				before, operationErr = changelog(ctx, tx, in.Query, id)
				if operationErr == nil {
					changed, operationErr = writeRow(ctx, tx, "release_changelogs", "release_id", id, in, false, in.Fields)
				}
				if operationErr == nil {
					after, operationErr = changelog(ctx, tx, in.Query, id)
				}
			case "delete":
				changed, operationErr = deleteResource(ctx, tx, in, id, before)
			default:
				changed, operationErr = writeRow(ctx, tx, table(in.Tool), "id", id, in, in.Action == "create", in.Fields)
				if operationErr == nil && in.Action == "create" && in.Tool == "release" {
					_, operationErr = tx.Exec(ctx, `INSERT INTO release_changelogs(release_id,workspace_id,created_by_id,updated_by_id) VALUES($1,$2,$3,$3)`, id, in.WorkspaceID, in.ActorID)
				}
				if operationErr == nil {
					after, operationErr = record(ctx, tx, in.Query, id)
				}
			}
			if operationErr != nil {
				return nil, operationErr
			}
			if changed {
				old, _ := json.Marshal(before)
				current, _ := json.Marshal(after)
				if _, operationErr = tx.Exec(ctx, `INSERT INTO release_events(workspace_id,actor_id,resource_id,resource_kind,action,before_value,after_value) VALUES($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb)`, in.WorkspaceID, in.ActorID, id, in.Tool, in.Action, old, current); operationErr != nil {
					return nil, operationErr
				}
			}
			if operationErr = tx.QueryRow(ctx, `SELECT revision FROM release_revisions WHERE workspace_id=$1`, in.WorkspaceID).Scan(&revision); operationErr != nil {
				return nil, operationErr
			}
			raw, operationErr := json.Marshal(after)
			if operationErr != nil {
				return nil, operationErr
			}
			return json.Marshal(projectmutations.Result{ID: id.String(), Revision: token(in.WorkspaceID, revision), Record: raw})
		},
	})
	if err != nil {
		var failure *pgconn.PgError
		if errors.As(err, &failure) && failure.Code == "23505" {
			err = projectmutations.ErrInvalid
		}
		return projectmutations.Result{}, err
	}
	var result projectmutations.Result
	if err = json.Unmarshal(receipt.Value, &result); err != nil {
		return result, err
	}
	if in.RequestID != nil {
		result.RequestID, result.ReplayExpiresAt, result.Replayed = in.RequestID, receipt.ReplayExpiresAt, &receipt.Replayed
	}
	return result, nil
}

// Column names come exclusively from the validated native field set.
func writeRow(ctx context.Context, tx pgx.Tx, name, key string, id uuid.UUID, in Command, create bool, values map[string]any) (bool, error) {
	fields := maps.Clone(values)
	fields["updated_by_id"] = in.ActorID.String()
	if create {
		fields["id"] = id.String()
		fields["workspace_id"] = in.WorkspaceID.String()
		fields["created_by_id"] = in.ActorID.String()
	}
	keys := slices.Sorted(maps.Keys(fields))
	columns, selected := []string{}, []string{}
	for _, key := range keys {
		quoted := pgx.Identifier{key}.Sanitize()
		columns = append(columns, quoted)
		selected = append(selected, "x."+quoted)
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return false, err
	}
	query := ""
	args := []any{raw}
	quoted := pgx.Identifier{name}.Sanitize()
	if create {
		query = `INSERT INTO ` + quoted + ` (` + strings.Join(columns, ",") + `) SELECT ` + strings.Join(selected, ",") + ` FROM jsonb_populate_record(NULL::` + quoted + `,$1::jsonb) x`
	} else {
		current := []string{}
		for _, column := range columns {
			current = append(current, "t."+column)
		}
		query = `UPDATE ` + quoted + ` t SET (` + strings.Join(columns, ",") + `)=ROW(` + strings.Join(selected, ",") + `),updated_at=clock_timestamp() FROM jsonb_populate_record(NULL::` + quoted + `,$1::jsonb) x WHERE t.` + pgx.Identifier{key}.Sanitize() + `=$2 AND t.workspace_id=$3 AND ROW(` + strings.Join(current, ",") + `) IS DISTINCT FROM ROW(` + strings.Join(selected, ",") + `)`
		args = append(args, id, in.WorkspaceID)
	}
	command, err := tx.Exec(ctx, query, args...)
	return command.RowsAffected() > 0, err
}
func deleteResource(ctx context.Context, tx pgx.Tx, in Command, id uuid.UUID, before any) (bool, error) {
	switch in.Tool {
	case "release":
		var used bool
		err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM (SELECT custom_properties FROM issues WHERE workspace_id=$1 UNION ALL SELECT custom_properties FROM draft_issues WHERE workspace_id=$1) item JOIN work_item_properties p ON p.workspace_id=$1 AND p.definition->>'kind'='Release picker' WHERE item.custom_properties->p.id::text=to_jsonb($2::text) OR item.custom_properties->p.id::text @> jsonb_build_array($2::text))`, in.WorkspaceID, id.String()).Scan(&used)
		if err != nil {
			return false, err
		}
		if used {
			return false, projectmutations.ErrInvalid
		}
		if _, err = removeAllItems(ctx, tx, in, id, before.(map[string]any)["name"].(string)); err != nil {
			return false, err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM release_item_labels WHERE release_id=$1 AND workspace_id=$2`, id, in.WorkspaceID); err != nil {
			return false, err
		}
	case "release_label":
		if _, err := tx.Exec(ctx, `DELETE FROM release_item_labels WHERE label_id=$1 AND workspace_id=$2`, id, in.WorkspaceID); err != nil {
			return false, err
		}
	case "release_tag":
		if _, err := tx.Exec(ctx, `UPDATE work_item_releases SET tag_id=NULL,updated_at=clock_timestamp(),updated_by_id=$3 WHERE tag_id=$1 AND workspace_id=$2`, id, in.WorkspaceID, in.ActorID); err != nil {
			return false, err
		}
	}
	command, err := tx.Exec(ctx, `UPDATE `+table(in.Tool)+` SET deleted_at=clock_timestamp(),updated_at=clock_timestamp(),updated_by_id=$3 WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL`, id, in.WorkspaceID, in.ActorID)
	return command.RowsAffected() > 0, err
}
