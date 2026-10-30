package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/platform"

	api "github.com/CaliLuke/go-plane/gen/instance_management"
	"github.com/CaliLuke/go-plane/internal/accountauthentication"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/authentication/password"
	"github.com/CaliLuke/go-plane/internal/authredirects"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/instancemail"
	"github.com/CaliLuke/go-plane/internal/instancemanagement"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	loomhttp "github.com/CaliLuke/loom/http"
	pkg "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

const instanceManagementInternalMessage = "Something went wrong please try again later"

type instanceManagementRepository interface {
	IsAdmin(context.Context, uuid.UUID) (bool, error)
	Root(context.Context) (json.RawMessage, error)
	PatchRoot(context.Context, uuid.UUID, map[string]json.RawMessage) (json.RawMessage, error)
	ListAdmins(context.Context) (json.RawMessage, error)
	CreateAdmin(context.Context, uuid.UUID, map[string]json.RawMessage) (json.RawMessage, error)
	DeleteAdmin(context.Context, uuid.UUID) error
	Configurations(context.Context) (json.RawMessage, error)
	Telemetry(context.Context) (json.RawMessage, error)
	PatchConfigurations(context.Context, map[string]json.RawMessage) (json.RawMessage, error)
	SMTPConfiguration(context.Context) (instancemail.SMTPConfig, string, error)
	DisableEmail(context.Context) error
	SignupVisited(context.Context) error
	Workspaces(context.Context, string, string, string) (json.RawMessage, error)
	CreateWorkspace(context.Context, uuid.UUID, map[string]json.RawMessage, string) (json.RawMessage, error)
}

type instanceManagementSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type InstanceManagementAdminAccounts interface {
	InstanceExists(context.Context) (bool, error)
	InitialAdminState(context.Context) (bool, bool, bool, error)
	IsInstanceAdmin(context.Context, bool, string) (bool, error)
	CreateInitialAdmin(context.Context, string, string, string, string, string, string, string, bool, time.Time) (accountauthentication.InitialAdminResult, error)
	UserByEmail(context.Context, string) (accountauthentication.User, error)
	UpdateLoginMetadata(context.Context, string, time.Time, string, string, string, time.Time) (bool, error)
	UpdateLogoutMetadata(context.Context, string, string, time.Time) error
}

type instanceManagementCipher interface {
	Encrypt(string) (string, error)
	Decrypt(string) (string, error)
}

type instanceManagementService struct {
	repository     instanceManagementRepository
	sessions       instanceManagementSessions
	cipher         instanceManagementCipher
	mailer         instanceMailSender
	cache          instanceManagementCache
	adminAccounts  InstanceManagementAdminAccounts
	passwords      *password.Hasher
	adminRedirects *authredirects.Service
}

type instanceManagementPrincipalKey struct{}

type instanceManagementCache interface {
	Invalidate(context.Context, string, string, bool) error
}

type instanceMailSender interface {
	Send(context.Context, instancemail.SMTPConfig, instancemail.Message) error
}

func NewInstanceManagement(repository instanceManagementRepository, sessions instanceManagementSessions, cipher instanceManagementCipher, mailer instanceMailSender, cache instanceManagementCache, adminAccounts InstanceManagementAdminAccounts, adminRedirects *authredirects.Service) (api.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("instance management repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("Django admin session validator is required")
	}
	if cipher == nil {
		return nil, fmt.Errorf("instance configuration cipher is required")
	}
	if mailer == nil {
		return nil, fmt.Errorf("instance mail sender is required")
	}
	if cache == nil {
		return nil, fmt.Errorf("instance management cache invalidator is required")
	}
	service := &instanceManagementService{repository: repository, sessions: sessions, cipher: cipher, mailer: mailer, cache: cache, passwords: password.Default(), adminAccounts: adminAccounts, adminRedirects: adminRedirects}
	return service, nil
}

func (service *instanceManagementService) AdminSignIn(ctx context.Context, _ *api.AdminSignInPayload, body io.ReadCloser) (*api.InstanceAdminRedirectResponse, io.ReadCloser, error) {
	return service.adminRedirect(ctx, body, func(request authredirects.Request) authredirects.Result {
		return service.adminRedirects.AdminSignIn(ctx, request)
	})
}

func (service *instanceManagementService) AdminSignUp(ctx context.Context, _ *api.AdminSignUpPayload, body io.ReadCloser) (*api.InstanceAdminRedirectResponse, io.ReadCloser, error) {
	return service.adminRedirect(ctx, body, func(request authredirects.Request) authredirects.Result {
		return service.adminRedirects.AdminSignUp(ctx, request)
	})
}

func (service *instanceManagementService) AdminSignOut(ctx context.Context, _ *api.AdminSignOutPayload, body io.ReadCloser) (*api.InstanceAdminRedirectResponse, io.ReadCloser, error) {
	return service.adminRedirect(ctx, body, func(request authredirects.Request) authredirects.Result {
		return service.adminRedirects.AdminSignOut(ctx, request, authRedirectSessionUser(request.CurrentSession))
	})
}

func (service *instanceManagementService) adminRedirect(ctx context.Context, body io.ReadCloser, run func(authredirects.Request) authredirects.Result) (*api.InstanceAdminRedirectResponse, io.ReadCloser, error) {
	if service.adminAccounts == nil || service.adminRedirects == nil {
		return nil, nil, instanceManagementInternal()
	}
	request, err := service.adminRedirectRequest(ctx, body)
	if err != nil {
		return nil, nil, instanceManagementInternal()
	}
	result := run(request)
	if result.Session != nil {
		djangosession.SetResponseSession(ctx, *result.Session)
	}
	if result.ExpireSession {
		djangosession.ExpireResponseSession(ctx)
	}
	if result.Status != http.StatusFound {
		return nil, nil, instanceManagementInternal()
	}
	return &api.InstanceAdminRedirectResponse{Location: result.Location, ContentType: "text/html; charset=utf-8"}, io.NopCloser(strings.NewReader(djangoRedirectBody(result.Location))), nil
}

func (service *instanceManagementService) adminRedirectRequest(ctx context.Context, body io.ReadCloser) (authredirects.Request, error) {
	if body != nil {
		defer platform.CloseRequestBody(ctx, body)
	}
	input, err := djangoinput.Parse(body, authRedirectRequestContentType(ctx))
	if err != nil {
		return authredirects.Request{}, err
	}
	text := func(key string) string {
		value, present := input.FieldValue(key, false, true, false)
		if !present || value == nil {
			return ""
		}
		parsed, _ := djangoinput.TextField(value, true, 0, nil)
		return parsed
	}
	base, err := service.adminRedirects.ConfigValue(ctx, "ADMIN_BASE_URL", "")
	if err != nil {
		return authredirects.Request{}, err
	}
	if base == "" {
		base, err = service.adminRedirects.ConfigValue(ctx, "WEB_URL", "")
		if err != nil {
			return authredirects.Request{}, err
		}
	}
	adminPath, err := service.adminRedirects.ConfigValue(ctx, "ADMIN_BASE_PATH", "/god-mode/")
	if err != nil {
		return authredirects.Request{}, err
	}
	base = strings.TrimRight(base, "/") + normalizeAdminBasePath(adminPath)
	allowed := make([]string, 0, 4)
	for _, key := range []string{"WEB_URL", "APP_BASE_URL", "ADMIN_BASE_URL", "SPACE_BASE_URL"} {
		configured, configErr := service.adminRedirects.ConfigValue(ctx, key, "")
		if configErr != nil {
			return authredirects.Request{}, configErr
		}
		if parsed, parseErr := url.Parse(configured); parseErr == nil && parsed.Host != "" {
			allowed = append(allowed, parsed.Host)
		}
	}
	var userAgent, clientIP string
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		userAgent, clientIP = metadata.UserAgent, metadata.ClientAddr
	}
	current, _ := djangosession.CurrentSessionFromContext(ctx)
	telemetry := true
	if raw, present := input.FieldValue("is_telemetry_enabled", false, true, false); present {
		telemetry = raw != nil && fmt.Sprint(raw) != ""
	}
	return authredirects.Request{Email: text("email"), Password: text("password"), FirstName: text("first_name"), LastName: text("last_name"), CompanyName: text("company_name"), TelemetryEnabled: telemetry, BaseURL: strings.TrimRight(base, "/"), AllowedHosts: allowed, UserAgent: userAgent, ClientIP: clientIP, CurrentSession: current, Now: time.Now().UTC()}, nil
}

func normalizeAdminBasePath(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
}

func (service *instanceManagementService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	userID, _, inspected := readServiceSessionPrincipal(ctx)
	if !inspected {
		principal, err := service.sessions.Validate(ctx, key)
		if err != nil {
			return ctx, instanceManagementUnauthenticated()
		}
		userID = principal.UserID
	}
	id, err := uuid.Parse(userID)
	if err != nil {
		return ctx, instanceManagementUnauthenticated()
	}
	return context.WithValue(ctx, instanceManagementPrincipalKey{}, id), nil
}

func (service *instanceManagementService) admin(ctx context.Context) (uuid.UUID, error) {
	id, ok := ctx.Value(instanceManagementPrincipalKey{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), instanceManagementUnauthenticated()
	}
	allowed, err := service.repository.IsAdmin(ctx, id)
	if err != nil {
		return uuid.Nil(), instanceManagementError("internal_error", 500, map[string]any{"error": instanceManagementInternalMessage})
	}
	if !allowed {
		return uuid.Nil(), instanceManagementError("forbidden", 403, map[string]any{"detail": instanceAdminForbiddenDetail})
	}
	if timezone, ok := ReadServiceSessionTimezone(ctx); ok {
		if err := djangosession.ValidateUserTimezone(timezone); err != nil {
			return uuid.Nil(), instanceManagementError("bad_request", 400, map[string]any{"error": "The required key does not exist."})
		}
	}
	return id, nil
}

func (service *instanceManagementService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	_, err := service.APIKeyAuth(ctx, credential, &security.APIKeyScheme{Name: "PlaneAdminSessionCookie"})
	return instanceManagementProtocolError(err)
}

func (service *instanceManagementService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, resource SessionProtocolResource) (int, map[string]any) {
	if resource.Pattern == "/api/instances/" && resource.Method != http.MethodPatch || resource.Pattern == "/api/instances/admins/sign-up-screen-visited/" || resource.Pattern == "/api/instances/admins/sign-in/" || resource.Pattern == "/api/instances/admins/sign-up/" || resource.Pattern == "/api/instances/admins/sign-out/" {
		return 0, nil
	}
	ctx, err := service.APIKeyAuth(ctx, credential, &security.APIKeyScheme{Name: "PlaneAdminSessionCookie"})
	if err != nil {
		return instanceManagementProtocolError(err)
	}
	if _, err = service.admin(ctx); err != nil {
		return instanceManagementProtocolError(err)
	}
	if timezone, ok := ReadServiceSessionTimezone(ctx); ok {
		if err := djangosession.ValidateUserTimezone(timezone); err != nil {
			return http.StatusBadRequest, map[string]any{"error": "The required key does not exist."}
		}
	}
	return 0, nil
}

func (service *instanceManagementService) RootGet(ctx context.Context) (*api.InstanceRootResponseMetadata, io.ReadCloser, error) {
	body, err := service.repository.Root(ctx)
	if err != nil {
		return nil, nil, instanceManagementInternal()
	}
	if len(body) == 0 {
		body = json.RawMessage(`null`)
	}
	body, err = formatInstanceManagementDateTimes(body, instanceManagementLocation(ctx))
	if err != nil {
		return nil, nil, instanceManagementInternal()
	}
	return &api.InstanceRootResponseMetadata{ContentType: "application/json", CacheControl: "private, max-age=12"}, rawJSONReadCloser(body), nil
}

func (service *instanceManagementService) RootPatch(ctx context.Context, _ *api.RootPatchPayload, bodyReader io.ReadCloser) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	id, err := service.admin(ctx)
	if err != nil {
		return nil, nil, err
	}
	if invalidateErr := service.invalidate(ctx, "/api/instances/"); invalidateErr != nil {
		return nil, nil, instanceManagementInternal()
	}
	fields, err := parseInstanceManagementBody(ctx, bodyReader)
	if err != nil {
		return nil, nil, instanceManagementInputError(err)
	}
	body, err := service.repository.PatchRoot(ctx, id, fields)
	if err != nil {
		return nil, nil, instanceManagementInternal()
	}
	return instanceManagementRaw(ctx, body, nil)
}

func (service *instanceManagementService) AdminsGet(ctx context.Context, _ *api.AdminsGetPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	if _, err := service.admin(ctx); err != nil {
		return nil, nil, err
	}
	body, err := service.repository.ListAdmins(ctx)
	return instanceManagementRaw(ctx, body, err)
}

func (service *instanceManagementService) AdminsPost(ctx context.Context, _ *api.AdminsPostPayload, bodyReader io.ReadCloser) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	id, err := service.admin(ctx)
	if err != nil {
		return nil, nil, err
	}
	if invalidateErr := service.invalidate(ctx, "/api/instances/"); invalidateErr != nil {
		return nil, nil, instanceManagementInternal()
	}
	fields, err := parseInstanceManagementBody(ctx, bodyReader)
	if err != nil {
		return nil, nil, instanceManagementInputError(err)
	}
	body, err := service.repository.CreateAdmin(ctx, id, fields)
	if err != nil {
		return nil, nil, instanceManagementInternal()
	}
	return instanceManagementRaw(ctx, body, nil)
}

func (service *instanceManagementService) AdminsDeleteCollection(ctx context.Context, _ *api.AdminsDeleteCollectionPayload) error {
	if _, err := service.admin(ctx); err != nil {
		return err
	}
	return instanceManagementError("method_not_allowed", 405, map[string]any{"detail": "Method \"DELETE\" not allowed."})
}

func (service *instanceManagementService) AdminsDetailGet(ctx context.Context, _ *api.AdminsDetailGetPayload) error {
	if _, err := service.admin(ctx); err != nil {
		return err
	}
	return instanceManagementError("method_not_allowed", 405, map[string]any{"detail": "Method \"GET\" not allowed."})
}

func (service *instanceManagementService) AdminsDetailDelete(ctx context.Context, payload *api.AdminsDetailDeletePayload) error {
	if _, err := service.admin(ctx); err != nil {
		return err
	}
	if err := service.invalidate(ctx, "/api/instances/"); err != nil {
		return instanceManagementInternal()
	}
	id, err := uuid.Parse(payload.Pk)
	if err != nil {
		return instanceManagementError("not_found", 404, map[string]any{"detail": "Not found."})
	}
	if err = service.repository.DeleteAdmin(ctx, id); err != nil {
		return instanceManagementInternal()
	}
	return nil
}

func (service *instanceManagementService) SignupVisited(ctx context.Context) error {
	if err := service.invalidate(ctx, "/api/instances/"); err != nil {
		return instanceManagementInternal()
	}
	if err := service.repository.SignupVisited(ctx); err != nil {
		return instanceManagementError("bad_request", 400, map[string]any{"error": "Instance is not configured"})
	}
	return nil
}

func (service *instanceManagementService) ConfigGet(ctx context.Context, payload *api.ConfigGetPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	if _, err := service.admin(ctx); err != nil {
		return nil, nil, err
	}
	body, err := service.repository.Configurations(ctx)
	return instanceManagementRaw(ctx, body, err)
}

func (service *instanceManagementService) TelemetryGet(ctx context.Context, _ *api.TelemetryGetPayload) (*api.InstanceRootResponseMetadata, io.ReadCloser, error) {
	if _, err := service.admin(ctx); err != nil {
		return nil, nil, err
	}
	body, err := service.repository.Telemetry(ctx)
	if err != nil {
		return nil, nil, instanceManagementInternal()
	}
	return &api.InstanceRootResponseMetadata{ContentType: "application/json", CacheControl: "no-store"}, rawJSONReadCloser(body), nil
}

func (service *instanceManagementService) ConfigPatch(ctx context.Context, _ *api.ConfigPatchPayload, bodyReader io.ReadCloser) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	if _, err := service.admin(ctx); err != nil {
		return nil, nil, err
	}
	if err := service.invalidate(ctx, "/api/instances/configurations/"); err != nil {
		return nil, nil, instanceManagementInternal()
	}
	if err := service.invalidate(ctx, "/api/instances/"); err != nil {
		return nil, nil, instanceManagementInternal()
	}
	fields, err := parseInstanceManagementBody(ctx, bodyReader)
	if err != nil {
		return nil, nil, instanceManagementInputError(err)
	}
	body, err := service.repository.PatchConfigurations(ctx, fields)
	if err != nil {
		return nil, nil, instanceManagementInternal()
	}
	return instanceManagementRaw(ctx, body, nil)
}

func (service *instanceManagementService) DisableEmail(ctx context.Context, _ *api.DisableEmailPayload) error {
	if _, err := service.admin(ctx); err != nil {
		return err
	}
	if err := service.invalidate(ctx, "/api/instances/"); err != nil {
		return instanceManagementInternal()
	}
	// Configuration GET is cached independently. Django's source decorator
	// omits this invalidation, so disabling mail leaves stale credentials in the
	// cached response; clear both projections before changing the rows.
	if err := service.invalidate(ctx, "/api/instances/configurations/"); err != nil {
		return instanceManagementInternal()
	}
	if err := service.repository.DisableEmail(ctx); err != nil {
		return instanceManagementError("bad_request", 400, map[string]any{"error": "Failed to disable email configuration"})
	}
	return nil
}

func (service *instanceManagementService) EmailCheck(ctx context.Context, _ *api.EmailCheckPayload, bodyReader io.ReadCloser) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	if _, err := service.admin(ctx); err != nil {
		return nil, nil, err
	}
	fields, err := parseInstanceManagementBody(ctx, bodyReader)
	if err != nil {
		return nil, nil, instanceManagementInputError(err)
	}
	var receiver string
	_ = json.Unmarshal(fields["receiver_email"], &receiver)
	if receiver == "" {
		return nil, nil, instanceManagementError("bad_request", 400, map[string]any{"error": "Receiver email is required"})
	}
	config, senderAddress, err := service.repository.SMTPConfiguration(ctx)
	if err != nil {
		return nil, nil, instanceManagementInternal()
	}
	err = service.mailer.Send(ctx, config, instancemail.Message{From: senderAddress, To: receiver, Subject: "Email Notification from Plane", Body: "This is a sample email notification sent from Plane application."})
	if err != nil {
		message := "Could not send email. Please check your configuration"
		var mailError *instancemail.Error
		if errors.As(err, &mailError) {
			switch mailError.Kind {
			case instancemail.KindHeader:
				message = "Invalid email header."
			case instancemail.KindAuthentication:
				message = "Invalid credentials provided"
			case instancemail.KindConnect:
				message = "Could not connect with the SMTP server."
			case instancemail.KindSender:
				message = "From address is invalid."
			case instancemail.KindDisconnected:
				message = "SMTP server disconnected unexpectedly."
			case instancemail.KindRecipients:
				message = "All recipient addresses were refused."
			case instancemail.KindTimeout:
				message = "Timeout error while trying to connect to the SMTP server."
			case instancemail.KindNetwork:
				message = "Network connection error. Please check your internet connection."
			}
		}
		return nil, nil, instanceManagementError("bad_request", 400, map[string]any{"error": message})
	}
	body, _ := json.Marshal(map[string]string{"message": "Email successfully sent."})
	return instanceManagementRaw(ctx, body, nil)
}

func (service *instanceManagementService) WorkspacesGet(ctx context.Context, payload *api.WorkspacesGetPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	if _, err := service.admin(ctx); err != nil {
		return nil, nil, err
	}
	var search, perPage, cursor string
	if payload.Search != nil {
		search = *payload.Search
	}
	if payload.PerPage != nil {
		perPage = *payload.PerPage
	}
	if payload.Cursor != nil {
		cursor = *payload.Cursor
	}
	body, err := service.repository.Workspaces(ctx, search, perPage, cursor)
	var pagination *instancemanagement.WorkspacePaginationError
	if errors.As(err, &pagination) {
		return nil, nil, instanceManagementError("bad_request", 400, map[string]string{"detail": pagination.Message})
	}
	return instanceManagementRaw(ctx, body, err)
}

func (service *instanceManagementService) WorkspacesPost(ctx context.Context, _ *api.WorkspacesPostPayload, bodyReader io.ReadCloser) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	id, err := service.admin(ctx)
	if err != nil {
		return nil, nil, err
	}
	fields, err := parseInstanceManagementBody(ctx, bodyReader)
	if err != nil {
		return nil, nil, instanceManagementInputError(err)
	}
	timezone, _ := ReadServiceSessionTimezone(ctx)
	body, err := service.repository.CreateWorkspace(ctx, id, fields, timezone)
	if err != nil {
		if validation := instanceManagementWorkspaceValidationError(err); validation != nil {
			return nil, nil, validation
		}
		return nil, nil, instanceManagementInternal()
	}
	return instanceManagementRaw(ctx, body, nil)
}

func (service *instanceManagementService) invalidate(ctx context.Context, path string) error {
	return service.cache.Invalidate(ctx, path, "", false)
}

func parseInstanceManagementBody(ctx context.Context, body io.ReadCloser) (map[string]json.RawMessage, error) {
	if body != nil {
		defer platform.CloseRequestBody(ctx, body)
	}
	contentType := ""
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		contentType = metadata.Headers().Get("Content-Type")
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return nil, err
	}
	if input.Kind != "" {
		return nil, instanceManagementBodyKindError{kind: input.Kind}
	}
	encoded, err := json.Marshal(input.Fields)
	if err != nil {
		return nil, err
	}
	var result map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func instanceManagementInputError(err error) error {
	var bodyKind instanceManagementBodyKindError
	if errors.As(err, &bodyKind) {
		return instanceManagementError("bad_request", 400, map[string]any{
			"non_field_errors": []string{"Invalid data. Expected a dictionary, but got " + bodyKind.kind + "."},
		})
	}
	return instanceManagementError("bad_request", 400, map[string]any{"detail": err.Error()})
}

func instanceManagementWorkspaceValidationError(err error) error {
	var validation *instancemanagement.WorkspaceValidationError
	if !errors.As(err, &validation) {
		return nil
	}
	if validation.Field != "" {
		return instanceManagementError("bad_request", 400, map[string][]string{validation.Field: {validation.Message}})
	}
	if len(validation.Fields) > 0 {
		return instanceManagementError("bad_request", 400, validation.Fields)
	}
	return instanceManagementError("bad_request", 400, map[string]string{"error": validation.Message})
}

type instanceManagementBodyKindError struct{ kind string }

func (err instanceManagementBodyKindError) Error() string {
	return fmt.Sprintf("invalid data: expected dictionary, got %s", err.kind)
}

func instanceManagementRaw(ctx context.Context, body json.RawMessage, err error) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	if err != nil {
		return nil, nil, instanceManagementInternal()
	}
	if len(body) == 0 {
		body = json.RawMessage(`null`)
	}
	body, err = formatInstanceManagementDateTimes(body, instanceManagementLocation(ctx))
	if err != nil {
		return nil, nil, instanceManagementInternal()
	}
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser(body), nil
}

func instanceManagementLocation(ctx context.Context) *time.Location {
	if timezone, ok := ReadServiceSessionTimezone(ctx); ok {
		if location, err := time.LoadLocation(timezone); err == nil {
			return location
		}
	}
	return time.UTC
}

func formatInstanceManagementDateTimes(raw []byte, location *time.Location) ([]byte, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	var visit func(any)
	visit = func(current any) {
		switch item := current.(type) {
		case []any:
			for _, nested := range item {
				visit(nested)
			}
		case map[string]any:
			for key, nested := range item {
				if text, ok := nested.(string); ok && (key == "created_at" || key == "updated_at" || key == "deleted_at" || key == "last_checked_at") {
					if parsed, err := time.Parse(time.RFC3339Nano, text); err == nil {
						item[key] = pythoncompat.DRFDateTimeString(parsed, location)
					}
				} else {
					visit(nested)
				}
			}
		}
	}
	visit(value)
	return json.Marshal(value)
}

func instanceManagementUnauthenticated() error {
	return &api.InstanceAdminSessionUnauthenticated{Detail: instanceAdminUnauthenticatedDetail}
}

func instanceManagementInternal() error {
	return instanceManagementError("internal_error", 500, map[string]any{"error": instanceManagementInternalMessage})
}

func instanceManagementError(kind string, _ int, body any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		encoded = []byte(`{"error":"` + instanceManagementInternalMessage + `"}`)
	}
	return &api.InstanceManagementError{Kind: kind, Body: pkg.JSONValue(strings.TrimSpace(string(encoded)))}
}

func instanceManagementProtocolError(err error) (int, map[string]any) {
	if err == nil {
		return 0, nil
	}
	var unauthenticated *api.InstanceAdminSessionUnauthenticated
	if errors.As(err, &unauthenticated) {
		return http.StatusUnauthorized, map[string]any{"detail": unauthenticated.Detail}
	}
	var forbidden *api.InstanceAdminSessionForbidden
	if errors.As(err, &forbidden) {
		return http.StatusForbidden, map[string]any{"detail": forbidden.Detail}
	}
	var domainError *api.InstanceManagementError
	if errors.As(err, &domainError) {
		var body map[string]any
		_ = json.Unmarshal([]byte(domainError.Body), &body)
		switch domainError.Kind {
		case "bad_request":
			return http.StatusBadRequest, body
		case "forbidden":
			return http.StatusForbidden, body
		case "method_not_allowed":
			return http.StatusMethodNotAllowed, body
		case "not_found":
			return http.StatusNotFound, body
		}
	}
	return http.StatusInternalServerError, map[string]any{"error": instanceManagementInternalMessage}
}
