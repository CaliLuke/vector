package systemapi

import (
	"context"
	"encoding/json"
	"encoding/json/jsontext"
	"errors"
	"io"
	"net/url"
	"reflect"
	"testing"

	api "github.com/CaliLuke/go-plane/gen/intake_inbox_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/intakeinboxreads"
	"github.com/CaliLuke/loom/security"
	"uuid"
)

const (
	intakeReadsUserID    = "00000000-0000-4000-8000-000000000001"
	intakeReadsProjectID = "00000000-0000-4000-8000-000000000002"
	intakeReadsObjectID  = "00000000-0000-4000-8000-000000000003"
	intakeReadsVersionID = "00000000-0000-4000-8000-000000000004"
)

const intakeReadsDraftFixture = `{"id":"00000000-0000-4000-8000-000000000003","name":"Draft","state_id":null,"sort_order":65535,"completed_at":null,"estimate_point":null,"priority":"none","start_date":null,"target_date":null,"project_id":"00000000-0000-4000-8000-000000000002","parent_id":null,"cycle_id":null,"module_ids":[],"label_ids":[],"assignee_ids":[],"created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-02T03:04:05Z","created_by":"00000000-0000-4000-8000-000000000001","updated_by":null,"type_id":null,"description_html":"<p>Draft</p>"}`
const intakeReadsPageFixture = `{"grouped_by":null,"sub_grouped_by":null,"total_count":1,"next_cursor":"20:1:0","prev_cursor":"20:-1:1","next_page_results":false,"prev_page_results":false,"count":1,"total_pages":1,"total_results":1,"extra_stats":null,"results":[{"id":"00000000-0000-4000-8000-000000000003","status":-2,"duplicate_to":null,"snoozed_till":null,"source":"IN_APP","issue":{"id":"00000000-0000-4000-8000-000000000004","name":"Request","priority":"high","sequence_id":7,"project_id":"00000000-0000-4000-8000-000000000002","created_at":"2026-01-02T03:04:05Z","label_ids":[],"created_by":"00000000-0000-4000-8000-000000000001"},"created_by":"00000000-0000-4000-8000-000000000001"}]}`
const intakeReadsIssueDetailFixture = `{"id":"00000000-0000-4000-8000-000000000003","status":-2,"duplicate_to":null,"snoozed_till":null,"duplicate_issue_detail":null,"source":"IN_APP","issue":{"id":"00000000-0000-4000-8000-000000000004","name":"Request","state_id":null,"sort_order":65535,"completed_at":null,"estimate_point":null,"priority":"high","start_date":null,"target_date":null,"sequence_id":7,"project_id":"00000000-0000-4000-8000-000000000002","parent_id":null,"label_ids":[],"assignee_ids":[],"created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-02T03:04:05Z","created_by":"00000000-0000-4000-8000-000000000001","updated_by":null,"is_draft":false,"archived_at":null,"description_html":"<p>Request</p>"}}`
const intakeReadsContainerFixture = `{"id":"00000000-0000-4000-8000-000000000003","created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-02T03:04:05Z","deleted_at":null,"created_by":"00000000-0000-4000-8000-000000000001","updated_by":null,"workspace":"00000000-0000-4000-8000-000000000005","project":"00000000-0000-4000-8000-000000000002","name":"Inbox","description":"","is_default":true,"view_props":{},"logo_props":{},"project_detail":{"id":"00000000-0000-4000-8000-000000000002","identifier":"DEMO","name":"Demo","cover_image":null,"cover_image_url":null,"logo_props":{},"description":""},"pending_issue_count":1}`
const intakeReadsVersionListFixture = `{"prev_cursor":"1000:-1:0","cursor":"1000:0:0","next_cursor":null,"prev_page_results":false,"next_page_results":false,"page_count":1,"total_results":1,"total_pages":1,"results":[{"id":"00000000-0000-4000-8000-000000000004","workspace":"00000000-0000-4000-8000-000000000005","project":"00000000-0000-4000-8000-000000000002","issue":"00000000-0000-4000-8000-000000000003","last_saved_at":"2026-01-02T03:04:05Z","owned_by":"00000000-0000-4000-8000-000000000001","created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-02T03:04:05Z","created_by":"00000000-0000-4000-8000-000000000001","updated_by":null}]}`
const intakeReadsVersionDetailFixture = `{"id":"00000000-0000-4000-8000-000000000004","workspace":"00000000-0000-4000-8000-000000000005","project":"00000000-0000-4000-8000-000000000002","issue":"00000000-0000-4000-8000-000000000003","last_saved_at":"2026-01-02T03:04:05Z","owned_by":"00000000-0000-4000-8000-000000000001","created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-02T03:04:05Z","created_by":"00000000-0000-4000-8000-000000000001","updated_by":null,"description_binary":null,"description_html":"<p>Version</p>","description_stripped":"Version","description_json":{"type":"doc"}}`

type intakeInboxReadsRepositoryStub struct {
	calls             []string
	data              map[string]jsontext.Value
	err               error
	listOptions       store.ListOptions
	descriptionCursor *string
}

func (stub *intakeInboxReadsRepositoryStub) result(name string) (jsontext.Value, error) {
	stub.calls = append(stub.calls, name)
	return stub.data[name], stub.err
}

func (stub *intakeInboxReadsRepositoryStub) DraftIssueList(_ context.Context, _ uuid.UUID, _ string, options store.ListOptions) (jsontext.Value, error) {
	stub.listOptions = options
	return stub.result("draft_issue_list")
}
func (stub *intakeInboxReadsRepositoryStub) DraftIssueDetail(context.Context, uuid.UUID, string, uuid.UUID) (jsontext.Value, error) {
	return stub.result("draft_issue_detail")
}
func (stub *intakeInboxReadsRepositoryStub) InboxIssueList(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, options store.ListOptions) (jsontext.Value, error) {
	stub.listOptions = options
	return stub.result("inbox_issue_list")
}
func (stub *intakeInboxReadsRepositoryStub) InboxIssueDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (jsontext.Value, error) {
	return stub.result("inbox_issue_detail")
}
func (stub *intakeInboxReadsRepositoryStub) InboxList(context.Context, uuid.UUID, string, uuid.UUID) (jsontext.Value, error) {
	return stub.result("inbox_list")
}
func (stub *intakeInboxReadsRepositoryStub) InboxDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (jsontext.Value, error) {
	return stub.result("inbox_detail")
}
func (stub *intakeInboxReadsRepositoryStub) IntakeIssueList(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, options store.ListOptions) (jsontext.Value, error) {
	stub.listOptions = options
	return stub.result("intake_issue_list")
}
func (stub *intakeInboxReadsRepositoryStub) IntakeIssueDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (jsontext.Value, error) {
	return stub.result("intake_issue_detail")
}
func (stub *intakeInboxReadsRepositoryStub) DescriptionVersionList(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ uuid.UUID, cursor *string) (jsontext.Value, error) {
	stub.descriptionCursor = cursor
	return stub.result("description_version_list")
}
func (stub *intakeInboxReadsRepositoryStub) DescriptionVersionDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, uuid.UUID) (jsontext.Value, error) {
	return stub.result("description_version_detail")
}
func (stub *intakeInboxReadsRepositoryStub) IntakeList(context.Context, uuid.UUID, string, uuid.UUID) (jsontext.Value, error) {
	return stub.result("intake_list")
}
func (stub *intakeInboxReadsRepositoryStub) IntakeDetail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (jsontext.Value, error) {
	return stub.result("intake_detail")
}

type intakeInboxReadsSessionsStub struct {
	principal djangosession.Principal
	err       error
}

func (stub intakeInboxReadsSessionsStub) Validate(context.Context, string) (djangosession.Principal, error) {
	if stub.err != nil {
		return djangosession.Principal{}, stub.err
	}
	return stub.principal, nil
}

type intakeInboxReadsCountingSessions struct {
	calls int
}

func (stub *intakeInboxReadsCountingSessions) Validate(context.Context, string) (djangosession.Principal, error) {
	stub.calls++
	return djangosession.Principal{}, errors.New("unexpected session validation")
}

func authenticatedIntakeReadsService(t *testing.T, repository *intakeInboxReadsRepositoryStub) (*intakeInboxReadsService, context.Context) {
	t.Helper()
	value, err := NewIntakeInboxReads(repository, intakeInboxReadsSessionsStub{principal: djangosession.Principal{UserID: intakeReadsUserID, UserTimezone: "UTC"}})
	if err != nil {
		t.Fatal(err)
	}
	service := value.(*intakeInboxReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "session", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	return service, ctx
}

func decodeIntakeReadsBody[T any](t *testing.T, body io.ReadCloser, err error) (T, []byte) {
	t.Helper()
	var result T
	if err != nil {
		t.Fatal(err)
	}
	if body == nil {
		t.Fatal("nil response body")
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result, data
}

func TestNewIntakeInboxReadsRequiresDependenciesAndSession(t *testing.T) {
	if _, err := NewIntakeInboxReads(nil, intakeInboxReadsSessionsStub{}); err == nil {
		t.Fatal("missing repository returned nil error")
	}
	if _, err := NewIntakeInboxReads(&intakeInboxReadsRepositoryStub{}, nil); err == nil {
		t.Fatal("missing sessions returned nil error")
	}
	service := &intakeInboxReadsService{repository: &intakeInboxReadsRepositoryStub{}, sessions: intakeInboxReadsSessionsStub{err: errors.New("invalid")}}
	_, err := service.APIKeyAuth(context.Background(), "bad", &security.APIKeyScheme{})
	var unauthenticated *api.IntakeInboxReadsUnauthenticated
	if !errors.As(err, &unauthenticated) {
		t.Fatalf("error=%T %v", err, err)
	}
}

func TestIntakeInboxReadsUsesPrevalidatedSessionPrincipalAndTimezone(t *testing.T) {
	sessions := &intakeInboxReadsCountingSessions{}
	service := &intakeInboxReadsService{repository: &intakeInboxReadsRepositoryStub{}, sessions: sessions}
	ctx := WithReadServiceSessionPrincipal(context.Background(), intakeReadsUserID, "America/New_York")
	authenticated, err := service.APIKeyAuth(ctx, "already-validated", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := service.principal(authenticated)
	if err != nil {
		t.Fatal(err)
	}
	if principal.userID.String() != intakeReadsUserID || principal.timezone != "America/New_York" || sessions.calls != 0 {
		t.Fatalf("principal=%+v validation calls=%d", principal, sessions.calls)
	}
}

func TestListServicesPreserveExplicitEmptyQueryPresence(t *testing.T) {
	repository := &intakeInboxReadsRepositoryStub{data: map[string]jsontext.Value{"inbox_issue_list": jsontext.Value(intakeReadsPageFixture)}}
	service, ctx := authenticatedIntakeReadsService(t, repository)
	ctx = WithReadServiceQueryPresence(ctx, url.Values{
		"per_page": {""},
		"cursor":   {""},
		"status":   {""},
		"type":     {""},
		"labels":   {""},
	})
	if _, body, err := service.InboxIssueList(ctx, &api.InboxIssueListPayload{Slug: "demo", ProjectID: intakeReadsProjectID}); err != nil {
		t.Fatal(err)
	} else {
		body.Close()
	}
	options := repository.listOptions
	if !options.PerPageSet || !options.CursorSet || !options.StatusSet || !options.PresentFilters["type"] || !options.PresentFilters["labels"] {
		t.Fatalf("query presence=%+v", options)
	}
}

func TestDescriptionServicePreservesEmptyCursorAsDjangoJSON500(t *testing.T) {
	repository := &intakeInboxReadsRepositoryStub{err: store.ErrInvalidDescriptionCursor}
	service, ctx := authenticatedIntakeReadsService(t, repository)
	ctx = WithReadServiceQueryPresence(ctx, url.Values{"cursor": {""}})
	ctx = WithReadServiceDjangoHTML500(ctx)
	_, _, err := service.DescriptionVersionList(ctx, &api.DescriptionVersionListPayload{
		Slug: "demo", ProjectID: intakeReadsProjectID, WorkItemID: intakeReadsObjectID,
	})
	var internal *api.IntakeInboxReadsInternalError
	if !errors.As(err, &internal) || repository.descriptionCursor == nil || *repository.descriptionCursor != "" || ReadServiceDjangoHTML500(ctx) {
		t.Fatalf("err=%T cursor=%v html500=%v", err, repository.descriptionCursor, ReadServiceDjangoHTML500(ctx))
	}
}

func TestIntakeInboxReadsDispatchAllTwelveDistinctRepositoryPaths(t *testing.T) {
	repository := &intakeInboxReadsRepositoryStub{data: map[string]jsontext.Value{
		"draft_issue_list":   jsontext.Value(`{"grouped_by":null,"sub_grouped_by":null,"total_count":1,"next_cursor":"1000:1:0","prev_cursor":"1000:-1:1","next_page_results":false,"prev_page_results":false,"count":1,"total_pages":1,"total_results":1,"extra_stats":null,"results":[` + intakeReadsDraftFixture + `]}`),
		"draft_issue_detail": jsontext.Value(intakeReadsDraftFixture),
		"inbox_issue_list":   jsontext.Value(intakeReadsPageFixture), "intake_issue_list": jsontext.Value(intakeReadsPageFixture),
		"inbox_issue_detail": jsontext.Value(intakeReadsIssueDetailFixture), "intake_issue_detail": jsontext.Value(intakeReadsIssueDetailFixture),
		"inbox_list": jsontext.Value(intakeReadsContainerFixture), "inbox_detail": jsontext.Value(intakeReadsContainerFixture),
		"intake_list": jsontext.Value(intakeReadsContainerFixture), "intake_detail": jsontext.Value(intakeReadsContainerFixture),
		"description_version_list": jsontext.Value(intakeReadsVersionListFixture), "description_version_detail": jsontext.Value(intakeReadsVersionDetailFixture),
	}}
	service, ctx := authenticatedIntakeReadsService(t, repository)
	_, _, _ = service.DraftIssueList(ctx, &api.DraftIssueListPayload{Slug: "demo"})
	_, _, _ = service.DraftIssueDetail(ctx, &api.DraftIssueDetailPayload{Slug: "demo", Pk: intakeReadsObjectID})
	_, _, _ = service.InboxIssueList(ctx, &api.InboxIssueListPayload{Slug: "demo", ProjectID: intakeReadsProjectID})
	_, _, _ = service.InboxIssueDetail(ctx, &api.InboxIssueDetailPayload{Slug: "demo", ProjectID: intakeReadsProjectID, Pk: intakeReadsObjectID})
	_, _, _ = service.InboxList(ctx, &api.InboxListPayload{Slug: "demo", ProjectID: intakeReadsProjectID})
	_, _, _ = service.InboxDetail(ctx, &api.InboxDetailPayload{Slug: "demo", ProjectID: intakeReadsProjectID, Pk: intakeReadsObjectID})
	_, _, _ = service.IntakeIssueList(ctx, &api.IntakeIssueListPayload{Slug: "demo", ProjectID: intakeReadsProjectID})
	_, _, _ = service.IntakeIssueDetail(ctx, &api.IntakeIssueDetailPayload{Slug: "demo", ProjectID: intakeReadsProjectID, Pk: intakeReadsObjectID})
	_, _, _ = service.DescriptionVersionList(ctx, &api.DescriptionVersionListPayload{Slug: "demo", ProjectID: intakeReadsProjectID, WorkItemID: intakeReadsObjectID})
	_, _, _ = service.DescriptionVersionDetail(ctx, &api.DescriptionVersionDetailPayload{Slug: "demo", ProjectID: intakeReadsProjectID, WorkItemID: intakeReadsObjectID, Pk: intakeReadsVersionID})
	_, _, _ = service.IntakeList(ctx, &api.IntakeListPayload{Slug: "demo", ProjectID: intakeReadsProjectID})
	_, _, _ = service.IntakeDetail(ctx, &api.IntakeDetailPayload{Slug: "demo", ProjectID: intakeReadsProjectID, Pk: intakeReadsObjectID})
	want := []string{"draft_issue_list", "draft_issue_detail", "inbox_issue_list", "inbox_issue_detail", "inbox_list", "inbox_detail", "intake_issue_list", "intake_issue_detail", "description_version_list", "description_version_detail", "intake_list", "intake_detail"}
	if !reflect.DeepEqual(repository.calls, want) {
		t.Fatalf("repository calls=%v want=%v", repository.calls, want)
	}
}

func TestIssueAliasesPreserveEnvelopeDetailAndFilterInputs(t *testing.T) {
	status, priority, orderBy, perPage, cursor := "-2,1", "high", "issue__priority", "20", "20:0:0"
	repository := &intakeInboxReadsRepositoryStub{data: map[string]jsontext.Value{"inbox_issue_list": jsontext.Value(intakeReadsPageFixture), "intake_issue_list": jsontext.Value(intakeReadsPageFixture), "inbox_issue_detail": jsontext.Value(intakeReadsIssueDetailFixture), "intake_issue_detail": jsontext.Value(intakeReadsIssueDetailFixture)}}
	service, ctx := authenticatedIntakeReadsService(t, repository)
	inboxMetadata, inboxBody, err := service.InboxIssueList(ctx, &api.InboxIssueListPayload{Slug: "demo", ProjectID: intakeReadsProjectID, Status: &status, Priority: &priority, OrderBy: &orderBy, PerPage: &perPage, Cursor: &cursor})
	_, inboxData := decodeIntakeReadsBody[map[string]any](t, inboxBody, err)
	if inboxMetadata == nil || inboxMetadata.ContentType != "application/json" {
		t.Fatalf("response metadata=%+v", inboxMetadata)
	}
	if repository.listOptions.Filters["status"] != status || repository.listOptions.Filters["priority"] != priority || repository.listOptions.OrderBy != orderBy || repository.listOptions.PerPage != perPage || repository.listOptions.Timezone != "UTC" {
		t.Fatalf("list options=%+v", repository.listOptions)
	}
	_, intakeBody, err := service.IntakeIssueList(ctx, &api.IntakeIssueListPayload{Slug: "demo", ProjectID: intakeReadsProjectID, Status: &status, Priority: &priority, OrderBy: &orderBy, PerPage: &perPage, Cursor: &cursor})
	page, intakeData := decodeIntakeReadsBody[map[string]any](t, intakeBody, err)
	if page["total_count"] != float64(1) || page["count"] != float64(1) || len(page["results"].([]any)) != 1 {
		t.Fatalf("page=%+v", page)
	}
	if string(inboxData) != string(intakeData) {
		t.Fatal("inbox/intake list aliases returned different wire envelopes")
	}
	_, inboxDetailBody, err := service.InboxIssueDetail(ctx, &api.InboxIssueDetailPayload{Slug: "demo", ProjectID: intakeReadsProjectID, Pk: intakeReadsObjectID})
	inboxDetail, _ := decodeIntakeReadsBody[map[string]any](t, inboxDetailBody, err)
	inboxIssue := inboxDetail["issue"].(map[string]any)
	if inboxIssue["description_html"] != "<p>Request</p>" {
		t.Fatalf("inbox detail=%+v", inboxDetail)
	}
	_, intakeDetailBody, err := service.IntakeIssueDetail(ctx, &api.IntakeIssueDetailPayload{Slug: "demo", ProjectID: intakeReadsProjectID, Pk: intakeReadsObjectID})
	intakeDetail, _ := decodeIntakeReadsBody[map[string]any](t, intakeDetailBody, err)
	intakeIssue := intakeDetail["issue"].(map[string]any)
	if intakeIssue["description_html"] != inboxIssue["description_html"] {
		t.Fatalf("intake detail=%+v", intakeDetail)
	}
}

func TestDraftAndDescriptionVersionProjectionsRemainDistinct(t *testing.T) {
	repository := &intakeInboxReadsRepositoryStub{data: map[string]jsontext.Value{
		"draft_issue_detail":         jsontext.Value(intakeReadsDraftFixture),
		"description_version_list":   jsontext.Value(intakeReadsVersionListFixture),
		"description_version_detail": jsontext.Value(intakeReadsVersionDetailFixture),
	}}
	service, ctx := authenticatedIntakeReadsService(t, repository)
	_, draftBody, err := service.DraftIssueDetail(ctx, &api.DraftIssueDetailPayload{Slug: "demo", Pk: intakeReadsObjectID})
	draft, _ := decodeIntakeReadsBody[map[string]any](t, draftBody, err)
	if draft["description_html"] != "<p>Draft</p>" || len(draft["module_ids"].([]any)) != 0 || len(draft["label_ids"].([]any)) != 0 || len(draft["assignee_ids"].([]any)) != 0 {
		t.Fatalf("draft=%+v", draft)
	}
	_, listBody, err := service.DescriptionVersionList(ctx, &api.DescriptionVersionListPayload{Slug: "demo", ProjectID: intakeReadsProjectID, WorkItemID: intakeReadsObjectID})
	list, _ := decodeIntakeReadsBody[map[string]any](t, listBody, err)
	if len(list["results"].([]any)) != 1 || list["page_count"] != float64(1) {
		t.Fatalf("version list=%+v", list)
	}
	row := list["results"].([]any)[0].(map[string]any)
	if _, exists := row["description_html"]; exists {
		t.Fatal("description-version list row contains detail content")
	}
	_, detailBody, err := service.DescriptionVersionDetail(ctx, &api.DescriptionVersionDetailPayload{Slug: "demo", ProjectID: intakeReadsProjectID, WorkItemID: intakeReadsObjectID, Pk: intakeReadsVersionID})
	detail, _ := decodeIntakeReadsBody[map[string]any](t, detailBody, err)
	if detail["description_html"] != "<p>Version</p>" {
		t.Fatalf("version detail=%+v", detail)
	}
	if detail["description_json"] == nil {
		t.Fatalf("description_json=%#v", detail["description_json"])
	}
}

func TestIntakeContainerAliasesReturnObjectsAndDoNotTriggerExtraEffects(t *testing.T) {
	repository := &intakeInboxReadsRepositoryStub{data: map[string]jsontext.Value{"inbox_list": jsontext.Value(intakeReadsContainerFixture), "intake_list": jsontext.Value(intakeReadsContainerFixture)}}
	service, ctx := authenticatedIntakeReadsService(t, repository)
	_, inboxBody, err := service.InboxList(ctx, &api.InboxListPayload{Slug: "demo", ProjectID: intakeReadsProjectID})
	inbox, _ := decodeIntakeReadsBody[map[string]any](t, inboxBody, err)
	if inbox["name"] != "Inbox" || inbox["pending_issue_count"] != float64(1) {
		t.Fatalf("inbox=%+v", inbox)
	}
	_, intakeBody, err := service.IntakeList(ctx, &api.IntakeListPayload{Slug: "demo", ProjectID: intakeReadsProjectID})
	intake, _ := decodeIntakeReadsBody[map[string]any](t, intakeBody, err)
	projectDetail := intake["project_detail"].(map[string]any)
	if intake["name"] != inbox["name"] || projectDetail["identifier"] != "DEMO" {
		t.Fatalf("intake=%+v", intake)
	}
	if !reflect.DeepEqual(repository.calls, []string{"inbox_list", "intake_list"}) {
		t.Fatalf("unexpected repository effects=%v", repository.calls)
	}
}

func TestIntakeContainerCollectionsPreserveEmptyObjectResponse(t *testing.T) {
	repository := &intakeInboxReadsRepositoryStub{data: map[string]jsontext.Value{
		"inbox_list":  jsontext.Value(`{}`),
		"intake_list": jsontext.Value(`{}`),
	}}
	service, ctx := authenticatedIntakeReadsService(t, repository)
	for name, call := range map[string]func() (*api.RawJSONResponseMetadata, io.ReadCloser, error){
		"inbox": func() (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
			return service.InboxList(ctx, &api.InboxListPayload{Slug: "demo", ProjectID: intakeReadsProjectID})
		},
		"intake": func() (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
			return service.IntakeList(ctx, &api.IntakeListPayload{Slug: "demo", ProjectID: intakeReadsProjectID})
		},
	} {
		metadata, body, err := call()
		result, _ := decodeIntakeReadsBody[map[string]any](t, body, err)
		if metadata == nil || metadata.ContentType != "application/json" || len(result) != 0 {
			t.Errorf("%s metadata=%+v body=%v", name, metadata, result)
		}
	}
}

func TestRepositoryErrorsMapToStableDjangoShapes(t *testing.T) {
	repository := &intakeInboxReadsRepositoryStub{data: map[string]jsontext.Value{}, err: store.ErrForbidden}
	service, ctx := authenticatedIntakeReadsService(t, repository)
	_, _, err := service.InboxList(ctx, &api.InboxListPayload{Slug: "other", ProjectID: intakeReadsProjectID})
	var denied *api.IntakeInboxReadsPermissionDenied
	if !errors.As(err, &denied) || denied.Message != "You don't have the required permissions." {
		t.Fatalf("denied=%T %v", err, err)
	}
	_, _, err = service.DraftIssueDetail(ctx, &api.DraftIssueDetailPayload{Slug: "demo", Pk: intakeReadsObjectID})
	if !errors.As(err, &denied) || denied.Message != "You don't have the required permissions." {
		t.Fatalf("draft detail denied=%T %v", err, err)
	}
	repository.err = store.ErrGuestRestricted
	_, _, err = service.InboxIssueDetail(ctx, &api.InboxIssueDetailPayload{Slug: "demo", ProjectID: intakeReadsProjectID, Pk: intakeReadsObjectID})
	if !errors.As(err, &denied) || denied.Message != "You are not allowed to view this issue" {
		t.Fatalf("guest issue denied=%T %v", err, err)
	}
	_, _, err = service.DescriptionVersionList(ctx, &api.DescriptionVersionListPayload{Slug: "demo", ProjectID: intakeReadsProjectID, WorkItemID: intakeReadsObjectID})
	if !errors.As(err, &denied) || denied.Message != "You are not allowed to view this issue" {
		t.Fatalf("guest version denied=%T %v", err, err)
	}
	repository.err = store.ErrInvalidDetail
	_, _, err = service.DraftIssueList(ctx, &api.DraftIssueListPayload{Slug: "demo"})
	var invalidDetail *api.IntakeInboxReadsInvalidDetail
	if !errors.As(err, &invalidDetail) || invalidDetail.Message != "Please provide valid detail" {
		t.Fatalf("invalid detail=%T %v", err, err)
	}
	repository.err = store.ErrNotFound
	_, _, err = service.InboxDetail(ctx, &api.InboxDetailPayload{Slug: "demo", ProjectID: intakeReadsProjectID, Pk: intakeReadsObjectID})
	var notFound *api.IntakeInboxReadsDetailNotFound
	if !errors.As(err, &notFound) || notFound.Detail != "No Intake matches the given query." {
		t.Fatalf("not found=%T %v", err, err)
	}
	repository.err = store.ErrMissingIntake
	_, _, err = service.InboxIssueDetail(ctx, &api.InboxIssueDetailPayload{Slug: "demo", ProjectID: intakeReadsProjectID, Pk: intakeReadsObjectID})
	var internal *api.IntakeInboxReadsInternalError
	if !errors.As(err, &internal) || internal.Message != intakeInboxReadsInternalErrorDetail {
		t.Fatalf("missing intake detail=%T %v", err, err)
	}
}

func TestPaginationErrorsMapToExactDjangoDetail(t *testing.T) {
	for _, test := range []struct {
		err    error
		detail string
	}{
		{err: store.ErrInvalidPerPage, detail: "Invalid per_page parameter."},
		{err: store.ErrPerPageTooLarge, detail: "Invalid per_page value. Cannot exceed 1000."},
		{err: store.ErrInvalidCursor, detail: "Invalid cursor parameter."},
	} {
		mapped := intakeInboxReadsRepositoryError(test.err, "")
		var badRequest *api.IntakeInboxReadsBadRequest
		if !errors.As(mapped, &badRequest) || badRequest.Detail != test.detail {
			t.Errorf("error %v mapped to %T %v", test.err, mapped, mapped)
		}
	}
}

func TestMalformedDescriptionCursorPreservesDjangoInternalError(t *testing.T) {
	mapped := intakeInboxReadsRepositoryError(store.ErrInvalidDescriptionCursor, "")
	var internal *api.IntakeInboxReadsInternalError
	if !errors.As(mapped, &internal) || internal.Message != intakeInboxReadsInternalErrorDetail {
		t.Fatalf("mapped=%T %v", mapped, mapped)
	}
}

func TestDescriptionVersionOptionsReturnAuthenticatedDjangoMetadata(t *testing.T) {
	service, ctx := authenticatedIntakeReadsService(t, &intakeInboxReadsRepositoryStub{})
	list, err := service.DescriptionVersionListOptions(ctx, &api.DescriptionVersionListOptionsPayload{})
	if err != nil {
		t.Fatal(err)
	}
	detail, err := service.DescriptionVersionDetailOptions(ctx, &api.DescriptionVersionDetailOptionsPayload{})
	if err != nil {
		t.Fatal(err)
	}
	for _, metadata := range []*api.IntakeInboxReadsMetadata{list, detail} {
		if metadata.Name != "Intake Work Item Description Version Endpoint" || len(metadata.Renders) != 1 || len(metadata.Parses) != 3 {
			t.Fatalf("metadata=%+v", metadata)
		}
	}
}

func TestDescriptionVersionListKeepsLastSavedAtUTCButConvertsAuditTimes(t *testing.T) {
	repository := &intakeInboxReadsRepositoryStub{data: map[string]jsontext.Value{
		"description_version_list":   jsontext.Value(intakeReadsVersionListFixture),
		"description_version_detail": jsontext.Value(intakeReadsVersionDetailFixture),
	}}
	value, err := NewIntakeInboxReads(repository, intakeInboxReadsSessionsStub{principal: djangosession.Principal{UserID: intakeReadsUserID, UserTimezone: "America/New_York"}})
	if err != nil {
		t.Fatal(err)
	}
	service := value.(*intakeInboxReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "session", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	_, listBody, err := service.DescriptionVersionList(ctx, &api.DescriptionVersionListPayload{Slug: "demo", ProjectID: intakeReadsProjectID, WorkItemID: intakeReadsObjectID})
	list, _ := decodeIntakeReadsBody[map[string]any](t, listBody, err)
	listRow := list["results"].([]any)[0].(map[string]any)
	if listRow["last_saved_at"] != "2026-01-02T03:04:05Z" || listRow["created_at"] != "2026-01-01T22:04:05-05:00" || listRow["updated_at"] != "2026-01-01T22:04:05-05:00" {
		t.Fatalf("description list timestamps=%v", listRow)
	}
	_, detailBody, err := service.DescriptionVersionDetail(ctx, &api.DescriptionVersionDetailPayload{Slug: "demo", ProjectID: intakeReadsProjectID, WorkItemID: intakeReadsObjectID, Pk: intakeReadsVersionID})
	detail, _ := decodeIntakeReadsBody[map[string]any](t, detailBody, err)
	if detail["last_saved_at"] != "2026-01-01T22:04:05-05:00" {
		t.Fatalf("description detail timestamps=%v", detail)
	}
}

func TestIntakeInboxTimesNormalizesModeledTimestampsWithoutChangingArbitraryJSON(t *testing.T) {
	containerData := jsontext.Value(`{
		"created_at":"2026-01-02T03:04:05.123000+00:00",
		"updated_at":"2026-01-02T03:04:05+00:00",
		"view_props":{"created_at":"2026-01-02T03:04:05+00:00"},
		"logo_props":{"updated_at":"2026-01-02T03:04:05+00:00"},
		"project_detail":{"logo_props":{"created_at":"2026-01-02T03:04:05+00:00"}}
	}`)
	normalized, err := intakeInboxTimes(containerData, "UTC", intakeInboxContainerShape)
	if err != nil {
		t.Fatal(err)
	}
	var container map[string]any
	if err := json.Unmarshal(normalized, &container); err != nil {
		t.Fatal(err)
	}
	if container["created_at"] != "2026-01-02T03:04:05.123000Z" || container["updated_at"] != "2026-01-02T03:04:05Z" {
		t.Fatalf("modeled UTC timestamps=%v", container)
	}
	if container["view_props"].(map[string]any)["created_at"] != "2026-01-02T03:04:05+00:00" ||
		container["logo_props"].(map[string]any)["updated_at"] != "2026-01-02T03:04:05+00:00" ||
		container["project_detail"].(map[string]any)["logo_props"].(map[string]any)["created_at"] != "2026-01-02T03:04:05+00:00" {
		t.Fatalf("arbitrary container JSON was rewritten: %v", container)
	}

	descriptionData := jsontext.Value(`{
		"created_at":"2026-01-02T03:04:05+00:00",
		"last_saved_at":"2026-01-02T03:04:05+00:00",
		"description_json":{"attrs":{"updated_at":"2026-01-02T03:04:05+00:00"}}
	}`)
	normalized, err = intakeInboxTimes(descriptionData, "America/New_York", intakeInboxDescriptionDetailShape)
	if err != nil {
		t.Fatal(err)
	}
	var description map[string]any
	if err := json.Unmarshal(normalized, &description); err != nil {
		t.Fatal(err)
	}
	if description["created_at"] != "2026-01-01T22:04:05-05:00" || description["last_saved_at"] != "2026-01-01T22:04:05-05:00" {
		t.Fatalf("description timestamps=%v", description)
	}
	if description["description_json"].(map[string]any)["attrs"].(map[string]any)["updated_at"] != "2026-01-02T03:04:05+00:00" {
		t.Fatalf("description_json was rewritten: %v", description)
	}
}

func TestDescriptionVersionListCanonicalizesUTCOffsetForLastSavedAt(t *testing.T) {
	data := jsontext.Value(`{"results":[{"last_saved_at":"2026-01-02T03:04:05+00:00","created_at":"2026-01-02T03:04:05+00:00","updated_at":"2026-01-02T03:04:05+00:00"}]}`)
	normalized, err := intakeInboxTimes(data, "America/New_York", intakeInboxDescriptionPageShape)
	if err != nil {
		t.Fatal(err)
	}
	var page map[string]any
	if err := json.Unmarshal(normalized, &page); err != nil {
		t.Fatal(err)
	}
	row := page["results"].([]any)[0].(map[string]any)
	if row["last_saved_at"] != "2026-01-02T03:04:05Z" {
		t.Fatalf("last_saved_at=%v", row["last_saved_at"])
	}
	if row["created_at"] != "2026-01-01T22:04:05-05:00" || row["updated_at"] != "2026-01-01T22:04:05-05:00" {
		t.Fatalf("audit timestamps=%v", row)
	}
}
