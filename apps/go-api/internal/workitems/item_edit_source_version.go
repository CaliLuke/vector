package workitems

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
)

// ItemEditSource is the canonical source for a saved-item form. The version is
// opaque: it covers values and active relationships, even for legacy writers
// that do not change updated_at. Proposed destination selections are excluded.
type ItemEditSource struct {
	Item          any    `json:"item"`
	SourceVersion string `json:"source_version"`
}

func (r *Repository) ReadItemEditSource(ctx context.Context, q Request, mode, item, project string) (_ Result, resultErr error) {
	if mode != "edit" && mode != "draft" {
		return Result{}, failure(400, "error", "Choose a saved work item or private draft.")
	}
	id, err := uuid.Parse(item)
	if err != nil {
		return Result{}, failure(400, "error", "Choose a valid work item or draft.")
	}
	if mode == "edit" {
		parsed, parseErr := uuid.Parse(project)
		if parseErr != nil {
			return Result{}, failure(400, "error", "Choose the work item's project.")
		}
		project = parsed.String()
	} else {
		project = ""
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	if _, err = tx.Exec(ctx, `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY`); err != nil {
		return Result{}, err
	}
	reader := *r
	reader.db = tx
	q.API = false
	q.ProjectID, q.IssueID = project, id.String()
	command := itemEditCommand{ItemEditInput: ItemEditInput{Mode: mode, ProjectID: project, ItemID: id.String(), Changes: map[string]any{}}}
	// This read uses a consistent snapshot without write/advisory locks. A writer
	// may commit concurrently; the locked preview rejects a superseded version.
	if err = reader.allowWorkspace(ctx, q, []int{5, 15, 20}); err != nil {
		return Result{}, err
	}
	var workspace string
	err = tx.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug=$1 AND deleted_at IS NULL`, q.Slug).Scan(&workspace)
	if err != nil {
		return Result{}, err
	}
	snapshot, err := reader.readItemSourceSnapshot(ctx, q, command, workspace)
	if err != nil {
		return Result{}, err
	}
	version, err := itemSourceVersion(snapshot)
	if err != nil {
		return Result{}, err
	}
	projection, err := reader.itemEditProjection(ctx, q, command, workspace)
	if err != nil {
		return Result{}, err
	}
	return Result{200, ItemEditSource{Item: projection.Body, SourceVersion: version}}, tx.Commit(ctx)
}

// Keep the raw row and relationship identities in the same snapshot as the
// form projection. Neither container metadata nor a proposed destination is
// part of the saved source identity.
func (r *Repository) readItemSourceSnapshot(ctx context.Context, q Request, command itemEditCommand, workspace string) (map[string]any, error) {
	query := `SELECT to_jsonb(d) FROM draft_issues d WHERE d.id=$1 AND d.workspace_id=$2 AND d.created_by_id=$3 AND d.deleted_at IS NULL`
	args := []any{q.IssueID, workspace, q.ActorID}
	if command.Mode == "edit" {
		if err := r.allowProject(ctx, q, []int{15, 20}, true); err != nil {
			return nil, err
		}
		if _, err := r.row(ctx, `SELECT to_jsonb(p) FROM projects p WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND archived_at IS NULL`, q.ProjectID, workspace); err != nil {
			return nil, err
		}
		query = `SELECT to_jsonb(i) FROM issues i WHERE i.id=$1 AND i.workspace_id=$2 AND i.project_id=$3 AND ` + liveIssue
		args[2] = q.ProjectID
	}
	row, err := r.row(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	relations := map[string]any{}
	for _, kind := range []string{"cycle", "module", "assignee", "label"} {
		table, column := itemEditRelationTable(command.Mode, kind)
		links, readErr := r.rows(ctx, `SELECT to_jsonb(link) FROM `+table+` link WHERE `+column+`=$1 AND deleted_at IS NULL ORDER BY id`, command.ItemID)
		if readErr != nil {
			return nil, readErr
		}
		relations[kind] = links
	}
	return map[string]any{"item": row, "relations": relations}, nil
}

func itemSourceVersion(snapshot map[string]any) (string, error) {
	links := map[string][]string{}
	relations := snapshot["relations"].(map[string]any)
	for _, kind := range []string{"cycle", "module", "assignee", "label"} {
		ids := []string{}
		for _, row := range relations[kind].([]map[string]any) {
			ids = append(ids, stringValue(row[kind+"_id"]))
		}
		slices.Sort(ids)
		links[kind] = slices.Compact(ids)
	}
	raw, err := json.Marshal(map[string]any{"item": snapshot["item"], "relations": links})
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:]), nil
}

func validateItemSourceVersion(input ItemEditInput, snapshot map[string]any) error {
	// Existing receipts without a precondition remain replayable. No new preview
	// can be issued without a version; committed receipts are returned earlier.
	if input.ExpectedSourceVersion == "" && input.ReviewID != "" {
		return nil
	}
	current, err := itemSourceVersion(snapshot)
	if err != nil {
		return err
	}
	if current == input.ExpectedSourceVersion {
		return nil
	}
	fields := map[string]any{"error": "This work item changed while the form was open. Reload the saved item before editing again.", "source_changed": true}
	if input.ReviewID != "" {
		fields["review_again"] = true
	}
	return &djangoapi.HTTPError{Status: 409, Body: fields}
}
