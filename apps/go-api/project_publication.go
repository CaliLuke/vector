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

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/projectpublication"
	api "github.com/CaliLuke/go-plane/projectpublication/gen/project_publication"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type projectPublicationSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type projectPublicationRepository interface {
	Authorize(context.Context, string, string, string, string) error
	Exists(context.Context, string, string, string) error
	Create(context.Context, string, string, string, projectpublication.CreateInput) (json.RawMessage, error)
	Update(context.Context, string, string, string, string, projectpublication.Patch) (json.RawMessage, error)
	Delete(context.Context, string, string, string, string) error
}
type projectPublicationPrincipal struct{ userID, timezone string }
type projectPublicationPrincipalKey struct{}
type projectPublicationService struct {
	repo     projectPublicationRepository
	sessions projectPublicationSessions
}

func NewProjectPublication(repo projectPublicationRepository, sessions projectPublicationSessions) (api.Service, error) {
	if repo == nil || sessions == nil {
		return nil, fmt.Errorf("project publication dependencies are required")
	}
	return &projectPublicationService{repo: repo, sessions: sessions}, nil
}
func (s *projectPublicationService) APIKeyAuth(ctx context.Context, credential string, _ *security.APIKeyScheme) (context.Context, error) {
	userID, timezone, inspected := readServiceSessionPrincipal(ctx)
	if !inspected {
		if credential == "" {
			credential = sessionCredentialFromContext(ctx)
		}
		if credential != "" {
			if p, err := s.sessions.Validate(ctx, credential); err == nil {
				userID, timezone = p.UserID, p.UserTimezone
			}
		}
	}
	if _, err := uuid.Parse(userID); err != nil {
		return ctx, s.err(http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail})
	}
	return context.WithValue(ctx, projectPublicationPrincipalKey{}, projectPublicationPrincipal{userID, timezone}), nil
}
func (s *projectPublicationService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	_, err := s.APIKeyAuth(ctx, credential, nil)
	return projectPublicationProtocolError(err)
}
func (s *projectPublicationService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, resource SessionProtocolResource) (int, map[string]any) {
	validated, err := s.APIKeyAuth(ctx, credential, nil)
	if status, body := projectPublicationProtocolError(err); status != 0 {
		return status, body
	}
	principal, _ := validated.Value(projectPublicationPrincipalKey{}).(projectPublicationPrincipal)
	if err := s.repo.Authorize(validated, principal.userID, resource.Method, resource.Slug, resource.ProjectID); err != nil {
		return projectPublicationProtocolError(s.mapError(err))
	}
	if err := djangosession.ValidateUserTimezone(principal.timezone); err != nil {
		return http.StatusBadRequest, map[string]any{"error": "The required key does not exist."}
	}
	return 0, nil
}
func projectPublicationProtocolError(err error) (int, map[string]any) {
	if err == nil {
		return 0, nil
	}
	var failure *api.ProjectPublicationErrorResponse
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
func (s *projectPublicationService) err(status int, body any) error {
	raw, e := json.Marshal(body)
	if e != nil {
		raw = []byte(`{"error":"Something went wrong please try again later"}`)
	}
	value, e := loom.JSONValueFrom(json.RawMessage(raw))
	if e != nil {
		value = loom.JSONValue(`null`)
	}
	return &api.ProjectPublicationErrorResponse{Name: "status_" + strconv.Itoa(status), Body: value}
}
func (s *projectPublicationService) principal(ctx context.Context) (projectPublicationPrincipal, error) {
	p, ok := ctx.Value(projectPublicationPrincipalKey{}).(projectPublicationPrincipal)
	if !ok {
		return p, s.err(401, map[string]any{"detail": unauthenticatedDetail})
	}
	if e := djangosession.ValidateUserTimezone(p.timezone); e != nil {
		return p, s.err(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	return p, nil
}
func publicationInput(ctx context.Context, body io.ReadCloser) (djangoinput.Input, error) {
	if body != nil {
		defer body.Close()
	}
	contentType := ""
	if m, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		contentType = m.Headers().Get("Content-Type")
	}
	return djangoinput.Parse(body, contentType)
}
func (s *projectPublicationService) Create(ctx context.Context, p *api.CreatePayload, body io.ReadCloser) (*api.ProjectPublicationRawJSON, io.ReadCloser, error) {
	principal, err := s.principal(ctx)
	if err != nil {
		closeCustomViewBody(body)
		return nil, nil, err
	}
	input, err := publicationInput(ctx, body)
	if err != nil {
		return nil, nil, s.parseError(err)
	}
	if input.Kind != "" {
		return nil, nil, s.err(400, map[string]any{"error": "Expected an object."})
	}
	views := any(map[string]any{"list": true, "kanban": true, "calendar": true, "gantt": true, "spreadsheet": true})
	if v, ok := input.Fields["views"]; ok {
		views = v
	}
	comments, e := publicationBool(input, "is_comments_enabled", false)
	if e != "" {
		return nil, nil, s.fieldError("is_comments_enabled", e)
	}
	reactions, e := publicationBool(input, "is_reactions_enabled", false)
	if e != "" {
		return nil, nil, s.fieldError("is_reactions_enabled", e)
	}
	votes, e := publicationBool(input, "is_votes_enabled", false)
	if e != "" {
		return nil, nil, s.fieldError("is_votes_enabled", e)
	}
	data, err := s.repo.Create(ctx, principal.userID, p.Slug, p.ProjectID, projectpublication.CreateInput{Comments: comments, Reactions: reactions, Votes: votes, Intake: input.Fields["intake"], Views: views})
	if errors.Is(err, projectpublication.ErrNotFound) {
		return nil, nil, s.err(404, map[string]any{"error": "The required object does not exist."})
	}
	if err != nil {
		return nil, nil, s.mapError(err)
	}
	return &api.ProjectPublicationRawJSON{ContentType: "application/json"}, io.NopCloser(strings.NewReader(string(data))), nil
}
func (s *projectPublicationService) Update(ctx context.Context, p *api.UpdatePayload, body io.ReadCloser) (*api.ProjectPublicationRawJSON, io.ReadCloser, error) {
	principal, err := s.principal(ctx)
	if err != nil {
		closeCustomViewBody(body)
		return nil, nil, err
	}
	if _, e := uuid.Parse(p.Pk); e != nil {
		closeCustomViewBody(body)
		return nil, nil, s.notFound()
	}
	if err := s.repo.Exists(ctx, p.Slug, p.ProjectID, p.Pk); err != nil {
		closeCustomViewBody(body)
		return nil, nil, s.mapError(err)
	}
	input, err := publicationInput(ctx, body)
	if err != nil {
		return nil, nil, s.parseError(err)
	}
	if input.Kind != "" {
		return nil, nil, s.err(400, map[string]any{"error": "Expected an object."})
	}
	fields := map[string]any{}
	if value, ok := input.Fields["view_props"]; ok {
		if value == nil {
			return nil, nil, s.fieldError("view_props", "This field may not be null.")
		}
		fields["view_props"] = value
	}
	for _, name := range []string{"intake", "entity_identifier"} {
		if value, ok := input.Fields[name]; ok {
			if value == nil {
				fields[name] = nil
				continue
			}
			parsed, err := djangoinput.UUID(value)
			if err != nil {
				message := "Must be a valid UUID."
				if name == "intake" {
					message = "Invalid pk \"" + fmt.Sprint(value) + "\" - object does not exist."
				}
				return nil, nil, s.fieldError(name, message)
			}
			fields[name] = parsed
		}
	}
	if value, ok := input.Fields["entity_name"]; ok {
		if value == nil {
			fields["entity_name"] = nil
		} else {
			text, fieldErrors := djangoinput.TextField(value, true, 30, input.Surrogates["entity_name"])
			if len(fieldErrors) > 0 {
				return nil, nil, s.fieldError("entity_name", fieldErrors[0])
			}
			fields["entity_name"] = text
		}
	}
	for _, name := range []string{"is_activity_enabled", "is_comments_enabled", "is_reactions_enabled", "is_votes_enabled", "is_disabled"} {
		if value, ok := input.Fields[name]; ok {
			parsed, e := djangoinput.BooleanField(value)
			if e != "" {
				return nil, nil, s.fieldError(name, e)
			}
			fields[name] = parsed
		}
	}
	data, err := s.repo.Update(ctx, principal.userID, p.Slug, p.ProjectID, p.Pk, projectpublication.Patch{Fields: fields})
	if err != nil {
		return nil, nil, s.mapError(err)
	}
	return &api.ProjectPublicationRawJSON{ContentType: "application/json"}, io.NopCloser(strings.NewReader(string(data))), nil
}
func (s *projectPublicationService) Delete(ctx context.Context, p *api.DeletePayload) error {
	principal, err := s.principal(ctx)
	if err != nil {
		return err
	}
	if _, e := uuid.Parse(p.Pk); e != nil {
		return s.notFound()
	}
	if err := s.repo.Delete(ctx, principal.userID, p.Slug, p.ProjectID, p.Pk); err != nil {
		return s.mapError(err)
	}
	return nil
}
func publicationBool(input djangoinput.Input, key string, def bool) (bool, string) {
	value, ok := input.Fields[key]
	if !ok {
		return def, ""
	}
	return djangoinput.BooleanField(value)
}
func (s *projectPublicationService) parseError(err error) error {
	var f *djangoapi.HTTPError
	if errors.As(err, &f) {
		return s.err(f.Status, f.Body)
	}
	return s.internal()
}
func (s *projectPublicationService) fieldError(key, message string) error {
	return s.err(400, map[string]any{key: []string{message}})
}
func (s *projectPublicationService) notFound() error {
	return s.err(404, map[string]any{"detail": "No DeployBoard matches the given query."})
}
func (s *projectPublicationService) mapError(err error) error {
	if errors.Is(err, projectpublication.ErrInvalidIntake) {
		return s.fieldError("intake", "Invalid intake for this project.")
	}
	if errors.Is(err, projectpublication.ErrNotFound) {
		return s.notFound()
	}
	if errors.Is(err, projectpublication.ErrForbidden) {
		return s.err(403, map[string]any{"detail": "You do not have permission to perform this action."})
	}
	return s.internal()
}
func (s *projectPublicationService) internal() error {
	return s.err(500, map[string]any{"error": "Something went wrong please try again later"})
}

var _ api.Service = (*projectPublicationService)(nil)
