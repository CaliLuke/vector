package systemapi

import (
	"context"
	"encoding/json"
	"fmt"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/user_settings"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/usersettings"
	"github.com/CaliLuke/loom/security"
)

type userSettingsRepository interface {
	Get(context.Context, uuid.UUID) (json.RawMessage, error)
}

type userSettingsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type userSettingsService struct {
	repository userSettingsRepository
	sessions   userSettingsSessions
}

type userSettingsPrincipal struct{}

// NewUserSettings constructs the authenticated user settings service.
func NewUserSettings(repository userSettingsRepository, sessions userSettingsSessions) (api.Service, error) {
	if repository == nil || sessions == nil {
		return nil, fmt.Errorf("user settings dependencies are required")
	}
	return &userSettingsService{repository: repository, sessions: sessions}, nil
}

func (s *userSettingsService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	principal, err := s.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &api.UserSettingsUnauthenticated{Detail: unauthenticatedDetail}
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return ctx, &api.UserSettingsUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, userSettingsPrincipal{}, userID), nil
}

func (s *userSettingsService) Get(ctx context.Context, _ *api.GetPayload) (*api.UserSettings, error) {
	userID, ok := ctx.Value(userSettingsPrincipal{}).(uuid.UUID)
	if !ok {
		return nil, &api.UserSettingsUnauthenticated{Detail: unauthenticatedDetail}
	}
	body, err := s.repository.Get(ctx, userID)
	if err != nil {
		return nil, userSettingsInternalError()
	}
	var result api.UserSettings
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, userSettingsInternalError()
	}
	return &result, nil
}

func userSettingsInternalError() error {
	return &api.UserSettingsInternal{Message: "Something went wrong please try again later"}
}

var _ userSettingsRepository = (*store.Repository)(nil)
