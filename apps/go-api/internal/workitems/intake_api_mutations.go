package workitems

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MutateIntake addresses the item rather than the intake association. Both native
// item writers and intake writers advance the item counter used by this token.
func (r *Repository) MutateIntake(ctx context.Context, cmd projectmutations.Command) (projectmutations.Result, error) {
	fields, err := intakeMutationFields(cmd.Action, cmd.Fields)
	if err != nil {
		return projectmutations.Result{}, err
	}
	if cmd.ParentID != uuid.Nil || cmd.ProjectID == uuid.Nil || cmd.Action != "create" && cmd.ResourceID == uuid.Nil || cmd.Action == "create" && (cmd.ResourceID != uuid.Nil || cmd.ExpectedRevision != nil) || cmd.ExpectedRevision != nil && *cmd.ExpectedRevision == "" {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	db, ok := r.db.(mutationreceipts.Database)
	if !ok {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	normalized, err := json.Marshal(map[string]any{"workitem_id": cmd.ResourceID, "fields": fields, "expected_revision": cmd.ExpectedRevision})
	if err != nil {
		return projectmutations.Result{}, err
	}
	q := Request{ActorID: cmd.ActorID.String(), Slug: cmd.WorkspaceSlug, ProjectID: cmd.ProjectID.String(), IssueID: cmd.ResourceID.String(), API: true, Timezone: cmd.Timezone, ExpectedWorkspaceID: &cmd.WorkspaceID, Params: map[string]string{}, Present: map[string]bool{}, Operation: "mcp_intake"}
	resource := projectmutations.Resource{SchemaLock: true, AllowGuests: true, Authorize: func(ctx context.Context, tx pgx.Tx) error { return authorizeIntakeMutation(ctx, tx, cmd, fields) }}
	receipt, err := mutationreceipts.Execute(ctx, db, mutationreceipts.Command{ActorID: cmd.ActorID, WorkspaceID: cmd.WorkspaceID, ProjectID: &cmd.ProjectID, Tool: "intake", Action: cmd.Action, RequestID: cmd.RequestID, NormalizedInput: normalized}, mutationreceipts.Operation{
		Authorize: func(ctx context.Context, tx pgx.Tx) error { return projectmutations.Authorize(ctx, tx, resource, cmd) },
		Apply: func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
			writer := *r
			writer.db, writer.transactionalEffects, writer.publisher = tx, true, projectmutations.Effects{Tx: tx}
			item := cmd.ResourceID.String()
			if cmd.Action != "create" {
				var live bool
				if scanErr := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM intake_issues t JOIN issues i ON i.id=t.issue_id AND i.project_id=t.project_id AND i.workspace_id=t.workspace_id WHERE t.issue_id=$1 AND t.project_id=$2 AND t.workspace_id=$3 AND t.deleted_at IS NULL AND i.deleted_at IS NULL)`, item, cmd.ProjectID, cmd.WorkspaceID).Scan(&live); scanErr != nil {
					return nil, scanErr
				}
				if !live {
					return nil, projectmutations.ErrNotFound
				}
				if cmd.ExpectedRevision != nil {
					token, intakeRevisionErr := intakeRevision(ctx, tx, item)
					if intakeRevisionErr != nil {
						return nil, intakeRevisionErr
					}
					if token != *cmd.ExpectedRevision {
						return nil, projectmutations.ErrRevision
					}
				}
			}
			q.Method = map[string]string{"create": "POST", "update": "PATCH", "delete": "DELETE"}[cmd.Action]
			payload := maps.Clone(fields)
			if cmd.Action == "create" {
				payload = map[string]any{"issue": payload}
			}
			body, marshalErr := json.Marshal(payload)
			if marshalErr != nil {
				return nil, marshalErr
			}
			value, marshalErr := writer.intakeTransaction(ctx, q, bytes.NewReader(body), "application/json")
			if marshalErr != nil {
				return nil, marshalErr
			}
			record := json.RawMessage("null")
			if cmd.Action != "delete" {
				created, ok := value.Body.(map[string]any)
				if !ok {
					return nil, errors.New("invalid native intake result")
				}
				item, ok = created["issue"].(string)
				if !ok {
					return nil, errors.New("intake result has no item identifier")
				}
				q.IssueID = item
				row, rowErr := writer.row(ctx, `SELECT to_jsonb(t) FROM intake_issues t WHERE t.id=$1 AND t.issue_id=$2 AND t.project_id=$3 AND t.workspace_id=$4 AND t.deleted_at IS NULL`, created["id"], item, cmd.ProjectID, cmd.WorkspaceID)
				if rowErr != nil {
					return nil, rowErr
				}
				if cmd.Action == "create" {
					// The native API creation path sanitizes HTML; use the native item
					// asset admission helper before committing the new description.
					issue, rowErr2 := writer.row(ctx, `SELECT to_jsonb(i) FROM issues i WHERE i.id=$1`, item)
					if rowErr2 != nil {
						return nil, rowErr2
					}
					assets := writer
					assets.itemEdit = &itemEditState{command: itemEditCommand{ItemEditInput: ItemEditInput{Mode: "edit", ItemID: item, ProjectID: cmd.ProjectID.String()}}}
					if rowErr2 = assets.associateItemEditAssets(ctx, cmd.ActorID.String(), issue, issue); rowErr2 != nil {
						return nil, rowErr2
					}
				}
				projected, rowErr := writer.intakeAPIRecord(ctx, q, row)
				if rowErr != nil {
					return nil, rowErr
				}
				record, rowErr = json.Marshal(projected)
				if rowErr != nil {
					return nil, rowErr
				}
			}
			token, marshalErr := intakeRevision(ctx, tx, item)
			if marshalErr != nil {
				return nil, marshalErr
			}
			return json.Marshal(projectmutations.Result{ID: item, Revision: token, Record: record})
		},
	})
	if err != nil {
		return projectmutations.Result{}, err
	}
	var result projectmutations.Result
	if err = json.Unmarshal(receipt.Value, &result); err != nil {
		return result, err
	}
	if cmd.RequestID != nil {
		result.RequestID, result.ReplayExpiresAt, result.Replayed = cmd.RequestID, receipt.ReplayExpiresAt, &receipt.Replayed
	}
	return result, nil
}

func intakeRevision(ctx context.Context, tx pgx.Tx, item string) (string, error) {
	var revision int64
	err := tx.QueryRow(ctx, `SELECT revision FROM issue_edit_revisions WHERE issue_id=$1 FOR UPDATE`, item).Scan(&revision)
	return fmt.Sprintf("intake:%s:%d", item, revision), err
}

func authorizeIntakeMutation(ctx context.Context, tx pgx.Tx, cmd projectmutations.Command, fields map[string]any) error {
	var role int
	if err := tx.QueryRow(ctx, `SELECT role FROM project_members WHERE project_id=$1 AND workspace_id=$2 AND member_id=$3 AND is_active AND deleted_at IS NULL`, cmd.ProjectID, cmd.WorkspaceID, cmd.ActorID).Scan(&role); err != nil {
		return err
	}
	if cmd.Action == "update" && role != 20 {
		return projectmutations.ErrForbidden
	}
	var intake string
	err := tx.QueryRow(ctx, `SELECT t.id::text FROM intakes t JOIN projects p ON p.id=t.project_id AND p.workspace_id=t.workspace_id WHERE t.project_id=$1 AND t.workspace_id=$2 AND t.deleted_at IS NULL AND p.intake_view ORDER BY t.name LIMIT 1 FOR SHARE OF t`, cmd.ProjectID, cmd.WorkspaceID).Scan(&intake)
	if errors.Is(err, pgx.ErrNoRows) {
		return projectmutations.ErrNotFound
	}
	if err != nil {
		return err
	}
	if cmd.Action != "create" {
		var creator, project, workspace string
		err = tx.QueryRow(ctx, `SELECT COALESCE(i.created_by_id::text,''),i.project_id::text,i.workspace_id::text FROM issues i WHERE i.id=$1 FOR UPDATE`, cmd.ResourceID).Scan(&creator, &project, &workspace)
		if errors.Is(err, pgx.ErrNoRows) && cmd.Action == "delete" {
			return nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return projectmutations.ErrNotFound
		}
		if err != nil {
			return err
		}
		if project != cmd.ProjectID.String() || workspace != cmd.WorkspaceID.String() {
			return projectmutations.ErrNotFound
		}
		var status int
		err = tx.QueryRow(ctx, `SELECT status FROM intake_issues WHERE issue_id=$1 AND intake_id=$2 AND project_id=$3 AND workspace_id=$4 ORDER BY deleted_at NULLS FIRST,created_at DESC LIMIT 1 FOR UPDATE`, cmd.ResourceID, intake, cmd.ProjectID, cmd.WorkspaceID).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) && cmd.Action == "delete" {
			return nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return projectmutations.ErrNotFound
		}
		if err != nil {
			return err
		}
		if role == 5 && creator != cmd.ActorID.String() || cmd.Action == "delete" && status != 1 && role != 20 && creator != cmd.ActorID.String() {
			return projectmutations.ErrForbidden
		}
	}
	if duplicate, ok := fields["duplicate_to"].(string); ok {
		var id string
		err = tx.QueryRow(ctx, `SELECT i.id::text FROM issues i JOIN projects p ON p.id=i.project_id AND p.workspace_id=i.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=p.workspace_id AND pm.member_id=$2 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20) WHERE i.id=$1 AND `+djangoapi.IssueVisibleSQL("i", "$2::uuid", "$3::uuid")+` FOR SHARE OF i,p,pm`, duplicate, cmd.ActorID, cmd.WorkspaceID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return projectmutations.ErrNotFound
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func intakeMutationFields(action string, fields map[string]any) (map[string]any, error) {
	out := maps.Clone(fields)
	if out == nil {
		out = map[string]any{}
	}
	allowed := []string{}
	switch action {
	case "create":
		allowed = []string{"name", "description_html", "priority"}
	case "update":
		allowed = []string{"status", "snoozed_till", "duplicate_to", "source", "source_email"}
	case "delete":
	default:
		return nil, projectmutations.ErrInvalid
	}
	for key, value := range out {
		if !slices.Contains(allowed, key) {
			return nil, projectmutations.ErrInvalid
		}
		if key == "status" {
			status, ok := value.(int)
			if !ok || status < -2 || status > 2 {
				return nil, projectmutations.ErrInvalid
			}
			continue
		}
		text, ok := value.(string)
		if !ok || strings.ContainsRune(text, 0) {
			return nil, projectmutations.ErrInvalid
		}
		if key == "name" && (strings.TrimSpace(text) == "" || len([]rune(text)) > 255) {
			return nil, projectmutations.ErrInvalid
		}
		if key == "priority" && !slices.Contains([]string{"urgent", "high", "medium", "low", "none"}, text) {
			return nil, projectmutations.ErrInvalid
		}
		if key == "duplicate_to" {
			id, err := uuid.Parse(text)
			if err != nil || id == uuid.Nil {
				return nil, projectmutations.ErrInvalid
			}
			out[key] = id.String()
		}
	}
	if action == "create" && out["name"] == nil || action == "update" && len(out) == 0 {
		return nil, projectmutations.ErrInvalid
	}
	if status, ok := out["status"].(int); ok && (status == 0 && (out["snoozed_till"] == nil || out["snoozed_till"] == "") || status == 2 && out["duplicate_to"] == nil) {
		return nil, projectmutations.ErrInvalid
	}
	return out, nil
}
