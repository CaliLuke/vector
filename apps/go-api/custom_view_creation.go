package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/platform"

	api "github.com/CaliLuke/go-plane/customviewcreation/gen/custom_view_creation"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/customviewcreation"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type customViewCreationSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type customViewCreationRepository interface {
	ProjectTarget(context.Context, string, string, string) error
	WorkspaceTarget(context.Context, string, string) error
	UpdateWorkspace(context.Context, string, string, any) error
	UpdateProject(context.Context, string, string, string, customviewcreation.ProjectViewChanges) error
}
type customViewCreationPrincipal struct{ userID, timezone string }
type customViewCreationPrincipalKey struct{}
type customViewCreationService struct {
	repo     customViewCreationRepository
	sessions customViewCreationSessions
}

func NewCustomViewCreation(repo customViewCreationRepository, sessions customViewCreationSessions) (api.Service, error) {
	if repo == nil || sessions == nil {
		return nil, fmt.Errorf("custom view creation dependencies are required")
	}
	return &customViewCreationService{repo: repo, sessions: sessions}, nil
}

func (s *customViewCreationService) APIKeyAuth(ctx context.Context, credential string, _ *security.APIKeyScheme) (context.Context, error) {
	userID, timezone, inspected := readServiceSessionPrincipal(ctx)
	if !inspected {
		if credential == "" {
			credential = sessionCredentialFromContext(ctx)
		}
		if credential != "" {
			principal, err := s.sessions.Validate(ctx, credential)
			if err == nil {
				userID, timezone = principal.UserID, principal.UserTimezone
			}
		}
	}
	if _, err := uuid.Parse(userID); err != nil {
		return ctx, s.error(http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail})
	}
	return context.WithValue(ctx, customViewCreationPrincipalKey{}, customViewCreationPrincipal{userID, timezone}), nil
}

func (s *customViewCreationService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	ctx, err := s.APIKeyAuth(ctx, credential, nil)
	if status, body := customViewCreationProtocolError(err); status != 0 {
		return status, body
	}
	principal, _ := ctx.Value(customViewCreationPrincipalKey{}).(customViewCreationPrincipal)
	if err := djangosession.ValidateUserTimezone(principal.timezone); err != nil {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	return 0, nil
}

func (s *customViewCreationService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, _ SessionProtocolResource) (int, map[string]any) {
	return s.AuthorizeSessionProtocol(ctx, credential)
}

func customViewCreationProtocolError(err error) (int, map[string]any) {
	if err == nil {
		return 0, nil
	}
	var failure *api.CustomViewCreationErrorResponse
	if !errors.As(err, &failure) {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	status, _ := strconv.Atoi(strings.TrimPrefix(failure.Name, "status_"))
	var body map[string]any
	raw, _ := json.Marshal(failure.Body)
	if json.Unmarshal(raw, &body) != nil || body == nil {
		body = map[string]any{}
	}
	return status, body
}

func (s *customViewCreationService) error(status int, body any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		raw = []byte(`{"error":"Something went wrong please try again later"}`)
	}
	value, err := loom.JSONValueFrom(json.RawMessage(raw))
	if err != nil {
		value = loom.JSONValue(`null`)
	}
	return &api.CustomViewCreationErrorResponse{Name: "status_" + strconv.Itoa(status), Body: value}
}

func (s *customViewCreationService) principal(ctx context.Context) (customViewCreationPrincipal, error) {
	principal, ok := ctx.Value(customViewCreationPrincipalKey{}).(customViewCreationPrincipal)
	if !ok {
		return principal, s.error(http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail})
	}
	return principal, nil
}

func (s *customViewCreationService) prepare(ctx context.Context) (customViewCreationPrincipal, error) {
	principal, err := s.principal(ctx)
	if err != nil {
		return principal, err
	}
	if err := djangosession.ValidateUserTimezone(principal.timezone); err != nil {
		return principal, s.error(http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"})
	}
	return principal, nil
}

func customViewCreationInput(ctx context.Context, body io.ReadCloser) (djangoinput.Input, error) {
	if body != nil {
		defer platform.CloseRequestBody(ctx, body)
	}
	contentType := ""
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		contentType = metadata.Headers().Get("Content-Type")
	}
	return djangoinput.Parse(body, contentType)
}

func (s *customViewCreationService) CreateProjectView(ctx context.Context, payload *api.CreateProjectViewPayload, body io.ReadCloser) error {
	principal, err := s.prepare(ctx)
	if err != nil {
		platform.CloseRequestBody(ctx, body)
		return err
	}
	if projectTargetErr := s.repo.ProjectTarget(ctx, principal.userID, payload.Slug, payload.ProjectID); projectTargetErr != nil {
		platform.CloseRequestBody(ctx, body)
		return s.mapError(projectTargetErr)
	}
	input, err := customViewCreationInput(ctx, body)
	if err != nil {
		return s.inputError(err)
	}
	if input.Kind != "" {
		return s.invalidNull()
	}
	changes := customviewcreation.ProjectViewChanges{}
	if changes.ViewProps, changes.HasViewProps = input.Fields["view_props"]; changes.HasViewProps {
		if changes.ViewProps == nil {
			return s.invalidNull()
		}
	}
	if changes.DefaultProps, changes.HasDefaultProps = input.Fields["default_props"]; changes.HasDefaultProps {
		if changes.DefaultProps == nil {
			return s.invalidNull()
		}
	}
	if changes.Preferences, changes.HasPreferences = input.Fields["preferences"]; changes.HasPreferences {
		if changes.Preferences == nil {
			return s.invalidNull()
		}
	}
	if raw, ok := input.Fields["sort_order"]; ok {
		value, message := djangoinput.FloatField(raw)
		if message != "" {
			return s.error(http.StatusBadRequest, map[string]any{"sort_order": []string{message}})
		}
		changes.SortOrder, changes.HasSortOrder = value, true
	}
	if err := s.repo.UpdateProject(ctx, principal.userID, payload.Slug, payload.ProjectID, changes); err != nil {
		return s.mapError(err)
	}
	return nil
}

func (s *customViewCreationService) CreateWorkspaceView(ctx context.Context, payload *api.CreateWorkspaceViewPayload, body io.ReadCloser) error {
	principal, err := s.prepare(ctx)
	if err != nil {
		platform.CloseRequestBody(ctx, body)
		return err
	}
	if workspaceTargetErr := s.repo.WorkspaceTarget(ctx, principal.userID, payload.Slug); workspaceTargetErr != nil {
		platform.CloseRequestBody(ctx, body)
		return s.mapError(workspaceTargetErr)
	}
	input, err := customViewCreationInput(ctx, body)
	if err != nil {
		return s.inputError(err)
	}
	if input.Kind != "" {
		return s.invalidNull()
	}
	viewProps, present := input.Fields["view_props"]
	if !present {
		viewProps = map[string]any{}
	}
	if viewProps == nil {
		return s.invalidNull()
	}
	if err := s.repo.UpdateWorkspace(ctx, principal.userID, payload.Slug, viewProps); err != nil {
		return s.mapError(err)
	}
	return nil
}

func (s *customViewCreationService) invalidNull() error {
	return s.error(http.StatusBadRequest, map[string]any{"error": "The payload is not valid"})
}

func (s *customViewCreationService) inputError(err error) error {
	var failure *djangoapi.HTTPError
	if errors.As(err, &failure) {
		return s.error(failure.Status, failure.Body)
	}
	return s.internal()
}

func (s *customViewCreationService) mapError(err error) error {
	switch {
	case errors.Is(err, customviewcreation.ErrNotFound):
		return s.error(http.StatusNotFound, map[string]any{"error": "The required object does not exist."})
	case errors.Is(err, customviewcreation.ErrForbidden):
		return s.error(http.StatusForbidden, map[string]any{"error": "Forbidden"})
	}
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) && (postgres.Code == "23502" || postgres.Code == "23505") {
		return s.error(http.StatusBadRequest, map[string]any{"error": "The payload is not valid"})
	}
	return s.internal()
}

func (s *customViewCreationService) internal() error {
	return s.error(http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"})
}

var _ api.Service = (*customViewCreationService)(nil)
