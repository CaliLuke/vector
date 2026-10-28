package projectconfiguration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrLabelRevision = errors.New("label revision changed")

type LabelChanges struct {
	Name, Color, Description, Parent, ExternalSource, ExternalID *string
	SortOrder                                                    *float64
}
type LabelMutation struct {
	ActorID, WorkspaceID, ProjectID, LabelID uuid.UUID
	WorkspaceSlug, Action                    string
	Changes                                  LabelChanges
	RequestID                                *uuid.UUID
	ExpectedRevision                         *string
	Revalidate                               func(context.Context) error
}
type LabelMutationResult struct {
	LabelID         string          `json:"label_id"`
	Revision        string          `json:"revision"`
	Result          json.RawMessage `json:"result"`
	RequestID       *uuid.UUID      `json:"request_id,omitempty"`
	ReplayExpiresAt *time.Time      `json:"replay_expires_at,omitempty"`
	Replayed        *bool           `json:"replayed,omitempty"`
}

func labelMutationFields(cmd LabelMutation) (map[string]any, error) {
	if cmd.ActorID == uuid.Nil || cmd.WorkspaceID == uuid.Nil || cmd.ProjectID == uuid.Nil || cmd.WorkspaceSlug == "" {
		return nil, failure(400, "error", "Invalid label scope")
	}
	if cmd.Action != "create" && cmd.Action != "update" && cmd.Action != "delete" {
		return nil, failure(400, "error", "Invalid label action")
	}
	if cmd.Action == "create" && (cmd.LabelID != uuid.Nil || cmd.ExpectedRevision != nil) || cmd.Action != "create" && cmd.LabelID == uuid.Nil {
		return nil, failure(400, "error", "Invalid label target")
	}
	if cmd.ExpectedRevision != nil && *cmd.ExpectedRevision == "" {
		return nil, failure(400, "error", "Invalid revision")
	}
	fields := map[string]any{}
	for key, value := range map[string]*string{"name": cmd.Changes.Name, "color": cmd.Changes.Color, "description": cmd.Changes.Description, "parent": cmd.Changes.Parent, "external_source": cmd.Changes.ExternalSource, "external_id": cmd.Changes.ExternalID} {
		if value != nil && *value != "" {
			fields[key] = strings.TrimSpace(*value)
		}
	}
	if parent, ok := fields["parent"].(string); ok {
		id, err := uuid.Parse(parent)
		if err != nil || id == uuid.Nil {
			return nil, failure(400, "error", "Invalid parent")
		}
		fields["parent"] = id.String()
	}
	if cmd.Changes.SortOrder != nil {
		value := *cmd.Changes.SortOrder
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, failure(400, "error", "Invalid sort order")
		}
		if value == 0 {
			value = 0
		}
		fields["sort_order"] = value
	}
	if cmd.Action == "delete" && len(fields) != 0 {
		return nil, failure(400, "error", "Delete does not accept fields")
	}
	if cmd.Action == "create" {
		for key, value := range map[string]any{"color": "", "description": "", "external_source": nil, "external_id": nil} {
			if _, ok := fields[key]; !ok {
				fields[key] = value
			}
		}
	}
	return fields, nil
}
func (r *Repository) MutateLabel(ctx context.Context, cmd LabelMutation) (LabelMutationResult, error) {
	fields, err := labelMutationFields(cmd)
	if err != nil {
		return LabelMutationResult{}, err
	}
	body, err := json.Marshal(fields)
	if err != nil {
		return LabelMutationResult{}, err
	}
	normalized, err := json.Marshal(map[string]any{"fields": fields, "label_id": cmd.LabelID, "expected_revision": cmd.ExpectedRevision})
	if err != nil {
		return LabelMutationResult{}, err
	}
	db, ok := r.db.(mutationreceipts.Database)
	if !ok {
		return LabelMutationResult{}, errors.New("label transaction database unavailable")
	}
	receipt, err := mutationreceipts.Execute(ctx, db, mutationreceipts.Command{ActorID: cmd.ActorID, WorkspaceID: cmd.WorkspaceID, ProjectID: &cmd.ProjectID, Tool: "label", Action: cmd.Action, RequestID: cmd.RequestID, NormalizedInput: normalized}, mutationreceipts.Operation{
		Authorize: func(ctx context.Context, tx pgx.Tx) error { return authorizeLabelMutation(ctx, tx, cmd) },
		Apply: func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
			return applyLabelMutation(ctx, tx, cmd, body)
		},
	})
	if err != nil {
		return LabelMutationResult{}, err
	}
	var result LabelMutationResult
	if err = json.Unmarshal(receipt.Value, &result); err != nil {
		return result, err
	}
	if cmd.RequestID != nil {
		result.RequestID, result.ReplayExpiresAt, result.Replayed = cmd.RequestID, receipt.ReplayExpiresAt, &receipt.Replayed
	}
	return result, nil
}
func authorizeLabelMutation(ctx context.Context, tx pgx.Tx, cmd LabelMutation) error {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `SELECT w.id FROM users u JOIN workspace_members wm ON wm.member_id=u.id JOIN workspaces w ON w.id=wm.workspace_id
 WHERE u.id=$1 AND u.is_active AND w.id=$2 AND w.slug=$3 AND w.deleted_at IS NULL
 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20) FOR SHARE OF u,wm,w`, cmd.ActorID, cmd.WorkspaceID, cmd.WorkspaceSlug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return failure(403, "error", "Label access denied")
	}
	if err != nil {
		return err
	}
	err = tx.QueryRow(ctx, `SELECT id FROM projects WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND archived_at IS NULL FOR UPDATE`, cmd.ProjectID, cmd.WorkspaceID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return failure(403, "error", "Label access denied")
	}
	if err != nil {
		return err
	}
	err = tx.QueryRow(ctx, `SELECT member_id FROM project_members WHERE project_id=$1 AND workspace_id=$2 AND member_id=$3 AND is_active AND deleted_at IS NULL AND role IN(5,15,20) FOR SHARE`, cmd.ProjectID, cmd.WorkspaceID, cmd.ActorID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return failure(403, "error", "Label access denied")
	}
	if err != nil {
		return err
	}
	if cmd.Revalidate != nil {
		return cmd.Revalidate(ctx)
	}
	return nil
}
func applyLabelMutation(ctx context.Context, tx pgx.Tx, cmd LabelMutation, body []byte) (json.RawMessage, error) {
	if cmd.Action != "create" {
		var id uuid.UUID
		err := tx.QueryRow(ctx, `SELECT id FROM labels WHERE id=$1 AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL FOR UPDATE`, cmd.LabelID, cmd.ProjectID, cmd.WorkspaceID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, failure(404, "error", "Label unavailable")
		}
		if err != nil {
			return nil, err
		}
		var revision int64
		if err = tx.QueryRow(ctx, `SELECT revision FROM label_edit_revisions WHERE label_id=$1`, id).Scan(&revision); err != nil {
			return nil, err
		}
		if cmd.ExpectedRevision != nil && *cmd.ExpectedRevision != fmt.Sprintf("label:%s:%d", id, revision) {
			return nil, ErrLabelRevision
		}
	}
	method := map[string]string{"create": http.MethodPost, "update": http.MethodPatch, "delete": http.MethodDelete}[cmd.Action]
	transaction := &Repository{db: tx}
	id := ""
	if cmd.LabelID != uuid.Nil {
		id = cmd.LabelID.String()
	}
	raw, err := transaction.writeAPILabel(ctx, tx, cmd.ActorID.String(), cmd.WorkspaceSlug, cmd.ProjectID.String(), id, method, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, err
	}
	if cmd.Action == "create" {
		var value struct{ ID string }
		if err = json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		id = value.ID
	}
	var revision int64
	if err = tx.QueryRow(ctx, `SELECT revision FROM label_edit_revisions WHERE label_id=$1`, id).Scan(&revision); err != nil {
		return nil, err
	}
	return json.Marshal(LabelMutationResult{LabelID: id, Revision: fmt.Sprintf("label:%s:%d", id, revision), Result: raw})
}

func (r *Repository) VisibleLabelWrites(ctx context.Context, actor, workspace uuid.UUID, slug string) (bool, error) {
	var visible bool
	if actor == uuid.Nil || workspace == uuid.Nil || slug == "" {
		return false, nil
	}
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users u JOIN workspace_members wm ON wm.member_id=u.id
 JOIN workspaces w ON w.id=wm.workspace_id JOIN projects p ON p.workspace_id=w.id
 JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=w.id AND pm.member_id=u.id
 WHERE u.id=$1 AND u.is_active AND w.id=$2 AND w.slug=$3 AND w.deleted_at IS NULL
 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20)
 AND p.deleted_at IS NULL AND p.archived_at IS NULL AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20))`, actor, workspace, slug).Scan(&visible)
	return visible, err
}
