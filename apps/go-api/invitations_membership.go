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
	stduuid "uuid"

	"github.com/CaliLuke/go-plane/internal/platform"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	store "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	api "github.com/CaliLuke/go-plane/invitationsmembership/gen/invitations_membership"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type invitationsMembershipRepository interface {
	AuthorizeWorkspaceRole(context.Context, uuid.UUID, string, ...int) error
	WorkspaceExists(context.Context, string) (bool, error)
	WorkspaceInvites(context.Context, string) (json.RawMessage, error)
	WorkspaceInvite(context.Context, string, uuid.UUID) (json.RawMessage, error)
	WorkspaceMemberUsers(context.Context, string) (json.RawMessage, error)
	ListWorkspaceMembers(context.Context, stduuid.UUID, string, store.WorkspaceMembersQuery) (store.WorkspaceMembersPage, error)
	ListRoles(context.Context, stduuid.UUID, string, string, int, int) ([]store.RoleDefinition, int, error)
	RetrieveRole(context.Context, stduuid.UUID, string, stduuid.UUID) (store.RoleDefinition, error)
	CreateAPIWorkspaceInvite(context.Context, string, uuid.UUID, io.Reader, string) (json.RawMessage, error)
	PatchAPIWorkspaceInvite(context.Context, string, uuid.UUID, io.Reader, string) (json.RawMessage, error)
	DeleteAPIWorkspaceInvite(context.Context, string, uuid.UUID) error
}

type invitationsMembershipService struct {
	repository invitationsMembershipRepository
	keys       activityReadsAPIKeys
	sessions   activityReadsSessions
	throttle   activityReadsThrottle
	cache      store.InvitationCacheInvalidator
}

type invitationsMembershipPrincipalKey struct{}

type invitationsMembershipPrincipal struct {
	UserID   uuid.UUID
	Timezone string
	Key      string
	APIKey   bool
}

func newInvitationsMembershipService(repository invitationsMembershipRepository, keys activityReadsAPIKeys, sessions activityReadsSessions, throttle activityReadsThrottle) (*invitationsMembershipService, error) {
	if repository == nil || keys == nil || sessions == nil || throttle == nil {
		return nil, errors.New("invitations membership repository, API-key validator, sessions, and throttle are required")
	}
	return &invitationsMembershipService{repository: repository, keys: keys, sessions: sessions, throttle: throttle}, nil
}

func NewInvitationsMembership(repository invitationsMembershipRepository, keys activityReadsAPIKeys, sessions activityReadsSessions, throttle activityReadsThrottle, cache store.InvitationCacheInvalidator) (api.Service, error) {
	service, err := newInvitationsMembershipService(repository, keys, sessions, throttle)
	if err != nil {
		return nil, err
	}
	if cache == nil {
		return nil, errors.New("invitation cache invalidator is required")
	}
	service.cache = cache
	return service, nil
}

type InvitationsMembershipProtocolAuthorizer interface {
	AuthorizeInvitationsMembershipProtocol(context.Context, string, SessionProtocolResource, bool) (int, map[string]any)
}

func (service *invitationsMembershipService) AuthorizeInvitationsMembershipProtocol(ctx context.Context, credential string, resource SessionProtocolResource, apiKey bool) (int, map[string]any) {
	// Public join lookups deliberately allow anonymous requests. Their POST
	// handlers check the invitation token before authentication, as Django does.
	if strings.Contains(resource.Pattern, "/join") {
		userID, timezone, inspected := readServiceSessionPrincipal(ctx)
		if !inspected && credential != "" {
			principal, sessionErr := service.sessions.Validate(ctx, credential)
			if sessionErr == nil {
				userID, timezone, inspected = principal.UserID, principal.UserTimezone, true
			}
		}
		if !inspected || userID == "" {
			return 0, nil
		}
		if zoneErr := djangosession.ValidateUserTimezone(timezone); zoneErr != nil {
			return 400, map[string]any{"error": "The required key does not exist."}
		}
		return 0, nil
	}
	scheme := &security.APIKeyScheme{Name: "PlaneSessionCookie"}
	if apiKey {
		scheme.Name = "ApiKeyAuth"
	}
	requestContext, err := service.APIKeyAuth(ctx, credential, scheme)
	if err == nil {
		principal, _ := requestContext.Value(invitationsMembershipPrincipalKey{}).(invitationsMembershipPrincipal)
		if apiKey {
			var roles []int
			throttle := false
			switch {
			case strings.Contains(resource.Pattern, "/members-lite"), strings.HasSuffix(strings.TrimSuffix(resource.Pattern, "/"), "/members"):
				roles = []int{15, 20}
				throttle = true
			case strings.Contains(resource.Pattern, "/roles/"):
				roles = []int{5, 15, 20}
				throttle = true
			case strings.Contains(resource.Pattern, "/invitations"):
				roles = []int{20}
			}
			if throttle {
				requestContext = store.WithInvitationReplica(requestContext)
			}
			if len(roles) != 0 {
				err = service.repository.AuthorizeWorkspaceRole(requestContext, principal.UserID, resource.Slug, roles...)
			}
			if err == nil && throttle {
				var rate apikey.Rate
				rate, err = service.throttle.Allow(requestContext, principal.Key)
				if effects := apikey.EffectsFromContext(requestContext); effects != nil {
					effects.Rate = &rate
				}
				if err == nil && !rate.Allowed {
					err = invitationsMembershipError(429, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait.Seconds()))})
				}
			}
		}

		if err == nil && !apiKey && strings.Contains(resource.Pattern, "/workspaces/") && strings.Contains(resource.Pattern, "/invitations") && !strings.Contains(resource.Pattern, "/projects/") && !strings.HasPrefix(resource.Pattern, "/api/users/") {
			err = service.repository.AuthorizeWorkspaceRole(requestContext, principal.UserID, resource.Slug, 15, 20)
		}
		if err == nil {
			if zoneErr := djangosession.ValidateUserTimezone(principal.Timezone); zoneErr != nil {
				if errors.Is(zoneErr, djangosession.ErrTimezoneNotFound) {
					err = invitationsMembershipError(400, map[string]any{"error": "The required key does not exist."})
				} else {
					err = zoneErr
				}
			}
		}
		// DRF metadata discovers PUT fields through get_object, after initial
		// authentication, permissions, and timezone activation.
		if err == nil && apiKey && resource.Method == http.MethodOptions && resource.ID != "" && strings.Contains(resource.Pattern, "/invitations/") {
			var id uuid.UUID
			id, err = uuid.Parse(resource.ID)
			if err != nil {
				err = invitationsMembershipError(400, map[string]any{"error": "Please provide valid detail"})
			} else {
				_, err = service.repository.WorkspaceInvite(requestContext, resource.Slug, id)
			}
		}
	}
	if err == nil {
		return 0, nil
	}
	var generated *api.InvitationsMembershipAPIKeyError
	if !errors.As(err, &generated) {
		err = invitationsMembershipRepositoryError(err)
		_ = errors.As(err, &generated)
	}
	data, _ := json.Marshal(generated)
	var body map[string]any
	_ = json.Unmarshal(data, &body)
	var status int
	if generated != nil {
		_, _ = fmt.Sscanf(generated.Name, "status_%d", &status)
	}
	return status, body
}

func (service *invitationsMembershipService) APIKeyAuth(ctx context.Context, key string, scheme *security.APIKeyScheme) (context.Context, error) {
	if scheme == nil {
		return ctx, invitationsMembershipError(401, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	switch scheme.Name {
	case "ApiKeyAuth":
		principal, err := service.keys.Validate(ctx, key)
		if err != nil {
			if key == "" {
				return ctx, invitationsMembershipError(401, map[string]any{"detail": "Authentication credentials were not provided."})
			}
			if errors.Is(err, apikey.ErrInvalid) {
				return ctx, invitationsMembershipError(403, map[string]any{"detail": "Given API token is not valid"})
			}
			return ctx, invitationsMembershipError(500, map[string]any{"detail": "Something went wrong please try again later"})
		}
		if effects := apikey.EffectsFromContext(ctx); effects != nil {
			effects.Authenticated = true
		}
		return context.WithValue(ctx, invitationsMembershipPrincipalKey{}, invitationsMembershipPrincipal{
			UserID: principal.UserID, Timezone: principal.Timezone, Key: key, APIKey: true,
		}), nil
	case "PlaneSessionCookie", "DjangoSessionCookie":
		if userID, timezone, inspected := readServiceSessionPrincipal(ctx); inspected {
			parsed, err := uuid.Parse(userID)
			if err != nil {
				return ctx, invitationsMembershipError(401, map[string]any{"detail": "Authentication credentials were not provided."})
			}
			return context.WithValue(ctx, invitationsMembershipPrincipalKey{}, invitationsMembershipPrincipal{UserID: parsed, Timezone: timezone}), nil
		}
		if key == "" {
			if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
				request := &http.Request{Header: metadata.Headers()}
				if cookie, err := request.Cookie("session-id"); err == nil {
					key = cookie.Value
				}
			}
		}
		principal, err := service.sessions.Validate(ctx, key)
		if err != nil {
			return ctx, invitationsMembershipError(401, map[string]any{"detail": "Authentication credentials were not provided."})
		}
		parsed, err := uuid.Parse(principal.UserID)
		if err != nil {
			return ctx, invitationsMembershipError(401, map[string]any{"detail": "Authentication credentials were not provided."})
		}
		return context.WithValue(ctx, invitationsMembershipPrincipalKey{}, invitationsMembershipPrincipal{UserID: parsed, Timezone: principal.UserTimezone}), nil
	default:
		return ctx, invitationsMembershipError(401, map[string]any{"detail": "Authentication credentials were not provided."})
	}
}

func invitationsMembershipError(status int, body map[string]any) error {
	response := &api.InvitationsMembershipAPIKeyError{Name: fmt.Sprintf("status_%d", status)}
	data, _ := json.Marshal(body)
	_ = json.Unmarshal(data, response)
	return response
}

func invitationsMembershipRepositoryError(err error) error {
	if err == nil {
		return nil
	}
	var httpError *djangoapi.HTTPError
	if errors.As(err, &httpError) {
		return invitationsMembershipError(httpError.Status, httpError.Body)
	}
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) {
		if strings.HasPrefix(databaseError.Code, "23") {
			return invitationsMembershipError(400, map[string]any{"error": "The payload is not valid"})
		}
		if databaseError.Code == "22P02" {
			return invitationsMembershipError(400, map[string]any{"error": "Please provide valid detail"})
		}
	}
	if errors.Is(err, store.ErrForbidden) {
		return invitationsMembershipError(403, map[string]any{"detail": "You do not have permission to perform this action."})
	}
	if errors.Is(err, store.ErrNotFound) {
		return invitationsMembershipError(404, map[string]any{"error": "The required object does not exist."})
	}
	return invitationsMembershipError(500, map[string]any{"error": "Something went wrong please try again later"})
}

func invitationContentType(ctx context.Context) string {
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		return metadata.Headers().Get("Content-Type")
	}
	return "application/json"
}

func (service *invitationsMembershipService) PostAPIV1WorkspacesBySlugInvitations(ctx context.Context, payload *api.PostAPIV1WorkspacesBySlugInvitationsPayload, body io.ReadCloser) (*api.PlaneAPISerializersInviteWorkspaceinviteserializerResponse, error) {
	principal, err := invitationsMembershipUser(ctx)
	if err != nil {
		return nil, err
	}
	if authorizeWorkspaceErr := service.authorizeWorkspace(ctx, principal, payload.Slug, false, 20); authorizeWorkspaceErr != nil {
		return nil, authorizeWorkspaceErr
	}
	if body != nil {
		defer platform.CloseRequestBody(ctx, body)
	}
	result, err := service.repository.CreateAPIWorkspaceInvite(ctx, payload.Slug, principal.UserID, body, invitationContentType(ctx))
	if err != nil {
		return nil, invitationsMembershipRepositoryError(err)
	}
	return decodeInvitationsMembershipResponse[*api.PlaneAPISerializersInviteWorkspaceinviteserializerResponse](ctx, result)
}

func (service *invitationsMembershipService) PatchAPIV1WorkspacesBySlugInvitationsByPk(ctx context.Context, payload *api.PatchAPIV1WorkspacesBySlugInvitationsByPkPayload, body io.ReadCloser) (*api.PlaneAPISerializersInviteWorkspaceinviteserializerResponse, error) {
	principal, err := invitationsMembershipUser(ctx)
	if err != nil {
		return nil, err
	}
	if authorizeWorkspaceErr := service.authorizeWorkspace(ctx, principal, payload.Slug, false, 20); authorizeWorkspaceErr != nil {
		return nil, authorizeWorkspaceErr
	}
	id, err := uuid.Parse(payload.Pk)
	if err != nil {
		return nil, invitationsMembershipError(400, map[string]any{"error": "Please provide valid detail"})
	}
	if body != nil {
		defer platform.CloseRequestBody(ctx, body)
	}
	result, err := service.repository.PatchAPIWorkspaceInvite(store.WithInvitationActor(ctx, principal.UserID.String()), payload.Slug, id, body, invitationContentType(ctx))
	if err != nil {
		return nil, invitationsMembershipRepositoryError(err)
	}
	return decodeInvitationsMembershipResponse[*api.PlaneAPISerializersInviteWorkspaceinviteserializerResponse](ctx, result)
}

func (service *invitationsMembershipService) DeleteAPIV1WorkspacesBySlugInvitationsByPk(ctx context.Context, payload *api.DeleteAPIV1WorkspacesBySlugInvitationsByPkPayload) error {
	principal, err := invitationsMembershipUser(ctx)
	if err != nil {
		return err
	}
	if authorizeWorkspaceErr := service.authorizeWorkspace(ctx, principal, payload.Slug, false, 20); authorizeWorkspaceErr != nil {
		return authorizeWorkspaceErr
	}
	id, err := uuid.Parse(payload.Pk)
	if err != nil {
		return invitationsMembershipError(400, map[string]any{"error": "Please provide valid detail"})
	}
	return invitationsMembershipRepositoryError(service.repository.DeleteAPIWorkspaceInvite(store.WithInvitationActor(ctx, principal.UserID.String()), payload.Slug, id))
}

func invitationsMembershipUser(ctx context.Context) (invitationsMembershipPrincipal, error) {
	principal, ok := ctx.Value(invitationsMembershipPrincipalKey{}).(invitationsMembershipPrincipal)
	if !ok {
		return invitationsMembershipPrincipal{}, invitationsMembershipError(401, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	return principal, nil
}

func (service *invitationsMembershipService) authorizeWorkspace(ctx context.Context, principal invitationsMembershipPrincipal, slug string, throttle bool, roles ...int) error {
	if throttle {
		ctx = store.WithInvitationReplica(ctx)
	}
	if err := service.repository.AuthorizeWorkspaceRole(ctx, principal.UserID, slug, roles...); err != nil {
		return invitationsMembershipRepositoryError(err)
	}
	if throttle && principal.APIKey {
		rate, err := service.throttle.Allow(ctx, principal.Key)
		if err != nil {
			return invitationsMembershipError(500, map[string]any{"detail": "Something went wrong please try again later"})
		}
		if effects := apikey.EffectsFromContext(ctx); effects != nil {
			effects.Rate = &rate
		}
		if !rate.Allowed {
			return invitationsMembershipError(429, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait.Seconds()))})
		}
	}
	if err := djangosession.ValidateUserTimezone(principal.Timezone); err != nil {
		if errors.Is(err, djangosession.ErrTimezoneNotFound) {
			return invitationsMembershipError(400, map[string]any{"error": "The required key does not exist."})
		}
		return invitationsMembershipError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	return nil
}

func decodeInvitationsMembershipResponse[T any](ctx context.Context, data json.RawMessage) (T, error) {
	if principal, ok := ctx.Value(invitationsMembershipPrincipalKey{}).(invitationsMembershipPrincipal); ok {
		data = normalizeInvitationResponse(data, principal.Timezone)
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return result, invitationsMembershipError(500, map[string]any{"detail": "Something went wrong please try again later"})
	}
	return result, nil
}

func (service *invitationsMembershipService) GetAPIV1WorkspacesBySlugInvitations(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugInvitationsPayload) ([]*api.PlaneAPISerializersInviteWorkspaceinviteserializerResponse, error) {
	principal, err := invitationsMembershipUser(ctx)
	if err != nil {
		return nil, err
	}
	if authorizeWorkspaceErr := service.authorizeWorkspace(ctx, principal, payload.Slug, false, 20); authorizeWorkspaceErr != nil {
		return nil, authorizeWorkspaceErr
	}
	data, err := service.repository.WorkspaceInvites(ctx, payload.Slug)
	if err != nil {
		return nil, invitationsMembershipRepositoryError(err)
	}
	return decodeInvitationsMembershipResponse[[]*api.PlaneAPISerializersInviteWorkspaceinviteserializerResponse](ctx, data)
}

func (service *invitationsMembershipService) GetAPIV1WorkspacesBySlugInvitationsByPk(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugInvitationsByPkPayload) (*api.PlaneAPISerializersInviteWorkspaceinviteserializerResponse, error) {
	principal, err := invitationsMembershipUser(ctx)
	if err != nil {
		return nil, err
	}
	if authorizeWorkspaceErr := service.authorizeWorkspace(ctx, principal, payload.Slug, false, 20); authorizeWorkspaceErr != nil {
		return nil, authorizeWorkspaceErr
	}
	id, err := uuid.Parse(payload.Pk)
	if err != nil {
		return nil, invitationsMembershipError(400, map[string]any{"error": "Please provide valid detail"})
	}
	data, err := service.repository.WorkspaceInvite(ctx, payload.Slug, id)
	if err != nil {
		return nil, invitationsMembershipRepositoryError(err)
	}
	return decodeInvitationsMembershipResponse[*api.PlaneAPISerializersInviteWorkspaceinviteserializerResponse](ctx, data)
}

func (service *invitationsMembershipService) GetAPIV1WorkspacesBySlugMembers(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugMembersPayload) ([]*api.PlaneAPISerializersMemberWorkspacememberuserliteserializerResponse, error) {
	principal, err := invitationsMembershipUser(ctx)
	if err != nil {
		return nil, err
	}
	if authorizeWorkspaceErr := service.authorizeWorkspace(ctx, principal, payload.Slug, true, 15, 20); authorizeWorkspaceErr != nil {
		return nil, authorizeWorkspaceErr
	}
	exists, existsErr := service.repository.WorkspaceExists(ctx, payload.Slug)
	if existsErr != nil {
		return nil, invitationsMembershipRepositoryError(existsErr)
	}
	if !exists {
		return nil, invitationsMembershipError(400, map[string]any{"error": "Provided workspace does not exist"})
	}
	data, err := service.repository.WorkspaceMemberUsers(ctx, payload.Slug)
	if err != nil {
		return nil, invitationsMembershipRepositoryError(err)
	}
	return decodeInvitationsMembershipResponse[[]*api.PlaneAPISerializersMemberWorkspacememberuserliteserializerResponse](ctx, data)
}

func (service *invitationsMembershipService) GetAPIV1WorkspacesBySlugMembersLite(ctx context.Context, payload *api.GetAPIV1WorkspacesBySlugMembersLitePayload) (*api.RestFrameworkSerializersPaginatedworkspacememberliteResponse, error) {
	principal, err := invitationsMembershipUser(ctx)
	if err != nil {
		return nil, err
	}
	if authorizeWorkspaceErr := service.authorizeWorkspace(ctx, principal, payload.Slug, true, 15, 20); authorizeWorkspaceErr != nil {
		return nil, authorizeWorkspaceErr
	}
	exists, err := service.repository.WorkspaceExists(ctx, payload.Slug)
	if err != nil {
		return nil, invitationsMembershipRepositoryError(err)
	}
	if !exists {
		return nil, invitationsMembershipError(404, map[string]any{"error": "Provided workspace does not exist"})
	}
	perPage := 1000
	if payload.PerPage != nil {
		perPage = *payload.PerPage
	}

	if original, present := ReadServiceQueryValues(ctx)["per_page"]; present && len(original) > 0 {
		parsed, parseErr := pythoncompat.DecimalInteger(original[len(original)-1])
		if parseErr != nil {
			return nil, invitationsMembershipError(400, map[string]any{"detail": "Invalid per_page parameter."})
		}
		if !parsed.IsInt64() {
			if parsed.Sign() > 0 {
				return nil, invitationsMembershipError(400, map[string]any{"detail": "Invalid per_page value. Cannot exceed 1000."})
			}
			return nil, invitationsMembershipError(500, map[string]any{"error": "Something went wrong please try again later"})
		}
		perPage = int(parsed.Int64())
	}
	if perPage > 1000 {
		return nil, invitationsMembershipError(400, map[string]any{"detail": "Invalid per_page value. Cannot exceed 1000."})
	}
	page, err := store.ParseMemberCursor(payload.Cursor, perPage)
	if err != nil {
		return nil, invitationsMembershipError(400, map[string]any{"detail": "Invalid cursor parameter."})
	}
	if perPage == 0 {
		return nil, invitationsMembershipError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	if page.Page < 0 {
		return nil, invitationsMembershipError(400, map[string]any{"detail": "Error in parsing"})
	}
	if page.Previous && page.Value != float64(perPage) {
		return nil, invitationsMembershipError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	if perPage < 0 {
		return nil, invitationsMembershipError(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	query, err := workspaceMemberFilters(ctx, payload)
	if err != nil {
		return nil, err
	}
	query.Limit, query.Page = perPage, page.Page
	result, err := service.repository.ListWorkspaceMembers(ctx, stduuid.UUID(principal.UserID), payload.Slug, query)
	if err != nil {
		return nil, memberDirectoryError(err)
	}
	var members []*api.PlaneAPISerializersMemberWorkspacememberliteapiserializerResponse
	if decodeErr := json.Unmarshal(result.Results, &members); decodeErr != nil {
		return nil, invitationsMembershipRepositoryError(decodeErr)
	}
	envelope, err := store.NewMemberPage(members, result.Total, perPage, page.Page)
	if err != nil {
		return nil, memberDirectoryError(err)
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return nil, invitationsMembershipRepositoryError(err)
	}
	return decodeInvitationsMembershipResponse[*api.RestFrameworkSerializersPaginatedworkspacememberliteResponse](ctx, encoded)
}

func (service *invitationsMembershipService) GetAPIUsersMeWorkspacesInvitations(ctx context.Context, payload *api.GetAPIUsersMeWorkspacesInvitationsPayload) ([]*api.PlaneAppSerializersWorkspaceWorkspacememberinviteserializerResponse, error) {
	data, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "user-workspace-list"})
	if err != nil {
		return nil, err
	}
	return decodeInvitationsMembershipResponse[[]*api.PlaneAppSerializersWorkspaceWorkspacememberinviteserializerResponse](ctx, data)
}

func (service *invitationsMembershipService) PostAPIUsersMeWorkspacesInvitations(ctx context.Context, payload *api.PostAPIUsersMeWorkspacesInvitationsPayload, body io.ReadCloser) error {
	_, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "user-workspace-accept", Body: body})
	return err
}

func (service *invitationsMembershipService) GetAPIUsersMeWorkspacesBySlugProjectsInvitations(ctx context.Context, payload *api.GetAPIUsersMeWorkspacesBySlugProjectsInvitationsPayload) ([]*api.PlaneAppSerializersProjectProjectmemberinviteserializerResponse, error) {
	data, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "user-project-list", Slug: payload.Slug})
	if err != nil {
		return nil, err
	}
	return decodeInvitationsMembershipResponse[[]*api.PlaneAppSerializersProjectProjectmemberinviteserializerResponse](ctx, data)
}

func (service *invitationsMembershipService) PostAPIUsersMeWorkspacesBySlugProjectsInvitations(ctx context.Context, payload *api.PostAPIUsersMeWorkspacesBySlugProjectsInvitationsPayload, body io.ReadCloser) (*api.RestFrameworkSerializersMessageresponseResponse, error) {
	data, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "user-project-join", Slug: payload.Slug, Body: body})
	if err != nil {
		return nil, err
	}
	return decodeInvitationsMembershipResponse[*api.RestFrameworkSerializersMessageresponseResponse](ctx, data)
}

func (service *invitationsMembershipService) PutAPIV1WorkspacesBySlugInvitationsByPk(ctx context.Context, payload *api.PutAPIV1WorkspacesBySlugInvitationsByPkPayload, body io.ReadCloser) (*api.PlaneAPISerializersInviteWorkspaceinviteserializerResponse, error) {
	data, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "v1-put", Slug: payload.Slug, ID: payload.Pk, Body: body})
	if err != nil {
		return nil, err
	}
	return decodeInvitationsMembershipResponse[*api.PlaneAPISerializersInviteWorkspaceinviteserializerResponse](ctx, data)
}

func (service *invitationsMembershipService) GetAPIWorkspacesBySlugInvitations(ctx context.Context, payload *api.GetAPIWorkspacesBySlugInvitationsPayload) ([]*api.PlaneAppSerializersWorkspaceWorkspacememberinviteserializerResponse, error) {
	data, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "workspace-list", Slug: payload.Slug})
	if err != nil {
		return nil, err
	}
	return decodeInvitationsMembershipResponse[[]*api.PlaneAppSerializersWorkspaceWorkspacememberinviteserializerResponse](ctx, data)
}

func (service *invitationsMembershipService) PostAPIWorkspacesBySlugInvitations(ctx context.Context, payload *api.PostAPIWorkspacesBySlugInvitationsPayload, body io.ReadCloser) (*api.RestFrameworkSerializersMessageresponseResponse, error) {
	data, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "workspace-create", Slug: payload.Slug, Body: body})
	if err != nil {
		return nil, err
	}
	return decodeInvitationsMembershipResponse[*api.RestFrameworkSerializersMessageresponseResponse](ctx, data)
}

func (service *invitationsMembershipService) DeleteAPIWorkspacesBySlugInvitationsByPk(ctx context.Context, payload *api.DeleteAPIWorkspacesBySlugInvitationsByPkPayload) error {
	_, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "workspace-delete", Slug: payload.Slug, ID: payload.Pk})
	return err
}

func (service *invitationsMembershipService) GetAPIWorkspacesBySlugInvitationsByPk(ctx context.Context, payload *api.GetAPIWorkspacesBySlugInvitationsByPkPayload) (*api.PlaneAppSerializersWorkspaceWorkspacememberinviteserializerResponse, error) {
	data, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "workspace-detail", Slug: payload.Slug, ID: payload.Pk})
	if err != nil {
		return nil, err
	}
	return decodeInvitationsMembershipResponse[*api.PlaneAppSerializersWorkspaceWorkspacememberinviteserializerResponse](ctx, data)
}

func (service *invitationsMembershipService) PatchAPIWorkspacesBySlugInvitationsByPk(ctx context.Context, payload *api.PatchAPIWorkspacesBySlugInvitationsByPkPayload, body io.ReadCloser) (*api.PlaneAppSerializersWorkspaceWorkspacememberinviteserializerResponse, error) {
	data, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "workspace-patch", Slug: payload.Slug, ID: payload.Pk, Body: body})
	if err != nil {
		return nil, err
	}
	return decodeInvitationsMembershipResponse[*api.PlaneAppSerializersWorkspaceWorkspacememberinviteserializerResponse](ctx, data)
}

func (service *invitationsMembershipService) GetAPIWorkspacesBySlugInvitationsByPkJoin(ctx context.Context, payload *api.GetAPIWorkspacesBySlugInvitationsByPkJoinPayload) (*api.PlaneAppSerializersWorkspaceWorkspacememberinvitepublicserializerResponse, error) {
	data, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "workspace-public", Slug: payload.Slug, ID: payload.Pk})
	if err != nil {
		return nil, err
	}
	return decodeInvitationsMembershipResponse[*api.PlaneAppSerializersWorkspaceWorkspacememberinvitepublicserializerResponse](ctx, data)
}

func (service *invitationsMembershipService) PostAPIWorkspacesBySlugInvitationsByPkJoin(ctx context.Context, payload *api.PostAPIWorkspacesBySlugInvitationsByPkJoinPayload, body io.ReadCloser) (*api.RestFrameworkSerializersMessageresponseResponse, error) {
	data, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "workspace-join", Slug: payload.Slug, ID: payload.Pk, Body: body})
	if err != nil {
		return nil, err
	}
	return decodeInvitationsMembershipResponse[*api.RestFrameworkSerializersMessageresponseResponse](ctx, data)
}

func (service *invitationsMembershipService) PostAPIWorkspacesBySlugMembersLeave(ctx context.Context, payload *api.PostAPIWorkspacesBySlugMembersLeavePayload, body io.ReadCloser) error {
	_, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "member-leave", Slug: payload.Slug, Body: body})
	return err
}

func (service *invitationsMembershipService) DeleteAPIWorkspacesBySlugMembersByPk(ctx context.Context, payload *api.DeleteAPIWorkspacesBySlugMembersByPkPayload) error {
	_, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "member-delete", Slug: payload.Slug, ID: payload.Pk})
	return err
}

func (service *invitationsMembershipService) PatchAPIWorkspacesBySlugMembersByPk(ctx context.Context, payload *api.PatchAPIWorkspacesBySlugMembersByPkPayload, body io.ReadCloser) (*api.PlaneAppSerializersWorkspaceWorkspacememberadminserializerResponse, error) {
	data, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "member-patch", Slug: payload.Slug, ID: payload.Pk, Body: body})
	if err != nil {
		return nil, err
	}
	return decodeInvitationsMembershipResponse[*api.PlaneAppSerializersWorkspaceWorkspacememberadminserializerResponse](ctx, data)
}

func (service *invitationsMembershipService) GetAPIWorkspacesBySlugProjectsByProjectIDInvitations(ctx context.Context, payload *api.GetAPIWorkspacesBySlugProjectsByProjectIDInvitationsPayload) ([]*api.PlaneAppSerializersProjectProjectmemberinviteserializerResponse, error) {
	data, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "project-list", Slug: payload.Slug, ProjectID: payload.ProjectID})
	if err != nil {
		return nil, err
	}
	return decodeInvitationsMembershipResponse[[]*api.PlaneAppSerializersProjectProjectmemberinviteserializerResponse](ctx, data)
}

func (service *invitationsMembershipService) PostAPIWorkspacesBySlugProjectsByProjectIDInvitations(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDInvitationsPayload, body io.ReadCloser) (*api.RestFrameworkSerializersMessageresponseResponse, error) {
	data, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "project-create", Slug: payload.Slug, ProjectID: payload.ProjectID, Body: body})
	if err != nil {
		return nil, err
	}
	return decodeInvitationsMembershipResponse[*api.RestFrameworkSerializersMessageresponseResponse](ctx, data)
}

func (service *invitationsMembershipService) DeleteAPIWorkspacesBySlugProjectsByProjectIDInvitationsByPk(ctx context.Context, payload *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDInvitationsByPkPayload) error {
	_, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "project-delete", Slug: payload.Slug, ProjectID: payload.ProjectID, ID: payload.Pk})
	return err
}

func (service *invitationsMembershipService) GetAPIWorkspacesBySlugProjectsByProjectIDInvitationsByPk(ctx context.Context, payload *api.GetAPIWorkspacesBySlugProjectsByProjectIDInvitationsByPkPayload) (*api.PlaneAppSerializersProjectProjectmemberinviteserializerResponse, error) {
	data, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "project-detail", Slug: payload.Slug, ProjectID: payload.ProjectID, ID: payload.Pk})
	if err != nil {
		return nil, err
	}
	return decodeInvitationsMembershipResponse[*api.PlaneAppSerializersProjectProjectmemberinviteserializerResponse](ctx, data)
}

func (service *invitationsMembershipService) GetAPIWorkspacesBySlugProjectsByProjectIDJoinByPk(ctx context.Context, payload *api.GetAPIWorkspacesBySlugProjectsByProjectIDJoinByPkPayload) (*api.PlaneAppSerializersProjectProjectmemberinvitepublicserializerResponse, error) {
	data, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "project-public", Slug: payload.Slug, ProjectID: payload.ProjectID, ID: payload.Pk})
	if err != nil {
		return nil, err
	}
	return decodeInvitationsMembershipResponse[*api.PlaneAppSerializersProjectProjectmemberinvitepublicserializerResponse](ctx, data)
}

func (service *invitationsMembershipService) PostAPIWorkspacesBySlugProjectsByProjectIDJoinByPk(ctx context.Context, payload *api.PostAPIWorkspacesBySlugProjectsByProjectIDJoinByPkPayload, body io.ReadCloser) (*api.RestFrameworkSerializersMessageresponseResponse, error) {
	data, err := service.executeInvitation(ctx, store.InvitationCommand{Operation: "project-join", Slug: payload.Slug, ProjectID: payload.ProjectID, ID: payload.Pk, Body: body})
	if err != nil {
		return nil, err
	}
	return decodeInvitationsMembershipResponse[*api.RestFrameworkSerializersMessageresponseResponse](ctx, data)
}

type invitationsMembershipCommands interface {
	Invitation(context.Context, store.InvitationCommand) (json.RawMessage, error)
	AuthorizeProjectInvitations(context.Context, string, string, string) error
}

func (service *invitationsMembershipService) optionalInvitationPrincipal(ctx context.Context) invitationsMembershipPrincipal {
	if principal, ok := ctx.Value(invitationsMembershipPrincipalKey{}).(invitationsMembershipPrincipal); ok {
		return principal
	}
	credential := ""
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		request := &http.Request{Header: metadata.Headers()}
		if cookie, err := request.Cookie("session-id"); err == nil {
			credential = cookie.Value
		}
	}
	principalContext, err := service.APIKeyAuth(ctx, credential, &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	if err != nil {
		return invitationsMembershipPrincipal{}
	}
	principal, _ := principalContext.Value(invitationsMembershipPrincipalKey{}).(invitationsMembershipPrincipal)
	return principal
}
func (service *invitationsMembershipService) executeInvitation(ctx context.Context, c store.InvitationCommand) (json.RawMessage, error) {
	repository, ok := service.repository.(invitationsMembershipCommands)
	if !ok {
		return nil, invitationsMembershipRepositoryError(errors.New("invitation command repository unavailable"))
	}
	public := c.Operation == "workspace-public" || c.Operation == "workspace-join" || c.Operation == "project-public" || c.Operation == "project-join"
	principal := service.optionalInvitationPrincipal(ctx)
	if !public && principal.UserID == uuid.Nil {
		return nil, invitationsMembershipError(401, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	if principal.UserID != uuid.Nil {
		c.UserID = principal.UserID.String()
	}
	c.Timezone = principal.Timezone
	c.ContentType = invitationContentType(ctx)
	c.Cache = service.cache
	// Class permissions run during initial(), before the app's timezone hook.
	if strings.HasPrefix(c.Operation, "workspace-") && !public {
		if err := service.repository.AuthorizeWorkspaceRole(ctx, principal.UserID, c.Slug, 15, 20); err != nil {
			return nil, invitationsMembershipRepositoryError(err)
		}
	}
	if c.Operation == "v1-put" {
		if err := service.authorizeWorkspace(ctx, principal, c.Slug, false, 20); err != nil {
			return nil, err
		}
	}
	if principal.UserID != uuid.Nil {
		if err := djangosession.ValidateUserTimezone(principal.Timezone); err != nil {
			if errors.Is(err, djangosession.ErrTimezoneNotFound) {
				return nil, invitationsMembershipError(400, map[string]any{"error": "The required key does not exist."})
			}
			return nil, invitationsMembershipRepositoryError(err)
		}
	}
	// Cache decorators precede action-level permission and body validation.
	if service.cache != nil {
		invalidate := func(path, user string, multiple bool) error {
			return service.cache.Invalidate(ctx, path, user, multiple)
		}
		var err error
		switch c.Operation {
		case "workspace-join":
			for _, entry := range []struct {
				path, user string
				multiple   bool
			}{{"/api/workspaces/", "", false}, {"/api/users/me/workspaces/", c.UserID, true}, {"/api/workspaces/" + c.Slug + "/members/", "", true}, {"/api/users/me/settings/", c.UserID, true}} {
				if err = invalidate(entry.path, entry.user, entry.multiple); err != nil {
					break
				}
			}
		case "user-workspace-accept":
			err = invalidate("/api/workspaces/", "", false)
			if err == nil {
				err = invalidate("/api/users/me/workspaces/", c.UserID, true)
			}
		case "member-leave":
			err = invalidate("/api/workspaces/"+c.Slug+"/members/", "", true)
			if err == nil {
				err = invalidate("/api/users/me/settings/", c.UserID, false)
			}
			if err == nil {
				err = invalidate("api/users/me/workspaces/", "", true)
			}
		}
		if err != nil {
			return nil, invitationsMembershipRepositoryError(err)
		}
	}
	var err error
	switch c.Operation {
	case "project-list", "project-detail", "project-delete", "project-create":
		err = repository.AuthorizeProjectInvitations(ctx, c.UserID, c.Slug, c.ProjectID)
	case "user-project-join":
		err = service.repository.AuthorizeWorkspaceRole(ctx, principal.UserID, c.Slug, 15, 20)
	case "member-patch", "member-delete":
		err = service.repository.AuthorizeWorkspaceRole(ctx, principal.UserID, c.Slug, 20)
	case "member-leave":
		err = service.repository.AuthorizeWorkspaceRole(ctx, principal.UserID, c.Slug, 5, 15, 20)
	}
	if errors.Is(err, store.ErrForbidden) {
		err = &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "You don't have the required permissions."}}
	}
	if err != nil {
		return nil, invitationsMembershipRepositoryError(err)
	}
	result, err := repository.Invitation(store.WithInvitationActor(ctx, c.UserID), c)
	if err == nil {
		result = normalizeInvitationResponse(result, principal.Timezone)
	}
	return result, invitationsMembershipRepositoryError(err)
}

// Walk serializer objects only. Timestamp-shaped strings in model JSON fields
// remain application data, just as DRF's JSONField leaves them unchanged.
func normalizeInvitationResponse(raw json.RawMessage, zone string) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var body any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		return raw
	}
	location := time.UTC
	if zone != "" {
		if value, err := time.LoadLocation(zone); err == nil {
			location = value
		}
	}
	var walk func(any)
	walk = func(value any) {
		switch v := value.(type) {
		case []any:
			for _, child := range v {
				walk(child)
			}
		case map[string]any:
			for _, field := range []string{"created_at", "updated_at", "responded_at", "deleted_at"} {
				if text, ok := v[field].(string); ok {
					if parsed, err := time.Parse(time.RFC3339Nano, text); err == nil {
						layout := time.RFC3339
						if parsed.Nanosecond() != 0 {
							layout = "2006-01-02T15:04:05.000000Z07:00"
						}
						v[field] = parsed.In(location).Format(layout)
					}
				}
			}
			for _, key := range []string{"workspace", "project", "member", "results"} {
				if child, ok := v[key]; ok {
					walk(child)
				}
			}
		}
	}
	walk(body)
	data, err := json.Marshal(body)
	if err != nil {
		return raw
	}
	return data
}
