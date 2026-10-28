package workitemtypes

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) MutatePropertyAPI(ctx context.Context, in PropertyAPIMutation) (projectmutations.Result, error) {
	if in.ProjectID != uuid.Nil && (in.Action != "create" || in.TypeID == uuid.Nil) && in.Action != "manage_type_properties" {
		return projectmutations.Result{}, ErrWorkspaceOwned
	}
	switch in.Action {
	case "create":
		if in.Changes.DisplayName == nil || in.Changes.Kind == nil {
			return projectmutations.Result{}, projectmutations.ErrInvalid
		}
	case "update", "delete", "create_option", "update_option", "delete_option":
		if in.PropertyID == uuid.Nil {
			return projectmutations.Result{}, projectmutations.ErrInvalid
		}
	case "manage_type_properties":
		if in.TypeID == uuid.Nil || len(in.Attach)+len(in.Detach) == 0 {
			return projectmutations.Result{}, projectmutations.ErrInvalid
		}
	default:
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	if strings.HasSuffix(in.Action, "_option") && in.Action != "create_option" && in.OptionID == uuid.Nil {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	if in.ExpectedRevision != nil && *in.ExpectedRevision == "" {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	db, ok := r.db.(mutationreceipts.Database)
	if !ok {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	in.Attach, in.Detach = canonicalAPIIDs(in.Attach), canonicalAPIIDs(in.Detach)
	raw, err := json.Marshal(struct {
		Property, Type, Option uuid.UUID
		Changes                PropertyAPIChanges
		OptionChanges          PropertyAPIOption
		Attach, Detach         []uuid.UUID
		Revision               *string
	}{in.PropertyID, in.TypeID, in.OptionID, in.Changes, in.Option, in.Attach, in.Detach, in.ExpectedRevision})
	if err != nil {
		return projectmutations.Result{}, err
	}
	var project *uuid.UUID
	if in.ProjectID != uuid.Nil {
		project = &in.ProjectID
	}
	receipt, err := mutationreceipts.Execute(ctx, db, mutationreceipts.Command{ActorID: in.ActorID, WorkspaceID: in.WorkspaceID, ProjectID: project, Tool: "workitem_property", Action: in.Action, RequestID: in.RequestID, NormalizedInput: raw}, mutationreceipts.Operation{
		Authorize: func(ctx context.Context, tx pgx.Tx) error {
			if _, execErr := tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, in.WorkspaceID); execErr != nil {
				return execErr
			}
			access, authorizeAPIErr := authorizeAPI(ctx, tx, in.APIScope, true)
			if authorizeAPIErr != nil {
				return authorizeAPIErr
			}
			if !access.workspaceAdmin {
				return projectmutations.ErrForbidden
			}
			if in.Revalidate != nil {
				return in.Revalidate(ctx)
			}
			return nil
		}, Apply: func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
			var revision int64
			if scanErr := tx.QueryRow(ctx, `SELECT revision FROM work_item_type_api_revisions WHERE workspace_id=$1 FOR UPDATE`, in.WorkspaceID).Scan(&revision); scanErr != nil {
				return nil, scanErr
			}
			if in.ExpectedRevision != nil && *in.ExpectedRevision != propertyRevision(in.WorkspaceID, revision) {
				return nil, projectmutations.ErrRevision
			}
			before, loadPropertiesErr := loadProperties(ctx, tx, in.WorkspaceID.String(), in.ActorID.String(), true)
			if loadPropertiesErr != nil {
				return nil, loadPropertiesErr
			}
			if !before.Enabled {
				return nil, reject(409, "Enable work item types first.")
			}
			id, record, loadPropertiesErr := applyPropertyAPI(ctx, tx, in, before)
			if loadPropertiesErr != nil {
				return nil, loadPropertiesErr
			}
			if _, loadPropertiesErr = tx.Exec(ctx, `UPDATE work_item_type_settings SET revision=revision+1 WHERE workspace_id=$1`, in.WorkspaceID); loadPropertiesErr != nil {
				return nil, loadPropertiesErr
			}
			if loadPropertiesErr = tx.QueryRow(ctx, `SELECT revision FROM work_item_type_api_revisions WHERE workspace_id=$1`, in.WorkspaceID).Scan(&revision); loadPropertiesErr != nil {
				return nil, loadPropertiesErr
			}
			return json.Marshal(projectmutations.Result{ID: id.String(), Revision: propertyRevision(in.WorkspaceID, revision), Record: record})
		}})
	if err != nil {
		return projectmutations.Result{}, err
	}
	var result projectmutations.Result
	if err = json.Unmarshal(receipt.Value, &result); err != nil {
		return result, err
	}
	if in.RequestID != nil {
		result.RequestID, result.Replayed, result.ReplayExpiresAt = in.RequestID, &receipt.Replayed, receipt.ReplayExpiresAt
	}
	return result, nil
}
func applyPropertyAPI(ctx context.Context, tx pgx.Tx, in PropertyAPIMutation, before PropertyCatalog) (uuid.UUID, json.RawMessage, error) {
	if in.Action == "manage_type_properties" {
		err := manageAPIProperties(ctx, tx, in, before)
		if err != nil {
			return in.TypeID, nil, err
		}
		if len(in.Attach) == 0 {
			return in.TypeID, json.RawMessage("null"), nil
		}
		raw, err := json.Marshal(in.Attach)
		return in.TypeID, raw, err
	}
	if in.TypeID != uuid.Nil && in.PropertyID != uuid.Nil {
		rows, err := loadAPIPropertyRows(ctx, tx, in.PropertyAPIQuery)
		if err != nil {
			return in.PropertyID, nil, err
		}
		selected, err := selectAPIProperties(ctx, tx, in.PropertyAPIQuery, rows)
		if err != nil {
			return in.PropertyID, nil, err
		}
		if len(selected) != 1 {
			return in.PropertyID, nil, projectmutations.ErrNotFound
		}
	}
	if in.Action == "delete" {
		err := removeProperty(ctx, tx, in.WorkspaceID.String(), PropertyInput{Action: "delete", ID: in.PropertyID.String()}, before, false)
		return in.PropertyID, json.RawMessage("null"), err
	}
	var id uuid.UUID
	var err error
	if strings.HasSuffix(in.Action, "_option") {
		id, err = changeAPIOption(ctx, tx, &in, before)
	} else {
		id, err = changeAPIProperty(ctx, tx, &in, before)
	}
	if err != nil {
		return id, nil, err
	}
	if in.Action == "delete_option" {
		return id, json.RawMessage("null"), nil
	}
	q := in.PropertyAPIQuery
	q.ProjectID = uuid.Nil
	q.TypeID = uuid.Nil
	q.Action = "retrieve"
	if strings.HasSuffix(in.Action, "_option") {
		q.Action = "retrieve_option"
		q.OptionID = id
	} else {
		q.PropertyID = id
	}
	record, err := readPropertyAPI(ctx, tx, q)
	if err != nil {
		return id, nil, err
	}
	raw, err := json.Marshal(record)
	return id, raw, err
}
func manageAPIProperties(ctx context.Context, tx pgx.Tx, in PropertyAPIMutation, before PropertyCatalog) error {
	types, err := loadCatalog(ctx, tx, in.WorkspaceID.String(), true)
	if err != nil {
		return err
	}
	ids := []string{}
	for _, link := range before.Attachments {
		if link.TypeID == in.TypeID.String() {
			ids = append(ids, link.PropertyIDs...)
		}
	}
	for _, id := range in.Attach {
		found := false
		for _, current := range ids {
			found = found || current == id.String()
		}
		if !found {
			ids = append(ids, id.String())
		}
	}
	if len(in.Attach) > 0 {
		if err = attachProperties(ctx, tx, PropertyInput{Action: "attach", TypeID: in.TypeID.String(), PropertyIDs: ids}, before, types); err != nil {
			return err
		}
	}
	for _, id := range in.Detach {
		current, err := loadProperties(ctx, tx, in.WorkspaceID.String(), in.ActorID.String(), true)
		if err != nil {
			return err
		}
		if err = removeProperty(ctx, tx, in.WorkspaceID.String(), PropertyInput{Action: "detach", ID: id.String(), TypeID: in.TypeID.String()}, current, false); err != nil {
			return err
		}
	}
	return nil
}
