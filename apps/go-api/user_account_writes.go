package systemapi

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"uuid"

	"math/big"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/emailvalidation"
	store "github.com/CaliLuke/go-plane/internal/useraccountwrites"
	api "github.com/CaliLuke/go-plane/user_account_writes/gen/user_account_writes"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

const userAccountWritesInternalMessage = "Something went wrong please try again later"

type userAccountWritesRepository interface {
	UpdateMe(context.Context, uuid.UUID, map[string]json.RawMessage) (json.RawMessage, error)
	Me(context.Context, uuid.UUID) (json.RawMessage, error)
	EmailExists(context.Context, uuid.UUID, string) (bool, error)
	UpdateEmail(context.Context, uuid.UUID, string) (string, error)
	Logout(context.Context, string) error
	DeactivationBlocker(context.Context, uuid.UUID) (string, error)
	Deactivate(context.Context, uuid.UUID, string, string) error
}

type userAccountWritesSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type userAccountWritesCache interface {
	Get(context.Context, string) ([]byte, bool, error)
	Set(context.Context, string, []byte, time.Duration) error
	Delete(context.Context, string) error
}

type userAccountWritesTasks interface {
	Task(context.Context, string, []any, map[string]any) error
}

type userAccountWritesService struct {
	repository userAccountWritesRepository
	sessions   userAccountWritesSessions
	cache      userAccountWritesCache
	tasks      userAccountWritesTasks
	throttle   *apikey.Throttle
	appBaseURL string
}

type userAccountWritesPrincipal struct {
	userID    uuid.UUID
	sessionID string
	timezone  string
}

type userAccountWritesPrincipalKey struct{}

func NewUserAccountWrites(repository userAccountWritesRepository, sessions userAccountWritesSessions, cache userAccountWritesCache, tasks userAccountWritesTasks, appBaseURL string) (api.Service, error) {
	if repository == nil || sessions == nil || cache == nil || tasks == nil {
		return nil, fmt.Errorf("user account writes dependencies are required")
	}
	throttle, err := apikey.NewThrottle(cache, "3/hour")
	if err != nil {
		return nil, err
	}
	return &userAccountWritesService{repository: repository, sessions: sessions, cache: cache, tasks: tasks, throttle: throttle, appBaseURL: appBaseURL}, nil
}

func (service *userAccountWritesService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	userID, timezone, inspected := readServiceSessionPrincipal(ctx)
	if inspected {
		if userID == "" {
			return ctx, userAccountWritesUnauthenticated()
		}
		parsed, err := uuid.Parse(userID)
		if err != nil {
			return ctx, userAccountWritesUnauthenticated()
		}
		return context.WithValue(ctx, userAccountWritesPrincipalKey{}, userAccountWritesPrincipal{
			userID: parsed, sessionID: sessionCredentialFromContext(ctx), timezone: timezone,
		}), nil
	}
	if key == "" {
		key = sessionCredentialFromContext(ctx)
	}
	if key == "" {
		return ctx, userAccountWritesUnauthenticated()
	}
	validated, err := service.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, userAccountWritesUnauthenticated()
	}
	id, err := uuid.Parse(validated.UserID)
	if err != nil {
		return ctx, userAccountWritesUnauthenticated()
	}
	return context.WithValue(ctx, userAccountWritesPrincipalKey{}, userAccountWritesPrincipal{userID: id, sessionID: validated.SessionKey, timezone: validated.UserTimezone}), nil
}

func (service *userAccountWritesService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	validated, err := service.APIKeyAuth(ctx, credential, &security.APIKeyScheme{Name: "PlaneUserAccountWritesSessionCookie"})
	if status, body := userAccountWritesProtocolError(err); status != 0 {
		return status, body
	}
	return service.validateSessionTimezone(validated)
}

func (service *userAccountWritesService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, _ SessionProtocolResource) (int, map[string]any) {
	return service.AuthorizeSessionProtocol(ctx, credential)
}

func (service *userAccountWritesService) validateSessionTimezone(ctx context.Context) (int, map[string]any) {
	principal, ok := ctx.Value(userAccountWritesPrincipalKey{}).(userAccountWritesPrincipal)
	if !ok {
		return 401, map[string]any{"detail": "Authentication credentials were not provided."}
	}
	if err := djangosession.ValidateUserTimezone(principal.timezone); err != nil {
		if errors.Is(err, djangosession.ErrTimezoneNotFound) {
			return 400, map[string]any{"error": "The required key does not exist."}
		}
		return 500, map[string]any{"error": userAccountWritesInternalMessage}
	}
	return 0, nil
}

func (service *userAccountWritesService) user(ctx context.Context) (userAccountWritesPrincipal, error) {
	principal, ok := ctx.Value(userAccountWritesPrincipalKey{}).(userAccountWritesPrincipal)
	if !ok || principal.userID == uuid.Nil() {
		return userAccountWritesPrincipal{}, userAccountWritesUnauthenticated()
	}
	return principal, nil
}

func (service *userAccountWritesService) PatchMe(ctx context.Context, payload *api.PatchMePayload, body io.ReadCloser) (*api.UserAccountWritesResponseMetadata, io.ReadCloser, error) {
	principal, err := service.user(ctx)
	if err != nil {
		return nil, nil, err
	}
	input, err := parseUserAccountInput(body, payload.ContentType)
	if err != nil {
		return nil, nil, userAccountWritesInputError(err)
	}
	if input.Null {
		return nil, nil, userAccountWritesBadRequest(map[string]any{"non_field_errors": []string{"No data provided"}})
	}
	if input.Kind != "" {
		return nil, nil, userAccountWritesBadRequest(map[string]any{
			"non_field_errors": []string{"Invalid data. Expected a dictionary, but got " + strings.ToLower(input.Kind) + "."},
		})
	}
	// UserSerializer applies the model CharField validators before saving.
	// Reuse the shared DRF coercion, Unicode length, and surrogate validation.
	fieldErrors := make(map[string][]string)
	for _, field := range []string{"first_name", "last_name", "display_name"} {
		if value, supplied := input.Fields[field]; supplied {
			text, failures := djangoinput.CharField(value, field != "display_name", input.Surrogates[field])
			if len(failures) != 0 {
				fieldErrors[field] = failures
			} else {
				input.Fields[field] = text
			}
		}
	}
	if len(fieldErrors) != 0 {
		return nil, nil, userAccountWritesBadRequest(fieldErrors)
	}
	fields := make(map[string]json.RawMessage, len(input.Fields))
	for key, value := range input.Fields {
		encoded, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			return nil, nil, userAccountWritesInternal()
		}
		fields[key] = encoded
	}
	result, err := service.repository.UpdateMe(ctx, principal.userID, fields)
	if err != nil {
		return nil, nil, userAccountWritesRepositoryError(err)
	}
	return userAccountWritesRaw(result)
}

func (service *userAccountWritesService) Deactivate(ctx context.Context, _ *api.DeactivatePayload) (io.ReadCloser, error) {
	principal, err := service.user(ctx)
	if err != nil {
		return nil, err
	}
	blocker, err := service.repository.DeactivationBlocker(ctx, principal.userID)
	if err != nil {
		return nil, userAccountWritesInternal()
	}
	if blocker != "" {
		return nil, userAccountWritesBadRequest(map[string]any{"error": blocker})
	}
	password, err := randomDeactivationPassword()
	if err != nil {
		return nil, userAccountWritesInternal()
	}
	hash, err := djangoPasswordHash(password)
	if err != nil {
		return nil, userAccountWritesInternal()
	}
	clientIP := ""
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		clientIP = metadata.ClientAddr
	}
	if err := service.repository.Deactivate(ctx, principal.userID, hash, clientIP); err != nil {
		return nil, userAccountWritesInternal()
	}
	// Django's logout(request) clears the session cookie after deleting every
	// persisted session for the deactivated user.
	djangosession.ExpireResponseSession(ctx)
	if err := service.tasks.Task(ctx, "plane.bgtasks.user_deactivation_email_task.user_deactivation_email", []any{service.appBaseURL, principal.userID.String()}, nil); err != nil {
		return nil, userAccountWritesInternal()
	}
	return io.NopCloser(strings.NewReader("")), nil
}

func (service *userAccountWritesService) UpdateEmail(ctx context.Context, payload *api.UpdateEmailPayload, body io.ReadCloser) (*api.UserAccountWritesResponseMetadata, io.ReadCloser, error) {
	principal, err := service.user(ctx)
	if err != nil {
		return nil, nil, err
	}
	input, err := parseUserAccountInput(body, payload.ContentType)
	if err != nil {
		return nil, nil, userAccountWritesInputError(err)
	}
	newEmail, ok := userAccountStringValue(input.Fields["email"])
	if !ok {
		return nil, nil, userAccountWritesBadRequest(map[string]any{"error": "Invalid email format"})
	}
	newEmail = normalizeUserAccountEmail(newEmail)
	code, ok := userAccountStringValue(input.Fields["code"])
	if !ok {
		code = ""
	}
	if err := service.validateNewEmail(ctx, principal.userID, newEmail); err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(code) == "" {
		return nil, nil, userAccountWritesBadRequest(map[string]any{"error": "Verification code is required"})
	}
	cacheKey := userEmailCacheKey(principal.userID, newEmail)
	cached, found, err := service.cache.Get(ctx, cacheKey)
	if err != nil {
		return nil, nil, userAccountWritesInternal()
	}
	if !found {
		return nil, nil, userAccountWritesBadRequest(map[string]any{"error": "Verification code has expired or is invalid"})
	}
	storedCode, err := store.DecodeEmailVerificationCodeValue(cached)
	if err != nil {
		return nil, nil, userAccountWritesBadRequest(map[string]any{"error": "Failed to verify code. Please try again."})
	}
	if storedCode != strings.TrimSpace(code) {
		return nil, nil, userAccountWritesBadRequest(map[string]any{"error": "Invalid verification code"})
	}
	// The source checks uniqueness again immediately before updating because
	// another account may claim the address after code generation.
	if err := service.validateNewEmail(ctx, principal.userID, newEmail); err != nil {
		return nil, nil, err
	}
	oldEmail, err := service.repository.UpdateEmail(ctx, principal.userID, newEmail)
	if err != nil {
		return nil, nil, userAccountWritesRepositoryError(err)
	}
	if err := service.cache.Delete(ctx, cacheKey); err != nil {
		return nil, nil, userAccountWritesInternal()
	}
	if err := service.repository.Logout(ctx, principal.sessionID); err != nil {
		return nil, nil, userAccountWritesInternal()
	}
	// Django logout() removes the session and expires the browser cookie in
	// SessionMiddleware's response finalizer.
	djangosession.ExpireResponseSession(ctx)
	if err := service.tasks.Task(ctx, "plane.bgtasks.user_email_update_task.send_email_update_confirmation", []any{newEmail}, nil); err != nil {
		return nil, nil, userAccountWritesInternal()
	}
	if err := service.tasks.Task(ctx, "plane.bgtasks.user_email_update_task.send_email_update_confirmation", []any{oldEmail}, nil); err != nil {
		return nil, nil, userAccountWritesInternal()
	}
	result, err := service.repository.Me(ctx, principal.userID)
	if err != nil {
		return nil, nil, userAccountWritesInternal()
	}
	return userAccountWritesRaw(result)
}

func (service *userAccountWritesService) GenerateEmailCode(ctx context.Context, payload *api.GenerateEmailCodePayload, body io.ReadCloser) (*api.UserAccountWritesResponseMetadata, io.ReadCloser, error) {
	principal, err := service.user(ctx)
	if err != nil {
		return nil, nil, err
	}
	// DRF applies UserRateThrottle in APIView.initial before parsing or running
	// the action body.
	rate, err := service.throttle.AllowCacheKey(ctx, ":1:throttle_email_verification_"+principal.userID.String(), "email verification")
	if err != nil {
		return nil, nil, userAccountWritesInternal()
	}
	if !rate.Allowed {
		return nil, nil, userAccountWritesRateLimited(rate.Wait)
	}
	if status, body := service.validateSessionTimezone(ctx); status != 0 {
		if status == 400 {
			return nil, nil, userAccountWritesBadRequest(body)
		}
		return nil, nil, userAccountWritesInternal()
	}
	input, err := parseUserAccountInput(body, payload.ContentType)
	if err != nil {
		return nil, nil, userAccountWritesInputError(err)
	}
	newEmail, ok := userAccountStringValue(input.Fields["email"])
	if !ok {
		newEmail = ""
	}
	newEmail = normalizeUserAccountEmail(newEmail)
	if err := service.validateNewEmail(ctx, principal.userID, newEmail); err != nil {
		return nil, nil, err
	}
	code, err := emailVerificationCode()
	if err != nil {
		return nil, nil, userAccountWritesBadRequest(map[string]any{"error": "Failed to generate verification code. Please try again."})
	}
	encoded, err := store.EncodeEmailVerificationCodeValue(code)
	if err != nil {
		return nil, nil, userAccountWritesInternal()
	}
	key := userEmailCacheKey(principal.userID, newEmail)
	if err := service.cache.Set(ctx, key, encoded, 10*time.Minute); err != nil {
		return nil, nil, userAccountWritesBadRequest(map[string]any{"error": "Failed to generate verification code. Please try again."})
	}
	if err := service.tasks.Task(ctx, "plane.bgtasks.user_email_update_task.send_email_update_magic_code", []any{newEmail, code}, nil); err != nil {
		return nil, nil, userAccountWritesBadRequest(map[string]any{"error": "Failed to generate verification code. Please try again."})
	}
	return userAccountWritesRaw(json.RawMessage(`{"message":"Verification code sent to email"}`))
}

func (service *userAccountWritesService) validateNewEmail(ctx context.Context, userID uuid.UUID, email string) error {
	if email == "" {
		return userAccountWritesBadRequest(map[string]any{"error": "Email is required"})
	}
	if !validEmail(email) {
		return userAccountWritesBadRequest(map[string]any{"error": "Invalid email format"})
	}
	current, err := service.repository.Me(ctx, userID)
	if err != nil {
		return userAccountWritesRepositoryError(err)
	}
	var me struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(current, &me); err != nil {
		return userAccountWritesInternal()
	}
	if email == me.Email {
		return userAccountWritesBadRequest(map[string]any{"error": "New email must be different from current email"})
	}
	exists, err := service.repository.EmailExists(ctx, userID, email)
	if err != nil {
		return userAccountWritesInternal()
	}
	if exists {
		return userAccountWritesBadRequest(map[string]any{"error": "An account with this email already exists"})
	}
	return nil
}

func parseUserAccountInput(body io.ReadCloser, contentType *string) (djangoinput.Input, error) {
	defer func() {
		if body != nil {
			_ = body.Close()
		}
	}()
	var reader io.Reader
	if body != nil {
		reader = body
	}
	content := ""
	if contentType != nil {
		content = *contentType
	}
	return djangoinput.Parse(reader, content)
}

func emailVerificationCode() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()+100000), nil
}

func validEmail(value string) bool {
	return emailvalidation.ValidDjangoEmail(value)
}

func normalizeUserAccountEmail(value string) string {
	return emailvalidation.NormalizePythonEmail(value)
}

func randomDeactivationPassword() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func djangoPasswordHash(password string) (string, error) {
	saltBytes := make([]byte, 12)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", err
	}
	salt := hex.EncodeToString(saltBytes)
	const iterations = 1_000_000
	derived, err := pbkdf2.Key(sha256.New, password, []byte(salt), iterations, 32)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pbkdf2_sha256$%d$%s$%s", iterations, salt, base64.StdEncoding.EncodeToString(derived)), nil
}

func userAccountStringValue(value any) (string, bool) {
	result, ok := value.(string)
	return result, ok
}
func userEmailCacheKey(userID uuid.UUID, email string) string {
	return ":1:magic_email_update_" + userID.String() + "_" + email
}

func userAccountWritesRaw(body json.RawMessage) (*api.UserAccountWritesResponseMetadata, io.ReadCloser, error) {
	if len(body) == 0 {
		body = json.RawMessage(`null`)
	}
	return &api.UserAccountWritesResponseMetadata{ContentType: "application/json"}, io.NopCloser(strings.NewReader(string(body))), nil
}

func userAccountWritesUnauthenticated() error {
	return userAccountWritesError("unauthenticated", map[string]any{"detail": "Authentication credentials were not provided."})
}
func userAccountWritesInternal() error {
	return userAccountWritesError("internal_error", map[string]any{"error": userAccountWritesInternalMessage})
}
func userAccountWritesBadRequest(body any) error { return userAccountWritesError("bad_request", body) }
func userAccountWritesRateLimited(wait time.Duration) error {
	err := userAccountWritesError("rate_limited", map[string]any{"error_code": 5900, "error_message": "RATE_LIMIT_EXCEEDED"})
	if wait > 0 {
		seconds := int(wait / time.Second)
		err.RetryAfter = &seconds
	}
	return err
}
func userAccountWritesError(name string, body any) *api.UserAccountWritesError {
	encoded, err := json.Marshal(body)
	if err != nil {
		encoded = []byte(`{"error":"Something went wrong please try again later"}`)
	}
	value, err := loom.JSONValueFrom(json.RawMessage(encoded))
	if err != nil {
		value = loom.JSONValue(`{"error":"Something went wrong please try again later"}`)
	}
	return &api.UserAccountWritesError{Name: name, Body: value}
}
func userAccountWritesInputError(err error) error {
	var inputErr *djangoapi.HTTPError
	if errors.As(err, &inputErr) {
		if inputErr.Status == http.StatusUnsupportedMediaType {
			return userAccountWritesError("unsupported_media_type", inputErr.Body)
		}
		return userAccountWritesBadRequest(inputErr.Body)
	}
	return userAccountWritesBadRequest(map[string]any{"detail": err.Error()})
}
func userAccountWritesRepositoryError(err error) error {
	var fieldError store.FieldError
	if errors.As(err, &fieldError) {
		return userAccountWritesBadRequest(fieldError)
	}
	if errors.Is(err, store.ErrInvalid) {
		return userAccountWritesBadRequest(map[string]any{"error": err.Error()})
	}
	return userAccountWritesInternal()
}
func userAccountWritesProtocolError(err error) (int, map[string]any) {
	if err == nil {
		return 0, nil
	}
	var typed *api.UserAccountWritesError
	if errors.As(err, &typed) {
		var body map[string]any
		_ = json.Unmarshal([]byte(typed.Body.String()), &body)
		status := 500
		switch typed.Name {
		case "unauthenticated":
			status = 401
		case "bad_request":
			status = 400
		case "rate_limited":
			status = 429
		case "unsupported_media_type":
			status = http.StatusUnsupportedMediaType
		}
		return status, body
	}
	return 500, map[string]any{"error": userAccountWritesInternalMessage}
}
