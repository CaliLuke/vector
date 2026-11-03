package authworkflow

import (
	"context"
	"errors"

	"github.com/CaliLuke/go-plane/internal/projectmembershipreads"
)

// PostUserAuthWorkflow joins accepted invitations through the same transaction
// boundary as explicit acceptance. Membership reads use primary data directly.
type PostUserAuthWorkflow struct {
	invitations *projectmembershipreads.InvitationsRepository
}

func NewPostUserAuthWorkflow(invitations *projectmembershipreads.InvitationsRepository) (*PostUserAuthWorkflow, error) {
	if invitations == nil {
		return nil, errors.New("post-user-auth invitation repository is required")
	}
	return &PostUserAuthWorkflow{invitations: invitations}, nil
}

func (workflow *PostUserAuthWorkflow) Run(ctx context.Context, userID, _ string) error {
	return workflow.invitations.JoinAcceptedInvitations(ctx, userID)
}
