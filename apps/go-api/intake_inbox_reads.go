package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"time"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/intake_inbox_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/intakeinboxreads"
	"github.com/CaliLuke/loom/security"
)

const intakeInboxReadsInternalErrorDetail = "Something went wrong please try again later"

type intakeInboxReadsRepository interface {
	DraftIssueList(context.Context, uuid.UUID, string, store.ListOptions) (jsontext.Value, error)
	DraftIssueDetail(context.Context, uuid.UUID, string, uuid.UUID) (jsontext.Value, error)
	InboxIssueList(context.Context, uuid.UUID, string, uuid.UUID, store.ListOptions) (jsontext.Value, error)
	InboxIssueDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (jsontext.Value, error)
	InboxList(context.Context, uuid.UUID, string, uuid.UUID) (jsontext.Value, error)
	InboxDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (jsontext.Value, error)
	IntakeIssueList(context.Context, uuid.UUID, string, uuid.UUID, store.ListOptions) (jsontext.Value, error)
	IntakeIssueDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (jsontext.Value, error)
	DescriptionVersionList(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, *string) (jsontext.Value, error)
	DescriptionVersionDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, uuid.UUID) (jsontext.Value, error)
	IntakeList(context.Context, uuid.UUID, string, uuid.UUID) (jsontext.Value, error)
	IntakeDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (jsontext.Value, error)
}

type intakeInboxReadsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type intakeInboxReadsService struct {
	repository intakeInboxReadsRepository
	sessions   intakeInboxReadsSessions
}

type intakeInboxReadsPrincipal struct {
	userID   uuid.UUID
	timezone string
}

func NewIntakeInboxReads(repository intakeInboxReadsRepository, sessions intakeInboxReadsSessions) (api.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("intake and inbox reads repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("Django session validator is required")
	}
	return &intakeInboxReadsService{repository: repository, sessions: sessions}, nil
}

func (service *intakeInboxReadsService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	if prevalidatedUserID, timezone, inspected := readServiceSessionPrincipal(ctx); inspected {
		userID, err := uuid.Parse(prevalidatedUserID)
		if err != nil {
			return ctx, &api.IntakeInboxReadsUnauthenticated{Detail: unauthenticatedDetail}
		}
		return context.WithValue(ctx, intakeInboxReadsPrincipal{}, intakeInboxReadsPrincipal{userID: userID, timezone: timezone}), nil
	}
	principal, err := service.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &api.IntakeInboxReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return ctx, &api.IntakeInboxReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, intakeInboxReadsPrincipal{}, intakeInboxReadsPrincipal{userID: userID, timezone: principal.UserTimezone}), nil
}

func (service *intakeInboxReadsService) AuthorizeMetadata(ctx context.Context, sessionID string) error {
	_, err := service.APIKeyAuth(ctx, sessionID, &security.APIKeyScheme{Name: "plane_session_cookie"})
	return err
}

func (service *intakeInboxReadsService) DraftIssueList(ctx context.Context, payload *api.DraftIssueListPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, err := service.principal(ctx)
	if err != nil {
		return nil, nil, err
	}
	options := withIntakeListQueryPresence(ctx, draftListOptions(payload, principal.timezone))
	data, err := service.repository.DraftIssueList(ctx, principal.userID, payload.Slug, options)
	if err != nil {
		return nil, nil, intakeInboxReadsRepositoryError(err, "")
	}
	return intakeInboxReadCloser(data, principal.timezone, intakeInboxDraftPageShape)
}

func (service *intakeInboxReadsService) DraftIssueDetail(ctx context.Context, payload *api.DraftIssueDetailPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, err := service.principal(ctx)
	if err != nil {
		return nil, nil, err
	}
	draftID, err := uuid.Parse(payload.Pk)
	if err != nil {
		return nil, nil, &api.IntakeInboxReadsNotFound{Message: "The required object does not exist."}
	}
	data, err := service.repository.DraftIssueDetail(ctx, principal.userID, payload.Slug, draftID)
	if err != nil {
		return nil, nil, intakeInboxReadsRepositoryError(err, "The required object does not exist.")
	}
	return intakeInboxReadCloser(data, principal.timezone, intakeInboxDraftDetailShape)
}

func (service *intakeInboxReadsService) InboxIssueList(ctx context.Context, payload *api.InboxIssueListPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, err := service.projectPrincipal(ctx, payload.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	options := withIntakeListQueryPresence(ctx, intakeIssueListOptions(
		principal.timezone, payload.Cursor, payload.PerPage, payload.OrderBy, payload.Status, payload.State, payload.StateGroup, payload.EstimatePoint,
		payload.Priority, payload.Parent, payload.Labels, payload.Assignees, payload.Mentions, payload.CreatedBy, payload.LoggedBy,
		payload.Name, payload.CreatedAt, payload.UpdatedAt, payload.StartDate, payload.TargetDate, payload.CompletedAt, payload.Type,
		payload.Project, payload.Cycle, payload.Module, payload.IntakeStatus, payload.InboxStatus, payload.SubIssue, payload.Subscriber,
		payload.StartTargetDate,
	))
	data, err := service.repository.InboxIssueList(ctx, principal.userID, payload.Slug, projectID, options)
	if err != nil {
		return nil, nil, intakeInboxReadsRepositoryError(err, "Intake not found")
	}
	return intakeInboxReadCloser(data, principal.timezone, intakeInboxIssuePageShape)
}

func (service *intakeInboxReadsService) InboxIssueDetail(ctx context.Context, payload *api.InboxIssueDetailPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, issueID, err := service.detailPrincipal(ctx, payload.ProjectID, payload.Pk)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.InboxIssueDetail(ctx, principal.userID, payload.Slug, projectID, issueID)
	if err != nil {
		return nil, nil, intakeInboxReadsRepositoryError(err, "The required object does not exist.")
	}
	return intakeInboxReadCloser(data, principal.timezone, intakeInboxIssueDetailShape)
}

func (service *intakeInboxReadsService) InboxList(ctx context.Context, payload *api.InboxListPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, err := service.projectPrincipal(ctx, payload.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.InboxList(ctx, principal.userID, payload.Slug, projectID)
	if err != nil {
		return nil, nil, intakeInboxReadsRepositoryError(err, "")
	}
	return intakeInboxReadCloser(data, principal.timezone, intakeInboxContainerShape)
}

func (service *intakeInboxReadsService) InboxDetail(ctx context.Context, payload *api.InboxDetailPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, intakeID, err := service.detailPrincipal(ctx, payload.ProjectID, payload.Pk)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.InboxDetail(ctx, principal.userID, payload.Slug, projectID, intakeID)
	if err != nil {
		return nil, nil, intakeInboxReadsDetailError(err)
	}
	return intakeInboxReadCloser(data, principal.timezone, intakeInboxContainerShape)
}

func (service *intakeInboxReadsService) IntakeIssueList(ctx context.Context, payload *api.IntakeIssueListPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, err := service.projectPrincipal(ctx, payload.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	options := withIntakeListQueryPresence(ctx, intakeIssueListOptions(
		principal.timezone, payload.Cursor, payload.PerPage, payload.OrderBy, payload.Status, payload.State, payload.StateGroup, payload.EstimatePoint,
		payload.Priority, payload.Parent, payload.Labels, payload.Assignees, payload.Mentions, payload.CreatedBy, payload.LoggedBy,
		payload.Name, payload.CreatedAt, payload.UpdatedAt, payload.StartDate, payload.TargetDate, payload.CompletedAt, payload.Type,
		payload.Project, payload.Cycle, payload.Module, payload.IntakeStatus, payload.InboxStatus, payload.SubIssue, payload.Subscriber,
		payload.StartTargetDate,
	))
	data, err := service.repository.IntakeIssueList(ctx, principal.userID, payload.Slug, projectID, options)
	if err != nil {
		return nil, nil, intakeInboxReadsRepositoryError(err, "Intake not found")
	}
	return intakeInboxReadCloser(data, principal.timezone, intakeInboxIssuePageShape)
}

func (service *intakeInboxReadsService) IntakeIssueDetail(ctx context.Context, payload *api.IntakeIssueDetailPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, issueID, err := service.detailPrincipal(ctx, payload.ProjectID, payload.Pk)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.IntakeIssueDetail(ctx, principal.userID, payload.Slug, projectID, issueID)
	if err != nil {
		return nil, nil, intakeInboxReadsRepositoryError(err, "The required object does not exist.")
	}
	return intakeInboxReadCloser(data, principal.timezone, intakeInboxIssueDetailShape)
}

func (service *intakeInboxReadsService) DescriptionVersionList(ctx context.Context, payload *api.DescriptionVersionListPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, issueID, err := service.detailPrincipal(ctx, payload.ProjectID, payload.WorkItemID)
	if err != nil {
		return nil, nil, err
	}
	cursor := payload.Cursor
	if cursor == nil && ReadServiceQueryPresent(ctx, "cursor") {
		empty := ""
		cursor = &empty
	}
	data, err := service.repository.DescriptionVersionList(ctx, principal.userID, payload.Slug, projectID, issueID, cursor)
	if err != nil {
		return nil, nil, intakeInboxReadsRepositoryError(err, "The required object does not exist.")
	}
	return intakeInboxReadCloser(data, principal.timezone, intakeInboxDescriptionPageShape)
}

func (service *intakeInboxReadsService) DescriptionVersionDetail(ctx context.Context, payload *api.DescriptionVersionDetailPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, issueID, err := service.detailPrincipal(ctx, payload.ProjectID, payload.WorkItemID)
	if err != nil {
		return nil, nil, err
	}
	versionID, err := uuid.Parse(payload.Pk)
	if err != nil {
		return nil, nil, &api.IntakeInboxReadsNotFound{Message: "The required object does not exist."}
	}
	data, err := service.repository.DescriptionVersionDetail(ctx, principal.userID, payload.Slug, projectID, issueID, versionID)
	if err != nil {
		return nil, nil, intakeInboxReadsRepositoryError(err, "The required object does not exist.")
	}
	return intakeInboxReadCloser(data, principal.timezone, intakeInboxDescriptionDetailShape)
}

func (service *intakeInboxReadsService) DescriptionVersionListOptions(ctx context.Context, _ *api.DescriptionVersionListOptionsPayload) (*api.IntakeInboxReadsMetadata, error) {
	if _, err := service.principal(ctx); err != nil {
		return nil, err
	}
	return newIntakeInboxReadsMetadata(), nil
}

func (service *intakeInboxReadsService) DescriptionVersionDetailOptions(ctx context.Context, _ *api.DescriptionVersionDetailOptionsPayload) (*api.IntakeInboxReadsMetadata, error) {
	if _, err := service.principal(ctx); err != nil {
		return nil, err
	}
	return newIntakeInboxReadsMetadata(), nil
}

func (service *intakeInboxReadsService) IntakeList(ctx context.Context, payload *api.IntakeListPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, err := service.projectPrincipal(ctx, payload.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.IntakeList(ctx, principal.userID, payload.Slug, projectID)
	if err != nil {
		return nil, nil, intakeInboxReadsRepositoryError(err, "")
	}
	return intakeInboxReadCloser(data, principal.timezone, intakeInboxContainerShape)
}

func (service *intakeInboxReadsService) IntakeDetail(ctx context.Context, payload *api.IntakeDetailPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	principal, projectID, intakeID, err := service.detailPrincipal(ctx, payload.ProjectID, payload.Pk)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.IntakeDetail(ctx, principal.userID, payload.Slug, projectID, intakeID)
	if err != nil {
		return nil, nil, intakeInboxReadsDetailError(err)
	}
	return intakeInboxReadCloser(data, principal.timezone, intakeInboxContainerShape)
}

func (service *intakeInboxReadsService) principal(ctx context.Context) (intakeInboxReadsPrincipal, error) {
	principal, ok := ctx.Value(intakeInboxReadsPrincipal{}).(intakeInboxReadsPrincipal)
	if !ok {
		return intakeInboxReadsPrincipal{}, &api.IntakeInboxReadsUnauthenticated{Detail: unauthenticatedDetail}
	}
	return principal, nil
}

func (service *intakeInboxReadsService) projectPrincipal(ctx context.Context, project string) (intakeInboxReadsPrincipal, uuid.UUID, error) {
	principal, err := service.principal(ctx)
	if err != nil {
		return intakeInboxReadsPrincipal{}, uuid.Nil(), err
	}
	projectID, err := uuid.Parse(project)
	if err != nil {
		return intakeInboxReadsPrincipal{}, uuid.Nil(), &api.IntakeInboxReadsPermissionDenied{Message: "You don't have the required permissions."}
	}
	return principal, projectID, nil
}

func (service *intakeInboxReadsService) detailPrincipal(ctx context.Context, project, detail string) (intakeInboxReadsPrincipal, uuid.UUID, uuid.UUID, error) {
	principal, projectID, err := service.projectPrincipal(ctx, project)
	if err != nil {
		return intakeInboxReadsPrincipal{}, uuid.Nil(), uuid.Nil(), err
	}
	detailID, err := uuid.Parse(detail)
	if err != nil {
		return intakeInboxReadsPrincipal{}, uuid.Nil(), uuid.Nil(), &api.IntakeInboxReadsNotFound{Message: "The required object does not exist."}
	}
	return principal, projectID, detailID, nil
}

func intakeInboxReadCloser(data jsontext.Value, timezone string, shape intakeInboxResponseShape) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	normalized, err := intakeInboxTimes(data, timezone, shape)
	if err != nil {
		return nil, nil, &api.IntakeInboxReadsInternalError{Message: intakeInboxReadsInternalErrorDetail}
	}
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser(normalized), nil
}

func intakeInboxReadsRepositoryError(err error, notFound string) error {
	if errors.Is(err, store.ErrGuestRestricted) {
		return &api.IntakeInboxReadsPermissionDenied{Message: "You are not allowed to view this issue"}
	}
	if errors.Is(err, store.ErrForbidden) {
		return &api.IntakeInboxReadsPermissionDenied{Message: "You don't have the required permissions."}
	}
	if errors.Is(err, store.ErrInvalidDetail) {
		return &api.IntakeInboxReadsInvalidDetail{Message: "Please provide valid detail"}
	}
	if errors.Is(err, store.ErrInvalidPerPage) {
		return &api.IntakeInboxReadsBadRequest{Detail: "Invalid per_page parameter."}
	}
	if errors.Is(err, store.ErrPerPageTooLarge) {
		return &api.IntakeInboxReadsBadRequest{Detail: "Invalid per_page value. Cannot exceed 1000."}
	}
	if errors.Is(err, store.ErrInvalidCursor) {
		return &api.IntakeInboxReadsBadRequest{Detail: "Invalid cursor parameter."}
	}
	if errors.Is(err, store.ErrPaginationParsing) {
		return &api.IntakeInboxReadsBadRequest{Detail: "Error in parsing"}
	}
	if errors.Is(err, store.ErrBadRequest) {
		return &api.IntakeInboxReadsBadRequest{Detail: "Invalid pagination or filter parameter."}
	}
	if errors.Is(err, store.ErrNotFound) && notFound != "" {
		return &api.IntakeInboxReadsNotFound{Message: notFound}
	}
	return &api.IntakeInboxReadsInternalError{Message: intakeInboxReadsInternalErrorDetail}
}

func intakeInboxReadsDetailError(err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return &api.IntakeInboxReadsDetailNotFound{Detail: "No Intake matches the given query."}
	}
	return intakeInboxReadsRepositoryError(err, "")
}

func draftListOptions(payload *api.DraftIssueListPayload, timezone string) store.ListOptions {
	return intakeIssueListOptions(timezone, payload.Cursor, payload.PerPage, nil, nil, payload.State, payload.StateGroup, payload.EstimatePoint,
		payload.Priority, payload.Parent, payload.Labels, payload.Assignees, payload.Mentions, payload.CreatedBy, payload.LoggedBy,
		payload.Name, payload.CreatedAt, payload.UpdatedAt, payload.StartDate, payload.TargetDate, payload.CompletedAt, payload.Type,
		payload.Project, payload.Cycle, payload.Module, payload.IntakeStatus, payload.InboxStatus, payload.SubIssue, payload.Subscriber,
		payload.StartTargetDate)
}

func intakeIssueListOptions(timezone string, cursor, perPage, orderBy, status, state, stateGroup, estimatePoint, priority, parent, labels, assignees, mentions, createdBy, loggedBy, name, createdAt, updatedAt, startDate, targetDate, completedAt, issueType, project, cycle, module, intakeStatus, inboxStatus, subIssue, subscriber, startTargetDate *string) store.ListOptions {
	filters := map[string]string{
		"status": stringValue(status), "state": stringValue(state), "state_group": stringValue(stateGroup),
		"estimate_point": stringValue(estimatePoint), "priority": stringValue(priority), "parent": stringValue(parent),
		"labels": stringValue(labels), "assignees": stringValue(assignees), "mentions": stringValue(mentions),
		"created_by": stringValue(createdBy), "logged_by": stringValue(loggedBy), "name": stringValue(name),
		"created_at": stringValue(createdAt), "updated_at": stringValue(updatedAt), "start_date": stringValue(startDate),
		"target_date": stringValue(targetDate), "completed_at": stringValue(completedAt), "type": stringValue(issueType),
		"project": stringValue(project), "cycle": stringValue(cycle), "module": stringValue(module),
		"intake_status": stringValue(intakeStatus), "inbox_status": stringValue(inboxStatus), "sub_issue": stringValue(subIssue),
		"subscriber": stringValue(subscriber), "start_target_date": stringValue(startTargetDate),
	}
	return store.ListOptions{
		Cursor: stringValue(cursor), CursorSet: cursor != nil,
		PerPage: stringValue(perPage), PerPageSet: perPage != nil,
		OrderBy: stringValue(orderBy), Timezone: timezone, Filters: filters, StatusSet: status != nil,
		PresentFilters: map[string]bool{
			"labels": labels != nil, "assignees": assignees != nil, "cycle": cycle != nil, "module": module != nil,
			"subscriber": subscriber != nil, "mentions": mentions != nil, "logged_by": loggedBy != nil,
			"intake_status": intakeStatus != nil, "inbox_status": inboxStatus != nil, "type": issueType != nil,
		},
	}
}

func withIntakeListQueryPresence(ctx context.Context, options store.ListOptions) store.ListOptions {
	options.CursorSet = options.CursorSet || ReadServiceQueryPresent(ctx, "cursor")
	options.PerPageSet = options.PerPageSet || ReadServiceQueryPresent(ctx, "per_page")
	options.StatusSet = options.StatusSet || ReadServiceQueryPresent(ctx, "status")
	if options.PresentFilters == nil {
		options.PresentFilters = make(map[string]bool)
	}
	for _, name := range []string{
		"labels", "assignees", "cycle", "module", "subscriber",
		"mentions", "logged_by", "intake_status", "inbox_status", "type",
	} {
		options.PresentFilters[name] = options.PresentFilters[name] || ReadServiceQueryPresent(ctx, name)
	}
	return options
}

func newIntakeInboxReadsMetadata() *api.IntakeInboxReadsMetadata {
	return &api.IntakeInboxReadsMetadata{
		Name:        "Intake Work Item Description Version Endpoint",
		Description: "",
		Renders:     []string{"application/json"},
		Parses:      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
	}
}
func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

type intakeInboxResponseShape uint8

const (
	intakeInboxDraftPageShape intakeInboxResponseShape = iota
	intakeInboxDraftDetailShape
	intakeInboxIssuePageShape
	intakeInboxIssueDetailShape
	intakeInboxContainerShape
	intakeInboxDescriptionPageShape
	intakeInboxDescriptionDetailShape
)

func intakeInboxTimes(data jsontext.Value, timezone string, shape intakeInboxResponseShape) (jsontext.Value, error) {
	if timezone == "" {
		timezone = "UTC"
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	convertIntakeInboxTimes(value, location, shape)
	return json.Marshal(value)
}

func convertIntakeInboxTimes(value any, location *time.Location, shape intakeInboxResponseShape) {
	root, ok := value.(map[string]any)
	if !ok {
		return
	}
	switch shape {
	case intakeInboxDraftPageShape:
		for _, value := range intakeInboxResponseResults(root) {
			if item, ok := value.(map[string]any); ok {
				convertIntakeInboxDraftTimes(item, location)
			}
		}
	case intakeInboxDraftDetailShape:
		convertIntakeInboxDraftTimes(root, location)
	case intakeInboxIssuePageShape:
		for _, value := range intakeInboxResponseResults(root) {
			if item, ok := value.(map[string]any); ok {
				convertIntakeInboxIssuePageItemTimes(item, location)
			}
		}
	case intakeInboxIssueDetailShape:
		convertIntakeInboxIssueDetailTimes(root, location)
	case intakeInboxContainerShape:
		convertIntakeInboxTimeFields(root, location, "created_at", "updated_at", "deleted_at")
	case intakeInboxDescriptionPageShape:
		for _, value := range intakeInboxResponseResults(root) {
			if item, ok := value.(map[string]any); ok {
				convertIntakeInboxTimeFields(item, location, "created_at", "updated_at")
				convertIntakeInboxTimeField(item, "last_saved_at", time.UTC)
			}
		}
	case intakeInboxDescriptionDetailShape:
		convertIntakeInboxTimeFields(root, location, "created_at", "updated_at", "last_saved_at")
	}
}

func intakeInboxResponseResults(root map[string]any) []any {
	results, _ := root["results"].([]any)
	return results
}

func convertIntakeInboxDraftTimes(item map[string]any, location *time.Location) {
	convertIntakeInboxTimeFields(item, location, "created_at", "updated_at", "completed_at")
}
func convertIntakeInboxIssuePageItemTimes(item map[string]any, location *time.Location) {
	convertIntakeInboxTimeField(item, "snoozed_till", location)
	if issue, ok := item["issue"].(map[string]any); ok {
		convertIntakeInboxTimeFields(issue, location, "created_at", "updated_at", "completed_at")
	}
}

func formatDjangoDateTime(value time.Time) string {
	layout := "2006-01-02T15:04:05Z07:00"
	if value.Nanosecond() != 0 {
		layout = "2006-01-02T15:04:05.000000Z07:00"
	}
	return value.Format(layout)
}

func convertIntakeInboxIssueDetailTimes(item map[string]any, location *time.Location) {
	convertIntakeInboxIssuePageItemTimes(item, location)
	if duplicate, ok := item["duplicate_issue_detail"].(map[string]any); ok {
		convertIntakeInboxTimeField(duplicate, "created_at", location)
	}
}

func convertIntakeInboxTimeFields(item map[string]any, location *time.Location, keys ...string) {
	for _, key := range keys {
		convertIntakeInboxTimeField(item, key, location)
	}
}

func convertIntakeInboxTimeField(item map[string]any, key string, location *time.Location) {
	text, ok := item[key].(string)
	if !ok {
		return
	}
	if parsed, err := time.Parse(time.RFC3339Nano, text); err == nil {
		item[key] = formatDjangoDateTime(parsed.In(location))
	}
}
