package systemapi

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/page_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	store "github.com/CaliLuke/go-plane/internal/pagereads"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

//go:embed page_lifecycle_options_metadata.json
var pageLifecycleOptionsMetadataJSON []byte

var pageLifecycleOptionsMetadata = func() map[string]json.RawMessage {
	var result map[string]json.RawMessage
	if err := json.Unmarshal(pageLifecycleOptionsMetadataJSON, &result); err != nil {
		panic(err)
	}
	return result
}()

func (service *pageReadsService) CreateOptions(ctx context.Context, payload *api.CreateOptionsPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	return service.pageLifecycleOptions(ctx, payload.Slug, payload.ProjectID, "", "collection")
}
func (service *pageReadsService) UpdateOptions(ctx context.Context, payload *api.UpdateOptionsPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	return service.pageLifecycleOptions(ctx, payload.Slug, payload.ProjectID, payload.PageID, "detail")
}
func (service *pageReadsService) DescriptionOptions(ctx context.Context, payload *api.DescriptionOptionsPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	return service.pageLifecycleOptions(ctx, payload.Slug, payload.ProjectID, payload.PageID, "description")
}
func (service *pageReadsService) AccessOptions(ctx context.Context, payload *api.AccessOptionsPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	return service.pageLifecycleOptions(ctx, payload.Slug, payload.ProjectID, payload.PageID, "access")
}
func (service *pageReadsService) ArchiveOptions(ctx context.Context, payload *api.ArchiveOptionsPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	return service.pageLifecycleOptions(ctx, payload.Slug, payload.ProjectID, payload.PageID, "archive")
}
func (service *pageReadsService) DuplicateOptions(ctx context.Context, payload *api.DuplicateOptionsPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	return service.pageLifecycleOptions(ctx, payload.Slug, payload.ProjectID, payload.PageID, "duplicate")
}
func (service *pageReadsService) LockOptions(ctx context.Context, payload *api.LockOptionsPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	return service.pageLifecycleOptions(ctx, payload.Slug, payload.ProjectID, payload.PageID, "lock")
}

func (service *pageReadsService) pageLifecycleOptions(ctx context.Context, slug, projectValue, pageValue, action string) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, err := service.identity(ctx, projectValue)
	if err != nil {
		return nil, nil, err
	}
	var pageID *uuid.UUID
	if pageValue != "" {
		parsed, parseErr := uuid.Parse(pageValue)
		if parseErr != nil {
			return nil, nil, pageReadsNotFound("The required object does not exist.")
		}
		pageID = &parsed
	}
	lifecycle, ok := service.repository.(pageLifecycleRepository)
	if !ok {
		return nil, nil, pageReadsInternalError()
	}
	var page uuid.UUID
	if pageID != nil {
		page = *pageID
	}
	if err := lifecycle.AuthorizeLifecycle(ctx, "options", principal.userID, slug, projectID, page); err != nil {
		return nil, nil, pageReadsOptionsError(err)
	}
	if err := pageLifecycleValidateTimezone(ctx); err != nil {
		return nil, nil, &api.PageReadsBadRequest{Message: "The required key does not exist."}
	}
	metadata, ok := pageLifecycleOptionsMetadata[action]
	if !ok {
		return nil, nil, pageReadsInternalError()
	}
	permissionOperation := map[string]string{
		"collection": "create", "access": "access", "archive": "archive",
		"duplicate": "duplicate", "lock": "lock",
	}[action]
	if permissionOperation != "" && pageLifecycleHasPostAction(metadata) {
		permissionErr := lifecycle.AuthorizeLifecycle(ctx, permissionOperation, principal.userID, slug, projectID, page)
		if permissionErr != nil {
			if !pageLifecyclePermissionDenied(permissionErr) {
				return nil, nil, pageReadsInternalError()
			}
			filtered, filterErr := pageLifecycleWithoutPostAction(metadata)
			if filterErr != nil {
				return nil, nil, pageReadsInternalError()
			}
			metadata = filtered
		}
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, metadata); err != nil {
		return nil, nil, pageReadsInternalError()
	}
	return pageJSONResponse(compact.Bytes())
}

func pageLifecycleHasPostAction(metadata json.RawMessage) bool {
	var body struct {
		Actions map[string]json.RawMessage `json:"actions"`
	}
	return json.Unmarshal(metadata, &body) == nil && body.Actions["POST"] != nil
}

func pageLifecycleWithoutPostAction(metadata json.RawMessage) (json.RawMessage, error) {
	var body map[string]json.RawMessage
	if err := json.Unmarshal(metadata, &body); err != nil {
		return nil, err
	}
	var actions map[string]json.RawMessage
	if err := json.Unmarshal(body["actions"], &actions); err != nil {
		return nil, err
	}
	delete(actions, "POST")
	if len(actions) == 0 {
		delete(body, "actions")
		return json.Marshal(body)
	}
	encodedActions, err := json.Marshal(actions)
	if err != nil {
		return nil, err
	}
	body["actions"] = encodedActions
	return json.Marshal(body)
}

func pageLifecyclePermissionDenied(err error) bool {
	if errors.Is(err, store.ErrForbidden) || errors.Is(err, store.ErrNotFound) {
		return true
	}
	var lifecycle *store.LifecycleError
	if errors.As(err, &lifecycle) && lifecycle.Status == http.StatusForbidden {
		return true
	}
	var source *djangoapi.HTTPError
	return errors.As(err, &source) && source.Status == http.StatusForbidden
}

func (service *pageReadsService) Create(ctx context.Context, payload *api.CreatePayload, body io.ReadCloser) (*api.PageLifecycleResponseMetadata, io.ReadCloser, error) {
	return service.pageLifecycleJSON(ctx, "create", payload.Slug, payload.ProjectID, "", body, http.StatusCreated)
}

func (service *pageReadsService) Update(ctx context.Context, payload *api.UpdatePayload, body io.ReadCloser) (*api.PageLifecycleResponseMetadata, io.ReadCloser, error) {
	return service.pageLifecycleJSON(ctx, "update", payload.Slug, payload.ProjectID, payload.PageID, body, http.StatusOK)
}

func (service *pageReadsService) Delete(ctx context.Context, payload *api.DeletePayload) error {
	_, _, err := service.pageLifecycle(ctx, "delete", payload.Slug, payload.ProjectID, payload.PageID, nil)
	return err
}

func (service *pageReadsService) Access(ctx context.Context, payload *api.AccessPayload, body io.ReadCloser) error {
	_, _, err := service.pageLifecycle(ctx, "access", payload.Slug, payload.ProjectID, payload.PageID, body)
	return err
}

func (service *pageReadsService) Archive(ctx context.Context, payload *api.ArchivePayload) (*api.PageLifecycleResponseMetadata, io.ReadCloser, error) {
	return service.pageLifecycleJSON(ctx, "archive", payload.Slug, payload.ProjectID, payload.PageID, nil, http.StatusOK)
}

func (service *pageReadsService) Unarchive(ctx context.Context, payload *api.UnarchivePayload) error {
	_, _, err := service.pageLifecycle(ctx, "unarchive", payload.Slug, payload.ProjectID, payload.PageID, nil)
	return err
}

func (service *pageReadsService) DescriptionUpdate(ctx context.Context, payload *api.DescriptionUpdatePayload, body io.ReadCloser) (*api.PageLifecycleResponseMetadata, io.ReadCloser, error) {
	return service.pageLifecycleJSON(ctx, "description_update", payload.Slug, payload.ProjectID, payload.PageID, body, http.StatusOK)
}

func (service *pageReadsService) Duplicate(ctx context.Context, payload *api.DuplicatePayload, body io.ReadCloser) (*api.PageLifecycleResponseMetadata, io.ReadCloser, error) {
	return service.pageLifecycleJSON(ctx, "duplicate", payload.Slug, payload.ProjectID, payload.PageID, body, http.StatusCreated)
}

func (service *pageReadsService) Lock(ctx context.Context, payload *api.LockPayload) error {
	_, _, err := service.pageLifecycle(ctx, "lock", payload.Slug, payload.ProjectID, payload.PageID, nil)
	return err
}

func (service *pageReadsService) Unlock(ctx context.Context, payload *api.UnlockPayload) error {
	_, _, err := service.pageLifecycle(ctx, "unlock", payload.Slug, payload.ProjectID, payload.PageID, nil)
	return err
}

// AuthorizeSessionResourceProtocol preserves DRF permission-before-method
// dispatch for session routes whose HTTP verb is not implemented by the
// generated service. Supported writes still authorize in pageLifecycle.
func (service *pageReadsService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, resource SessionProtocolResource) (int, map[string]any) {
	ctx, err := service.APIKeyAuth(ctx, credential, nil)
	if err != nil {
		return http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail}
	}
	principal, projectID, err := service.identity(ctx, resource.ProjectID)
	if err != nil {
		return http.StatusForbidden, map[string]any{"detail": "You do not have permission to perform this action."}
	}
	pageValue := resource.ID
	if pageValue == "" {
		pageValue = resource.PathParameters["page_id"]
	}
	var pageID uuid.UUID
	if pageValue != "" {
		pageID, err = uuid.Parse(pageValue)
		if err != nil {
			return http.StatusForbidden, map[string]any{"detail": "You do not have permission to perform this action."}
		}
	}
	repository, ok := service.repository.(pageLifecycleRepository)
	if !ok {
		return http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"}
	}
	operation := pageLifecyclePermissionOperation(resource.Method, pageID != uuid.Nil())
	if err := repository.AuthorizeLifecycle(ctx, operation, principal.userID, resource.Slug, projectID, pageID); err != nil {
		return pageLifecycleProtocolFailure(err)
	}
	if err := pageLifecycleValidateTimezone(ctx); err != nil {
		return http.StatusBadRequest, map[string]any{"error": "The required key does not exist."}
	}
	return 0, nil
}

func pageLifecyclePermissionOperation(method string, hasPage bool) string {
	if !hasPage {
		switch method {
		case http.MethodPut, http.MethodPatch:
			return "create"
		case http.MethodDelete:
			return "delete_collection"
		default:
			return "unsupported_collection"
		}
	}
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return "options"
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return "unsupported"
	}
}

func pageLifecycleProtocolFailure(err error) (int, map[string]any) {
	var source *djangoapi.HTTPError
	if errors.As(err, &source) {
		return source.Status, source.Body
	}
	var lifecycle *store.LifecycleError
	if errors.As(err, &lifecycle) {
		if body, ok := lifecycle.Body.(map[string]any); ok {
			return lifecycle.Status, body
		}
		return lifecycle.Status, map[string]any{"detail": lifecycle.Body}
	}
	if errors.Is(err, store.ErrForbidden) || errors.Is(err, store.ErrNotFound) {
		return http.StatusForbidden, map[string]any{"detail": "You do not have permission to perform this action."}
	}
	return http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"}
}

func (service *pageReadsService) pageLifecycleJSON(ctx context.Context, operation, slug, projectValue, pageValue string, body io.ReadCloser, status int) (*api.PageLifecycleResponseMetadata, io.ReadCloser, error) {
	resultStatus, result, err := service.pageLifecycle(ctx, operation, slug, projectValue, pageValue, body)
	if err != nil {
		return nil, nil, err
	}
	if resultStatus != status {
		return nil, nil, pageLifecycleError(http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"})
	}
	return &api.PageLifecycleResponseMetadata{ContentType: pageJSONContentType}, rawJSONReadCloser(result), nil
}

func (service *pageReadsService) pageLifecycle(ctx context.Context, operation, slug, projectValue, pageValue string, body io.ReadCloser) (int, json.RawMessage, error) {
	principal, projectID, err := service.identity(ctx, projectValue)
	if err != nil {
		return 0, nil, err
	}
	pageID := uuid.Nil()
	if pageValue != "" {
		pageID, err = uuid.Parse(pageValue)
		if err != nil {
			return 0, nil, pageReadsNotFound("The required object does not exist.")
		}
	}
	repository, ok := service.repository.(pageLifecycleRepository)
	if !ok {
		return 0, nil, pageLifecycleError(http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"})
	}
	if err := repository.AuthorizeLifecycle(ctx, operation, principal.userID, slug, projectID, pageID); err != nil {
		return 0, nil, pageLifecycleStoreError(err)
	}
	if err := pageLifecycleValidateTimezone(ctx); err != nil {
		return 0, nil, err
	}
	if err := repository.ValidateLifecycleHandler(ctx, operation, principal.userID, slug, projectID, pageID); err != nil {
		return 0, nil, pageLifecycleStoreError(err)
	}
	input := djangoinput.Input{Fields: map[string]any{}, Surrogates: map[string][]string{}}
	if operation == "duplicate" {
		if body != nil {
			_ = body.Close()
		}
	} else if body != nil {
		defer body.Close()
		contentType := "application/json"
		if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok && metadata.Headers().Get("Content-Type") != "" {
			contentType = metadata.Headers().Get("Content-Type")
		}
		parsedInput, parseErr := djangoinput.Parse(body, contentType)
		if parseErr != nil {
			return 0, nil, pageLifecycleStoreError(parseErr)
		}
		if parsedInput.Kind != "" || parsedInput.Null {
			return 0, nil, pageLifecycleError(http.StatusBadRequest, map[string]any{"non_field_errors": []string{"Invalid data. Expected a dictionary, but got " + strings.ToLower(parsedInput.Kind) + "."}})
		}
		input = parsedInput
	}
	status, result, err := repository.Lifecycle(ctx, operation, principal.userID, slug, projectID, pageID, principal.timezone, input)
	if err != nil {
		return 0, nil, pageLifecycleStoreError(err)
	}
	return status, result, nil
}

func pageLifecycleValidateTimezone(ctx context.Context) error {
	timezone, ok := ReadServiceSessionTimezone(ctx)
	if !ok {
		return pageLifecycleError(http.StatusBadRequest, map[string]any{"error": "The required key does not exist."})
	}
	if err := djangosession.ValidateUserTimezone(timezone); err != nil {
		return pageLifecycleError(http.StatusBadRequest, map[string]any{"error": "The required key does not exist."})
	}
	return nil
}

func pageLifecycleStoreError(err error) error {
	var raw *store.LifecycleError
	if errors.As(err, &raw) {
		return pageLifecycleError(raw.Status, raw.Body)
	}
	var source *djangoapi.HTTPError
	if errors.As(err, &source) {
		return pageLifecycleError(source.Status, source.Body)
	}
	return pageLifecycleError(http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"})
}

func pageLifecycleError(status int, body any) error {
	if status == http.StatusUnauthorized {
		return &api.PageReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
	encoded, err := loom.JSONValueFrom(body)
	if err != nil {
		return fmt.Errorf("encode page lifecycle error: %w", err)
	}
	return &api.PageLifecycleError{Kind: fmt.Sprintf("status_%d", status), Body: encoded}
}
