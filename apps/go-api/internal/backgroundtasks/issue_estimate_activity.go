package backgroundtasks

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Estimate history resolves retained catalogue rows so queued removals keep
// their labels after soft deletion. References remain bound to the event project.
func estimateIssueActivityRows(ctx context.Context, tx pgx.Tx, event issueActivityEvent) ([]issueActivityRow, error) {
	requested, present := event.requested["estimate_point"]
	if !present || sameActivityJSON(event.current["estimate_point"], requested) {
		return nil, nil
	}
	oldPoint, err := issueActivityEstimatePoint(ctx, tx, event.ProjectID, activityText(event.current["estimate_point"]))
	if err != nil {
		return nil, err
	}
	newPoint, err := issueActivityEstimatePoint(ctx, tx, event.ProjectID, activityText(requested))
	if err != nil {
		return nil, err
	}
	if oldPoint == nil && newPoint == nil {
		return nil, nil
	}
	row := event.baseRow()
	row.Verb = "removed"
	row.Comment = "updated the estimate point to "
	if oldPoint != nil {
		row.OldIdentifier, row.OldValue = &oldPoint.id, &oldPoint.value
		row.Field = "estimate_" + oldPoint.kind
	}
	if newPoint != nil {
		row.NewIdentifier, row.NewValue = &newPoint.id, &newPoint.value
		row.Field = "estimate_" + newPoint.kind
		row.Verb = "updated"
	}
	return []issueActivityRow{row}, nil
}

type estimateActivityPoint struct{ id, value, kind string }

func issueActivityEstimatePoint(ctx context.Context, tx pgx.Tx, project string, id *string) (*estimateActivityPoint, error) {
	if id == nil || *id == "" {
		return nil, nil
	}
	if _, err := uuid.Parse(*id); err != nil {
		return nil, fmt.Errorf("%w: invalid estimate point identifier", ErrPermanentIssueActivity)
	}
	var point estimateActivityPoint
	err := tx.QueryRow(ctx, `SELECT ep.id::text,ep.value,e.type FROM estimate_points ep
 JOIN estimates e ON e.id=ep.estimate_id AND e.project_id=ep.project_id AND e.workspace_id=ep.workspace_id
 JOIN projects p ON p.id=ep.project_id AND p.workspace_id=ep.workspace_id
 WHERE ep.id=$1 AND p.id=$2`, *id, project).Scan(&point.id, &point.value, &point.kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load issue estimate point: %w", err)
	}
	return &point, nil
}
