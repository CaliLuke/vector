package systemapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"uuid"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	loomhttp "github.com/CaliLuke/loom/http"
)

type serviceSessionValidatorStub struct {
	principal  djangosession.Principal
	err        error
	calls      int
	credential string
}

func (stub *serviceSessionValidatorStub) Validate(_ context.Context, credential string) (djangosession.Principal, error) {
	stub.calls++
	stub.credential = credential
	return stub.principal, stub.err
}

func TestServiceSessionIdentityUsesCachedPrincipalWithoutValidation(t *testing.T) {
	validator := &serviceSessionValidatorStub{}
	ctx := WithReadServiceSessionPrincipal(context.Background(), "cached-user", "America/Los_Angeles")

	userID, timezone, err := serviceSessionIdentity(ctx, "ignored-credential", validator)
	if err != nil || userID != "cached-user" || timezone != "America/Los_Angeles" {
		t.Fatalf("identity=(%q, %q) err=%v", userID, timezone, err)
	}
	if validator.calls != 0 {
		t.Fatalf("Validate called %d times for cached principal", validator.calls)
	}
}

func TestServiceSessionIdentityDoesNotFallbackFromCachedEmptyPrincipal(t *testing.T) {
	validator := &serviceSessionValidatorStub{principal: djangosession.Principal{UserID: "fallback-user", UserTimezone: "UTC"}}
	ctx := WithReadServiceSessionPrincipal(context.Background(), "", "")

	userID, timezone, err := serviceSessionIdentityFromRequest(ctx, "", validator)
	if err != nil || userID != "" || timezone != "" {
		t.Fatalf("identity=(%q, %q) err=%v", userID, timezone, err)
	}
	if validator.calls != 0 {
		t.Fatalf("Validate called %d times for inspected empty principal", validator.calls)
	}
}

func TestServiceSessionIdentityValidatesCredentialAndPreservesResult(t *testing.T) {
	validator := &serviceSessionValidatorStub{principal: djangosession.Principal{
		UserID: "validated-user", UserTimezone: "Asia/Kolkata",
	}}
	userID, timezone, err := serviceSessionIdentity(context.Background(), "session-token", validator)
	if err != nil || userID != "validated-user" || timezone != "Asia/Kolkata" {
		t.Fatalf("identity=(%q, %q) err=%v", userID, timezone, err)
	}
	if validator.calls != 1 || validator.credential != "session-token" {
		t.Fatalf("Validate calls=%d credential=%q", validator.calls, validator.credential)
	}
}

func TestServiceSessionIdentityFromRequestUsesCookieWhenCredentialIsEmpty(t *testing.T) {
	policy, err := loomhttp.NewRequestMetadataPolicy([]string{"Cookie"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	validator := &serviceSessionValidatorStub{principal: djangosession.Principal{
		UserID: "cookie-user", UserTimezone: "UTC",
	}}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "cookie-token"})
	loomhttp.RequestMetadataMiddleware(policy)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		userID, timezone, err := serviceSessionIdentityFromRequest(request.Context(), "", validator)
		if err != nil || userID != "cookie-user" || timezone != "UTC" {
			t.Errorf("identity=(%q, %q) err=%v", userID, timezone, err)
		}
	})).ServeHTTP(httptest.NewRecorder(), request)
	if validator.calls != 1 || validator.credential != "cookie-token" {
		t.Fatalf("Validate calls=%d credential=%q", validator.calls, validator.credential)
	}
}

func TestServiceSessionIdentityWithoutRequestFallbackUsesEmptyCredential(t *testing.T) {
	policy, err := loomhttp.NewRequestMetadataPolicy([]string{"Cookie"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	validator := &serviceSessionValidatorStub{principal: djangosession.Principal{UserID: "validated-user"}}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "cookie-token"})
	loomhttp.RequestMetadataMiddleware(policy)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		_, _, _ = serviceSessionIdentity(request.Context(), "", validator)
	})).ServeHTTP(httptest.NewRecorder(), request)
	if validator.calls != 1 || validator.credential != "" {
		t.Fatalf("Validate calls=%d credential=%q", validator.calls, validator.credential)
	}
}

func TestServiceSessionIdentityReturnsPrincipalAlongsideValidationError(t *testing.T) {
	validationErr := errors.New("session backend unavailable")
	validator := &serviceSessionValidatorStub{
		principal: djangosession.Principal{UserID: "partial-user", UserTimezone: "UTC"},
		err:       validationErr,
	}
	userID, timezone, err := serviceSessionIdentity(context.Background(), "session-token", validator)
	if !errors.Is(err, validationErr) || userID != "partial-user" || timezone != "UTC" {
		t.Fatalf("identity=(%q, %q) err=%v", userID, timezone, err)
	}
}

func TestValidateSessionUserUUIDAlwaysValidatesCredential(t *testing.T) {
	wantID, err := uuid.Parse("f3d3d93e-70c4-44b7-a62b-8f14f7bb7e11")
	if err != nil {
		t.Fatal(err)
	}
	validator := &serviceSessionValidatorStub{principal: djangosession.Principal{UserID: wantID.String()}}
	ctx := WithReadServiceSessionPrincipal(context.Background(), "cached-user", "UTC")

	gotID, err := validateSessionUserUUID(ctx, "explicit-session", validator)
	if err != nil || gotID != wantID {
		t.Fatalf("user ID=%s err=%v want=%s", gotID, err, wantID)
	}
	if validator.calls != 1 || validator.credential != "explicit-session" {
		t.Fatalf("Validate calls=%d credential=%q", validator.calls, validator.credential)
	}
}

func TestValidateSessionUserUUIDReturnsValidationError(t *testing.T) {
	wantErr := errors.New("session validation failed")
	validator := &serviceSessionValidatorStub{err: wantErr}
	if _, err := validateSessionUserUUID(context.Background(), "session-token", validator); !errors.Is(err, wantErr) {
		t.Fatalf("validation error=%v want=%v", err, wantErr)
	}
}

func TestValidateSessionUserUUIDRejectsInvalidUserID(t *testing.T) {
	validator := &serviceSessionValidatorStub{principal: djangosession.Principal{UserID: "not-a-uuid"}}
	if _, err := validateSessionUserUUID(context.Background(), "session-token", validator); err == nil {
		t.Fatal("invalid user ID was accepted")
	}
}
