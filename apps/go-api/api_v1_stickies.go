package systemapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"uuid"

	api "github.com/CaliLuke/go-plane/apiv1stickies/gen/api_v1_stickies"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	stickystore "github.com/CaliLuke/go-plane/internal/stickies"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type apiV1StickiesRepository interface {
	ResolveAPIWorkspace(context.Context, string, stickystore.UserID) (stickystore.WorkspaceID, error)
	UserExists(context.Context, stickystore.UserID) (bool, error)
	List(context.Context, stickystore.Scope, stickystore.ListOptions) (stickystore.StickyPage, error)
	Create(context.Context, stickystore.StickyID, stickystore.Scope, stickystore.CreateStickyInput) (stickystore.Sticky, error)
	Retrieve(context.Context, stickystore.StickyID, stickystore.Scope) (stickystore.Sticky, error)
	PatchOwned(context.Context, stickystore.StickyID, stickystore.Scope, stickystore.PatchStickyInput) (stickystore.Sticky, error)
	DeleteOwned(context.Context, stickystore.StickyID, stickystore.Scope) error
}

type apiV1StickiesKeys interface {
	Validate(context.Context, string) (apikey.Principal, error)
}

type apiV1StickiesEffects interface {
	SoftDelete(context.Context, string, string, string) error
}

type apiV1StickiesThrottler interface {
	Allow(context.Context, string) (apikey.Rate, error)
}

type apiV1StickiesPrincipal struct {
	UserID   stickystore.UserID
	Timezone string
	Key      string
}

type apiV1StickiesPrincipalKey struct{}

type apiV1StickiesService struct {
	repository apiV1StickiesRepository
	keys       apiV1StickiesKeys
	throttle   apiV1StickiesThrottler
	effects    apiV1StickiesEffects
}

func NewAPIV1Stickies(repository apiV1StickiesRepository, keys apiV1StickiesKeys, throttle apiV1StickiesThrottler, effects apiV1StickiesEffects) (api.Service, error) {
	if repository == nil || keys == nil || throttle == nil || effects == nil {
		return nil, fmt.Errorf("API v1 Sticky repository, API-key validator, throttle and effects publisher are required")
	}
	return &apiV1StickiesService{repository: repository, keys: keys, throttle: throttle, effects: effects}, nil
}

func (service *apiV1StickiesService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	principal, err := service.keys.Validate(ctx, key)
	if err != nil {
		if key == "" {
			if effects := apikey.EffectsFromContext(ctx); effects != nil {
				effects.Authenticated = false
			}
			return ctx, apiV1StickyError(http.StatusUnauthorized, map[string]any{"detail": "Authentication credentials were not provided."})
		}
		if errors.Is(err, apikey.ErrInvalid) {
			if effects := apikey.EffectsFromContext(ctx); effects != nil {
				effects.Authenticated = false
			}
			return ctx, apiV1StickyError(http.StatusForbidden, map[string]any{"detail": "Given API token is not valid"})
		}
		return ctx, apiV1StickyStickyInternalError()
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Authenticated = true
	}
	zone := principal.Timezone
	if zone == "" {
		zone = "UTC"
	}
	return context.WithValue(ctx, apiV1StickiesPrincipalKey{}, apiV1StickiesPrincipal{UserID: stickystore.UserID(principal.UserID), Timezone: zone, Key: key}), nil
}

func (service *apiV1StickiesService) List(ctx context.Context, payload *api.ListPayload) (*api.APIV1StickyRawResponseMetadata, io.ReadCloser, error) {
	scope, err := service.scope(ctx, payload.Slug, stickystore.OperationList)
	if err != nil {
		return nil, nil, err
	}
	if throttleRequestErr := service.throttleRequest(ctx); throttleRequestErr != nil {
		return nil, nil, throttleRequestErr
	}
	params := map[string]string{}
	present := map[string]bool{}
	limit := 20
	if payload.PerPage != nil {
		params["per_page"], present["per_page"] = *payload.PerPage, true
		parsed, parseErr := djangoapi.PageSizeWithMinimum(djangoapi.Query{Params: params, Present: present}, 1)
		if parseErr != nil {
			return nil, nil, apiV1StickyMapError(parseErr)
		}
		limit = parsed
	}
	if limit > 1000 {
		return nil, nil, apiV1StickyError(http.StatusBadRequest, map[string]any{"detail": "Invalid per_page value. Cannot exceed 1000."})
	}
	if payload.Cursor != nil {
		params["cursor"], present["cursor"] = *payload.Cursor, true
	}
	page, err := djangoapi.CursorPage(djangoapi.Query{Params: params, Present: present}, limit)
	if err != nil {
		return nil, nil, apiV1StickyMapError(err)
	}
	if limit <= 0 || page > int(^uint(0)>>1)/limit {
		return nil, nil, apiV1StickyError(http.StatusBadRequest, map[string]any{"detail": "Error in parsing"})
	}
	query := ""
	if payload.Query != nil {
		query = *payload.Query
	}
	result, err := service.repository.List(ctx, scope, stickystore.ListOptions{
		Query: query, Cursor: &stickystore.StickyCursor{Value: float64(limit), Page: page}, Limit: limit, CreatedAtOrder: true, LiteralQuery: true,
	})
	if err != nil {
		return nil, nil, apiV1StickyMapError(err)
	}
	rows := make([]any, 0, len(result.Stickies))
	for _, sticky := range result.Stickies {
		rows = append(rows, apiV1StickyResponse(sticky, apiV1StickyTimezone(ctx)))
	}
	envelope := djangoapi.Envelope(rows, result.TotalCount, limit, result.Page)
	envelope["next_page_results"] = result.HasNext
	return apiV1StickyJSON(envelope)
}

func (service *apiV1StickiesService) Create(ctx context.Context, payload *api.CreatePayload, body io.ReadCloser) (*api.APIV1StickyRawResponseMetadata, io.ReadCloser, error) {
	defer closeStickyBody(body)
	scope, err := service.scope(ctx, payload.Slug, stickystore.OperationCreate)
	if err != nil {
		return nil, nil, err
	}
	if throttleRequestErr := service.throttleRequest(ctx); throttleRequestErr != nil {
		return nil, nil, throttleRequestErr
	}
	input, err := apiV1StickyParseBody(ctx, body)
	if err != nil {
		return nil, nil, apiV1StickyMapError(err)
	}
	if apiV1StickyRequireObjectErr := apiV1StickyRequireObject(input); apiV1StickyRequireObjectErr != nil {
		return nil, nil, apiV1StickyRequireObjectErr
	}
	values, failures := apiV1StickyCreateFields(input, apiV1StickyTimezoneLocation(ctx))
	if validateAuditUsersErr := service.validateAuditUsers(ctx, input, failures); validateAuditUsersErr != nil {
		return nil, nil, apiV1StickyMapError(validateAuditUsersErr)
	}
	if len(failures) != 0 {
		return nil, nil, apiV1StickyError(http.StatusBadRequest, failures)
	}
	created, err := service.repository.Create(ctx, stickystore.StickyID(uuid.New()), scope, values)
	if err != nil {
		return nil, nil, apiV1StickyMapError(err)
	}
	return apiV1StickyJSON(apiV1StickyResponse(created, apiV1StickyTimezone(ctx)))
}

func (service *apiV1StickiesService) ListOptions(ctx context.Context, payload *api.ListOptionsPayload) (*api.APIV1StickyRawResponseMetadata, io.ReadCloser, error) {
	if _, err := service.scope(ctx, payload.Slug, stickystore.OperationList); err != nil {
		return nil, nil, err
	}
	if err := service.throttleRequest(ctx); err != nil {
		return nil, nil, err
	}
	return apiV1StickyJSON(apiV1StickyOptions("Sticky List", "POST"))
}

func (service *apiV1StickiesService) DetailOptions(ctx context.Context, payload *api.DetailOptionsPayload) (*api.APIV1StickyRawResponseMetadata, io.ReadCloser, error) {
	scope, err := service.scope(ctx, payload.Slug, stickystore.OperationRetrieve)
	if err != nil {
		return nil, nil, err
	}
	if throttleRequestErr := service.throttleRequest(ctx); throttleRequestErr != nil {
		return nil, nil, throttleRequestErr
	}
	id, err := uuid.Parse(payload.Pk)
	if err != nil {
		return apiV1StickyJSON(apiV1StickyOptionsWithoutActions("Sticky Instance"))
	}
	if _, err := service.repository.Retrieve(ctx, stickystore.StickyID(id), scope); err != nil {
		if errors.Is(err, stickystore.ErrStickyNotFound) {
			return apiV1StickyJSON(apiV1StickyOptionsWithoutActions("Sticky Instance"))
		}
		return nil, nil, apiV1StickyMapError(err)
	}
	return apiV1StickyJSON(apiV1StickyOptions("Sticky Instance", "PUT"))
}

func (service *apiV1StickiesService) Retrieve(ctx context.Context, payload *api.RetrievePayload) (*api.APIV1StickyRawResponseMetadata, io.ReadCloser, error) {
	scope, err := service.scope(ctx, payload.Slug, stickystore.OperationRetrieve)
	if err != nil {
		return nil, nil, err
	}
	if throttleRequestErr := service.throttleRequest(ctx); throttleRequestErr != nil {
		return nil, nil, throttleRequestErr
	}
	id, err := uuid.Parse(payload.Pk)
	if err != nil {
		return nil, nil, apiV1StickyMalformedNotFound()
	}
	sticky, err := service.repository.Retrieve(ctx, stickystore.StickyID(id), scope)
	if err != nil {
		return nil, nil, apiV1StickyMapError(err)
	}
	return apiV1StickyJSON(apiV1StickyResponse(sticky, apiV1StickyTimezone(ctx)))
}

func (service *apiV1StickiesService) Patch(ctx context.Context, payload *api.PatchPayload, body io.ReadCloser) (*api.APIV1StickyRawResponseMetadata, io.ReadCloser, error) {
	return service.update(ctx, payload.Slug, payload.Pk, body, false)
}

func (service *apiV1StickiesService) Update(ctx context.Context, payload *api.UpdatePayload, body io.ReadCloser) (*api.APIV1StickyRawResponseMetadata, io.ReadCloser, error) {
	return service.update(ctx, payload.Slug, payload.Pk, body, true)
}

func (service *apiV1StickiesService) update(ctx context.Context, slug, rawID string, body io.ReadCloser, full bool) (*api.APIV1StickyRawResponseMetadata, io.ReadCloser, error) {
	defer closeStickyBody(body)
	scope, err := service.scope(ctx, slug, stickystore.OperationUpdate)
	if err != nil {
		return nil, nil, err
	}
	if throttleRequestErr := service.throttleRequest(ctx); throttleRequestErr != nil {
		return nil, nil, throttleRequestErr
	}
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, nil, apiV1StickyMalformedNotFound()
	}
	readScope := scope
	readScope.Operation = stickystore.OperationRetrieve
	current, err := service.repository.Retrieve(ctx, stickystore.StickyID(id), readScope)
	if err != nil {
		return nil, nil, apiV1StickyMapError(err)
	}
	input, err := apiV1StickyParseBody(ctx, body)
	if err != nil {
		return nil, nil, apiV1StickyMapError(err)
	}
	if apiV1StickyRequireObjectErr := apiV1StickyRequireObject(input); apiV1StickyRequireObjectErr != nil {
		return nil, nil, apiV1StickyRequireObjectErr
	}
	patch, failures := apiV1StickyPatchFields(input, current, full, apiV1StickyTimezoneLocation(ctx))
	if validateAuditUsersErr := service.validateAuditUsers(ctx, input, failures); validateAuditUsersErr != nil {
		return nil, nil, apiV1StickyMapError(validateAuditUsersErr)
	}
	if len(failures) != 0 {
		return nil, nil, apiV1StickyError(http.StatusBadRequest, failures)
	}
	updated, err := service.repository.PatchOwned(ctx, stickystore.StickyID(id), scope, patch)
	if err != nil {
		return nil, nil, apiV1StickyMapError(err)
	}
	return apiV1StickyJSON(apiV1StickyResponse(updated, apiV1StickyTimezone(ctx)))
}

func (service *apiV1StickiesService) validateAuditUsers(ctx context.Context, input djangoinput.Input, failures map[string]any) error {
	for _, field := range []string{"created_by", "updated_by"} {
		raw, present := input.Fields[field]
		if !present || raw == nil {
			continue
		}
		parsed, err := djangoinput.UUID(raw)
		if err != nil {
			failures[field] = []string{"Must be a valid UUID."}
			continue
		}
		exists, err := service.repository.UserExists(ctx, stickystore.UserID(parsed))
		if err != nil {
			return err
		}
		if !exists {
			failures[field] = []string{fmt.Sprintf("Invalid pk %q - object does not exist.", uuid.UUID(parsed).String())}
		}
	}
	return nil
}

func (service *apiV1StickiesService) Delete(ctx context.Context, payload *api.DeletePayload) error {
	scope, err := service.scope(ctx, payload.Slug, stickystore.OperationDelete)
	if err != nil {
		return err
	}
	if throttleRequestErr := service.throttleRequest(ctx); throttleRequestErr != nil {
		return throttleRequestErr
	}
	id, err := uuid.Parse(payload.Pk)
	if err != nil {
		return apiV1StickyMalformedNotFound()
	}
	if err := service.repository.DeleteOwned(ctx, stickystore.StickyID(id), scope); err != nil {
		return apiV1StickyMapError(err)
	}
	if err := service.effects.SoftDelete(ctx, "db", "sticky", id.String()); err != nil {
		return apiV1StickyStickyInternalError()
	}
	return nil
}

func (service *apiV1StickiesService) throttleRequest(ctx context.Context) error {
	principal, ok := ctx.Value(apiV1StickiesPrincipalKey{}).(apiV1StickiesPrincipal)
	if !ok {
		return apiV1StickyError(http.StatusUnauthorized, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	rate, err := service.throttle.Allow(ctx, principal.Key)
	if err != nil {
		return apiV1StickyStickyInternalError()
	}
	if !rate.Allowed {
		if effects := apikey.EffectsFromContext(ctx); effects != nil {
			effects.Rate = &rate
		}
		return apiV1StickyError(http.StatusTooManyRequests, map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))})
	}
	return nil
}

func (service *apiV1StickiesService) scope(ctx context.Context, slug string, operation stickystore.Operation) (stickystore.Scope, error) {
	principal, ok := ctx.Value(apiV1StickiesPrincipalKey{}).(apiV1StickiesPrincipal)
	if !ok {
		return stickystore.Scope{}, apiV1StickyError(http.StatusUnauthorized, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	workspace, err := service.repository.ResolveAPIWorkspace(ctx, slug, principal.UserID)
	if errors.Is(err, stickystore.ErrWorkspaceAccessDenied) {
		return stickystore.Scope{}, apiV1StickyError(http.StatusForbidden, map[string]any{"detail": "You do not have permission to perform this action."})
	}
	if errors.Is(err, stickystore.ErrStickyWorkspaceDeleted) {
		if operation == stickystore.OperationCreate {
			return stickystore.Scope{}, apiV1StickyWorkspaceNotFound()
		}
		// Sticky's Django queryset filters by the requested workspace slug and
		// owner, but does not hide rows when their workspace is soft-deleted.
		err = nil
	}
	if err != nil {
		return stickystore.Scope{}, apiV1StickyStickyInternalError()
	}
	scope := stickystore.Scope{Operation: operation, WorkspaceID: workspace, UserID: principal.UserID, Ownership: stickystore.RequireOwner}
	if operation == stickystore.OperationCreate {
		scope.Ownership = stickystore.AssignOwner
		scope.AssignCreator = true
	}
	return scope, nil
}

func apiV1StickyParseBody(ctx context.Context, body io.Reader) (djangoinput.Input, error) {
	contentType := ""
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		contentType = metadata.Headers().Get("Content-Type")
	}
	return djangoinput.Parse(body, contentType)
}

func apiV1StickyRequireObject(input djangoinput.Input) error {
	if input.Kind != "" || input.Null {
		if input.Null {
			return apiV1StickyError(http.StatusBadRequest, map[string]any{"non_field_errors": []string{"No data provided"}})
		}
		kind := strings.ToLower(input.Kind)
		return apiV1StickyError(http.StatusBadRequest, map[string]any{
			"non_field_errors": []string{"Invalid data. Expected a dictionary, but got " + kind + "."},
		})
	}
	return nil
}

func apiV1StickyCreateFields(input djangoinput.Input, location *time.Location) (stickystore.CreateStickyInput, map[string]any) {
	values := stickystore.CreateStickyInput{}
	failures := map[string]any{}
	if raw, present := input.Fields["name"]; present {
		if raw == nil {
			values.Name = nil
		} else if value, fieldErrors := djangoinput.TextField(raw, true, 0, input.Surrogates["name"]); len(fieldErrors) != 0 {
			failures["name"] = fieldErrors
		} else {
			values.Name = &value
		}
	}
	for field, target := range map[string]**string{"color": &values.Color, "background_color": &values.BackgroundColor} {
		if raw, present := input.Fields[field]; present {
			if raw == nil {
				*target = nil
			} else if value, fieldErrors := djangoinput.TextField(raw, true, 255, input.Surrogates[field]); len(fieldErrors) != 0 {
				failures[field] = fieldErrors
			} else {
				*target = &value
			}
		}
	}
	description, err := apiV1StickyJSONField(input, "description", []byte("{}"))
	if err != nil {
		failures["description"] = []string{"Value must be valid JSON."}
	} else {
		values.Description = description
	}
	logo, err := apiV1StickyJSONField(input, "logo_props", []byte("{}"))
	if err != nil {
		failures["logo_props"] = []string{"Value must be valid JSON."}
	} else {
		values.LogoProps = logo
	}
	html := "<p></p>"
	if raw, present := input.Fields["description_html"]; present {
		value, ok := raw.(string)
		if !ok {
			failures["description_html"] = []string{"Not a valid string."}
		} else {
			html = value
		}
	}
	if _, bad := failures["description_html"]; !bad {
		clean, hasClean, htmlErr := content.ValidateHTML(html)
		if htmlErr != nil {
			failures["error"] = []string{"html content is not valid"}
		} else {
			if hasClean {
				html = clean
			}
			values.DescriptionHTML = &html
			if html == "" {
				values.DescriptionStripped = nil
			} else {
				stripped := content.StripTags(html)
				values.DescriptionStripped = &stripped
			}
		}
	}
	if raw, present := input.Fields["sort_order"]; present {
		value, message := djangoinput.FloatField(raw)
		if message != "" {
			failures["sort_order"] = []string{message}
		} else {
			values.SortOrder = &value
		}
	}
	if raw, present := input.Fields["deleted_at"]; present && raw != nil {
		value, parseErr := apiV1StickyDatetime(raw, location)
		if parseErr != nil {
			failures["deleted_at"] = []string{apiV1StickyDatetimeError}
		} else {
			values.DeletedAt = value
		}
	}
	return values, failures
}

func apiV1StickyPatchFields(input djangoinput.Input, current stickystore.Sticky, _ bool, location *time.Location) (stickystore.PatchStickyInput, map[string]any) {
	patch := stickystore.PatchStickyInput{}
	failures := map[string]any{}
	if raw, present := input.Fields["name"]; present {
		var value *string
		if raw != nil {
			parsed, fieldErrors := djangoinput.TextField(raw, true, 0, input.Surrogates["name"])
			if len(fieldErrors) != 0 {
				failures["name"] = fieldErrors
			} else {
				value = &parsed
			}
		}
		patch.Name = &value
	}
	for field, target := range map[string]***string{"color": &patch.Color, "background_color": &patch.BackgroundColor} {
		if raw, present := input.Fields[field]; present {
			var value *string
			if raw != nil {
				parsed, fieldErrors := djangoinput.TextField(raw, true, 255, input.Surrogates[field])
				if len(fieldErrors) != 0 {
					failures[field] = fieldErrors
				} else {
					value = &parsed
				}
			}
			*target = &value
		}
	}
	if raw, present := input.Fields["description"]; present {
		if raw == nil {
			failures["description"] = []string{"This field may not be null."}
		} else if normalized, err := JSONValue(raw, nil); err != nil {
			failures["description"] = []string{"Value must be valid JSON."}
		} else {
			patch.Description = &normalized
		}
	}
	if raw, present := input.Fields["logo_props"]; present {
		if raw == nil {
			failures["logo_props"] = []string{"This field may not be null."}
		} else if normalized, err := JSONValue(raw, nil); err != nil {
			failures["logo_props"] = []string{"Value must be valid JSON."}
		} else {
			patch.LogoProps = &normalized
		}
	}
	if raw, present := input.Fields["description_html"]; present {
		if raw == nil {
			failures["description_html"] = []string{"This field may not be null."}
		} else if value, ok := raw.(string); !ok {
			failures["description_html"] = []string{"Not a valid string."}
		} else {
			clean, hasClean, err := content.ValidateHTML(value)
			if err != nil {
				failures["error"] = []string{"html content is not valid"}
			} else {
				if hasClean {
					value = clean
				}
				patch.DescriptionHTML = &value
				if value == "" {
					var stripped *string
					patch.DescriptionStripped = &stripped
				} else {
					stripped := content.StripTags(value)
					strippedPtr := &stripped
					patch.DescriptionStripped = &strippedPtr
				}
			}
		}
	}
	if _, present := input.Fields["description_html"]; !present && current.DescriptionHTML == "" {
		var stripped *string
		patch.DescriptionStripped = &stripped
	} else if _, present := input.Fields["description_html"]; !present {
		stripped := content.StripTags(current.DescriptionHTML)
		strippedPtr := &stripped
		patch.DescriptionStripped = &strippedPtr
	}
	if raw, present := input.Fields["deleted_at"]; present {
		var value *time.Time
		if raw != nil {
			parsed, err := apiV1StickyDatetime(raw, location)
			if err != nil {
				failures["deleted_at"] = []string{apiV1StickyDatetimeError}
			} else {
				value = parsed
			}
		}
		patch.DeletedAt = &value
	}
	if raw, present := input.Fields["sort_order"]; present {
		value, message := djangoinput.FloatField(raw)
		if message != "" {
			failures["sort_order"] = []string{message}
		} else {
			patch.SortOrder = &value
		}
	}
	if raw, present := input.Fields["created_by"]; present {
		var value *stickystore.UserID
		if raw != nil {
			parsed, err := djangoinput.UUID(raw)
			if err != nil {
				failures["created_by"] = []string{"Must be a valid UUID."}
			} else {
				owner := stickystore.UserID(parsed)
				value = &owner
			}
		}
		patch.CreatedByID = &value
	}
	return patch, failures
}

const apiV1StickyDatetimeError = "Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z]."

func apiV1StickyOptions(name, action string) map[string]any {
	type field struct {
		Label     string `json:"label"`
		MaxLength int    `json:"max_length,omitempty"`
		ReadOnly  bool   `json:"read_only"`
		Required  bool   `json:"required"`
		Type      string `json:"type"`
	}
	fields := map[string]field{
		"background_color":     {"Background color", 255, false, false, "string"},
		"color":                {"Color", 255, false, false, "string"},
		"created_at":           {"Created At", 0, true, false, "datetime"},
		"created_by":           {"Created By", 0, false, false, "field"},
		"deleted_at":           {"Deleted At", 0, false, false, "datetime"},
		"description":          {"Description", 0, false, false, "field"},
		"description_binary":   {"Description binary", 0, true, false, "field"},
		"description_html":     {"Description html", 0, false, false, "string"},
		"description_stripped": {"Description stripped", 0, false, false, "string"},
		"id":                   {"Id", 0, true, false, "field"},
		"logo_props":           {"Logo props", 0, false, false, "field"},
		"name":                 {"Name", 0, false, false, "string"},
		"owner":                {"Owner", 0, true, false, "field"},
		"sort_order":           {"Sort order", 0, false, false, "float"},
		"updated_at":           {"Last Modified At", 0, true, false, "datetime"},
		"updated_by":           {"Last Modified By", 0, false, false, "field"},
		"workspace":            {"Workspace", 0, true, false, "field"},
	}
	return map[string]any{
		"actions":     map[string]any{action: fields},
		"description": "",
		"name":        name,
		"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		"renders":     []string{"application/json"},
	}
}

func apiV1StickyOptionsWithoutActions(name string) map[string]any {
	metadata := apiV1StickyOptions(name, "PUT")
	delete(metadata, "actions")
	return metadata
}

func apiV1StickyJSONField(input djangoinput.Input, field string, fallback []byte) ([]byte, error) {
	value, present := input.Fields[field]
	if !present {
		return append([]byte(nil), fallback...), nil
	}
	if value == nil {
		return nil, fmt.Errorf("field %s may not be null", field)
	}
	return JSONValue(value, nil)
}

func apiV1StickyDatetime(value any, location *time.Location) (*time.Time, error) {
	text, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("not a datetime string")
	}
	if location == nil {
		location = time.UTC
	}
	parsed, err := pythoncompat.DjangoDateTime(text, location)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func apiV1StickyResponse(sticky stickystore.Sticky, timezone string) map[string]any {
	return map[string]any{
		"id": uuid.UUID(sticky.ID).String(), "workspace": uuid.UUID(sticky.WorkspaceID).String(), "owner": uuid.UUID(sticky.OwnerID).String(),
		"created_by": apiV1StickyUUID(sticky.CreatedByID), "updated_by": apiV1StickyUUID(sticky.UpdatedByID),
		"created_at": apiV1StickyTimeInZone(sticky.CreatedAt, timezone), "updated_at": apiV1StickyTimeInZone(sticky.UpdatedAt, timezone), "deleted_at": apiV1StickyTimePtrInZone(sticky.DeletedAt, timezone),
		"name": apiV1StickyStringPtr(sticky.Name), "color": apiV1StickyStringPtr(sticky.Color), "background_color": apiV1StickyStringPtr(sticky.BackgroundColor),
		"description": apiV1StickyRawJSON(sticky.Description), "description_html": sticky.DescriptionHTML, "description_stripped": apiV1StickyStringPtr(sticky.DescriptionStripped),
		"description_binary": apiV1StickyBinary(sticky.DescriptionBinary), "logo_props": apiV1StickyRawJSON(sticky.LogoProps), "sort_order": sticky.SortOrder,
	}
}

func apiV1StickyUUID(value *stickystore.UserID) any {
	if value == nil {
		return nil
	}
	return uuid.UUID(*value).String()
}

func apiV1StickyTimezone(ctx context.Context) string {
	principal, ok := ctx.Value(apiV1StickiesPrincipalKey{}).(apiV1StickiesPrincipal)
	if !ok || principal.Timezone == "" {
		return "UTC"
	}
	return principal.Timezone
}

func apiV1StickyTimezoneLocation(ctx context.Context) *time.Location {
	location, err := time.LoadLocation(apiV1StickyTimezone(ctx))
	if err != nil {
		return time.UTC
	}
	return location
}

func apiV1StickyTimeInZone(value time.Time, timezone string) string {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		location = time.UTC
	}
	return pythoncompat.DRFDateTimeString(value, location)
}

func apiV1StickyTimePtrInZone(value *time.Time, timezone string) any {
	if value == nil {
		return nil
	}
	return apiV1StickyTimeInZone(*value, timezone)
}

func apiV1StickyStringPtr(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func apiV1StickyBinary(value []byte) any {
	if value == nil {
		return nil
	}
	return base64.StdEncoding.EncodeToString(value)
}

func apiV1StickyRawJSON(value []byte) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage("null")
	}
	return json.RawMessage(append([]byte(nil), value...))
}

func apiV1StickyJSON(value any) (*api.APIV1StickyRawResponseMetadata, io.ReadCloser, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, nil, apiV1StickyStickyInternalError()
	}
	return &api.APIV1StickyRawResponseMetadata{ContentType: "application/json"}, io.NopCloser(bytes.NewReader(data)), nil
}

func apiV1StickyError(status int, body any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		raw = []byte(`{"error":"Something went wrong please try again later"}`)
		status = http.StatusInternalServerError
	}
	return &api.APIV1StickyErrorResponse{Name: "status_" + strconv.Itoa(status), Body: loom.JSONValue(raw)}
}

func apiV1StickyMapError(err error) error {
	if err == nil {
		return nil
	}
	var response *djangoapi.HTTPError
	if errors.As(err, &response) {
		return apiV1StickyError(response.Status, response.Body)
	}
	if errors.Is(err, stickystore.ErrStickyNotFound) || errors.Is(err, stickystore.ErrWorkspaceAccessDenied) {
		return apiV1StickyNotFound()
	}
	return apiV1StickyStickyInternalError()
}

func apiV1StickyNotFound() error {
	return apiV1StickyError(http.StatusNotFound, map[string]any{"detail": "No Sticky matches the given query."})
}

func apiV1StickyMalformedNotFound() error {
	return apiV1StickyError(http.StatusNotFound, map[string]any{"detail": "Not found."})
}

func apiV1StickyWorkspaceNotFound() error {
	return apiV1StickyError(http.StatusNotFound, map[string]any{"error": "The required object does not exist."})
}

func apiV1StickyStickyInternalError() error {
	return apiV1StickyError(http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"})
}

func closeStickyBody(body io.Closer) {
	if body != nil {
		_ = body.Close()
	}
}

var _ api.Service = (*apiV1StickiesService)(nil)
