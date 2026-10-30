package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/platform"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	store "github.com/CaliLuke/go-plane/internal/workspacewebhooks"
	api "github.com/CaliLuke/go-plane/workspace_webhooks/gen/workspace_webhooks"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type workspaceWebhooksRepo interface {
	Authorized(context.Context, uuid.UUID, string) (bool, error)
	List(context.Context, string) (json.RawMessage, error)
	Get(context.Context, string, uuid.UUID) (json.RawMessage, error)
	Create(context.Context, store.CreateParams) (json.RawMessage, error)
	Update(context.Context, store.PatchParams) (json.RawMessage, error)
	SoftDelete(context.Context, string, uuid.UUID, uuid.UUID) error
	Regenerate(context.Context, string, uuid.UUID, uuid.UUID, string) (json.RawMessage, error)
	Logs(context.Context, string, uuid.UUID) (json.RawMessage, error)
}

type workspaceWebhooksSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type workspaceWebhooksPublisher interface {
	SoftDelete(context.Context, string, string, string) error
}

type workspaceWebhooksService struct {
	repo      workspaceWebhooksRepo
	sessions  workspaceWebhooksSessions
	publisher workspaceWebhooksPublisher
}

type workspaceWebhooksActor struct{}

type webhookCreateRequest struct {
	URL          loom.Nullable[string] `json:"url"`
	IsActive     *bool                 `json:"is_active"`
	Project      *bool                 `json:"project"`
	Issue        *bool                 `json:"issue"`
	Module       *bool                 `json:"module"`
	Milestone    *bool                 `json:"milestone"`
	Cycle        *bool                 `json:"cycle"`
	IssueComment *bool                 `json:"issue_comment"`
	IsInternal   *bool                 `json:"is_internal"`
	Version      *string               `json:"version"`
	CreatedBy    loom.Nullable[string] `json:"created_by"`
}

type webhookPatchRequest webhookCreateRequest

func parseWebhookBody[T any](ctx context.Context, body io.ReadCloser) (*T, error) {
	if body != nil {
		defer platform.CloseRequestBody(ctx, body)
	}
	contentType := ""
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		contentType = metadata.Headers().Get("Content-Type")
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return nil, &api.WebhookInvalid{NonFieldErrors: []string{err.Error()}}
	}
	if input.Kind != "" {
		return nil, &api.WebhookInvalid{NonFieldErrors: []string{"Invalid data. Expected a dictionary, but got " + input.Kind + "."}}
	}
	encoded, err := json.Marshal(input.Fields)
	if err != nil {
		return nil, &api.WebhookInvalid{NonFieldErrors: []string{"Invalid data."}}
	}
	var request T
	if err := json.Unmarshal(encoded, &request); err != nil {
		return nil, &api.WebhookInvalid{NonFieldErrors: []string{"Invalid data."}}
	}
	return &request, nil
}

func NewWorkspaceWebhooks(repo workspaceWebhooksRepo, sessions workspaceWebhooksSessions, publisher workspaceWebhooksPublisher) (api.Service, error) {
	if repo == nil || sessions == nil || publisher == nil {
		return nil, fmt.Errorf("workspace webhooks dependencies are required")
	}
	return &workspaceWebhooksService{repo: repo, sessions: sessions, publisher: publisher}, nil
}

func (s *workspaceWebhooksService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	userID, timezone, inspected := readServiceSessionPrincipal(ctx)
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
			userID, timezone = principal.UserID, principal.UserTimezone
		}
	}
	actor, err := uuid.Parse(userID)
	if userID == "" || err != nil {
		return ctx, &api.WebhookUnauthenticated{Detail: unauthenticatedDetail}
	}
	ctx = context.WithValue(ctx, workspaceWebhooksActor{}, actor)
	ctx = context.WithValue(ctx, workspaceWebhooksPrincipal{}, djangosession.Principal{UserID: userID, UserTimezone: timezone})
	return ctx, nil
}

type workspaceWebhooksPrincipal struct{}

func (s *workspaceWebhooksService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	_, err := s.APIKeyAuth(ctx, credential, nil)
	if err != nil {
		return http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail}
	}
	return 0, nil
}

func (s *workspaceWebhooksService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, resource SessionProtocolResource) (int, map[string]any) {
	ctx, err := s.APIKeyAuth(ctx, credential, nil)
	if err != nil {
		return http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail}
	}
	// DRF BaseAPIView authenticates protocol-only OPTIONS and unsupported
	// methods, but workspace role checks are reached by decorated business
	// actions and by HEAD's implicit GET dispatch.
	switch resource.Method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPatch, http.MethodDelete:
	default:
		return 0, nil
	}
	if _, err = s.actor(ctx, resource.Slug); err != nil {
		var forbidden *api.WebhookForbidden
		if errors.As(err, &forbidden) {
			return http.StatusForbidden, map[string]any{"error": forbidden.Message}
		}
		return http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"}
	}
	return 0, nil
}

func (s *workspaceWebhooksService) SessionProtocolOptionsMetadata(_ context.Context, _ string, resource SessionProtocolResource, _ json.RawMessage, _ json.RawMessage) (int, any, bool) {
	if resource.Pattern != "/api/workspaces/{slug}/webhooks/" {
		return 0, nil, false
	}
	return http.StatusOK, map[string]any{
		"name":        "Webhook Endpoint",
		"description": "",
		"renders":     []string{"application/json"},
		"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
	}, true
}

func (s *workspaceWebhooksService) actor(ctx context.Context, slug string) (uuid.UUID, error) {
	actor, ok := ctx.Value(workspaceWebhooksActor{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.WebhookUnauthenticated{Detail: unauthenticatedDetail}
	}
	allowed, err := s.repo.Authorized(ctx, actor, slug)
	if err != nil {
		return uuid.Nil(), internalWebhookError()
	}
	if !allowed {
		return uuid.Nil(), &api.WebhookForbidden{Message: "You don't have the required permissions."}
	}
	return actor, nil
}

func (s *workspaceWebhooksService) List(ctx context.Context, p *api.ListPayload) ([]*api.Webhook, error) {
	if _, err := s.actor(ctx, p.Slug); err != nil {
		return nil, err
	}
	data, err := s.repo.List(ctx, p.Slug)
	if err != nil {
		return nil, internalWebhookError()
	}
	var result []*api.Webhook
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, internalWebhookError()
	}
	return result, nil
}

func (s *workspaceWebhooksService) Create(ctx context.Context, p *api.CreatePayload, body io.ReadCloser) (*api.Webhook, error) {
	actor, err := s.actor(ctx, p.Slug)
	if err != nil {
		return nil, err
	}
	request, err := parseWebhookBody[webhookCreateRequest](ctx, body)
	if err != nil {
		return nil, err
	}
	url, urlPresent := request.URL.Value()
	if !urlPresent {
		return nil, &api.WebhookInvalid{URL: []string{"This field is required."}}
	}
	if request.URL.IsNull() {
		return nil, &api.WebhookInvalid{URL: []string{"This field may not be null."}}
	}
	metadata, _ := loomhttp.RequestMetadataFromContext(ctx)
	if message, validationErr := store.ValidateURL(ctx, url, metadata.Host); validationErr != nil {
		return nil, &api.WebhookInvalid{URL: []string{message}}
	}
	createdBy, createdByPresent, err := optionalUUID(request.CreatedBy)
	if err != nil {
		return nil, &api.WebhookInvalid{CreatedBy: []string{"Must be a valid UUID."}}
	}
	// BaseModel.save replaces a null created_by on insert with the current
	// request user; explicit null remains writable during partial updates.
	createdByPresent = createdByPresent && createdBy != nil
	input := store.CreateParams{
		ActorID: actor, Slug: p.Slug, CreatedBy: createdBy, CreatedByPresent: createdByPresent,
		URL: url, IsActive: boolDefault(request.IsActive, true), SecretKey: "plane_wh_" + strings.ReplaceAll(uuid.New().String(), "-", ""),
		Project: boolDefault(request.Project, false), Issue: boolDefault(request.Issue, false), Module: boolDefault(request.Module, false),
		Milestone: boolDefault(request.Milestone, false),
		Cycle:     boolDefault(request.Cycle, false), IssueComment: boolDefault(request.IssueComment, false),
		IsInternal: boolDefault(request.IsInternal, false), Version: stringDefault(request.Version, "v1"),
	}
	data, err := s.repo.Create(ctx, input)
	if errors.Is(err, store.ErrDuplicateURL) {
		return nil, &api.WebhookConflict{Message: "URL already exists for the workspace"}
	}
	if errors.Is(err, store.ErrIntegrity) {
		return nil, webhookPayloadInvalidError()
	}
	if err != nil {
		return nil, internalWebhookError()
	}
	return decodeWebhook(data)
}

func (s *workspaceWebhooksService) PatchCollection(ctx context.Context, p *api.PatchCollectionPayload) error {
	if _, err := s.actor(ctx, p.Slug); err != nil {
		return err
	}
	return &api.WebhookMethodNotAllowed{Detail: `Method "PATCH" not allowed.`}
}

func (s *workspaceWebhooksService) DeleteCollection(ctx context.Context, p *api.DeleteCollectionPayload) error {
	if _, err := s.actor(ctx, p.Slug); err != nil {
		return err
	}
	return &api.WebhookMethodNotAllowed{Detail: `Method "DELETE" not allowed.`}
}

func (s *workspaceWebhooksService) Detail(ctx context.Context, p *api.DetailPayload) (*api.Webhook, error) {
	if _, err := s.actor(ctx, p.Slug); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(p.Pk)
	if err != nil {
		return nil, webhookNotFoundError()
	}
	data, err := s.repo.Get(ctx, p.Slug, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil, webhookNotFoundError()
	}
	if err != nil {
		return nil, internalWebhookError()
	}
	return decodeWebhook(data)
}

func (s *workspaceWebhooksService) PatchDetail(ctx context.Context, p *api.PatchDetailPayload, body io.ReadCloser) (*api.Webhook, error) {
	actor, err := s.actor(ctx, p.Slug)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(p.Pk)
	if err != nil {
		return nil, webhookNotFoundError()
	}
	// Django's UpdateAPIView looks up the object before UpdateModelMixin reads
	// and validates request.data. Keep malformed bodies from changing a missing
	// resource response from 404 to 400.
	if _, getErr := s.repo.Get(ctx, p.Slug, id); errors.Is(getErr, store.ErrNotFound) {
		return nil, webhookNotFoundError()
	} else if getErr != nil {
		return nil, internalWebhookError()
	}
	request, err := parseWebhookBody[webhookPatchRequest](ctx, body)
	if err != nil {
		return nil, err
	}
	var url string
	if request.URL.Present() {
		if request.URL.IsNull() {
			return nil, &api.WebhookInvalid{URL: []string{"This field may not be null."}}
		}
		url, _ = request.URL.Value()
		if url == "" {
			return nil, &api.WebhookInvalid{URL: []string{"This field may not be blank."}}
		}
		metadata, _ := loomhttp.RequestMetadataFromContext(ctx)
		if message, validationErr := store.ValidateURL(ctx, url, metadata.Host); validationErr != nil {
			return nil, &api.WebhookInvalid{URL: []string{message}}
		}
	}
	createdBy, createdByPresent, err := optionalUUID(request.CreatedBy)
	if err != nil {
		return nil, &api.WebhookInvalid{CreatedBy: []string{"Must be a valid UUID."}}
	}
	data, err := s.repo.Update(ctx, store.PatchParams{
		ActorID: actor, Slug: p.Slug, ID: id,
		URL: url, URLPresent: request.URL.Present(),
		IsActive: boolDefault(request.IsActive, false), IsActivePresent: request.IsActive != nil,
		Project: boolDefault(request.Project, false), ProjectPresent: request.Project != nil,
		Issue: boolDefault(request.Issue, false), IssuePresent: request.Issue != nil,
		Module: boolDefault(request.Module, false), ModulePresent: request.Module != nil,
		Cycle: boolDefault(request.Cycle, false), CyclePresent: request.Cycle != nil,
		Milestone: boolDefault(request.Milestone, false), MilestonePresent: request.Milestone != nil,
		IssueComment: boolDefault(request.IssueComment, false), IssueCommentPresent: request.IssueComment != nil,
		IsInternal: boolDefault(request.IsInternal, false), IsInternalPresent: request.IsInternal != nil,
		Version: stringDefault(request.Version, ""), VersionPresent: request.Version != nil,
		CreatedBy: createdBy, CreatedByPresent: createdByPresent,
	})
	if errors.Is(err, store.ErrNotFound) {
		return nil, webhookNotFoundError()
	}
	if errors.Is(err, store.ErrIntegrity) {
		return nil, webhookPayloadInvalidError()
	}
	if err != nil {
		return nil, internalWebhookError()
	}
	return decodeWebhook(data)
}

func (s *workspaceWebhooksService) DeleteDetail(ctx context.Context, p *api.DeleteDetailPayload) error {
	actor, err := s.actor(ctx, p.Slug)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(p.Pk)
	if err != nil {
		return webhookNotFoundError()
	}
	if err = s.repo.SoftDelete(ctx, p.Slug, id, actor); errors.Is(err, store.ErrNotFound) {
		return webhookNotFoundError()
	} else if err != nil {
		return internalWebhookError()
	}
	if err = s.publisher.SoftDelete(ctx, "db", "webhook", id.String()); err != nil {
		return internalWebhookError()
	}
	return nil
}

func (s *workspaceWebhooksService) Regenerate(ctx context.Context, p *api.RegeneratePayload) (*api.Webhook, error) {
	actor, err := s.actor(ctx, p.Slug)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(p.Pk)
	if err != nil {
		return nil, webhookNotFoundError()
	}
	secret := "plane_wh_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	data, err := s.repo.Regenerate(ctx, p.Slug, id, actor, secret)
	if errors.Is(err, store.ErrNotFound) {
		return nil, webhookNotFoundError()
	}
	if err != nil {
		return nil, internalWebhookError()
	}
	return decodeWebhook(data)
}

func (s *workspaceWebhooksService) Logs(ctx context.Context, p *api.LogsPayload) ([]*api.WebhookLog, error) {
	if _, err := s.actor(ctx, p.Slug); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(p.WebhookID)
	if err != nil {
		return nil, &api.WebhookNotFound{Message: "Page not found."}
	}
	data, err := s.repo.Logs(ctx, p.Slug, id)
	if err != nil {
		return nil, internalWebhookError()
	}
	var result []*api.WebhookLog
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, internalWebhookError()
	}
	return result, nil
}

func decodeWebhook(data json.RawMessage) (*api.Webhook, error) {
	var result api.Webhook
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, internalWebhookError()
	}
	return &result, nil
}

func optionalUUID(value loom.Nullable[string]) (*uuid.UUID, bool, error) {
	if !value.Present() {
		return nil, false, nil
	}
	if value.IsNull() {
		return nil, true, nil
	}
	text, _ := value.Value()
	id, err := uuid.Parse(text)
	if err != nil {
		return nil, true, err
	}
	return &id, true, nil
}

func boolDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func stringDefault(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}

func webhookNotFoundError() error {
	return &api.WebhookNotFound{Message: "The required object does not exist."}
}

func webhookPayloadInvalidError() *api.WebhookInvalid {
	message := "The payload is not valid"
	return &api.WebhookInvalid{Message: &message}
}

func internalWebhookError() error {
	return &api.WebhookInternal{Message: "Something went wrong please try again later"}
}
