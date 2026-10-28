package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	api "github.com/CaliLuke/go-plane/accountauthentication/gen/account_authentication"
	store "github.com/CaliLuke/go-plane/internal/accountauthentication"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/authentication/password"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
)

type accountAuthenticationTestRepo struct {
	configured          bool
	config              map[string]string
	user                store.User
	userErr             error
	updatePasswordCalls int
	setPasswordCalls    int
	updateLoginCalls    int
}

func (r *accountAuthenticationTestRepo) InstanceConfigured(context.Context) (bool, error) {
	return r.configured, nil
}
func (r *accountAuthenticationTestRepo) ConfigValue(_ context.Context, key, fallback string) (string, error) {
	if v, ok := r.config[key]; ok {
		return v, nil
	}
	return fallback, nil
}
func (r *accountAuthenticationTestRepo) UserByEmail(context.Context, string) (store.User, error) {
	return r.user, r.userErr
}
func (r *accountAuthenticationTestRepo) UserByID(context.Context, string) (store.User, error) {
	return r.user, r.userErr
}
func (r *accountAuthenticationTestRepo) UpdatePassword(context.Context, string, string, bool, time.Time) error {
	r.updatePasswordCalls++
	return nil
}
func (r *accountAuthenticationTestRepo) SetPassword(context.Context, string, string, bool) error {
	r.setPasswordCalls++
	return nil
}
func (r *accountAuthenticationTestRepo) UpdateLoginMetadata(context.Context, string, time.Time, string, string, string, time.Time) (bool, error) {
	r.updateLoginCalls++
	return false, nil
}
func (r *accountAuthenticationTestRepo) UserJSON(context.Context, string) (json.RawMessage, error) {
	return json.RawMessage(`{"id":"user"}`), nil
}

type accountAuthenticationTestThrottle struct {
	allowed bool
	calls   int
}

type accountAuthenticationTestCache struct{ paths []string }

func (c *accountAuthenticationTestCache) Invalidate(_ context.Context, path, _ string, _ bool) error {
	c.paths = append(c.paths, path)
	return nil
}

func (t *accountAuthenticationTestThrottle) AllowWithWait(context.Context, string) (bool, time.Duration, error) {
	t.calls++
	return t.allowed, time.Minute, nil
}

type accountAuthenticationTestTasks struct {
	calls []string
	args  [][]any
}

func (t *accountAuthenticationTestTasks) Task(_ context.Context, name string, args []any, _ map[string]any) error {
	t.calls = append(t.calls, name)
	t.args = append(t.args, args)
	return nil
}

type accountAuthenticationTestMagic struct {
	key, code string
	err       error
	calls     *int
}

func (m accountAuthenticationTestMagic) Issue(context.Context, string) (string, string, error) {
	if m.calls != nil {
		*m.calls++
	}
	return m.key, m.code, m.err
}

type accountAuthenticationTestResetTokens struct{ uid, token string }

func (m accountAuthenticationTestResetTokens) Issue(context.Context, store.User) (string, string, error) {
	return m.uid, m.token, nil
}

type accountAuthenticationTestSessionReader struct{}

func (accountAuthenticationTestSessionReader) ReadSession(context.Context, string) (djangosession.Session, error) {
	return djangosession.Session{}, errors.New("unused")
}
func (accountAuthenticationTestSessionReader) ReadSessionForRefresh(context.Context, string) (djangosession.Session, error) {
	return djangosession.Session{}, errors.New("unused")
}
func (accountAuthenticationTestSessionReader) ReadUser(context.Context, string) (djangosession.User, error) {
	return djangosession.User{}, errors.New("unused")
}
func (accountAuthenticationTestSessionReader) CreateSession(context.Context, string, string, time.Time, any, any) error {
	return nil
}
func (accountAuthenticationTestSessionReader) UpdateSession(context.Context, string, string, time.Time, any, any) error {
	return nil
}

func newAccountAuthenticationTestService(t *testing.T, repo *accountAuthenticationTestRepo, throttle *accountAuthenticationTestThrottle, tasks *accountAuthenticationTestTasks, magic accountAuthenticationTestMagic) *accountAuthenticationService {
	t.Helper()
	sessions, err := djangosession.New(accountAuthenticationTestSessionReader{}, djangosession.Config{SecretKey: "test-secret", AllowedBackends: []string{accountAuthenticationBackend}})
	if err != nil {
		t.Fatal(err)
	}
	hasher, err := password.New(100)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewAccountAuthentication(repo, throttle, &accountAuthenticationTestCache{}, tasks, magic, accountAuthenticationTestResetTokens{uid: "dXNlcg", token: "token"}, hasher, sessions, "https://app.example", "https://space.example")
	if err != nil {
		t.Fatal(err)
	}
	return service.(*accountAuthenticationService)
}

func TestAccountAuthenticationEmailCheckUsesConfiguredSourcePolicy(t *testing.T) {
	for _, test := range []struct {
		name         string
		user         store.User
		userErr      error
		wantExisting bool
		wantStatus   string
	}{
		{name: "existing autoset uses magic", user: store.User{PasswordAutoset: true}, wantExisting: true, wantStatus: "MAGIC_CODE"},
		{name: "existing password user uses credentials", user: store.User{}, wantExisting: true, wantStatus: "CREDENTIAL"},
		{name: "new user uses magic", userErr: store.ErrUserNotFound, wantStatus: "MAGIC_CODE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &accountAuthenticationTestRepo{configured: true, config: map[string]string{"EMAIL_HOST": "smtp.test", "ENABLE_MAGIC_LINK_LOGIN": "1"}, user: test.user, userErr: test.userErr}
			throttle := &accountAuthenticationTestThrottle{allowed: true}
			service := newAccountAuthenticationTestService(t, repo, throttle, &accountAuthenticationTestTasks{}, accountAuthenticationTestMagic{})
			contentType := "application/json"
			meta, body, err := service.EmailCheck(context.Background(), &api.EmailCheckPayload{ContentType: &contentType}, io.NopCloser(strings.NewReader(`{"email":" User@Example.COM "}`)))
			if err != nil {
				t.Fatal(err)
			}
			defer testsupport.Close(t, body)
			var got struct {
				Existing bool   `json:"existing"`
				Status   string `json:"status"`
			}
			if err := json.NewDecoder(body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			if meta.ContentType != "application/json" || got.Existing != test.wantExisting || got.Status != test.wantStatus {
				t.Fatalf("response metadata=%+v body=%+v", meta, got)
			}
			if throttle.calls != 1 {
				t.Fatalf("throttle calls=%d, want 1", throttle.calls)
			}
		})
	}
}

func TestPasswordWriteRefreshesSessionWithoutMutatingLoginMetadata(t *testing.T) {
	repo := &accountAuthenticationTestRepo{
		configured: true,
		user:       store.User{ID: "00000000-0000-4000-8000-000000000001", Email: "auth@example.test", PasswordAutoset: true, Active: true},
	}
	service := newAccountAuthenticationTestService(t, repo, &accountAuthenticationTestThrottle{allowed: true}, &accountAuthenticationTestTasks{}, accountAuthenticationTestMagic{})
	ctx, _ := djangosession.WithResponseSessionAction(context.Background())
	ctx = context.WithValue(ctx, accountAuthenticationPrincipalKey{}, accountAuthenticationPrincipal{userID: repo.user.ID, timezone: "UTC"})
	contentType := "application/json"
	_, body, err := service.SetPassword(ctx, &api.SetPasswordPayload{ContentType: &contentType}, io.NopCloser(strings.NewReader(`{"password":"correct-horse-battery-staple-47!"}`)))
	if err != nil {
		t.Fatalf("set password: %T %#v", err, err)
	}
	if body != nil {
		defer testsupport.Close(t, body)
	}
	if repo.setPasswordCalls != 1 || repo.updateLoginCalls != 0 {
		t.Fatalf("password writes=%d login metadata writes=%d", repo.setPasswordCalls, repo.updateLoginCalls)
	}
	refresh, expire, found := djangosession.ResponseSessionActionFromContext(ctx)
	if !found || refresh == nil || expire {
		t.Fatalf("session action found=%t refresh=%v expire=%t", found, refresh != nil, expire)
	}
	value, ok := refresh.Get("device_info")
	if !ok || value == nil {
		t.Fatalf("session device_info=%#v present=%t", value, ok)
	}
}

func TestPasswordWriteWithoutPrincipalUsesDeclaredUnauthenticatedError(t *testing.T) {
	repo := &accountAuthenticationTestRepo{}
	service := newAccountAuthenticationTestService(t, repo, &accountAuthenticationTestThrottle{allowed: true}, &accountAuthenticationTestTasks{}, accountAuthenticationTestMagic{})
	contentType := "application/json"
	_, _, err := service.SetPassword(context.Background(), &api.SetPasswordPayload{ContentType: &contentType}, io.NopCloser(strings.NewReader(`{}`)))
	var failure *api.AccountAuthenticationError
	if !errors.As(err, &failure) || failure.Name != "unauthenticated" {
		t.Fatalf("password write error=%#v, want declared unauthenticated error", err)
	}
	status, body := accountAuthenticationProtocolError(err)
	if status != http.StatusUnauthorized || body["detail"] != unauthenticatedDetail {
		t.Fatalf("password write protocol status=%d body=%#v, want declared 401 response", status, body)
	}
}

func TestPasswordEndpointsDoNotValidateTimezoneForPlainAPIView(t *testing.T) {
	for _, test := range []struct {
		name string
		set  bool
	}{
		{name: "change password"},
		{name: "set password", set: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &accountAuthenticationTestRepo{user: store.User{ID: "user-1", PasswordAutoset: true}}
			service := newAccountAuthenticationTestService(t, repo, &accountAuthenticationTestThrottle{allowed: true}, &accountAuthenticationTestTasks{}, accountAuthenticationTestMagic{})
			ctx := context.WithValue(context.Background(), accountAuthenticationPrincipalKey{}, accountAuthenticationPrincipal{userID: repo.user.ID, timezone: "Not/AZone"})
			contentType := "application/json"
			var err error
			if test.set {
				_, _, err = service.SetPassword(ctx, &api.SetPasswordPayload{ContentType: &contentType}, io.NopCloser(strings.NewReader(`{}`)))
			} else {
				_, _, err = service.ChangePassword(ctx, &api.ChangePasswordPayload{ContentType: &contentType}, io.NopCloser(strings.NewReader(`{}`)))
			}
			status, _ := accountAuthenticationProtocolError(err)
			if status != http.StatusBadRequest {
				t.Fatalf("invalid-timezone missing-password status=%d error=%#v, want normal validation 400", status, err)
			}
		})
	}
}

func TestAccountAuthenticationEmailCheckThrottleAndInstancePrecedence(t *testing.T) {
	contentType := "application/json"
	for _, test := range []struct {
		name                string
		configured, allowed bool
		wantStatus          int
	}{
		{name: "throttle precedes view", configured: false, allowed: false, wantStatus: 429},
		{name: "unconfigured instance", configured: false, allowed: true, wantStatus: 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &accountAuthenticationTestRepo{configured: test.configured}
			throttle := &accountAuthenticationTestThrottle{allowed: test.allowed}
			service := newAccountAuthenticationTestService(t, repo, throttle, &accountAuthenticationTestTasks{}, accountAuthenticationTestMagic{})
			_, _, err := service.EmailCheck(context.Background(), &api.EmailCheckPayload{ContentType: &contentType}, io.NopCloser(strings.NewReader(`{"email":"a@example.com"}`)))
			var failure interface{ LoomErrorName() string }
			if !errors.As(err, &failure) {
				t.Fatalf("error=%v", err)
			}
			if got := failure.LoomErrorName(); got != "status_"+strconv.Itoa(test.wantStatus) {
				t.Fatalf("error status=%s", got)
			}
			if test.wantStatus == 429 {
				var rateError *api.AccountAuthenticationError
				if !errors.As(err, &rateError) || rateError.RetryAfter == nil || *rateError.RetryAfter != 60 {
					t.Fatalf("rate limit error=%+v, want Retry-After 60", rateError)
				}
			}
			if repo.configured == false && test.allowed && throttle.calls != 1 {
				t.Fatalf("expected one throttle before config query")
			}
			if errors.Is(err, io.EOF) {
				t.Fatal("unexpected EOF")
			}
		})
	}
}

func TestAccountAuthenticationThrottleIsSkippedForAuthenticatedSession(t *testing.T) {
	repo := &accountAuthenticationTestRepo{
		configured: true,
		config:     map[string]string{"EMAIL_HOST": "smtp.test", "ENABLE_MAGIC_LINK_LOGIN": "1"},
		userErr:    store.ErrUserNotFound,
	}
	throttle := &accountAuthenticationTestThrottle{allowed: false}
	service := newAccountAuthenticationTestService(t, repo, throttle, &accountAuthenticationTestTasks{}, accountAuthenticationTestMagic{})
	contentType := "application/json"
	ctx := WithReadServiceSessionPrincipal(context.Background(), "authenticated-user", "UTC")
	meta, body, err := service.EmailCheck(ctx, &api.EmailCheckPayload{ContentType: &contentType}, io.NopCloser(strings.NewReader(`{"email":"a@example.com"}`)))
	if err != nil {
		t.Fatalf("authenticated session was throttled: %v", err)
	}
	defer testsupport.Close(t, body)
	if meta == nil || throttle.calls != 0 {
		t.Fatalf("response metadata=%+v throttle calls=%d, want no throttle", meta, throttle.calls)
	}
}

func TestAccountAuthenticationProtocolThrottlePreservesRetryAfterAndSessionBypass(t *testing.T) {
	repo := &accountAuthenticationTestRepo{configured: true, config: map[string]string{}}
	throttle := &accountAuthenticationTestThrottle{allowed: false}
	service := newAccountAuthenticationTestService(t, repo, throttle, &accountAuthenticationTestTasks{}, accountAuthenticationTestMagic{})

	status, body, headers := service.AuthorizePublicProtocolRequest(context.Background())
	code, ok := body["error_code"].(float64)
	if status != http.StatusTooManyRequests || !ok || code != 5900 || headers.Get("Retry-After") != "60" {
		t.Fatalf("throttled protocol response = %d, %v, %v", status, body, headers)
	}
	if throttle.calls != 1 {
		t.Fatalf("anonymous protocol throttle calls = %d, want 1", throttle.calls)
	}

	ctx := WithReadServiceSessionPrincipal(context.Background(), "authenticated-user", "UTC")
	status, body, headers = service.AuthorizePublicProtocolRequest(ctx)
	if status != 0 || body != nil || len(headers) != 0 {
		t.Fatalf("authenticated protocol response = %d, %v, %v", status, body, headers)
	}
	if throttle.calls != 1 {
		t.Fatalf("authenticated protocol consumed throttle; calls=%d", throttle.calls)
	}
}

func TestAccountAuthenticationForgotPasswordPublishesResetTokenInputs(t *testing.T) {
	repo := &accountAuthenticationTestRepo{configured: true, config: map[string]string{"EMAIL_HOST": "smtp.test"}, user: store.User{ID: "user-id", FirstName: "Ada", Email: "ada@example.com"}}
	tasks := &accountAuthenticationTestTasks{}
	service := newAccountAuthenticationTestService(t, repo, &accountAuthenticationTestThrottle{allowed: true}, tasks, accountAuthenticationTestMagic{})
	contentType := "application/json"
	meta, body, err := service.ForgotPasswordSpace(context.Background(), &api.ForgotPasswordSpacePayload{ContentType: &contentType}, io.NopCloser(strings.NewReader(`{"email":"ada@example.com"}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, body)
	if meta.ContentType != "application/json" {
		t.Fatalf("content type=%q", meta.ContentType)
	}
	if len(tasks.calls) != 1 || tasks.calls[0] != "plane.bgtasks.forgot_password_task.forgot_password" {
		t.Fatalf("tasks=%v", tasks.calls)
	}
	if got := tasks.args[0]; len(got) != 5 || got[0] != "Ada" || got[1] != "ada@example.com" || got[2] != "dXNlcg" || got[3] != "token" || got[4] != "https://space.example" {
		t.Fatalf("task args=%#v", got)
	}
}

func TestAccountAuthenticationMagicGenerationPersistsAndQueuesSameCode(t *testing.T) {
	repo := &accountAuthenticationTestRepo{configured: true, config: map[string]string{"EMAIL_HOST": "smtp.test", "ENABLE_MAGIC_LINK_LOGIN": "1"}}
	tasks := &accountAuthenticationTestTasks{}
	service := newAccountAuthenticationTestService(t, repo, &accountAuthenticationTestThrottle{allowed: true}, tasks, accountAuthenticationTestMagic{key: "magic_a@example.com", code: "123456"})
	contentType := "application/json"
	_, body, err := service.MagicGenerate(context.Background(), &api.MagicGeneratePayload{ContentType: &contentType}, io.NopCloser(strings.NewReader(`{"email":"A@EXAMPLE.COM"}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, body)
	var got map[string]string
	if err := json.NewDecoder(body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got["key"] != "magic_a@example.com" {
		t.Fatalf("response=%v", got)
	}
	if len(tasks.args) != 1 || tasks.args[0][0] != "a@example.com" || tasks.args[0][1] != "magic_a@example.com" || tasks.args[0][2] != "123456" {
		t.Fatalf("task args=%#v", tasks.args)
	}
}

func TestAccountAuthenticationMagicGenerationRejectsInvalidEmailWithoutMagicOrQueueEffects(t *testing.T) {
	for _, test := range []struct {
		name string
		call func(*accountAuthenticationService, *string, io.ReadCloser) error
	}{
		{
			name: "app",
			call: func(service *accountAuthenticationService, contentType *string, body io.ReadCloser) error {
				_, _, err := service.MagicGenerate(context.Background(), &api.MagicGeneratePayload{ContentType: contentType}, body)
				return err
			},
		},
		{
			name: "space",
			call: func(service *accountAuthenticationService, contentType *string, body io.ReadCloser) error {
				_, _, err := service.MagicGenerateSpace(context.Background(), &api.MagicGenerateSpacePayload{ContentType: contentType}, body)
				return err
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &accountAuthenticationTestRepo{
				configured: true,
				config:     map[string]string{"EMAIL_HOST": "smtp.test", "ENABLE_MAGIC_LINK_LOGIN": "1"},
			}
			throttle := &accountAuthenticationTestThrottle{allowed: true}
			tasks := &accountAuthenticationTestTasks{}
			magicCalls := 0
			service := newAccountAuthenticationTestService(t, repo, throttle, tasks, accountAuthenticationTestMagic{calls: &magicCalls})
			contentType := "application/json"
			err := test.call(service, &contentType, io.NopCloser(strings.NewReader(`{"email":"not-an-email"}`)))
			status, body := accountAuthenticationProtocolError(err)
			wantBody := map[string]any{"error_code": float64(5005), "error_message": "INVALID_EMAIL"}
			if status != http.StatusBadRequest || !reflect.DeepEqual(body, wantBody) {
				t.Fatalf("status=%d body=%v, want %d %v", status, body, http.StatusBadRequest, wantBody)
			}
			if magicCalls != 0 || len(tasks.calls) != 0 {
				t.Fatalf("invalid email had magic/queue effects: issuer_calls=%d tasks=%v", magicCalls, tasks.calls)
			}
			if throttle.calls != 1 {
				t.Fatalf("authentication throttle calls=%d, want the APIView preflight call", throttle.calls)
			}
		})
	}
}

func TestAccountAuthenticationMagicGenerationPreservesIssuerEmailPayload(t *testing.T) {
	repo := &accountAuthenticationTestRepo{configured: true, config: map[string]string{"EMAIL_HOST": "smtp.test", "ENABLE_MAGIC_LINK_LOGIN": "1"}}
	service := newAccountAuthenticationTestService(t, repo, &accountAuthenticationTestThrottle{allowed: true}, &accountAuthenticationTestTasks{}, accountAuthenticationTestMagic{
		err: &MagicCodeError{Code: 5100, Message: "EMAIL_CODE_ATTEMPT_EXHAUSTED_SIGN_IN", Email: "a@example.com"},
	})
	contentType := "application/json"
	_, _, err := service.MagicGenerate(context.Background(), &api.MagicGeneratePayload{ContentType: &contentType}, io.NopCloser(strings.NewReader(`{"email":"a@example.com"}`)))
	if err == nil {
		t.Fatal("MagicGenerate() error = nil, want exhausted-code error")
	}
	status, body := accountAuthenticationProtocolError(err)
	want := map[string]any{
		"error_code": float64(5100), "error_message": "EMAIL_CODE_ATTEMPT_EXHAUSTED_SIGN_IN", "email": "a@example.com",
	}
	if status != http.StatusBadRequest || !reflect.DeepEqual(body, want) {
		t.Fatalf("status=%d body=%v, want %d %v", status, body, http.StatusBadRequest, want)
	}
}

func TestAccountAuthenticationPrivateRouteAuthorizationRequiresValidSession(t *testing.T) {
	repo := &accountAuthenticationTestRepo{}
	service := newAccountAuthenticationTestService(t, repo, &accountAuthenticationTestThrottle{allowed: true}, &accountAuthenticationTestTasks{}, accountAuthenticationTestMagic{})
	_, err := service.APIKeyAuth(context.Background(), "", nil)
	var unauthorized *api.AccountAuthenticationError
	if !errors.As(err, &unauthorized) || unauthorized.Name != "unauthenticated" {
		t.Fatalf("APIKeyAuth() error = %#v, want generated unauthenticated error", err)
	}
	status, body := service.AuthorizeSessionProtocol(context.Background(), "")
	if status != http.StatusUnauthorized || body["detail"] != "Authentication credentials were not provided." {
		t.Fatalf("status=%d body=%v", status, body)
	}
}
