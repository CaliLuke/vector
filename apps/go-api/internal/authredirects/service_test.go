package authredirects

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/accountauthentication"
	"github.com/CaliLuke/go-plane/internal/authentication/browsersession"
)

type accountStub struct {
	configured bool
	user       accountauthentication.User
	userErr    error
	config     map[string]string
	invited    bool
	calls      []string
}

func (stub *accountStub) InstanceConfigured(context.Context) (bool, error) {
	stub.calls = append(stub.calls, "instance")
	return stub.configured, nil
}
func (stub *accountStub) ConfigValue(_ context.Context, name, fallback string) (string, error) {
	stub.calls = append(stub.calls, "config:"+name)
	if value, found := stub.config[name]; found {
		return value, nil
	}
	return fallback, nil
}
func (stub *accountStub) UserByEmail(context.Context, string) (accountauthentication.User, error) {
	stub.calls = append(stub.calls, "user")
	return stub.user, stub.userErr
}
func (stub *accountStub) UserByID(context.Context, string) (accountauthentication.User, error) {
	stub.calls = append(stub.calls, "user-id")
	return stub.user, stub.userErr
}
func (stub *accountStub) CreateUser(context.Context, string, string, string, bool, bool) (accountauthentication.User, error) {
	stub.calls = append(stub.calls, "create")
	return accountauthentication.User{}, nil
}
func (stub *accountStub) SetPassword(_ context.Context, _, encoded string, autoset bool) error {
	stub.calls = append(stub.calls, "set-password")
	stub.user.Password = encoded
	stub.user.PasswordAutoset = autoset
	return nil
}
func (stub *accountStub) UpdateLoginMetadata(context.Context, accountauthentication.LoginMetadata) (bool, error) {
	stub.calls = append(stub.calls, "login-metadata")
	return false, nil
}
func (stub *accountStub) HasWorkspaceInvite(context.Context, string) (bool, error) {
	stub.calls = append(stub.calls, "invite")
	return stub.invited, nil
}
func (stub *accountStub) RedirectProfile(context.Context, string) (bool, string, string, int, error) {
	return false, "", "", 0, nil
}
func (stub *accountStub) EnsureProfile(context.Context, string) error { return nil }
func (stub *accountStub) UpdateLogoutMetadata(context.Context, string, string, time.Time) error {
	return nil
}

type passwordStub struct {
	valid bool
}

func (stub passwordStub) Encode(string) (string, error)       { return "encoded", nil }
func (stub passwordStub) Verify(string, string) (bool, error) { return stub.valid, nil }

type strengthStub int

func (stub strengthStub) Score(string, string) (int, error) { return int(stub), nil }

type resetTokenVerifierStub bool

func (stub resetTokenVerifierStub) Verify(context.Context, accountauthentication.User, string, string) (bool, error) {
	return bool(stub), nil
}

type sessionStore struct{}

func (sessionStore) ReadSession(context.Context, string) (browsersession.Session, error) {
	return browsersession.Session{}, browsersession.ErrSessionNotFound
}

func (sessionStore) ReadUser(context.Context, string) (browsersession.User, error) {
	return browsersession.User{}, browsersession.ErrUserNotFound
}
func (sessionStore) UpdateSession(context.Context, string, string, time.Time, any, any) error {
	return nil
}
func (sessionStore) CreateSession(context.Context, string, string, time.Time, any, any) error {
	return nil
}
func (sessionStore) CycleSession(context.Context, string, string, string, time.Time, any, any) error {
	return nil
}
func (sessionStore) DeleteSession(context.Context, string) error { return nil }

func newTestService(t *testing.T, accounts *accountStub, score int) *Service {
	t.Helper()
	sessions, err := browsersession.New(sessionStore{}, browsersession.Config{
		SecretKey: "test-secret-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(accounts, passwordStub{valid: true}, strengthStub(score), sessions, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SetPasswordResetTokenVerifier(resetTokenVerifierStub(true)); err != nil {
		t.Fatal(err)
	}
	return service
}

func TestEmailSignInChecksSetupThenRequiredFieldsBeforeLookup(t *testing.T) {
	accounts := &accountStub{configured: false}
	service := newTestService(t, accounts, 3)
	result := service.EmailSignIn(context.Background(), Request{BaseURL: "https://app.example.test"}, true)
	if result.Status != http.StatusFound || result.Location != "https://app.example.test?error_code=5000&error_message=INSTANCE_NOT_CONFIGURED" {
		t.Fatalf("result = %#v", result)
	}
	if got := accounts.calls; len(got) != 1 || got[0] != "instance" {
		t.Fatalf("repository calls = %#v, want instance only", got)
	}

	accounts = &accountStub{configured: true}
	service = newTestService(t, accounts, 3)
	result = service.EmailSignIn(context.Background(), Request{BaseURL: "https://app.example.test", Email: ""}, true)
	if result.Status != http.StatusFound || result.Location != "https://app.example.test?error_code=5070&error_message=REQUIRED_EMAIL_PASSWORD_SIGN_IN&email=" {
		t.Fatalf("result = %#v", result)
	}
	if got := accounts.calls; len(got) != 1 || got[0] != "instance" {
		t.Fatalf("repository calls = %#v, want instance only", got)
	}
}

func TestAdminSignOutExpiresOnlyAnAuthenticatedSession(t *testing.T) {
	service := newTestService(t, &accountStub{}, 3)
	anonymous := service.AdminSignOut(context.Background(), Request{BaseURL: "https://admin.example.test/god-mode"}, "")
	if anonymous.Status != http.StatusFound || anonymous.ExpireSession {
		t.Fatalf("anonymous sign-out result = %#v, want redirect without cookie expiry", anonymous)
	}

	authenticated := service.AdminSignOut(context.Background(), Request{
		BaseURL:        "https://admin.example.test/god-mode",
		CurrentSession: &browsersession.SessionRefresh{SessionKey: "admin-session"},
	}, "admin-user")
	if authenticated.Status != http.StatusFound || !authenticated.ExpireSession {
		t.Fatalf("authenticated sign-out result = %#v, want redirect with cookie expiry", authenticated)
	}
}

func TestEmailSignInLooksUpUserBeforeProviderConfiguration(t *testing.T) {
	accounts := &accountStub{configured: true, userErr: accountauthentication.ErrUserNotFound}
	service := newTestService(t, accounts, 3)
	result := service.EmailSignIn(context.Background(), Request{
		BaseURL: "https://app.example.test", Email: " Missing@Example.test ", Password: "secret",
	}, false)
	if result.Status != http.StatusFound || result.Location != "https://app.example.test?error_code=5060&error_message=USER_DOES_NOT_EXIST&email=missing%40example.test" {
		t.Fatalf("result = %#v", result)
	}
	want := []string{"instance", "user"}
	if len(accounts.calls) != len(want) || accounts.calls[0] != want[0] || accounts.calls[1] != want[1] {
		t.Fatalf("repository calls = %#v, want %#v", accounts.calls, want)
	}
}

func TestEmailSignUpChecksExistingAccountBeforeProviderConfiguration(t *testing.T) {
	accounts := &accountStub{configured: true, user: accountauthentication.User{Email: "member@example.test"}}
	service := newTestService(t, accounts, 3)
	result := service.EmailSignUp(context.Background(), Request{
		BaseURL: "https://app.example.test", Email: "member@example.test", Password: "good password",
	}, false)
	if result.Status != http.StatusFound || result.Location != "https://app.example.test?error_code=5030&error_message=USER_ALREADY_EXIST&email=member%40example.test" {
		t.Fatalf("result = %#v", result)
	}
	want := []string{"instance", "user"}
	if len(accounts.calls) != len(want) || accounts.calls[0] != want[0] || accounts.calls[1] != want[1] {
		t.Fatalf("repository calls = %#v, want %#v", accounts.calls, want)
	}
}

func TestEmailSignUpWeakPasswordDoesNotCreateUser(t *testing.T) {
	accounts := &accountStub{configured: true, userErr: accountauthentication.ErrUserNotFound}
	service := newTestService(t, accounts, 2)
	result := service.EmailSignUp(context.Background(), Request{
		BaseURL: "https://app.example.test", Email: " NEW@Example.test ", Password: "weak",
	}, true)
	if result.Status != http.StatusFound || result.Location != "https://app.example.test?error_code=5021&error_message=PASSWORD_TOO_WEAK&email=new%40example.test" {
		t.Fatalf("result = %#v", result)
	}
	for _, call := range accounts.calls {
		if call == "create" || call == "login-metadata" {
			t.Fatalf("invalid password reached write path: %#v", accounts.calls)
		}
	}
}

func TestResetPasswordUpdatesCredentialsAndUsesAppSuccessRedirect(t *testing.T) {
	accounts := &accountStub{user: accountauthentication.User{
		ID: "user-123", Email: "member@example.test", Password: "old-hash", PasswordAutoset: true,
	}}
	service := newTestService(t, accounts, 3)
	service.now = func() time.Time { return time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC) }
	result := service.ResetPassword(context.Background(), Request{
		BaseURL: "https://app.example.test", PasswordUID: base64.RawURLEncoding.EncodeToString([]byte("user-123")),
		PasswordToken: "valid-token", Password: "a secure password",
	})
	if result.Status != http.StatusFound || result.Location != "https://app.example.test/sign-in?success=True" {
		t.Fatalf("result = %#v", result)
	}
	if accounts.user.Password != "encoded" || accounts.user.PasswordAutoset {
		t.Fatalf("password state = %#v", accounts.user)
	}
	if strings.Join(accounts.calls, ",") != "user-id,set-password" {
		t.Fatalf("repository calls = %#v", accounts.calls)
	}
}

func TestResetPasswordInvalidTokenAndWeakPasswordRedirectWithoutWrites(t *testing.T) {
	uid := base64.RawURLEncoding.EncodeToString([]byte("user-123"))
	accounts := &accountStub{user: accountauthentication.User{ID: "user-123", Email: "member@example.test"}}
	service := newTestService(t, accounts, 3)
	if err := service.SetPasswordResetTokenVerifier(resetTokenVerifierStub(false)); err != nil {
		t.Fatal(err)
	}
	result := service.ResetPassword(context.Background(), Request{
		BaseURL: "https://app.example.test", PasswordUID: uid, PasswordToken: "invalid", Password: "strong password",
	})
	if result.Status != http.StatusFound || result.Location != "https://app.example.test/accounts/reset-password?error_code=5125&error_message=INVALID_PASSWORD_TOKEN" {
		t.Fatalf("invalid-token result = %#v", result)
	}
	if len(accounts.calls) != 1 || accounts.calls[0] != "user-id" {
		t.Fatalf("invalid token reached write path: %#v", accounts.calls)
	}

	accounts = &accountStub{user: accountauthentication.User{ID: "user-123", Email: "member@example.test"}}
	service = newTestService(t, accounts, 2)
	result = service.ResetPassword(context.Background(), Request{
		BaseURL: "https://app.example.test/spaces/", IsSpace: true, PasswordUID: uid,
		PasswordToken: "valid", Password: "weak",
	})
	if result.Status != http.StatusFound || result.Location != "https://app.example.test/spaces//accounts/reset-password/?error_code=5021&error_message=PASSWORD_TOO_WEAK" {
		t.Fatalf("weak-password result = %#v", result)
	}
	if len(accounts.calls) != 1 || accounts.calls[0] != "user-id" {
		t.Fatalf("weak password reached write path: %#v", accounts.calls)
	}
}

func TestSignupUnexpectedLookupErrorDoesNotTreatUserAsMissing(t *testing.T) {
	accounts := &accountStub{configured: true, userErr: errors.New("database unavailable")}
	service := newTestService(t, accounts, 3)
	result := service.EmailSignUp(context.Background(), Request{Email: "a@example.test", Password: "strong"}, true)
	if result.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", result.Status)
	}
	for _, call := range accounts.calls {
		if strings.HasPrefix(call, "config:") || call == "create" {
			t.Fatalf("unexpected lookup error continued to %q", call)
		}
	}
}

func TestStoredAccountRemainsReachableOutsideNewEmailGrammar(t *testing.T) {
	accounts := &accountStub{configured: true, user: accountauthentication.User{
		ID: "00000000-0000-4000-8000-000000000099", Email: "existing@local", Password: "encoded", Active: true,
	}}
	service := newTestService(t, accounts, 3)
	result := service.EmailSignIn(context.Background(), Request{BaseURL: "https://app.example.test", Email: " Existing@Local ", Password: "secret"}, true)
	if result.Status != http.StatusFound || result.Session == nil || strings.Contains(result.Location, "error_code") {
		t.Fatalf("stored identity could not log in: %#v", result)
	}
	accounts.userErr = accountauthentication.ErrUserNotFound
	result = service.EmailSignIn(context.Background(), Request{BaseURL: "https://app.example.test", Email: "new@local", Password: "secret"}, true)
	if !strings.Contains(result.Location, "INVALID_EMAIL_SIGN_IN") {
		t.Fatalf("new malformed identity accepted: %#v", result)
	}
}
