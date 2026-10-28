package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	store "github.com/CaliLuke/go-plane/internal/projectintakewrites"
	api "github.com/CaliLuke/go-plane/projectintakewrites/gen/project_intake_writes"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type projectIntakeWritesRepository interface {
	Authorize(context.Context, string, string, string) error
	ValidateCreate(djangoinput.Input) (map[string]any, error)
	Create(context.Context, string, string, string, map[string]any) (json.RawMessage, error)
	Patch(context.Context, string, string, string, string, map[string]any) (json.RawMessage, error)
	Delete(context.Context, string, string, string, string) error
}
type projectIntakeWritesSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type projectIntakeWritesService struct {
	repository projectIntakeWritesRepository
	sessions   projectIntakeWritesSessions
}
type projectIntakeWritesPrincipal struct{ userID, timezone string }

func NewProjectIntakeWrites(repository projectIntakeWritesRepository, sessions projectIntakeWritesSessions) (api.Service, error) {
	if repository == nil || sessions == nil {
		return nil, errors.New("project intake writes repository and sessions are required")
	}
	return &projectIntakeWritesService{repository: repository, sessions: sessions}, nil
}

func (s *projectIntakeWritesService) APIKeyAuth(ctx context.Context, credential string, _ *security.APIKeyScheme) (context.Context, error) {
	if userID, timezone, inspected := readServiceSessionPrincipal(ctx); inspected {
		if userID == "" {
			return ctx, intakeWriteError(401, map[string]any{"detail": unauthenticatedDetail})
		}
		return context.WithValue(ctx, projectIntakeWritesPrincipal{}, projectIntakeWritesPrincipal{userID, timezone}), nil
	}
	if credential == "" {
		credential = sessionCredentialFromContext(ctx)
	}
	if credential == "" {
		return ctx, intakeWriteError(401, map[string]any{"detail": unauthenticatedDetail})
	}
	validated, err := s.sessions.Validate(ctx, credential)
	if err != nil {
		return ctx, intakeWriteError(401, map[string]any{"detail": unauthenticatedDetail})
	}
	return context.WithValue(ctx, projectIntakeWritesPrincipal{}, projectIntakeWritesPrincipal{validated.UserID, validated.UserTimezone}), nil
}

func (s *projectIntakeWritesService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	validated, err := s.APIKeyAuth(ctx, credential, &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	ctx = validated
	if err != nil {
		return projectIntakeProtocolError(err)
	}
	principal, ok := ctx.Value(projectIntakeWritesPrincipal{}).(projectIntakeWritesPrincipal)
	if !ok {
		return 401, map[string]any{"detail": unauthenticatedDetail}
	}
	if err := djangosession.ValidateUserTimezone(principal.timezone); err != nil {
		return 400, map[string]any{"error": "Invalid timezone"}
	}
	return 0, nil
}

func (s *projectIntakeWritesService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, resource SessionProtocolResource) (int, map[string]any) {
	validated, err := s.APIKeyAuth(ctx, credential, &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	if err != nil {
		return projectIntakeProtocolError(err)
	}
	ctx = validated
	principal, ok := ctx.Value(projectIntakeWritesPrincipal{}).(projectIntakeWritesPrincipal)
	if !ok {
		return 401, map[string]any{"detail": unauthenticatedDetail}
	}
	if err := djangosession.ValidateUserTimezone(principal.timezone); err != nil {
		return 400, map[string]any{"error": "Invalid timezone"}
	}
	// POST checks the project role in perform_create, after serializer/body
	// validation. DELETE checks it in destroy, before the object lookup.
	if resource.Method == "DELETE" {
		if err := s.repository.Authorize(ctx, principal.userID, resource.Slug, resource.ProjectID); err != nil {
			return projectIntakeProtocolError(mapProjectIntakeError(err))
		}
	}
	return 0, nil
}

func projectIntakeProtocolError(err error) (int, map[string]any) {
	if err == nil {
		return 0, nil
	}
	var failure *api.ProjectIntakeWritesErrorResponse
	if !errors.As(err, &failure) {
		return 500, map[string]any{"detail": "Something went wrong please try again later"}
	}
	status, _ := strconv.Atoi(strings.TrimPrefix(failure.Name, "status_"))
	var body map[string]any
	if json.Unmarshal([]byte(failure.Body), &body) != nil || body == nil {
		body = map[string]any{}
	}
	if status == 0 {
		status = 500
	}
	return status, body
}

func intakeWriteError(status int, body map[string]any) error {
	encoded, _ := json.Marshal(body)
	return &api.ProjectIntakeWritesErrorResponse{Name: "status_" + strconv.Itoa(status), Body: loom.JSONValue(encoded)}
}

func (s *projectIntakeWritesService) principal(ctx context.Context) (projectIntakeWritesPrincipal, error) {
	p, ok := ctx.Value(projectIntakeWritesPrincipal{}).(projectIntakeWritesPrincipal)
	if !ok || p.userID == "" {
		return p, intakeWriteError(401, map[string]any{"detail": unauthenticatedDetail})
	}
	if err := djangosession.ValidateUserTimezone(p.timezone); err != nil {
		return p, intakeWriteError(400, map[string]any{"error": "Invalid timezone"})
	}
	return p, nil
}
func projectIntakeFields(ctx context.Context, body io.Reader) (map[string]any, error) {
	parsed, err := djangoinput.Parse(body, requestContentType(ctx))
	if err != nil {
		return nil, intakeWriteError(400, map[string]any{"detail": "Invalid data"})
	}
	return parsed.Fields, nil
}
func projectIntakeInput(ctx context.Context, body io.Reader) (djangoinput.Input, error) {
	parsed, err := djangoinput.Parse(body, requestContentType(ctx))
	if err != nil {
		return djangoinput.Input{}, intakeWriteError(400, map[string]any{"detail": "Invalid data"})
	}
	return parsed, nil
}
func mapProjectIntakeError(err error) error {
	if err == nil {
		return nil
	}
	var response *store.HTTPError
	if errors.As(err, &response) {
		return intakeWriteError(response.Status, response.Body)
	}
	return intakeWriteError(500, map[string]any{"detail": "Something went wrong please try again later"})
}
func projectIntakeResponse(data json.RawMessage, err error) (*api.ProjectIntakeWritesResponseMetadata, io.ReadCloser, error) {
	if err != nil {
		return nil, nil, mapProjectIntakeError(err)
	}
	return &api.ProjectIntakeWritesResponseMetadata{ContentType: "application/json"}, io.NopCloser(bytes.NewReader(data)), nil
}
func (s *projectIntakeWritesService) create(ctx context.Context, slug, project string, body io.ReadCloser) (*api.ProjectIntakeWritesResponseMetadata, io.ReadCloser, error) {
	defer body.Close()
	p, err := s.principal(ctx)
	if err != nil {
		return nil, nil, err
	}
	input, err := projectIntakeInput(ctx, body)
	if err != nil {
		return nil, nil, err
	}
	fields, err := s.repository.ValidateCreate(input)
	if err != nil {
		return nil, nil, mapProjectIntakeError(err)
	}
	if authorizeErr := s.repository.Authorize(ctx, p.userID, slug, project); authorizeErr != nil {
		return nil, nil, mapProjectIntakeError(authorizeErr)
	}
	data, err := s.repository.Create(ctx, p.userID, slug, project, fields)
	return projectIntakeResponse(data, err)
}
func (s *projectIntakeWritesService) patch(ctx context.Context, slug, project, id string, body io.ReadCloser) (*api.ProjectIntakeWritesResponseMetadata, io.ReadCloser, error) {
	defer body.Close()
	p, err := s.principal(ctx)
	if err != nil {
		return nil, nil, err
	}
	if authorizeErr := s.repository.Authorize(ctx, p.userID, slug, project); authorizeErr != nil {
		return nil, nil, mapProjectIntakeError(authorizeErr)
	}
	fields, err := projectIntakeFields(ctx, body)
	if err != nil {
		return nil, nil, err
	}
	data, err := s.repository.Patch(ctx, p.userID, slug, project, id, fields)
	return projectIntakeResponse(data, err)
}
func (s *projectIntakeWritesService) delete(ctx context.Context, slug, project, id string) error {
	p, err := s.principal(ctx)
	if err != nil {
		return err
	}
	if err := s.repository.Authorize(ctx, p.userID, slug, project); err != nil {
		return mapProjectIntakeError(err)
	}
	return mapProjectIntakeError(s.repository.Delete(ctx, p.userID, slug, project, id))
}
func (s *projectIntakeWritesService) CreateInbox(ctx context.Context, payload *api.CreateInboxPayload, body io.ReadCloser) (*api.ProjectIntakeWritesResponseMetadata, io.ReadCloser, error) {
	return s.create(ctx, payload.Slug, payload.ProjectID, body)
}
func (s *projectIntakeWritesService) PatchInbox(ctx context.Context, payload *api.PatchInboxPayload, body io.ReadCloser) (*api.ProjectIntakeWritesResponseMetadata, io.ReadCloser, error) {
	return s.patch(ctx, payload.Slug, payload.ProjectID, payload.Pk, body)
}
func (s *projectIntakeWritesService) DeleteInbox(ctx context.Context, payload *api.DeleteInboxPayload) error {
	return s.delete(ctx, payload.Slug, payload.ProjectID, payload.Pk)
}
func (s *projectIntakeWritesService) CreateIntake(ctx context.Context, payload *api.CreateIntakePayload, body io.ReadCloser) (*api.ProjectIntakeWritesResponseMetadata, io.ReadCloser, error) {
	return s.create(ctx, payload.Slug, payload.ProjectID, body)
}
func (s *projectIntakeWritesService) PatchIntake(ctx context.Context, payload *api.PatchIntakePayload, body io.ReadCloser) (*api.ProjectIntakeWritesResponseMetadata, io.ReadCloser, error) {
	return s.patch(ctx, payload.Slug, payload.ProjectID, payload.Pk, body)
}
func (s *projectIntakeWritesService) DeleteIntake(ctx context.Context, payload *api.DeleteIntakePayload) error {
	return s.delete(ctx, payload.Slug, payload.ProjectID, payload.Pk)
}
