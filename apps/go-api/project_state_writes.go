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

	api "github.com/CaliLuke/go-plane/gen/project_state_reads"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	store "github.com/CaliLuke/go-plane/internal/projectstatereads"
)

type projectStateWritesRepository interface {
	InvalidateAppCache(context.Context, string)
	AuthorizeApp(context.Context, string, string, string, string) (bool, error)
	ResolveForUpdate(context.Context, store.StateWriteInput) error
	Create(context.Context, store.StateWriteInput) (json.RawMessage, error)
	Update(context.Context, store.StateWriteInput) (json.RawMessage, error)
	Delete(context.Context, store.StateWriteInput) error
	MarkDefault(context.Context, store.StateWriteInput) error
}

func (service *projectStateReadsService) appStateWrite(ctx context.Context, operation, method, slug, projectID, stateID string, body io.Reader, contentType string) (*api.ProjectStateWritesAppResponseMetadata, io.ReadCloser, error) {
	writes, ok := service.repository.(projectStateWritesRepository)
	if !ok {
		return nil, nil, projectStateAppFailure(http.StatusInternalServerError, map[string]string{"error": projectStateReadsInternalErrorDetail})
	}
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, &api.ProjectStateWritesAppError{Kind: "unauthenticated", Body: json.RawMessage(`{"detail":"Authentication credentials were not provided."}`)}
	}
	// Django authentication runs before the view decorator; after that,
	// invalidate_cache wraps permission checks, parsing, and the handler.
	if operation == "create" || operation == "delete" || operation == "mark_default" {
		writes.InvalidateAppCache(ctx, slug)
	}
	allowed, err := writes.AuthorizeApp(ctx, userID.String(), slug, projectID, method)
	if err != nil {
		return nil, nil, projectStateAppFailure(500, map[string]string{"error": projectStateReadsInternalErrorDetail})
	}
	if !allowed {
		return nil, nil, projectStateAppFailure(403, map[string]string{"error": "You don't have the required permissions."})
	}
	input := store.StateWriteInput{UserID: userID.String(), Slug: slug, ProjectID: projectID, StateID: stateID, Method: method, App: true}
	if operation == "update" {
		if resolveForUpdateErr := writes.ResolveForUpdate(ctx, input); resolveForUpdateErr != nil {
			return nil, nil, projectStateAppError(resolveForUpdateErr)
		}
	}
	parsed := djangoinput.Input{Fields: map[string]any{}}
	if method == http.MethodPost || method == http.MethodPatch {
		parsed, err = store.ParseStateWriteBody(body, contentType)
		if err != nil {
			return nil, nil, projectStateAppError(err)
		}
	}
	input.Fields, input.Parsed = parsed.Fields, &parsed
	var result json.RawMessage
	switch operation {
	case "create":
		result, err = writes.Create(ctx, input)
	case "update":
		result, err = writes.Update(ctx, input)
	case "delete":
		err = writes.Delete(ctx, input)
	case "mark_default":
		err = writes.MarkDefault(ctx, input)
	default:
		err = fmt.Errorf("unknown project state operation %q", operation)
	}
	if err != nil {
		return nil, nil, projectStateAppError(err)
	}
	if operation == "delete" || operation == "mark_default" {
		return nil, nil, nil
	}
	return &api.ProjectStateWritesAppResponseMetadata{ContentType: "application/json"}, io.NopCloser(bytes.NewReader(result)), nil
}

func projectStateAppFailure(status int, body any) error {
	kind := map[int]string{400: "bad_request", 401: "unauthenticated", 403: "permission_denied", 404: "not_found", 415: "unsupported_media_type"}[status]
	if kind == "" {
		kind = "internal_error"
	}
	raw, err := json.Marshal(body)
	if err != nil {
		raw = []byte(`{"error":"Something went wrong please try again later"}`)
	}
	return &api.ProjectStateWritesAppError{Kind: kind, Body: json.RawMessage(raw)}
}

func projectStateAppError(err error) error {
	var failure *store.WriteFailure
	if errors.As(err, &failure) {
		return projectStateAppFailure(failure.Status, failure.Body)
	}
	return projectStateAppFailure(500, map[string]string{"error": projectStateReadsInternalErrorDetail})
}

// AuthorizeSessionProtocol and AuthorizeSessionResourceProtocol keep app state
// protocol companions inside the same authentication boundary as generated
// methods. Django's action decorators do not run for OPTIONS or unsupported
// verbs, so these methods intentionally do not apply project-role checks.
func (service *projectStateReadsService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	if _, err := service.APIKeyAuth(ctx, credential, nil); err != nil {
		return 401, map[string]any{"detail": unauthenticatedDetail}
	}
	return 0, nil
}

func (service *projectStateReadsService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, _ SessionProtocolResource) (int, map[string]any) {
	return service.AuthorizeSessionProtocol(ctx, credential)
}

func (service *projectStateReadsService) SessionProtocolOptionsMetadata(_ context.Context, _ string, resource SessionProtocolResource, metadata, _ json.RawMessage) (int, any, bool) {
	if !strings.Contains(resource.Pattern, "/projects/{project_id}/states") {
		return 0, nil, false
	}
	if len(metadata) > 0 {
		return 200, metadata, true
	}
	return 200, map[string]any{
		"name": "State Endpoint", "description": "", "renders": []string{"application/json"},
		"parses": []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
	}, true
}

func (service *projectStateReadsService) Create(ctx context.Context, payload *api.CreatePayload, body io.ReadCloser) (*api.ProjectStateWritesAppResponseMetadata, io.ReadCloser, error) {
	contentType := ""
	if payload.ContentType != nil {
		contentType = *payload.ContentType
	}
	return service.appStateWrite(ctx, "create", http.MethodPost, payload.Slug, payload.ProjectID, "", body, contentType)
}
func (service *projectStateReadsService) Update(ctx context.Context, payload *api.UpdatePayload, body io.ReadCloser) (*api.ProjectStateWritesAppResponseMetadata, io.ReadCloser, error) {
	contentType := ""
	if payload.ContentType != nil {
		contentType = *payload.ContentType
	}
	return service.appStateWrite(ctx, "update", http.MethodPatch, payload.Slug, payload.ProjectID, payload.Pk, body, contentType)
}
func (service *projectStateReadsService) Delete(ctx context.Context, payload *api.DeletePayload) error {
	_, _, err := service.appStateWrite(ctx, "delete", http.MethodDelete, payload.Slug, payload.ProjectID, payload.Pk, nil, "")
	return err
}
func (service *projectStateReadsService) MarkDefault(ctx context.Context, payload *api.MarkDefaultPayload) error {
	_, _, err := service.appStateWrite(ctx, "mark_default", http.MethodPost, payload.Slug, payload.ProjectID, payload.Pk, nil, "")
	return err
}
