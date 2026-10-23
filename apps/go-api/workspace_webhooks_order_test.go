package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"uuid"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workspacewebhooks"
	api "github.com/CaliLuke/go-plane/workspace_webhooks/gen/workspace_webhooks"
	loomhttp "github.com/CaliLuke/loom/http"
)

type missingWebhookRepository struct {
	getCalled    bool
	updateCalled bool
}

type existingWebhookRepository struct {
	missingWebhookRepository
}

func (r *existingWebhookRepository) Get(context.Context, string, uuid.UUID) (json.RawMessage, error) {
	r.getCalled = true
	return json.RawMessage(`{}`), nil
}

func (*missingWebhookRepository) Authorized(context.Context, uuid.UUID, string) (bool, error) {
	return true, nil
}
func (*missingWebhookRepository) List(context.Context, string) (json.RawMessage, error) {
	return json.RawMessage(`[]`), nil
}
func (r *missingWebhookRepository) Get(context.Context, string, uuid.UUID) (json.RawMessage, error) {
	r.getCalled = true
	return nil, store.ErrNotFound
}
func (*missingWebhookRepository) Create(context.Context, store.CreateParams) (json.RawMessage, error) {
	return nil, errors.New("unexpected create")
}
func (r *missingWebhookRepository) Update(context.Context, store.PatchParams) (json.RawMessage, error) {
	r.updateCalled = true
	return nil, errors.New("unexpected update")
}
func (*missingWebhookRepository) SoftDelete(context.Context, string, uuid.UUID, uuid.UUID) error {
	return errors.New("unexpected delete")
}
func (*missingWebhookRepository) Regenerate(context.Context, string, uuid.UUID, uuid.UUID, string) (json.RawMessage, error) {
	return nil, errors.New("unexpected regenerate")
}
func (*missingWebhookRepository) Logs(context.Context, string, uuid.UUID) (json.RawMessage, error) {
	return json.RawMessage(`[]`), nil
}

type unusedWebhookSessions struct{}

func (unusedWebhookSessions) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{}, errors.New("unexpected session validation")
}

type unusedWebhookPublisher struct{}

func (unusedWebhookPublisher) SoftDelete(context.Context, string, string, string) error {
	return errors.New("unexpected publish")
}

func TestWorkspaceWebhookPatchMissingObjectPrecedesMalformedBody(t *testing.T) {
	repository := &missingWebhookRepository{}
	service, err := NewWorkspaceWebhooks(repository, unusedWebhookSessions{}, unusedWebhookPublisher{})
	if err != nil {
		t.Fatal(err)
	}
	actor := uuid.MustParse("00000000-0000-4000-8000-000000034201")
	ctx := context.WithValue(context.Background(), workspaceWebhooksActor{}, actor)
	_, err = service.(*workspaceWebhooksService).PatchDetail(ctx, &api.PatchDetailPayload{
		Slug: "workspace-webhooks",
		Pk:   "00000000-0000-4000-8000-000000099999",
	}, io.NopCloser(strings.NewReader("[")))
	var notFound *api.WebhookNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("PatchDetail error = %v, want 404 not found", err)
	}
	if !repository.getCalled || repository.updateCalled {
		t.Fatalf("repository calls: Get=%t Update=%t; want lookup only", repository.getCalled, repository.updateCalled)
	}
}

func TestWorkspaceWebhookPatchNullURLRejectsBeforeUpdate(t *testing.T) {
	repository := &existingWebhookRepository{}
	service, err := NewWorkspaceWebhooks(repository, unusedWebhookSessions{}, unusedWebhookPublisher{})
	if err != nil {
		t.Fatal(err)
	}
	actor := uuid.MustParse("00000000-0000-4000-8000-000000034201")
	ctx := context.WithValue(context.Background(), workspaceWebhooksActor{}, actor)
	metadata, err := loomhttp.NewRequestMetadataPolicy([]string{"Content-Type"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var patchErr error
	handler := loomhttp.RequestMetadataMiddleware(metadata)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		_, patchErr = service.(*workspaceWebhooksService).PatchDetail(request.Context(), &api.PatchDetailPayload{
			Slug: "workspace-webhooks",
			Pk:   "00000000-0000-4000-8000-000000034202",
		}, io.NopCloser(strings.NewReader(`{"url":null}`)))
	}))
	request := httptest.NewRequest(http.MethodPatch, "/api/workspaces/workspace-webhooks/webhooks/00000000-0000-4000-8000-000000034202/", nil)
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(httptest.NewRecorder(), request.WithContext(ctx))
	var invalid *api.WebhookInvalid
	if !errors.As(patchErr, &invalid) || len(invalid.URL) != 1 || invalid.URL[0] != "This field may not be null." {
		t.Fatalf("PatchDetail error = %#v, want url null validation error", patchErr)
	}
	if !repository.getCalled || repository.updateCalled {
		t.Fatalf("repository calls: Get=%t Update=%t; want lookup only", repository.getCalled, repository.updateCalled)
	}
}

func TestWorkspaceWebhookMalformedLogsIDIsNotFound(t *testing.T) {
	service, err := NewWorkspaceWebhooks(&missingWebhookRepository{}, unusedWebhookSessions{}, unusedWebhookPublisher{})
	if err != nil {
		t.Fatal(err)
	}
	actor := uuid.MustParse("00000000-0000-4000-8000-000000034201")
	ctx := context.WithValue(context.Background(), workspaceWebhooksActor{}, actor)
	_, err = service.(*workspaceWebhooksService).Logs(ctx, &api.LogsPayload{
		Slug:      "workspace-webhooks",
		WebhookID: "not-a-uuid",
	})
	var notFound *api.WebhookNotFound
	if !errors.As(err, &notFound) || notFound.Message != "Page not found." {
		t.Fatalf("Logs error = %#v, want 404 page-not-found error", err)
	}
}

func TestWorkspaceWebhookForbiddenUsesSourceErrorEnvelope(t *testing.T) {
	data, err := json.Marshal(&api.WebhookForbidden{Message: "You don't have the required permissions."})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"error":"You don't have the required permissions."}` {
		t.Fatalf("forbidden JSON = %s, want source permission envelope", data)
	}
}
