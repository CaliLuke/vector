package workitems

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type itemEditState struct {
	command           itemEditCommand
	plan              workitemtypes.ItemTransition
	assets            []map[string]any
	publicationAssets []string
}

type itemEditReceipt struct {
	fingerprint        string
	committed, expired bool
}

func (r *Repository) PreviewItemChange(ctx context.Context, q Request, input ItemEditInput) (Result, error) {
	input.ReviewID = ""
	return r.reviewItemChange(ctx, q, input)
}
func (r *Repository) CommitItemChange(ctx context.Context, q Request, input ItemEditInput) (Result, error) {
	if _, err := uuid.Parse(input.ReviewID); err != nil {
		return Result{}, failure(400, "error", "Choose a valid review.")
	}
	return r.reviewItemChange(ctx, q, input)
}

func (r *Repository) reviewItemChange(ctx context.Context, q Request, input ItemEditInput) (Result, error) {
	command, raw, err := normalizedItemEdit(input)
	if err != nil {
		return Result{}, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	writer := *r
	writer.db, writer.transactionalEffects = tx, true
	q.API = false
	q.ProjectID, q.IssueID = command.ProjectID, command.ItemID
	q.Params = map[string]string{"pk": command.ItemID, "draft_id": command.ItemID}
	workspace, err := writer.lockItemEditWorkspace(ctx, q, command.Mode)
	if err != nil {
		return Result{}, err
	}
	receipt := itemEditReceipt{}
	if input.ReviewID != "" {
		receipt, err = loadItemEditReceipt(ctx, tx, workspace, q.ActorID, input.ReviewID, raw)
		if err != nil {
			return Result{}, err
		}
		if receipt.committed {
			return writer.itemEditProjection(ctx, q, command, workspace)
		}
		if receipt.expired {
			return Result{}, uncommittedItemReview("This review expired. Review the change again.")
		}
	}
	var revision int64
	if err = tx.QueryRow(ctx, `SELECT COALESCE((SELECT revision FROM work_item_type_settings WHERE workspace_id=$1),0)`, workspace).Scan(&revision); err != nil {
		return Result{}, err
	}
	if revision != command.Revision {
		if input.ReviewID != "" {
			return Result{}, uncommittedItemReview("The work item configuration changed. Reload and review again.")
		}
		return Result{}, failure(409, "error", "The work item configuration changed. Reload and review again.")
	}
	snapshot, err := writer.lockItemEditSource(ctx, q, command, workspace)
	if err != nil {
		return Result{}, err
	}
	if err = validateItemSourceVersion(command.ItemEditInput, snapshot); err != nil {
		return Result{}, err
	}
	savepoint, err := tx.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	operation := writer
	operation.db = savepoint
	operation.itemEdit = &itemEditState{command: command}
	if err = operation.applyItemEdit(ctx, q, command, snapshot); err != nil {
		return Result{}, err
	}
	if _, err = savepoint.Exec(ctx, `SET CONSTRAINTS ALL IMMEDIATE`); err != nil {
		return Result{}, err
	}
	fingerprint, err := itemEditFingerprint(snapshot, operation.itemEdit)
	if err != nil {
		return Result{}, err
	}
	if input.ReviewID == "" {
		if err = savepoint.Rollback(ctx); err != nil {
			return Result{}, err
		}
		id := uuid.NewString()
		if _, err = tx.Exec(ctx, `INSERT INTO work_item_schema_reviews(id,workspace_id,actor_id,command,fingerprint) VALUES($1,$2,$3,$4,$5)`, id, workspace, q.ActorID, raw, fingerprint); err != nil {
			return Result{}, err
		}
		review := ItemEditReview{ID: id, PreservedProperties: len(operation.itemEdit.plan.Preserved), Description: operation.itemEdit.plan.Description}
		return Result{200, review}, tx.Commit(ctx)
	}
	if fingerprint != receipt.fingerprint {
		return Result{}, uncommittedItemReview("The work item or its selections changed. Review the change again.")
	}
	if err = savepoint.Commit(ctx); err != nil {
		return Result{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE work_item_schema_reviews SET committed_at=clock_timestamp() WHERE id=$1 AND workspace_id=$2`, input.ReviewID, workspace); err != nil {
		return Result{}, err
	}
	result, err := writer.itemEditProjection(ctx, q, command, workspace)
	if err != nil {
		return Result{}, err
	}
	return result, tx.Commit(ctx)
}

func (r *Repository) lockItemEditWorkspace(ctx context.Context, q Request, mode string) (string, error) {
	roles := []int{5, 15, 20}
	if mode == "publish" {
		roles = []int{15, 20}
	}
	if err := r.allowWorkspace(ctx, q, roles); err != nil {
		return "", err
	}
	var workspace string
	if err := r.db.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug=$1 AND deleted_at IS NULL`, q.Slug).Scan(&workspace); err != nil {
		return "", err
	}
	if _, err := r.db.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, workspace); err != nil {
		return "", err
	}
	if err := r.allowWorkspace(ctx, q, roles); err != nil {
		return "", err
	}
	var current string
	if err := r.db.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug=$1 AND deleted_at IS NULL`, q.Slug).Scan(&current); err != nil {
		return "", err
	}
	if current != workspace {
		return "", failure(409, "error", "The workspace changed. Reload and try again.")
	}
	return workspace, nil
}

func loadItemEditReceipt(ctx context.Context, tx pgx.Tx, workspace, actor, id string, command []byte) (itemEditReceipt, error) {
	status := itemEditReceipt{}
	var matches bool
	err := tx.QueryRow(ctx, `SELECT fingerprint,committed_at IS NOT NULL,created_at<clock_timestamp()-interval '30 minutes',command=$4::jsonb FROM work_item_schema_reviews WHERE id=$1 AND workspace_id=$2 AND actor_id=$3 FOR UPDATE`, id, workspace, actor, command).Scan(&status.fingerprint, &status.committed, &status.expired, &matches)
	if errors.Is(err, pgx.ErrNoRows) {
		return status, requiredObject()
	}
	if err != nil {
		return status, err
	}
	if !matches {
		return status, failure(409, "error", "The proposed edit no longer matches this review.")
	}
	return status, nil
}

func itemEditFingerprint(snapshot map[string]any, state *itemEditState) (string, error) {
	value := map[string]any{"source": snapshot, "plan": state.plan, "assets": state.assets}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func itemEditBody(command itemEditCommand) *bytes.Reader {
	// normalizedItemEdit already verified that this is JSON encodable.
	data, _ := json.Marshal(command.Changes)
	return bytes.NewReader(data)
}

// Only call after an authorized, exact receipt lookup proves no commit occurred.
func uncommittedItemReview(message string) error {
	return &djangoapi.HTTPError{Status: 409, Body: map[string]any{"error": message, "review_again": true}}
}
