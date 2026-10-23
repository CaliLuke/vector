package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/user_identity"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/useridentity"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

const userIdentityInternalErrorDetail = "Something went wrong please try again later"

type userIdentityService struct {
	repository *store.Repository
	sessions   interface {
		Validate(context.Context, string) (djangosession.Principal, error)
	}
}

type userIdentityAuthentication struct {
	inspected    bool
	userID       uuid.UUID
	userTimezone string
}

type userIdentityAuthenticationKey struct{}

type UserIdentityResponseCapture struct{ Data json.RawMessage }

type userIdentityResponseCaptureKey struct{}

func NewUserIdentity(repository *store.Repository, sessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}) (api.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("user identity repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("Django session validator is required")
	}
	return &userIdentityService{repository: repository, sessions: sessions}, nil
}

func (service *userIdentityService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	if authentication, ok := ctx.Value(userIdentityAuthenticationKey{}).(userIdentityAuthentication); ok && authentication.inspected {
		if authentication.userID == uuid.Nil() {
			return ctx, &api.UserIdentityUnauthenticated{Detail: unauthenticatedDetail}
		}
		return ctx, nil
	}
	principal, err := service.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &api.UserIdentityUnauthenticated{Detail: unauthenticatedDetail}
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return ctx, &api.UserIdentityUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, userIdentityAuthenticationKey{}, userIdentityAuthentication{
		inspected:    true,
		userID:       userID,
		userTimezone: principal.UserTimezone,
	}), nil
}

// WithUserIdentitySessionInspection carries the authoritative outcome from the
// HTTP session-compatibility layer into Goa's generated security hook. An empty
// user ID is an inspected anonymous session, not a request to validate again.
func WithUserIdentitySessionInspection(ctx context.Context, userID string) (context.Context, error) {
	authentication := userIdentityAuthentication{inspected: true}
	if userID != "" {
		parsed, err := uuid.Parse(userID)
		if err != nil {
			return ctx, fmt.Errorf("parse user identity principal: %w", err)
		}
		authentication.userID = parsed
	}
	return context.WithValue(ctx, userIdentityAuthenticationKey{}, authentication), nil
}

func WithUserIdentityResponseCapture(ctx context.Context, capture *UserIdentityResponseCapture) context.Context {
	return context.WithValue(ctx, userIdentityResponseCaptureKey{}, capture)
}

func (service *userIdentityService) Session(ctx context.Context, _ *api.SessionPayload) (*api.UserSession, error) {
	authentication, ok := ctx.Value(userIdentityAuthenticationKey{}).(userIdentityAuthentication)
	if !ok || authentication.userID == uuid.Nil() {
		return &api.UserSession{IsAuthenticated: false}, nil
	}
	if err := djangosession.ValidateUserTimezone(authentication.userTimezone); err != nil {
		if errors.Is(err, djangosession.ErrTimezoneNotFound) {
			return nil, &api.UserIdentityTimezoneNotFound{Message: "The required key does not exist."}
		}
		return nil, &api.UserIdentityInternalError{Message: userIdentityInternalErrorDetail}
	}
	body, err := service.repository.SessionUser(ctx, authentication.userID)
	if errors.Is(err, store.ErrNotFound) {
		return &api.UserSession{IsAuthenticated: false}, nil
	}
	if err != nil {
		return nil, &api.UserIdentityInternalError{Message: userIdentityInternalErrorDetail}
	}
	user, err := decodeSessionUser(body)
	if err != nil {
		return nil, err
	}
	return &api.UserSession{IsAuthenticated: true, User: loom.NullableValue(*user)}, nil
}

func (service *userIdentityService) Me(ctx context.Context, _ *api.MePayload) (*api.SessionUser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	body, err := service.repository.SessionUser(ctx, userID)
	if err != nil {
		return nil, &api.UserIdentityInternalError{Message: userIdentityInternalErrorDetail}
	}
	service.captureSessionUser(ctx, body)
	return decodeSessionUser(body)
}

func (service *userIdentityService) Profile(ctx context.Context, _ *api.ProfilePayload) (*api.UserProfile, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	body, err := service.repository.Profile(ctx, userID)
	if err != nil {
		return nil, identityRepositoryError(err)
	}
	return decodeUserProfile(body)
}

func (service *userIdentityService) UpdateProfile(ctx context.Context, payload *api.UpdateProfilePayload) (*api.UserProfile, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(payload.Body)
	if err != nil {
		return nil, &api.UserIdentityInternalError{Message: userIdentityInternalErrorDetail}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, &api.UserIdentityInternalError{Message: userIdentityInternalErrorDetail}
	}
	if invalid := normalizeJSONFields(fields, "theme", "onboarding_step", "billing_address", "mobile_onboarding_step", "goals", "product_tour"); invalid != nil {
		if _, lookupErr := service.repository.Profile(ctx, userID); lookupErr != nil {
			return nil, identityRepositoryError(lookupErr)
		}
		return nil, api.JSONFieldValidation(invalid)
	}
	body, err = json.Marshal(fields)
	if err != nil {
		return nil, &api.UserIdentityInternalError{Message: userIdentityInternalErrorDetail}
	}
	updated, err := service.repository.UpdateProfile(ctx, userID, body)
	if err != nil {
		return nil, identityRepositoryError(err)
	}
	return decodeUserProfile(updated)
}

func (service *userIdentityService) userID(ctx context.Context) (uuid.UUID, error) {
	authentication, ok := ctx.Value(userIdentityAuthenticationKey{}).(userIdentityAuthentication)
	if !ok || authentication.userID == uuid.Nil() {
		return uuid.Nil(), &api.UserIdentityUnauthenticated{Detail: unauthenticatedDetail}
	}
	return authentication.userID, nil
}

func (service *userIdentityService) captureSessionUser(ctx context.Context, body json.RawMessage) {
	if capture, ok := ctx.Value(userIdentityResponseCaptureKey{}).(*UserIdentityResponseCapture); ok && capture != nil {
		capture.Data = append(capture.Data[:0], body...)
	}
}

func decodeUserProfile(body json.RawMessage) (*api.UserProfile, error) {
	var profile api.UserProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, &api.UserIdentityInternalError{Message: userIdentityInternalErrorDetail}
	}
	return &profile, nil
}

func decodeSessionUser(body json.RawMessage) (*api.SessionUser, error) {
	var user api.SessionUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, &api.UserIdentityInternalError{Message: userIdentityInternalErrorDetail}
	}
	return &user, nil
}

func identityRepositoryError(err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return &api.UserIdentityNotFound{Message: "The required object does not exist."}
	}
	return &api.UserIdentityInternalError{Message: userIdentityInternalErrorDetail}
}
