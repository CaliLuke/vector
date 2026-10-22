package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	store "github.com/CaliLuke/go-plane/internal/projectmemberpreferences"
	api "github.com/CaliLuke/go-plane/projectmemberpreferences/gen/project_member_preferences"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type projectMemberPreferencesRepository interface {
	Authorize(context.Context, string, string, string) error
	Lookup(context.Context, string, string, string) error
	Patch(context.Context, string, string, string, string, map[string]any) (map[string]any, error)
}

type projectMemberPreferencesSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type projectMemberPreferencesService struct {
	repository projectMemberPreferencesRepository
	sessions   projectMemberPreferencesSessions
}

type projectMemberPreferencesPrincipal struct{ userID, timezone string }

func NewProjectMemberPreferences(repository projectMemberPreferencesRepository, sessions projectMemberPreferencesSessions) (api.Service, error) {
	if repository == nil || sessions == nil {
		return nil, errors.New("project member preferences repository and sessions are required")
	}
	return &projectMemberPreferencesService{repository: repository, sessions: sessions}, nil
}

func (s *projectMemberPreferencesService) APIKeyAuth(ctx context.Context, credential string, _ *security.APIKeyScheme) (context.Context, error) {
	if userID, timezone, inspected := readServiceSessionPrincipal(ctx); inspected {
		if userID == "" {
			return ctx, preferenceError(401, map[string]any{"detail": unauthenticatedDetail})
		}
		return context.WithValue(ctx, projectMemberPreferencesPrincipal{}, projectMemberPreferencesPrincipal{userID, timezone}), nil
	}
	if credential == "" {
		credential = sessionCredentialFromContext(ctx)
	}
	if credential == "" {
		return ctx, preferenceError(401, map[string]any{"detail": unauthenticatedDetail})
	}
	validated, err := s.sessions.Validate(ctx, credential)
	if err != nil {
		return ctx, preferenceError(401, map[string]any{"detail": unauthenticatedDetail})
	}
	return context.WithValue(ctx, projectMemberPreferencesPrincipal{}, projectMemberPreferencesPrincipal{validated.UserID, validated.UserTimezone}), nil
}

func (s *projectMemberPreferencesService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	_, err := s.APIKeyAuth(ctx, credential, &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	return preferenceProtocolError(err)
}

func (s *projectMemberPreferencesService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, resource SessionProtocolResource) (int, map[string]any) {
	validated, err := s.APIKeyAuth(ctx, credential, &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	if status, body := preferenceProtocolError(err); status != 0 {
		return status, body
	}
	ctx = validated
	if resource.Slug == "" || resource.ProjectID == "" {
		return 0, nil
	}
	principal, ok := ctx.Value(projectMemberPreferencesPrincipal{}).(projectMemberPreferencesPrincipal)
	if !ok || principal.userID == "" {
		return http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail}
	}
	if err := djangosession.ValidateUserTimezone(principal.timezone); err != nil {
		return http.StatusBadRequest, map[string]any{"error": "Invalid timezone"}
	}
	// The source endpoint has a role decorator on PATCH only. OPTIONS and
	// unsupported methods still pass through session/timezone middleware, then
	// resolve to their normal protocol response without a project-role check.
	if resource.Method != http.MethodPatch {
		return 0, nil
	}
	if err := s.repository.Authorize(ctx, principal.userID, resource.Slug, resource.ProjectID); err != nil {
		return preferenceProtocolError(preferenceErrorFromRepository(err))
	}
	return 0, nil
}

func preferenceErrorFromRepository(err error) error {
	var response *store.HTTPError
	if errors.As(err, &response) {
		return preferenceError(response.Status, response.Body)
	}
	return preferenceError(500, map[string]any{"detail": "Something went wrong please try again later"})
}

func preferenceProtocolError(err error) (int, map[string]any) {
	if err == nil {
		return 0, nil
	}
	var failure *api.PreferenceError
	if !errors.As(err, &failure) {
		return http.StatusInternalServerError, map[string]any{"detail": "Something went wrong please try again later"}
	}
	status, _ := strconv.Atoi(strings.TrimPrefix(failure.Name, "status_"))
	var body map[string]any
	if json.Unmarshal([]byte(failure.Body), &body) != nil || body == nil {
		body = map[string]any{}
	}
	if status == 0 {
		status = http.StatusInternalServerError
	}
	return status, body
}

func preferenceError(status int, body map[string]any) error {
	name := "status_500"
	if statusName, ok := map[int]string{400: "status_400", 401: "status_401", 403: "status_403", 404: "status_404", 415: "status_415"}[status]; ok {
		name = statusName
	}
	encoded, _ := json.Marshal(body)
	return &api.PreferenceError{Name: name, Body: loom.JSONValue(encoded)}
}

func (s *projectMemberPreferencesService) Patch(ctx context.Context, payload *api.PatchPayload, body io.ReadCloser) (*api.Preferences, error) {
	if body != nil {
		defer body.Close()
	}
	identity, ok := ctx.Value(projectMemberPreferencesPrincipal{}).(projectMemberPreferencesPrincipal)
	if !ok || identity.userID == "" {
		return nil, preferenceError(401, map[string]any{"detail": unauthenticatedDetail})
	}
	if err := djangosession.ValidateUserTimezone(identity.timezone); err != nil {
		return nil, preferenceError(400, map[string]any{"error": "Invalid timezone"})
	}
	if err := s.repository.Authorize(ctx, identity.userID, payload.Slug, payload.ProjectID); err != nil {
		return nil, preferenceErrorFromRepository(err)
	}
	if err := s.repository.Lookup(ctx, payload.Slug, payload.ProjectID, payload.MemberID); err != nil {
		return nil, preferenceErrorFromRepository(err)
	}
	contentType := requestContentType(ctx)
	parsed, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return nil, preferenceError(400, map[string]any{"detail": "Invalid data"})
	}
	if parsed.Kind != "" {
		return nil, preferenceError(400, map[string]any{"preferences": []string{"Expected an object."}})
	}
	values := map[string]any{}
	for key, value := range parsed.Fields {
		values[key] = value
	}
	result, err := s.repository.Patch(ctx, identity.userID, payload.Slug, payload.ProjectID, payload.MemberID, values)
	if err != nil {
		var httpError *store.HTTPError
		if errors.As(err, &httpError) {
			return nil, preferenceError(httpError.Status, httpError.Body)
		}
		return nil, preferenceError(500, map[string]any{"detail": "Something went wrong please try again later"})
	}
	encoded, err := json.Marshal(result["preferences"])
	if err != nil {
		return nil, preferenceError(500, map[string]any{"detail": "Something went wrong please try again later"})
	}
	return &api.Preferences{Preferences: json.RawMessage(encoded)}, nil
}
