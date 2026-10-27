package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func validateMilestoneActivityInput(input IssueActivityInput) error {
	requested := decodeActivityObject(input.RequestedData)
	_, milestone := requested["milestone"]
	if input.WorkspaceID != "" || milestone {
		id, err := uuid.Parse(input.WorkspaceID)
		if err != nil || id == uuid.Nil {
			return fmt.Errorf("%w: workspace ID is required for scoped activity", ErrPermanentIssueActivity)
		}
	}
	if !milestone {
		return nil
	}
	if input.Type != "issue.activity.updated" {
		return fmt.Errorf("%w: milestone changes require issue update activity", ErrPermanentIssueActivity)
	}
	_, err := milestoneActivityRow(input)
	return err
}

func milestoneActivityRow(input IssueActivityInput) (*issueActivityRow, error) {
	requested, current := decodeActivityObject(input.RequestedData), decodeActivityObject(input.CurrentInstance)
	if _, present := requested["milestone"]; !present {
		return nil, nil
	}
	oldID, oldTitle, err := milestoneActivitySnapshot(current)
	if err != nil {
		return nil, err
	}
	newID, newTitle, err := milestoneActivitySnapshot(requested)
	if err != nil {
		return nil, err
	}
	if sameActivityJSON(current["milestone"], requested["milestone"]) {
		return nil, nil
	}
	comment := "set the milestone to"
	if newID == nil {
		comment = "removed the milestone"
	}
	return &issueActivityRow{IssueID: input.IssueID, ActorID: input.ActorID, Verb: "updated", Field: "milestone",
		OldIdentifier: oldID, NewIdentifier: newID, OldValue: oldTitle, NewValue: newTitle, Comment: comment, Epoch: input.Epoch}, nil
}

func milestoneActivitySnapshot(data map[string]json.RawMessage) (*string, *string, error) {
	var id, title *string
	if err := json.Unmarshal(data["milestone"], &id); err != nil {
		return nil, nil, fmt.Errorf("%w: invalid milestone identifier snapshot", ErrPermanentIssueActivity)
	}
	if err := json.Unmarshal(data["milestone_title"], &title); err != nil || (id == nil) != (title == nil) {
		return nil, nil, fmt.Errorf("%w: invalid milestone title snapshot", ErrPermanentIssueActivity)
	}
	if id != nil {
		parsed, err := uuid.Parse(*id)
		if err != nil || parsed == uuid.Nil || *title == "" {
			return nil, nil, fmt.Errorf("%w: invalid milestone snapshot", ErrPermanentIssueActivity)
		}
	}
	return id, title, nil
}

// Scoped producers cannot attach an audit snapshot to a foreign or moved item.
// Lock in the same workspace/project/item order as the native writer. Recording
// updates the item's timestamp, so take its write lock without later promotion.
func lockBoundIssueActivity(ctx context.Context, tx pgx.Tx, input IssueActivityInput, workspaceID uuid.UUID, write bool) error {
	if input.WorkspaceID == "" {
		return nil
	}
	if uuid.MustParse(input.WorkspaceID) != workspaceID || input.IssueID == nil {
		return fmt.Errorf("%w: activity scope mismatch", ErrPermanentIssueActivity)
	}
	queries := []string{
		`SELECT id FROM workspaces WHERE id=$1 AND deleted_at IS NULL FOR SHARE`,
		`SELECT id FROM projects WHERE id=$2 AND workspace_id=$1 AND deleted_at IS NULL FOR SHARE`,
		`SELECT id FROM issues WHERE id=$3 AND project_id=$2 AND workspace_id=$1 AND deleted_at IS NULL AND is_draft=false FOR SHARE`,
	}
	if write {
		queries[2] = `SELECT id FROM issues WHERE id=$3 AND project_id=$2 AND workspace_id=$1 AND deleted_at IS NULL AND is_draft=false FOR UPDATE`
	}
	args := []any{workspaceID, input.ProjectID, *input.IssueID}
	for i, query := range queries {
		var id uuid.UUID
		err := tx.QueryRow(ctx, query, args[:i+1]...).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: activity scope is no longer available", ErrPermanentIssueActivity)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func notificationActivityValue(field string, value *string) string {
	if field == "milestone" && value == nil {
		return ""
	}
	return stringifiedNullable(value)
}
