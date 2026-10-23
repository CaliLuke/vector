package authredirects

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/accountauthentication"
	"github.com/CaliLuke/go-plane/internal/authentication/authworkflow"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/emailvalidation"
	"github.com/google/uuid"
)

const (
	djangoModelBackend = "django.contrib.auth.backends.ModelBackend"
	defaultSessionAge  = int64(14 * 24 * 60 * 60)
)

type PasswordHasher interface {
	Encode(string) (string, error)
	Verify(encoded, raw string) (bool, error)
}

type PasswordStrength interface {
	Score(password, email string) (int, error)
}

type MagicCodeVerifier interface {
	Verify(context.Context, string, string, bool) error
}

type PasswordResetTokenVerifier interface {
	Verify(context.Context, accountauthentication.User, string, string) (bool, error)
}

type AuthenticationThrottle interface {
	Allow(context.Context, string) (bool, error)
}

type CodeError struct {
	Code    int
	Message string
	Email   string
}

func (err *CodeError) Error() string { return err.Message }

type AccountRepository interface {
	InstanceConfigured(context.Context) (bool, error)
	ConfigValue(context.Context, string, string) (string, error)
	UserByEmail(context.Context, string) (accountauthentication.User, error)
	UserByID(context.Context, string) (accountauthentication.User, error)
	CreateUser(context.Context, string, string, string, bool, bool) (accountauthentication.User, error)
	SetPassword(context.Context, string, string, bool) error
	UpdateLoginMetadata(context.Context, string, time.Time, string, string, string, time.Time) (bool, error)
	HasWorkspaceInvite(context.Context, string) (bool, error)
	RedirectProfile(context.Context, string) (onboarded bool, lastWorkspace, earliestWorkspace string, inviteCount int, err error)
	EnsureProfile(context.Context, string) error
	UpdateLogoutMetadata(context.Context, string, string, time.Time) error
}

type AdminAccountRepository interface {
	InstanceExists(context.Context) (bool, error)
	InitialAdminState(context.Context) (exists, setupDone, hasAdmins bool, err error)
	IsInstanceAdmin(context.Context, bool, string) (bool, error)
	CreateInitialAdmin(context.Context, string, string, string, string, string, string, string, bool, time.Time) (accountauthentication.InitialAdminResult, error)
}

type Request struct {
	Email             string
	Password          string
	FirstName         string
	LastName          string
	CompanyName       string
	TelemetryEnabled  bool
	Code              string
	PasswordUID       string
	PasswordToken     string
	NextPath          string
	BaseURL           string
	AllowedHosts      []string
	UserAgent         string
	ClientIP          string
	IsSpace           bool
	SessionAgeSeconds int64
	BaseOrigin        string
	CurrentSession    *djangosession.SessionRefresh
	Now               time.Time
}

type Result struct {
	Status        int
	Location      string
	Body          map[string]any
	Session       *djangosession.SessionRefresh
	ExpireSession bool
}

type Service struct {
	accounts    AccountRepository
	passwords   PasswordHasher
	strength    PasswordStrength
	sessions    *djangosession.Validator
	workflow    *authworkflow.PostUserAuthWorkflow
	activation  ActivationPublisher
	resetTokens PasswordResetTokenVerifier
	magic       MagicCodeVerifier
	throttle    AuthenticationThrottle
	now         func() time.Time
}

type ActivationPublisher interface {
	Task(context.Context, string, []any, map[string]any) error
}

func NewService(accounts AccountRepository, passwords PasswordHasher, strength PasswordStrength, sessions *djangosession.Validator, workflow *authworkflow.PostUserAuthWorkflow, activation ActivationPublisher) (*Service, error) {
	if accounts == nil || passwords == nil || strength == nil || sessions == nil {
		return nil, errors.New("auth redirect service requires accounts, password, strength, and session dependencies")
	}
	return &Service{accounts: accounts, passwords: passwords, strength: strength, sessions: sessions, workflow: workflow, activation: activation, now: time.Now}, nil
}

func (service *Service) SetMagicCodeDependencies(verifier MagicCodeVerifier, throttle AuthenticationThrottle) error {
	if verifier == nil || throttle == nil {
		return errors.New("magic authentication requires a code verifier and throttle")
	}
	service.magic = verifier
	service.throttle = throttle
	return nil
}

func (service *Service) SetPasswordResetTokenVerifier(verifier PasswordResetTokenVerifier) error {
	if verifier == nil {
		return errors.New("password reset requires a token verifier")
	}
	service.resetTokens = verifier
	return nil
}

// ConfigValue exposes the same effective configuration source used by the
// authentication workflow so the HTTP adapter can construct source URLs.
func (service *Service) ConfigValue(ctx context.Context, key, fallback string) (string, error) {
	return service.accounts.ConfigValue(ctx, key, fallback)
}

// EmailSignIn implements the app and space email sign-in flow. The caller
// selects app=true for app-only post-login invitation processing and routing.
func (service *Service) EmailSignIn(ctx context.Context, request Request, app bool) Result {
	configured, err := service.accounts.InstanceConfigured(ctx)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if !configured {
		return service.failure(request, 5000, "INSTANCE_NOT_CONFIGURED", nil)
	}
	if request.Email == "" || request.Password == "" {
		return service.failure(request, 5070, "REQUIRED_EMAIL_PASSWORD_SIGN_IN", map[string]string{"email": request.Email})
	}
	email, ok := normalizeEmail(request.Email)
	if !ok {
		return service.failure(request, 5075, "INVALID_EMAIL_SIGN_IN", map[string]string{"email": strings.ToLower(strings.TrimSpace(request.Email))})
	}
	user, err := service.accounts.UserByEmail(ctx, email)
	if errors.Is(err, accountauthentication.ErrUserNotFound) {
		return service.failure(request, 5060, "USER_DOES_NOT_EXIST", map[string]string{"email": email})
	}
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	enabled, err := service.accounts.ConfigValue(ctx, "ENABLE_EMAIL_PASSWORD", "")
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if enabled == "0" {
		return service.failure(request, 5056, "EMAIL_PASSWORD_AUTHENTICATION_DISABLED", nil)
	}
	valid, err := service.passwords.Verify(user.Password, request.Password)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if !valid {
		return service.failure(request, 5065, "AUTHENTICATION_FAILED_SIGN_IN", map[string]string{"email": email})
	}
	if result := service.loginEligibility(request, user); result != nil {
		return *result
	}
	return service.completeLogin(ctx, request, user, app, "email", false, false)
}

// EmailSignUp implements app and space password signup. Account existence and
// provider configuration are checked before signup policy and password score,
// matching CredentialAdapter's source ordering.
func (service *Service) EmailSignUp(ctx context.Context, request Request, app bool) Result {
	configured, err := service.accounts.InstanceConfigured(ctx)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if !configured {
		return service.failure(request, 5000, "INSTANCE_NOT_CONFIGURED", nil)
	}
	if request.Email == "" || request.Password == "" {
		return service.failure(request, 5040, "REQUIRED_EMAIL_PASSWORD_SIGN_UP", map[string]string{"email": request.Email})
	}
	email, ok := normalizeEmail(request.Email)
	if !ok {
		return service.failure(request, 5045, "INVALID_EMAIL_SIGN_UP", map[string]string{"email": strings.ToLower(strings.TrimSpace(request.Email))})
	}
	_, err = service.accounts.UserByEmail(ctx, email)
	if err == nil {
		return service.failure(request, 5030, "USER_ALREADY_EXIST", map[string]string{"email": email})
	}
	if !errors.Is(err, accountauthentication.ErrUserNotFound) {
		return Result{Status: http.StatusInternalServerError}
	}
	enabled, err := service.accounts.ConfigValue(ctx, "ENABLE_EMAIL_PASSWORD", "")
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if enabled == "0" {
		return service.failure(request, 5056, "EMAIL_PASSWORD_AUTHENTICATION_DISABLED", nil)
	}
	signupEnabled, err := service.accounts.ConfigValue(ctx, "ENABLE_SIGNUP", "1")
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if signupEnabled == "0" {
		invited, inviteErr := service.accounts.HasWorkspaceInvite(ctx, email)
		if inviteErr != nil {
			return Result{Status: http.StatusInternalServerError}
		}
		if !invited {
			return service.failure(request, 5015, "SIGNUP_DISABLED", map[string]string{"email": email})
		}
	}
	score, err := service.strength.Score(request.Password, email)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if score < 3 {
		return service.failure(request, 5021, "PASSWORD_TOO_WEAK", map[string]string{"email": email})
	}
	encoded, err := service.passwords.Encode(request.Password)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	user, err := service.accounts.CreateUser(ctx, email, strings.ReplaceAll(uuid.NewString(), "-", ""), encoded, false, false)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	return service.completeLogin(ctx, request, user, app, "email", true, false)
}

// ResetPassword verifies a Django password-reset token and follows the
// source app/space redirect behavior. A missing or malformed user identifier
// is treated as an invalid reset token rather than exposing account existence.
func (service *Service) ResetPassword(ctx context.Context, request Request) Result {
	if service.resetTokens == nil {
		return Result{Status: http.StatusInternalServerError}
	}
	decodedID, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(request.PasswordUID, "="))
	if err != nil || len(decodedID) == 0 {
		return passwordResetFailure(request, 5125, "INVALID_PASSWORD_TOKEN")
	}
	user, err := service.accounts.UserByID(ctx, string(decodedID))
	if errors.Is(err, accountauthentication.ErrUserNotFound) {
		return passwordResetFailure(request, 5125, "INVALID_PASSWORD_TOKEN")
	}
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	valid, err := service.resetTokens.Verify(ctx, user, request.PasswordUID, request.PasswordToken)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if !valid {
		return passwordResetFailure(request, 5125, "INVALID_PASSWORD_TOKEN")
	}
	if request.Password == "" {
		return passwordResetFailure(request, 5020, "INVALID_PASSWORD")
	}
	score, err := service.strength.Score(request.Password, user.Email)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if score < 3 {
		return passwordResetFailure(request, 5021, "PASSWORD_TOO_WEAK")
	}
	encoded, err := service.passwords.Encode(request.Password)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if err := service.accounts.SetPassword(ctx, user.ID, encoded, false); err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if request.IsSpace {
		return Result{Status: http.StatusFound, Location: request.BaseURL}
	}
	return Result{Status: http.StatusFound, Location: strings.TrimRight(request.BaseURL, "/") + "/sign-in?success=True"}
}

func passwordResetFailure(request Request, code int, message string) Result {
	query := encodeQuery([]QueryParam{
		{Name: "error_code", Value: fmt.Sprint(code)},
		{Name: "error_message", Value: message},
	})
	if request.IsSpace {
		return Result{Status: http.StatusFound, Location: request.BaseURL + "/accounts/reset-password/?" + query}
	}
	return Result{Status: http.StatusFound, Location: strings.TrimRight(request.BaseURL, "/") + "/accounts/reset-password?" + query}
}

func (service *Service) MagicSignIn(ctx context.Context, request Request, app bool) Result {
	return service.magicAuthenticate(ctx, request, app, false)
}

func (service *Service) MagicSignUp(ctx context.Context, request Request, app bool) Result {
	return service.magicAuthenticate(ctx, request, app, true)
}

func (service *Service) magicAuthenticate(ctx context.Context, request Request, app, signup bool) Result {
	if service.magic == nil || service.throttle == nil {
		return Result{Status: http.StatusInternalServerError}
	}
	allowed, err := service.throttle.Allow(ctx, request.ClientIP)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if !allowed {
		return service.failure(request, 5900, "RATE_LIMIT_EXCEEDED", nil)
	}
	request.Email = strings.ToLower(strings.TrimSpace(request.Email))
	request.Code = strings.TrimSpace(request.Code)
	if request.Code == "" || request.Email == "" {
		if signup {
			return service.failure(request, 5055, "MAGIC_SIGN_UP_EMAIL_CODE_REQUIRED", nil)
		}
		return service.failure(request, 5085, "MAGIC_SIGN_IN_EMAIL_CODE_REQUIRED", nil)
	}
	user, lookupErr := service.accounts.UserByEmail(ctx, request.Email)
	if signup && lookupErr == nil {
		return service.failure(request, 5030, "USER_ALREADY_EXIST", nil)
	}
	if signup && !errors.Is(lookupErr, accountauthentication.ErrUserNotFound) {
		return Result{Status: http.StatusInternalServerError}
	}
	if !signup && errors.Is(lookupErr, accountauthentication.ErrUserNotFound) {
		return service.failure(request, 5060, "USER_DOES_NOT_EXIST", nil)
	}
	if !signup && lookupErr != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	smtp, err := service.accounts.ConfigValue(ctx, "EMAIL_HOST", "")
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if smtp == "" {
		return service.failure(request, 5025, "SMTP_NOT_CONFIGURED", map[string]string{"email": request.Email})
	}
	enabled, err := service.accounts.ConfigValue(ctx, "ENABLE_MAGIC_LINK_LOGIN", "1")
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if enabled == "0" {
		return service.failure(request, 5016, "MAGIC_LINK_LOGIN_DISABLED", map[string]string{"email": request.Email})
	}
	if err := service.magic.Verify(ctx, "magic_"+request.Email, request.Code, lookupErr == nil); err != nil {
		var codeError *CodeError
		if errors.As(err, &codeError) {
			payload := map[string]string(nil)
			if codeError.Email != "" {
				payload = map[string]string{"email": codeError.Email}
			}
			return service.failure(request, codeError.Code, codeError.Message, payload)
		}
		return Result{Status: http.StatusInternalServerError}
	}
	if signup {
		signupEnabled, configErr := service.accounts.ConfigValue(ctx, "ENABLE_SIGNUP", "1")
		if configErr != nil {
			return Result{Status: http.StatusInternalServerError}
		}
		if signupEnabled == "0" {
			invited, inviteErr := service.accounts.HasWorkspaceInvite(ctx, request.Email)
			if inviteErr != nil {
				return Result{Status: http.StatusInternalServerError}
			}
			if !invited {
				return service.failure(request, 5015, "SIGNUP_DISABLED", map[string]string{"email": request.Email})
			}
		}
		randomPassword := strings.ReplaceAll(uuid.NewString(), "-", "")
		encoded, encodeErr := service.passwords.Encode(randomPassword)
		if encodeErr != nil {
			return Result{Status: http.StatusInternalServerError}
		}
		user, err = service.accounts.CreateUser(ctx, request.Email, strings.ReplaceAll(uuid.NewString(), "-", ""), encoded, true, true)
		if err != nil {
			return Result{Status: http.StatusInternalServerError}
		}
	} else if result := service.loginEligibility(request, user); result != nil {
		return *result
	}
	return service.completeLogin(ctx, request, user, app, "magic-code", signup, true)
}

func (service *Service) SignOut(ctx context.Context, request Request, userID string) Result {
	if err := service.accounts.UpdateLogoutMetadata(ctx, userID, request.ClientIP, service.requestNow(request)); err != nil {
		return Result{Status: http.StatusFound, Location: service.baseRedirect(request)}
	}
	if err := service.sessions.Flush(ctx, request.CurrentSession); err != nil {
		return Result{Status: http.StatusFound, Location: service.baseRedirect(request)}
	}
	location := request.BaseURL
	if request.IsSpace {
		location = SafeRedirectURL(request.BaseURL, request.NextPath, nil, request.AllowedHosts)
	}
	return Result{Status: http.StatusFound, Location: location, ExpireSession: true}
}

func (service *Service) AdminSignIn(ctx context.Context, request Request) Result {
	repository, ok := service.accounts.(AdminAccountRepository)
	if !ok {
		return Result{Status: http.StatusInternalServerError}
	}
	exists, err := repository.InstanceExists(ctx)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if !exists {
		return service.failure(request, 5000, "INSTANCE_NOT_CONFIGURED", nil)
	}
	if request.Email == "" || request.Password == "" {
		return service.failure(request, 5170, "REQUIRED_ADMIN_EMAIL_PASSWORD", map[string]string{"email": request.Email})
	}
	request.Email = emailvalidation.NormalizePythonEmail(request.Email)
	if !emailvalidation.ValidDjangoEmail(request.Email) {
		return service.failure(request, 5160, "INVALID_ADMIN_EMAIL", map[string]string{"email": request.Email})
	}
	user, err := service.accounts.UserByEmail(ctx, request.Email)
	if errors.Is(err, accountauthentication.ErrUserNotFound) {
		return service.failure(request, 5185, "ADMIN_USER_DOES_NOT_EXIST", map[string]string{"email": request.Email})
	}
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if user.IsBot {
		return service.failure(request, 5175, "ADMIN_AUTHENTICATION_FAILED", map[string]string{"email": request.Email})
	}
	if !user.Active {
		return service.failure(request, 5190, "ADMIN_USER_DEACTIVATED", nil)
	}
	validPassword, verifyErr := service.passwords.Verify(user.Password, request.Password)
	if verifyErr != nil || !validPassword {
		return service.failure(request, 5175, "ADMIN_AUTHENTICATION_FAILED", map[string]string{"email": request.Email})
	}
	admin, err := repository.IsInstanceAdmin(ctx, true, user.ID)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if !admin {
		return service.failure(request, 5175, "ADMIN_AUTHENTICATION_FAILED", map[string]string{"email": request.Email})
	}
	now := service.requestNow(request)
	if _, err := service.accounts.UpdateLoginMetadata(ctx, user.ID, now, request.ClientIP, request.UserAgent, "email", now); err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	device := map[string]string{"user_agent": request.UserAgent, "ip_address": request.ClientIP, "domain": request.BaseURL}
	age := request.SessionAgeSeconds
	if age <= 0 {
		age = defaultSessionAge
	}
	refresh, err := service.sessions.Login(ctx, request.CurrentSession, djangosession.AuthenticatedSession{
		UserID: user.ID, PasswordHash: user.Password, Backend: djangoModelBackend, DeviceInfo: device,
	}, age, now)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	return Result{Status: http.StatusFound, Location: strings.TrimRight(request.BaseURL, "/") + "/general/", Session: &refresh}
}

func (service *Service) AdminSignUp(ctx context.Context, request Request) Result {
	repository, ok := service.accounts.(AdminAccountRepository)
	if !ok {
		return Result{Status: http.StatusInternalServerError}
	}
	exists, setupDone, hasAdmins, err := repository.InitialAdminState(ctx)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if !exists {
		return service.failure(request, 5000, "INSTANCE_NOT_CONFIGURED", nil)
	}
	if setupDone || hasAdmins {
		return service.failure(request, 5150, "ADMIN_ALREADY_EXIST", nil)
	}
	if request.Email == "" || request.Password == "" || request.FirstName == "" {
		return service.failure(request, 5155, "REQUIRED_ADMIN_EMAIL_PASSWORD_FIRST_NAME", map[string]string{
			"email": request.Email, "first_name": request.FirstName, "last_name": request.LastName,
			"company_name": request.CompanyName,
		})
	}
	request.Email = emailvalidation.NormalizePythonEmail(request.Email)
	if !emailvalidation.ValidDjangoEmail(request.Email) {
		return service.failure(request, 5160, "INVALID_ADMIN_EMAIL", map[string]string{
			"email": request.Email, "first_name": request.FirstName, "last_name": request.LastName,
			"company_name": request.CompanyName,
		})
	}
	if _, err := service.accounts.UserByEmail(ctx, request.Email); err == nil {
		return service.failure(request, 5180, "ADMIN_USER_ALREADY_EXIST", map[string]string{
			"email": request.Email, "first_name": request.FirstName, "last_name": request.LastName,
			"company_name": request.CompanyName,
		})
	} else if !errors.Is(err, accountauthentication.ErrUserNotFound) {
		return Result{Status: http.StatusInternalServerError}
	}
	score, err := service.strength.Score(request.Password, "")
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if score < 3 {
		return service.failure(request, 5021, "PASSWORD_TOO_WEAK", map[string]string{
			"email": request.Email, "first_name": request.FirstName, "last_name": request.LastName,
			"company_name": request.CompanyName,
		})
	}
	encoded, err := service.passwords.Encode(request.Password)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	created, err := repository.CreateInitialAdmin(ctx, request.Email, request.FirstName, request.LastName,
		request.CompanyName, encoded, request.ClientIP, request.UserAgent, request.TelemetryEnabled, service.requestNow(request))
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if !created.Exists {
		return service.failure(request, 5000, "INSTANCE_NOT_CONFIGURED", nil)
	}
	// A successful creation also marks setup complete. Only a result without
	// the newly created user represents another request completing setup first.
	if created.AlreadyExists || (created.SetupDone && created.User.ID == "") {
		return service.failure(request, 5150, "ADMIN_ALREADY_EXIST", nil)
	}
	if created.UserAlreadyExists {
		return service.failure(request, 5180, "ADMIN_USER_ALREADY_EXIST", map[string]string{
			"email": request.Email, "first_name": request.FirstName, "last_name": request.LastName,
			"company_name": request.CompanyName,
		})
	}
	user := created.User
	now := service.requestNow(request)
	if _, err := service.accounts.UpdateLoginMetadata(ctx, user.ID, now, request.ClientIP, request.UserAgent, "email", now); err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	device := map[string]string{"user_agent": request.UserAgent, "ip_address": request.ClientIP, "domain": request.BaseURL}
	age := request.SessionAgeSeconds
	if age <= 0 {
		age = defaultSessionAge
	}
	refresh, err := service.sessions.Login(ctx, request.CurrentSession, djangosession.AuthenticatedSession{
		UserID: user.ID, PasswordHash: user.Password, Backend: djangoModelBackend, DeviceInfo: device,
	}, age, now)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	return Result{Status: http.StatusFound, Location: strings.TrimRight(request.BaseURL, "/") + "/general/", Session: &refresh}
}

func (service *Service) AdminSignOut(ctx context.Context, request Request, userID string) Result {
	if userID != "" {
		_ = service.accounts.UpdateLogoutMetadata(ctx, userID, request.ClientIP, service.requestNow(request))
		_ = service.sessions.Flush(ctx, request.CurrentSession)
	}
	return Result{Status: http.StatusFound, Location: SafeRedirectURL(request.BaseURL, "", nil, request.AllowedHosts), ExpireSession: userID != ""}
}

func (service *Service) completeLogin(ctx context.Context, request Request, user accountauthentication.User, app bool, medium string, signup, autoset bool) Result {
	now := service.requestNow(request)
	activated, err := service.accounts.UpdateLoginMetadata(ctx, user.ID, now, request.ClientIP, request.UserAgent, medium, now)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	if activated && service.activation != nil {
		baseOrigin := request.BaseOrigin
		if baseOrigin == "" {
			baseOrigin = request.BaseURL
		}
		_ = service.activation.Task(ctx, "plane.bgtasks.user_activation_email_task.user_activation_email", []any{baseOrigin, user.ID}, map[string]any{})
	}
	if app && service.workflow != nil {
		if err := service.workflow.Run(ctx, user.ID, user.Email); err != nil {
			return Result{Status: http.StatusInternalServerError}
		}
	}
	if medium == "magic-code" {
		if err := service.accounts.EnsureProfile(ctx, user.ID); err != nil {
			return Result{Status: http.StatusInternalServerError}
		}
	}
	device := map[string]string{"user_agent": request.UserAgent, "ip_address": request.ClientIP, "domain": request.BaseURL}
	refresh, err := service.sessions.Login(ctx, request.CurrentSession, djangosession.AuthenticatedSession{
		UserID: user.ID, PasswordHash: user.Password, Backend: djangoModelBackend, DeviceInfo: device,
	}, service.sessionAge(request), now)
	if err != nil {
		return Result{Status: http.StatusInternalServerError}
	}
	path := request.NextPath
	if path == "" || autoset {
		path, err = service.redirectionPath(ctx, user.ID)
		if err != nil {
			return Result{Status: http.StatusInternalServerError}
		}
	}
	if autoset {
		onboarded, _, _, _, profileErr := service.accounts.RedirectProfile(ctx, user.ID)
		if profileErr != nil {
			return Result{Status: http.StatusInternalServerError}
		}
		if onboarded {
			path = "/"
		}
	}
	location := ""
	if request.IsSpace {
		location = SpaceRedirectURL(request.BaseURL, path, request.AllowedHosts)
	} else {
		location = SafeRedirectURL(request.BaseURL, path, nil, request.AllowedHosts)
	}
	return Result{Status: http.StatusFound, Location: location, Session: &refresh}
}

func (service *Service) loginEligibility(request Request, user accountauthentication.User) *Result {
	if !user.Active && user.LastLogoutTime != nil {
		result := service.failure(request, 5019, "USER_ACCOUNT_DEACTIVATED", map[string]string{"email": user.Email})
		return &result
	}
	if user.IsBot {
		result := service.failure(request, 5017, "BOT_USER_LOGIN_FORBIDDEN", map[string]string{"email": user.Email})
		return &result
	}
	return nil
}

func (service *Service) redirectionPath(ctx context.Context, userID string) (string, error) {
	onboarded, lastWorkspace, earliestWorkspace, inviteCount, err := service.accounts.RedirectProfile(ctx, userID)
	if err != nil {
		return "", err
	}
	if !onboarded {
		return "onboarding", nil
	}
	if lastWorkspace != "" {
		return lastWorkspace, nil
	}
	if earliestWorkspace != "" {
		return earliestWorkspace, nil
	}
	if inviteCount > 0 {
		return "invitations", nil
	}
	return "create-workspace", nil
}

func (service *Service) failure(request Request, code int, message string, payload map[string]string) Result {
	params := []QueryParam{{Name: "error_code", Value: fmt.Sprint(code)}, {Name: "error_message", Value: message}}
	for name, value := range payload {
		params = append(params, QueryParam{Name: name, Value: value})
	}
	return Result{Status: http.StatusFound, Location: SafeRedirectURL(request.BaseURL, request.NextPath, params, request.AllowedHosts)}
}

func (service *Service) requestNow(request Request) time.Time {
	if !request.Now.IsZero() {
		return request.Now
	}
	return service.now().UTC()
}

func (service *Service) sessionAge(request Request) int64 {
	if request.SessionAgeSeconds > 0 {
		return request.SessionAgeSeconds
	}
	return defaultSessionAge
}

func (service *Service) baseRedirect(request Request) string {
	if request.IsSpace {
		return SafeRedirectURL(request.BaseURL, request.NextPath, nil, request.AllowedHosts)
	}
	return request.BaseURL
}

func normalizeEmail(value string) (string, bool) {
	value = emailvalidation.NormalizePythonEmail(value)
	return value, value != "" && emailvalidation.ValidDjangoEmail(value)
}
