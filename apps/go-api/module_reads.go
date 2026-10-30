package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/module_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	store "github.com/CaliLuke/go-plane/internal/modulereads"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

const moduleReadsInternalErrorDetail = "Something went wrong please try again later"

type moduleReadsRepository interface {
	WorkspaceList(context.Context, uuid.UUID, string) (json.RawMessage, error)
	ArchivedList(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	ArchivedDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
	List(context.Context, uuid.UUID, string, uuid.UUID, bool) (json.RawMessage, error)
	ArchiveDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
	Issues(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, store.IssueListOptions) (store.IssuePage, error)
	IssueDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, uuid.UUID) (json.RawMessage, error)
	Links(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
	LinkDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, uuid.UUID) (json.RawMessage, error)
	Detail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error)
}

type moduleReadsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type moduleRecentVisits interface {
	Record(context.Context, uuid.UUID, string, string, uuid.UUID, uuid.UUID) error
}

type moduleReadsService struct {
	repository   moduleReadsRepository
	sessions     moduleReadsSessions
	recentVisits moduleRecentVisits
}

type moduleReadsPrincipal struct{}

func NewModuleReads(repository moduleReadsRepository, sessions moduleReadsSessions, recentVisits moduleRecentVisits) (api.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("module reads repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("session validator for Django is required")
	}
	if recentVisits == nil {
		return nil, fmt.Errorf("module recent visits repository is required")
	}
	return &moduleReadsService{repository: repository, sessions: sessions, recentVisits: recentVisits}, nil
}

func (service *moduleReadsService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	userID, err := readServiceSessionUserUUID(ctx, key, service.sessions)
	if err != nil {
		return ctx, &api.ModuleReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, moduleReadsPrincipal{}, userID), nil
}

func (service *moduleReadsService) WorkspaceList(ctx context.Context, payload *api.WorkspaceListPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.WorkspaceList(ctx, userID, payload.Slug)
	if err != nil {
		return nil, nil, moduleReadFailure(ctx, "workspace_list", err, modulePermissionRepositoryError(err))
	}
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser(data), nil
}

func (service *moduleReadsService) WorkspaceListOptions(ctx context.Context, payload *api.WorkspaceListOptionsPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	if _, err := service.repository.WorkspaceList(ctx, userID, payload.Slug); err != nil {
		return nil, nil, moduleReadFailure(ctx, "workspace_list_options", err, modulePermissionRepositoryError(err))
	}
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser([]byte(`{"name":"Workspace Modules Endpoint","description":"","renders":["application/json"],"parses":["application/json","application/x-www-form-urlencoded","multipart/form-data"]}`)), nil
}

func (service *moduleReadsService) ArchivedList(ctx context.Context, payload *api.ArchivedListPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, projectID, err := service.projectIdentity(ctx, payload.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.ArchivedList(ctx, userID, payload.Slug, projectID)
	if err != nil {
		return nil, nil, moduleReadFailure(ctx, "archived_list", err, modulePermissionRepositoryError(err))
	}
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser(data), nil
}

func (service *moduleReadsService) ArchivedDetail(ctx context.Context, payload *api.ArchivedDetailPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	return service.readModuleResource(ctx, "archived_detail", payload.Slug, payload.ProjectID, payload.Pk, service.repository.ArchivedDetail)
}

func (service *moduleReadsService) List(ctx context.Context, payload *api.ListPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, projectID, err := service.projectIdentity(ctx, payload.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	fields := optionalModuleString(payload.Fields)
	data, err := service.repository.List(ctx, userID, payload.Slug, projectID, moduleReadFieldsRequested(fields))
	if err != nil {
		return nil, nil, moduleReadFailure(ctx, "list", err, moduleReadsRepositoryError(err))
	}
	if moduleReadFieldsRequested(fields) {
		data, err = selectModuleReadFields(data, fields)
		if err != nil {
			workItemLogError(ctx, "module_reads.list", "project_fields", err)
			return nil, nil, moduleReadsInternalError()
		}
	}
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser(data), nil
}

func (service *moduleReadsService) ArchiveDetail(ctx context.Context, payload *api.ArchiveDetailPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	return service.readModuleResource(ctx, "archive_detail", payload.Slug, payload.ProjectID, payload.ModuleID, service.repository.ArchiveDetail)
}

func (service *moduleReadsService) Issues(ctx context.Context, payload *api.IssuesPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, projectID, moduleID, err := service.moduleIdentity(ctx, payload.ProjectID, payload.ModuleID)
	if err != nil {
		return nil, nil, err
	}
	perPagePresent := payload.PerPage != nil || ReadServiceQueryPresent(ctx, "per_page")
	cursorPresent := payload.Cursor != nil || ReadServiceQueryPresent(ctx, "cursor")
	orderByPresent := payload.OrderBy != nil || ReadServiceQueryPresent(ctx, "order_by")
	typePresent := payload.Type != nil || ReadServiceQueryPresent(ctx, "type")
	assigneesPresent := payload.Assignees != nil || ReadServiceQueryPresent(ctx, "assignees")
	labelsPresent := payload.Labels != nil || ReadServiceQueryPresent(ctx, "labels")
	cyclePresent := payload.Cycle != nil || ReadServiceQueryPresent(ctx, "cycle")
	subscriberPresent := payload.Subscriber != nil || ReadServiceQueryPresent(ctx, "subscriber")
	limit, err := moduleIssuePerPage(optionalModuleString(payload.PerPage), perPagePresent)
	if err != nil {
		return nil, nil, moduleIssueParseError(err.Error())
	}
	page, cursorValue, err := moduleIssueCursorPage(optionalModuleString(payload.Cursor), cursorPresent, limit)
	if err != nil {
		if err.Error() == "Error in parsing" {
			return nil, nil, moduleIssueParseError(err.Error())
		}
		return nil, nil, moduleIssueParseError("Invalid cursor parameter.")
	}
	groupBy := optionalModuleString(payload.GroupBy)
	subGroupBy := optionalModuleString(payload.SubGroupBy)
	responseSubGroupBy := subGroupBy
	if groupBy == "" {
		responseSubGroupBy = ""
	}
	if validateModuleErr := validateModuleIssueGrouping(groupBy, subGroupBy); validateModuleErr != nil {
		if strings.HasPrefix(validateModuleErr.Error(), "Invalid ") {
			return nil, nil, moduleIssueParseError(validateModuleErr.Error())
		}
		return nil, nil, moduleIssueBadRequest(validateModuleErr.Error())
	}
	if groupBy != "" && float64(page)*cursorValue < 0 {
		return nil, nil, moduleIssueParseError("Error in parsing")
	}
	if groupBy == "" && limit <= 0 {
		return nil, nil, moduleReadsInternalError()
	}
	options := store.IssueListOptions{
		IncludeProperties: ReadServiceQueryValues(ctx).Get("include_properties") == "true",
		Limit:             limit,
		Page:              page,
		OrderBy:           optionalModuleString(payload.OrderBy),
		GroupBy:           groupBy,
		CursorValue:       cursorValue,
		CursorSet:         cursorPresent,
		OrderBySet:        orderByPresent,
		SubGroupBy:        subGroupBy,
		State:             optionalModuleString(payload.State),
		StateGroup:        optionalModuleString(payload.StateGroup),
		EstimatePoint:     optionalModuleString(payload.EstimatePoint),
		Priority:          optionalModuleString(payload.Priority),
		Assignees:         optionalModuleString(payload.Assignees),
		AssigneesSet:      assigneesPresent,
		Labels:            optionalModuleString(payload.Labels),
		LabelsSet:         labelsPresent,
		Project:           optionalModuleString(payload.Project),
		LoggedBy:          optionalModuleString(payload.LoggedBy),
		CreatedBy:         optionalModuleString(payload.CreatedBy),
		Parent:            optionalModuleString(payload.Parent),
		Mentions:          optionalModuleString(payload.Mentions),
		Name:              optionalModuleString(payload.Name),
		CreatedAt:         optionalModuleString(payload.CreatedAt),
		UpdatedAt:         optionalModuleString(payload.UpdatedAt),
		StartDate:         optionalModuleString(payload.StartDate),
		TargetDate:        optionalModuleString(payload.TargetDate),
		CompletedAt:       optionalModuleString(payload.CompletedAt),
		Cycle:             optionalModuleString(payload.Cycle),
		CycleSet:          cyclePresent,
		Module:            optionalModuleString(payload.Module),
		ModuleSet:         payload.Module != nil,
		IntakeStatus:      optionalModuleString(payload.IntakeStatus),
		InboxStatus:       optionalModuleString(payload.InboxStatus),
		SubIssue:          optionalModuleString(payload.SubIssue),
		Subscriber:        optionalModuleString(payload.Subscriber),
		SubscriberSet:     subscriberPresent,
		StartTargetDate:   optionalModuleString(payload.StartTargetDate),
		Type:              optionalModuleString(payload.Type),
		TypeSet:           typePresent,
		Filters:           optionalModuleString(payload.Filters),
	}
	issuePage, err := service.repository.Issues(ctx, userID, payload.Slug, projectID, moduleID, options)
	if err != nil {
		return nil, nil, moduleReadFailure(ctx, "issues", err, moduleReadsRepositoryError(err))
	}
	var issues []map[string]any
	if unmarshalErr := json.Unmarshal(issuePage.Items, &issues); unmarshalErr != nil {
		return nil, nil, moduleReadFailure(ctx, "issues", unmarshalErr, moduleReadsInternalError())
	}
	results := any(issues)
	count := len(issues)
	if groupBy == "" && moduleIssueGroupArrayField(subGroupBy) != "" {
		count = issuePage.PageCount
	}
	maximumGroup := issuePage.Total
	groupedNext := false
	if groupBy != "" {
		results, count, maximumGroup, groupedNext = groupModuleIssues(
			issues, groupBy, subGroupBy, issuePage.GroupValues, issuePage.SubGroupValues,
			issuePage.GroupCounts, issuePage.SubGroupCounts, limit, page,
		)
		groupedNext = groupedNext || issuePage.GroupedHasNext
	}
	totalResults := issuePage.Total
	nextPage := (page+1)*limit < issuePage.Total
	if groupBy != "" {
		nextPage = groupedNext
	} else if moduleIssueGroupArrayField(subGroupBy) != "" {
		nextPage = issuePage.GroupedHasNext
	}
	previousPage := page > 0
	totalPages, err := moduleIssueTotalPages(maximumGroup, limit)
	if err != nil {
		return nil, nil, moduleReadFailure(ctx, "issues", err, moduleReadsInternalError())
	}
	envelope := map[string]any{
		"grouped_by":        nullableModuleString(groupBy),
		"sub_grouped_by":    nullableModuleString(responseSubGroupBy),
		"total_count":       totalResults,
		"next_cursor":       fmt.Sprintf("%d:%d:0", limit, page+1),
		"prev_cursor":       fmt.Sprintf("%d:%d:1", limit, page-1),
		"next_page_results": nextPage,
		"prev_page_results": previousPage,
		"count":             count,
		"total_pages":       totalPages,
		"total_results":     totalResults,
		"extra_stats":       nil,
		"results":           results,
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return nil, nil, moduleReadFailure(ctx, "issues", err, moduleReadsInternalError())
	}
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser(data), nil
}

func (service *moduleReadsService) IssueDetail(ctx context.Context, payload *api.IssueDetailPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, projectID, moduleID, err := service.moduleIdentity(ctx, payload.ProjectID, payload.ModuleID)
	if err != nil {
		return nil, nil, err
	}
	issueID, err := uuid.Parse(payload.IssueID)
	if err != nil {
		return nil, nil, moduleReadsNotFound()
	}
	data, err := service.repository.IssueDetail(ctx, userID, payload.Slug, projectID, moduleID, issueID)
	if err != nil {
		return nil, nil, moduleReadFailure(ctx, "issue_detail", err, moduleReadsRepositoryError(err))
	}
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser(data), nil
}

func (service *moduleReadsService) Links(ctx context.Context, payload *api.LinksPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	return service.readModuleResource(ctx, "links", payload.Slug, payload.ProjectID, payload.ModuleID, service.repository.Links)
}

func (service *moduleReadsService) LinkDetail(ctx context.Context, payload *api.LinkDetailPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	projectID, projectError := uuid.Parse(payload.ProjectID)
	moduleID, moduleError := uuid.Parse(payload.ModuleID)
	linkID, linkError := uuid.Parse(payload.Pk)
	if projectError != nil || moduleError != nil || linkError != nil {
		return nil, nil, moduleReadsLinkNotFound()
	}
	data, err := service.repository.LinkDetail(ctx, userID, payload.Slug, projectID, moduleID, linkID)
	if err != nil {
		return nil, nil, moduleReadFailure(ctx, "link_detail", err, moduleReadsLinkRepositoryError(err))
	}
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser(data), nil
}

func (service *moduleReadsService) Detail(ctx context.Context, payload *api.DetailPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	projectID, projectError := uuid.Parse(payload.ProjectID)
	moduleID, moduleError := uuid.Parse(payload.Pk)
	if projectError != nil || moduleError != nil {
		return nil, nil, moduleReadsModuleNotFound()
	}
	data, err := service.repository.Detail(ctx, userID, payload.Slug, projectID, moduleID)
	if err != nil {
		return nil, nil, moduleReadFailure(ctx, "detail", err, moduleReadsModuleRepositoryError(err))
	}
	_ = service.recentVisits.Record(ctx, userID, payload.Slug, "module", moduleID, projectID)
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser(data), nil
}

func (service *moduleReadsService) projectIdentity(ctx context.Context, projectValue string) (uuid.UUID, uuid.UUID, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return uuid.Nil(), uuid.Nil(), err
	}
	projectID, err := uuid.Parse(projectValue)
	if err != nil {
		return uuid.Nil(), uuid.Nil(), moduleReadsNotFound()
	}
	return userID, projectID, nil
}

func (service *moduleReadsService) readModuleResource(
	ctx context.Context,
	operation, slug, projectValue, moduleValue string,
	read func(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error),
) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, projectID, moduleID, err := service.moduleIdentity(ctx, projectValue, moduleValue)
	if err != nil {
		return nil, nil, err
	}
	data, err := read(ctx, userID, slug, projectID, moduleID)
	if err != nil {
		return nil, nil, moduleReadFailure(ctx, operation, err, modulePermissionRepositoryError(err))
	}
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser(data), nil
}

func (service *moduleReadsService) moduleIdentity(ctx context.Context, projectValue, moduleValue string) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	userID, projectID, err := service.projectIdentity(ctx, projectValue)
	if err != nil {
		return uuid.Nil(), uuid.Nil(), uuid.Nil(), err
	}
	moduleID, err := uuid.Parse(moduleValue)
	if err != nil {
		return uuid.Nil(), uuid.Nil(), uuid.Nil(), moduleReadsNotFound()
	}
	return userID, projectID, moduleID, nil
}

func (service *moduleReadsService) userID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(moduleReadsPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.ModuleReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
	return userID, nil
}

func moduleReadsRepositoryError(err error) error {
	var filterError *store.FilterError
	switch {
	case errors.As(err, &filterError):
		message, code := filterError.Message, filterError.Code
		response := &api.ModuleIssueListBadRequest{
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
	case errors.Is(err, store.ErrInvalidDetail):
		return moduleIssueBadRequest("Please provide valid detail")
	case errors.Is(err, store.ErrForbidden):
		return &api.ModuleReadsForbidden{Message: "You don't have the required permissions."}
	case errors.Is(err, store.ErrNotFound):
		return moduleReadsNotFound()
	default:
		return moduleReadsInternalError()
	}
}

func moduleReadsNotFound() error {
	return &api.ModuleReadsNotFound{Detail: "The required object does not exist."}
}

func moduleReadsLinkRepositoryError(err error) error {
	if errors.Is(err, store.ErrForbidden) {
		return moduleReadsPermissionDenied()
	}
	if errors.Is(err, store.ErrNotFound) {
		return moduleReadsLinkNotFound()
	}
	return moduleReadsInternalError()
}

func modulePermissionRepositoryError(err error) error {
	if errors.Is(err, store.ErrForbidden) {
		return moduleReadsPermissionDenied()
	}
	if errors.Is(err, store.ErrNotFound) {
		return moduleReadsNotFound()
	}
	return moduleReadsInternalError()
}

func moduleReadsPermissionDenied() error {
	return &api.ModuleReadsPermissionDenied{Detail: "You do not have permission to perform this action."}
}

func moduleReadsModuleRepositoryError(err error) error {
	if errors.Is(err, store.ErrForbidden) {
		return &api.ModuleReadsForbidden{Message: "You don't have the required permissions."}
	}
	if errors.Is(err, store.ErrNotFound) {
		return moduleReadsModuleNotFound()
	}
	return moduleReadsInternalError()
}

func moduleReadsLinkNotFound() error {
	return &api.ModuleReadsLinkNotFound{Detail: "No ModuleLink matches the given query."}
}

func moduleReadsModuleNotFound() error {
	return &api.ModuleReadsModuleNotFound{Message: "Module not found"}
}

func moduleIssueBadRequest(message string) error {
	return &api.ModuleIssueListBadRequest{ErrorMessage: &message}
}

func moduleIssueParseError(detail string) error {
	return &api.ModuleIssueListBadRequest{Detail: &detail}
}

func moduleReadsInternalError() error {
	return &api.ModuleReadsInternalError{Message: moduleReadsInternalErrorDetail}
}

func moduleReadFailure(ctx context.Context, operation string, cause, response error) error {
	var internal *api.ModuleReadsInternalError
	if errors.As(response, &internal) {
		workItemLogError(ctx, "module_reads."+operation, "read", cause)
	}
	return response
}

func moduleReadFieldsRequested(value string) bool {
	return len(strings.FieldsFunc(value, func(character rune) bool { return character == ',' })) > 0
}

func selectModuleReadFields(data json.RawMessage, value string) (json.RawMessage, error) {
	hasField := moduleReadFieldsRequested(value)
	if !hasField {
		return data, nil
	}
	var modules []map[string]any
	if err := json.Unmarshal(data, &modules); err != nil {
		return nil, err
	}
	for _, module := range modules {
		module["archived_at"] = nil
	}
	encoded, err := json.Marshal(modules)
	return json.RawMessage(encoded), err
}

func optionalModuleString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nullableModuleString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func moduleIssuePerPage(value string, present bool) (int, error) {
	if !present {
		return 1000, nil
	}
	perPage, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, errors.New("Invalid per_page parameter.")
	}
	if perPage > 1000 {
		return 0, errors.New("Invalid per_page value. Cannot exceed 1000.")
	}
	return perPage, nil
}

func moduleIssueCursorPage(cursor string, present bool, limit int) (int, float64, error) {
	if limit > 1000 {
		return 0, 0, errors.New("invalid per_page")
	}
	if !present {
		return 0, 0, nil
	}
	parts := strings.Split(cursor, ":")
	if len(parts) != 3 {
		return 0, 0, errors.New("invalid cursor")
	}
	first := strings.TrimSpace(parts[0])
	var cursorValue float64
	var err error
	if strings.Contains(first, ".") {
		cursorValue, err = strconv.ParseFloat(first, 64)
	} else {
		var integerValue int
		integerValue, err = strconv.Atoi(first)
		cursorValue = float64(integerValue)
	}
	if err != nil {
		return 0, 0, errors.New("invalid cursor value")
	}
	page, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, errors.New("invalid cursor offset")
	}
	if page < 0 {
		return 0, 0, errors.New("Error in parsing")
	}
	if _, err := strconv.Atoi(strings.TrimSpace(parts[2])); err != nil {
		return 0, 0, errors.New("invalid cursor direction")
	}
	return page, cursorValue, nil
}
func moduleIssueTotalPages(maximumGroup, limit int) (int, error) {
	if maximumGroup <= 0 {
		return 0, nil
	}
	if limit == 0 {
		return 0, errors.New("division by zero")
	}
	return int(math.Ceil(float64(maximumGroup) / float64(limit))), nil
}

// These wrappers preserve modulereads' established test and service call names.
func validateModuleIssueGrouping(groupBy, subGroupBy string) error {
	return issuequery.ValidateGrouping(groupBy, subGroupBy)
}

func groupModuleIssues(
	issues []map[string]any,
	groupBy, subGroupBy string,
	groupValues, _ []any,
	groupCounts, subgroupCounts map[string]int,
	limit, page int,
) (any, int, int, bool) {
	return issuequery.Group(issues, groupBy, subGroupBy, groupValues, nil, groupCounts, subgroupCounts, limit, page)
}

func moduleIssueGroupArrayField(field string) string {
	return issuequery.GroupArrayField(field)
}
