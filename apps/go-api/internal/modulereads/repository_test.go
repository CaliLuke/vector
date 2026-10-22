package modulereads

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"uuid"
)

type moduleRowFunc func(...any) error

func (row moduleRowFunc) Scan(destinations ...any) error { return row(destinations...) }

type moduleDatabaseStub struct {
	queries []string
	args    [][]any
	rows    []pgx.Row
}

func (database *moduleDatabaseStub) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	database.queries = append(database.queries, query)
	database.args = append(database.args, args)
	if len(database.rows) == 0 {
		return moduleRowFunc(func(...any) error { return errors.New("unexpected query") })
	}
	row := database.rows[0]
	database.rows = database.rows[1:]
	return row
}

func moduleBoolRow(value bool) pgx.Row {
	return moduleRowFunc(func(destinations ...any) error {
		*destinations[0].(*bool) = value
		return nil
	})
}

func moduleJSONRow(value string) pgx.Row {
	return moduleRowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(value)
		return nil
	})
}

func modulePageRow(value string, total int) pgx.Row {
	return moduleRowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(value)
		*destinations[1].(*int) = total
		*destinations[2].(*[]byte) = []byte(`[]`)
		*destinations[3].(*[]byte) = []byte(`[]`)
		*destinations[4].(*[]byte) = []byte(`{}`)
		*destinations[5].(*[]byte) = []byte(`{}`)
		*destinations[6].(*bool) = false
		*destinations[7].(*int) = len(json.RawMessage(value))
		return nil
	})
}

func moduleGroupedPageRow(value string, total int, groups, subgroups string) pgx.Row {
	return moduleRowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(value)
		*destinations[1].(*int) = total
		*destinations[2].(*[]byte) = []byte(groups)
		*destinations[3].(*[]byte) = []byte(subgroups)
		*destinations[4].(*[]byte) = []byte(`{}`)
		*destinations[5].(*[]byte) = []byte(`{}`)
		*destinations[6].(*bool) = false
		*destinations[7].(*int) = 0
		return nil
	})
}

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) returned nil error")
	}
}

func TestWorkspaceListRequiresWorkspaceAndProjectMembership(t *testing.T) {
	database := &moduleDatabaseStub{rows: []pgx.Row{moduleBoolRow(true), moduleJSONRow(`[]`)}}
	repository, _ := NewRepository(database)
	userID := uuid.New()
	data, err := repository.WorkspaceList(context.Background(), userID, "demo")
	if err != nil || string(data) != `[]` {
		t.Fatalf("data=%s err=%v", data, err)
	}
	for _, fragment := range []string{
		"workspace_member.member_id = $1", "workspace.slug = $2",
		"workspace_member.is_active = true", "workspace_member.deleted_at IS NULL",
	} {
		if !strings.Contains(database.queries[0], fragment) {
			t.Errorf("workspace authorization missing %q", fragment)
		}
	}
	for _, fragment := range []string{
		"project_member.member_id = $1", "project_member.is_active = true",
		"project_member.deleted_at IS NULL", "project.archived_at IS NULL",
		"module.archived_at IS NULL", "module.deleted_at IS NULL",
	} {
		if !strings.Contains(database.queries[1], fragment) {
			t.Errorf("workspace module query missing %q", fragment)
		}
	}
	for _, omitted := range []string{"'is_favorite'", "'total_estimate_points'", "'completed_estimate_points'"} {
		if strings.Contains(database.queries[1], omitted) {
			t.Errorf("workspace module query exposes source-omitted field %q", omitted)
		}
	}
	if strings.Contains(database.queries[1], `manager_state."group" <> 'triage'`) {
		t.Error("workspace module counts must include triage issues")
	}
}

func TestProjectAuthorizationDistinguishesGuestAndElevatedReads(t *testing.T) {
	database := &moduleDatabaseStub{rows: []pgx.Row{
		moduleBoolRow(true), moduleJSONRow(`[]`),
		moduleBoolRow(true), modulePageRow(`[]`, 0),
	}}
	repository, _ := NewRepository(database)
	userID, projectID, moduleID := uuid.New(), uuid.New(), uuid.New()
	if _, err := repository.Links(context.Background(), userID, "demo", projectID, moduleID); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(database.queries[0], "project_member.role IN (15, 20)") {
		t.Fatal("safe ProjectEntityPermission read rejected a guest")
	}
	if strings.Contains(database.queries[1], "AND module.archived_at IS NULL") {
		t.Fatal("archived modules must retain access to their links")
	}
	if _, err := repository.Issues(context.Background(), userID, "demo", projectID, moduleID, IssueListOptions{}); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"project_member.role IN (15, 20)", "workspace_admin.role = 20",
		"project_member.is_active = true", "project_member.deleted_at IS NULL",
		"workspace.deleted_at IS NULL", "project.deleted_at IS NULL",
	} {
		if !strings.Contains(database.queries[2], fragment) {
			t.Errorf("elevated authorization missing %q", fragment)
		}
	}
	if strings.Contains(database.queries[3], "AND module.archived_at IS NULL") {
		t.Fatal("archived modules must retain access to their issues")
	}
}

func TestActiveAndArchivedListsUseOppositeScopesAndAggregates(t *testing.T) {
	database := &moduleDatabaseStub{rows: []pgx.Row{
		moduleBoolRow(true), moduleJSONRow(`[]`),
		moduleBoolRow(true), moduleJSONRow(`[]`),
	}}
	repository, _ := NewRepository(database)
	userID, projectID := uuid.New(), uuid.New()
	if _, err := repository.List(context.Background(), userID, "demo", projectID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.ArchivedList(context.Background(), userID, "demo", projectID); err != nil {
		t.Fatal(err)
	}
	activeQuery, archivedQuery := database.queries[1], database.queries[3]
	if !strings.Contains(activeQuery, "module.archived_at IS NULL") || strings.Contains(activeQuery, "module.archived_at IS NOT NULL") {
		t.Fatalf("active query has wrong archive scope: %s", activeQuery)
	}
	if !strings.Contains(archivedQuery, "module.archived_at IS NOT NULL") {
		t.Fatalf("archived query has wrong archive scope: %s", archivedQuery)
	}
	for _, fragment := range []string{"total_issues", "completed_issues", "member_ids", "user_favorites", "ORDER BY module_row.favorite DESC"} {
		if !strings.Contains(activeQuery, fragment) || !strings.Contains(archivedQuery, fragment) {
			t.Errorf("module list query missing aggregate/order %q", fragment)
		}
	}
	if strings.Contains(activeQuery, "'archived_at'") {
		t.Error("active project module list exposes source-omitted archived_at")
	}
	for _, omitted := range []string{"'logo_props'", "'total_estimate_points'", "'completed_estimate_points'"} {
		if strings.Contains(archivedQuery, omitted) {
			t.Errorf("archived module query exposes source-omitted field %q", omitted)
		}
	}
}

func TestArchiveDetailAcceptsModuleIDAndUsesArchivedDetailProjection(t *testing.T) {
	database := &moduleDatabaseStub{rows: []pgx.Row{moduleBoolRow(true), moduleJSONRow(`{"id":"ok"}`)}}
	repository, _ := NewRepository(database)
	userID, projectID, moduleID := uuid.New(), uuid.New(), uuid.New()
	if _, err := repository.ArchiveDetail(context.Background(), userID, "demo", projectID, moduleID); err != nil {
		t.Fatal(err)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"module.id = $4", "module.archived_at IS NOT NULL", "'link_module'",
		"'distribution'", "'estimate_distribution'", "'sub_issues'",
		"LEFT JOIN issue_assignees distribution_assignee",
		"LEFT JOIN issue_labels distribution_issue_label",
		"LEFT JOIN estimate_points distribution_point",
		`manager_state."group" <> 'triage'`,
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("archive detail query missing %q", fragment)
		}
	}
	if args := database.args[1]; len(args) != 4 || args[3] != moduleID {
		t.Fatalf("archive detail args=%#v, want module_id as fourth argument", args)
	}
}
func TestArchivedDetailBurndownKeepsZeroSeriesForValidDates(t *testing.T) {
	active := moduleBurndown("module", false, false)
	archived := moduleBurndown("module", false, true)
	if !strings.Contains(active, "= 0 THEN '{}'::jsonb") {
		t.Fatalf("active detail lost zero-total suppression: %s", active)
	}
	if strings.Contains(archived, "= 0 THEN '{}'::jsonb") || !strings.Contains(archived, "generate_series(module.start_date, module.target_date") {
		t.Fatalf("archived detail must emit dated zero series: %s", archived)
	}
	if detail := moduleDetailJSON("module", true); strings.Contains(detail, "= 0 THEN '{}'::jsonb") {
		t.Fatalf("archived detail wired active burndown mode: %s", detail)
	}
}

func TestIssueListAppliesScopedFiltersOrderingAndPagination(t *testing.T) {
	database := &moduleDatabaseStub{rows: []pgx.Row{moduleBoolRow(true), modulePageRow(`[]`, 17)}}
	repository, _ := NewRepository(database)
	userID, projectID, moduleID := uuid.New(), uuid.New(), uuid.New()
	page, err := repository.Issues(context.Background(), userID, "demo", projectID, moduleID, IssueListOptions{
		Limit: 25, Page: 2, OrderBy: "-priority", State: uuid.New().String() + "," + uuid.New().String(), Priority: "urgent,high",
		Assignees: uuid.New().String(), Labels: uuid.New().String(), CreatedBy: uuid.New().String(), StartDate: "2026-01-01",
		TargetDate: "2026-02-01", Type: "active",
	})
	if err != nil || page.Total != 17 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"module.id = $4", "module_issue.deleted_at IS NULL", "issue.deleted_at IS NULL",
		"issue.archived_at IS NULL", "issue.is_draft = false", "string_to_array",
		"issue_assignees filter_assignee", "issue_labels filter_label",
		"CASE issue.priority WHEN 'urgent' THEN 0", "ELSE 5 END ASC",
		"LIMIT $12 OFFSET $13", "'assignee_ids'", "'label_ids'", "'module_ids'",
		"(SELECT count(*) FROM filtered)", `base_state."group" <> 'triage'`,
		`child_state."group" <> 'triage'`,
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("issue list query missing %q", fragment)
		}
	}
	args := database.args[1]
	if args[len(args)-2] != 25 || args[len(args)-1] != 50 {
		t.Fatalf("pagination args=%#v", args)
	}
}

func TestGroupedIssueListFetchesTheFullSetAndReturnsFieldDomains(t *testing.T) {
	database := &moduleDatabaseStub{rows: []pgx.Row{
		moduleBoolRow(true),
		moduleGroupedPageRow(`[]`, 4, `["low","high","urgent"]`, `["None","label-a"]`),
	}}
	repository, _ := NewRepository(database)
	page, err := repository.Issues(
		context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(),
		IssueListOptions{Limit: 2, Page: 1, CursorValue: 1.5, CursorSet: true, GroupBy: "priority", SubGroupBy: "labels__id"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.GroupValues) != 3 || len(page.SubGroupValues) != 2 {
		t.Fatalf("group values=%#v subgroup values=%#v", page.GroupValues, page.SubGroupValues)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		`'["low","medium","high","urgent","none"]'::json`,
		"SELECT label.id::text AS value", "UNION ALL SELECT 'None'",
		"row_number() OVER", "PARTITION BY issue.group_key, issue.subgroup_key",
		"issue.group_position::double precision <", "_group_page_count", "_group_has_next", "(SELECT count(*) FROM expanded)",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("grouped issue query missing %q", fragment)
		}
	}
	if casts := strings.Count(query, "::double precision"); casts < 7 {
		t.Fatalf("grouped fractional cursor bounds lack explicit float comparison casts: %d\n%s", casts, query)
	}
	args := database.args[1]
	if args[len(args)-2] != float64(4) || args[len(args)-1] != float64(1.5) {
		t.Fatalf("fractional grouped cursor pagination args=%#v", args)
	}
}

func TestIssueListHandlesNullFiltersAndIndependentModuleFilter(t *testing.T) {
	database := &moduleDatabaseStub{rows: []pgx.Row{moduleBoolRow(true), modulePageRow(`[]`, 0)}}
	repository, _ := NewRepository(database)
	_, err := repository.Issues(
		context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(),
		IssueListOptions{CreatedBy: "None", Cycle: "None", Module: uuid.New().String()},
	)
	if err != nil {
		t.Fatal(err)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"issue.created_by_id IS NULL",
		"NOT EXISTS (SELECT 1 FROM cycle_issues filter_cycle",
		"EXISTS (SELECT 1 FROM module_issues filter_module_relation",
		"filter_module_relation.module_id::text",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("filtered issue query missing %q", fragment)
		}
	}
}

func TestIssueFiltersMatchDjangoSanitizingAndDateSemantics(t *testing.T) {
	where, args := addIssueDateFilters(nil, nil, "2026-08,2_months;after;fromnow", "issue.start_date")
	joined := strings.Join(where, " ")
	if !strings.Contains(joined, "::text LIKE $1 ESCAPE") || !strings.Contains(joined, "CURRENT_DATE + ($2 * INTERVAL '1 day')") {
		t.Fatalf("date predicates=%q", joined)
	}
	if len(args) != 2 || args[0] != "%2026-08%" || args[1] != 60 {
		t.Fatalf("date args=%#v", args)
	}

	database := &moduleDatabaseStub{rows: []pgx.Row{moduleBoolRow(true), modulePageRow(`[]`, 0)}}
	repository, _ := NewRepository(database)
	_, err := repository.Issues(
		context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(),
		IssueListOptions{
			State: "not-a-uuid", Assignees: "bad", Labels: "bad", CreatedBy: "bad",
			Parent: "bad", Mentions: "bad", Cycle: "bad", Module: "bad", Subscriber: "bad",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	query := database.queries[1]
	for _, omitted := range []string{
		"filter_assignee", "filter_label", "filter_cycle", "filter_module_relation",
		"filter_mention", "filter_subscriber", "issue.created_by_id::text", "issue.parent_id::text",
	} {
		if strings.Contains(query, omitted) {
			t.Errorf("invalid UUID token retained filter %q", omitted)
		}
	}
}

func TestIssueNameEscapesLikeWildcardsAndInboxOverridesIntake(t *testing.T) {
	database := &moduleDatabaseStub{rows: []pgx.Row{moduleBoolRow(true), modulePageRow(`[]`, 0)}}
	repository, _ := NewRepository(database)
	_, err := repository.Issues(
		context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(),
		IssueListOptions{Name: `%_\`, IntakeStatus: " 1 ", InboxStatus: "\u20032\u2003"},
	)
	if err != nil {
		t.Fatal(err)
	}
	query := database.queries[1]
	if !strings.Contains(query, `issue.name ILIKE`) || !strings.Contains(query, `ESCAPE '\'`) {
		t.Fatalf("name predicate=%s", query)
	}
	if strings.Count(query, "filter_intake.status::text") != 1 {
		t.Fatalf("intake status was not replaced by inbox status: %s", query)
	}
	args := database.args[1]
	if args[4] != `%\%\_\\%` || args[5] != "2" {
		t.Fatalf("filter args=%#v", args)
	}
}

func TestComplexIssueFiltersSupportLogicalTreesAndRelationScopes(t *testing.T) {
	assigneeID := uuid.New().String()
	database := &moduleDatabaseStub{rows: []pgx.Row{moduleBoolRow(true), modulePageRow(`[]`, 0)}}
	repository, _ := NewRepository(database)
	_, err := repository.Issues(
		context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(),
		IssueListOptions{Filters: `{"or":[{"priority__in":["high","urgent"]},{"assignee_id":"` + assigneeID + `"}]}`},
	)
	if err != nil {
		t.Fatal(err)
	}
	query := database.queries[1]
	for _, fragment := range []string{" OR ", "issue.priority = ANY", "issue_assignees complex_assignee", "complex_assignee.deleted_at IS NULL"} {
		if !strings.Contains(query, fragment) {
			t.Errorf("complex filter query missing %q", fragment)
		}
	}
}

func TestComplexIssueFilterErrorsPreserveBackendEnvelope(t *testing.T) {
	args := []any{uuid.New(), "demo", uuid.New(), uuid.New()}
	_, err := buildIssueComplexFilter(`{"unknown":"value"}`, &args)
	var filterError *FilterError
	if !errors.As(err, &filterError) || filterError.Code != "invalid_filter_field" ||
		filterError.Message != "Filtering on field 'unknown' is not allowed" {
		t.Fatalf("invalid field error=%#v", err)
	}
	_, err = buildIssueComplexFilter(`{`, &args)
	if !errors.As(err, &filterError) || filterError.Code != "invalid_json" {
		t.Fatalf("invalid JSON error=%#v", err)
	}
}

func TestModuleTimestampAndAggregateSQLUseRequesterTimezoneAndTenantScope(t *testing.T) {
	for _, fragment := range []string{"requester_user.user_timezone", "AT TIME ZONE", "to_char"} {
		if sql := moduleJSON("module", "detail"); !strings.Contains(sql, fragment) {
			t.Errorf("module timestamp SQL missing %q", fragment)
		}
	}
	for _, projection := range []string{"active", "archived"} {
		sql := moduleJSON("module", projection)
		for _, fragment := range []string{"requester_user.user_timezone", "'MS'"} {
			if !strings.Contains(sql, fragment) {
				t.Errorf("%s raw list timestamp SQL missing %q", projection, fragment)
			}
		}
	}
	if sql := moduleJSON("module", "active_serializer"); !strings.Contains(sql, "'US'") {
		t.Errorf("serializer timestamp SQL lost microseconds")
	}
	scope := moduleAggregateScope("module", "module_issue", "issue")
	for _, fragment := range []string{
		"module_issue.workspace_id = module.workspace_id",
		"module_issue.project_id = module.project_id",
		"issue.workspace_id = module.workspace_id",
		"issue.project_id = module.project_id",
	} {
		if !strings.Contains(scope, fragment) {
			t.Errorf("aggregate scope missing %q", fragment)
		}
	}
}

func TestModuleWriteProjectionUsesMicrosecondRequesterTimestamps(t *testing.T) {
	query := moduleWriteProjectionSQL()
	if got := strings.Count(query, "'US'"); got != 4 {
		t.Fatalf("module serializer timestamp expressions=%d, want 4 microsecond-preserving fields", got)
	}
	if strings.Contains(query, "'MS'") {
		t.Fatal("module serializer projection uses millisecond timestamp formatting")
	}
	if !strings.Contains(query, "m.archived_at") {
		t.Fatal("module serializer projection omitted archived_at timestamp")
	}
}

func TestModuleIssueOrderingMatchesDjangoRanksAndAllowlist(t *testing.T) {
	tests := map[string]string{
		"":                           "issue.created_at DESC",
		"unknown":                    "issue.created_at DESC",
		"sequence_id":                "issue.sequence_id ASC NULLS LAST",
		"-completed_at":              "issue.completed_at DESC NULLS LAST",
		"priority":                   "WHEN 'urgent' THEN 0",
		"-priority":                  "ELSE 5 END ASC",
		"state__group":               "WHEN 'backlog' THEN 0",
		"-state__group":              "WHEN 'cancelled' THEN 0",
		"assignees__first_name":      "SELECT min(users.first_name)",
		"-labels__name":              "SELECT min(labels.name)",
		"issue_module__module__name": "SELECT modules.name FROM modules WHERE modules.id = $4",
	}
	for value, fragment := range tests {
		if order := moduleIssueOrder(value, true); !strings.Contains(order, fragment) {
			t.Errorf("moduleIssueOrder(%q) = %q, want fragment %q", value, order, fragment)
		}
	}
	if order := moduleIssueOrder("", false); order != "issue.created_at ASC" {
		t.Errorf("absent order = %q", order)
	}
}

func TestIssueDetailLooksUpIssueIDAndReturnsRelationship(t *testing.T) {
	database := &moduleDatabaseStub{rows: []pgx.Row{moduleBoolRow(true), moduleJSONRow(`{"id":"relationship"}`)}}
	repository, _ := NewRepository(database)
	userID, projectID, moduleID, issueID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	if _, err := repository.IssueDetail(context.Background(), userID, "demo", projectID, moduleID, issueID); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(database.queries[0], "project_member.role IN") {
		t.Fatal("module issue detail must remain accessible to active guest project members")
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"FROM module_issues module_issue", "issue.id = $5", "module.id = $4",
		"module.project_id = module_issue.project_id", "module.workspace_id = module_issue.workspace_id",
		"issue.project_id = module_issue.project_id", "issue.workspace_id = module_issue.workspace_id",
		"project.workspace_id = module_issue.workspace_id",
		"module_issue.deleted_at IS NULL", "'module_detail'", "'issue_detail'",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("issue detail query missing %q", fragment)
		}
	}
	if args := database.args[1]; len(args) != 5 || args[4] != issueID {
		t.Fatalf("issue detail args=%#v, want issue_id as fifth argument", args)
	}
	if strings.Contains(query, "AND module.archived_at IS NULL") {
		t.Error("archived modules must retain access to their issue relationships")
	}
}

func TestModuleLinkDetailCannotEscapeModuleOrTenant(t *testing.T) {
	database := &moduleDatabaseStub{rows: []pgx.Row{moduleBoolRow(true), moduleJSONRow(`{"id":"link"}`)}}
	repository, _ := NewRepository(database)
	userID, projectID, moduleID, linkID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	if _, err := repository.LinkDetail(context.Background(), userID, "demo", projectID, moduleID, linkID); err != nil {
		t.Fatal(err)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"workspace.slug = $2", "project.id = $3", "module.id = $4", "module_link.id = $5",
		"module_link.deleted_at IS NULL", "project.archived_at IS NULL",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("link detail query missing %q", fragment)
		}
	}
	if strings.Contains(query, "module.archived_at IS NULL") {
		t.Error("archived modules must retain access to their links")
	}
}

func TestDetailMapsMissingScopedObjectToNotFound(t *testing.T) {
	database := &moduleDatabaseStub{rows: []pgx.Row{
		moduleBoolRow(true), moduleRowFunc(func(...any) error { return pgx.ErrNoRows }),
	}}
	repository, _ := NewRepository(database)
	_, err := repository.Detail(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestArchivedDetailPreservesSparseSuccessForMissingModule(t *testing.T) {
	database := &moduleDatabaseStub{rows: []pgx.Row{
		moduleBoolRow(true), moduleRowFunc(func(...any) error { return pgx.ErrNoRows }),
	}}
	repository, _ := NewRepository(database)
	data, err := repository.ArchivedDetail(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New())
	const expected = `{"distribution":{"assignees":[],"completion_chart":{},"labels":[]},"estimate_distribution":{},"member_ids":[]}`
	if err != nil || string(data) != expected {
		t.Fatalf("data=%s err=%v", data, err)
	}
}

func TestMissingMembershipIsForbidden(t *testing.T) {
	database := &moduleDatabaseStub{rows: []pgx.Row{moduleBoolRow(false)}}
	repository, _ := NewRepository(database)
	_, err := repository.List(context.Background(), uuid.New(), "demo", uuid.New(), false)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err=%v", err)
	}
}
func TestModuleIssueFilterCanonicalizationAndUngroupedM2MSubgroup(t *testing.T) {
	identifier := uuid.New()
	upper := strings.ToUpper(identifier.String())
	ids, err := issueFilterUUIDs([]string{upper}, "state_id")
	if err != nil || len(ids) != 1 || ids[0] != identifier.String() {
		t.Fatalf("complex UUID normalization=%v err=%v", ids, err)
	}
	if normalized, valid := normalizeLegacyUUIDList(upper); !valid || normalized != identifier.String() {
		t.Fatalf("legacy UUID normalization=%q valid=%v", normalized, valid)
	}
	if normalized, valid := normalizeLegacyIntegerList(" 01 , -01 "); !valid || normalized != "1,-1" {
		t.Fatalf("integer normalization=%q valid=%v", normalized, valid)
	}

	database := &moduleDatabaseStub{rows: []pgx.Row{moduleBoolRow(true), modulePageRow(`[]`, 1)}}
	repository, _ := NewRepository(database)
	_, err = repository.Issues(
		context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(),
		IssueListOptions{SubGroupBy: "labels__id", LabelsSet: true, Project: identifier.String(), LoggedBy: "None"},
	)
	if err != nil {
		t.Fatal(err)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"ungrouped_subgroup_axis.subgroup_key", "- 'label_ids'",
		"issue.project_id::text", "issue.logged_by_id IS NULL",
		"NOT EXISTS (SELECT 1 FROM issue_labels presence_relation",
		"EXISTS (SELECT 1 FROM issue_labels presence_relation",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("ungrouped/filter query missing %q", fragment)
		}
	}
}
func TestComplexIssueFilterDjangoCoercionEdges(t *testing.T) {
	identifier := uuid.New()
	args := []any{uuid.New(), "demo", uuid.New(), uuid.New()}
	predicate, err := buildIssueComplexFilter(
		`{"and":[{"state_id__in":"`+identifier.String()+`,"},{"priority__in":" high , low "},{"is_archived":1.0}]}`,
		&args,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(predicate, "issue.state_id::text") || !strings.Contains(predicate, "TRUE") {
		t.Fatalf("predicate=%s", predicate)
	}
	if args[len(args)-1] != "high,low" {
		t.Fatalf("char CSV was not stripped: %#v", args)
	}
	if ids, err := issueFilterUUIDs([]string{identifier.String() + ","}, "state_id__in"); err != nil || len(ids) != 1 || ids[0] != identifier.String() {
		t.Fatalf("UUID CSV normalization=%v err=%v", ids, err)
	}

	for raw, expected := range map[string]string{
		`{"label_id__in":[]}`: "FALSE",
		`{"label_id":null}`:   "NOT EXISTS",
		`{"is_draft":"bad"}`:  "issue.is_draft IS NULL",
		`{"is_archived":NaN}`: "TRUE",
		`{"is_draft":-0}`:     "issue.is_draft =",
	} {
		filterArgs := []any{uuid.New(), "demo", uuid.New(), uuid.New()}
		rendered, renderErr := buildIssueComplexFilter(raw, &filterArgs)
		if renderErr != nil || !strings.Contains(rendered, expected) {
			t.Errorf("%s => %q err=%v, want %q", raw, rendered, renderErr, expected)
		}
	}
	filterArgs := []any{uuid.New(), "demo", uuid.New(), uuid.New()}
	if rendered, err := buildIssueComplexFilter(`{"created_at__range":"2024-01-01,"}`, &filterArgs); err != nil || rendered != "FALSE" {
		t.Fatalf("nullable date range=%q err=%v", rendered, err)
	}
	filterArgs = []any{uuid.New(), "demo", uuid.New(), uuid.New()}
	if _, err := buildIssueComplexFilter(`{"created_at__range":"2024-01-01"}`, &filterArgs); !errors.Is(err, ErrInvalidComplexRange) {
		t.Fatalf("wrong range arity err=%v", err)
	}
}

func TestRelatedOrderingUsesDjangoFilterAliases(t *testing.T) {
	identifier := uuid.New().String()
	for _, test := range []struct {
		options IssueListOptions
		values  [3]string
		want    []string
	}{
		{
			options: IssueListOptions{OrderBy: "assignees__first_name", OrderBySet: true, AssigneesSet: true},
			want:    []string{"issue_assignees order_relation", "order_relation.deleted_at IS NULL"},
		},
		{
			options: IssueListOptions{OrderBy: "labels__name", OrderBySet: true, Filters: `{"label_id":"` + identifier + `"}`},
			want:    []string{"issue_labels order_relation", "order_relation.label_id::text", "order_relation.deleted_at IS NULL"},
		},
		{
			options: IssueListOptions{OrderBy: "issue_module__module__name", OrderBySet: true, Module: identifier, ModuleSet: true},
			values:  [3]string{"", "", identifier},
			want:    []string{"module_issues order_relation", "order_relation.module_id::text", "min(order_name.name)"},
		},
	} {
		args := []any{uuid.New(), "demo", uuid.New(), uuid.New()}
		order, err := moduleIssueOrderWithRelations(test.options, test.values[0], test.values[1], test.values[2], &args)
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range test.want {
			if !strings.Contains(order, fragment) {
				t.Errorf("order %q missing %q", order, fragment)
			}
		}
	}
}

func TestUngroupedM2MSubgroupSeparatesPageMetadataFromExpandedItems(t *testing.T) {
	database := &moduleDatabaseStub{rows: []pgx.Row{moduleBoolRow(true), modulePageRow(`[]`, 1)}}
	repository, _ := NewRepository(database)
	_, err := repository.Issues(
		context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(),
		IssueListOptions{SubGroupBy: "labels__id", Limit: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"subgroup_page_base AS", "FROM subgroup_page_base",
		"page_relation.deleted_at IS NULL", "FROM filtered issue\n  CROSS JOIN LATERAL",
		") page_slice",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("subgroup page query missing %q", fragment)
		}
	}
}
