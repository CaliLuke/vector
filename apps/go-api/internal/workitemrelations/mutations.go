package workitemrelations

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func validate(in *Command) error {
	if in.ExpectedRevision != nil && *in.ExpectedRevision == "" {
		return projectmutations.ErrInvalid
	}
	switch in.Action {
	case "create_definition":
		if in.Changes.Name == nil || in.DefinitionID != uuid.Nil {
			return projectmutations.ErrInvalid
		}
	case "update_definition", "delete_definition":
		if in.DefinitionID == uuid.Nil {
			return projectmutations.ErrInvalid
		}
	case "create", "delete":
		if in.IssueID == uuid.Nil || in.ProjectID == uuid.Nil {
			return projectmutations.ErrInvalid
		}
		if in.Action == "create" {
			if len(in.Targets) == 0 || len(in.Targets) > 1000 {
				return projectmutations.ErrInvalid
			}
			if in.RelationType != "" {
				if !slices.Contains(DependencyTypes, in.RelationType) {
					return projectmutations.ErrInvalid
				}
			} else if in.DefinitionID == uuid.Nil || in.DefinitionLabel == "" {
				return projectmutations.ErrInvalid
			}
		} else {
			if in.RelatedID == uuid.Nil {
				return projectmutations.ErrInvalid
			}
			in.Targets = nil
		}
	default:
		return projectmutations.ErrInvalid
	}
	for _, id := range append(slices.Clone(in.Targets), in.RelatedID) {
		if id == in.IssueID && id != uuid.Nil {
			return projectmutations.ErrInvalid
		}
	}
	for _, id := range in.Targets {
		if id == uuid.Nil {
			return projectmutations.ErrInvalid
		}
	}
	slices.SortFunc(in.Targets, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	in.Targets = slices.Compact(in.Targets)
	for _, value := range []*string{in.Changes.Name, in.Changes.Outward, in.Changes.Inward, in.Changes.Color} {
		if value != nil {
			trimmed := strings.TrimSpace(*value)
			if strings.ContainsRune(trimmed, 0) || utf8.RuneCountInString(trimmed) > 255 || trimmed == "" {
				return projectmutations.ErrInvalid
			}
			*value = trimmed
		}
	}
	return nil
}
func (r *Repository) Mutate(ctx context.Context, in Command) (projectmutations.Result, error) {
	if err := validate(&in); err != nil {
		return projectmutations.Result{}, err
	}
	raw, err := json.Marshal(struct {
		Definition, Related uuid.UUID
		Targets             []uuid.UUID
		Type, Label         string
		Dependency          bool
		Changes             Changes
		Revision            *string
		Issue               uuid.UUID
	}{in.DefinitionID, in.RelatedID, in.Targets, in.RelationType, in.DefinitionLabel, in.IsDependency, in.Changes, in.ExpectedRevision, in.IssueID})
	if err != nil {
		return projectmutations.Result{}, err
	}
	var project *uuid.UUID
	if in.ProjectID != uuid.Nil {
		project = &in.ProjectID
	}
	receipt, err := mutationreceipts.Execute(ctx, r.db, mutationreceipts.Command{ActorID: in.ActorID, WorkspaceID: in.WorkspaceID, ProjectID: project, Tool: "workitem_relation", Action: in.Action, RequestID: in.RequestID, NormalizedInput: raw}, mutationreceipts.Operation{
		Authorize: func(ctx context.Context, tx pgx.Tx) error { return authorizeMutation(ctx, tx, in) },
		Apply: func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
			var revision int64
			if scanErr := tx.QueryRow(ctx, `SELECT revision FROM work_item_relation_revisions WHERE workspace_id=$1 FOR UPDATE`, in.WorkspaceID).Scan(&revision); scanErr != nil {
				return nil, scanErr
			}
			if in.ExpectedRevision != nil && *in.ExpectedRevision != revisionToken(in.WorkspaceID, revision) {
				return nil, projectmutations.ErrRevision
			}
			var record json.RawMessage
			id := in.IssueID
			var operationErr error
			if strings.HasSuffix(in.Action, "_definition") {
				id, record, operationErr = mutateDefinition(ctx, tx, in)
			} else {
				record, operationErr = mutateEdges(ctx, tx, in)
			}
			if operationErr != nil {
				return nil, operationErr
			}
			if operationErr = tx.QueryRow(ctx, `SELECT revision FROM work_item_relation_revisions WHERE workspace_id=$1`, in.WorkspaceID).Scan(&revision); operationErr != nil {
				return nil, operationErr
			}
			return json.Marshal(projectmutations.Result{ID: id.String(), Revision: revisionToken(in.WorkspaceID, revision), Record: record})
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
func mutateDefinition(ctx context.Context, tx pgx.Tx, in Command) (uuid.UUID, json.RawMessage, error) {
	id := in.DefinitionID
	if in.Action == "create_definition" {
		id = uuid.New()
		outward, inward := *in.Changes.Name, *in.Changes.Name
		if in.Changes.Outward != nil {
			outward = *in.Changes.Outward
		}
		if in.Changes.Inward != nil {
			inward = *in.Changes.Inward
		}
		_, err := tx.Exec(ctx, `INSERT INTO work_item_relation_definitions(id,workspace_id,name,outward,inward,is_active,color,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$5,COALESCE($6,true),COALESCE($7,''),$8,$8)`, id, in.WorkspaceID, *in.Changes.Name, outward, inward, in.Changes.Active, in.Changes.Color, in.ActorID)
		if err != nil {
			return id, nil, err
		}
	} else {
		var found uuid.UUID
		err := tx.QueryRow(ctx, `SELECT id FROM work_item_relation_definitions WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL FOR UPDATE`, id, in.WorkspaceID).Scan(&found)
		if errors.Is(err, pgx.ErrNoRows) {
			return id, nil, projectmutations.ErrNotFound
		}
		if err != nil {
			return id, nil, err
		}
		if in.Action == "delete_definition" {
			_, err = tx.Exec(ctx, `UPDATE work_item_relation_definitions SET deleted_at=now(),updated_at=now(),updated_by_id=$2 WHERE id=$1`, id, in.ActorID)
			return id, json.RawMessage("null"), err
		}
		_, err = tx.Exec(ctx, `UPDATE work_item_relation_definitions SET name=COALESCE($2,name),outward=COALESCE($3,outward),inward=COALESCE($4,inward),is_active=COALESCE($5,is_active),color=COALESCE($6,color),updated_at=now(),updated_by_id=$7 WHERE id=$1 AND (name,outward,inward,is_active,color) IS DISTINCT FROM (COALESCE($2,name),COALESCE($3,outward),COALESCE($4,inward),COALESCE($5,is_active),COALESCE($6,color))`, id, in.Changes.Name, in.Changes.Outward, in.Changes.Inward, in.Changes.Active, in.Changes.Color, in.ActorID)
		if err != nil {
			return id, nil, err
		}
	}
	rows, err := definitionRows(ctx, tx, in.Query, id)
	if err != nil {
		return id, nil, err
	}
	if len(rows) != 1 {
		return id, nil, projectmutations.ErrNotFound
	}
	return id, rows[0], nil
}
