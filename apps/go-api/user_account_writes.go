package systemapi

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/platform"

	"math/big"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/browsersession"
	"github.com/CaliLuke/go-plane/internal/emailvalidation"
	"github.com/CaliLuke/go-plane/internal/ratelimit"
	"github.com/CaliLuke/go-plane/internal/requestinput"
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
	QueueEmailCode(context.Context, uuid.UUID, string, string, string) error
	ChangeEmail(context.Context, uuid.UUID, string, string, string) (json.RawMessage, error)
	Deactivate(context.Context, uuid.UUID, string, string, string) error
}

type userAccountWritesSessions interface {
	Validate(context.Context, string) (browsersession.Principal, error)
}

type userAccountWritesService struct {
	repository userAccountWritesRepository
	sessions   userAccountWritesSessions
	throttle   *apikey.Throttle
	appBaseURL string
}

type userAccountWritesPrincipal struct {
	userID    uuid.UUID
	sessionID string
	timezone  string
}

type userAccountWritesPrincipalKey struct{}

func NewUserAccountWrites(repository userAccountWritesRepository, sessions userAccountWritesSessions, quota ratelimit.Store, appBaseURL string) (api.Service, error) {
	if repository == nil || sessions == nil || quota == nil {
		return nil, fmt.Errorf("user account writes dependencies are required")
	}
	throttle, err := apikey.NewThrottle(quota, "3/hour")
	if err != nil {
		return nil, err
	}
	return &userAccountWritesService{repository: repository, sessions: sessions, throttle: throttle, appBaseURL: appBaseURL}, nil
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
	if err := browsersession.ValidateUserTimezone(principal.timezone); err != nil {
		if errors.Is(err, browsersession.ErrTimezoneNotFound) {
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
	input, err := parseUserAccountInput(ctx, body, payload.ContentType)
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
	// Validate profile text with the shared input rules.
	fieldErrors := make(map[string][]string)
	for _, field := range []string{"first_name", "last_name", "display_name"} {
		if value, supplied := input.Fields[field]; supplied {
			text, failures := requestinput.CharField(value, field != "display_name")
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
	clientIP := ""
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		clientIP = metadata.ClientAddr
	}
	if err := service.repository.Deactivate(ctx, principal.userID, principal.sessionID, service.appBaseURL, clientIP); err != nil {
		return nil, userAccountWritesRepositoryError(err)
	}
	browsersession.ExpireResponseSession(ctx)
	return io.NopCloser(strings.NewReader("")), nil
}

func (service *userAccountWritesService) UpdateEmail(ctx context.Context, payload *api.UpdateEmailPayload, body io.ReadCloser) (*api.UserAccountWritesResponseMetadata, io.ReadCloser, error) {
	principal, err := service.user(ctx)
	if err != nil {
		return nil, nil, err
	}
	input, err := parseUserAccountInput(ctx, body, payload.ContentType)
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
	if validateNewEmailErr := service.validateNewEmail(newEmail); validateNewEmailErr != nil {
		return nil, nil, validateNewEmailErr
	}
	if strings.TrimSpace(code) == "" {
		return nil, nil, userAccountWritesBadRequest(map[string]any{"error": "Verification code is required"})
	}
	result, err := service.repository.ChangeEmail(ctx, principal.userID, principal.sessionID, newEmail, strings.TrimSpace(code))
	if err != nil {
		return nil, nil, userAccountWritesRepositoryError(err)
	}
	browsersession.ExpireResponseSession(ctx)
	return userAccountWritesRaw(result)
}

func (service *userAccountWritesService) GenerateEmailCode(ctx context.Context, payload *api.GenerateEmailCodePayload, body io.ReadCloser) (*api.UserAccountWritesResponseMetadata, io.ReadCloser, error) {
	principal, err := service.user(ctx)
	if err != nil {
		return nil, nil, err
	}
	// Charge the verification request before parsing its body.
	rate, err := service.throttle.AllowScoped(ctx, "email-verification", principal.userID.String())
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
	input, err := parseUserAccountInput(ctx, body, payload.ContentType)
	if err != nil {
		return nil, nil, userAccountWritesInputError(err)
	}
	newEmail, ok := userAccountStringValue(input.Fields["email"])
	if !ok {
		newEmail = ""
	}
	newEmail = normalizeUserAccountEmail(newEmail)
	if validateNewEmailErr := service.validateNewEmail(newEmail); validateNewEmailErr != nil {
		return nil, nil, validateNewEmailErr
	}
	code, err := emailVerificationCode()
	if err != nil {
		return nil, nil, userAccountWritesBadRequest(map[string]any{"error": "Failed to generate verification code. Please try again."})
	}
	if err := service.repository.QueueEmailCode(ctx, principal.userID, principal.sessionID, newEmail, code); err != nil {
		return nil, nil, userAccountWritesRepositoryError(err)
	}
	return userAccountWritesRaw(json.RawMessage(`{"message":"Verification code sent to email"}`))
}

func (service *userAccountWritesService) validateNewEmail(email string) error {
	if email == "" {
		return userAccountWritesBadRequest(map[string]any{"error": "Email is required"})
	}
	if utf8.RuneCountInString(email) > 255 || !validEmail(email) {
		return userAccountWritesBadRequest(map[string]any{"error": "Invalid email format"})
	}

	return nil
}

func parseUserAccountInput(ctx context.Context, body io.ReadCloser, contentType *string) (requestinput.Input, error) {
	defer platform.CloseRequestBody(ctx, body)
	var reader io.Reader
	if body != nil {
		reader = body
	}
	content := ""
	if contentType != nil {
		content = *contentType
	}
	return requestinput.Parse(reader, content)
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

func userAccountStringValue(value any) (string, bool) {
	result, ok := value.(string)
	return result, ok
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
	var inputErr *apicontract.HTTPError
	if errors.As(err, &inputErr) {
		if inputErr.Status == http.StatusUnsupportedMediaType {
			return userAccountWritesError("unsupported_media_type", inputErr.Body)
		}
		return userAccountWritesBadRequest(inputErr.Body)
	}
	return userAccountWritesBadRequest(map[string]any{"detail": err.Error()})
}
func userAccountWritesRepositoryError(err error) error {
	if errors.Is(err, store.ErrUnauthenticated) {
		return userAccountWritesUnauthenticated()
	}
	var blocker store.DeactivationBlocked
	if errors.As(err, &blocker) {
		return userAccountWritesBadRequest(map[string]any{"error": blocker.Error()})
	}

	messages := []struct {
		err     error
		message string
	}{
		{store.ErrEmailTaken, "An account with this email already exists"},
		{store.ErrEmailUnchanged, "New email must be different from current email"},
		{store.ErrCodeInvalid, "Invalid verification code"},
		{store.ErrCodeExpired, "Verification code has expired or is invalid"},
	}
	for _, entry := range messages {
		if errors.Is(err, entry.err) {
			return userAccountWritesBadRequest(map[string]any{"error": entry.message})
		}
	}
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
