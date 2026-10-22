package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	api "github.com/CaliLuke/go-plane/gen/user_identity"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/useridentity"
	"github.com/CaliLuke/loom/security"
	"github.com/jackc/pgx/v5"
	"uuid"
)

type identitySessionValidator struct {
	principal string
	timezone  string
	err       error
}

func (validator identitySessionValidator) Validate(context.Context, string) (djangosession.Principal, error) {
	timezone := validator.timezone
	if timezone == "" {
		timezone = "UTC"
	}
	return djangosession.Principal{UserID: validator.principal, UserTimezone: timezone}, validator.err
}

func TestNewUserIdentityRejectsMissingDependencies(t *testing.T) {
	if _, err := NewUserIdentity(nil, identitySessionValidator{}); err == nil {
		t.Fatal("expected repository validation error")
	}
	repository, err := store.NewRepository(identityDatabase{row: identityJSONRowForService(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewUserIdentity(repository, nil); err == nil {
		t.Fatal("expected session validator validation error")
	}
}

func TestUserIdentitySessionUnauthenticated(t *testing.T) {
	repository, err := store.NewRepository(identityDatabase{row: identityJSONRowForService(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewUserIdentity(repository, identitySessionValidator{err: errors.New("invalid session")})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(interface {
		APIKeyAuth(context.Context, string, *security.APIKeyScheme) (context.Context, error)
	}).APIKeyAuth(context.Background(), "", &security.APIKeyScheme{})
	if err == nil || ctx == nil {
		t.Fatal("expected unauthenticated auth error")
	}
	result, err := service.Session(context.Background(), &api.SessionPayload{})
	if err != nil || result.IsAuthenticated || result.User.Present() {
		t.Fatalf("Session() = %#v, %v", result, err)
	}
}

func TestUserIdentitySessionRejectsUnknownTimezone(t *testing.T) {
	repository, err := store.NewRepository(identityDatabase{row: identityJSONRowForService(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewUserIdentity(repository, identitySessionValidator{
		principal: "00000000-0000-4000-8000-000000000b01",
		timezone:  "Mars/Olympus",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(interface {
		APIKeyAuth(context.Context, string, *security.APIKeyScheme) (context.Context, error)
	}).APIKeyAuth(context.Background(), "session-cookie", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Session(ctx, &api.SessionPayload{})
	var timezoneNotFound *api.UserIdentityTimezoneNotFound
	if !errors.As(err, &timezoneNotFound) || timezoneNotFound.Message != "The required key does not exist." {
		t.Fatalf("Session() error = %#v", err)
	}
}

func TestUserIdentitySessionMapsInvalidTimezoneToInternalError(t *testing.T) {
	repository, err := store.NewRepository(identityDatabase{row: identityJSONRowForService(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewUserIdentity(repository, identitySessionValidator{
		principal: "00000000-0000-4000-8000-000000000b01",
		timezone:  "../UTC",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(interface {
		APIKeyAuth(context.Context, string, *security.APIKeyScheme) (context.Context, error)
	}).APIKeyAuth(context.Background(), "session-cookie", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Session(ctx, &api.SessionPayload{})
	var internalError *api.UserIdentityInternalError
	if !errors.As(err, &internalError) || internalError.Message != userIdentityInternalErrorDetail {
		t.Fatalf("Session() error = %#v", err)
	}
}

func TestUserIdentityMeReturnsAuthenticatedSessionUser(t *testing.T) {
	body, err := json.Marshal(map[string]any{
		"id":                  "00000000-0000-4000-8000-000000000b01",
		"avatar":              nil,
		"avatar_url":          nil,
		"cover_image":         nil,
		"cover_image_url":     nil,
		"date_joined":         "2026-07-01T20:04:05.123456-07:00",
		"display_name":        "Current User",
		"email":               "user-me@example.com",
		"first_name":          "Current",
		"last_name":           "User",
		"is_active":           true,
		"is_bot":              false,
		"is_email_verified":   true,
		"user_timezone":       "America/Los_Angeles",
		"username":            "user-me@example.com",
		"is_password_autoset": true,
		"last_login_medium":   "magic-code",
		"last_login_time":     "2026-07-01T20:04:05.123456-07:00",
	})
	if err != nil {
		t.Fatal(err)
	}
	repository, err := store.NewRepository(identityDatabase{row: identityJSONRowForService(string(body))})
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewUserIdentity(repository, identitySessionValidator{principal: "00000000-0000-4000-8000-000000000b01"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(interface {
		APIKeyAuth(context.Context, string, *security.APIKeyScheme) (context.Context, error)
	}).APIKeyAuth(context.Background(), "session-cookie", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Me(ctx, &api.MePayload{})
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != uuid.MustParse("00000000-0000-4000-8000-000000000b01").String() ||
		result.DateJoined != "2026-07-01T20:04:05.123456-07:00" || result.UserTimezone != "America/Los_Angeles" {
		t.Fatalf("Me() = %#v", result)
	}
}

func TestUserIdentityUsesPrincipalResolvedByHTTPSessionLayer(t *testing.T) {
	repository, err := store.NewRepository(identityDatabase{row: identityJSONRowForService(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewUserIdentity(repository, identitySessionValidator{err: errors.New("must not validate again")})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := WithUserIdentitySessionInspection(context.Background(), "00000000-0000-4000-8000-000000000b01")
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := service.(interface {
		APIKeyAuth(context.Context, string, *security.APIKeyScheme) (context.Context, error)
	}).APIKeyAuth(ctx, "stale-cookie", &security.APIKeyScheme{})
	if err != nil || resolved == nil {
		t.Fatalf("APIKeyAuth() = %v, %v", resolved, err)
	}
	anonymous, err := WithUserIdentitySessionInspection(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.(interface {
		APIKeyAuth(context.Context, string, *security.APIKeyScheme) (context.Context, error)
	}).APIKeyAuth(anonymous, "otherwise-valid-cookie", &security.APIKeyScheme{})
	var unauthenticated *api.UserIdentityUnauthenticated
	if !errors.As(err, &unauthenticated) {
		t.Fatalf("anonymous APIKeyAuth() error = %v", err)
	}
}

func TestUserIdentityCapturesExactRepositoryJSON(t *testing.T) {
	const body = `{"id":"00000000-0000-4000-8000-000000000b01","avatar":null,"avatar_url":null,"cover_image":null,"cover_image_url":null,"date_joined":"2026-07-02T03:04:05Z","display_name":"R&D <ops>","email":null,"first_name":"","last_name":"","is_active":true,"is_bot":false,"is_email_verified":false,"user_timezone":"UTC","username":"user","is_password_autoset":false,"last_login_medium":null,"last_login_time":null}`
	repository, err := store.NewRepository(identityDatabase{row: identityJSONRowForService(body)})
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewUserIdentity(repository, identitySessionValidator{principal: "00000000-0000-4000-8000-000000000b01"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(interface {
		APIKeyAuth(context.Context, string, *security.APIKeyScheme) (context.Context, error)
	}).APIKeyAuth(context.Background(), "session-cookie", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	capture := &UserIdentityResponseCapture{}
	ctx = WithUserIdentityResponseCapture(ctx, capture)
	result, err := service.Me(ctx, &api.MePayload{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Email.Present() || !result.Email.IsNull() || string(capture.Data) != body {
		t.Fatalf("Me() email=%#v capture=%q", result.Email, capture.Data)
	}
}

// The adapter only needs pgx.Row for repository construction; this row also
// makes the dependency test deterministic without a live database.
type identityDatabase struct{ row pgx.Row }

func (database identityDatabase) QueryRow(context.Context, string, ...any) pgx.Row {
	return database.row
}

type identityServiceRow func(...any) error

func (row identityServiceRow) Scan(destinations ...any) error { return row(destinations...) }

func identityJSONRowForService(value string) pgx.Row {
	return identityServiceRow(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(value)
		if len(destinations) > 1 {
			*destinations[1].(*bool) = false
			*destinations[2].(*bool) = false
		}
		return nil
	})
}
