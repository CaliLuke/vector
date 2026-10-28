package systemapi

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"io"
	"net/url"
	"testing"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/workspace_project_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	store "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	"github.com/CaliLuke/loom/security"
)

type workspaceProjectRepositoryStub struct {
	userWorkspaces jsontext.Value
	workspaces     jsontext.Value
	workspace      jsontext.Value
	stats          jsontext.Value
	projects       jsontext.Value
	details        jsontext.Value
	project        jsontext.Value
	err            error
	workspaceOpts  store.WorkspaceOptions
	fields         []string
	projectIDs     *string
	detailsOpts    store.ProjectDetailsOptions
	called         string
}

func (stub *workspaceProjectRepositoryStub) UserWorkspaces(_ context.Context, _ uuid.UUID, options store.WorkspaceOptions) (jsontext.Value, error) {
	stub.called, stub.workspaceOpts = "user_workspaces", options
	return stub.userWorkspaces, stub.err
}
func (stub *workspaceProjectRepositoryStub) Workspaces(_ context.Context, _ uuid.UUID, options store.WorkspaceOptions) (jsontext.Value, error) {
	stub.called, stub.workspaceOpts = "workspaces", options
	return stub.workspaces, stub.err
}
func (stub *workspaceProjectRepositoryStub) Workspace(_ context.Context, _ uuid.UUID, _ string, options store.WorkspaceOptions) (jsontext.Value, error) {
	stub.called, stub.workspaceOpts = "workspace", options
	return stub.workspace, stub.err
}
func (stub *workspaceProjectRepositoryStub) ProjectStats(_ context.Context, _ uuid.UUID, _ string, fields []string, projectIDs *string) (jsontext.Value, error) {
	stub.called, stub.fields, stub.projectIDs = "project_stats", fields, projectIDs
	return stub.stats, stub.err
}
func (stub *workspaceProjectRepositoryStub) Projects(context.Context, uuid.UUID, string) (jsontext.Value, error) {
	stub.called = "projects"
	return stub.projects, stub.err
}
func (stub *workspaceProjectRepositoryStub) ProjectDetails(_ context.Context, _ uuid.UUID, _ string, options store.ProjectDetailsOptions) (jsontext.Value, error) {
	stub.called, stub.detailsOpts = "project_details", options
	return stub.details, stub.err
}
func (stub *workspaceProjectRepositoryStub) Project(context.Context, uuid.UUID, string, uuid.UUID) (jsontext.Value, error) {
	stub.called = "project"
	return stub.project, stub.err
}

type workspaceProjectSessionsStub struct{ err error }

func (stub workspaceProjectSessionsStub) Validate(context.Context, string) (djangosession.Principal, error) {
	if stub.err != nil {
		return djangosession.Principal{}, stub.err
	}
	return djangosession.Principal{UserID: "00000000-0000-4000-8000-000000000010"}, nil
}

type workspaceProjectRecentVisitsStub struct {
	calls      int
	user       uuid.UUID
	slug       string
	entityName string
	entityID   uuid.UUID
	projectID  uuid.UUID
	err        error
}

func (stub *workspaceProjectRecentVisitsStub) Record(_ context.Context, user uuid.UUID, slug, entityName string, entityID, projectID uuid.UUID) error {
	stub.calls++
	stub.user, stub.slug, stub.entityName, stub.entityID, stub.projectID = user, slug, entityName, entityID, projectID
	return stub.err
}

func authenticatedWorkspaceProjectContext() context.Context {
	return context.WithValue(context.Background(), workspaceProjectReadsPrincipal{}, uuid.MustParse("00000000-0000-4000-8000-000000000010"))
}

func workspaceProjectResponseBody(t *testing.T, metadata *api.RawJSONResponseMetadata, response io.ReadCloser) string {
	t.Helper()
	if metadata == nil || metadata.ContentType != "application/json" {
		t.Fatalf("raw response metadata = %#v", metadata)
	}
	defer testsupport.Close(t, response)
	body, err := io.ReadAll(response)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestNewWorkspaceProjectReadsRequiresAllDependencies(t *testing.T) {
	repository := &workspaceProjectRepositoryStub{}
	sessions := workspaceProjectSessionsStub{}
	visits := &workspaceProjectRecentVisitsStub{}
	for _, testCase := range []struct {
		name       string
		repository workspaceProjectReadsRepository
		sessions   workspaceProjectReadsSessions
		visits     workspaceProjectRecentVisits
	}{
		{name: "repository", sessions: sessions, visits: visits},
		{name: "sessions", repository: repository, visits: visits},
		{name: "recent visits", repository: repository, sessions: sessions},
	} {
		if _, err := NewWorkspaceProjectReads(testCase.repository, testCase.sessions, testCase.visits); err == nil {
			t.Errorf("missing %s returned nil error", testCase.name)
		}
	}
}

func TestWorkspaceProjectReadsAuthenticationAndMetadata(t *testing.T) {
	service := &workspaceProjectReadsService{repository: &workspaceProjectRepositoryStub{}, sessions: workspaceProjectSessionsStub{}, recentVisits: &workspaceProjectRecentVisitsStub{}}
	ctx, err := service.APIKeyAuth(context.Background(), "session", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ctx.Value(workspaceProjectReadsPrincipal{}).(uuid.UUID); !ok {
		t.Fatal("authenticated context does not contain UUID principal")
	}
	service.sessions = workspaceProjectSessionsStub{err: errors.New("invalid")}
	_, err = service.APIKeyAuth(context.Background(), "bad", &security.APIKeyScheme{})
	var unauthenticated *api.WorkspaceProjectUnauthenticated
	if !errors.As(err, &unauthenticated) {
		t.Fatalf("invalid session error = %T %v", err, err)
	}
	prevalidated := WithReadServiceSessionPrincipal(context.Background(), "00000000-0000-4000-8000-000000000010", "UTC")
	if _, err := service.APIKeyAuth(prevalidated, "already-validated", &security.APIKeyScheme{}); err != nil {
		t.Fatalf("prevalidated transport principal error = %v", err)
	}
	if err := service.AuthorizeMetadata(context.Background(), "bad"); !errors.As(err, &unauthenticated) {
		t.Fatalf("metadata auth error = %T %v", err, err)
	}
}

func TestWorkspaceProjectOptionsRequireAuthenticationAndReturnDjangoMetadata(t *testing.T) {
	service := &workspaceProjectReadsService{repository: &workspaceProjectRepositoryStub{}, sessions: workspaceProjectSessionsStub{}, recentVisits: &workspaceProjectRecentVisitsStub{}}
	ctx := authenticatedWorkspaceProjectContext()
	tests := []struct {
		name string
		call func() (*api.WorkspaceProjectOptions, error)
		want string
	}{
		{name: "user workspaces", call: func() (*api.WorkspaceProjectOptions, error) {
			return service.UserWorkspacesOptions(ctx, &api.UserWorkspacesOptionsPayload{})
		}, want: "User Work Spaces Endpoint"},
		{name: "project stats", call: func() (*api.WorkspaceProjectOptions, error) {
			return service.ProjectStatsOptions(ctx, &api.ProjectStatsOptionsPayload{Slug: "demo"})
		}, want: "Project Stats Endpoint"},
		{name: "project details", call: func() (*api.WorkspaceProjectOptions, error) {
			return service.ProjectDetailsOptions(ctx, &api.ProjectDetailsOptionsPayload{Slug: "demo"})
		}, want: "Project"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			metadata, err := testCase.call()
			if err != nil || metadata.Name != testCase.want || metadata.Description != "" {
				t.Fatalf("OPTIONS metadata = %#v, %v", metadata, err)
			}
			if len(metadata.Renders) != 1 || metadata.Renders[0] != "application/json" {
				t.Fatalf("OPTIONS renders = %v", metadata.Renders)
			}
			if len(metadata.Parses) != 3 || metadata.Parses[0] != "application/json" || metadata.Parses[1] != "application/x-www-form-urlencoded" || metadata.Parses[2] != "multipart/form-data" {
				t.Fatalf("OPTIONS parses = %v", metadata.Parses)
			}
		})
	}
	_, err := service.UserWorkspacesOptions(context.Background(), &api.UserWorkspacesOptionsPayload{})
	var unauthenticated *api.WorkspaceProjectUnauthenticated
	if !errors.As(err, &unauthenticated) {
		t.Fatalf("anonymous OPTIONS error = %T %v", err, err)
	}
}

func TestWorkspaceListsReturnAuthorizedRowsAndPreserveQueryPrecedence(t *testing.T) {
	workspace := `[{
		"id":"00000000-0000-4000-8000-000000000001",
		"name":"Plane","slug":"plane","owner":"00000000-0000-4000-8000-000000000002",
		"background_color":"#ffffff","created_at":"2026-01-01T00:00:00Z","created_by":null,
		"deleted_at":null,"logo":null,"logo_asset":null,"logo_url":null,"organization_size":null,
		"timezone":"UTC","total_members":3,"role":20,"updated_at":"2026-01-01T00:00:00Z","updated_by":null
	}]`
	repository := &workspaceProjectRepositoryStub{
		userWorkspaces: jsontext.Value(workspace),
		workspaces:     jsontext.Value(workspace),
		workspace:      jsontext.Value(workspace[1 : len(workspace)-1]),
	}
	service := &workspaceProjectReadsService{repository: repository, sessions: workspaceProjectSessionsStub{}, recentVisits: &workspaceProjectRecentVisitsStub{}}
	owner := "00000000-0000-4000-8000-000000000002"
	search, fields := `"Core Team", Plane%_`, "name"
	metadata, result, err := service.UserWorkspaces(authenticatedWorkspaceProjectContext(), &api.UserWorkspacesPayload{Owner: &owner, Search: &search, Fields: &fields})
	if err != nil || workspaceProjectResponseBody(t, metadata, result) != workspace {
		t.Fatalf("UserWorkspaces() body mismatch: %v", err)
	}
	if repository.workspaceOpts.Owner == nil || *repository.workspaceOpts.Owner != uuid.MustParse(owner) || len(repository.workspaceOpts.SearchTerms) != 2 || repository.workspaceOpts.SearchTerms[0] != "Core Team" || repository.workspaceOpts.SearchTerms[1] != `Plane\%\_` {
		t.Fatalf("user-workspace options = %#v", repository.workspaceOpts)
	}
	metadata, workspaceResult, err := service.Workspaces(authenticatedWorkspaceProjectContext(), &api.WorkspacesPayload{Owner: &owner, Search: &search})
	if err != nil || workspaceProjectResponseBody(t, metadata, workspaceResult) != workspace || repository.called != "workspaces" {
		t.Fatalf("fixed authenticated /api/workspaces/ called=%s, err=%v", repository.called, err)
	}
	if len(repository.workspaceOpts.SearchTerms) != 2 || repository.workspaceOpts.SearchTerms[0] != "Core Team" || repository.workspaceOpts.SearchTerms[1] != `Plane\%\_` {
		t.Fatalf("workspace-list options = %#v", repository.workspaceOpts)
	}
	metadata, detailResult, err := service.Workspace(authenticatedWorkspaceProjectContext(), &api.WorkspacePayload{Slug: "plane", Owner: &owner, Search: &search})
	if err != nil || workspaceProjectResponseBody(t, metadata, detailResult) != workspace[1:len(workspace)-1] {
		t.Fatalf("Workspace() body mismatch: %v", err)
	}
	if len(repository.workspaceOpts.SearchTerms) != 2 || repository.workspaceOpts.SearchTerms[0] != "Core Team" || repository.workspaceOpts.SearchTerms[1] != `Plane\%\_` {
		t.Fatalf("workspace-detail options = %#v", repository.workspaceOpts)
	}
}

func TestWorkspaceOwnerValidationRunsAfterAuthentication(t *testing.T) {
	invalidOwner := "not-a-uuid"
	service := &workspaceProjectReadsService{repository: &workspaceProjectRepositoryStub{}, sessions: workspaceProjectSessionsStub{}, recentVisits: &workspaceProjectRecentVisitsStub{}}
	_, _, err := service.UserWorkspaces(context.Background(), &api.UserWorkspacesPayload{Owner: &invalidOwner})
	var unauthenticated *api.WorkspaceProjectUnauthenticated
	if !errors.As(err, &unauthenticated) {
		t.Fatalf("anonymous invalid owner error = %T %v", err, err)
	}
	_, _, err = service.UserWorkspaces(authenticatedWorkspaceProjectContext(), &api.UserWorkspacesPayload{Owner: &invalidOwner})
	var badOwner *api.WorkspaceProjectOwnerBadRequest
	if !errors.As(err, &badOwner) || len(badOwner.Owner) != 1 || badOwner.Owner[0] != "Enter a valid UUID." {
		t.Fatalf("authenticated invalid owner error = %T %v", err, err)
	}
}
func TestWorkspaceExplicitEmptyOwnerIsAnUnfilteredResponse(t *testing.T) {
	repository := &workspaceProjectRepositoryStub{userWorkspaces: jsontext.Value(`[]`)}
	service := &workspaceProjectReadsService{
		repository:   repository,
		sessions:     workspaceProjectSessionsStub{},
		recentVisits: &workspaceProjectRecentVisitsStub{},
	}
	emptyOwner := ""
	ctx := WithReadServiceQueryPresence(authenticatedWorkspaceProjectContext(), url.Values{"owner": {""}})
	metadata, result, err := service.UserWorkspaces(ctx, &api.UserWorkspacesPayload{Owner: &emptyOwner})
	if err != nil {
		t.Fatal(err)
	}
	if body := workspaceProjectResponseBody(t, metadata, result); body != "[]" {
		t.Fatalf("body=%q, want unfiltered response", body)
	}
	if repository.workspaceOpts.Owner != nil {
		t.Fatalf("owner=%v, want omitted filter", repository.workspaceOpts.Owner)
	}
}

func TestWorkspaceDetailMapsTenantMissToDjango404(t *testing.T) {
	service := &workspaceProjectReadsService{repository: &workspaceProjectRepositoryStub{err: store.ErrNotFound}, sessions: workspaceProjectSessionsStub{}, recentVisits: &workspaceProjectRecentVisitsStub{}}
	_, _, err := service.Workspace(authenticatedWorkspaceProjectContext(), &api.WorkspacePayload{Slug: "foreign"})
	var notFound *api.WorkspaceProjectDetailNotFound
	if !errors.As(err, &notFound) || notFound.Detail != "No Workspace matches the given query." {
		t.Fatalf("Workspace() error = %T %v", err, err)
	}
}

func TestProjectStatsPreserveFieldsProjectIDsAndIgnoreGeneratedSearch(t *testing.T) {
	projectID := "00000000-0000-4000-8000-000000000003"
	fields, ids, search := "completed_issues,total_members", projectID, "ignored"
	repository := &workspaceProjectRepositoryStub{stats: jsontext.Value(`[{"id":"` + projectID + `","completed_issues":7,"total_members":4}]`)}
	service := &workspaceProjectReadsService{repository: repository, sessions: workspaceProjectSessionsStub{}, recentVisits: &workspaceProjectRecentVisitsStub{}}
	metadata, result, err := service.ProjectStats(authenticatedWorkspaceProjectContext(), &api.ProjectStatsPayload{Slug: "demo", Fields: &fields, ProjectIds: &ids, Search: &search})
	if err != nil || workspaceProjectResponseBody(t, metadata, result) != string(repository.stats) {
		t.Fatalf("ProjectStats() body mismatch: %v", err)
	}
	if len(repository.fields) != 2 || repository.fields[0] != "completed_issues" || repository.fields[1] != "total_members" || repository.projectIDs == nil || *repository.projectIDs != projectID {
		t.Fatalf("stats filters = fields %v ids %v", repository.fields, repository.projectIDs)
	}
}

func TestProjectStatsInvalidProjectIDsMapAfterAuthorization(t *testing.T) {
	invalidIDs := "not-a-uuid"
	service := &workspaceProjectReadsService{repository: &workspaceProjectRepositoryStub{err: store.ErrForbidden}, sessions: workspaceProjectSessionsStub{}, recentVisits: &workspaceProjectRecentVisitsStub{}}
	_, _, err := service.ProjectStats(authenticatedWorkspaceProjectContext(), &api.ProjectStatsPayload{Slug: "foreign", ProjectIds: &invalidIDs})
	var forbidden *api.WorkspaceProjectForbidden
	if !errors.As(err, &forbidden) {
		t.Fatalf("unauthorized invalid project_ids error = %T %v", err, err)
	}
	service.repository = &workspaceProjectRepositoryStub{err: store.ErrBadRequest}
	_, _, err = service.ProjectStats(authenticatedWorkspaceProjectContext(), &api.ProjectStatsPayload{Slug: "demo", ProjectIds: &invalidIDs})
	var validation *api.WorkspaceProjectValidationError
	if !errors.As(err, &validation) || validation.Message != "Please provide valid detail" {
		t.Fatalf("authorized invalid project_ids error = %T %v", err, err)
	}
}

func TestProjectsPreserveCompactRawProjection(t *testing.T) {
	body := `[{"id":"00000000-0000-4000-8000-000000000003","name":"Demo","identifier":"DEMO","sort_order":null,"logo_props":{},"member_role":null,"intake_count":2,"archived_at":null,"workspace":"00000000-0000-4000-8000-000000000001","cycle_view":true,"issue_views_view":true,"module_view":true,"page_view":true,"inbox_view":true,"guest_view_all_features":false,"project_lead":null,"network":2,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","created_by":null,"updated_by":null}]`
	repository := &workspaceProjectRepositoryStub{projects: jsontext.Value(body)}
	service := &workspaceProjectReadsService{repository: repository, sessions: workspaceProjectSessionsStub{}, recentVisits: &workspaceProjectRecentVisitsStub{}}
	search := "ignored"
	metadata, result, err := service.Projects(authenticatedWorkspaceProjectContext(), &api.ProjectsPayload{Slug: "demo", Search: &search})
	if err != nil || workspaceProjectResponseBody(t, metadata, result) != body {
		t.Fatalf("Projects() body mismatch: %v", err)
	}
}

func TestProjectDetailsPreservePlainArrayAndPaginatorEnvelope(t *testing.T) {
	perPage, cursor, orderBy, fields, search := "25", "25:0:0", "name", "name", "ignored"
	tests := []struct {
		name    string
		body    string
		payload *api.ProjectDetailsPayload
	}{
		{
			name:    "plain array",
			body:    `[{"id":"00000000-0000-4000-8000-000000000003","name":"Demo","identifier":"DEMO","workspace":"00000000-0000-4000-8000-000000000001","members":[],"next_work_item_sequence":1}]`,
			payload: &api.ProjectDetailsPayload{Slug: "demo", Fields: &fields, Search: &search},
		},
		{
			name:    "pagination envelope",
			body:    `{"grouped_by":null,"sub_grouped_by":null,"total_count":1,"next_cursor":"25:1:0","prev_cursor":"25:-1:1","next_page_results":false,"prev_page_results":false,"count":1,"total_pages":1,"total_results":1,"extra_stats":null,"results":[]}`,
			payload: &api.ProjectDetailsPayload{Slug: "demo", PerPage: &perPage, Cursor: &cursor, OrderBy: &orderBy, Fields: &fields, Search: &search},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repository := &workspaceProjectRepositoryStub{details: jsontext.Value(testCase.body)}
			service := &workspaceProjectReadsService{repository: repository, sessions: workspaceProjectSessionsStub{}, recentVisits: &workspaceProjectRecentVisitsStub{}}
			metadata, result, err := service.ProjectDetails(authenticatedWorkspaceProjectContext(), testCase.payload)
			if err != nil || workspaceProjectResponseBody(t, metadata, result) != testCase.body {
				t.Fatalf("ProjectDetails() body mismatch: %v", err)
			}
			if repository.detailsOpts.PerPage != testCase.payload.PerPage || repository.detailsOpts.Cursor != testCase.payload.Cursor || repository.detailsOpts.OrderBy != testCase.payload.OrderBy {
				t.Fatalf("details options = %#v", repository.detailsOpts)
			}
		})
	}
}

func TestProjectDetailsMapsBadQueryAndWorkspaceDenial(t *testing.T) {
	service := &workspaceProjectReadsService{repository: &workspaceProjectRepositoryStub{err: store.ErrBadRequest}, sessions: workspaceProjectSessionsStub{}, recentVisits: &workspaceProjectRecentVisitsStub{}}
	_, _, err := service.ProjectDetails(authenticatedWorkspaceProjectContext(), &api.ProjectDetailsPayload{Slug: "demo"})
	var badRequest *api.WorkspaceProjectBadRequest
	if !errors.As(err, &badRequest) || badRequest.Detail != "Invalid cursor parameter." {
		t.Fatalf("bad query error = %T %v", err, err)
	}
	service.repository = &workspaceProjectRepositoryStub{err: store.ErrForbidden}
	_, _, err = service.Projects(authenticatedWorkspaceProjectContext(), &api.ProjectsPayload{Slug: "foreign"})
	var forbidden *api.WorkspaceProjectForbidden
	if !errors.As(err, &forbidden) || forbidden.Message != "You don't have the required permissions." {
		t.Fatalf("forbidden error = %T %v", err, err)
	}
}

func TestMalformedProjectIDMapsToProject404(t *testing.T) {
	service := &workspaceProjectReadsService{repository: &workspaceProjectRepositoryStub{}, sessions: workspaceProjectSessionsStub{}, recentVisits: &workspaceProjectRecentVisitsStub{}}
	_, _, err := service.Project(authenticatedWorkspaceProjectContext(), &api.ProjectPayload{Slug: "demo", Pk: "not-a-uuid"})
	var notFound *api.WorkspaceProjectNotFound
	if !errors.As(err, &notFound) || notFound.Message != "Project does not exist" {
		t.Fatalf("malformed project id error = %T %v", err, err)
	}
	_, _, err = service.Project(context.Background(), &api.ProjectPayload{Slug: "demo", Pk: "not-a-uuid"})
	var unauthenticated *api.WorkspaceProjectUnauthenticated
	if !errors.As(err, &unauthenticated) {
		t.Fatalf("anonymous malformed project id error = %T %v", err, err)
	}
}

func TestProjectDetailProjectionErrorsAndBestEffortRecentVisit(t *testing.T) {
	projectID := "00000000-0000-4000-8000-000000000003"
	project := `{"id":"` + projectID + `","name":"Demo","identifier":"DEMO","workspace":"00000000-0000-4000-8000-000000000001","members":["00000000-0000-4000-8000-000000000010"],"next_work_item_sequence":42}`
	visits := &workspaceProjectRecentVisitsStub{err: errors.New("database unavailable")}
	repository := &workspaceProjectRepositoryStub{project: jsontext.Value(project)}
	service := &workspaceProjectReadsService{repository: repository, sessions: workspaceProjectSessionsStub{}, recentVisits: visits}
	metadata, result, err := service.Project(authenticatedWorkspaceProjectContext(), &api.ProjectPayload{Slug: "demo", Pk: projectID})
	if err != nil || workspaceProjectResponseBody(t, metadata, result) != project {
		t.Fatalf("Project() body mismatch: %v", err)
	}
	if visits.calls != 1 || visits.slug != "demo" || visits.entityName != "project" || visits.entityID.String() != projectID || visits.projectID.String() != projectID {
		t.Fatalf("recent visit = %+v", visits)
	}

	for _, testCase := range []struct {
		err     error
		assert  func(error) bool
		message string
	}{
		{err: store.ErrNotFound, assert: func(err error) bool {
			var value *api.WorkspaceProjectNotFound
			return errors.As(err, &value) && value.Message == "Project does not exist"
		}, message: "not found"},
		{err: store.ErrProjectForbidden, assert: func(err error) bool {
			var value *api.WorkspaceProjectForbidden
			return errors.As(err, &value) && value.Message == "You do not have permission"
		}, message: "secret forbidden"},
		{err: store.ErrProjectConflict, assert: func(err error) bool {
			var value *api.WorkspaceProjectConflict
			return errors.As(err, &value) && value.Message == "You are not a member of this project"
		}, message: "public conflict"},
	} {
		repository.err = testCase.err
		_, _, err := service.Project(authenticatedWorkspaceProjectContext(), &api.ProjectPayload{Slug: "demo", Pk: projectID})
		if !testCase.assert(err) {
			t.Errorf("%s error = %T %v", testCase.message, err, err)
		}
	}
	if visits.calls != 1 {
		t.Fatalf("failed project reads recorded %d visits, want no additional calls", visits.calls)
	}
}
