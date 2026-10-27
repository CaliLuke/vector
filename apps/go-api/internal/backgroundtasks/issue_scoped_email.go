package backgroundtasks

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Recheck scoped notifications at delivery, after queueing or retry delays.
// Legacy unbound activities keep their existing delivery policy. Filtering by
// each log identity avoids suppressing unrelated activities in a mixed batch.
func (store *issueEmailStore) filterScopedIssueEmailBatch(ctx context.Context, batch IssueEmailBatch) (IssueEmailBatch, error) {
	filtered := batch
	filtered.IDs = nil
	filtered.Activities = nil
	for i, activity := range batch.Activities {
		if i >= len(batch.IDs) {
			return IssueEmailBatch{}, fmt.Errorf("email activity has no log identity")
		}
		if activity.WorkspaceID != "" {
			var slug, email string
			err := store.pool.QueryRow(ctx, scopedMilestoneEmailRecipientSQL, batch.IssueID, activity.ProjectID, activity.WorkspaceID, batch.ReceiverID, batch.IDs[i]).Scan(&slug, &email)
			if errors.Is(err, pgx.ErrNoRows) {
				if markErr := store.markEmailBatchProcessed(ctx, []string{batch.IDs[i]}); markErr != nil {
					return IssueEmailBatch{}, markErr
				}
				continue
			}
			if err != nil {
				return IssueEmailBatch{}, err
			}
			filtered.WorkspaceSlug, filtered.ReceiverEmail = slug, email
		}
		filtered.IDs = append(filtered.IDs, batch.IDs[i])
		filtered.Activities = append(filtered.Activities, activity)
	}
	// Existing callers can construct metadata-only batches. They do not contain
	// a scoped activity and retain their previous handling.
	if len(batch.Activities) == 0 {
		return batch, nil
	}
	return filtered, nil
}

const scopedMilestoneEmailRecipientSQL = `SELECT workspace.slug,receiver.email
 FROM issues issue JOIN projects project ON project.id=issue.project_id AND project.workspace_id=issue.workspace_id
 JOIN workspaces workspace ON workspace.id=project.workspace_id JOIN users receiver ON receiver.id=$4
 WHERE issue.id=$1 AND issue.project_id=$2 AND issue.workspace_id=$3
 AND issue.deleted_at IS NULL AND issue.is_draft=false AND project.deleted_at IS NULL AND project.archived_at IS NULL AND workspace.deleted_at IS NULL AND receiver.is_active=true
 AND EXISTS(SELECT 1 FROM project_members pm JOIN workspace_members wm ON wm.member_id=pm.member_id AND wm.workspace_id=pm.workspace_id
 WHERE pm.project_id=project.id AND pm.workspace_id=workspace.id AND pm.member_id=receiver.id
 AND pm.is_active=true AND pm.deleted_at IS NULL AND wm.is_active=true AND wm.deleted_at IS NULL AND wm.role IN (5,15,20)
 AND (pm.role IN (15,20) OR pm.role=5 AND project.guest_view_all_features))
 AND EXISTS(SELECT 1 FROM email_notification_logs log WHERE log.id=$5 AND log.receiver_id=receiver.id AND log.entity_identifier=issue.id
 AND log.deleted_at IS NULL AND log.processed_at IS NULL AND log.sent_at IS NULL
 AND log.data#>>'{issue,workspace_id}'=$3::text AND log.data#>>'{issue,project_id}'=$2::text)
 AND COALESCE((SELECT property_change FROM user_notification_preferences WHERE user_id=receiver.id AND deleted_at IS NULL ORDER BY created_at DESC,id DESC LIMIT 1),true)`
