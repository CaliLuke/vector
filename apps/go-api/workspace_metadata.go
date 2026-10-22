package systemapi

import (
	"context"
	"encoding/json"
	"fmt"

	api "github.com/CaliLuke/go-plane/gen/workspace_metadata"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workspacemetadata"
	"github.com/CaliLuke/loom/security"
	"uuid"
)

type workspaceMetadataRepository interface {
	Authorized(context.Context, uuid.UUID, string) (bool, error)
	ListLabels(context.Context, uuid.UUID, string) (json.RawMessage, error)
	ListStates(context.Context, uuid.UUID, string) (json.RawMessage, error)
}

type workspaceMetadataSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type workspaceMetadataService struct {
	repository workspaceMetadataRepository
	sessions   workspaceMetadataSessions
}

type workspaceMetadataPrincipal struct{}

// NewWorkspaceMetadata constructs the workspace labels and states service.
func NewWorkspaceMetadata(repository workspaceMetadataRepository, sessions workspaceMetadataSessions) (api.Service, error) {
	if repository == nil || sessions == nil {
		return nil, fmt.Errorf("workspace metadata dependencies are required")
	}
	return &workspaceMetadataService{repository: repository, sessions: sessions}, nil
}

func (s *workspaceMetadataService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	principal, err := s.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &api.WorkspaceMetadataUnauthenticated{Detail: unauthenticatedDetail}
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return ctx, &api.WorkspaceMetadataUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, workspaceMetadataPrincipal{}, userID), nil
}

func (s *workspaceMetadataService) authorize(ctx context.Context, slug string) (uuid.UUID, error) {
	userID, ok := ctx.Value(workspaceMetadataPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.WorkspaceMetadataUnauthenticated{Detail: unauthenticatedDetail}
	}
	authorized, err := s.repository.Authorized(ctx, userID, slug)
	if err != nil {
		return uuid.Nil(), workspaceMetadataInternalError()
	}
	if !authorized {
		return uuid.Nil(), &api.WorkspaceMetadataForbidden{Detail: "You do not have the required permissions."}
	}
	return userID, nil
}

func (s *workspaceMetadataService) Labels(ctx context.Context, payload *api.LabelsPayload) ([]*api.WorkspaceLabel, error) {
	userID, err := s.authorize(ctx, payload.Slug)
	if err != nil {
		return nil, err
	}
	body, err := s.repository.ListLabels(ctx, userID, payload.Slug)
	if err != nil {
		return nil, workspaceMetadataInternalError()
	}
	var labels []*api.WorkspaceLabel
	if err := json.Unmarshal(body, &labels); err != nil {
		return nil, workspaceMetadataInternalError()
	}
	return labels, nil
}

func (s *workspaceMetadataService) States(ctx context.Context, payload *api.StatesPayload) ([]*api.WorkspaceState, error) {
	userID, err := s.authorize(ctx, payload.Slug)
	if err != nil {
		return nil, err
	}
	body, err := s.repository.ListStates(ctx, userID, payload.Slug)
	if err != nil {
		return nil, workspaceMetadataInternalError()
	}
	var states []*api.WorkspaceState
	if err := json.Unmarshal(body, &states); err != nil {
		return nil, workspaceMetadataInternalError()
	}
	return states, nil
}

func workspaceMetadataInternalError() error {
	return &api.WorkspaceMetadataInternal{Message: "Something went wrong please try again later"}
}

var _ workspaceMetadataRepository = (*store.Repository)(nil)
