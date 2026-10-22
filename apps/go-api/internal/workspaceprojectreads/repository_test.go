package workspaceprojectreads

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/jackc/pgx/v5"
	"uuid"
)

type workspaceProjectRow func(...any) error

func (row workspaceProjectRow) Scan(destinations ...any) error { return row(destinations...) }

type workspaceProjectDatabase struct {
	queries []string
	args    [][]any
	rows    []pgx.Row
}

func (database *workspaceProjectDatabase) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	database.queries = append(database.queries, query)
	database.args = append(database.args, args)
	if len(database.rows) == 0 {
		return workspaceProjectRow(func(...any) error { return errors.New("unexpected query") })
	}
	row := database.rows[0]
	database.rows = database.rows[1:]
	return row
}

func workspaceProjectJSONRow(value string) pgx.Row {
	return workspaceProjectRow(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(value)
		return nil
	})
}

func workspaceProjectRoleRow(role int) pgx.Row {
	return workspaceProjectRow(func(destinations ...any) error {
		*destinations[0].(*int) = role
		return nil
	})
}

func workspaceProjectDetailRow(value string, network int, member bool) pgx.Row {
	return workspaceProjectRow(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(value)
		*destinations[1].(*int) = network
		*destinations[2].(*bool) = member
		return nil
	})
}

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) returned nil error")
	}
}

func TestAPIProjectLiteIncludesArchivedOnlyWhenRequested(t *testing.T) {
	user := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	include := true
	database := &workspaceProjectDatabase{rows: []pgx.Row{
		workspaceProjectRoleRow(15),
		workspaceProjectRow(func(destinations ...any) error {
			*destinations[0].(*[]byte) = []byte(`[]`)
			*destinations[1].(*int) = 0
			return nil
		}),
	}}
	repository, _ := NewRepository(database)
	includeText := "1"
	if _, err := repository.APIProjects(context.Background(), user, "demo", APIProjectListOptions{IncludeArchived: include, PerPage: &includeText}); err != nil {
		t.Fatal(err)
	}
	if database.args[1][2] != true || database.args[1][3] != 0 || database.args[1][4] != 1 {
		t.Fatalf("include_archived/page arguments not preserved: %#v", database.args[1])
	}
	if !strings.Contains(database.queries[1], "project.archived_at") || !strings.Contains(database.queries[1], "cover_image_url") {
		t.Fatalf("lite projection/query incomplete: %s", database.queries[1])
	}
}

func TestAPIProjectsRejectsInvalidPaginationBeforeProjectQuery(t *testing.T) {
	user := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	bad := "not-a-number"
	database := &workspaceProjectDatabase{rows: []pgx.Row{workspaceProjectRoleRow(20)}}
	repository, _ := NewRepository(database)
	_, err := repository.APIProjects(context.Background(), user, "demo", APIProjectListOptions{PerPage: &bad})
	var response *djangoapi.HTTPError
	if !errors.As(err, &response) || response.Status != 400 || response.Body["detail"] != "Invalid per_page parameter." {
		t.Fatalf("invalid pagination error = %#v", err)
	}
	if len(database.queries) != 1 {
		t.Fatalf("project query ran after invalid pagination: %#v", database.queries)
	}
}

func TestAPIProjectSummaryAllowsMemberButRejectsGuest(t *testing.T) {
	user := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	project := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	database := &workspaceProjectDatabase{rows: []pgx.Row{
		workspaceProjectRoleRow(15),
		workspaceProjectJSONRow(`{"counts":{"issues":1},"id":"p","identifier":"ABC","name":"Core"}`),
	}}
	repository, _ := NewRepository(database)
	body, err := repository.APIProjectSummary(context.Background(), user, "demo", project, APIProjectSummaryOptions{Fields: []string{"issues", "unknown"}})
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"counts":{"issues":1},"id":"p","identifier":"ABC","name":"Core"}` {
		t.Fatalf("summary body = %s", body)
	}
	if !strings.Contains(database.queries[1], "state.\"group\" IS DISTINCT FROM 'triage'") ||
		!strings.Contains(database.queries[1], "issue.deleted_at IS NULL") {
		t.Fatalf("summary count projection changed: %s", database.queries[1])
	}
	guestDB := &workspaceProjectDatabase{rows: []pgx.Row{workspaceProjectRoleRow(5)}}
	guestRepository, _ := NewRepository(guestDB)
	if _, err := guestRepository.APIProjectSummary(context.Background(), user, "demo", project, APIProjectSummaryOptions{}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("guest summary error = %v", err)
	}
	if len(guestDB.queries) != 1 {
		t.Fatalf("guest summary queried target: %#v", guestDB.queries)
	}
}

func TestAPIProjectWorkspaceLookupDistinguishesMissingWorkspaceFromNonmember(t *testing.T) {
	user := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	missingDB := &workspaceProjectDatabase{rows: []pgx.Row{workspaceProjectRow(func(...any) error { return pgx.ErrNoRows })}}
	missingRepo, _ := NewRepository(missingDB)
	if _, err := missingRepo.APIProjects(context.Background(), user, "missing", APIProjectListOptions{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing workspace error = %v, want ErrNotFound", err)
	}
	nonmemberDB := &workspaceProjectDatabase{rows: []pgx.Row{workspaceProjectRoleRow(0)}}
	nonmemberRepo, _ := NewRepository(nonmemberDB)
	if _, err := nonmemberRepo.APIProjects(context.Background(), user, "private", APIProjectListOptions{}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("nonmember error = %v, want ErrForbidden", err)
	}
}

func TestWorkspaceListsBatchAuthorizedProjectionAndApplyFiltersTogether(t *testing.T) {
	owner := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	searchTerms := []string{"Core", "Workspace"}
	user := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	database := &workspaceProjectDatabase{rows: []pgx.Row{workspaceProjectJSONRow(`[]`), workspaceProjectJSONRow(`[]`)}}
	repository, _ := NewRepository(database)
	options := WorkspaceOptions{Owner: &owner, SearchTerms: searchTerms}
	if body, err := repository.UserWorkspaces(context.Background(), user, options); err != nil || string(body) != `[]` {
		t.Fatalf("UserWorkspaces() = %s, %v", body, err)
	}
	if body, err := repository.Workspaces(context.Background(), user, options); err != nil || string(body) != `[]` {
		t.Fatalf("Workspaces() = %s, %v", body, err)
	}
	for index, query := range database.queries {
		for _, want := range []string{
			"membership.member_id = $1", "membership.is_active = true", "membership.deleted_at IS NULL",
			"workspace.deleted_at IS NULL", "workspace.owner_id = $2", "unnest($3::text[])", "counted_user.is_bot=false",
			"logo_asset", "logo_url", "/api/assets/v2/static/", "total_members", "background_color", "set_config('TimeZone'", "to_char(workspace.created_at",
		} {
			if !strings.Contains(query, want) {
				t.Errorf("workspace query %d missing %q", index, want)
			}
		}
		terms, termsOK := database.args[index][2].([]string)
		if len(database.args[index]) != 3 || database.args[index][0] != user || database.args[index][1] != owner || !termsOK || strings.Join(terms, ",") != strings.Join(searchTerms, ",") {
			t.Errorf("workspace args %d = %#v", index, database.args[index])
		}
	}
	if !strings.Contains(database.queries[0], "'role'") || !strings.Contains(database.queries[0], "ORDER BY created_at DESC") {
		t.Errorf("users/me workspace projection or order is wrong: %s", database.queries[0])
	}
	if strings.Contains(database.queries[1], "'role'") || !strings.Contains(database.queries[1], "ORDER BY name ASC") {
		t.Errorf("fixed workspace collection projection or order is wrong: %s", database.queries[1])
	}
}

func TestWorkspaceDetailHidesForeignDeletedAndFilteredRows(t *testing.T) {
	database := &workspaceProjectDatabase{rows: []pgx.Row{workspaceProjectRow(func(...any) error { return pgx.ErrNoRows })}}
	repository, _ := NewRepository(database)
	_, err := repository.Workspace(context.Background(), uuid.New(), "foreign", WorkspaceOptions{})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Workspace() error = %v", err)
	}
	query := database.queries[0]
	for _, want := range []string{"membership.member_id = $1", "workspace.slug = $2", "workspace.deleted_at IS NULL", "workspace.owner_id = $3", "unnest($4::text[])", "set_config('TimeZone'", "to_char(workspace.updated_at"} {
		if !strings.Contains(query, want) {
			t.Errorf("workspace detail query missing %q", want)
		}
	}
	if strings.Contains(query, "'role'") {
		t.Error("workspace detail must omit the unannotated role field")
	}
}

func TestWorkspaceRoleRequiresActiveAllowedMembershipInLiveTenant(t *testing.T) {
	database := &workspaceProjectDatabase{rows: []pgx.Row{workspaceProjectRow(func(...any) error { return pgx.ErrNoRows })}}
	repository, _ := NewRepository(database)
	_, err := repository.Projects(context.Background(), uuid.New(), "demo")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Projects() error = %v", err)
	}
	query := database.queries[0]
	for _, want := range []string{"membership.role IN (5,15,20)", "membership.is_active=true", "membership.deleted_at IS NULL", "workspace.deleted_at IS NULL"} {
		if !strings.Contains(query, want) {
			t.Errorf("authorization query missing %q", want)
		}
	}
	if len(database.queries) != 1 {
		t.Fatalf("forbidden request performed %d queries, want authorization only", len(database.queries))
	}
}

func TestProjectStatsUsesRequestedProjectionAndBatchedCounts(t *testing.T) {
	projectID := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	database := &workspaceProjectDatabase{rows: []pgx.Row{workspaceProjectRoleRow(20), workspaceProjectJSONRow(`[]`)}}
	repository, _ := NewRepository(database)
	projectIDs := projectID.String()
	body, err := repository.ProjectStats(context.Background(), uuid.New(), "demo", []string{"completed_issues", "bogus"}, &projectIDs)
	if err != nil || string(body) != `[]` {
		t.Fatalf("ProjectStats() = %s, %v", body, err)
	}
	query := database.queries[1]
	for _, want := range []string{"'id', project.id", "'completed_issues'", "issue_state.\"group\" IN ('completed','cancelled')", "project.id = ANY($3::uuid[])", "project.deleted_at IS NULL", "workspace.deleted_at IS NULL", "ORDER BY project.created_at DESC"} {
		if !strings.Contains(query, want) {
			t.Errorf("stats query missing %q", want)
		}
	}
	for _, absent := range []string{"'total_issues'", "'total_members'", "'total_cycles'", "'total_modules'"} {
		if strings.Contains(query, absent) {
			t.Errorf("selected stats query unexpectedly contains %q", absent)
		}
	}
	if len(database.args[1]) != 3 || database.args[1][1] != true {
		t.Fatalf("stats args = %#v", database.args[1])
	}
}

func TestProjectStatsValidatesProjectIDsAfterAuthorization(t *testing.T) {
	invalid := "not-a-uuid"
	for _, testCase := range []struct {
		name    string
		row     pgx.Row
		wantErr error
	}{
		{name: "forbidden first", row: workspaceProjectRow(func(...any) error { return pgx.ErrNoRows }), wantErr: ErrForbidden},
		{name: "authorized validation", row: workspaceProjectRoleRow(20), wantErr: ErrBadRequest},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			database := &workspaceProjectDatabase{rows: []pgx.Row{testCase.row}}
			repository, _ := NewRepository(database)
			_, err := repository.ProjectStats(context.Background(), uuid.New(), "demo", nil, &invalid)
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("ProjectStats() error = %v, want %v", err, testCase.wantErr)
			}
			if len(database.queries) != 1 {
				t.Fatalf("invalid project_ids query count = %d, want authorization only", len(database.queries))
			}
		})
	}
}

func TestProjectStatsInvalidOrEmptyFieldsSelectAllFiveCounters(t *testing.T) {
	selected := selectedStatFields([]string{"unknown", ""})
	want := []string{"total_issues", "completed_issues", "total_members", "total_cycles", "total_modules"}
	if strings.Join(selected, ",") != strings.Join(want, ",") {
		t.Fatalf("selectedStatFields() = %v, want %v", selected, want)
	}
	for _, field := range want {
		if expression := projectStatExpression(field); !strings.Contains(expression, "SELECT count(*)") {
			t.Errorf("%s expression = %s", field, expression)
		}
	}
}

func TestProjectIssueStatsMirrorIssueManagerVisibility(t *testing.T) {
	for _, field := range []string{"total_issues", "completed_issues"} {
		expression := projectStatExpression(field)
		for _, predicate := range []string{
			"LEFT JOIN states issue_state ON issue_state.id=issue.state_id",
			"issue.deleted_at IS NULL",
			"issue.archived_at IS NULL",
			"issue.is_draft=false",
			"project.archived_at IS NULL",
			`(issue_state."group" IS NULL OR issue_state."group" <> 'triage')`,
		} {
			if !strings.Contains(expression, predicate) {
				t.Errorf("%s expression missing IssueManager predicate %q: %s", field, predicate, expression)
			}
		}
	}
	completed := projectStatExpression("completed_issues")
	if !strings.Contains(completed, `issue_state."group" IN ('completed','cancelled')`) {
		t.Errorf("completed_issues expression missing completed-state filter: %s", completed)
	}
}

func TestCompactProjectsPreserveRoleVisibilityProjectionAndArchivedRows(t *testing.T) {
	database := &workspaceProjectDatabase{rows: []pgx.Row{workspaceProjectRoleRow(15), workspaceProjectJSONRow(`[]`)}}
	repository, _ := NewRepository(database)
	if _, err := repository.Projects(context.Background(), uuid.New(), "demo"); err != nil {
		t.Fatal(err)
	}
	query := database.queries[1]
	for _, want := range []string{
		"$3=20", "$3=15", "$3=5", "project.network=2", "visible_member.member_id=$1", "visible_guest.member_id=$1",
		"'member_role'", "'intake_count'", "intake.status=-2", "'inbox_view', project.intake_view", "'sort_order'",
		"project.deleted_at IS NULL", "ORDER BY created_at ASC", "set_config('TimeZone'", "to_char(project.created_at",
	} {
		if !strings.Contains(query, want) {
			t.Errorf("compact project query missing %q", want)
		}
	}
	if strings.Contains(query, "AND project.archived_at IS NULL") {
		t.Error("compact list must include archived projects")
	}
	if len(database.queries) != 2 {
		t.Fatalf("compact list query count = %d, want authorization plus one batched projection", len(database.queries))
	}
}

func TestProjectDetailsPlainAndPaginatedVariantsUseBothParameterGate(t *testing.T) {
	tests := []struct {
		name      string
		options   ProjectDetailsOptions
		wantPage  bool
		wantOrder string
	}{
		{name: "plain without parameters"},
		{name: "plain with only per_page", options: ProjectDetailsOptions{PerPage: stringPointer("25")}},
		{name: "plain with only cursor", options: ProjectDetailsOptions{Cursor: stringPointer("25:0:0")}},
		{name: "paginated", options: ProjectDetailsOptions{PerPage: stringPointer("25"), Cursor: stringPointer("25:2:0"), OrderBy: stringPointer("name")}, wantPage: true, wantOrder: "name ASC NULLS LAST"},
		{name: "invalid order defaults", options: ProjectDetailsOptions{PerPage: stringPointer("25"), Cursor: stringPointer("25:0:0"), OrderBy: stringPointer("DROP TABLE projects")}, wantPage: true, wantOrder: "created_at DESC NULLS LAST"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			database := &workspaceProjectDatabase{rows: []pgx.Row{workspaceProjectRoleRow(20), workspaceProjectJSONRow(`[]`)}}
			repository, _ := NewRepository(database)
			if _, err := repository.ProjectDetails(context.Background(), uuid.New(), "demo", testCase.options); err != nil {
				t.Fatal(err)
			}
			query := database.queries[1]
			isPage := strings.Contains(query, "'next_cursor'")
			if isPage != testCase.wantPage {
				t.Fatalf("pagination variant = %v, want %v; query=%s", isPage, testCase.wantPage, query)
			}
			if testCase.wantPage {
				for _, want := range []string{testCase.wantOrder, "LIMIT $4 OFFSET $5", "'total_count'", "'total_pages'", "'results'"} {
					if !strings.Contains(query, want) {
						t.Errorf("paginated query missing %q", want)
					}
				}
				if len(database.args[1]) != 6 || database.args[1][3] != 25 {
					t.Errorf("paginated args = %#v", database.args[1])
				} else if testCase.name == "paginated" && (database.args[1][4] != 50 || database.args[1][5] != 2) {
					t.Errorf("second-page args = %#v", database.args[1])
				} else if testCase.name == "invalid order defaults" && (database.args[1][4] != 0 || database.args[1][5] != 0) {
					t.Errorf("first-page args = %#v", database.args[1])
				}
			} else if !strings.Contains(query, "ORDER BY sort_order ASC NULLS LAST, name ASC") {
				t.Error("plain details must preserve sort_order/name ordering")
			}
		})
	}
}

func TestProjectDetailsRejectMalformedPagination(t *testing.T) {
	for _, options := range []ProjectDetailsOptions{
		{PerPage: stringPointer("not-a-number"), Cursor: stringPointer("25:0:0")},
		{PerPage: stringPointer("25"), Cursor: stringPointer("bad")},
		{PerPage: stringPointer("0"), Cursor: stringPointer("0:0:0")},
		{PerPage: stringPointer("-1"), Cursor: stringPointer("-1:0:0")},
	} {
		database := &workspaceProjectDatabase{rows: []pgx.Row{workspaceProjectRoleRow(20)}}
		repository, _ := NewRepository(database)
		_, err := repository.ProjectDetails(context.Background(), uuid.New(), "demo", options)
		if !errors.Is(err, ErrBadRequest) {
			t.Errorf("ProjectDetails(%#v) error = %v", options, err)
		}
		if len(database.queries) != 1 {
			t.Errorf("malformed pagination performed %d queries", len(database.queries))
		}
	}
}

func TestDjangoTimestampUsesUserOffsetAndCanonicalUTC(t *testing.T) {
	expression := djangoTimestamp("project.created_at")
	for _, want := range []string{"to_char(project.created_at", "'US'", "= '000000' THEN ''", "'TZH:TZM'", "= '+00:00' THEN 'Z'"} {
		if !strings.Contains(expression, want) {
			t.Errorf("timestamp expression missing %q", want)
		}
	}
}

func TestDetailedProjectionBatchesMembersSequenceFavoritesAndAssets(t *testing.T) {
	database := &workspaceProjectDatabase{rows: []pgx.Row{workspaceProjectRoleRow(5), workspaceProjectJSONRow(`[]`)}}
	repository, _ := NewRepository(database)
	if _, err := repository.ProjectDetails(context.Background(), uuid.New(), "demo", ProjectDetailsOptions{}); err != nil {
		t.Fatal(err)
	}
	query := database.queries[1]
	for _, want := range []string{
		"json_agg(membership.member_id ORDER BY membership.created_at DESC)", "member_user.is_bot=false",
		"max(issue_sequence.sequence)+1", "user_favorites", "deploy_boards", "project_user_properties",
		"cover_image_url", "/api/assets/v2/static/", "'next_work_item_sequence'", "'members'",
		"set_config('TimeZone'", "to_char(project.created_at", "to_char(project.updated_at", "to_char(project.archived_at",
	} {
		if !strings.Contains(query, want) {
			t.Errorf("detailed projection query missing %q", want)
		}
	}
	if len(database.queries) != 2 {
		t.Fatalf("detailed list query count = %d, want authorization plus one projection query", len(database.queries))
	}
}

func TestProjectDetailScopesTenantArchiveAndMapsMembershipStates(t *testing.T) {
	tests := []struct {
		name    string
		row     pgx.Row
		wantErr error
	}{
		{name: "member", row: workspaceProjectDetailRow(`{"id":"ok"}`, 0, true)},
		{name: "secret nonmember", row: workspaceProjectDetailRow(`{"id":"secret"}`, 0, false), wantErr: ErrProjectForbidden},
		{name: "public nonmember", row: workspaceProjectDetailRow(`{"id":"public"}`, 2, false), wantErr: ErrProjectConflict},
		{name: "missing", row: workspaceProjectRow(func(...any) error { return pgx.ErrNoRows }), wantErr: ErrNotFound},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			database := &workspaceProjectDatabase{rows: []pgx.Row{workspaceProjectRoleRow(20), testCase.row}}
			repository, _ := NewRepository(database)
			_, err := repository.Project(context.Background(), uuid.New(), "demo", uuid.New())
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("Project() error = %v, want %v", err, testCase.wantErr)
			}
			query := database.queries[1]
			for _, want := range []string{"workspace.slug=$2", "project.id=$3", "workspace.deleted_at IS NULL", "project.deleted_at IS NULL", "project.archived_at IS NULL", "membership.is_active=true", "membership.deleted_at IS NULL"} {
				if !strings.Contains(query, want) {
					t.Errorf("project detail query missing %q", want)
				}
			}
		})
	}
}

func stringPointer(value string) *string { return &value }
