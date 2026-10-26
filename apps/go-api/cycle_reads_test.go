package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/cycle_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/loom/security"
)

const cycleFixture = `{"id":"00000000-0000-4000-8000-000000000001","workspace_id":"00000000-0000-4000-8000-000000000002","project_id":"00000000-0000-4000-8000-000000000003","name":"Sprint 1","description":"","start_date":"2026-09-01T00:00:00-04:00","end_date":"2026-09-14T23:59:00-04:00","owned_by_id":"00000000-0000-4000-8000-000000000004","view_props":{},"sort_order":100,"external_source":null,"external_id":null,"progress_snapshot":{},"logo_props":{},"is_favorite":false,"total_issues":0,"completed_issues":0,"assignee_ids":[],"status":"CURRENT","version":1,"created_by":null}`

const cycleIssueFixture = `{"id":"00000000-0000-4000-8000-000000000011","workspace":"00000000-0000-4000-8000-000000000002","project":"00000000-0000-4000-8000-000000000003","cycle":"00000000-0000-4000-8000-000000000001","issue":"00000000-0000-4000-8000-000000000012","created_at":"2026-09-01T12:00:00Z","updated_at":"2026-09-01T12:00:00Z","created_by":null,"updated_by":null,"deleted_at":null,"sub_issues_count":0,"issue_detail":{"id":"00000000-0000-4000-8000-000000000012"}}`

const cycleIssuePageFixture = `{"grouped_by":null,"sub_grouped_by":null,"total_count":0,"next_cursor":"1000:1:0","prev_cursor":"1000:-1:1","next_page_results":false,"prev_page_results":false,"count":0,"total_pages":0,"total_results":0,"extra_stats":null,"results":[]}`

const cycleProgressFixture = `{"backlog_estimate_points":0,"unstarted_estimate_points":0,"started_estimate_points":0,"cancelled_estimate_points":0,"completed_estimate_points":0,"total_estimate_points":0,"backlog_issues":0,"unstarted_issues":0,"started_issues":0,"cancelled_issues":0,"completed_issues":0,"total_issues":0}`

const cycleAnalyticsFixture = `{"assignees":[],"labels":[],"completion_chart":{"2026-09-01":2,"2026-09-02":null}}`

type cycleReadsRepositoryStub struct {
	called        []string
	data          json.RawMessage
	err           error
	options       store.IssueListOptions
	analyticsType string
}

func (stub *cycleReadsRepositoryStub) result(name string) (json.RawMessage, error) {
	stub.called = append(stub.called, name)
	return stub.data, stub.err
}

func (stub *cycleReadsRepositoryStub) AuthorizeWorkspace(context.Context, uuid.UUID, string) error {
	return stub.err
}
func (stub *cycleReadsRepositoryStub) WorkspaceList(context.Context, uuid.UUID, string) (json.RawMessage, error) {
	return stub.result("workspace_list")
}

func (stub *cycleReadsRepositoryStub) ArchivedList(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error) {
	return stub.result("archived_list")
}

func (stub *cycleReadsRepositoryStub) ArchivedDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error) {
	return stub.result("archived_detail")
}

func (stub *cycleReadsRepositoryStub) List(context.Context, uuid.UUID, string, uuid.UUID, string) (json.RawMessage, error) {
	return stub.result("list")
}

func (stub *cycleReadsRepositoryStub) Analytics(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ uuid.UUID, analyticsType string) (json.RawMessage, error) {
	stub.analyticsType = analyticsType
	return stub.result("analytics")
}

func (stub *cycleReadsRepositoryStub) ArchiveDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error) {
	return stub.result("archive_detail")
}

func (stub *cycleReadsRepositoryStub) Issues(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ uuid.UUID, options store.IssueListOptions) (json.RawMessage, error) {
	stub.options = options
	return stub.result("issues")
}

func (stub *cycleReadsRepositoryStub) IssueDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, uuid.UUID) (json.RawMessage, error) {
	return stub.result("issue_detail")
}

func (stub *cycleReadsRepositoryStub) Progress(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error) {
	return stub.result("progress")
}

func (stub *cycleReadsRepositoryStub) Detail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error) {
	return stub.result("detail")
}

type cycleReadsSessionsStub struct {
	principal djangosession.Principal
	err       error
}

func (stub cycleReadsSessionsStub) Validate(context.Context, string) (djangosession.Principal, error) {
	if stub.err != nil {
		return djangosession.Principal{}, stub.err
	}
	if stub.principal.UserID == "" {
		return djangosession.Principal{UserID: "00000000-0000-4000-8000-000000000020"}, nil
	}
	return stub.principal, nil
}

type cycleRecentVisitsStub struct {
	calls      int
	userID     uuid.UUID
	slug       string
	entityName string
	entityID   uuid.UUID
	projectID  uuid.UUID
	err        error
}

func (stub *cycleRecentVisitsStub) Record(
	_ context.Context,
	userID uuid.UUID,
	slug string,
	entityName string,
	entityID uuid.UUID,
	projectID uuid.UUID,
) error {
	stub.calls++
	stub.userID = userID
	stub.slug = slug
	stub.entityName = entityName
	stub.entityID = entityID
	stub.projectID = projectID
	return stub.err
}

func TestNewCycleReadsRequiresAllDependencies(t *testing.T) {
	repository := &cycleReadsRepositoryStub{}
	sessions := cycleReadsSessionsStub{}
	visits := &cycleRecentVisitsStub{}
	if _, err := NewCycleReads(nil, sessions, visits); err == nil {
		t.Fatal("missing repository returned nil error")
	}
	if _, err := NewCycleReads(repository, nil, visits); err == nil {
		t.Fatal("missing sessions returned nil error")
	}
	if _, err := NewCycleReads(repository, sessions, nil); err == nil {
		t.Fatal("missing recent visits returned nil error")
	}
}

func TestCycleReadsRejectInvalidSession(t *testing.T) {
	service := &cycleReadsService{sessions: cycleReadsSessionsStub{err: errors.New("invalid")}}
	_, err := service.APIKeyAuth(context.Background(), "bad", &security.APIKeyScheme{})
	var unauthenticated *api.CycleReadsUnauthenticated
	if !errors.As(err, &unauthenticated) {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestCycleReadsReusesTransportSessionPrincipal(t *testing.T) {
	service := &cycleReadsService{sessions: cycleReadsSessionsStub{err: errors.New("must not revalidate")}}
	userID := uuid.New()
	ctx := WithReadServiceSessionPrincipal(context.Background(), userID.String(), "Asia/Kolkata")
	authenticated, err := service.APIKeyAuth(ctx, "already-validated", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := authenticated.Value(cycleReadsPrincipal{}).(uuid.UUID); !ok || got != userID {
		t.Fatalf("principal=%v ok=%v", got, ok)
	}
}

func TestCycleReadsTenMethodsUseDistinctRepositoryPaths(t *testing.T) {
	repository := &cycleReadsRepositoryStub{}
	visits := &cycleRecentVisitsStub{}
	serviceValue, _ := NewCycleReads(repository, cycleReadsSessionsStub{}, visits)
	service := serviceValue.(*cycleReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "session", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	projectID := "00000000-0000-4000-8000-000000000003"
	cycleID := "00000000-0000-4000-8000-000000000001"
	issueID := "00000000-0000-4000-8000-000000000012"

	repository.data = json.RawMessage(`[` + cycleFixture + `]`)
	if _, body, err := service.WorkspaceList(ctx, &api.WorkspaceListPayload{Slug: "alpha"}); err != nil {
		t.Fatal(err)
	} else {
		body.Close()
	}
	if _, body, err := service.ArchivedList(ctx, &api.ArchivedListPayload{Slug: "alpha", ProjectID: projectID}); err != nil {
		t.Fatal(err)
	} else {
		body.Close()
	}
	repository.data = json.RawMessage(cycleFixture)
	if _, body, err := service.ArchivedDetail(ctx, &api.ArchivedDetailPayload{Slug: "alpha", ProjectID: projectID, Pk: cycleID}); err != nil {
		t.Fatal(err)
	} else {
		body.Close()
	}
	repository.data = json.RawMessage(`[` + cycleFixture + `]`)
	if _, body, err := service.List(ctx, &api.ListPayload{Slug: "alpha", ProjectID: projectID}); err != nil {
		t.Fatal(err)
	} else {
		body.Close()
	}
	repository.data = json.RawMessage(cycleAnalyticsFixture)
	if _, body, err := service.Analytics(ctx, &api.AnalyticsPayload{Slug: "alpha", ProjectID: projectID, CycleID: cycleID}); err != nil {
		t.Fatal(err)
	} else {
		body.Close()
	}
	repository.data = json.RawMessage(cycleFixture)
	if _, body, err := service.ArchiveDetail(ctx, &api.ArchiveDetailPayload{Slug: "alpha", ProjectID: projectID, CycleID: cycleID}); err != nil {
		t.Fatal(err)
	} else {
		body.Close()
	}
	repository.data = json.RawMessage(cycleIssuePageFixture)
	if _, body, err := service.Issues(ctx, &api.IssuesPayload{Slug: "alpha", ProjectID: projectID, CycleID: cycleID}); err != nil {
		t.Fatal(err)
	} else {
		body.Close()
	}
	repository.data = json.RawMessage(cycleIssueFixture)
	if _, body, err := service.IssueDetail(ctx, &api.IssueDetailPayload{Slug: "alpha", ProjectID: projectID, CycleID: cycleID, IssueID: issueID}); err != nil {
		t.Fatal(err)
	} else {
		body.Close()
	}
	repository.data = json.RawMessage(cycleProgressFixture)
	if _, body, err := service.Progress(ctx, &api.ProgressPayload{Slug: "alpha", ProjectID: projectID, CycleID: cycleID}); err != nil {
		t.Fatal(err)
	} else {
		body.Close()
	}
	repository.data = json.RawMessage(cycleFixture)
	if _, body, err := service.Detail(ctx, &api.DetailPayload{Slug: "alpha", ProjectID: projectID, Pk: cycleID}); err != nil {
		t.Fatal(err)
	} else {
		body.Close()
	}

	want := []string{"workspace_list", "archived_list", "archived_detail", "list", "analytics", "archive_detail", "issues", "issue_detail", "progress", "detail"}
	if len(repository.called) != len(want) {
		t.Fatalf("called=%v", repository.called)
	}
	for index := range want {
		if repository.called[index] != want[index] {
			t.Fatalf("called=%v, want %v", repository.called, want)
		}
	}
}

func TestCycleAnalyticsPreservesNullablePointKeysAndSnapshotIdentifiers(t *testing.T) {
	fixture := `{"assignees":[{"display_name":null,"assignee_id":"None","avatar_url":null,"total_estimates":null,"completed_estimates":null,"pending_estimates":null}],"labels":[{"label_name":null,"color":null,"label_id":"None","total_estimates":null,"completed_estimates":null,"pending_estimates":null}],"completion_chart":{"2026-09-01":null}}`
	repository := &cycleReadsRepositoryStub{data: json.RawMessage(fixture)}
	service := &cycleReadsService{repository: repository}
	ctx := context.WithValue(context.Background(), cycleReadsPrincipal{}, uuid.New())
	metadata, result, err := service.Analytics(ctx, &api.AnalyticsPayload{
		Slug: "alpha", ProjectID: uuid.New().String(), CycleID: uuid.New().String(),
	})
	if err != nil || result == nil {
		t.Fatalf("result=%v err=%v", result, err)
	}
	if metadata == nil || metadata.ContentType != "application/json" {
		t.Fatalf("metadata=%+v", metadata)
	}
	defer result.Close()
	body, err := io.ReadAll(result)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != fixture {
		t.Fatalf("body=%s", body)
	}
	if repository.analyticsType != "issues" {
		t.Fatalf("omitted analytics type=%q", repository.analyticsType)
	}
}

func TestCycleAnalyticsPreservesExplicitEmptyType(t *testing.T) {
	repository := &cycleReadsRepositoryStub{data: json.RawMessage(`{"assignees":[],"labels":[],"completion_chart":{}}`)}
	service := &cycleReadsService{repository: repository}
	ctx := context.WithValue(context.Background(), cycleReadsPrincipal{}, uuid.New())
	ctx = WithReadServiceQueryPresence(ctx, url.Values{"type": {""}})
	_, body, err := service.Analytics(ctx, &api.AnalyticsPayload{
		Slug: "alpha", ProjectID: uuid.New().String(), CycleID: uuid.New().String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	if repository.analyticsType != "" {
		t.Fatalf("explicit empty analytics type=%q", repository.analyticsType)
	}
}

func TestCycleDetailRecordsRecentVisitBestEffortAfterSuccess(t *testing.T) {
	repository := &cycleReadsRepositoryStub{data: json.RawMessage(cycleFixture)}
	visits := &cycleRecentVisitsStub{err: errors.New("write failed")}
	service := &cycleReadsService{repository: repository, recentVisits: visits}
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000020")
	ctx := context.WithValue(context.Background(), cycleReadsPrincipal{}, userID)
	metadata, cycle, err := service.Detail(ctx, &api.DetailPayload{
		Slug: "alpha", ProjectID: "00000000-0000-4000-8000-000000000003", Pk: "00000000-0000-4000-8000-000000000001",
	})
	if err != nil || cycle == nil {
		t.Fatalf("cycle=%v err=%v", cycle, err)
	}
	if metadata == nil || metadata.ContentType != "application/json" {
		t.Fatalf("metadata=%+v", metadata)
	}
	defer cycle.Close()
	body, err := io.ReadAll(cycle)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != cycleFixture {
		t.Fatalf("body=%s", body)
	}
	if visits.calls != 1 || visits.userID != userID || visits.slug != "alpha" || visits.entityName != "cycle" || visits.entityID.String() != "00000000-0000-4000-8000-000000000001" || visits.projectID.String() != "00000000-0000-4000-8000-000000000003" {
		t.Fatalf("recent visit=%+v", visits)
	}
}

func TestCycleDetailDoesNotRecordVisitOnLookupFailure(t *testing.T) {
	repository := &cycleReadsRepositoryStub{err: store.ErrNotFound}
	visits := &cycleRecentVisitsStub{}
	service := &cycleReadsService{repository: repository, recentVisits: visits}
	ctx := context.WithValue(context.Background(), cycleReadsPrincipal{}, uuid.New())
	_, _, err := service.Detail(ctx, &api.DetailPayload{ProjectID: uuid.New().String(), Pk: uuid.New().String()})
	var notFound *api.CycleReadsNotFound
	if !errors.As(err, &notFound) || visits.calls != 0 {
		t.Fatalf("err=%T visits=%d", err, visits.calls)
	}
}

func TestCorrectedMissingArchivedDetailAndAnalyticsReturnNotFound(t *testing.T) {
	repository := &cycleReadsRepositoryStub{err: store.ErrNotFound}
	service := &cycleReadsService{repository: repository}
	ctx := context.WithValue(context.Background(), cycleReadsPrincipal{}, uuid.New())
	projectID, cycleID := uuid.New().String(), uuid.New().String()
	_, _, archivedErr := service.ArchivedDetail(ctx, &api.ArchivedDetailPayload{
		Slug: "alpha", ProjectID: projectID, Pk: cycleID,
	})
	_, _, analyticsErr := service.Analytics(ctx, &api.AnalyticsPayload{
		Slug: "alpha", ProjectID: projectID, CycleID: cycleID,
	})
	for _, err := range []error{archivedErr, analyticsErr} {
		var notFound *api.CycleReadsNotFound
		if !errors.As(err, &notFound) || notFound.Message != "Cycle not found" {
			t.Fatalf("err=%T %v", err, err)
		}
	}
}

func TestWorkspaceListUsesDRFPermissionBody(t *testing.T) {
	repository := &cycleReadsRepositoryStub{err: store.ErrForbidden}
	service := &cycleReadsService{repository: repository}
	ctx := context.WithValue(context.Background(), cycleReadsPrincipal{}, uuid.New())
	_, _, err := service.WorkspaceList(ctx, &api.WorkspaceListPayload{Slug: "alpha"})
	var forbidden *api.CycleReadsWorkspaceForbidden
	if !errors.As(err, &forbidden) {
		t.Fatalf("err=%T", err)
	}
	if forbidden.Detail != "You do not have permission to perform this action." {
		t.Fatalf("detail=%q", forbidden.Detail)
	}
}

func TestCycleOptionsPreserveDjangoMetadataAndPermissions(t *testing.T) {
	repository := &cycleReadsRepositoryStub{}
	service := &cycleReadsService{repository: repository}
	ctx := context.WithValue(context.Background(), cycleReadsPrincipal{}, uuid.New())

	workspace, err := service.WorkspaceListOptions(ctx, &api.WorkspaceListOptionsPayload{Slug: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	assertCycleMetadata(t, workspace, "Workspace Cycles Endpoint")

	analytics, err := service.AnalyticsOptions(ctx, &api.AnalyticsOptionsPayload{})
	if err != nil {
		t.Fatal(err)
	}
	assertCycleMetadata(t, analytics, "Cycle Analytics Endpoint")

	progress, err := service.ProgressOptions(ctx, &api.ProgressOptionsPayload{})
	if err != nil {
		t.Fatal(err)
	}
	assertCycleMetadata(t, progress, "Cycle Progress Endpoint")

	repository.err = store.ErrForbidden
	_, err = service.WorkspaceListOptions(ctx, &api.WorkspaceListOptionsPayload{Slug: "alpha"})
	var forbidden *api.CycleReadsWorkspaceForbidden
	if !errors.As(err, &forbidden) || forbidden.Detail != "You do not have permission to perform this action." {
		t.Fatalf("workspace OPTIONS err=%T %v", err, err)
	}
}

func assertCycleMetadata(t *testing.T, metadata *api.CycleReadsMetadata, name string) {
	t.Helper()
	if metadata.Name != name || metadata.Description != "" {
		t.Fatalf("metadata=%+v", metadata)
	}
	if len(metadata.Renders) != 1 || metadata.Renders[0] != "application/json" {
		t.Fatalf("renders=%v", metadata.Renders)
	}
	wantParses := []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}
	if len(metadata.Parses) != len(wantParses) {
		t.Fatalf("parses=%v", metadata.Parses)
	}
	for index := range wantParses {
		if metadata.Parses[index] != wantParses[index] {
			t.Fatalf("parses=%v", metadata.Parses)
		}
	}
}
func TestCycleIssueInvalidPageSizesMarkDjangoHTML500(t *testing.T) {
	for _, repositoryError := range []error{
		store.ErrZeroPageSize,
		store.ErrNegativePageSize,
		store.ErrPreviousPageSlice,
		store.ErrPaginationOverflow,
		store.ErrDynamicGrouping,
	} {
		repository := &cycleReadsRepositoryStub{err: repositoryError}
		service := &cycleReadsService{repository: repository}
		ctx := context.WithValue(context.Background(), cycleReadsPrincipal{}, uuid.New())
		ctx = WithReadServiceDjangoHTML500(ctx)
		_, _, err := service.Issues(ctx, &api.IssuesPayload{
			Slug: "alpha", ProjectID: uuid.New().String(), CycleID: uuid.New().String(),
		})
		var internal *api.CycleReadsInternalError
		if !errors.As(err, &internal) {
			t.Fatalf("%v: err=%T %v", repositoryError, err, err)
		}
		if !ReadServiceDjangoHTML500(ctx) {
			t.Fatalf("%v did not mark the Django HTML 500 response", repositoryError)
		}
	}
}

func TestCycleReadsMapPermissionNotFoundDatesGroupingAndPaginationErrors(t *testing.T) {
	cases := []struct {
		err  error
		want any
	}{
		{store.ErrForbidden, (*api.CycleReadsForbidden)(nil)},
		{store.ErrNotFound, (*api.CycleReadsNotFound)(nil)},
		{store.ErrCycleDatesMissing, (*api.CycleReadsBadRequest)(nil)},
		{store.ErrInvalidGrouping, (*api.CycleReadsBadRequest)(nil)},
		{store.ErrMissingGroupKey, (*api.CycleReadsBadRequest)(nil)},
		{store.ErrInvalidPagination, (*api.CycleReadsInvalidPagination)(nil)},
		{store.ErrPaginationParsing, (*api.CycleReadsParseError)(nil)},
		{store.ErrZeroPageSize, (*api.CycleReadsInternalError)(nil)},
		{store.ErrNegativePageSize, (*api.CycleReadsInternalError)(nil)},
		{store.ErrPreviousPageSlice, (*api.CycleReadsInternalError)(nil)},
		{store.ErrPaginationOverflow, (*api.CycleReadsInternalError)(nil)},
		{store.ErrDynamicGrouping, (*api.CycleReadsInternalError)(nil)},
		{&store.InvalidGroupingFieldError{Field: "group_by", Value: "bogus"}, (*api.CycleReadsInvalidGrouping)(nil)},
		{&store.FilterError{Message: "Invalid filter parameters", Code: "invalid_filterset", Errors: map[string]any{"state_id": []string{"Enter a valid UUID."}}}, (*api.CycleReadsFilterError)(nil)},
	}
	for _, test := range cases {
		mapped := cycleReadsRepositoryError(test.err)
		switch test.want.(type) {
		case *api.CycleReadsForbidden:
			var target *api.CycleReadsForbidden
			if !errors.As(mapped, &target) {
				t.Fatalf("err=%T", mapped)
			}
		case *api.CycleReadsNotFound:
			var target *api.CycleReadsNotFound
			if !errors.As(mapped, &target) {
				t.Fatalf("err=%T", mapped)
			}
		case *api.CycleReadsBadRequest:
			var target *api.CycleReadsBadRequest
			if !errors.As(mapped, &target) {
				t.Fatalf("err=%T", mapped)
			}
			if errors.Is(test.err, store.ErrMissingGroupKey) && cycleOptionalString(target.ErrorMessage) != "The required key does not exist." {
				t.Fatalf("message=%q", cycleOptionalString(target.ErrorMessage))
			}
		case *api.CycleReadsInvalidPagination:
			var target *api.CycleReadsInvalidPagination
			if !errors.As(mapped, &target) {
				t.Fatalf("err=%T", mapped)
			}
		case *api.CycleReadsInternalError:
			var target *api.CycleReadsInternalError
			if !errors.As(mapped, &target) {
				t.Fatalf("err=%T", mapped)
			}
		case *api.CycleReadsParseError:
			var target *api.CycleReadsParseError
			if !errors.As(mapped, &target) || cycleOptionalString(target.Detail) != "Error in parsing" {
				t.Fatalf("err=%T detail=%q", mapped, cycleOptionalString(target.Detail))
			}
		case *api.CycleReadsInvalidGrouping:
			var target *api.CycleReadsInvalidGrouping
			if !errors.As(mapped, &target) || cycleOptionalString(target.Detail) != "Invalid group_by field: bogus" {
				t.Fatalf("err=%T detail=%q", mapped, cycleOptionalString(target.Detail))
			}
		case *api.CycleReadsFilterError:
			var target *api.CycleReadsFilterError
			if !errors.As(mapped, &target) || cycleOptionalString(target.Code) != "invalid_filterset" {
				t.Fatalf("err=%T code=%q", mapped, cycleOptionalString(target.Code))
			}
		}
	}
}

func TestCycleIssueOptionsPreserveLegacyAndComplexFilters(t *testing.T) {
	order, group, subGroup, cursor, perPage := "-priority", "state__group", "priority", "25:2:0", "25"
	priority := "urgent,high"
	complexFilter := `{"or":[{"priority":"urgent"},{"state_group":"started"}]}`
	payload := struct {
		OrderBy    *string `form:"order_by"`
		GroupBy    *string `form:"group_by"`
		SubGroupBy *string `form:"sub_group_by"`
		Cursor     *string `form:"cursor"`
		PerPage    *string `form:"per_page"`
		Priority   *string `form:"priority"`
		Filters    *string `form:"filters"`
	}{&order, &group, &subGroup, &cursor, &perPage, &priority, &complexFilter}
	options, err := cycleIssueOptions(context.Background(), &payload)
	if err != nil {
		t.Fatal(err)
	}
	if options.OrderBy != order || options.GroupBy != group || options.SubGroupBy != subGroup || options.Cursor != cursor || options.PerPage != 25 {
		t.Fatalf("options=%+v", options)
	}
	if options.Filters["priority"] != priority || options.ComplexFilter != complexFilter {
		t.Fatalf("filters=%v complex=%q", options.Filters, options.ComplexFilter)
	}
}

func TestCycleIssueOptionsReturnsExactDjangoPerPageErrors(t *testing.T) {
	for _, test := range []struct {
		perPage string
		detail  string
	}{
		{"many", "Invalid per_page parameter."},
		{"1001", "Invalid per_page value. Cannot exceed 1000."},
		{"99999999999999999999", "Invalid per_page value. Cannot exceed 1000."},
		{strings.Repeat("9", 4301), "Invalid per_page parameter."},
	} {
		payload := struct {
			PerPage *string `form:"per_page"`
		}{&test.perPage}
		_, err := cycleIssueOptions(context.Background(), &payload)
		var invalid *api.CycleReadsInvalidPagination
		if !errors.As(err, &invalid) {
			t.Errorf("per_page=%q err=%T", test.perPage, err)
			continue
		}
		if cycleOptionalString(invalid.Detail) != test.detail {
			t.Errorf("per_page=%q detail=%q", test.perPage, cycleOptionalString(invalid.Detail))
		}
	}
}

func TestCycleIssueOptionsPreserveExplicitEmptyAndZeroQueryValues(t *testing.T) {
	empty := ""
	emptyPerPage := struct {
		PerPage *string `form:"per_page"`
	}{&empty}
	perPageContext := WithReadServiceQueryPresence(
		context.Background(),
		url.Values{"per_page": {""}},
	)
	_, err := cycleIssueOptions(perPageContext, &emptyPerPage)
	var invalid *api.CycleReadsInvalidPagination
	if !errors.As(err, &invalid) || cycleOptionalString(invalid.Detail) != "Invalid per_page parameter." {
		t.Fatalf("empty per_page err=%T %v", err, err)
	}

	emptyCursor := struct {
		Cursor *string `form:"cursor"`
	}{&empty}
	cursorContext := WithReadServiceQueryPresence(
		context.Background(),
		url.Values{"cursor": {""}},
	)
	options, err := cycleIssueOptions(cursorContext, &emptyCursor)
	if err != nil {
		t.Fatal(err)
	}
	if !options.CursorSet || options.Cursor != "" {
		t.Fatalf("empty cursor options=%+v", options)
	}

	zero := "0"
	zeroPerPage := struct {
		PerPage *string `form:"per_page"`
	}{&zero}
	options, err = cycleIssueOptions(context.Background(), &zeroPerPage)
	if err != nil {

		t.Fatal(err)
	}
	if !options.PerPageSet || options.PerPage != 0 {
		t.Fatalf("zero per_page options=%+v", options)
	}

	negative := "-1"
	negativePerPage := struct {
		PerPage *string `form:"per_page"`
	}{&negative}
	options, err = cycleIssueOptions(context.Background(), &negativePerPage)
	if err != nil {
		t.Fatal(err)
	}
	if !options.PerPageSet || options.PerPage != -1 {
		t.Fatalf("negative per_page options=%+v", options)
	}

	spaced := "\u00a02_٥\u00a0"
	spacedPerPage := struct {
		PerPage *string `form:"per_page"`
	}{&spaced}
	options, err = cycleIssueOptions(context.Background(), &spacedPerPage)
	if err != nil {
		t.Fatal(err)
	}
	if !options.PerPageSet || options.PerPage != 25 {
		t.Fatalf("whitespace per_page options=%+v", options)
	}
}
func cycleOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func TestCycleIssuePropertyProjectionRequiresExplicitOptIn(t *testing.T) {
	for _, value := range []string{"", "false", "true"} {
		ctx := WithReadServiceQueryPresence(context.Background(), url.Values{"include_properties": {value}})
		options, err := cycleIssueOptions(ctx, &struct{}{})
		if err != nil {
			t.Fatal(err)
		}
		if options.IncludeProperties != (value == "true") {
			t.Fatalf("projection flag %q = %v", value, options.IncludeProperties)
		}
	}
}
