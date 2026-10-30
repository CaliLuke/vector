package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"uuid"

	usernotificationapi "github.com/CaliLuke/go-plane/gen/user_notification_preferences"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	usernotificationstore "github.com/CaliLuke/go-plane/internal/usernotificationpreferences"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type userNotificationPreferencesRepository interface {
	Get(context.Context, uuid.UUID) (json.RawMessage, error)
	GetByID(context.Context, uuid.UUID) (json.RawMessage, error)
	Patch(context.Context, uuid.UUID, map[string]any) (uuid.UUID, error)
}
type userNotificationPreferencesSessionValidator interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type userNotificationPreferencesService struct {
	repository userNotificationPreferencesRepository
	sessions   userNotificationPreferencesSessionValidator
}

func NewUserNotificationPreferences(repository userNotificationPreferencesRepository, sessions userNotificationPreferencesSessionValidator) (usernotificationapi.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("user notification preferences repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("session validator for Django is required")
	}
	return &userNotificationPreferencesService{repository: repository, sessions: sessions}, nil
}

func (service *userNotificationPreferencesService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	principal, err := service.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &usernotificationapi.UserNotificationPreferenceUnauthenticated{Detail: unauthenticatedDetail}
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return ctx, &usernotificationapi.UserNotificationPreferenceUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, userNotificationPreferencesPrincipalKey{}, userID), nil
}

func (service *userNotificationPreferencesService) Get(ctx context.Context, _ *usernotificationapi.GetPayload) (*usernotificationapi.UserNotificationPreference, error) {
	return service.get(ctx)
}
func (service *userNotificationPreferencesService) Patch(ctx context.Context, payload *usernotificationapi.PatchPayload) (*usernotificationapi.UserNotificationPreference, error) {
	userID, ok := ctx.Value(userNotificationPreferencesPrincipalKey{}).(uuid.UUID)
	if !ok {
		return nil, &usernotificationapi.UserNotificationPreferenceUnauthenticated{Detail: unauthenticatedDetail}
	}
	values := map[string]any{}
	if payload.Body != nil {
		if err := addNullable(values, "created_by", payload.Body.CreatedBy); err != nil {
			return nil, err
		}
		if err := addNullable(values, "updated_by", payload.Body.UpdatedBy); err != nil {
			return nil, err
		}
		if err := addNullable(values, "deleted_at", payload.Body.DeletedAt); err != nil {
			return nil, err
		}
		if err := addNullable(values, "user", payload.Body.User); err != nil {
			return nil, err
		}
		if err := addNullable(values, "workspace", payload.Body.Workspace); err != nil {
			return nil, err
		}
		if err := addNullable(values, "project", payload.Body.Project); err != nil {
			return nil, err
		}
		if err := addNullable(values, "property_change", payload.Body.PropertyChange); err != nil {
			return nil, err
		}
		if err := addNullable(values, "state_change", payload.Body.StateChange); err != nil {
			return nil, err
		}
		if err := addNullable(values, "comment", payload.Body.Comment); err != nil {
			return nil, err
		}
		if err := addNullable(values, "mention", payload.Body.Mention); err != nil {
			return nil, err
		}
		if err := addNullable(values, "issue_completed", payload.Body.IssueCompleted); err != nil {
			return nil, err
		}
	}
	preferenceID, err := service.repository.Patch(ctx, userID, values)
	if err != nil {
		var fieldError *usernotificationstore.FieldError
		if errors.As(err, &fieldError) {
			return nil, preferenceValidationError(fieldError.Field, fieldError.Message)
		}
		if errors.Is(err, usernotificationstore.ErrNotFound) {
			return nil, &usernotificationapi.UserNotificationPreferenceError{Message: "The required object does not exist."}
		}
		return nil, profilePreferenceInternalError()
	}
	data, err := service.repository.GetByID(ctx, preferenceID)
	if err != nil {
		return nil, profilePreferenceInternalError()
	}
	var result usernotificationapi.UserNotificationPreference
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, profilePreferenceInternalError()
	}
	return &result, nil
}

func preferenceValidationError(field, message string) error {
	validation := &usernotificationapi.UserNotificationPreferenceValidationError{}
	switch field {
	case "property_change":
		validation.PropertyChange = []string{message}
	case "state_change":
		validation.StateChange = []string{message}
	case "comment":
		validation.Comment = []string{message}
	case "mention":
		validation.Mention = []string{message}
	case "issue_completed":
		validation.IssueCompleted = []string{message}
	case "created_by":
		validation.CreatedBy = []string{message}
	case "updated_by":
		validation.UpdatedBy = []string{message}
	case "deleted_at":
		validation.DeletedAt = []string{message}
	case "user":
		validation.User = []string{message}
	case "workspace":
		validation.Workspace = []string{message}
	case "project":
		validation.Project = []string{message}
	default:
		return profilePreferenceInternalError()
	}
	return validation
}

func addNullable[T any](values map[string]any, name string, value loom.Nullable[T]) error {
	if !value.Present() {
		return nil
	}
	if value.IsNull() {
		if booleanName(name) {
			return preferenceValidationError(name, "This field may not be null.")
		}
		values[name] = nil
		return nil
	}
	if concrete, ok := value.Value(); ok {
		if booleanName(name) {
			boolean, valid := normalizeDRFBoolean(concrete)
			if !valid {
				error := &usernotificationapi.UserNotificationPreferenceValidationError{}
				switch name {
				case "property_change":
					error.PropertyChange = []string{"Must be a valid boolean."}
				case "state_change":
					error.StateChange = []string{"Must be a valid boolean."}
				case "comment":
					error.Comment = []string{"Must be a valid boolean."}
				case "mention":
					error.Mention = []string{"Must be a valid boolean."}
				case "issue_completed":
					error.IssueCompleted = []string{"Must be a valid boolean."}
				}
				return error
			}
			values[name] = boolean
			return nil
		}
		values[name] = concrete
	}
	return nil
}

func booleanName(name string) bool {
	return name == "property_change" || name == "state_change" || name == "comment" || name == "mention" || name == "issue_completed"
}

func normalizeDRFBoolean(value any) (bool, bool) {
	if boolean, ok := value.(bool); ok {
		return boolean, true
	}
	switch value := value.(type) {
	case loom.JSONValue:
		var decoded any
		if err := json.Unmarshal(value, &decoded); err != nil {
			return false, false
		}
		return normalizeDRFBoolean(decoded)
	case float64:
		if value == 0 {
			return false, true
		}
		if value == 1 {
			return true, true
		}
	case string:
		switch value {
		case "t", "T", "true", "True", "TRUE", "y", "Y", "yes", "Yes", "YES", "on", "On", "ON", "1":
			return true, true
		case "f", "F", "false", "False", "FALSE", "n", "N", "no", "No", "NO", "off", "Off", "OFF", "0":
			return false, true
		}
	}
	return false, false
}
func (service *userNotificationPreferencesService) get(ctx context.Context) (*usernotificationapi.UserNotificationPreference, error) {
	userID, ok := ctx.Value(userNotificationPreferencesPrincipalKey{}).(uuid.UUID)
	if !ok {
		return nil, &usernotificationapi.UserNotificationPreferenceUnauthenticated{Detail: unauthenticatedDetail}
	}
	data, err := service.repository.Get(ctx, userID)
	if errors.Is(err, usernotificationstore.ErrNotFound) {
		return nil, &usernotificationapi.UserNotificationPreferenceError{Message: "The required object does not exist."}
	}
	if err != nil {
		return nil, profilePreferenceInternalError()
	}
	var result usernotificationapi.UserNotificationPreference
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, profilePreferenceInternalError()
	}
	return &result, nil
}
func profilePreferenceInternalError() error {
	return &usernotificationapi.UserNotificationPreferenceInternalError{Message: "Something went wrong please try again later"}
}

type userNotificationPreferencesPrincipalKey struct{}
