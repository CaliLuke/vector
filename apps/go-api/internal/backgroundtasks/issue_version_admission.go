package backgroundtasks

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// IssueDescriptionVersionAdmission adapts the work-item request hook to the
// typed durable workflow. The caller supplies a per-operation identity so
// retries of one admission reconcile without suppressing later identical work.
type IssueDescriptionVersionAdmission struct {
	dispatcher *VersionHistoryDispatcher
}

func NewIssueDescriptionVersionAdmission(dispatcher *VersionHistoryDispatcher) (*IssueDescriptionVersionAdmission, error) {
	if dispatcher == nil {
		return nil, fmt.Errorf("version history dispatcher is required")
	}
	return &IssueDescriptionVersionAdmission{dispatcher: dispatcher}, nil
}

func (admission *IssueDescriptionVersionAdmission) AdmitIssueDescriptionVersion(ctx context.Context, eventID, updatedIssue, issueID, userID string, isCreating bool) error {
	if admission == nil || admission.dispatcher == nil {
		return fmt.Errorf("native issue description version admission is not configured")
	}
	if _, err := uuid.Parse(eventID); err != nil {
		return fmt.Errorf("issue description version operation ID must be a UUID: %w", err)
	}
	input := IssueDescriptionVersionInput{
		IssueID:      issueID,
		UpdatedIssue: updatedIssue,
		UserID:       userID,
		IsCreating:   isCreating,
	}
	if _, err := admission.dispatcher.AdmitIssueDescription(ctx, eventID, input); err != nil {
		return fmt.Errorf("admit issue description version workflow: %w", err)
	}
	return nil
}
