package systemapi

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	userprofileapi "github.com/CaliLuke/go-plane/gen/user_profile_flags"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/userprofileflags"
	"github.com/CaliLuke/loom/security"
)

type userProfileFlagsRepository interface {
	UpdateOnboarded(context.Context, uuid.UUID, bool) error
	UpdateTourCompleted(context.Context, uuid.UUID, bool) error
}

type userProfileFlagsSessionValidator interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type userProfileFlagsService struct {
	repository userProfileFlagsRepository
	sessions   userProfileFlagsSessionValidator
}

func NewUserProfileFlags(repository userProfileFlagsRepository, sessions userProfileFlagsSessionValidator) (userprofileapi.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("user profile flags repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("session validator for Django is required")
	}
	return &userProfileFlagsService{repository: repository, sessions: sessions}, nil
}

func (service *userProfileFlagsService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	principal, err := service.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &userprofileapi.UserProfileFlagUnauthenticated{Detail: unauthenticatedDetail}
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return ctx, &userprofileapi.UserProfileFlagUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, userProfileFlagsPrincipalKey{}, userID), nil
}

func (service *userProfileFlagsService) UpdateOnboarded(ctx context.Context, payload *userprofileapi.UpdateOnboardedPayload) (*userprofileapi.UserProfileFlagResult, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	value := false
	if payload.IsOnboarded != nil {
		value = *payload.IsOnboarded
	}
	if err := service.repository.UpdateOnboarded(ctx, userID, value); err != nil {
		return nil, profileFlagError(err)
	}
	return &userprofileapi.UserProfileFlagResult{Message: "Updated successfully"}, nil
}

func (service *userProfileFlagsService) UpdateTourCompleted(ctx context.Context, payload *userprofileapi.UpdateTourCompletedPayload) (*userprofileapi.UserProfileFlagResult, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	value := false
	if payload.IsTourCompleted != nil {
		value = *payload.IsTourCompleted
	}
	if err := service.repository.UpdateTourCompleted(ctx, userID, value); err != nil {
		return nil, profileFlagError(err)
	}
	return &userprofileapi.UserProfileFlagResult{Message: "Updated successfully"}, nil
}

func (service *userProfileFlagsService) userID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(userProfileFlagsPrincipalKey{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &userprofileapi.UserProfileFlagUnauthenticated{Detail: unauthenticatedDetail}
	}
	return userID, nil
}

func profileFlagError(err error) error {
	if errors.Is(err, userprofileflags.ErrProfileNotFound) {
		return &userprofileapi.UserProfileFlagNotFound{Message: "The required object does not exist."}
	}
	return &userprofileapi.UserProfileFlagInternalError{Message: "Something went wrong please try again later"}
}

type userProfileFlagsPrincipalKey struct{}
