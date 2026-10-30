package systemapi

import (
	"context"
	"fmt"
	"uuid"

	workspaceapi "github.com/CaliLuke/go-plane/gen/workspace_slug_check"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/workspacevalidation"
	"github.com/CaliLuke/loom/security"
)

type workspaceSlugSessionValidator interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type workspaceSlugService struct {
	repository workspaceSlugRepository
	sessions   workspaceSlugSessionValidator
}

type workspaceSlugRepository interface {
	Exists(context.Context, string) (bool, error)
}

func NewWorkspaceSlugCheck(repository workspaceSlugRepository, sessions workspaceSlugSessionValidator) (workspaceapi.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("workspace slug repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("session validator for Django is required")
	}
	return &workspaceSlugService{repository: repository, sessions: sessions}, nil
}

func (service *workspaceSlugService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	principal, err := service.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &workspaceapi.WorkspaceSlugUnauthenticated{Detail: unauthenticatedDetail}
	}
	if _, err := uuid.Parse(principal.UserID); err != nil {
		return ctx, &workspaceapi.WorkspaceSlugUnauthenticated{Detail: unauthenticatedDetail}
	}
	return ctx, nil
}

func (service *workspaceSlugService) Check(ctx context.Context, payload *workspaceapi.CheckPayload) (*workspaceapi.WorkspaceSlugAvailability, error) {
	if payload.Slug == nil || *payload.Slug == "" {
		return nil, &workspaceapi.WorkspaceSlugError{Message: "Workspace Slug is required"}
	}
	slug := *payload.Slug
	exists, err := service.repository.Exists(ctx, slug)
	if err != nil {
		return nil, &workspaceapi.WorkspaceSlugInternalError{Message: "Something went wrong please try again later"}
	}
	if restrictedWorkspaceSlug(slug) {
		exists = true
	}
	return &workspaceapi.WorkspaceSlugAvailability{Status: !exists}, nil
}

func restrictedWorkspaceSlug(slug string) bool {
	return workspacevalidation.RestrictedSlug(slug)
}
