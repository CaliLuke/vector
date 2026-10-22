package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	usernotificationapi "github.com/CaliLuke/go-plane/gen/user_notification_preferences"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	usernotificationstore "github.com/CaliLuke/go-plane/internal/usernotificationpreferences"
	loom "github.com/CaliLuke/loom/pkg"
)

const notificationPreferenceJSON = `{"id":"00000000-0000-4000-8000-000000000011","created_by":null,"updated_by":null,"created_at":"2026-08-26T21:00:00Z","updated_at":"2026-08-26T21:00:00Z","deleted_at":null,"user":"00000000-0000-4000-8000-000000000010","workspace":null,"project":null,"property_change":true,"state_change":true,"comment":true,"mention":true,"issue_completed":true}`

type notificationPreferencesRepositoryStub struct {
	values map[string]any
	err    error
}

func (stub *notificationPreferencesRepositoryStub) Get(context.Context, uuid.UUID) (json.RawMessage, error) {
	return json.RawMessage(notificationPreferenceJSON), stub.err
}
func (stub *notificationPreferencesRepositoryStub) GetByID(context.Context, uuid.UUID) (json.RawMessage, error) {
	return json.RawMessage(notificationPreferenceJSON), stub.err
}
func (stub *notificationPreferencesRepositoryStub) Patch(_ context.Context, _ uuid.UUID, values map[string]any) (uuid.UUID, error) {
	stub.values = values
	return uuid.MustParse("00000000-0000-4000-8000-000000000011"), stub.err
}

type notificationPreferencesSessionStub struct{ err error }

func (stub notificationPreferencesSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{UserID: "00000000-0000-4000-8000-000000000010"}, stub.err
}

func notificationPreferencesContext(t *testing.T, service usernotificationapi.Service) context.Context {
	t.Helper()
	ctx, err := service.(usernotificationapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TestUserNotificationPreferencesGetAndBooleanPatch(t *testing.T) {
	repository := &notificationPreferencesRepositoryStub{}
	service, err := NewUserNotificationPreferences(repository, notificationPreferencesSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := notificationPreferencesContext(t, service)
	result, err := service.Get(ctx, &usernotificationapi.GetPayload{})
	if err != nil || result.ID == "" || !result.PropertyChange {
		t.Fatalf("Get() = %#v, %v", result, err)
	}
	payload := &usernotificationapi.PatchPayload{Body: &usernotificationapi.UserNotificationPreferencePatch{
		PropertyChange: loom.NullableValue(loom.JSONValue(`"false"`)),
		IssueCompleted: loom.NullableValue(loom.JSONValue(`1`)),
	}}
	result, err = service.Patch(ctx, payload)
	if err != nil || result.ID == "" || repository.values["property_change"] != false || repository.values["issue_completed"] != true {
		t.Fatalf("Patch() = %#v, %v, values %#v", result, err, repository.values)
	}
}

func TestUserNotificationPreferencesValidationAndErrors(t *testing.T) {
	repository := &notificationPreferencesRepositoryStub{}
	service, _ := NewUserNotificationPreferences(repository, notificationPreferencesSessionStub{})
	ctx := notificationPreferencesContext(t, service)
	_, err := service.Patch(ctx, &usernotificationapi.PatchPayload{Body: &usernotificationapi.UserNotificationPreferencePatch{
		PropertyChange: loom.NullValue[loom.JSONValue](),
	}})
	var validation *usernotificationapi.UserNotificationPreferenceValidationError
	if !errors.As(err, &validation) || len(validation.PropertyChange) != 1 {
		t.Fatalf("null boolean error = %#v", err)
	}

	repository.err = usernotificationstore.ErrNotFound
	_, err = service.Get(ctx, &usernotificationapi.GetPayload{})
	var notFound *usernotificationapi.UserNotificationPreferenceError
	if !errors.As(err, &notFound) || notFound.Message != "The required object does not exist." {
		t.Fatalf("not found error = %#v", err)
	}

	service, _ = NewUserNotificationPreferences(&notificationPreferencesRepositoryStub{}, notificationPreferencesSessionStub{err: errors.New("bad session")})
	_, err = service.(usernotificationapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	var unauthenticated *usernotificationapi.UserNotificationPreferenceUnauthenticated
	if !errors.As(err, &unauthenticated) {
		t.Fatalf("auth error = %#v", err)
	}
}
