package workitemtypes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrWorkspaceOwned = errors.New("work item types are managed by the workspace")

type APIChanges struct {
	Name, Description, ExternalSource, ExternalID *string
	Active                                        *bool
	ProjectIDs                                    []uuid.UUID
}
type APIMutation struct {
	APIScope
	Action           string
	TypeID           uuid.UUID
	TypeIDs          []uuid.UUID
	Changes          APIChanges
	RequestID        *uuid.UUID
	ExpectedRevision *string
	Revalidate       func(context.Context) error
}

func validateAPIMutation(in *APIMutation) error {
	if in.ExpectedRevision != nil && *in.ExpectedRevision == "" {
		return projectmutations.ErrInvalid
	}
	switch in.Action {
	case "create", "update", "delete":
		if in.ProjectID != uuid.Nil {
			return ErrWorkspaceOwned
		}
		if in.Action == "create" && in.TypeID != uuid.Nil || in.Action != "create" && in.TypeID == uuid.Nil {
			return projectmutations.ErrInvalid
		}
	case "resolve":
		if in.ProjectID == uuid.Nil || in.TypeID != uuid.Nil || in.Changes.Name == nil {
			return projectmutations.ErrInvalid
		}
	case "import_to_project":
		if in.ProjectID == uuid.Nil || in.TypeID != uuid.Nil || len(in.TypeIDs) == 0 || len(in.TypeIDs) > 100 {
			return projectmutations.ErrInvalid
		}
	default:
		return projectmutations.ErrInvalid
	}
	if in.Action == "create" && in.Changes.Name == nil {
		return projectmutations.ErrInvalid
	}
	if len(in.TypeIDs) > 0 && in.Action != "import_to_project" {
		return projectmutations.ErrInvalid
	}
	if in.Action == "delete" || in.Action == "import_to_project" {
		if in.Changes.Name != nil || in.Changes.Description != nil || in.Changes.Active != nil || in.Changes.ExternalSource != nil || in.Changes.ExternalID != nil || in.Changes.ProjectIDs != nil {
			return projectmutations.ErrInvalid
		}
	}
	for _, value := range []*string{in.Changes.Name, in.Changes.Description, in.Changes.ExternalSource, in.Changes.ExternalID} {
		if value != nil && strings.ContainsRune(*value, 0) {
			return projectmutations.ErrInvalid
		}
	}
	for _, value := range []*string{in.Changes.ExternalSource, in.Changes.ExternalID} {
		if value != nil && utf8.RuneCountInString(*value) > 255 {
			return projectmutations.ErrInvalid
		}
	}
	if in.Changes.Name != nil {
		value := strings.TrimSpace(*in.Changes.Name)
		if value == "" || utf8.RuneCountInString(value) > 255 {
			return projectmutations.ErrInvalid
		}
		in.Changes.Name = &value
	}
	for _, ids := range [][]uuid.UUID{in.TypeIDs, in.Changes.ProjectIDs} {
		for _, id := range ids {
			if id == uuid.Nil {
				return projectmutations.ErrInvalid
			}
		}
	}
	in.TypeIDs = canonicalAPIIDs(in.TypeIDs)
	in.Changes.ProjectIDs = canonicalAPIIDs(in.Changes.ProjectIDs)
	return nil
}
func canonicalAPIIDs(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return nil
	}
	ids = slices.Clone(ids)
	slices.SortFunc(ids, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	return slices.Compact(ids)
}
func (r *Repository) MutateAPI(ctx context.Context, in APIMutation) (projectmutations.Result, error) {
	if err := validateAPIMutation(&in); err != nil {
		return projectmutations.Result{}, err
	}
	db, ok := r.db.(mutationreceipts.Database)
	if !ok {
		return projectmutations.Result{}, errors.New("type transaction database unavailable")
	}
	var project *uuid.UUID
	if in.ProjectID != uuid.Nil {
		project = &in.ProjectID
	}
	raw, err := json.Marshal(map[string]any{"type_id": in.TypeID, "type_ids": in.TypeIDs, "changes": in.Changes, "expected_revision": in.ExpectedRevision})
	if err != nil {
		return projectmutations.Result{}, err
	}
	var access apiAccess
	receipt, err := mutationreceipts.Execute(ctx, db, mutationreceipts.Command{ActorID: in.ActorID, WorkspaceID: in.WorkspaceID, ProjectID: project, Tool: "workitem_type", Action: in.Action, RequestID: in.RequestID, NormalizedInput: raw}, mutationreceipts.Operation{
		Authorize: func(ctx context.Context, tx pgx.Tx) error {
			if _, execErr := tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, in.WorkspaceID); execErr != nil {
				return execErr
			}
			var operationErr error
			access, operationErr = authorizeAPI(ctx, tx, in.APIScope, true)
			if operationErr != nil {
				return operationErr
			}
			if (in.Action == "create" || in.Action == "update" || in.Action == "delete") && !access.workspaceAdmin || in.Action == "import_to_project" && !access.projectAdmin {
				return projectmutations.ErrForbidden
			}
			if in.Revalidate != nil {
				return in.Revalidate(ctx)
			}
			return nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
			var revision int64
			if scanErr := tx.QueryRow(ctx, `SELECT revision FROM work_item_type_api_revisions WHERE workspace_id=$1 FOR UPDATE`, in.WorkspaceID).Scan(&revision); scanErr != nil {
				return nil, scanErr
			}
			token := func() string { return fmt.Sprintf("workitem_type:%s:%d", in.WorkspaceID, revision) }
			if in.ExpectedRevision != nil && *in.ExpectedRevision != token() {
				return nil, projectmutations.ErrRevision
			}
			before, loadCatalogErr := loadCatalog(ctx, tx, in.WorkspaceID.String(), access.workspaceAdmin)
			if loadCatalogErr != nil {
				return nil, loadCatalogErr
			}
			if !before.Enabled {
				return nil, reject(409, "Enable work item types in workspace settings first.")
			}
			id, loadCatalogErr := applyAPITypeMutation(ctx, tx, in, access, before)
			if loadCatalogErr != nil {
				return nil, loadCatalogErr
			}
			var current int64
			if loadCatalogErr = tx.QueryRow(ctx, `SELECT revision FROM work_item_type_api_revisions WHERE workspace_id=$1`, in.WorkspaceID).Scan(&current); loadCatalogErr != nil {
				return nil, loadCatalogErr
			}
			// Keep native editor revisions in sync with changes made through the API.
			if current != revision {
				if _, loadCatalogErr = tx.Exec(ctx, `UPDATE work_item_type_settings SET revision=revision+1 WHERE workspace_id=$1`, in.WorkspaceID); loadCatalogErr != nil {
					return nil, loadCatalogErr
				}
			}
			var record json.RawMessage
			if in.Action != "delete" && in.Action != "import_to_project" {
				record, loadCatalogErr = readAPITypeRows(ctx, tx, APIRead{APIScope: in.APIScope, TypeID: id})
				if loadCatalogErr != nil {
					return nil, loadCatalogErr
				}
			}
			if loadCatalogErr = tx.QueryRow(ctx, `SELECT revision FROM work_item_type_api_revisions WHERE workspace_id=$1`, in.WorkspaceID).Scan(&revision); loadCatalogErr != nil {
				return nil, loadCatalogErr
			}
			return json.Marshal(projectmutations.Result{ID: id.String(), Revision: token(), Record: record})
		},
	})
	if err != nil {
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
func applyAPITypeMutation(ctx context.Context, tx pgx.Tx, in APIMutation, access apiAccess, before Catalog) (uuid.UUID, error) {
	switch in.Action {
	case "delete":
		return in.TypeID, remove(ctx, tx, in.WorkspaceID.String(), in.ActorID.String(), in.TypeID.String(), before)
	case "import_to_project":
		for _, id := range in.TypeIDs {
			if err := adoptAPIType(ctx, tx, in.APIScope, id); err != nil {
				return uuid.Nil, err
			}
		}
		return in.ProjectID, nil
	case "resolve":
		return resolveAPIType(ctx, tx, in, access, before)
	default:
		id, err := saveAPIType(ctx, tx, in, before)
		if err != nil {
			return id, err
		}
		if in.Changes.ProjectIDs != nil {
			err = replaceAPITypeProjects(ctx, tx, in.APIScope, id, in.Changes.ProjectIDs)
		}
		return id, err
	}
}
