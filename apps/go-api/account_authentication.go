package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	api "github.com/CaliLuke/go-plane/accountauthentication/gen/account_authentication"
	store "github.com/CaliLuke/go-plane/internal/accountauthentication"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/authentication/password"
	"github.com/CaliLuke/go-plane/internal/authentication/passwordstrength"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/emailvalidation"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

const accountAuthenticationInternalMessage = "Something went wrong please try again later"
const accountAuthenticationBackend = "django.contrib.auth.backends.ModelBackend"

type accountAuthenticationRepository interface {
	InstanceConfigured(context.Context) (bool, error)
	ConfigValue(context.Context, string, string) (string, error)
	UserByEmail(context.Context, string) (store.User, error)
	UserByID(context.Context, string) (store.User, error)
	UpdatePassword(context.Context, string, string, bool, time.Time) error
	SetPassword(context.Context, string, string, bool) error
	UpdateLoginMetadata(context.Context, string, time.Time, string, string, string, time.Time) (bool, error)
	UserJSON(context.Context, string) (json.RawMessage, error)
}
type accountAuthenticationThrottle interface {
	AllowWithWait(context.Context, string) (bool, time.Duration, error)
}
type accountAuthenticationCache interface {
	Invalidate(context.Context, string, string, bool) error
}
type accountAuthenticationTasks interface {
	Task(context.Context, string, []any, map[string]any) error
}
type accountAuthenticationMagicIssuer interface {
	Issue(context.Context, string) (string, string, error)
}
type accountAuthenticationResetTokenIssuer interface {
	Issue(context.Context, store.User) (string, string, error)
}

type accountAuthenticationService struct {
	repository   accountAuthenticationRepository
	throttle     accountAuthenticationThrottle
	cache        accountAuthenticationCache
	tasks        accountAuthenticationTasks
	magic        accountAuthenticationMagicIssuer
	resetTokens  accountAuthenticationResetTokenIssuer
	passwords    *password.Hasher
	strength     passwordstrength.Scorer
	sessions     *djangosession.Validator
	appBaseURL   string
	spaceBaseURL string
	now          func() time.Time
}

func NewAccountAuthentication(repository accountAuthenticationRepository, throttle accountAuthenticationThrottle, cache accountAuthenticationCache, tasks accountAuthenticationTasks, magic accountAuthenticationMagicIssuer, resetTokens accountAuthenticationResetTokenIssuer, passwords *password.Hasher, sessions *djangosession.Validator, appBaseURL, spaceBaseURL string) (api.Service, error) {
	if repository == nil || throttle == nil || cache == nil || tasks == nil || magic == nil || resetTokens == nil || passwords == nil || sessions == nil {
		return nil, errors.New("account authentication dependencies are required")
	}
	return &accountAuthenticationService{repository: repository, throttle: throttle, cache: cache, tasks: tasks, magic: magic, resetTokens: resetTokens, passwords: passwords, strength: passwordstrength.Scorer{}, sessions: sessions, appBaseURL: appBaseURL, spaceBaseURL: spaceBaseURL, now: time.Now}, nil
}

var _ api.Service = (*accountAuthenticationService)(nil)

type accountAuthenticationPrincipalKey struct{}
type accountAuthenticationPrincipal struct{ userID, timezone string }

func (s *accountAuthenticationService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	if userID, timezone, inspected := readServiceSessionPrincipal(ctx); inspected {
		if userID == "" {
			return ctx, s.unauthenticated()
		}
		if key == "" {
			key = sessionCredentialFromContext(ctx)
		}
		if key == "" {
			return ctx, s.unauthenticated()
		}
		principal, current, err := s.sessions.Inspect(ctx, key)
		if err != nil || principal.UserID != userID {
			return ctx, s.unauthenticated()
		}
		ctx = djangosession.WithCurrentSession(ctx, current)
		return context.WithValue(ctx, accountAuthenticationPrincipalKey{}, accountAuthenticationPrincipal{userID, timezone}), nil
	}
	if key == "" {
		key = sessionCredentialFromContext(ctx)
	}
	if key == "" {
		return ctx, s.unauthenticated()
	}
	principal, current, err := s.sessions.Inspect(ctx, key)
	if err != nil {
		return ctx, s.unauthenticated()
	}
	ctx = djangosession.WithCurrentSession(ctx, current)
	return context.WithValue(ctx, accountAuthenticationPrincipalKey{}, accountAuthenticationPrincipal{principal.UserID, principal.UserTimezone}), nil
}
func (s *accountAuthenticationService) AuthorizeSessionProtocol(ctx context.Context, key string) (int, map[string]any) {
	_, err := s.APIKeyAuth(ctx, key, nil)
	return accountAuthenticationProtocolError(err)
}
func (s *accountAuthenticationService) AuthorizeSessionResourceProtocol(ctx context.Context, key string, _ SessionProtocolResource) (int, map[string]any) {
	return s.AuthorizeSessionProtocol(ctx, key)
}

// AuthorizePublicProtocolRequest preserves DRF APIView throttling for OPTIONS
// and unsupported methods, which are dispatched before the generated POST
// handler. Generated requests continue to use allow exactly once in the
// service operation.
func (s *accountAuthenticationService) AuthorizePublicProtocolRequest(ctx context.Context) (int, map[string]any, http.Header) {
	err := s.allow(ctx)
	if err == nil {
		return 0, nil, nil
	}
	status, body := accountAuthenticationProtocolError(err)
	headers := make(http.Header)
	var failure *api.AccountAuthenticationError
	if errors.As(err, &failure) && failure.RetryAfter != nil {
		headers.Set("Retry-After", strconv.Itoa(*failure.RetryAfter))
	}
	return status, body, headers
}

func (s *accountAuthenticationService) EmailCheck(ctx context.Context, p *api.EmailCheckPayload, body io.ReadCloser) (*api.AccountAuthenticationRawResponse, io.ReadCloser, error) {
	return s.emailCheck(ctx, p.ContentType, body)
}
func (s *accountAuthenticationService) EmailCheckSpace(ctx context.Context, p *api.EmailCheckSpacePayload, body io.ReadCloser) (*api.AccountAuthenticationRawResponse, io.ReadCloser, error) {
	return s.emailCheck(ctx, p.ContentType, body)
}
func (s *accountAuthenticationService) ForgotPassword(ctx context.Context, p *api.ForgotPasswordPayload, body io.ReadCloser) (*api.AccountAuthenticationRawResponse, io.ReadCloser, error) {
	return s.forgotPassword(ctx, p.ContentType, body, false)
}
func (s *accountAuthenticationService) ForgotPasswordSpace(ctx context.Context, p *api.ForgotPasswordSpacePayload, body io.ReadCloser) (*api.AccountAuthenticationRawResponse, io.ReadCloser, error) {
	return s.forgotPassword(ctx, p.ContentType, body, true)
}
func (s *accountAuthenticationService) MagicGenerate(ctx context.Context, p *api.MagicGeneratePayload, body io.ReadCloser) (*api.AccountAuthenticationRawResponse, io.ReadCloser, error) {
	return s.magicGenerate(ctx, p.ContentType, body)
}
func (s *accountAuthenticationService) MagicGenerateSpace(ctx context.Context, p *api.MagicGenerateSpacePayload, body io.ReadCloser) (*api.AccountAuthenticationRawResponse, io.ReadCloser, error) {
	return s.magicGenerate(ctx, p.ContentType, body)
}
func (s *accountAuthenticationService) ChangePassword(ctx context.Context, p *api.ChangePasswordPayload, body io.ReadCloser) (*api.AccountAuthenticationRawResponse, io.ReadCloser, error) {
	return s.passwordChange(ctx, p.ContentType, body, false)
}
func (s *accountAuthenticationService) SetPassword(ctx context.Context, p *api.SetPasswordPayload, body io.ReadCloser) (*api.AccountAuthenticationRawResponse, io.ReadCloser, error) {
	return s.passwordChange(ctx, p.ContentType, body, true)
}

func (s *accountAuthenticationService) emailCheck(ctx context.Context, contentType *string, body io.ReadCloser) (*api.AccountAuthenticationRawResponse, io.ReadCloser, error) {
	if err := s.allow(ctx); err != nil {
		return nil, nil, err
	}
	configured, err := s.repository.InstanceConfigured(ctx)
	if err != nil {
		return nil, nil, s.internal()
	}
	if !configured {
		return nil, nil, s.failure(400, authAuthenticationBody(5000, "INSTANCE_NOT_CONFIGURED", nil))
	}
	input, err := s.parse(body, contentType)
	if err != nil {
		return nil, nil, s.inputFailure(err)
	}
	emailValue, ok := input.Fields["email"]
	if !ok || !authenticationTruthy(emailValue) {
		return nil, nil, s.failure(400, authAuthenticationBody(5010, "EMAIL_REQUIRED", nil))
	}
	email := emailvalidation.NormalizePythonEmail(authenticationString(emailValue))
	if !emailvalidation.ValidDjangoEmail(email) {
		return nil, nil, s.failure(400, authAuthenticationBody(5005, "INVALID_EMAIL", nil))
	}
	user, userErr := s.repository.UserByEmail(ctx, email)
	if userErr != nil && !errors.Is(userErr, store.ErrUserNotFound) {
		return nil, nil, s.internal()
	}
	mailHost, err := s.repository.ConfigValue(ctx, "EMAIL_HOST", "")
	if err != nil {
		return nil, nil, s.internal()
	}
	enabled, err := s.repository.ConfigValue(ctx, "ENABLE_MAGIC_LINK_LOGIN", "1")
	if err != nil {
		return nil, nil, s.internal()
	}
	canUseMagic := mailHost != "" && enabled == "1"
	status := "CREDENTIAL"
	if canUseMagic && (errors.Is(userErr, store.ErrUserNotFound) || user.PasswordAutoset) {
		status = "MAGIC_CODE"
	}
	return accountAuthenticationRaw(map[string]any{"existing": userErr == nil, "status": status})
}

func (s *accountAuthenticationService) forgotPassword(ctx context.Context, contentType *string, body io.ReadCloser, space bool) (*api.AccountAuthenticationRawResponse, io.ReadCloser, error) {
	if err := s.allow(ctx); err != nil {
		return nil, nil, err
	}
	configured, err := s.repository.InstanceConfigured(ctx)
	if err != nil {
		return nil, nil, s.internal()
	}
	if !configured {
		return nil, nil, s.failure(400, authAuthenticationBody(5000, "INSTANCE_NOT_CONFIGURED", nil))
	}
	input, err := s.parse(body, contentType)
	if err != nil {
		return nil, nil, s.inputFailure(err)
	}
	email := authenticationString(input.Fields["email"])
	if !emailvalidation.ValidDjangoEmail(email) {
		return nil, nil, s.failure(400, authAuthenticationBody(5005, "INVALID_EMAIL", nil))
	}
	mailHost, err := s.repository.ConfigValue(ctx, "EMAIL_HOST", "")
	if err != nil {
		return nil, nil, s.internal()
	}
	if mailHost == "" {
		return nil, nil, s.failure(400, authAuthenticationBody(5025, "SMTP_NOT_CONFIGURED", nil))
	}
	user, err := s.repository.UserByEmail(ctx, email)
	if errors.Is(err, store.ErrUserNotFound) {
		return nil, nil, s.failure(400, authAuthenticationBody(5060, "USER_DOES_NOT_EXIST", nil))
	}
	if err != nil {
		return nil, nil, s.internal()
	}
	uid, token, err := s.resetTokens.Issue(ctx, user)
	if err != nil {
		return nil, nil, s.internal()
	}
	base := s.appBaseURL
	if space {
		base = s.spaceBaseURL
	}
	if err := s.tasks.Task(ctx, "plane.bgtasks.forgot_password_task.forgot_password", []any{user.FirstName, user.Email, uid, token, base}, nil); err != nil {
		return nil, nil, s.internal()
	}
	return accountAuthenticationRaw(map[string]any{"message": "Check your email to reset your password"})
}

func (s *accountAuthenticationService) magicGenerate(ctx context.Context, contentType *string, body io.ReadCloser) (*api.AccountAuthenticationRawResponse, io.ReadCloser, error) {
	if err := s.allow(ctx); err != nil {
		return nil, nil, err
	}
	configured, err := s.repository.InstanceConfigured(ctx)
	if err != nil {
		return nil, nil, s.internal()
	}
	if !configured {
		return nil, nil, s.failure(400, authAuthenticationBody(5000, "INSTANCE_NOT_CONFIGURED", nil))
	}
	input, err := s.parse(body, contentType)
	if err != nil {
		return nil, nil, s.inputFailure(err)
	}
	email := emailvalidation.NormalizePythonEmail(authenticationString(input.Fields["email"]))
	if !emailvalidation.ValidDjangoEmail(email) {
		return nil, nil, s.failure(400, authAuthenticationBody(5005, "INVALID_EMAIL", nil))
	}
	mailHost, err := s.repository.ConfigValue(ctx, "EMAIL_HOST", "")
	if err != nil {
		return nil, nil, s.internal()
	}
	if mailHost == "" {
		return nil, nil, s.failure(400, authAuthenticationBody(5025, "SMTP_NOT_CONFIGURED", map[string]any{"email": email}))
	}
	enabled, err := s.repository.ConfigValue(ctx, "ENABLE_MAGIC_LINK_LOGIN", "1")
	if err != nil {
		return nil, nil, s.internal()
	}
	if enabled == "0" {
		return nil, nil, s.failure(400, authAuthenticationBody(5016, "MAGIC_LINK_LOGIN_DISABLED", map[string]any{"email": email}))
	}
	key, code, err := s.magic.Issue(ctx, email)
	if err != nil {
		return nil, nil, s.failure(400, authenticationCodeBody(err))
	}
	if err := s.tasks.Task(ctx, "plane.bgtasks.magic_link_code_task.magic_link", []any{email, key, code}, nil); err != nil {
		return nil, nil, s.internal()
	}
	return accountAuthenticationRaw(map[string]any{"key": key})
}

func (s *accountAuthenticationService) passwordChange(ctx context.Context, contentType *string, body io.ReadCloser, set bool) (*api.AccountAuthenticationRawResponse, io.ReadCloser, error) {
	principal, ok := ctx.Value(accountAuthenticationPrincipalKey{}).(accountAuthenticationPrincipal)
	if !ok || principal.userID == "" {
		return nil, nil, s.unauthenticated()
	}
	if set {
		if err := s.cache.Invalidate(ctx, "/api/users/me/", "", false); err != nil {
			return nil, nil, s.internal()
		}
	}
	input, err := s.parse(body, contentType)
	if err != nil {
		return nil, nil, s.inputFailure(err)
	}
	user, err := s.repository.UserByID(ctx, principal.userID)
	if err != nil {
		return nil, nil, s.internal()
	}
	if set && !user.PasswordAutoset {
		return nil, nil, s.failure(400, authAuthenticationBody(5145, "PASSWORD_ALREADY_SET", map[string]any{"error": "Your password is already set please change your password from profile"}))
	}
	field := "new_password"
	if set {
		field = "password"
	}
	newValue := input.Fields[field]
	if !authenticationTruthy(newValue) {
		if set {
			return nil, nil, s.failure(400, authAuthenticationBody(5020, "INVALID_PASSWORD", nil))
		}
		return nil, nil, s.failure(400, authAuthenticationBody(5138, "MISSING_PASSWORD", map[string]any{"error": "Old or new password is missing"}))
	}
	if !set && !user.PasswordAutoset && !authenticationTruthy(input.Fields["old_password"]) {
		return nil, nil, s.failure(400, authAuthenticationBody(5138, "MISSING_PASSWORD", map[string]any{"error": "Old password is missing"}))
	}
	if !set && !user.PasswordAutoset {
		valid, verifyErr := s.passwords.Verify(user.Password, authenticationString(input.Fields["old_password"]))
		if verifyErr != nil {
			return nil, nil, s.internal()
		}
		if !valid {
			return nil, nil, s.failure(400, authAuthenticationBody(5135, "INCORRECT_OLD_PASSWORD", map[string]any{"error": "Old password is not correct"}))
		}
	}
	newPassword := authenticationString(newValue)
	score, err := s.strength.Score(newPassword, user.Email)
	if err != nil {
		return nil, nil, s.internal()
	}
	if score < 3 {
		if set {
			return nil, nil, s.failure(400, authAuthenticationBody(5020, "INVALID_PASSWORD", nil))
		}
		return nil, nil, s.failure(400, authAuthenticationBody(5021, "PASSWORD_TOO_WEAK", nil))
	}
	hash, err := s.passwords.Encode(newPassword)
	if err != nil {
		return nil, nil, s.internal()
	}
	now := s.now()
	if set {
		err = s.repository.SetPassword(ctx, user.ID, hash, false)
	} else {
		err = s.repository.UpdatePassword(ctx, user.ID, hash, false, now)
	}
	if err != nil {
		return nil, nil, s.internal()
	}
	user.Password, user.PasswordAutoset = hash, false
	if err := s.completePasswordLogin(ctx, user, now); err != nil {
		return nil, nil, s.internal()
	}
	if set {
		encoded, err := s.repository.UserJSON(ctx, user.ID)
		if err != nil {
			return nil, nil, s.internal()
		}
		return accountAuthenticationRaw(encoded)
	}
	return accountAuthenticationRaw(map[string]any{"message": "Password updated successfully"})
}

func (s *accountAuthenticationService) completePasswordLogin(ctx context.Context, user store.User, now time.Time) error {
	metadata, _ := loomhttp.RequestMetadataFromContext(ctx)
	clientIP, userAgent := metadata.ClientAddr, metadata.UserAgent
	current, hasCurrent := djangosession.CurrentSessionFromContext(ctx)
	var currentPtr *djangosession.SessionRefresh
	if hasCurrent {
		currentPtr = current
	}
	deviceInfo := map[string]any{"user_agent": userAgent, "ip_address": clientIP, "domain": s.appBaseURL}
	refresh, err := s.sessions.Login(ctx, currentPtr, djangosession.AuthenticatedSession{UserID: user.ID, PasswordHash: user.Password, Backend: accountAuthenticationBackend, DeviceInfo: deviceInfo}, 14*24*60*60, now)
	if err != nil {
		return err
	}
	if !djangosession.SetResponseSession(ctx, refresh) {
		return errors.New("session response middleware is unavailable")
	}
	return nil
}

func (s *accountAuthenticationService) allow(ctx context.Context) error {
	if userID, _, inspected := readServiceSessionPrincipal(ctx); inspected && userID != "" {
		return nil
	}
	meta, _ := loomhttp.RequestMetadataFromContext(ctx)
	allowed, wait, err := s.throttle.AllowWithWait(ctx, meta.ClientAddr)
	if err != nil {
		return s.internal()
	}
	if !allowed {
		failure := s.failure(429, authAuthenticationBody(5900, "RATE_LIMIT_EXCEEDED", nil))
		if wait > 0 {
			seconds := int(wait / time.Second)
			failure.RetryAfter = &seconds
		}
		return failure
	}
	return nil
}
func (s *accountAuthenticationService) parse(body io.ReadCloser, contentType *string) (djangoinput.Input, error) {
	if body != nil {
		defer body.Close()
	}
	ct := ""
	if contentType != nil {
		ct = *contentType
	}
	return djangoinput.Parse(body, ct)
}
func (s *accountAuthenticationService) inputFailure(err error) error {
	var httpError *djangoapi.HTTPError
	if errors.As(err, &httpError) {
		return s.failure(httpError.Status, httpError.Body)
	}
	return s.failure(400, map[string]any{"detail": "Invalid data"})
}
func (s *accountAuthenticationService) internal() error {
	return s.failure(500, map[string]any{"error": accountAuthenticationInternalMessage})
}
func (s *accountAuthenticationService) unauthenticated() error {
	encoded, _ := json.Marshal(map[string]any{"detail": unauthenticatedDetail})
	value, err := loom.JSONValueFrom(json.RawMessage(encoded))
	if err != nil {
		value = loom.JSONValue(`null`)
	}
	return &api.AccountAuthenticationError{Name: "unauthenticated", Body: value}
}
func (s *accountAuthenticationService) failure(status int, body any) *api.AccountAuthenticationError {
	encoded, err := json.Marshal(body)
	if err != nil {
		encoded = []byte(`{"error":"Something went wrong please try again later"}`)
	}
	value, err := loom.JSONValueFrom(json.RawMessage(encoded))
	if err != nil {
		value = loom.JSONValue(`null`)
	}
	return &api.AccountAuthenticationError{Name: "status_" + strconv.Itoa(status), Body: value}
}
func accountAuthenticationProtocolError(err error) (int, map[string]any) {
	if err == nil {
		return 0, nil
	}
	var failure *api.AccountAuthenticationError
	if !errors.As(err, &failure) {
		return http.StatusInternalServerError, map[string]any{"error": accountAuthenticationInternalMessage}
	}
	if failure.Name == "unauthenticated" {
		var body map[string]any
		_ = json.Unmarshal([]byte(failure.Body), &body)
		return http.StatusUnauthorized, body
	}
	status, _ := strconv.Atoi(strings.TrimPrefix(failure.Name, "status_"))
	var body map[string]any
	_ = json.Unmarshal([]byte(failure.Body), &body)
	return status, body
}
func accountAuthenticationRaw(body any) (*api.AccountAuthenticationRawResponse, io.ReadCloser, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, nil, err
	}
	return &api.AccountAuthenticationRawResponse{ContentType: "application/json"}, io.NopCloser(strings.NewReader(string(encoded))), nil
}
func authAuthenticationBody(code int, message string, extra map[string]any) map[string]any {
	body := map[string]any{"error_code": code, "error_message": message}
	for key, value := range extra {
		body[key] = value
	}
	return body
}
func authenticationCodeBody(err error) map[string]any {
	var code interface {
		ErrorCode() int
		ErrorMessage() string
	}
	if errors.As(err, &code) {
		extra := map[string]any(nil)
		var magicError *MagicCodeError
		if errors.As(err, &magicError) && magicError.Email != "" {
			extra = map[string]any{"email": magicError.Email}
		}
		return authAuthenticationBody(code.ErrorCode(), code.ErrorMessage(), extra)
	}
	return authAuthenticationBody(5999, "AUTHENTICATION_FAILED", nil)
}

// MagicCodeError carries the AuthenticationException envelope returned by a
// Redis-backed magic-code issuer without coupling the service to that adapter.
type MagicCodeError struct {
	Code    int
	Message string
	Email   string
}

func (err *MagicCodeError) Error() string        { return err.Message }
func (err *MagicCodeError) ErrorCode() int       { return err.Code }
func (err *MagicCodeError) ErrorMessage() string { return err.Message }
func authenticationString(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
func authenticationTruthy(value any) bool { return pythoncompat.Truthy(value) }
