package customers

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *Repository) Mutate(ctx context.Context, in Command) (projectmutations.Result, error) {
	if err := normalize(&in); err != nil {
		return projectmutations.Result{}, err
	}
	normalized, err := json.Marshal(map[string]any{"id": in.ID, "customer_id": in.CustomerID, "request_id": in.RequestID, "fields": in.Fields, "options": in.Options, "values": in.Values, "add": in.Add, "remove": in.Remove, "expected_revision": in.ExpectedRevision})
	if err != nil {
		return projectmutations.Result{}, err
	}
	receipt, err := mutationreceipts.Execute(ctx, r.db, mutationreceipts.Command{ActorID: in.ActorID, WorkspaceID: in.WorkspaceID, Tool: in.Tool, Action: in.Action, RequestID: in.RetryID, NormalizedInput: normalized}, mutationreceipts.Operation{
		Authorize: func(ctx context.Context, tx pgx.Tx) error { return authorize(ctx, tx, in) },
		Apply: func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
			var rev int64
			if scanErr := tx.QueryRow(ctx, `SELECT revision FROM customer_revisions WHERE workspace_id=$1 FOR UPDATE`, in.WorkspaceID).Scan(&rev); scanErr != nil {
				return nil, scanErr
			}
			if in.ExpectedRevision != nil && *in.ExpectedRevision != token(in.WorkspaceID, rev) {
				return nil, projectmutations.ErrRevision
			}
			id, after, before, applyErr := apply(ctx, tx, in)
			if applyErr != nil {
				return nil, applyErr
			}
			old, applyErr := json.Marshal(before)
			if applyErr != nil {
				return nil, applyErr
			}
			current, applyErr := json.Marshal(after)
			if applyErr != nil {
				return nil, applyErr
			}
			if _, applyErr = tx.Exec(ctx, `INSERT INTO customer_events(workspace_id,resource_id,resource_kind,action,actor_id,input,before_value,after_value) VALUES($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8::jsonb)`, in.WorkspaceID, id, in.Tool, in.Action, in.ActorID, normalized, old, current); applyErr != nil {
				return nil, applyErr
			}
			if applyErr = tx.QueryRow(ctx, `SELECT revision FROM customer_revisions WHERE workspace_id=$1`, in.WorkspaceID).Scan(&rev); applyErr != nil {
				return nil, applyErr
			}
			// set_values and associations have null SDK results; their audit keeps full input.
			if in.Action == "set_values" || in.Action == "manage_workitems" {
				current = json.RawMessage("null")
			}
			// Freeze the accepted representation before persisting the receipt.
			// Timezone preferences are not part of the normalized mutation identity.
			current, applyErr = djangoapi.JSONTimestamps(current, in.Timezone)
			if applyErr != nil {
				return nil, applyErr
			}
			return json.Marshal(projectmutations.Result{ID: id.String(), Revision: token(in.WorkspaceID, rev), Record: current})
		},
	})
	if err != nil {
		var failure *pgconn.PgError
		if errors.As(err, &failure) && (failure.Code == "23505" || failure.Code == "23514" || failure.Code == "23503") {
			err = projectmutations.ErrInvalid
		}
		return projectmutations.Result{}, err
	}
	var result projectmutations.Result
	if err = json.Unmarshal(receipt.Value, &result); err != nil {
		return result, err
	}
	if in.RetryID != nil {
		result.RequestID, result.ReplayExpiresAt, result.Replayed = in.RetryID, receipt.ReplayExpiresAt, &receipt.Replayed
	}
	return result, nil
}
func customerMatch(ctx context.Context, tx pgx.Tx, in Command) (uuid.UUID, error) {
	query := `SELECT jsonb_build_object('id',id) FROM customers WHERE workspace_id=$1 AND deleted_at IS NULL`
	args := []any{in.WorkspaceID}
	if in.Fields["external_source"] != nil && in.Fields["external_id"] != nil {
		query += ` AND external_source=$2 AND external_id=$3`
		args = append(args, in.Fields["external_source"], in.Fields["external_id"])
	} else {
		query += ` AND name=$2`
		args = append(args, in.Fields["name"])
	}
	found, err := rows(ctx, tx, query+` ORDER BY id FOR UPDATE`, args...)
	if err != nil {
		return uuid.Nil, err
	}
	if len(found) > 1 {
		return uuid.Nil, projectmutations.ErrInvalid
	}
	if len(found) == 0 {
		return uuid.Nil, nil
	}
	return uuid.Parse(found[0]["id"].(string))
}
func apply(ctx context.Context, tx pgx.Tx, in Command) (uuid.UUID, any, any, error) {
	id := in.ID
	create := in.Action == "create"
	var before map[string]any
	var err error
	if in.Tool == "customer" && (create || in.Action == "delete" && id == uuid.Nil) {
		id, err = customerMatch(ctx, tx, in)
		if err != nil {
			return id, nil, nil, err
		}
		if id != uuid.Nil {
			create = false
		} else if in.Action == "delete" {
			return uuid.Nil, nil, nil, nil
		}
	}
	if in.Action == "manage_workitems" {
		id = in.CustomerID
	}
	if in.Action == "set_values" {
		before, readValuesErr := readValues(ctx, tx, in.Query)
		if readValuesErr != nil {
			return in.CustomerID, nil, nil, readValuesErr
		}
		if readValuesErr = setValues(ctx, tx, in); readValuesErr != nil {
			return in.CustomerID, nil, nil, readValuesErr
		}
		after, readValuesErr := readValues(ctx, tx, in.Query)
		return in.CustomerID, after, before, readValuesErr
	}
	if !create {
		before, err = record(ctx, tx, in.Query, id)
		if err != nil {
			return id, nil, nil, err
		}
	} else {
		id = uuid.New()
	}
	if in.Action == "delete" {
		err = deleteResource(ctx, tx, in, id, before)
		return id, nil, before, err
	}
	if in.Action == "manage_workitems" {
		err = manageItems(ctx, tx, in, before["name"].(string))
		return id, nil, before, err
	}
	if in.Tool == "customer_property" {
		if err = prepareProperty(&in, before); err != nil {
			return id, nil, nil, err
		}
	}
	if err = writeRow(ctx, tx, in, id, create); err != nil {
		return id, nil, nil, err
	}
	if in.Tool == "customer_property" {
		if err = writeOptions(ctx, tx, in, id); err == nil {
			err = checkProperty(ctx, tx, in, id)
		}
		if err != nil {
			return id, nil, nil, err
		}
	}
	if in.Tool == "customer" && create {
		defaults := in
		defaults.CustomerID = id
		if err = applyDefaults(ctx, tx, defaults); err != nil {
			return id, nil, nil, err
		}
	}
	if in.Tool == "customer_request" && create && len(in.Add) > 0 {
		links := in
		links.RequestID = id
		customerQuery := in.Query
		customerQuery.Tool = "customer"
		customer, recordErr := record(ctx, tx, customerQuery, in.CustomerID)
		if recordErr != nil {
			return id, nil, nil, recordErr
		}
		if recordErr = manageItems(ctx, tx, links, customer["name"].(string)); recordErr != nil {
			return id, nil, nil, recordErr
		}
	}
	after, err := record(ctx, tx, in.Query, id)
	if err != nil {
		return id, nil, nil, err
	}
	projected, err := projection(ctx, tx, in.Query, after)
	return id, projected, before, err
}
func writeRow(ctx context.Context, tx pgx.Tx, in Command, id uuid.UUID, create bool) error {
	fields := maps.Clone(in.Fields)
	if html, ok := fields["description_html"].(string); ok {
		fields["description_stripped"] = content.StripTags(html)
	}
	fields["updated_by_id"] = in.ActorID.String()
	if create {
		fields["id"], fields["workspace_id"], fields["created_by_id"] = id.String(), in.WorkspaceID.String(), in.ActorID.String()
		if in.Tool == "customer_request" {
			fields["customer_id"] = in.CustomerID.String()
		}
	}
	keys := slices.Sorted(maps.Keys(fields))
	columns, values := []string{}, []string{}
	for _, key := range keys {
		quoted := pgx.Identifier{key}.Sanitize()
		columns = append(columns, quoted)
		values = append(values, "x."+quoted)
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	name := table(in.Tool)
	query := `INSERT INTO ` + name + ` (` + strings.Join(columns, ",") + `) SELECT ` + strings.Join(values, ",") + ` FROM jsonb_populate_record(NULL::` + name + `,$1::jsonb) x`
	args := []any{raw}
	if !create {
		query = `UPDATE ` + name + ` t SET (` + strings.Join(columns, ",") + `)=ROW(` + strings.Join(values, ",") + `),updated_at=now() FROM jsonb_populate_record(NULL::` + name + `,$1::jsonb) x WHERE t.id=$2 AND t.workspace_id=$3`
		args = append(args, id, in.WorkspaceID)
	}
	_, err = tx.Exec(ctx, query, args...)
	return err
}
func deleteResource(ctx context.Context, tx pgx.Tx, in Command, id uuid.UUID, before map[string]any) error {
	if in.Tool == "customer_property" {
		if _, err := tx.Exec(ctx, `DELETE FROM customer_property_values WHERE property_id=$1 AND workspace_id=$2`, id, in.WorkspaceID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE customer_property_options SET deleted_at=now(),updated_at=now(),updated_by_id=$3 WHERE property_id=$1 AND workspace_id=$2 AND deleted_at IS NULL`, id, in.WorkspaceID, in.ActorID); err != nil {
			return err
		}
	} else {
		name := before["name"].(string)
		if in.Tool == "customer" {
			in.CustomerID = id
			in.RequestID = uuid.Nil
		} else {
			in.RequestID = id
			q := in.Query
			q.Tool = "customer"
			customer, err := record(ctx, tx, q, in.CustomerID)
			if err != nil {
				return err
			}
			name = customer["name"].(string)
		}
		pairs, err := unlinkItems(ctx, tx, in, nil)
		if err != nil {
			return err
		}
		for _, pair := range pairs {
			if err = activity(ctx, tx, in, pair[0], pair[1], name, "deleted"); err != nil {
				return err
			}
		}
		if in.Tool == "customer" {
			if _, err = tx.Exec(ctx, `UPDATE customer_requests SET deleted_at=now(),updated_at=now(),updated_by_id=$3 WHERE customer_id=$1 AND workspace_id=$2 AND deleted_at IS NULL`, id, in.WorkspaceID, in.ActorID); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `DELETE FROM customer_property_values WHERE customer_id=$1 AND workspace_id=$2`, id, in.WorkspaceID); err != nil {
				return err
			}
		}
	}
	_, err := tx.Exec(ctx, `UPDATE `+table(in.Tool)+` SET deleted_at=now(),updated_at=now(),updated_by_id=$3 WHERE id=$1 AND workspace_id=$2`, id, in.WorkspaceID, in.ActorID)
	return err
}
