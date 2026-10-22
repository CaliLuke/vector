package systemapi

import (
	"context"
	"errors"
	"testing"
	"uuid"

	userprofileapi "github.com/CaliLuke/go-plane/gen/user_profile_flags"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/userprofileflags"
)

type userProfileFlagsRepositoryStub struct {
	onboarded, tour bool
	err             error
	userID          uuid.UUID
	value           bool
}

func (stub *userProfileFlagsRepositoryStub) UpdateOnboarded(_ context.Context, userID uuid.UUID, value bool) error {
	stub.userID, stub.value = userID, value
	return stub.err
}
func (stub *userProfileFlagsRepositoryStub) UpdateTourCompleted(_ context.Context, userID uuid.UUID, value bool) error {
	stub.userID, stub.value = userID, value
	return stub.err
}

type userProfileFlagsSessionStub struct {
	principal djangosession.Principal
	err       error
}

func (stub userProfileFlagsSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return stub.principal, stub.err
}

func TestUserProfileFlagsDefaultsAndResponses(t *testing.T) {
	repository := &userProfileFlagsRepositoryStub{}
	service, err := NewUserProfileFlags(repository, userProfileFlagsSessionStub{principal: djangosession.Principal{UserID: "00000000-0000-0000-0000-000000000001"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(userprofileapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.UpdateOnboarded(ctx, &userprofileapi.UpdateOnboardedPayload{})
	if err != nil || response == nil || response.Message != "Updated successfully" || repository.value {
		t.Fatalf("response = %#v, error = %v, value = %v", response, err, repository.value)
	}
	value := true
	response, err = service.UpdateTourCompleted(ctx, &userprofileapi.UpdateTourCompletedPayload{IsTourCompleted: &value})
	if err != nil || response == nil || response.Message != "Updated successfully" || !repository.value {
		t.Fatalf("response = %#v, error = %v, value = %v", response, err, repository.value)
	}
}

func TestUserProfileFlagsAuthenticationAndFailure(t *testing.T) {
	service, err := NewUserProfileFlags(&userProfileFlagsRepositoryStub{}, userProfileFlagsSessionStub{err: errors.New("invalid")})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.(userprofileapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if !containsLoomError(err, "unauthenticated") {
		t.Fatalf("error = %v", err)
	}
	service, err = NewUserProfileFlags(&userProfileFlagsRepositoryStub{err: errors.New("boom")}, userProfileFlagsSessionStub{principal: djangosession.Principal{UserID: "00000000-0000-0000-0000-000000000001"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(userprofileapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.UpdateOnboarded(ctx, &userprofileapi.UpdateOnboardedPayload{})
	if !containsLoomError(err, "internal_error") {
		t.Fatalf("error = %v", err)
	}
	service, err = NewUserProfileFlags(&userProfileFlagsRepositoryStub{err: userprofileflags.ErrProfileNotFound}, userProfileFlagsSessionStub{principal: djangosession.Principal{UserID: "00000000-0000-0000-0000-000000000001"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err = service.(userprofileapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.UpdateTourCompleted(ctx, &userprofileapi.UpdateTourCompletedPayload{})
	var notFound *userprofileapi.UserProfileFlagNotFound
	if !errors.As(err, &notFound) || notFound.Message != "The required object does not exist." {
		t.Fatalf("error = %v, want Django-compatible not found", err)
	}
}
