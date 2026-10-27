package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	store "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	api "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/project_membership_lifecycle"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
)

type projectMembershipLifecycleRepository interface {
	AuthorizeLifecycle(context.Context, store.LifecycleCommand) error
	ExecuteLifecycle(context.Context, store.LifecycleCommand) (any, error)
}
type projectMembershipLifecyclePrincipal struct {
	UserID, Timezone, Key string
	APIKey                bool
}
type projectMembershipLifecyclePrincipalKey struct{}
type projectMembershipLifecycleService struct {
	repository projectMembershipLifecycleRepository
	members    *store.MemberReader
	keys       activityReadsAPIKeys
	throttle   activityReadsThrottle
	sessions   projectLabelReadsSessions
}

func NewProjectMembershipLifecycle(repository projectMembershipLifecycleRepository, keys activityReadsAPIKeys, throttle activityReadsThrottle, sessions projectLabelReadsSessions) (api.Service, error) {
	members, err := store.NewMemberReader(repository)
	if err != nil || keys == nil || throttle == nil || sessions == nil {
		return nil, errors.New("membership repository, API-key validator, throttle and sessions are required")
	}
	return &projectMembershipLifecycleService{repository: repository, members: members, keys: keys, throttle: throttle, sessions: sessions}, nil
}
func (s *projectMembershipLifecycleService) APIKeyAuth(ctx context.Context, key string, scheme *security.APIKeyScheme) (context.Context, error) {
	if scheme != nil && scheme.Name == "PlaneSessionCookie" {
		user, zone, inspected := readServiceSessionPrincipal(ctx)
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
				user, zone = principal.UserID, principal.UserTimezone
			}
		}
		if user == "" {
			return ctx, projectMembershipLifecycleError(401, map[string]any{"detail": unauthenticatedDetail})
		}
		return context.WithValue(ctx, projectMembershipLifecyclePrincipalKey{}, projectMembershipLifecyclePrincipal{UserID: user, Timezone: zone}), nil
	}
	principal, err := s.keys.Validate(ctx, key)
	if err != nil {
		if key == "" {
			return ctx, projectMembershipLifecycleError(401, map[string]any{"detail": unauthenticatedDetail})
		}
		if errors.Is(err, apikey.ErrInvalid) {
			return ctx, projectMembershipLifecycleError(403, map[string]any{"detail": "Given API token is not valid"})
		}
		return ctx, projectMembershipLifecycleError(500, map[string]any{"detail": "Something went wrong please try again later"})
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Authenticated = true
	}
	return context.WithValue(ctx, projectMembershipLifecyclePrincipalKey{}, projectMembershipLifecyclePrincipal{UserID: principal.UserID.String(), Timezone: principal.Timezone, Key: key, APIKey: true}), nil
}
func (s *projectMembershipLifecycleService) execute(ctx context.Context, method, operation string, payload any, body io.ReadCloser) (any, error) {
	if body != nil {
		defer body.Close()
	}
	principal, ok := ctx.Value(projectMembershipLifecyclePrincipalKey{}).(projectMembershipLifecyclePrincipal)
	if !ok {
		return nil, projectMembershipLifecycleError(401, map[string]any{"detail": unauthenticatedDetail})
	}
	request := activityPayloadRequest(ctx, operation, payload)
	command := store.LifecycleCommand{Operation: operation, Method: method, UserID: principal.UserID, Slug: request.Params["slug"], ProjectID: request.Params["project_id"], MemberID: request.Params["pk"], Params: request.Params, Present: request.Present, Body: body, Timezone: principal.Timezone}
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		command.ContentType = metadata.Headers().Get("Content-Type")
	}
	if err := s.repository.AuthorizeLifecycle(ctx, command); err != nil {
		return nil, projectMembershipLifecycleMapError(err)
	}
	if principal.APIKey {
		rate, err := s.throttle.Allow(ctx, principal.Key)
		if err != nil {
			return nil, projectMembershipLifecycleMapError(err)
		}
		if effects := apikey.EffectsFromContext(ctx); effects != nil {
			effects.Rate = &rate
		}
		if !rate.Allowed {
			return nil, projectMembershipLifecycleError(429, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))})
		}
	}
	if err := djangosession.ValidateUserTimezone(principal.Timezone); err != nil {
		if errors.Is(err, djangosession.ErrTimezoneNotFound) {
			return nil, projectMembershipLifecycleError(400, map[string]any{"error": "The required key does not exist."})
		}
		return nil, projectMembershipLifecycleMapError(err)
	}
	result, err := s.executeAuthorized(ctx, command)
	if err != nil {
		if errors.Is(err, store.ErrLifecycleRendering) {
			MarkReadServiceDjangoHTML500(ctx)
		}
		return nil, projectMembershipLifecycleMapError(err)
	}
	return activityTimestamps(result, principal.Timezone), nil
}

func (s *projectMembershipLifecycleService) executeAuthorized(ctx context.Context, command store.LifecycleCommand) (any, error) {
	if command.Operation != "v1-list-project-members" || command.Method != http.MethodGet {
		return s.repository.ExecuteLifecycle(ctx, command)
	}
	actor, err := uuid.Parse(command.UserID)
	if err != nil {
		return nil, err
	}
	project, err := uuid.Parse(command.ProjectID)
	if err != nil {
		return nil, err
	}
	return s.members.ListProjectMembers(ctx, actor, command.Slug, project)
}
func (s *projectMembershipLifecycleService) raw(ctx context.Context, method, operation string, payload any, body io.ReadCloser) (*api.MembershipLifecycleRawJSONResponseMetadata, io.ReadCloser, error) {
	value, err := s.execute(ctx, method, operation, payload, body)
	if err != nil {
		return nil, nil, err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, nil, projectMembershipLifecycleMapError(err)
	}
	return &api.MembershipLifecycleRawJSONResponseMetadata{ContentType: "application/json"}, io.NopCloser(bytes.NewReader(data)), nil
}
func projectMembershipLifecycleError(status int, body map[string]any) error {
	result := &api.MembershipLifecycleErrorResponseBody{Name: fmt.Sprintf("status_%d", status)}
	data, _ := json.Marshal(body)
	_ = json.Unmarshal(data, result)
	return result
}
func projectMembershipLifecycleMapError(err error) error {
	var response *djangoapi.HTTPError
	if errors.As(err, &response) {
		return projectMembershipLifecycleError(response.Status, response.Body)
	}
	return projectMembershipLifecycleError(500, map[string]any{"error": "Something went wrong please try again later"})
}

func (s *projectMembershipLifecycleService) GetAPIV1WorkspacesBySlugProjectsByProjectIDMembers(ctx context.Context, p *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDMembersPayload) (*api.MembershipLifecycleRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.raw(ctx, http.MethodGet, "v1-list-members", p, nil)
}

func (s *projectMembershipLifecycleService) PostAPIV1WorkspacesBySlugProjectsByProjectIDMembers(ctx context.Context, p *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDMembersPayload, b io.ReadCloser) (*api.MembershipLifecycleRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.raw(ctx, http.MethodPost, "v1-create-members", p, b)
}

func (s *projectMembershipLifecycleService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDMembersByPk(ctx context.Context, p *api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDMembersByPkPayload) error {
	_, err := s.execute(ctx, http.MethodDelete, "v1-delete-members", p, nil)
	return err
}

func (s *projectMembershipLifecycleService) GetAPIV1WorkspacesBySlugProjectsByProjectIDMembersByPk(ctx context.Context, p *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDMembersByPkPayload) (*api.MembershipLifecycleRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.raw(ctx, http.MethodGet, "v1-detail-members", p, nil)
}

func (s *projectMembershipLifecycleService) PatchAPIV1WorkspacesBySlugProjectsByProjectIDMembersByPk(ctx context.Context, p *api.PatchAPIV1WorkspacesBySlugProjectsByProjectIDMembersByPkPayload, b io.ReadCloser) (*api.MembershipLifecycleRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.raw(ctx, http.MethodPatch, "v1-patch-members", p, b)
}

func (s *projectMembershipLifecycleService) GetAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembersLite(ctx context.Context, p *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembersLitePayload) (*api.MembershipLifecycleRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.raw(ctx, http.MethodGet, "v1-list-members-lite", p, nil)
}

func (s *projectMembershipLifecycleService) GetAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembers(ctx context.Context, p *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembersPayload) (*api.MembershipLifecycleRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.raw(ctx, http.MethodGet, "v1-list-project-members", p, nil)
}

func (s *projectMembershipLifecycleService) PostAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembers(ctx context.Context, p *api.PostAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembersPayload, b io.ReadCloser) (*api.MembershipLifecycleRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.raw(ctx, http.MethodPost, "v1-create-project-members", p, b)
}

func (s *projectMembershipLifecycleService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembersByPk(ctx context.Context, p *api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembersByPkPayload) error {
	_, err := s.execute(ctx, http.MethodDelete, "v1-delete-project-members", p, nil)
	return err
}

func (s *projectMembershipLifecycleService) GetAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembersByPk(ctx context.Context, p *api.GetAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembersByPkPayload) (*api.MembershipLifecycleRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.raw(ctx, http.MethodGet, "v1-detail-project-members", p, nil)
}

func (s *projectMembershipLifecycleService) PatchAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembersByPk(ctx context.Context, p *api.PatchAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembersByPkPayload, b io.ReadCloser) (*api.MembershipLifecycleRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.raw(ctx, http.MethodPatch, "v1-patch-project-members", p, b)
}

func (s *projectMembershipLifecycleService) PostAPIWorkspacesBySlugProjectsByProjectIDMembers(ctx context.Context, p *api.PostAPIWorkspacesBySlugProjectsByProjectIDMembersPayload, b io.ReadCloser) (*api.MembershipLifecycleRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.raw(ctx, http.MethodPost, "legacy-create-members", p, b)
}

func (s *projectMembershipLifecycleService) PostAPIWorkspacesBySlugProjectsByProjectIDMembersLeave(ctx context.Context, p *api.PostAPIWorkspacesBySlugProjectsByProjectIDMembersLeavePayload, b io.ReadCloser) error {
	_, err := s.execute(ctx, http.MethodPost, "legacy-leave-members", p, b)
	return err
}

func (s *projectMembershipLifecycleService) DeleteAPIWorkspacesBySlugProjectsByProjectIDMembersByPk(ctx context.Context, p *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDMembersByPkPayload) error {
	_, err := s.execute(ctx, http.MethodDelete, "legacy-delete-members", p, nil)
	return err
}

func (s *projectMembershipLifecycleService) PatchAPIWorkspacesBySlugProjectsByProjectIDMembersByPk(ctx context.Context, p *api.PatchAPIWorkspacesBySlugProjectsByProjectIDMembersByPkPayload, b io.ReadCloser) (*api.MembershipLifecycleRawJSONResponseMetadata, io.ReadCloser, error) {
	return s.raw(ctx, http.MethodPatch, "legacy-patch-members", p, b)
}

var _ api.Service = (*projectMembershipLifecycleService)(nil)

type MembershipLifecycleProtocolAuthorizer interface {
	AuthorizeMembershipLifecycleProtocol(context.Context, string, SessionProtocolResource, bool) (int, map[string]any)
}

func (s *projectMembershipLifecycleService) AuthorizeMembershipLifecycleProtocol(ctx context.Context, key string, resource SessionProtocolResource, apiKey bool) (int, map[string]any) {
	scheme := &security.APIKeyScheme{Name: "PlaneSessionCookie"}
	if apiKey {
		scheme.Name = "ApiKeyAuth"
	}
	ctx, err := s.APIKeyAuth(ctx, key, scheme)
	if err == nil {
		principal, _ := ctx.Value(projectMembershipLifecyclePrincipalKey{}).(projectMembershipLifecyclePrincipal)
		if apiKey {
			operation := "v1-protocol"
			if strings.Contains(resource.Pattern, "project-members-lite") {
				operation = "v1-list-members-lite"
			}
			command := store.LifecycleCommand{Operation: operation, Method: resource.Method, UserID: principal.UserID, Slug: resource.Slug, ProjectID: resource.ProjectID}
			err = s.repository.AuthorizeLifecycle(ctx, command)
			if err == nil {
				var rate apikey.Rate
				rate, err = s.throttle.Allow(ctx, principal.Key)
				if effects := apikey.EffectsFromContext(ctx); effects != nil {
					effects.Rate = &rate
				}
				if err == nil && !rate.Allowed {
					err = projectMembershipLifecycleError(429, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))})
				}
			}
		}
		if err == nil {
			if zoneErr := djangosession.ValidateUserTimezone(principal.Timezone); zoneErr != nil {
				if errors.Is(zoneErr, djangosession.ErrTimezoneNotFound) {
					err = projectMembershipLifecycleError(400, map[string]any{"error": "The required key does not exist."})
				} else {
					err = zoneErr
				}
			}
		}
	}
	if err == nil {
		return 0, nil
	}
	var generated *api.MembershipLifecycleErrorResponseBody
	if !errors.As(err, &generated) {
		err = projectMembershipLifecycleMapError(err)
		_ = errors.As(err, &generated)
	}
	data, _ := json.Marshal(generated)
	var body map[string]any
	_ = json.Unmarshal(data, &body)
	var status int
	_, _ = fmt.Sscanf(generated.Name, "status_%d", &status)
	return status, body
}
