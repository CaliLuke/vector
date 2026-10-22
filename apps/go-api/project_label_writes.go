package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/projectlabelreads"
	api "github.com/CaliLuke/go-plane/project_label_writes/gen/project_label_writes"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type labelWritesRepository interface {
	Authorize(context.Context, string, string, string, string) error
	Load(context.Context, string, string, string, string) (*store.Label, error)
	Write(context.Context, string, string, string, string, string, io.Reader, string) (*store.Label, error)
}
type labelWritesCache interface {
	Invalidate(context.Context, string, bool) error
}
type projectLabelWritesService struct {
	repository labelWritesRepository
	sessions   projectLabelReadsSessions
	cache      labelWritesCache
}
type labelWritesPrincipalKey struct{}

func NewProjectLabelWrites(repository labelWritesRepository, sessions projectLabelReadsSessions, cache labelWritesCache) (api.Service, error) {
	if repository == nil || sessions == nil || cache == nil {
		return nil, errors.New("label write repository, sessions and cache are required")
	}
	return &projectLabelWritesService{repository: repository, sessions: sessions, cache: cache}, nil
}

func labelWriteError(err error) error {
	if err == nil {
		return nil
	}
	var generated *api.LabelError
	if errors.As(err, &generated) {
		return err
	}
	failure := &store.HTTPError{Status: 500, Body: map[string]any{"error": "Something went wrong please try again later"}}
	var domain *store.HTTPError
	if errors.As(err, &domain) {
		failure = domain
	}
	data, _ := json.Marshal(failure.Body)
	result := &api.LabelError{Kind: fmt.Sprintf("status_%d", failure.Status)}
	if err := json.Unmarshal(data, result); err != nil {
		panic(err)
	}
	return result
}

func (s *projectLabelWritesService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	id, zone, inspected := readServiceSessionPrincipal(ctx)
	if !inspected {
		if key == "" {
			if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
				request := &http.Request{Header: metadata.Headers()}
				if cookie, err := request.Cookie("session-id"); err == nil {
					key = cookie.Value
				}
			}
		}
		principal, err := s.sessions.Validate(ctx, key)
		if err == nil {
			id, zone = principal.UserID, principal.UserTimezone
		}
	}
	if id == "" {
		return ctx, labelWriteError(store.Failure(401, "detail", unauthenticatedDetail))
	}
	return context.WithValue(ctx, labelWritesPrincipalKey{}, djangosession.Principal{UserID: id, UserTimezone: zone}), nil
}

func (s *projectLabelWritesService) principal(ctx context.Context, slug, project, method string) (string, error) {
	p, _ := ctx.Value(labelWritesPrincipalKey{}).(djangosession.Principal)
	if p.UserID == "" {
		return "", store.Failure(401, "detail", unauthenticatedDetail)
	}
	// ProjectBasePermission runs inside initial(), before TimezoneMixin dispatch.
	if err := s.repository.Authorize(ctx, p.UserID, slug, project, method); err != nil {
		return "", err
	}
	if err := djangosession.ValidateUserTimezone(p.UserTimezone); err != nil {
		if errors.Is(err, djangosession.ErrTimezoneNotFound) {
			return "", store.Failure(400, "error", "The required key does not exist.")
		}
		return "", err
	}
	return p.UserID, nil
}

func (s *projectLabelWritesService) write(ctx context.Context, slug, project, id, method string, body io.ReadCloser) (*api.Label, error) {
	if body != nil {
		defer body.Close()
	}
	user, err := s.principal(ctx, slug, project, method)
	if err != nil {
		return nil, labelWriteError(err)
	}
	if method != http.MethodPut {
		if err = s.cache.Invalidate(ctx, slug, method == http.MethodPost); err != nil {
			return nil, labelWriteError(err)
		}
		if err = s.repository.Authorize(ctx, user, slug, project, http.MethodDelete); err != nil {
			var denied *store.HTTPError
			if errors.As(err, &denied) && denied.Status == 403 {
				err = store.Failure(403, "error", "You don't have the required permissions.")
			}
			return nil, labelWriteError(err)
		}
	}
	contentType := ""
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		contentType = metadata.Headers().Get("Content-Type")
	}
	label, err := s.repository.Write(ctx, user, slug, project, id, method, body, contentType)
	if err != nil {
		return nil, labelWriteError(err)
	}
	if label == nil {
		return nil, nil
	}
	if math.IsNaN(label.SortOrder) || math.IsInf(label.SortOrder, 0) {
		MarkReadServiceDjangoHTML500(ctx)
		return nil, labelWriteError(errors.New("Django cannot render a nonfinite label sort_order"))
	}
	parent, projectValue := loom.NullValue[string](), loom.NullValue[string]()
	if label.Parent != nil {
		parent = loom.NullableValue(*label.Parent)
	}
	if label.ProjectID != nil {
		projectValue = loom.NullableValue(*label.ProjectID)
	}
	return &api.Label{ID: label.ID, Name: label.Name, Color: label.Color, Parent: parent, ProjectID: projectValue, WorkspaceID: label.WorkspaceID, SortOrder: label.SortOrder}, nil
}

func (s *projectLabelWritesService) Create(ctx context.Context, p *api.CreatePayload, body io.ReadCloser) (*api.Label, error) {
	return s.write(ctx, p.Slug, p.ProjectID, "", http.MethodPost, body)
}
func (s *projectLabelWritesService) Patch(ctx context.Context, p *api.PatchPayload, body io.ReadCloser) (*api.Label, error) {
	return s.write(ctx, p.Slug, p.ProjectID, p.Pk, http.MethodPatch, body)
}
func (s *projectLabelWritesService) Update(ctx context.Context, p *api.UpdatePayload, body io.ReadCloser) (*api.Label, error) {
	return s.write(ctx, p.Slug, p.ProjectID, p.Pk, http.MethodPut, body)
}
func (s *projectLabelWritesService) Delete(ctx context.Context, p *api.DeletePayload) error {
	_, err := s.write(ctx, p.Slug, p.ProjectID, p.Pk, http.MethodDelete, nil)
	return err
}

func (s *projectLabelWritesService) AuthorizeSessionProtocol(ctx context.Context, key string) (int, map[string]any) {
	_, err := s.APIKeyAuth(ctx, key, nil)
	if err != nil {
		return 401, map[string]any{"detail": unauthenticatedDetail}
	}
	return 0, nil
}
func (s *projectLabelWritesService) AuthorizeSessionResourceProtocol(ctx context.Context, key string, resource SessionProtocolResource) (int, map[string]any) {
	ctx, err := s.APIKeyAuth(ctx, key, nil)
	if err == nil {
		_, err = s.principal(ctx, resource.Slug, resource.ProjectID, resource.Method)
	}
	if err == nil {
		return 0, nil
	}
	var failure *api.LabelError
	if errors.As(labelWriteError(err), &failure) {
		data, _ := json.Marshal(failure)
		var body map[string]any
		_ = json.Unmarshal(data, &body)
		var status int
		_, _ = fmt.Sscanf(failure.Kind, "status_%d", &status)
		return status, body
	}
	panic("unmapped label protocol error")
}

func (s *projectLabelWritesService) metadata(ctx context.Context, slug, project, id, action string) (*api.LabelMetadata, error) {
	user, err := s.principal(ctx, slug, project, http.MethodOptions)
	if err != nil {
		return nil, labelWriteError(err)
	}
	result := &api.LabelMetadata{Name: "Label", Description: "", Renders: []string{"application/json"}, Parses: []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}}
	// DRF metadata clones the request and checks only the action's permission
	// classes, plus target existence for PUT. It never invokes the action decorator.
	if err = s.repository.Authorize(ctx, user, slug, project, action); err != nil {
		var failure *store.HTTPError
		if errors.As(err, &failure) && failure.Status == 403 {
			return result, nil
		}
		return nil, labelWriteError(err)
	}
	if action == http.MethodPut {
		if _, err = s.repository.Load(ctx, user, slug, project, id); err != nil {
			var failure *store.HTTPError
			if errors.As(err, &failure) && failure.Status == 404 {
				return result, nil
			}
			return nil, labelWriteError(err)
		}
	}
	maxLength := 255
	result.Actions = map[string]map[string]*api.LabelMetadataField{action: {
		"parent":       {Type: "field", Label: "Parent"},
		"name":         {Type: "string", Label: "Name", Required: true, MaxLength: &maxLength},
		"color":        {Type: "string", Label: "Color", MaxLength: &maxLength},
		"id":           {Type: "field", Label: "Id", ReadOnly: true},
		"project_id":   {Type: "field", Label: "Project id", ReadOnly: true},
		"workspace_id": {Type: "field", Label: "Workspace id", ReadOnly: true},
		"sort_order":   {Type: "float", Label: "Sort order"},
	}}
	return result, nil
}
func (s *projectLabelWritesService) CollectionOptions(ctx context.Context, p *api.CollectionOptionsPayload) (*api.LabelMetadata, error) {
	return s.metadata(ctx, p.Slug, p.ProjectID, "", http.MethodPost)
}
func (s *projectLabelWritesService) DetailOptions(ctx context.Context, p *api.DetailOptionsPayload) (*api.LabelMetadata, error) {
	return s.metadata(ctx, p.Slug, p.ProjectID, p.Pk, http.MethodPut)
}
