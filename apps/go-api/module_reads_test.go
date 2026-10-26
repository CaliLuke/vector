package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/module_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/modulereads"
	"github.com/CaliLuke/go-plane/internal/platform"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/CaliLuke/loom/security"
)

const moduleIDFixture = "00000000-0000-4000-8000-000000000001"
const projectIDFixture = "00000000-0000-4000-8000-000000000002"
const workspaceIDFixture = "00000000-0000-4000-8000-000000000003"
const issueIDFixture = "00000000-0000-4000-8000-000000000004"
const linkIDFixture = "00000000-0000-4000-8000-000000000005"
const userIDFixture = "00000000-0000-4000-8000-000000000006"

const moduleFixture = `{"id":"` + moduleIDFixture + `","workspace_id":"` + workspaceIDFixture + `","project_id":"` + projectIDFixture + `","name":"Launch","description":"","description_text":null,"description_html":null,"start_date":null,"target_date":null,"status":"planned","lead_id":null,"member_ids":[],"view_props":{},"sort_order":100.0,"external_source":null,"external_id":null,"logo_props":{},"total_estimate_points":0,"completed_estimate_points":0,"is_favorite":false,"total_issues":1,"cancelled_issues":0,"completed_issues":0,"started_issues":1,"unstarted_issues":0,"backlog_issues":0,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","archived_at":null}`
const archivedModuleFixture = `{"id":"` + moduleIDFixture + `","workspace_id":"` + workspaceIDFixture + `","project_id":"` + projectIDFixture + `","name":"Launch","description":"","description_text":null,"description_html":null,"start_date":null,"target_date":null,"status":"completed","lead_id":null,"member_ids":[],"view_props":{},"sort_order":100.0,"external_source":null,"external_id":null,"is_favorite":false,"total_issues":1,"cancelled_issues":0,"completed_issues":1,"started_issues":0,"unstarted_issues":0,"backlog_issues":0,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","archived_at":"2026-02-01T00:00:00Z"}`
const moduleLinkFixture = `{"id":"` + linkIDFixture + `","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","created_by":null,"updated_by":null,"deleted_at":null,"workspace":"` + workspaceIDFixture + `","project":"` + projectIDFixture + `","module":"` + moduleIDFixture + `","title":null,"url":"https://example.com","metadata":{}}`
const moduleDetailFixture = `{"id":"` + moduleIDFixture + `","workspace_id":"` + workspaceIDFixture + `","project_id":"` + projectIDFixture + `","name":"Launch","description":"","description_text":null,"description_html":null,"start_date":null,"target_date":null,"status":"planned","lead_id":null,"member_ids":[],"view_props":{},"sort_order":100.0,"external_source":null,"external_id":null,"logo_props":{},"total_estimate_points":0,"completed_estimate_points":0,"is_favorite":false,"total_issues":1,"cancelled_issues":0,"completed_issues":0,"started_issues":1,"unstarted_issues":0,"backlog_issues":0,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","archived_at":null,"link_module":[],"sub_issues":0,"backlog_estimate_points":0,"unstarted_estimate_points":0,"started_estimate_points":0,"cancelled_estimate_points":0,"distribution":{"assignees":[],"labels":[],"completion_chart":{}},"estimate_distribution":{}}`
const moduleIssueFixture = `{"id":"00000000-0000-4000-8000-000000000007","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","created_by":null,"updated_by":null,"deleted_at":null,"workspace":"` + workspaceIDFixture + `","project":"` + projectIDFixture + `","module":"` + moduleIDFixture + `","issue":"` + issueIDFixture + `","module_detail":{"id":"` + moduleIDFixture + `","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","created_by":null,"updated_by":null,"deleted_at":null,"workspace":"` + workspaceIDFixture + `","project":"` + projectIDFixture + `","name":"Launch","description":"","description_text":null,"description_html":null,"start_date":null,"target_date":null,"status":"planned","lead":null,"members":[],"view_props":{},"sort_order":100,"external_source":null,"external_id":null,"logo_props":{},"archived_at":null},"issue_detail":{"id":"` + issueIDFixture + `","name":"Fix bug"},"sub_issues_count":0}`
const moduleIssueItemFixture = `{"id":"` + issueIDFixture + `","name":"Fix bug","state_id":null,"state__group":"started","sort_order":100,"completed_at":null,"estimate_point":null,"priority":"high","start_date":null,"target_date":null,"sequence_id":1,"project_id":"` + projectIDFixture + `","parent_id":null,"cycle_id":null,"sub_issues_count":0,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","created_by":null,"updated_by":null,"attachment_count":0,"link_count":0,"is_draft":false,"archived_at":null,"assignee_ids":["` + userIDFixture + `"],"label_ids":[],"module_ids":["` + moduleIDFixture + `"]}`

type moduleReadsRepositoryStub struct {
	err             error
	issueOptions    store.IssueListOptions
	archiveModuleID uuid.UUID
	issueID         uuid.UUID
}

func (stub *moduleReadsRepositoryStub) WorkspaceList(context.Context, uuid.UUID, string) (json.RawMessage, error) {
	return json.RawMessage(`[` + moduleFixture + `]`), stub.err
}
func (stub *moduleReadsRepositoryStub) ArchivedList(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error) {
	return json.RawMessage(`[` + archivedModuleFixture + `]`), stub.err
}
func (stub *moduleReadsRepositoryStub) ArchivedDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error) {
	return json.RawMessage(moduleDetailFixture), stub.err
}
func (stub *moduleReadsRepositoryStub) List(context.Context, uuid.UUID, string, uuid.UUID, bool) (json.RawMessage, error) {
	return json.RawMessage(`[` + moduleFixture + `]`), stub.err
}
func (stub *moduleReadsRepositoryStub) ArchiveDetail(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, moduleID uuid.UUID) (json.RawMessage, error) {
	stub.archiveModuleID = moduleID
	return json.RawMessage(moduleDetailFixture), stub.err
}
func (stub *moduleReadsRepositoryStub) Issues(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ uuid.UUID, options store.IssueListOptions) (store.IssuePage, error) {
	stub.issueOptions = options
	return store.IssuePage{
		Items: json.RawMessage(`[` + moduleIssueItemFixture + `]`), Total: 3,
		GroupValues:    []any{"backlog", "unstarted", "started", "completed", "cancelled"},
		SubGroupValues: []any{userIDFixture},
		PageCount:      1,
	}, stub.err
}
func (stub *moduleReadsRepositoryStub) IssueDetail(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ uuid.UUID, issueID uuid.UUID) (json.RawMessage, error) {
	stub.issueID = issueID
	return json.RawMessage(moduleIssueFixture), stub.err
}
func (stub *moduleReadsRepositoryStub) Links(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error) {
	return json.RawMessage(`[` + moduleLinkFixture + `]`), stub.err
}
func (stub *moduleReadsRepositoryStub) LinkDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, uuid.UUID) (json.RawMessage, error) {
	return json.RawMessage(moduleLinkFixture), stub.err
}
func (stub *moduleReadsRepositoryStub) Detail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error) {
	return json.RawMessage(moduleDetailFixture), stub.err
}

type moduleReadsSessionsStub struct{ err error }

func (stub moduleReadsSessionsStub) Validate(context.Context, string) (djangosession.Principal, error) {
	if stub.err != nil {
		return djangosession.Principal{}, stub.err
	}
	return djangosession.Principal{UserID: userIDFixture}, nil
}

type moduleRecentVisitsStub struct {
	calls      int
	entityName string
	entityID   uuid.UUID
	projectID  uuid.UUID
	err        error
}

func (stub *moduleRecentVisitsStub) Record(_ context.Context, _ uuid.UUID, _ string, entityName string, entityID, projectID uuid.UUID) error {
	stub.calls++
	stub.entityName = entityName
	stub.entityID = entityID
	stub.projectID = projectID
	return stub.err
}

func authenticatedModuleReads(t *testing.T, repository *moduleReadsRepositoryStub, visits *moduleRecentVisitsStub) (*moduleReadsService, context.Context) {
	t.Helper()
	serviceValue, err := NewModuleReads(repository, moduleReadsSessionsStub{}, visits)
	if err != nil {
		t.Fatal(err)
	}
	service := serviceValue.(*moduleReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "session", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	return service, ctx
}

func TestModuleListInternalFailureLogsSafeRequestContext(t *testing.T) {
	const requestID = "123e4567-e89b-42d3-a456-426614174000"
	const privateDetail = "sensitive database detail"

	repository := &moduleReadsRepositoryStub{err: errors.New(privateDetail)}
	serviceValue, err := NewModuleReads(repository, moduleReadsSessionsStub{}, &moduleRecentVisitsStub{})
	if err != nil {
		t.Fatal(err)
	}
	service := serviceValue.(*moduleReadsService)
	var logs bytes.Buffer
	handler := platform.RequestIDMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		ctx := log.Context(request.Context(),
			log.WithOutputs(log.Output{Writer: &logs, Format: log.FormatJSON}),
			log.WithFunc(func(ctx context.Context) []log.KV {
				id, ok := platform.RequestID(ctx)
				if !ok {
					return nil
				}
				return []log.KV{{K: log.RequestIDKey, V: id}}
			}),
			log.WithDisableBuffering(func(context.Context) bool { return true }),
		)
		ctx = context.WithValue(ctx, moduleReadsPrincipal{}, uuid.MustParse(userIDFixture))
		_, _, readErr := service.List(ctx, &api.ListPayload{Slug: "demo", ProjectID: projectIDFixture})
		var internal *api.ModuleReadsInternalError
		if !errors.As(readErr, &internal) {
			t.Errorf("error=%T %v, want unchanged generic internal response", readErr, readErr)
		}
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/workspaces/demo/projects/"+projectIDFixture+"/modules/", nil)
	request.Header.Set(platform.RequestIDHeader, requestID)
	handler.ServeHTTP(httptest.NewRecorder(), request)

	var entries []map[string]any
	decoder := json.NewDecoder(&logs)
	for {
		var entry map[string]any
		if err := decoder.Decode(&entry); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatalf("decode log entry: %v; logs=%s", err, logs.String())
		}
		entries = append(entries, entry)
	}
	if len(entries) != 1 {
		t.Fatalf("entries=%#v; logs=%s", entries, logs.String())
	}
	entry := entries[0]
	if entry[log.RequestIDKey] != requestID || entry["error_kind"] != "other" || entry["operation"] != "module_reads.list" {
		t.Fatalf("unsafe or incomplete classification: %#v", entry)
	}
	if strings.Contains(logs.String(), privateDetail) {
		t.Fatalf("log leaked raw repository error: %s", logs.String())
	}
}

func TestNewModuleReadsRequiresDependencies(t *testing.T) {
	repository, sessions, visits := &moduleReadsRepositoryStub{}, moduleReadsSessionsStub{}, &moduleRecentVisitsStub{}
	if _, err := NewModuleReads(nil, sessions, visits); err == nil {
		t.Fatal("missing repository returned nil error")
	}
	if _, err := NewModuleReads(repository, nil, visits); err == nil {
		t.Fatal("missing sessions returned nil error")
	}
	if _, err := NewModuleReads(repository, sessions, nil); err == nil {
		t.Fatal("missing recent visits returned nil error")
	}
}

func TestModuleReadsRejectsInvalidSession(t *testing.T) {
	service := &moduleReadsService{sessions: moduleReadsSessionsStub{err: errors.New("invalid")}}
	_, err := service.APIKeyAuth(context.Background(), "bad", &security.APIKeyScheme{})
	var unauthenticated *api.ModuleReadsUnauthenticated
	if !errors.As(err, &unauthenticated) {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestModuleReadsExposesAllListAndDetailPaths(t *testing.T) {
	repository, visits := &moduleReadsRepositoryStub{}, &moduleRecentVisitsStub{}
	service, ctx := authenticatedModuleReads(t, repository, visits)
	project := projectIDFixture
	module := moduleIDFixture

	if body := readRawResponse(t, captureRawResponse(service.WorkspaceList(ctx, &api.WorkspaceListPayload{Slug: "demo"}))); body == "" {
		t.Fatal("workspace modules body is empty")
	}
	if body := readRawResponse(t, captureRawResponse(service.ArchivedList(ctx, &api.ArchivedListPayload{Slug: "demo", ProjectID: project}))); body == "" {
		t.Fatal("archived modules body is empty")
	}
	readRawResponse(t, captureRawResponse(service.ArchivedDetail(ctx, &api.ArchivedDetailPayload{Slug: "demo", ProjectID: project, Pk: module})))
	if body := readRawResponse(t, captureRawResponse(service.List(ctx, &api.ListPayload{Slug: "demo", ProjectID: project}))); body == "" {
		t.Fatal("modules body is empty")
	}
	readRawResponse(t, captureRawResponse(service.ArchiveDetail(ctx, &api.ArchiveDetailPayload{Slug: "demo", ProjectID: project, ModuleID: module})))
	if repository.archiveModuleID.String() != module {
		t.Fatalf("archive module id=%s", repository.archiveModuleID)
	}
	readRawResponse(t, captureRawResponse(service.IssueDetail(ctx, &api.IssueDetailPayload{Slug: "demo", ProjectID: project, ModuleID: module, IssueID: issueIDFixture})))
	if repository.issueID.String() != issueIDFixture {
		t.Fatalf("issue id=%s", repository.issueID)
	}
	if body := readRawResponse(t, captureRawResponse(service.Links(ctx, &api.LinksPayload{Slug: "demo", ProjectID: project, ModuleID: module}))); body == "" {
		t.Fatal("links body is empty")
	}
	readRawResponse(t, captureRawResponse(service.LinkDetail(ctx, &api.LinkDetailPayload{Slug: "demo", ProjectID: project, ModuleID: module, Pk: linkIDFixture})))
	readRawResponse(t, captureRawResponse(service.Detail(ctx, &api.DetailPayload{Slug: "demo", ProjectID: project, Pk: module})))
}

func TestModuleIssueListBuildsGroupedPaginationEnvelope(t *testing.T) {
	repository, visits := &moduleReadsRepositoryStub{}, &moduleRecentVisitsStub{}
	service, ctx := authenticatedModuleReads(t, repository, visits)
	cursor := "1:0:0"
	perPage := "1"
	orderBy, groupBy, subGroupBy := "-priority", "state__group", "assignees__id"
	state, priority := "state-a,state-b", "high"
	body := readRawResponse(t, captureRawResponse(service.Issues(ctx, &api.IssuesPayload{
		Slug: "demo", ProjectID: projectIDFixture, ModuleID: moduleIDFixture,
		Cursor: &cursor, PerPage: &perPage, OrderBy: &orderBy, GroupBy: &groupBy,
		SubGroupBy: &subGroupBy, State: &state, Priority: &priority,
	})))
	var response map[string]any
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatal(err)
	}
	if repository.issueOptions.Page != 0 || repository.issueOptions.Limit != 1 || repository.issueOptions.OrderBy != "-priority" || repository.issueOptions.State != state {
		t.Fatalf("options=%+v", repository.issueOptions)
	}
	results, ok := response["results"].(map[string]any)
	if !ok {
		t.Fatalf("results=%T %#v", response["results"], response["results"])
	}
	groups := results
	started, ok := groups["started"].(map[string]any)
	if !ok {
		t.Fatalf("groups=%#v", groups)
	}
	subgroups, ok := started["results"].(map[string]any)
	if !ok || subgroups[userIDFixture] == nil {
		t.Fatalf("started group=%#v", started)
	}
}

func TestModuleIssueListRejectsInvalidGrouping(t *testing.T) {
	service, ctx := authenticatedModuleReads(t, &moduleReadsRepositoryStub{}, &moduleRecentVisitsStub{})
	groupBy := "state_id"
	_, _, err := service.Issues(ctx, &api.IssuesPayload{
		Slug: "demo", ProjectID: projectIDFixture, ModuleID: moduleIDFixture,
		GroupBy: &groupBy, SubGroupBy: &groupBy,
	})
	var badRequest *api.ModuleIssueListBadRequest
	if !errors.As(err, &badRequest) || badRequest.ErrorMessage == nil || *badRequest.ErrorMessage != "Group by and sub group by cannot have same parameters" {
		t.Fatalf("grouping error=%T %#v", err, badRequest)
	}
}

func TestModuleIssueGroupingAcceptsAllDjangoFields(t *testing.T) {
	for _, field := range []string{"cycle_id", "target_date", "start_date"} {
		if err := validateModuleIssueGrouping(field, ""); err != nil {
			t.Errorf("grouping %q rejected: %v", field, err)
		}
	}
}

func TestModuleIssueParsingPreservesDjangoErrorBodies(t *testing.T) {
	if perPage, err := moduleIssuePerPage("0", true); err != nil || perPage != 0 {
		t.Fatalf("zero per_page=%d error=%v", perPage, err)
	}
	if _, err := moduleIssuePerPage("1001", true); err == nil || err.Error() != "Invalid per_page value. Cannot exceed 1000." {
		t.Fatalf("large per_page error=%v", err)
	}
	page, cursorValue, err := moduleIssueCursorPage("5:2:7", true, 10)
	if err != nil || page != 2 || cursorValue != 5 {
		t.Fatalf("cursor parse page=%d value=%v err=%v", page, cursorValue, err)
	}
	if totalPages, err := moduleIssueTotalPages(10, -5); err != nil || totalPages != -2 {
		t.Fatalf("negative grouped total_pages=%d error=%v", totalPages, err)
	}
	if _, err := moduleIssueTotalPages(1, 0); err == nil {
		t.Fatal("zero per_page with selected grouped rows must preserve Django's division error")
	}
	if totalPages, err := moduleIssueTotalPages(0, 0); err != nil || totalPages != 0 {
		t.Fatalf("empty zero-size grouped page total_pages=%d error=%v", totalPages, err)
	}
	service, ctx := authenticatedModuleReads(t, &moduleReadsRepositoryStub{}, &moduleRecentVisitsStub{})
	invalidGroup := "not-a-group"
	_, _, err = service.Issues(ctx, &api.IssuesPayload{
		Slug: "demo", ProjectID: projectIDFixture, ModuleID: moduleIDFixture, GroupBy: &invalidGroup,
	})
	var parseError *api.ModuleIssueListBadRequest
	if !errors.As(err, &parseError) || parseError.Detail == nil || *parseError.Detail != "Invalid group_by field: not-a-group" {
		t.Fatalf("group parse error=%T %v", err, err)
	}
	mapped := moduleReadsRepositoryError(&store.FilterError{Message: "Invalid filters", Code: "INVALID_FILTERS"})
	var filterError *api.ModuleIssueListBadRequest
	if !errors.As(mapped, &filterError) || filterError.Message == nil || *filterError.Message != "Invalid filters" || filterError.Code == nil || *filterError.Code != "INVALID_FILTERS" {
		t.Fatalf("filter error=%T %#v", mapped, filterError)
	}
}
func TestModuleIssueExplicitEmptyQueryPresenceMatchesDjango(t *testing.T) {
	repository := &moduleReadsRepositoryStub{}
	service, ctx := authenticatedModuleReads(t, repository, &moduleRecentVisitsStub{})
	payload := func() *api.IssuesPayload {
		return &api.IssuesPayload{Slug: "demo", ProjectID: projectIDFixture, ModuleID: moduleIDFixture}
	}

	perPageContext := WithReadServiceQueryPresence(ctx, url.Values{"per_page": {""}})
	_, _, err := service.Issues(perPageContext, payload())
	var invalid *api.ModuleIssueListBadRequest
	if !errors.As(err, &invalid) || invalid.Detail == nil || *invalid.Detail != "Invalid per_page parameter." {
		t.Fatalf("explicit empty per_page error=%T %#v", err, invalid)
	}

	cursorContext := WithReadServiceQueryPresence(ctx, url.Values{"cursor": {""}})
	_, _, err = service.Issues(cursorContext, payload())
	if !errors.As(err, &invalid) || invalid.Detail == nil || *invalid.Detail != "Invalid cursor parameter." {
		t.Fatalf("explicit empty cursor error=%T %#v", err, invalid)
	}

	presenceContext := WithReadServiceQueryPresence(ctx, url.Values{
		"order_by": {""}, "type": {""}, "assignees": {""}, "labels": {""},
		"cycle": {""}, "subscriber": {""},
	})
	if _, _, err := service.Issues(presenceContext, payload()); err != nil {
		t.Fatal(err)
	}
	options := repository.issueOptions
	if !options.OrderBySet || options.OrderBy != "" || !options.TypeSet ||
		!options.AssigneesSet || !options.LabelsSet || !options.CycleSet || !options.SubscriberSet {
		t.Fatalf("explicit empty query presence lost: %+v", options)
	}
}

func TestModuleListFieldsPreservesIgnoredFieldBehavior(t *testing.T) {
	data := json.RawMessage(`[` + moduleFixture + `]`)
	filtered, err := selectModuleReadFields(data, "id,does_not_exist")
	if err != nil {
		t.Fatal(err)
	}
	var modules []map[string]any
	if err := json.Unmarshal(filtered, &modules); err != nil {
		t.Fatal(err)
	}
	if len(modules) != 1 || len(modules[0]) < 20 || modules[0]["archived_at"] != nil {
		t.Fatalf("fields branch did not preserve full ModuleSerializer object: %#v", modules)
	}
	unchanged, err := selectModuleReadFields(data, ",,")
	if err != nil || string(unchanged) != string(data) {
		t.Fatalf("empty field tokens changed default projection: %s err=%v", unchanged, err)
	}
}

func TestModuleIssuePreservesRawSubgroupWithoutEnvelopeGrouping(t *testing.T) {
	repository := &moduleReadsRepositoryStub{}
	service, ctx := authenticatedModuleReads(t, repository, &moduleRecentVisitsStub{})
	subgroup := "labels__id"
	body := readRawResponse(t, captureRawResponse(service.Issues(ctx, &api.IssuesPayload{
		Slug: "demo", ProjectID: projectIDFixture, ModuleID: moduleIDFixture, SubGroupBy: &subgroup,
	})))
	var response map[string]any
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatal(err)
	}
	if repository.issueOptions.SubGroupBy != subgroup || response["sub_grouped_by"] != nil {
		t.Fatalf("raw subgroup/envelope mismatch: options=%+v response=%+v", repository.issueOptions, response)
	}
	if response["count"] != float64(1) || response["total_count"] != float64(3) || response["next_page_results"] != false {
		t.Fatalf("subgroup metadata=%+v", response)
	}
}

func TestModuleDetailAlwaysRecordsRecentVisitBestEffort(t *testing.T) {
	visits := &moduleRecentVisitsStub{err: errors.New("write failed")}
	service, ctx := authenticatedModuleReads(t, &moduleReadsRepositoryStub{}, visits)
	if _, _, err := service.Detail(ctx, &api.DetailPayload{Slug: "demo", ProjectID: projectIDFixture, Pk: moduleIDFixture}); err != nil {
		t.Fatalf("best-effort visit changed detail response: %v", err)
	}
	if visits.calls != 1 || visits.entityName != "module" || visits.entityID.String() != moduleIDFixture || visits.projectID.String() != projectIDFixture {
		t.Fatalf("visits=%+v", visits)
	}
}

func TestModuleReadsMapsIsolationAndMissingObjects(t *testing.T) {
	repository := &moduleReadsRepositoryStub{err: store.ErrForbidden}
	service, ctx := authenticatedModuleReads(t, repository, &moduleRecentVisitsStub{})
	_, _, err := service.List(ctx, &api.ListPayload{Slug: "other-tenant", ProjectID: projectIDFixture})
	var forbidden *api.ModuleReadsForbidden
	if !errors.As(err, &forbidden) {
		t.Fatalf("err=%T %v", err, err)
	}
	repository.err = store.ErrNotFound
	_, _, err = service.LinkDetail(ctx, &api.LinkDetailPayload{Slug: "demo", ProjectID: projectIDFixture, ModuleID: moduleIDFixture, Pk: linkIDFixture})
	var linkNotFound *api.ModuleReadsLinkNotFound
	if !errors.As(err, &linkNotFound) || linkNotFound.Detail != "No ModuleLink matches the given query." {
		t.Fatalf("link err=%T %v", err, err)
	}
	_, _, err = service.Detail(ctx, &api.DetailPayload{Slug: "demo", ProjectID: projectIDFixture, Pk: moduleIDFixture})
	var moduleNotFound *api.ModuleReadsModuleNotFound
	if !errors.As(err, &moduleNotFound) || moduleNotFound.Message != "Module not found" {
		t.Fatalf("module err=%T %v", err, err)
	}
}

func TestModulePermissionClassEndpointsUseDRFPermissionEnvelope(t *testing.T) {
	repository := &moduleReadsRepositoryStub{err: store.ErrForbidden}
	service, ctx := authenticatedModuleReads(t, repository, &moduleRecentVisitsStub{})
	_, _, err := service.WorkspaceList(ctx, &api.WorkspaceListPayload{Slug: "other-tenant"})
	var denied *api.ModuleReadsPermissionDenied
	if !errors.As(err, &denied) || denied.Detail != "You do not have permission to perform this action." {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestGroupedModuleIssuesPaginatesEachPreseededBucket(t *testing.T) {
	issues := []map[string]any{
		{"id": "one", "priority": "high", "assignee_ids": []any{"member-a", "member-b"}, "_count_for_pages": false},
		{"id": "two", "priority": "high", "assignee_ids": []any{"member-a"}, "_count_for_pages": true},
		{"id": "three", "priority": "low", "assignee_ids": []any{}, "_count_for_pages": true},
	}
	result, count, maximum, next := groupModuleIssues(
		issues, "priority", "assignees__id",
		[]any{"low", "high", "urgent"}, []any{"member-a", "member-b", "ghost", nil},
		map[string]int{"high": 0, "low": 1, "urgent": 0},
		map[string]int{"high:::member-a": 1, "high:::member-b": 0, "high:::exhausted": 2, "low:::None": 1},
		1, 0,
	)
	groups := result.(map[string]any)
	if groups["urgent"] == nil {
		t.Fatalf("empty domain bucket missing: %#v", groups)
	}
	high := groups["high"].(map[string]any)
	if high["total_results"] != 1 || count != 3 || maximum != 1 || !next {
		t.Fatalf("high=%#v count=%d maximum=%d next=%v", high, count, maximum, next)
	}
	subgroups := high["results"].(map[string]any)
	if subgroups["member-a"].(map[string]any)["total_results"] != 1 {
		t.Fatalf("subgroups=%#v", subgroups)
	}
	if exhausted := subgroups["exhausted"].(map[string]any); exhausted["total_results"] != 2 || len(exhausted["results"].([]map[string]any)) != 0 {
		t.Fatalf("exhausted subgroup=%#v", exhausted)
	}
	if urgent := groups["urgent"].(map[string]any); urgent["total_results"] != 1 {
		t.Fatalf("observed pending-only group=%#v", urgent)
	}
	pendingOnly, _, pendingMaximum, _ := groupModuleIssues(
		issues[:1], "priority", "", []any{"high"}, nil,
		map[string]int{"high": 0}, nil, 1, 0,
	)
	pendingHigh := pendingOnly.(map[string]any)["high"].(map[string]any)
	if pendingMaximum != 0 || pendingHigh["total_results"] != 1 {
		t.Fatalf("pending-only maximum=%d group=%#v", pendingMaximum, pendingHigh)
	}
	if _, exists := subgroups["ghost"]; exists {
		t.Fatalf("unobserved subgroup was cross-product seeded: %#v", subgroups)
	}
	memberA := subgroups["member-a"].(map[string]any)["results"].([]map[string]any)[0]
	if memberA["assignees__id"] != "member-a" {
		t.Fatalf("grouped relation field=%#v", memberA)
	}
	if values, exists := memberA["assignee_ids"].([]any); !exists || len(values) != 2 {
		t.Fatalf("grouped relation lost array annotation: %#v", memberA)
	}
	empty, emptyCount, _, _ := groupModuleIssues(
		issues, "priority", "assignees__id",
		[]any{"low", "high", "urgent"}, []any{"member-a", "member-b", nil},
		nil, nil, 1, 9,
	)
	emptyGroups := empty.(map[string]any)
	if emptyCount != 3 || len(emptyGroups) != 3 {
		t.Fatalf("out-of-range grouped page=%#v count=%d", emptyGroups, emptyCount)
	}
	for groupKey, rawGroup := range emptyGroups {
		for subgroupKey, rawSubgroup := range rawGroup.(map[string]any)["results"].(map[string]any) {
			results := rawSubgroup.(map[string]any)["results"].([]map[string]any)
			if len(results) != 0 {
				t.Fatalf("out-of-range group=%s subgroup=%s results=%#v", groupKey, subgroupKey, results)
			}
		}
	}
	single, _, _, _ := groupModuleIssues(
		issues, "assignees__id", "",
		[]any{"ghost"}, nil,
		nil, nil, 1, 1,
	)
	singleGroups := single.(map[string]any)
	if len(singleGroups) != 1 || singleGroups["member-a"] == nil {
		t.Fatalf("single M2M groups=%#v", singleGroups)
	}
	domainOnly, _, _, _ := groupModuleIssues(
		[]map[string]any{{"id": "known", "state_id": "state-a"}, {"id": "null", "state_id": nil}},
		"state_id", "", []any{"state-a"}, nil, nil, nil, 10, 0,
	)
	domainGroups := domainOnly.(map[string]any)
	if domainGroups["state-a"] == nil || domainGroups["None"] != nil {
		t.Fatalf("scalar grouping escaped declared domain: %#v", domainGroups)
	}
	onlyOutside, outsideCount, _, _ := groupModuleIssues(
		[]map[string]any{{"id": "null", "state_id": nil}},
		"state_id", "", []any{"state-a"}, nil, nil, nil, 10, 0,
	)
	if outsideCount != 1 || onlyOutside.(map[string]any)["state-a"] == nil {
		t.Fatalf("raw grouped count/domain map mismatch: result=%#v count=%d", onlyOutside, outsideCount)
	}
}

func TestModuleAndLinkDetailsUseEndpointSpecificNotFoundEnvelopesForInvalidIDs(t *testing.T) {
	service, ctx := authenticatedModuleReads(t, &moduleReadsRepositoryStub{}, &moduleRecentVisitsStub{})
	_, _, err := service.LinkDetail(ctx, &api.LinkDetailPayload{Slug: "demo", ProjectID: projectIDFixture, ModuleID: moduleIDFixture, Pk: "invalid"})
	var linkNotFound *api.ModuleReadsLinkNotFound
	if !errors.As(err, &linkNotFound) || linkNotFound.Detail != "No ModuleLink matches the given query." {
		t.Fatalf("link err=%T %v", err, err)
	}
	_, _, err = service.Detail(ctx, &api.DetailPayload{Slug: "demo", ProjectID: projectIDFixture, Pk: "invalid"})
	var moduleNotFound *api.ModuleReadsModuleNotFound
	if !errors.As(err, &moduleNotFound) || moduleNotFound.Message != "Module not found" {
		t.Fatalf("module err=%T %v", err, err)
	}
}

func TestModuleIssuePropertyProjectionRequiresExplicitOptIn(t *testing.T) {
	repository, visits := &moduleReadsRepositoryStub{}, &moduleRecentVisitsStub{}
	service, ctx := authenticatedModuleReads(t, repository, visits)
	for _, value := range []string{"", "false", "true"} {
		scoped := WithReadServiceQueryPresence(ctx, url.Values{"include_properties": {value}})
		readRawResponse(t, captureRawResponse(service.Issues(scoped, &api.IssuesPayload{Slug: "demo", ProjectID: projectIDFixture, ModuleID: moduleIDFixture})))
		if repository.issueOptions.IncludeProperties != (value == "true") {
			t.Fatalf("projection flag %q = %v", value, repository.issueOptions.IncludeProperties)
		}
	}
}
