package systemapi

import (
	"context"
	"fmt"
	"uuid"

	userinstanceadminapi "github.com/CaliLuke/go-plane/gen/user_instance_admin"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/loom/security"
)

type userInstanceAdminRepository interface {
	IsInstanceAdmin(context.Context, uuid.UUID) (bool, error)
}

type userInstanceAdminSessionValidator interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type userInstanceAdminService struct {
	repository userInstanceAdminRepository
	sessions   userInstanceAdminSessionValidator
}

func NewUserInstanceAdmin(repository userInstanceAdminRepository, sessions userInstanceAdminSessionValidator) (userinstanceadminapi.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("user instance admin repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("session validator for Django is required")
	}
	return &userInstanceAdminService{repository: repository, sessions: sessions}, nil
}

func (service *userInstanceAdminService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	principal, err := service.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &userinstanceadminapi.UserInstanceAdminUnauthenticated{Detail: unauthenticatedDetail}
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return ctx, &userinstanceadminapi.UserInstanceAdminUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, userInstanceAdminPrincipalKey{}, userID), nil
}

func (service *userInstanceAdminService) Retrieve(ctx context.Context, _ *userinstanceadminapi.RetrievePayload) (*userinstanceadminapi.UserInstanceAdminResult, error) {
	userID, ok := ctx.Value(userInstanceAdminPrincipalKey{}).(uuid.UUID)
	if !ok {
		return nil, &userinstanceadminapi.UserInstanceAdminUnauthenticated{Detail: unauthenticatedDetail}
	}
	isAdmin, err := service.repository.IsInstanceAdmin(ctx, userID)
	if err != nil {
		return nil, &userinstanceadminapi.UserInstanceAdminInternalError{Message: "Something went wrong please try again later"}
	}
	return &userinstanceadminapi.UserInstanceAdminResult{IsInstanceAdmin: isAdmin}, nil
}

type userInstanceAdminPrincipalKey struct{}
