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

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/page_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	store "github.com/CaliLuke/go-plane/internal/pagereads"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
)

const (
	pageReadsInternalErrorDetail = "Something went wrong please try again later"
	pageDescriptionDisposition   = `attachment; filename="page_description.bin"`
	pageJSONContentType          = "application/json"
	pageSummaryOptionsJSON       = `{"name":"Page","description":"","renders":["application/json"],"parses":["application/json","application/x-www-form-urlencoded","multipart/form-data"]}`
	pageVersionOptionsJSON       = `{"name":"Page Version Endpoint","description":"","renders":["application/json"],"parses":["application/json","application/x-www-form-urlencoded","multipart/form-data"]}`
)

type pageReadsRepository interface {
	Summary(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	List(context.Context, uuid.UUID, string, uuid.UUID, string, string, string) (json.RawMessage, error)
	Detail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, string, string, string) (json.RawMessage, error)
	Description(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) ([]byte, error)
	VersionList(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, string) (json.RawMessage, error)
	VersionDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, uuid.UUID, string) (json.RawMessage, error)
	AuthorizeOptions(context.Context, uuid.UUID, string, uuid.UUID, *uuid.UUID) error
}

type pageLifecycleRepository interface {
	AuthorizeLifecycle(context.Context, string, uuid.UUID, string, uuid.UUID, uuid.UUID) error
	ValidateLifecycleHandler(context.Context, string, uuid.UUID, string, uuid.UUID, uuid.UUID) error
	Lifecycle(context.Context, string, uuid.UUID, string, uuid.UUID, uuid.UUID, string, djangoinput.Input) (int, json.RawMessage, error)
}

type pageReadsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type pageRecentVisits interface {
	Record(context.Context, uuid.UUID, string, string, uuid.UUID, uuid.UUID) error
}

type pageReadsService struct {
	repository   pageReadsRepository
	sessions     pageReadsSessions
	recentVisits pageRecentVisits
}

type pageReadsPrincipalKey struct{}

type pageReadsPrincipal struct {
	userID   uuid.UUID
	timezone string
}

func NewPageReads(
	repository pageReadsRepository,
	sessions pageReadsSessions,
	recentVisits pageRecentVisits,
) (api.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("page reads repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("session validator for Django is required")
	}
	if recentVisits == nil {
		return nil, fmt.Errorf("page recent visits repository is required")
	}
	return &pageReadsService{repository: repository, sessions: sessions, recentVisits: recentVisits}, nil
}

func (service *pageReadsService) APIKeyAuth(
	ctx context.Context,
	key string,
	_ *security.APIKeyScheme,
) (context.Context, error) {
	principalID, timezone, inspected := readServiceSessionPrincipal(ctx)
	if !inspected {
		principal, err := service.sessions.Validate(ctx, key)
		if err != nil {
			return ctx, &api.PageReadsUnauthenticated{Detail: unauthenticatedDetail}
		}
		principalID = principal.UserID
		timezone = principal.UserTimezone
	}
	userID, err := uuid.Parse(principalID)
	if err != nil {
		return ctx, &api.PageReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
	if timezone == "" {
		timezone = "UTC"
	}
	return context.WithValue(ctx, pageReadsPrincipalKey{}, pageReadsPrincipal{userID: userID, timezone: timezone}), nil
}

func (service *pageReadsService) Summary(
	ctx context.Context,
	payload *api.SummaryPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, err := service.identity(ctx, payload.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.Summary(ctx, principal.userID, payload.Slug, projectID)
	if err != nil {
		return nil, nil, pageReadsRepositoryError(err, "The required object does not exist.")
	}
	return pageJSONResponse(data)
}
func (service *pageReadsService) SummaryOptions(
	ctx context.Context,
	payload *api.SummaryOptionsPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, err := service.identity(ctx, payload.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	if err := service.repository.AuthorizeOptions(ctx, principal.userID, payload.Slug, projectID, nil); err != nil {
		return nil, nil, pageReadsOptionsError(err)
	}
	return pageJSONResponse([]byte(pageSummaryOptionsJSON))
}

func (service *pageReadsService) List(
	ctx context.Context,
	payload *api.ListPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, err := service.identity(ctx, payload.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	search := optionalString(payload.Search, "")
	orderBy := pageOrderBy(ctx, payload.OrderBy)
	data, err := service.repository.List(
		ctx, principal.userID, payload.Slug, projectID, search, orderBy, principal.timezone,
	)
	if err != nil {
		return nil, nil, pageReadsRepositoryError(err, "The required object does not exist.")
	}
	return pageJSONResponse(data)
}

func (service *pageReadsService) Detail(
	ctx context.Context,
	payload *api.DetailPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, err := service.identity(ctx, payload.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	pageID, err := uuid.Parse(payload.PageID)
	if err != nil {
		return nil, nil, pageReadsNotFound("Page not found")
	}
	search := optionalString(payload.Search, "")
	orderBy := pageOrderBy(ctx, payload.OrderBy)
	data, err := service.repository.Detail(
		ctx, principal.userID, payload.Slug, projectID, pageID, search, orderBy, principal.timezone,
	)
	if err != nil {
		return nil, nil, pageReadsRepositoryError(err, "Page not found")
	}
	trackVisit := (payload.TrackVisit == nil && !ReadServiceQueryPresent(ctx, "track_visit")) ||
		(payload.TrackVisit != nil && strings.EqualFold(*payload.TrackVisit, "true"))
	if trackVisit {
		_ = service.recentVisits.Record(ctx, principal.userID, payload.Slug, "page", pageID, projectID)
	}
	return pageJSONResponse(data)
}

func (service *pageReadsService) Description(
	ctx context.Context,
	payload *api.DescriptionPayload,
) (*api.PageDescriptionMetadata, *loomhttp.FileResponse, error) {
	principal, projectID, err := service.identity(ctx, payload.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	pageID, err := uuid.Parse(payload.PageID)
	if err != nil {
		return nil, nil, pageReadsNotFound("The required object does not exist.")
	}
	data, err := service.repository.Description(ctx, principal.userID, payload.Slug, projectID, pageID)
	if err != nil {
		return nil, nil, pageReadsRepositoryError(err, "The required object does not exist.")
	}
	metadata := &api.PageDescriptionMetadata{Disposition: pageDescriptionDisposition}
	file := &loomhttp.FileResponse{Name: "page_description.bin", Content: bytes.NewReader(data)}
	return metadata, file, nil
}

func (service *pageReadsService) VersionList(
	ctx context.Context,
	payload *api.VersionListPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, pageID, err := service.pageIdentity(ctx, payload.ProjectID, payload.PageID)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.VersionList(
		ctx, principal.userID, payload.Slug, projectID, pageID, principal.timezone,
	)
	if err != nil {
		return nil, nil, pageReadsRepositoryError(err, "The required object does not exist.")
	}
	return pageJSONResponse(data)
}
func (service *pageReadsService) VersionListOptions(
	ctx context.Context,
	payload *api.VersionListOptionsPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, pageID, err := service.pageIdentity(ctx, payload.ProjectID, payload.PageID)
	if err != nil {
		return nil, nil, pageReadsOptionsIdentityError(err)
	}
	if err := service.repository.AuthorizeOptions(ctx, principal.userID, payload.Slug, projectID, &pageID); err != nil {
		return nil, nil, pageReadsOptionsError(err)
	}
	return pageJSONResponse([]byte(pageVersionOptionsJSON))
}

func (service *pageReadsService) VersionDetail(
	ctx context.Context,
	payload *api.VersionDetailPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, pageID, err := service.pageIdentity(ctx, payload.ProjectID, payload.PageID)
	if err != nil {
		return nil, nil, err
	}
	versionID, err := uuid.Parse(payload.Pk)
	if err != nil {
		return nil, nil, pageReadsNotFound("The required object does not exist.")
	}
	data, err := service.repository.VersionDetail(
		ctx, principal.userID, payload.Slug, projectID, pageID, versionID, principal.timezone,
	)
	if err != nil {
		return nil, nil, pageReadsRepositoryError(err, "The required object does not exist.")
	}
	return pageJSONResponse(data)
}
func (service *pageReadsService) VersionDetailOptions(
	ctx context.Context,
	payload *api.VersionDetailOptionsPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, pageID, err := service.pageIdentity(ctx, payload.ProjectID, payload.PageID)
	if err != nil {
		return nil, nil, pageReadsOptionsIdentityError(err)
	}
	if err := service.repository.AuthorizeOptions(ctx, principal.userID, payload.Slug, projectID, &pageID); err != nil {
		return nil, nil, pageReadsOptionsError(err)
	}
	return pageJSONResponse([]byte(pageVersionOptionsJSON))
}

func (service *pageReadsService) identity(
	ctx context.Context,
	projectValue string,
) (pageReadsPrincipal, uuid.UUID, error) {
	principal, ok := ctx.Value(pageReadsPrincipalKey{}).(pageReadsPrincipal)
	if !ok {
		return pageReadsPrincipal{}, uuid.Nil(), &api.PageReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
	projectID, err := uuid.Parse(projectValue)
	if err != nil {
		return pageReadsPrincipal{}, uuid.Nil(), &api.PageReadsForbidden{Detail: "You do not have permission to perform this action."}
	}
	return principal, projectID, nil
}

func (service *pageReadsService) pageIdentity(
	ctx context.Context,
	projectValue string,
	pageValue string,
) (pageReadsPrincipal, uuid.UUID, uuid.UUID, error) {
	principal, projectID, err := service.identity(ctx, projectValue)
	if err != nil {
		return pageReadsPrincipal{}, uuid.Nil(), uuid.Nil(), err
	}
	pageID, err := uuid.Parse(pageValue)
	if err != nil {
		return pageReadsPrincipal{}, uuid.Nil(), uuid.Nil(), pageReadsNotFound("The required object does not exist.")
	}
	return principal, projectID, pageID, nil
}

func pageReadsRepositoryError(err error, notFoundMessage string) error {
	switch {
	case errors.Is(err, store.ErrForbidden):
		return &api.PageReadsForbidden{Detail: "You do not have permission to perform this action."}
	case errors.Is(err, store.ErrRestrictedGuest):
		return &api.PageReadsBadRequest{Message: "You are not allowed to view this page"}
	case errors.Is(err, store.ErrInvalidSearch):
		return api.InvalidSearch{"Null characters are not allowed."}
	case errors.Is(err, store.ErrNotFound):
		return pageReadsNotFound(notFoundMessage)
	case errors.Is(err, store.ErrInvalidOrder):
		return err
	default:
		return pageReadsInternalError()
	}
}

func pageReadsOptionsIdentityError(err error) error {
	var unauthenticated *api.PageReadsUnauthenticated
	if errors.As(err, &unauthenticated) {
		return err
	}
	return &api.PageReadsForbidden{Detail: "You do not have permission to perform this action."}
}

func pageReadsOptionsError(err error) error {
	if errors.Is(err, store.ErrForbidden) || errors.Is(err, store.ErrNotFound) {
		return &api.PageReadsForbidden{Detail: "You do not have permission to perform this action."}
	}
	var lifecycle *store.LifecycleError
	if errors.As(err, &lifecycle) && lifecycle.Status == http.StatusForbidden {
		return &api.PageReadsForbidden{Detail: "You do not have permission to perform this action."}
	}
	var source *djangoapi.HTTPError
	if errors.As(err, &source) && source.Status == http.StatusForbidden {
		return &api.PageReadsForbidden{Detail: "You do not have permission to perform this action."}
	}
	return pageReadsInternalError()
}

func pageJSONResponse(data []byte) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	return &api.RawJSONResponseMetadata{ContentType: pageJSONContentType}, rawJSONReadCloser(data), nil
}

func pageReadsNotFound(message string) error {
	return &api.PageReadsNotFound{Message: message}
}

func pageReadsInternalError() error {
	return &api.PageReadsInternalError{Message: pageReadsInternalErrorDetail}
}

func pageOrderBy(ctx context.Context, value *string) string {
	if value == nil && ReadServiceQueryPresent(ctx, "order_by") {
		return ""
	}
	return optionalString(value, "-created_at")
}

func optionalString(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
