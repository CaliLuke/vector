package systemapi

import (
	"context"
	"fmt"
	"uuid"

	instanceworkspaceapi "github.com/CaliLuke/go-plane/gen/instance_workspace_slug_check"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/loom/security"
)

type instanceWorkspaceSlugSessionValidator interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type instanceWorkspaceSlugRepository interface {
	IsInstanceAdmin(context.Context, uuid.UUID) (bool, error)
	Exists(context.Context, string) (bool, error)
}

type instanceWorkspaceSlugService struct {
	repository instanceWorkspaceSlugRepository
	sessions   instanceWorkspaceSlugSessionValidator
}

func NewInstanceWorkspaceSlugCheck(repository instanceWorkspaceSlugRepository, sessions instanceWorkspaceSlugSessionValidator) (instanceworkspaceapi.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("instance workspace slug repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("session validator for Django is required")
	}
	return &instanceWorkspaceSlugService{repository: repository, sessions: sessions}, nil
}

func (service *instanceWorkspaceSlugService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	principal, err := service.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &instanceworkspaceapi.InstanceWorkspaceSlugUnauthenticated{Detail: unauthenticatedDetail}
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return ctx, &instanceworkspaceapi.InstanceWorkspaceSlugUnauthenticated{Detail: unauthenticatedDetail}
	}
	authorized, err := service.repository.IsInstanceAdmin(ctx, userID)
	if err != nil {
		return ctx, &instanceworkspaceapi.InstanceWorkspaceSlugInternalError{Message: "Something went wrong please try again later"}
	}
	if !authorized {
		return ctx, &instanceworkspaceapi.InstanceWorkspaceSlugForbidden{Detail: forbiddenDetail}
	}
	return ctx, nil
}

func (service *instanceWorkspaceSlugService) Check(ctx context.Context, payload *instanceworkspaceapi.CheckPayload) (*instanceworkspaceapi.InstanceWorkspaceSlugAvailability, error) {
	if payload.Slug == nil || *payload.Slug == "" {
		return nil, &instanceworkspaceapi.InstanceWorkspaceSlugError{Message: "Workspace Slug is required"}
	}
	exists, err := service.repository.Exists(ctx, *payload.Slug)
	if err != nil {
		return nil, &instanceworkspaceapi.InstanceWorkspaceSlugInternalError{Message: "Something went wrong please try again later"}
	}
	return &instanceworkspaceapi.InstanceWorkspaceSlugAvailability{Status: !exists && !restrictedWorkspaceSlug(*payload.Slug)}, nil
}
