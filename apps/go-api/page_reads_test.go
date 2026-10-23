package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"uuid"

	pagereadssvr "github.com/CaliLuke/go-plane/gen/http/page_reads/server"
	api "github.com/CaliLuke/go-plane/gen/page_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	store "github.com/CaliLuke/go-plane/internal/pagereads"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
)

const (
	pageReadsUserID    = "00000000-0000-4000-8000-000000000001"
	pageReadsProjectID = "00000000-0000-4000-8000-000000000002"
	pageReadsPageID    = "00000000-0000-4000-8000-000000000003"
	pageReadsVersionID = "00000000-0000-4000-8000-000000000004"
	pageReadFixture    = `{"id":"00000000-0000-4000-8000-000000000003","name":"Roadmap","owned_by":"00000000-0000-4000-8000-000000000001","access":0,"color":"#123456","parent":null,"is_favorite":true,"is_locked":false,"archived_at":null,"workspace":"00000000-0000-4000-8000-000000000005","created_at":"2026-08-01T12:00:00Z","updated_at":"2026-08-02T12:00:00Z","created_by":"00000000-0000-4000-8000-000000000001","updated_by":null,"view_props":{"full_width":false},"logo_props":{"emoji":"map"},"label_ids":["00000000-0000-4000-8000-000000000006"],"project_ids":["00000000-0000-4000-8000-000000000002"]}`
	pageDetailFixture  = `{"id":"00000000-0000-4000-8000-000000000003","name":"Roadmap","owned_by":"00000000-0000-4000-8000-000000000001","access":0,"color":"#123456","parent":null,"is_favorite":true,"is_locked":false,"archived_at":null,"workspace":"00000000-0000-4000-8000-000000000005","created_at":"2026-08-01T12:00:00Z","updated_at":"2026-08-02T12:00:00Z","created_by":"00000000-0000-4000-8000-000000000001","updated_by":null,"view_props":{"full_width":false},"logo_props":{"emoji":"map"},"label_ids":[],"project_ids":["00000000-0000-4000-8000-000000000002"],"description_html":"<p>Stored HTML</p>","issue_ids":["00000000-0000-4000-8000-000000000007"]}`
	pageVersionFixture = `{"id":"00000000-0000-4000-8000-000000000004","workspace":"00000000-0000-4000-8000-000000000005","page":"00000000-0000-4000-8000-000000000003","last_saved_at":"2026-08-03T12:00:00Z","owned_by":"00000000-0000-4000-8000-000000000001","created_at":"2026-08-03T12:00:00Z","updated_at":"2026-08-03T12:00:00Z","created_by":"00000000-0000-4000-8000-000000000001","updated_by":null}`
)

type pageSummaryReadFixture struct {
	PublicPages   int `json:"public_pages"`
	PrivatePages  int `json:"private_pages"`
	ArchivedPages int `json:"archived_pages"`
}

type pageReadFixtureValue struct {
	Name       string `json:"name"`
	IsFavorite bool   `json:"is_favorite"`
}

type pageDetailReadFixture struct {
	DescriptionHTML string   `json:"description_html"`
	IssueIDs        []string `json:"issue_ids"`
}

type pageVersionReadFixture struct {
	ID string `json:"id"`
}

type pageVersionDetailReadFixture struct {
	DescriptionBinary *string        `json:"description_binary"`
	DescriptionHTML   string         `json:"description_html"`
	DescriptionJSON   map[string]any `json:"description_json"`
}

type pageReadsRepositoryStub struct {
	summary                       json.RawMessage
	list                          json.RawMessage
	detail                        json.RawMessage
	description                   []byte
	versions                      json.RawMessage
	versionDetail                 json.RawMessage
	err                           error
	search                        string
	orderBy                       string
	detailSearch                  string
	detailOrderBy                 string
	timezone                      string
	optionsPageID                 *uuid.UUID
	lifecyclePermissionErr        error
	lifecyclePermissionErrors     map[string]error
	lifecyclePermissionOperations []string
	lifecycleHandlerErr           error
	lifecycleCalls                []string
	lifecycleOperation            string
}

func (stub *pageReadsRepositoryStub) Summary(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error) {
	return stub.summary, stub.err
}

func (stub *pageReadsRepositoryStub) List(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, search, orderBy, timezone string) (json.RawMessage, error) {
	stub.search = search
	stub.orderBy = orderBy
	stub.timezone = timezone
	return stub.list, stub.err
}

func (stub *pageReadsRepositoryStub) Detail(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ uuid.UUID, search, orderBy, timezone string) (json.RawMessage, error) {
	stub.detailSearch = search
	stub.detailOrderBy = orderBy
	stub.timezone = timezone
	return stub.detail, stub.err
}

func (stub *pageReadsRepositoryStub) Description(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) ([]byte, error) {
	return stub.description, stub.err
}

func (stub *pageReadsRepositoryStub) VersionList(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ uuid.UUID, timezone string) (json.RawMessage, error) {
	stub.timezone = timezone
	return stub.versions, stub.err
}

func (stub *pageReadsRepositoryStub) VersionDetail(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ uuid.UUID, _ uuid.UUID, timezone string) (json.RawMessage, error) {
	stub.timezone = timezone
	return stub.versionDetail, stub.err
}

func (stub *pageReadsRepositoryStub) AuthorizeOptions(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, pageID *uuid.UUID) error {
	stub.optionsPageID = pageID
	return stub.err
}

func (stub *pageReadsRepositoryStub) AuthorizeLifecycle(_ context.Context, operation string, _ uuid.UUID, _ string, _ uuid.UUID, _ uuid.UUID) error {
	stub.lifecycleOperation = operation
	stub.lifecyclePermissionOperations = append(stub.lifecyclePermissionOperations, operation)
	stub.lifecycleCalls = append(stub.lifecycleCalls, "permission")
	if err, ok := stub.lifecyclePermissionErrors[operation]; ok {
		return err
	}
	return stub.lifecyclePermissionErr
}

func (stub *pageReadsRepositoryStub) ValidateLifecycleHandler(context.Context, string, uuid.UUID, string, uuid.UUID, uuid.UUID) error {
	stub.lifecycleCalls = append(stub.lifecycleCalls, "handler")
	return stub.lifecycleHandlerErr
}

func (stub *pageReadsRepositoryStub) Lifecycle(context.Context, string, uuid.UUID, string, uuid.UUID, uuid.UUID, string, djangoinput.Input) (int, json.RawMessage, error) {
	stub.lifecycleCalls = append(stub.lifecycleCalls, "dispatch")
	return 200, json.RawMessage(`{}`), nil
}

type pageReadsSessionsStub struct {
	err      error
	calls    *int
	timezone string
}

func (stub pageReadsSessionsStub) Validate(context.Context, string) (djangosession.Principal, error) {
	if stub.calls != nil {
		*stub.calls++
	}
	if stub.err != nil {
		return djangosession.Principal{}, stub.err
	}
	timezone := stub.timezone
	if timezone == "" {
		timezone = "America/New_York"
	}
	return djangosession.Principal{UserID: pageReadsUserID, UserTimezone: timezone}, nil
}

type pageRecentVisitsStub struct {
	calls      int
	userID     uuid.UUID
	slug       string
	entityName string
	entityID   uuid.UUID
	projectID  uuid.UUID
	err        error
}

func (stub *pageRecentVisitsStub) Record(_ context.Context, userID uuid.UUID, slug, entityName string, entityID, projectID uuid.UUID) error {
	stub.calls++
	stub.userID = userID
	stub.slug = slug
	stub.entityName = entityName
	stub.entityID = entityID
	stub.projectID = projectID
	return stub.err
}

func TestNewPageReadsRequiresDependencies(t *testing.T) {
	repository := &pageReadsRepositoryStub{}
	sessions := pageReadsSessionsStub{}
	visits := &pageRecentVisitsStub{}
	if _, err := NewPageReads(nil, sessions, visits); err == nil {
		t.Fatal("missing repository returned nil error")
	}
	if _, err := NewPageReads(repository, nil, visits); err == nil {
		t.Fatal("missing sessions returned nil error")
	}
	if _, err := NewPageReads(repository, sessions, nil); err == nil {
		t.Fatal("missing recent visits returned nil error")
	}
}

func TestPageReadsSummaryAndFavoriteOrderedList(t *testing.T) {
	repository := &pageReadsRepositoryStub{
		summary: json.RawMessage(`{"public_pages":4,"private_pages":2,"archived_pages":1}`),
		list:    json.RawMessage(`[` + pageReadFixture + `]`),
	}
	service, ctx := authenticatedPageReadsService(t, repository, &pageRecentVisitsStub{})

	summary := decodePageResponse[pageSummaryReadFixture](t, capturePageResponse(
		service.Summary(ctx, &api.SummaryPayload{Slug: "demo", ProjectID: pageReadsProjectID}),
	))
	if summary.PublicPages != 4 || summary.PrivatePages != 2 || summary.ArchivedPages != 1 {
		t.Fatalf("summary=%+v", summary)
	}
	search, orderBy := "road", "name"
	pages := decodePageResponse[[]pageReadFixtureValue](t, capturePageResponse(service.List(ctx, &api.ListPayload{
		Slug: "demo", ProjectID: pageReadsProjectID, Search: &search, OrderBy: &orderBy,
	})))
	if len(pages) != 1 || pages[0].Name != "Roadmap" || !pages[0].IsFavorite {
		t.Fatalf("pages=%+v", pages)
	}
	if repository.search != "road" || repository.orderBy != "name" || repository.timezone != "America/New_York" {
		t.Fatalf("search=%q orderBy=%q timezone=%q", repository.search, repository.orderBy, repository.timezone)
	}
}
func TestPageReadsPreservesExplicitEmptyOrderByForDjangoFailure(t *testing.T) {
	repository := &pageReadsRepositoryStub{list: json.RawMessage(`[]`)}
	service, ctx := authenticatedPageReadsService(t, repository, &pageRecentVisitsStub{})
	empty := ""
	_ = decodePageResponse[[]pageReadFixtureValue](t, capturePageResponse(service.List(ctx, &api.ListPayload{
		Slug: "demo", ProjectID: pageReadsProjectID, OrderBy: &empty,
	})))
	if repository.orderBy != "" {
		t.Fatalf("orderBy=%q, want explicit empty", repository.orderBy)
	}
	ctx = WithReadServiceQueryPresence(ctx, url.Values{"order_by": {""}})
	_ = decodePageResponse[[]pageReadFixtureValue](t, capturePageResponse(service.List(ctx, &api.ListPayload{
		Slug: "demo", ProjectID: pageReadsProjectID,
	})))
	if repository.orderBy != "" {
		t.Fatalf("decoded empty orderBy=%q, want explicit empty", repository.orderBy)
	}
}
func TestPageReadsLeavesInvalidOrderUnhandledForDjangoHTML500(t *testing.T) {
	service, ctx := authenticatedPageReadsService(
		t,
		&pageReadsRepositoryStub{err: store.ErrInvalidOrder},
		&pageRecentVisitsStub{},
	)
	_, _, err := service.List(ctx, &api.ListPayload{Slug: "demo", ProjectID: pageReadsProjectID})
	if !errors.Is(err, store.ErrInvalidOrder) {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestPageReadsDetailLeavesInvalidOrderUnhandledForDjangoHTML500(t *testing.T) {
	service, ctx := authenticatedPageReadsService(
		t,
		&pageReadsRepositoryStub{err: store.ErrInvalidOrder},
		&pageRecentVisitsStub{},
	)
	invalid := ""
	_, _, err := service.Detail(ctx, &api.DetailPayload{
		Slug: "demo", ProjectID: pageReadsProjectID, PageID: pageReadsPageID, OrderBy: &invalid,
	})
	if !errors.Is(err, store.ErrInvalidOrder) {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestPageReadsMapsNullSearchValidationToBadRequest(t *testing.T) {
	service, ctx := authenticatedPageReadsService(
		t,
		&pageReadsRepositoryStub{err: store.ErrInvalidSearch},
		&pageRecentVisitsStub{},
	)
	_, _, err := service.List(ctx, &api.ListPayload{Slug: "demo", ProjectID: pageReadsProjectID})
	var invalidSearch api.InvalidSearch
	if !errors.As(err, &invalidSearch) || len(invalidSearch) != 1 || invalidSearch[0] != "Null characters are not allowed." {
		t.Fatalf("list err=%T %v", err, err)
	}
	_, _, err = service.Detail(ctx, &api.DetailPayload{
		Slug: "demo", ProjectID: pageReadsProjectID, PageID: pageReadsPageID,
	})
	if !errors.As(err, &invalidSearch) || len(invalidSearch) != 1 || invalidSearch[0] != "Null characters are not allowed." {
		t.Fatalf("detail err=%T %v", err, err)
	}
}

func TestPageReadsOptionsUseProjectPagePermissionAndDjangoMetadata(t *testing.T) {
	repository := &pageReadsRepositoryStub{}
	service, ctx := authenticatedPageReadsService(t, repository, &pageRecentVisitsStub{})

	summary := decodePageResponse[map[string]any](t, capturePageResponse(service.SummaryOptions(ctx, &api.SummaryOptionsPayload{
		Slug: "demo", ProjectID: pageReadsProjectID,
	})))
	if summary["name"] != "Page" || repository.optionsPageID != nil {
		t.Fatalf("summary metadata=%#v pageID=%v", summary, repository.optionsPageID)
	}

	versionList := decodePageResponse[map[string]any](t, capturePageResponse(service.VersionListOptions(ctx, &api.VersionListOptionsPayload{
		Slug: "demo", ProjectID: pageReadsProjectID, PageID: pageReadsPageID,
	})))
	if versionList["name"] != "Page Version Endpoint" ||
		repository.optionsPageID == nil || repository.optionsPageID.String() != pageReadsPageID {
		t.Fatalf("version metadata=%#v pageID=%v", versionList, repository.optionsPageID)
	}

	_ = decodePageResponse[map[string]any](t, capturePageResponse(service.VersionDetailOptions(ctx, &api.VersionDetailOptionsPayload{
		Slug: "demo", ProjectID: pageReadsProjectID, PageID: pageReadsPageID, Pk: "opaque-to-permission",
	})))

	repository.err = store.ErrForbidden
	_, _, err := service.SummaryOptions(ctx, &api.SummaryOptionsPayload{Slug: "demo", ProjectID: pageReadsProjectID})
	var forbidden *api.PageReadsForbidden
	if !errors.As(err, &forbidden) {
		t.Fatalf("denied OPTIONS err=%T %v", err, err)
	}
}

func TestPageLifecycleOptionsGeneratedTransportPreservesAuthorizationAndTimezoneErrors(t *testing.T) {
	for _, test := range []struct {
		name       string
		timezone   string
		permission error
		wantStatus int
	}{
		{name: "invalid timezone after role-five permission", timezone: "Mars/Olympus", wantStatus: http.StatusBadRequest},
		{name: "permission denial before invalid timezone", timezone: "Mars/Olympus", permission: store.ErrForbidden, wantStatus: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &pageReadsRepositoryStub{lifecyclePermissionErr: test.permission}
			serviceValue, err := NewPageReads(repository, pageReadsSessionsStub{timezone: test.timezone}, &pageRecentVisitsStub{})
			if err != nil {
				t.Fatal(err)
			}
			mux := loomhttp.NewMuxer()
			server := pagereadssvr.New(api.NewEndpoints(serviceValue), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
			pagereadssvr.Mount(mux, server)
			request := httptest.NewRequest(http.MethodOptions, "/api/workspaces/demo/projects/"+pageReadsProjectID+"/pages/", nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
			request = request.WithContext(WithReadServiceSessionPrincipal(request.Context(), pageReadsUserID, test.timezone))
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status=%d body=%q, want %d", response.Code, response.Body.String(), test.wantStatus)
			}
			if repository.lifecycleOperation != "options" {
				t.Fatalf("authorization operation=%q, want options", repository.lifecycleOperation)
			}
			if response.Code == http.StatusInternalServerError {
				t.Fatalf("generated transport failed to encode declared OPTIONS error: %s", response.Body.String())
			}
		})
	}
}

func TestPageLifecycleOptionsOmitsPostSchemaWhenSourcePermissionDeniesAction(t *testing.T) {
	repository := &pageReadsRepositoryStub{
		lifecyclePermissionErrors: map[string]error{"create": store.ErrForbidden},
	}
	service, ctx := authenticatedPageReadsService(t, repository, &pageRecentVisitsStub{})
	ctx = WithReadServiceSessionPrincipal(ctx, pageReadsUserID, "UTC")
	response := capturePageResponse(service.CreateOptions(ctx, &api.CreateOptionsPayload{
		Slug: "demo", ProjectID: pageReadsProjectID,
	}))
	metadata := decodePageResponse[map[string]any](t, response)
	if _, exists := metadata["actions"]; exists {
		t.Fatalf("metadata includes an unavailable POST action: %#v", metadata)
	}
	if !reflect.DeepEqual(repository.lifecyclePermissionOperations, []string{"options", "create"}) {
		t.Fatalf("permission operations=%v, want options then create", repository.lifecyclePermissionOperations)
	}
}

func TestPageLifecycleOptionsPreserveSourceRenderedBodyLengths(t *testing.T) {
	for _, test := range []struct {
		action string
		want   int
	}{
		{action: "detail", want: 151},
		{action: "description", want: 164},
		{action: "duplicate", want: 170},
	} {
		t.Run(test.action, func(t *testing.T) {
			repository := &pageReadsRepositoryStub{}
			service, ctx := authenticatedPageReadsService(t, repository, &pageRecentVisitsStub{})
			ctx = WithReadServiceSessionPrincipal(ctx, pageReadsUserID, "UTC")
			metadata, body, err := service.pageLifecycleOptions(ctx, "demo", pageReadsProjectID, pageReadsPageID, test.action)
			if err != nil {
				t.Fatal(err)
			}
			defer body.Close()
			payload, err := io.ReadAll(body)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(payload); got != test.want {
				t.Fatalf("metadata body length=%d, want source-rendered %d (content type %q)", got, test.want, metadata.ContentType)
			}
		})
	}
}

func TestPageReadsDetailTracksDefaultAndTrueButHonorsFalse(t *testing.T) {
	repository := &pageReadsRepositoryStub{detail: json.RawMessage(pageDetailFixture)}
	visits := &pageRecentVisitsStub{err: errors.New("best effort failure")}
	service, ctx := authenticatedPageReadsService(t, repository, visits)

	search := "Roadmap"
	orderBy := "owned_by__email"
	page := decodePageResponse[pageDetailReadFixture](t, capturePageResponse(service.Detail(ctx, &api.DetailPayload{
		Slug: "demo", ProjectID: pageReadsProjectID, PageID: pageReadsPageID, Search: &search, OrderBy: &orderBy,
	})))
	if page.DescriptionHTML != "<p>Stored HTML</p>" || len(page.IssueIDs) != 1 {
		t.Fatalf("page=%+v", page)
	}
	if repository.detailSearch != "Roadmap" || repository.detailOrderBy != "owned_by__email" ||
		repository.timezone != "America/New_York" {
		t.Fatalf("detail search=%q orderBy=%q timezone=%q", repository.detailSearch, repository.detailOrderBy, repository.timezone)
	}
	if visits.calls != 1 || visits.userID.String() != pageReadsUserID || visits.slug != "demo" || visits.entityName != "page" || visits.entityID.String() != pageReadsPageID || visits.projectID.String() != pageReadsProjectID {
		t.Fatalf("visit=%+v", visits)
	}

	trackVisit := "TRUE"
	_ = decodePageResponse[pageDetailReadFixture](t, capturePageResponse(service.Detail(ctx, &api.DetailPayload{
		Slug: "demo", ProjectID: pageReadsProjectID, PageID: pageReadsPageID, TrackVisit: &trackVisit,
	})))
	if visits.calls != 2 {
		t.Fatalf("track_visit=true calls=%d", visits.calls)
	}
	for _, value := range []string{"false", "1", ""} {
		trackVisit = value
		_ = decodePageResponse[pageDetailReadFixture](t, capturePageResponse(service.Detail(ctx, &api.DetailPayload{
			Slug: "demo", ProjectID: pageReadsProjectID, PageID: pageReadsPageID, TrackVisit: &trackVisit,
		})))
		if visits.calls != 2 {
			t.Fatalf("track_visit=%q recorded %d visits", value, visits.calls)
		}
	}
	explicitEmptyCtx := WithReadServiceQueryPresence(ctx, url.Values{"track_visit": {""}})
	_ = decodePageResponse[pageDetailReadFixture](t, capturePageResponse(service.Detail(explicitEmptyCtx, &api.DetailPayload{
		Slug: "demo", ProjectID: pageReadsProjectID, PageID: pageReadsPageID,
	})))
	if visits.calls != 2 {
		t.Fatalf("decoded empty track_visit recorded %d visits", visits.calls)
	}
}

func TestPageReadsDescriptionReturnsRawAttachmentMetadata(t *testing.T) {
	repository := &pageReadsRepositoryStub{description: []byte{0x00, 0x01, 0xff, 0x02}}
	service, ctx := authenticatedPageReadsService(t, repository, &pageRecentVisitsStub{})
	metadata, file, err := service.Description(ctx, &api.DescriptionPayload{
		Slug: "demo", ProjectID: pageReadsProjectID, PageID: pageReadsPageID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Disposition != `attachment; filename="page_description.bin"` || file.Name != "page_description.bin" || !file.ModTime.IsZero() {
		t.Fatalf("metadata=%+v file=%+v", metadata, file)
	}
	data, err := io.ReadAll(file.Content)
	if err != nil || string(data) != string(repository.description) {
		t.Fatalf("data=%v err=%v", data, err)
	}
}

func TestPageReadsVersionsPreserveStoredContent(t *testing.T) {
	detail := pageVersionFixture[:len(pageVersionFixture)-1] + `,"description_binary":"AAH/Ag==","description_html":"<p>Version HTML</p>","description_json":{"type":"doc"}}`
	repository := &pageReadsRepositoryStub{
		versions:      json.RawMessage(`[` + pageVersionFixture + `]`),
		versionDetail: json.RawMessage(detail),
	}
	service, ctx := authenticatedPageReadsService(t, repository, &pageRecentVisitsStub{})
	ignoredSearch := "not applied"
	versions := decodePageResponse[[]pageVersionReadFixture](t, capturePageResponse(service.VersionList(ctx, &api.VersionListPayload{
		Slug: "demo", ProjectID: pageReadsProjectID, PageID: pageReadsPageID, Search: &ignoredSearch,
	})))
	if len(versions) != 1 || versions[0].ID != pageReadsVersionID {
		t.Fatalf("versions=%+v", versions)
	}
	version := decodePageResponse[pageVersionDetailReadFixture](t, capturePageResponse(service.VersionDetail(ctx, &api.VersionDetailPayload{
		Slug: "demo", ProjectID: pageReadsProjectID, PageID: pageReadsPageID, Pk: pageReadsVersionID,
	})))
	if version.DescriptionHTML != "<p>Version HTML</p>" {
		t.Fatalf("version=%+v", version)
	}
	if version.DescriptionBinary == nil || *version.DescriptionBinary != "AAH/Ag==" {
		t.Fatalf("description_binary=%v", version.DescriptionBinary)
	}
	if version.DescriptionJSON["type"] != "doc" {
		t.Fatalf("description_json=%#v", version.DescriptionJSON)
	}
	nullDetail := pageVersionFixture[:len(pageVersionFixture)-1] + `,"description_binary":null,"description_html":"<p></p>","description_json":{}}`
	repository.versionDetail = json.RawMessage(nullDetail)
	version = decodePageResponse[pageVersionDetailReadFixture](t, capturePageResponse(service.VersionDetail(ctx, &api.VersionDetailPayload{
		Slug: "demo", ProjectID: pageReadsProjectID, PageID: pageReadsPageID, Pk: pageReadsVersionID,
	})))
	if version.DescriptionBinary != nil {
		t.Fatalf("nullable version=%+v", version)
	}
}

func TestPageReadsMapsAuthorizationGuestAndNotFoundErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want any
	}{
		{name: "forbidden", err: store.ErrForbidden, want: &api.PageReadsForbidden{}},
		{name: "restricted guest", err: store.ErrRestrictedGuest, want: &api.PageReadsBadRequest{}},
		{name: "not found", err: store.ErrNotFound, want: &api.PageReadsNotFound{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			visits := &pageRecentVisitsStub{}
			service, ctx := authenticatedPageReadsService(t, &pageReadsRepositoryStub{err: test.err}, visits)
			_, _, err := service.Detail(ctx, &api.DetailPayload{Slug: "demo", ProjectID: pageReadsProjectID, PageID: pageReadsPageID})
			switch test.want.(type) {
			case *api.PageReadsForbidden:
				var target *api.PageReadsForbidden
				if !errors.As(err, &target) {
					t.Fatalf("err=%T %v", err, err)
				}
			case *api.PageReadsBadRequest:
				var target *api.PageReadsBadRequest
				if !errors.As(err, &target) || target.Message != "You are not allowed to view this page" {
					t.Fatalf("err=%T %v", err, err)
				}
			case *api.PageReadsNotFound:
				var target *api.PageReadsNotFound
				if !errors.As(err, &target) || target.Message != "Page not found" {
					t.Fatalf("err=%T %v", err, err)
				}
			}
			if visits.calls != 0 {
				t.Fatalf("failed detail recorded %d visits", visits.calls)
			}
		})
	}
}
func TestPageReadsLeavesDjangoNilDereferenceUnhandled(t *testing.T) {
	service, ctx := authenticatedPageReadsService(
		t,
		&pageReadsRepositoryStub{err: store.ErrDjangoPageNil},
		&pageRecentVisitsStub{},
	)
	_, _, err := service.Detail(ctx, &api.DetailPayload{
		Slug: "demo", ProjectID: pageReadsProjectID, PageID: pageReadsPageID,
	})
	if !errors.Is(err, store.ErrDjangoPageNil) {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestPageReadsRejectInvalidSession(t *testing.T) {
	service := &pageReadsService{sessions: pageReadsSessionsStub{err: errors.New("invalid")}}
	_, err := service.APIKeyAuth(context.Background(), "bad", &security.APIKeyScheme{})
	var unauthenticated *api.PageReadsUnauthenticated
	if !errors.As(err, &unauthenticated) {
		t.Fatalf("err=%T %v", err, err)
	}
}
func TestPageReadsUsesPrevalidatedPrincipalWithoutSecondSessionRead(t *testing.T) {
	repository := &pageReadsRepositoryStub{list: json.RawMessage(`[]`)}
	calls := 0
	serviceValue, err := NewPageReads(repository, pageReadsSessionsStub{calls: &calls}, &pageRecentVisitsStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithReadServiceSessionPrincipal(context.Background(), pageReadsUserID, "Asia/Kolkata")
	ctx, err = serviceValue.(*pageReadsService).APIKeyAuth(ctx, "already-inspected", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	_ = decodePageResponse[[]pageReadFixtureValue](t, capturePageResponse(serviceValue.(*pageReadsService).List(ctx, &api.ListPayload{
		Slug: "demo", ProjectID: pageReadsProjectID,
	})))
	if calls != 0 || repository.timezone != "Asia/Kolkata" {
		t.Fatalf("session calls=%d timezone=%q", calls, repository.timezone)
	}
}

func TestPageReadsDefaultsEmptyTimezoneToUTC(t *testing.T) {
	repository := &pageReadsRepositoryStub{list: json.RawMessage(`[]`)}
	serviceValue, err := NewPageReads(repository, pageReadsSessionsStub{}, &pageRecentVisitsStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithReadServiceSessionPrincipal(context.Background(), pageReadsUserID, "")
	ctx, err = serviceValue.(*pageReadsService).APIKeyAuth(ctx, "already-inspected", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	_ = decodePageResponse[[]pageReadFixtureValue](t, capturePageResponse(serviceValue.(*pageReadsService).List(ctx, &api.ListPayload{
		Slug: "demo", ProjectID: pageReadsProjectID,
	})))
	if repository.timezone != "UTC" {
		t.Fatalf("timezone=%q", repository.timezone)
	}
}

func TestPageLifecyclePermissionTimezoneAndHandlerOrder(t *testing.T) {
	for _, test := range []struct {
		name           string
		timezone       string
		permissionErr  error
		handlerErr     error
		wantStatusKind string
		wantBody       string
		wantCalls      []string
	}{
		{
			name:           "permission denial precedes invalid timezone",
			timezone:       "Mars/Olympus",
			permissionErr:  &store.LifecycleError{Status: 403, Body: map[string]any{"detail": "You do not have permission to perform this action."}},
			wantStatusKind: "status_403",
			wantBody:       `{"detail":"You do not have permission to perform this action."}`,
			wantCalls:      []string{"permission"},
		},
		{
			name:           "invalid timezone precedes locked handler error",
			timezone:       "Mars/Olympus",
			handlerErr:     &store.LifecycleError{Status: 400, Body: map[string]any{"error": "Page is locked"}},
			wantStatusKind: "status_400",
			wantBody:       `{"error":"The required key does not exist."}`,
			wantCalls:      []string{"permission"},
		},
		{
			name:           "locked handler error follows valid timezone",
			timezone:       "UTC",
			handlerErr:     &store.LifecycleError{Status: 400, Body: map[string]any{"error": "Page is locked"}},
			wantStatusKind: "status_400",
			wantBody:       `{"error":"Page is locked"}`,
			wantCalls:      []string{"permission", "handler"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &pageReadsRepositoryStub{lifecyclePermissionErr: test.permissionErr, lifecycleHandlerErr: test.handlerErr}
			serviceValue, err := NewPageReads(repository, pageReadsSessionsStub{}, &pageRecentVisitsStub{})
			if err != nil {
				t.Fatal(err)
			}
			ctx := WithReadServiceSessionPrincipal(context.Background(), pageReadsUserID, test.timezone)
			ctx, err = serviceValue.(*pageReadsService).APIKeyAuth(ctx, "already-inspected", &security.APIKeyScheme{})
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = serviceValue.(*pageReadsService).Update(ctx, &api.UpdatePayload{Slug: "demo", ProjectID: pageReadsProjectID, PageID: pageReadsPageID}, nil)
			var lifecycle *api.PageLifecycleError
			if !errors.As(err, &lifecycle) || lifecycle.Kind != test.wantStatusKind {
				t.Fatalf("error=%T %v, want %s", err, err, test.wantStatusKind)
			}
			body, marshalErr := json.Marshal(lifecycle.Body)
			if marshalErr != nil || string(body) != test.wantBody {
				t.Fatalf("error body=%s, want %s (marshal err %v)", body, test.wantBody, marshalErr)
			}
			if !reflect.DeepEqual(repository.lifecycleCalls, test.wantCalls) {
				t.Fatalf("lifecycle call order=%v, want %v", repository.lifecycleCalls, test.wantCalls)
			}
		})
	}
}

func TestPageLifecycleUnsupportedMethodPermissionPrecedesTimezone(t *testing.T) {
	for _, test := range []struct {
		name             string
		permissionErr    error
		wantStatus       int
		wantPermissionOp string
	}{
		{name: "guest POST to detail denied before invalid timezone", permissionErr: store.ErrForbidden, wantStatus: http.StatusForbidden, wantPermissionOp: "update"},
		{name: "authorized unsupported method reaches timezone validation", wantStatus: http.StatusBadRequest, wantPermissionOp: "update"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &pageReadsRepositoryStub{lifecyclePermissionErr: test.permissionErr}
			serviceValue, err := NewPageReads(repository, pageReadsSessionsStub{}, &pageRecentVisitsStub{})
			if err != nil {
				t.Fatal(err)
			}
			ctx := WithReadServiceSessionPrincipal(context.Background(), pageReadsUserID, "Mars/Olympus")
			ctx, err = serviceValue.(*pageReadsService).APIKeyAuth(ctx, "already-inspected", nil)
			if err != nil {
				t.Fatal(err)
			}
			status, body := serviceValue.(*pageReadsService).AuthorizeSessionResourceProtocol(ctx, "already-inspected", SessionProtocolResource{
				Method: http.MethodPost, Pattern: "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/",
				Slug: "demo", ProjectID: pageReadsProjectID, PathParameters: map[string]string{"page_id": pageReadsPageID},
			})
			if status != test.wantStatus || repository.lifecycleOperation != test.wantPermissionOp {
				t.Fatalf("status=%d body=%v operation=%q, want status=%d operation=%q", status, body, repository.lifecycleOperation, test.wantStatus, test.wantPermissionOp)
			}
			if status == http.StatusForbidden && !reflect.DeepEqual(body, map[string]any{"detail": "You do not have permission to perform this action."}) {
				t.Fatalf("permission body=%v", body)
			}
			if !reflect.DeepEqual(repository.lifecycleCalls, []string{"permission"}) {
				t.Fatalf("lifecycle calls=%v, want permission only", repository.lifecycleCalls)
			}
		})
	}
}

type pageResponse struct {
	metadata *api.RawJSONResponseMetadata
	body     io.ReadCloser
	err      error
}

func capturePageResponse(metadata *api.RawJSONResponseMetadata, body io.ReadCloser, err error) pageResponse {
	return pageResponse{metadata: metadata, body: body, err: err}
}

func decodePageResponse[T any](t *testing.T, response pageResponse) T {
	t.Helper()
	var result T
	if response.err != nil {
		t.Fatal(response.err)
	}
	if response.metadata == nil || response.metadata.ContentType != pageJSONContentType {
		t.Fatalf("metadata=%+v", response.metadata)
	}
	defer response.body.Close()
	if err := json.NewDecoder(response.body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

func authenticatedPageReadsService(
	t *testing.T,
	repository *pageReadsRepositoryStub,
	visits *pageRecentVisitsStub,
) (*pageReadsService, context.Context) {
	t.Helper()
	serviceValue, err := NewPageReads(repository, pageReadsSessionsStub{}, visits)
	if err != nil {
		t.Fatal(err)
	}
	service := serviceValue.(*pageReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "session", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	return service, ctx
}
