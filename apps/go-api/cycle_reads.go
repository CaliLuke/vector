package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	api "github.com/CaliLuke/go-plane/gen/cycle_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/cyclereads"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"uuid"
)

const cycleReadsInternalErrorDetail = "Something went wrong please try again later"

type cycleReadsRepository interface {
	AuthorizeWorkspace(context.Context, uuid.UUID, string) error
	WorkspaceList(context.Context, uuid.UUID, string) (json.RawMessage, error)
	ArchivedList(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	ArchivedDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
	List(context.Context, uuid.UUID, string, uuid.UUID, string) (json.RawMessage, error)
	Analytics(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, string) (json.RawMessage, error)
	ArchiveDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
	Issues(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, store.IssueListOptions) (json.RawMessage, error)
	IssueDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, uuid.UUID) (json.RawMessage, error)
	Progress(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
	Detail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
}

type cycleReadsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type cycleRecentVisits interface {
	Record(context.Context, uuid.UUID, string, string, uuid.UUID, uuid.UUID) error
}

type cycleReadsService struct {
	repository   cycleReadsRepository
	sessions     cycleReadsSessions
	recentVisits cycleRecentVisits
}

type cycleReadsPrincipal struct{}

func NewCycleReads(
	repository cycleReadsRepository,
	sessions cycleReadsSessions,
	recentVisits cycleRecentVisits,
) (api.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("cycle reads repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("Django session validator is required")
	}
	if recentVisits == nil {
		return nil, fmt.Errorf("cycle recent visits recorder is required")
	}
	return &cycleReadsService{repository: repository, sessions: sessions, recentVisits: recentVisits}, nil
}

func (service *cycleReadsService) APIKeyAuth(
	ctx context.Context,
	key string,
	_ *security.APIKeyScheme,
) (context.Context, error) {
	principalID, _, ok := readServiceSessionPrincipal(ctx)
	if !ok {
		principal, err := service.sessions.Validate(ctx, key)
		if err != nil {
			return ctx, cycleUnauthenticated()
		}
		principalID = principal.UserID
	}
	userID, err := uuid.Parse(principalID)
	if err != nil {
		return ctx, cycleUnauthenticated()
	}
	return context.WithValue(ctx, cycleReadsPrincipal{}, userID), nil
}

func (service *cycleReadsService) WorkspaceList(
	ctx context.Context,
	payload *api.WorkspaceListPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.WorkspaceList(ctx, userID, payload.Slug)
	if err != nil {
		if errors.Is(err, store.ErrForbidden) {
			return nil, nil, &api.CycleReadsWorkspaceForbidden{Detail: "You do not have permission to perform this action."}
		}
		return nil, nil, cycleReadsRepositoryError(err)
	}
	return cycleRawJSONResponse(data)
}

func (service *cycleReadsService) ArchivedList(
	ctx context.Context,
	payload *api.ArchivedListPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, projectID, err := service.project(ctx, payload.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.ArchivedList(ctx, userID, payload.Slug, projectID)
	if err != nil {
		return nil, nil, cycleReadsRepositoryError(err)
	}
	return cycleRawJSONResponse(data)
}

func (service *cycleReadsService) ArchivedDetail(
	ctx context.Context,
	payload *api.ArchivedDetailPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, projectID, cycleID, err := service.projectCycle(ctx, payload.ProjectID, payload.Pk)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.ArchivedDetail(ctx, userID, payload.Slug, projectID, cycleID)
	if err != nil {
		return nil, nil, cycleReadsRepositoryError(err)
	}
	return cycleRawJSONResponse(data)
}

func (service *cycleReadsService) List(
	ctx context.Context,
	payload *api.ListPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, projectID, err := service.project(ctx, payload.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	cycleView := optionalPayloadString(payload, "cycle_view")
	if cycleView == "" {
		cycleView = "all"
	}
	data, err := service.repository.List(ctx, userID, payload.Slug, projectID, cycleView)
	if err != nil {
		return nil, nil, cycleReadsRepositoryError(err)
	}
	return cycleRawJSONResponse(data)
}

func (service *cycleReadsService) Analytics(
	ctx context.Context,
	payload *api.AnalyticsPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, projectID, cycleID, err := service.projectCycle(ctx, payload.ProjectID, payload.CycleID)
	if err != nil {
		return nil, nil, err
	}
	analyticsType := optionalPayloadString(payload, "type")
	if analyticsType == "" && !ReadServiceQueryPresent(ctx, "type") {
		analyticsType = "issues"
	}
	data, err := service.repository.Analytics(ctx, userID, payload.Slug, projectID, cycleID, analyticsType)
	if err != nil {
		return nil, nil, cycleReadsRepositoryError(err)
	}
	return cycleRawJSONResponse(data)
}

func (service *cycleReadsService) ArchiveDetail(
	ctx context.Context,
	payload *api.ArchiveDetailPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, projectID, cycleID, err := service.projectCycle(ctx, payload.ProjectID, payload.CycleID)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.ArchiveDetail(ctx, userID, payload.Slug, projectID, cycleID)
	if err != nil {
		return nil, nil, cycleReadsRepositoryError(err)
	}
	return cycleRawJSONResponse(data)
}

func (service *cycleReadsService) Issues(
	ctx context.Context,
	payload *api.IssuesPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, projectID, cycleID, err := service.projectCycle(ctx, payload.ProjectID, payload.CycleID)
	if err != nil {
		return nil, nil, err
	}
	options, err := cycleIssueOptions(ctx, payload)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.Issues(ctx, userID, payload.Slug, projectID, cycleID, options)
	if err != nil {
		if errors.Is(err, store.ErrZeroPageSize) ||
			errors.Is(err, store.ErrNegativePageSize) ||
			errors.Is(err, store.ErrPreviousPageSlice) ||
			errors.Is(err, store.ErrPaginationOverflow) ||
			errors.Is(err, store.ErrDynamicGrouping) {
			MarkReadServiceDjangoHTML500(ctx)
		}
		return nil, nil, cycleReadsRepositoryError(err)
	}
	return cycleRawJSONResponse(data)
}

func (service *cycleReadsService) IssueDetail(
	ctx context.Context,
	payload *api.IssueDetailPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, projectID, cycleID, err := service.projectCycle(ctx, payload.ProjectID, payload.CycleID)
	if err != nil {
		return nil, nil, err
	}
	issueID, err := uuid.Parse(payload.IssueID)
	if err != nil {
		return nil, nil, cycleNotFound()
	}
	data, err := service.repository.IssueDetail(ctx, userID, payload.Slug, projectID, cycleID, issueID)
	if err != nil {
		return nil, nil, cycleReadsRepositoryError(err)
	}
	return cycleRawJSONResponse(data)
}

func (service *cycleReadsService) Progress(
	ctx context.Context,
	payload *api.ProgressPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, projectID, cycleID, err := service.projectCycle(ctx, payload.ProjectID, payload.CycleID)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.Progress(ctx, userID, payload.Slug, projectID, cycleID)
	if err != nil {
		return nil, nil, cycleReadsRepositoryError(err)
	}
	return cycleRawJSONResponse(data)
}

func (service *cycleReadsService) Detail(
	ctx context.Context,
	payload *api.DetailPayload,
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, projectID, cycleID, err := service.projectCycle(ctx, payload.ProjectID, payload.Pk)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.Detail(ctx, userID, payload.Slug, projectID, cycleID)
	if err != nil {
		return nil, nil, cycleReadsRepositoryError(err)
	}
	// Django enqueues this write. The replacement records it synchronously so
	// the visit is durable before the request ends; failure remains best-effort.
	_ = service.recentVisits.Record(ctx, userID, payload.Slug, "cycle", cycleID, projectID)
	return cycleRawJSONResponse(data)
}

func cycleRawJSONResponse(data json.RawMessage) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser(data), nil
}

func (service *cycleReadsService) WorkspaceListOptions(
	ctx context.Context,
	payload *api.WorkspaceListOptionsPayload,
) (*api.CycleReadsMetadata, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	if err := service.repository.AuthorizeWorkspace(ctx, userID, payload.Slug); err != nil {
		if errors.Is(err, store.ErrForbidden) {
			return nil, &api.CycleReadsWorkspaceForbidden{Detail: "You do not have permission to perform this action."}
		}
		return nil, cycleReadsRepositoryError(err)
	}
	return newCycleReadsMetadata("Workspace Cycles Endpoint"), nil
}

func (service *cycleReadsService) AnalyticsOptions(
	context.Context,
	*api.AnalyticsOptionsPayload,
) (*api.CycleReadsMetadata, error) {
	return newCycleReadsMetadata("Cycle Analytics Endpoint"), nil
}

func (service *cycleReadsService) ProgressOptions(
	context.Context,
	*api.ProgressOptionsPayload,
) (*api.CycleReadsMetadata, error) {
	return newCycleReadsMetadata("Cycle Progress Endpoint"), nil
}

func newCycleReadsMetadata(name string) *api.CycleReadsMetadata {
	return &api.CycleReadsMetadata{
		Name:        name,
		Description: "",
		Renders:     []string{"application/json"},
		Parses:      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
	}
}

func (service *cycleReadsService) project(ctx context.Context, project string) (uuid.UUID, uuid.UUID, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return uuid.Nil(), uuid.Nil(), err
	}
	projectID, err := uuid.Parse(project)
	if err != nil {
		return uuid.Nil(), uuid.Nil(), cycleForbidden()
	}
	return userID, projectID, nil
}

func (service *cycleReadsService) projectCycle(
	ctx context.Context,
	project string,
	cycle string,
) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	userID, projectID, err := service.project(ctx, project)
	if err != nil {
		return uuid.Nil(), uuid.Nil(), uuid.Nil(), err
	}
	cycleID, err := uuid.Parse(cycle)
	if err != nil {
		return uuid.Nil(), uuid.Nil(), uuid.Nil(), cycleNotFound()
	}
	return userID, projectID, cycleID, nil
}

func (service *cycleReadsService) userID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(cycleReadsPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), cycleUnauthenticated()
	}
	return userID, nil
}

func cycleIssueOptions(ctx context.Context, payload any) (store.IssueListOptions, error) {
	values := payloadValues(payload)
	rawPerPage := values[normalizePayloadName("per_page")]
	perPageSet := rawPerPage != "" || ReadServiceQueryPresent(ctx, "per_page")
	perPage := 0
	if perPageSet {
		var tooLarge bool
		var err error
		perPage, tooLarge, err = store.ParsePageSize(rawPerPage)
		if err != nil {
			return store.IssueListOptions{}, cycleInvalidPagination("Invalid per_page parameter.")
		}
		if tooLarge {
			return store.IssueListOptions{}, cycleInvalidPagination("Invalid per_page value. Cannot exceed 1000.")
		}
	}
	filters := make(map[string]string)
	for _, name := range cycleIssueFilterNames {
		if value := values[normalizePayloadName(name)]; value != "" {
			filters[name] = value
		}
	}
	cursor := values[normalizePayloadName("cursor")]
	return store.IssueListOptions{
		OrderBy:       values[normalizePayloadName("order_by")],
		GroupBy:       values[normalizePayloadName("group_by")],
		SubGroupBy:    values[normalizePayloadName("sub_group_by")],
		Cursor:        cursor,
		CursorSet:     cursor != "" || ReadServiceQueryPresent(ctx, "cursor"),
		PerPage:       perPage,
		PerPageSet:    perPageSet,
		Filters:       filters,
		ComplexFilter: values[normalizePayloadName("filters")],
	}, nil
}

var cycleIssueFilterNames = []string{
	"state", "state_group", "estimate_point", "priority", "assignees", "labels", "created_by", "logged_by", "parent",
	"mentions", "name", "created_at", "updated_at", "start_date", "target_date", "completed_at", "project", "cycle",
	"module", "intake_status", "inbox_status", "sub_issue", "subscriber", "start_target_date", "type",
}

func optionalPayloadString(payload any, name string) string {
	return payloadValues(payload)[normalizePayloadName(name)]
}

func payloadValues(payload any) map[string]string {
	result := make(map[string]string)
	value := reflect.ValueOf(payload)
	if !value.IsValid() {
		return result
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return result
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return result
	}
	typeOf := value.Type()
	for index := range value.NumField() {
		field := value.Field(index)
		if field.Kind() == reflect.Pointer {
			if field.IsNil() {
				continue
			}
			field = field.Elem()
		}
		var text string
		switch field.Kind() {
		case reflect.String:
			text = field.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			text = strconv.FormatInt(field.Int(), 10)
		case reflect.Bool:
			text = strconv.FormatBool(field.Bool())
		default:
			continue
		}
		if text == "" {
			continue
		}
		structField := typeOf.Field(index)
		result[normalizePayloadName(structField.Name)] = text
		for _, tagName := range []string{"json", "form"} {
			if tag := strings.Split(structField.Tag.Get(tagName), ",")[0]; tag != "" && tag != "-" {
				result[normalizePayloadName(tag)] = text
			}
		}
	}
	return result
}

func normalizePayloadName(value string) string {
	value = strings.ToLower(value)
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, "-", "")
	return value
}

func cycleReadsRepositoryError(err error) error {
	var filterError *store.FilterError
	if errors.As(err, &filterError) {
		message, code := filterError.Message, filterError.Code
		response := &api.CycleReadsFilterError{
			Message: &message,
			Code:    &code,
		}
		if filterError.Errors != nil {
			encoded, encodeErr := json.Marshal(filterError.Errors)
			if encodeErr != nil {
				return encodeErr
			}
			response.Errors = loom.NullableValue(loom.JSONValue(encoded))
		}
		return response
	}
	var invalidGrouping *store.InvalidGroupingFieldError
	if errors.As(err, &invalidGrouping) {
		detail := fmt.Sprintf("Invalid %s field: %s", invalidGrouping.Field, invalidGrouping.Value)
		return &api.CycleReadsInvalidGrouping{Detail: &detail}
	}
	switch {
	case errors.Is(err, store.ErrForbidden):
		return cycleForbidden()
	case errors.Is(err, store.ErrNotFound):
		return cycleNotFound()
	case errors.Is(err, store.ErrCycleDatesMissing):
		return cycleBadRequest("Cycle has no start or end date")
	case errors.Is(err, store.ErrInvalidGrouping):
		return cycleBadRequest("Group by and sub group by cannot have same parameters")
	case errors.Is(err, store.ErrMissingGroupKey):
		return cycleBadRequest("The required key does not exist.")
	case errors.Is(err, store.ErrPaginationParsing):
		return cycleParseError("Error in parsing")
	case errors.Is(err, store.ErrInvalidPagination):
		return cycleInvalidPagination("Invalid cursor parameter.")
	default:
		return cycleInternalError()
	}
}

func cycleBadRequest(message string) error {
	return &api.CycleReadsBadRequest{ErrorMessage: &message}
}

func cycleInvalidPagination(detail string) error {
	return &api.CycleReadsInvalidPagination{Detail: &detail}
}

func cycleParseError(detail string) error {
	return &api.CycleReadsParseError{Detail: &detail}
}

func cycleUnauthenticated() error {
	return &api.CycleReadsUnauthenticated{Detail: unauthenticatedDetail}
}

func cycleForbidden() error {
	return &api.CycleReadsForbidden{Message: "You don't have the required permissions."}
}

func cycleNotFound() error {
	return &api.CycleReadsNotFound{Message: "Cycle not found"}
}

func cycleInternalError() error {
	return &api.CycleReadsInternalError{Message: cycleReadsInternalErrorDetail}
}
