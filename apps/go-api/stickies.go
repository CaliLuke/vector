package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"math"
	"strconv"
	"strings"
	"time"
	"uuid"

	stickyapi "github.com/CaliLuke/go-plane/gen/stickies"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	stickystore "github.com/CaliLuke/go-plane/internal/stickies"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

const (
	unauthenticatedDetail = "Authentication credentials were not provided."
	forbiddenDetail       = "You don't have the required permissions."
	notFoundDetail        = "No Sticky matches the given query."
)

type stickySessionValidator interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type stickyRepository interface {
	ResolveWorkspace(context.Context, string, stickystore.UserID) (stickystore.WorkspaceID, error)
	List(context.Context, stickystore.Scope, stickystore.ListOptions) (stickystore.StickyPage, error)
	Create(context.Context, stickystore.StickyID, stickystore.Scope, stickystore.CreateStickyInput) (stickystore.Sticky, error)
	Retrieve(context.Context, stickystore.StickyID, stickystore.Scope) (stickystore.Sticky, error)
	Patch(context.Context, stickystore.StickyID, stickystore.Scope, stickystore.PatchStickyInput) (stickystore.Sticky, error)
	Delete(context.Context, stickystore.StickyID, stickystore.Scope) error
}

type stickyPrincipalKey struct{}

// StickyWorkspaceAuthorizer checks whether the authenticated principal belongs to a workspace.
type StickyWorkspaceAuthorizer interface {
	AuthorizeStickyWorkspace(context.Context, string) error
}
type stickyService struct {
	repository stickyRepository
	sessions   stickySessionValidator
}

// NewStickies returns the reviewed frontend Sticky service.
func NewStickies(repository stickyRepository, sessions stickySessionValidator) (stickyapi.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("Sticky repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("Django session validator is required")
	}
	return &stickyService{repository: repository, sessions: sessions}, nil
}

// APIKeyAuth validates Loom's transport-owned session-id cookie without
// changing Django session state.
func (service *stickyService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	if _, ok := ctx.Value(stickyPrincipalKey{}).(stickystore.UserID); ok {
		return ctx, nil
	}
	principal, err := service.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &stickyapi.StickyUnauthenticatedError{Detail: unauthenticatedDetail}
	}
	identifier, err := uuid.Parse(principal.UserID)
	if err != nil {
		return ctx, &stickyapi.StickyUnauthenticatedError{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, stickyPrincipalKey{}, stickystore.UserID(identifier)), nil
}

func (service *stickyService) AuthorizeStickyWorkspace(ctx context.Context, slug string) error {
	_, err := service.scope(ctx, slug, stickystore.OperationList)
	return err
}
func (service *stickyService) List(ctx context.Context, payload *stickyapi.ListPayload) (*stickyapi.StickyPage, error) {
	scope, err := service.scope(ctx, payload.Slug, stickystore.OperationList)
	if err != nil {
		return nil, err
	}
	limit := 20
	if payload.PerPage != nil {
		limit = *payload.PerPage
	}
	if limit > 1000 {
		return nil, invalidPagination("Invalid per_page value. Cannot exceed 1000.")
	}
	cursor, err := parseStickyCursor(payload.Cursor, limit)
	if err != nil {
		return nil, invalidPagination("Invalid cursor parameter.")
	}
	query := ""
	if payload.Query != nil {
		query = *payload.Query
	}
	page, err := service.repository.List(ctx, scope, stickystore.ListOptions{Query: query, Cursor: cursor, Limit: limit})
	if err != nil {
		if errors.Is(err, stickystore.ErrInvalidPagination) {
			return nil, invalidPagination("Invalid cursor parameter.")
		}
		return nil, err
	}
	results := make([]*stickyapi.StickyResponse, 0, len(page.Stickies))
	for _, sticky := range page.Stickies {
		response, conversionErr := stickyResponse(sticky)
		if conversionErr != nil {
			return nil, conversionErr
		}
		results = append(results, response)
	}
	totalPages := 0
	if limit > 0 {
		totalPages = int(math.Ceil(float64(page.TotalCount) / float64(limit)))
	}
	return &stickyapi.StickyPage{
		GroupedBy: nil, SubGroupedBy: nil, TotalCount: page.TotalCount,
		NextCursor:      fmt.Sprintf("%d:%d:0", limit, page.Page+1),
		PrevCursor:      fmt.Sprintf("%d:%d:1", limit, page.Page-1),
		NextPageResults: page.HasNext, PrevPageResults: page.HasPrevious,
		Count: len(results), TotalPages: totalPages, TotalResults: page.TotalCount,
		ExtraStats: nil, Results: results,
	}, nil
}

func (service *stickyService) Create(ctx context.Context, payload *stickyapi.CreatePayload) (*stickyapi.StickyResponse, error) {
	scope, err := service.scope(ctx, payload.Slug, stickystore.OperationCreate)
	if err != nil {
		return nil, err
	}
	description, err := nullableJSONValue(payload.Description, []byte("{}"))
	if err != nil {
		return nil, validationError("description", "Value must be valid JSON.")
	}
	logo, err := nullableJSONValue(payload.LogoProps, []byte("{}"))
	if err != nil {
		return nil, validationError("logo_props", "Value must be valid JSON.")
	}
	html, stripped, err := stickyHTML(payload.DescriptionHTML, true)
	if err != nil {
		return nil, err
	}
	var deletedAt *time.Time
	if value, present := payload.DeletedAt.Value(); present {
		parsed, parseErr := time.Parse(time.RFC3339, value)
		if parseErr != nil {
			return nil, validationError("deleted_at", "Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z].")
		}
		deletedAt = &parsed
	}
	created, err := service.repository.Create(ctx, stickystore.StickyID(uuid.New()), scope, stickystore.CreateStickyInput{
		Name: nullableStringValue(payload.Name), Description: description, DescriptionHTML: html,
		DescriptionStripped: stripped, LogoProps: logo, Color: nullableStringValue(payload.Color),
		BackgroundColor: nullableStringValue(payload.BackgroundColor), DeletedAt: deletedAt, SortOrder: payload.SortOrder,
	})
	if err != nil {
		return nil, mapRepositoryError(err, false)
	}
	return stickyResponse(created)
}

func (service *stickyService) Retrieve(ctx context.Context, payload *stickyapi.RetrievePayload) (*stickyapi.StickyResponse, error) {
	id, err := stickyID(payload.Pk)
	if err != nil {
		return nil, notFound()
	}
	scope, err := service.scope(ctx, payload.Slug, stickystore.OperationRetrieve)
	if err != nil {
		return nil, notFound()
	}
	sticky, err := service.repository.Retrieve(ctx, id, scope)
	if err != nil {
		return nil, mapRepositoryError(err, true)
	}
	return stickyResponse(sticky)
}

func (service *stickyService) Update(ctx context.Context, payload *stickyapi.UpdatePayload) (*stickyapi.StickyResponse, error) {
	id, err := stickyID(payload.Pk)
	if err != nil {
		return nil, notFound()
	}
	scope, err := service.scope(ctx, payload.Slug, stickystore.OperationUpdate)
	if err != nil {
		return nil, &stickyapi.StickyPermissionError{Message: forbiddenDetail}
	}
	patch := stickystore.PatchStickyInput{SortOrder: payload.SortOrder}
	if payload.Name.Present() {
		patch.Name = nullableStringPatch(payload.Name)
	}
	if payload.Color.Present() {
		patch.Color = nullableStringPatch(payload.Color)
	}
	if payload.BackgroundColor.Present() {
		patch.BackgroundColor = nullableStringPatch(payload.BackgroundColor)
	}
	if payload.CreatedBy.Present() {
		var createdBy *stickystore.UserID
		if value, concrete := payload.CreatedBy.Value(); concrete {
			identifier, parseErr := uuid.Parse(value)
			if parseErr != nil {
				return nil, validationError("created_by", "Must be a valid UUID.")
			}
			value := stickystore.UserID(identifier)
			createdBy = &value
		}
		patch.CreatedByID = &createdBy
	}
	if payload.DeletedAt.Present() {
		var deletedAt *time.Time
		if value, concrete := payload.DeletedAt.Value(); concrete {
			parsed, parseErr := time.Parse(time.RFC3339, value)
			if parseErr != nil {
				return nil, validationError("deleted_at", "Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z].")
			}
			deletedAt = &parsed
		}
		patch.DeletedAt = &deletedAt
	}
	if len(payload.Description) > 0 {
		description, marshalErr := nullableJSONValue(payload.Description, nil)
		if marshalErr != nil {
			return nil, validationError("description", "Value must be valid JSON.")
		}
		patch.Description = &description
	}
	if len(payload.LogoProps) > 0 {
		logo, marshalErr := nullableJSONValue(payload.LogoProps, nil)
		if marshalErr != nil {
			return nil, validationError("logo_props", "Value must be valid JSON.")
		}
		patch.LogoProps = &logo
	}
	if payload.DescriptionHTML != nil {
		html, stripped, htmlErr := stickyHTML(payload.DescriptionHTML, false)
		if htmlErr != nil {
			return nil, htmlErr
		}
		patch.DescriptionHTML = html
		patch.DescriptionStripped = &stripped
	}
	sticky, err := service.repository.Patch(ctx, id, scope, patch)
	if err != nil {
		return nil, mapRepositoryError(err, true)
	}
	return stickyResponse(sticky)
}

func (service *stickyService) Delete(ctx context.Context, payload *stickyapi.DeletePayload) error {
	id, err := stickyID(payload.Pk)
	if err != nil {
		return notFound()
	}
	scope, err := service.scope(ctx, payload.Slug, stickystore.OperationDelete)
	if err != nil {
		return &stickyapi.StickyPermissionError{Message: forbiddenDetail}
	}
	return mapRepositoryError(service.repository.Delete(ctx, id, scope), true)
}

func (service *stickyService) scope(ctx context.Context, slug string, operation stickystore.Operation) (stickystore.Scope, error) {
	userID, ok := ctx.Value(stickyPrincipalKey{}).(stickystore.UserID)
	if !ok {
		return stickystore.Scope{}, &stickyapi.StickyUnauthenticatedError{Detail: unauthenticatedDetail}
	}
	workspaceID, err := service.repository.ResolveWorkspace(ctx, slug, userID)
	if err != nil {
		if errors.Is(err, stickystore.ErrWorkspaceAccessDenied) {
			return stickystore.Scope{}, &stickyapi.StickyPermissionError{Message: forbiddenDetail}
		}
		return stickystore.Scope{}, err
	}
	scope := stickystore.Scope{Operation: operation, WorkspaceID: workspaceID, UserID: userID, Ownership: stickystore.RequireOwner}
	if operation == stickystore.OperationCreate {
		scope.Ownership = stickystore.AssignOwner
		scope.AssignCreator = true
	}
	if operation == stickystore.OperationUpdate || operation == stickystore.OperationDelete {
		scope.RequireCreator = true
	}
	return scope, nil
}

func parseStickyCursor(raw *string, limit int) (*stickystore.StickyCursor, error) {
	if raw == nil {
		return &stickystore.StickyCursor{Value: float64(limit)}, nil
	}
	parts := strings.Split(*raw, ":")
	if len(parts) != 3 {
		return nil, errors.New("invalid cursor")
	}
	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return nil, err
	}
	page, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}
	previous, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, err
	}
	return &stickystore.StickyCursor{Value: value, Page: page, IsPrevious: previous != 0}, nil
}

func stickyHTML(value *string, create bool) (*string, *string, error) {
	if value == nil {
		if !create {
			return nil, nil, nil
		}
		fallback, stripped := "<p></p>", ""
		return &fallback, &stripped, nil
	}
	clean, hasClean, err := content.ValidateHTML(*value)
	if err != nil {
		return nil, nil, validationError("error", "html content is not valid")
	}
	if hasClean {
		*value = clean
	}
	if *value == "" {
		return value, nil, nil
	}
	stripped := content.StripTags(*value)
	return value, &stripped, nil
}

// JSONValue marshals a request value or returns a copied fallback.
func JSONValue(value any, fallback []byte) ([]byte, error) {
	if value == nil {
		return append([]byte(nil), fallback...), nil
	}
	return json.Marshal(value)
}

func nullableJSONValue(value loom.JSONValue, fallback []byte) ([]byte, error) {
	if len(value) == 0 {
		return append([]byte(nil), fallback...), nil
	}
	return pythoncompat.NormalizeJSONNumbers(value)
}

func nullableStringValue(value loom.Nullable[string]) *string {
	concrete, ok := value.Value()
	if !ok {
		return nil
	}
	return &concrete
}

func nullableStringPatch(value loom.Nullable[string]) **string {
	concrete := nullableStringValue(value)
	return &concrete
}

func stickyResponse(sticky stickystore.Sticky) (*stickyapi.StickyResponse, error) {
	description, err := decodeJSON(sticky.Description)
	if err != nil {
		return nil, fmt.Errorf("decode Sticky description: %w", err)
	}
	logo, err := decodeJSON(sticky.LogoProps)
	if err != nil {
		return nil, fmt.Errorf("decode Sticky logo properties: %w", err)
	}
	var createdBy, updatedBy, name, color, background, stripped *string
	binary := loom.NullValue[[]byte]()
	if sticky.CreatedByID != nil {
		value := uuid.UUID(*sticky.CreatedByID).String()
		createdBy = &value
	}
	if sticky.UpdatedByID != nil {
		value := uuid.UUID(*sticky.UpdatedByID).String()
		updatedBy = &value
	}
	if sticky.Name != nil {
		value := *sticky.Name
		name = &value
	}
	if sticky.Color != nil {
		value := *sticky.Color
		color = &value
	}
	if sticky.BackgroundColor != nil {
		value := *sticky.BackgroundColor
		background = &value
	}
	if sticky.DescriptionStripped != nil {
		value := *sticky.DescriptionStripped
		stripped = &value
	}
	if sticky.DescriptionBinary != nil {
		binary = loom.NullableValue(append([]byte(nil), sticky.DescriptionBinary...))
	}
	return &stickyapi.StickyResponse{
		ID: uuid.UUID(sticky.ID).String(), Workspace: uuid.UUID(sticky.WorkspaceID).String(), Owner: uuid.UUID(sticky.OwnerID).String(),
		CreatedBy: createdBy, UpdatedBy: updatedBy, CreatedAt: formatDjangoTime(sticky.CreatedAt), UpdatedAt: formatDjangoTime(sticky.UpdatedAt),
		DeletedAt: formatOptionalDjangoTime(sticky.DeletedAt), Name: name, Color: color, BackgroundColor: background, Description: description,
		DescriptionHTML: sticky.DescriptionHTML, DescriptionStripped: stripped, DescriptionBinary: binary,
		LogoProps: logo, SortOrder: sticky.SortOrder,
	}, nil
}

func decodeJSON(value []byte) (loom.JSONValue, error) {
	if len(value) == 0 {
		return loom.JSONValue("null"), nil
	}
	var decoded loom.JSONValue
	if err := json.Unmarshal(value, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func formatDjangoTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
func formatOptionalDjangoTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := formatDjangoTime(*value)
	return &formatted
}
func stickyID(value string) (stickystore.StickyID, error) {
	id, err := uuid.Parse(value)
	return stickystore.StickyID(id), err
}
func invalidPagination(detail string) error {
	return &stickyapi.StickyInvalidPaginationError{Detail: detail}
}
func validationError(field, detail string) error {
	return stickyapi.StickyValidationErrors{field: []string{detail}}
}
func notFound() error { return &stickyapi.StickyNotFoundError{Detail: notFoundDetail} }

func mapRepositoryError(err error, hideMembership bool) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, stickystore.ErrStickyForbidden) {
		return &stickyapi.StickyPermissionError{Message: forbiddenDetail}
	}
	if errors.Is(err, stickystore.ErrStickyNotFound) || hideMembership && errors.Is(err, stickystore.ErrWorkspaceAccessDenied) {
		return notFound()
	}
	if errors.Is(err, stickystore.ErrWorkspaceAccessDenied) {
		return &stickyapi.StickyPermissionError{Message: forbiddenDetail}
	}
	return err
}
