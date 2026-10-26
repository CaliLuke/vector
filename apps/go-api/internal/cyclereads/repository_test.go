package cyclereads

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"

	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type rowFunc func(...any) error

func (row rowFunc) Scan(destinations ...any) error { return row(destinations...) }

type databaseStub struct {
	queries []string
	args    [][]any
	rows    []pgx.Row
}

func (database *databaseStub) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("unexpected transaction")
}

func (database *databaseStub) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	database.queries = append(database.queries, query)
	database.args = append(database.args, args)
	if len(database.rows) == 0 {
		return rowFunc(func(...any) error { return errors.New("unexpected query") })
	}
	row := database.rows[0]
	database.rows = database.rows[1:]
	return row
}

func (database *databaseStub) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	database.queries = append(database.queries, query)
	database.args = append(database.args, args)
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func boolRow(value bool) pgx.Row {
	return rowFunc(func(destinations ...any) error {
		*destinations[0].(*bool) = value
		return nil
	})
}

func jsonRow(value string) pgx.Row {
	return rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(value)
		return nil
	})
}

func pageRow(value string, total int) pgx.Row {
	return pageRowWithExpanded(value, total, total)
}

func pageRowWithExpanded(value string, total int, expandedTotal int) pgx.Row {
	return rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(value)
		*destinations[1].(*int) = total
		*destinations[2].(*[]byte) = []byte(`[]`)
		*destinations[3].(*[]byte) = []byte(`[]`)
		*destinations[4].(*int) = expandedTotal
		return nil
	})
}

func groupedPageRow(value string, total int, count int, maximum int, next bool) pgx.Row {
	return rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(value)
		*destinations[1].(*int) = total
		*destinations[2].(*int) = count
		*destinations[3].(*int) = maximum
		*destinations[4].(*bool) = next
		*destinations[5].(*bool) = false
		return nil
	})
}

func missingGroupedPageRow() pgx.Row {
	return rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(`{}`)
		*destinations[1].(*int) = 1
		*destinations[2].(*int) = 1
		*destinations[3].(*int) = 1
		*destinations[4].(*bool) = false
		*destinations[5].(*bool) = true
		return nil
	})
}

func analyticsRow(datesPresent bool, value string) pgx.Row {
	return rowFunc(func(destinations ...any) error {
		*destinations[0].(*bool) = datesPresent
		*destinations[1].(*[]byte) = []byte(value)
		return nil
	})
}

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) returned nil error")
	}
}

func TestAuthorizationScopesSessionPrincipalRoleAndTenant(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true), boolRow(true)}}
	repository, _ := NewRepository(database)
	userID, projectID := uuid.New(), uuid.New()
	if err := repository.AuthorizeWorkspace(context.Background(), userID, "alpha"); err != nil {
		t.Fatal(err)
	}
	if err := repository.AuthorizeProject(context.Background(), userID, "alpha", projectID, false); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"workspace_member.member_id = $1", "workspace.slug = $2", "workspace_member.is_active = true",
		"workspace_member.deleted_at IS NULL", "workspace.deleted_at IS NULL",
	} {
		if !strings.Contains(database.queries[0], fragment) {
			t.Errorf("workspace authorization missing %q", fragment)
		}
	}
	for _, fragment := range []string{
		"project_member.member_id = $1", "workspace.slug = $2", "project.id = $3",
		"project_member.role IN (15, 20)", "$4::boolean AND project_member.role = 5",
		"workspace_admin.role = 20", "project_member.is_active = true", "project_member.deleted_at IS NULL",
	} {
		if !strings.Contains(database.queries[1], fragment) {
			t.Errorf("project authorization missing %q", fragment)
		}
	}
	if database.args[1][3] != false {
		t.Fatalf("allowGuest=%v, want false", database.args[1][3])
	}
}

func TestAuthorizationRejectsMissingActiveMembership(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(false), boolRow(false)}}
	repository, _ := NewRepository(database)
	if err := repository.AuthorizeWorkspace(context.Background(), uuid.New(), "alpha"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("workspace err=%v", err)
	}
	if err := repository.AuthorizeProject(context.Background(), uuid.New(), "alpha", uuid.New(), true); !errors.Is(err, ErrForbidden) {
		t.Fatalf("project err=%v", err)
	}
}

func TestWorkspaceListRequiresWorkspaceAndPerProjectMembership(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true), jsonRow(`[]`)}}
	repository, _ := NewRepository(database)
	if _, err := repository.WorkspaceList(context.Background(), uuid.New(), "alpha"); err != nil {
		t.Fatal(err)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"workspace.slug=$2", "cycle.archived_at IS NULL", "project.archived_at IS NULL",
		"project_member.member_id=$1", "project_member.is_active=true", "cycle_issue.deleted_at IS NULL",
		"issue.is_draft=false", "issue.deleted_at IS NULL", "requester_user.user_timezone", "|| ':' ||",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("workspace list query missing %q", fragment)
		}
	}
	if !strings.Contains(workspaceCycleIssueCountsSQL, "count(cycle_issue.id)") ||
		strings.Contains(workspaceCycleIssueCountsSQL, "count(DISTINCT issue.id)") ||
		!strings.Contains(cycleIssueCountsSQL, "count(DISTINCT issue.id)") {
		t.Fatal("workspace counts must preserve duplicate bridges without changing project counts")
	}
}

func TestActiveAndArchivedListsUseDistinctScopesAndNoCurrentFallback(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{
		boolRow(true), jsonRow(`[]`), boolRow(true), jsonRow(`[]`),
	}}
	repository, _ := NewRepository(database)
	userID, projectID := uuid.New(), uuid.New()
	if _, err := repository.List(context.Background(), userID, "alpha", projectID, "current"); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.ArchivedList(context.Background(), userID, "alpha", projectID); err != nil {
		t.Fatal(err)
	}
	if database.args[0][3] != true {
		t.Fatalf("active cycle list must permit guest role: allowGuest=%v", database.args[0][3])
	}
	if database.args[2][3] != false {
		t.Fatalf("archived cycle list must reject guest role: allowGuest=%v", database.args[2][3])
	}
	activeQuery := database.queries[1]
	for _, fragment := range []string{
		"cycle.archived_at IS NULL", "cycle.start_date <= now()",
		"cycle.start_date AT TIME ZONE project.timezone", "ORDER BY favorite DESC, created_at DESC",
	} {
		if !strings.Contains(activeQuery, fragment) {
			t.Errorf("active query missing %q", fragment)
		}
	}
	if strings.Contains(activeQuery, "OR NOT EXISTS") {
		t.Error("cycle_view=current must not return every active cycle when there is no current cycle")
	}
	archivedQuery := database.queries[3]
	for _, fragment := range []string{"cycle.archived_at IS NOT NULL", "cycle.archived_at AT TIME ZONE 'UTC'", "|| 'Z'"} {
		if !strings.Contains(archivedQuery, fragment) {
			t.Fatalf("archived query missing %q: %s", fragment, archivedQuery)
		}
	}
	if strings.Contains(archivedQuery, "AND cycle.archived_at IS NULL") {
		t.Fatalf("archived query includes active-cycle predicate: %s", archivedQuery)
	}
}

func TestArchiveDetailPathsUseCycleIDAndDistinctArchivedScopes(t *testing.T) {
	missing := rowFunc(func(...any) error { return pgx.ErrNoRows })
	database := &databaseStub{rows: []pgx.Row{
		boolRow(true), missing,
		boolRow(true), jsonRow(`{"id":"00000000-0000-4000-8000-000000000001","archived_at":null}`),
	}}
	repository, _ := NewRepository(database)
	userID, projectID, cycleID := uuid.New(), uuid.New(), uuid.New()
	_, archivedErr := repository.ArchivedDetail(context.Background(), userID, "alpha", projectID, cycleID)
	if !errors.Is(archivedErr, ErrNotFound) {
		t.Fatalf("archivedErr=%v", archivedErr)
	}
	if _, err := repository.ArchiveDetail(context.Background(), userID, "alpha", projectID, cycleID); err != nil {
		t.Fatalf("active archive route err=%v", err)
	}
	if !strings.Contains(database.queries[1], "cycle.id=$4") || !strings.Contains(database.queries[1], "cycle.archived_at IS NOT NULL") {
		t.Errorf("archived detail query does not use route cycle id and archived scope")
	}
	if !strings.Contains(database.queries[3], "cycle.id=$4") ||
		!strings.Contains(database.queries[3], "cycle.archived_at IS NULL") ||
		strings.Contains(database.queries[3], "cycle.archived_at IS NOT NULL") {
		t.Errorf("active archive query does not use route cycle id and active scope")
	}
	if !strings.Contains(database.queries[1], "WHERE total_ci.cycle_id=cycle.id AND total_issue.deleted_at IS NULL") {
		t.Error("archived issue burndown must start from all cycle bridges")
	}
	if !strings.Contains(database.queries[3], "WHERE total_ci.cycle_id=cycle.id AND total_ci.deleted_at IS NULL AND total_issue.deleted_at IS NULL") {
		t.Error("active archive issue burndown must start from active cycle bridges")
	}
	if got := database.args[3][3]; got != cycleID {
		t.Errorf("archive detail arg=%v, want %v", got, cycleID)
	}
}

func TestCycleIssueDetailLooksUpRelationshipForAnyActiveProjectMember(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{jsonRow(`{"id":"00000000-0000-4000-8000-000000000001"}`)}}
	repository, _ := NewRepository(database)
	userID, projectID, cycleID, issueID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	if _, err := repository.IssueDetail(context.Background(), userID, "alpha", projectID, cycleID, issueID); err != nil {
		t.Fatal(err)
	}
	query := database.queries[0]
	for _, fragment := range []string{
		"cycle_issue.cycle_id=$4", "cycle_issue.issue_id=$5", "cycle_issue.deleted_at IS NULL",
		"project.archived_at IS NULL", "'issue_detail'", "'sub_issues_count'",
		"detail_member.member_id=$1", "detail_member.is_active=true",
		"'is_bot',user_account.is_bot", "'logo_props',project.logo_props",
		"ORDER BY user_account.created_at DESC", "ORDER BY label.created_at DESC",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("cycle-issue detail query missing %q", fragment)
		}
	}
	if strings.Contains(query, "detail_member.role") {
		t.Error("cycle-issue detail must permit active guest members")
	}
	for _, leakedProjection := range []string{
		"to_jsonb(user_account)", "to_jsonb(label)", "'email',user_account.email", "'icon_prop',project.icon_prop",
	} {
		if strings.Contains(query, leakedProjection) {
			t.Errorf("cycle-issue detail leaks non-lite projection %q", leakedProjection)
		}
	}
	if !strings.Contains(query, "'archived_at',issue.archived_at") {
		t.Error("cycle-issue archived_at must remain a Django DateField")
	}
	if database.args[0][4] != issueID {
		t.Fatalf("lookup arg=%v, want issue id %v", database.args[0][4], issueID)
	}
}

func TestCycleIssueDetailReturnsNotFoundForOutsider(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{rowFunc(func(...any) error { return pgx.ErrNoRows })}}
	repository, _ := NewRepository(database)
	_, err := repository.IssueDetail(context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestIssueListAppliesFiltersOrderingPaginationAndEnvelope(t *testing.T) {
	issueID := uuid.New()
	rows := `[{"id":"` + issueID.String() + `","priority":"high","state__group":"started","assignee_ids":[],"label_ids":[],"module_ids":[]}]`
	database := &databaseStub{rows: []pgx.Row{boolRow(true), pageRow(rows, 3)}}
	repository, _ := NewRepository(database)
	data, err := repository.Issues(context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(), IssueListOptions{
		OrderBy: "-priority", Cursor: "1:1:0", PerPage: 1,
		Filters: map[string]string{"priority": "high,urgent", "name": "login", "sub_issue": "false"},
	})
	if err != nil {
		t.Fatal(err)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"issue.priority = ANY", "issue.name ILIKE", "issue.parent_id IS NULL",
		"CASE issue.priority WHEN 'urgent' THEN 0", "ASC NULLS LAST", "LIMIT $", "OFFSET $",
		"cycle_issue.deleted_at IS NULL", "issue.completed_at AT TIME ZONE 'UTC'",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("issue list query missing %q", fragment)
		}
	}
	var envelope map[string]any
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{
		"grouped_by", "sub_grouped_by", "total_count", "next_cursor", "prev_cursor",
		"next_page_results", "prev_page_results", "count", "total_pages", "total_results", "extra_stats", "results",
	} {
		if _, ok := envelope[field]; !ok {
			t.Errorf("pagination envelope missing %q", field)
		}
	}
	if envelope["total_results"] != float64(3) || envelope["prev_page_results"] != true {
		t.Fatalf("envelope=%v", envelope)
	}
}

func TestIssueFiltersCoverLegacyDatesNullsAndAdvertisedRelations(t *testing.T) {
	args := []any{}
	predicates := issueFilterPredicates(map[string]string{
		"labels":        "None",
		"assignees":     "None",
		"module":        "None",
		"cycle":         "None",
		"created_by":    "None",
		"logged_by":     "None",
		"created_at":    "2_weeks;after;fromnow",
		"updated_at":    "2026-09-01;before",
		"start_date":    "2026-09",
		"target_date":   "1_months;before;fromnow",
		"completed_at":  "2026-10-01;after",
		"intake_status": "-2,0",
		"inbox_status":  "1",
		"mentions":      uuid.New().String(),
	}, &args)
	query := strings.Join(predicates, " ")
	for _, fragment := range []string{
		"NOT EXISTS (SELECT 1 FROM issue_labels no_label",
		"NOT EXISTS (SELECT 1 FROM issue_assignees no_assignee",
		"NOT EXISTS (SELECT 1 FROM module_issues no_module",
		"NOT EXISTS (SELECT 1 FROM cycle_issues no_cycle",
		"issue.logged_by_id IS NULL",
		"issue.created_by_id IS NULL",
		"requester_user.user_timezone",
		"issue.start_date::date::text LIKE",
		"issue.target_date::date <= CURRENT_DATE",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("filters missing %q: %s", fragment, query)
		}
	}
	for _, fragment := range []string{
		"no_assignee.deleted_at",
		"no_module.deleted_at",
		"filter_mention.deleted_at",
		"filter_intake.deleted_at",
	} {
		if strings.Contains(query, fragment) {
			t.Errorf("source relation filter must include soft-deleted through rows, found %q: %s", fragment, query)
		}
	}
	overwrittenArgs := []any{}
	overwritten := legacyDatePredicates(
		"2026-01-01;after,2026-02-01;after", "issue.created_at", &overwrittenArgs,
	)
	if len(overwritten) != 1 || len(overwrittenArgs) != 1 || overwrittenArgs[0] != "2026-02-01" {
		t.Fatalf("repeated Django date lookup did not use last clause: predicates=%v args=%v", overwritten, overwrittenArgs)
	}
	falseNullArgs := make([]any, 0)
	falseNullPredicates := issueFilterPredicates(map[string]string{
		"parent":     "xNone",
		"labels":     "NotNone",
		"assignees":  "NoneSuch",
		"module":     "ModuleNone",
		"cycle":      "CycleNone",
		"created_by": "SomeoneNone",
		"logged_by":  "NoneOther",
	}, &falseNullArgs)
	if len(falseNullPredicates) != 0 || len(falseNullArgs) != 0 {
		t.Fatalf("None substrings must not activate null filters: predicates=%v args=%v", falseNullPredicates, falseNullArgs)
	}
	if got := CSV("started, completed"); len(got) != 2 || got[1] != " completed" {
		t.Fatalf("legacy CSV must preserve source whitespace: %v", got)
	}
	id := uuid.New()
	if got := UUIDs(id.String() + ", " + uuid.New().String()); len(got) != 1 || got[0] != id {
		t.Fatalf("legacy UUID parsing must reject whitespace-wrapped tokens: %v", got)
	}
}

func TestIssueOrderingPreservesDjangoDirectionQuirks(t *testing.T) {
	for input, suffix := range map[string]string{
		"priority":      "DESC NULLS LAST, issue.created_at DESC",
		"-priority":     "ASC NULLS LAST, issue.created_at DESC",
		"state__group":  "ASC NULLS LAST, issue.created_at DESC",
		"-state__group": "ASC NULLS LAST, issue.created_at DESC",
	} {
		order := issueOrder(input)
		if !strings.Contains(order, "CASE ") || !strings.HasSuffix(order, suffix) {
			t.Errorf("issueOrder(%q)=%q, want suffix %q", input, order, suffix)
		}
	}
	for _, field := range []string{"assignees__first_name", "issue_module__module__name"} {
		if order := issueOrder(field); strings.Contains(order, "deleted_at") {
			t.Errorf("relation ordering must include soft-deleted through rows: %s", order)
		}
	}
}

func TestUserTimestampSQLUsesRequesterTimezoneForWorkspaceAndIssueDetail(t *testing.T) {
	expression := userTimestampSQL("issue.created_at")
	for _, fragment := range []string{"requester_user.user_timezone", "requester_user.id = $1", "THEN '' ELSE '.' || to_char"} {
		if !strings.Contains(expression, fragment) {
			t.Errorf("user timestamp missing %q: %s", fragment, expression)
		}
	}
	if strings.Contains(workspaceCycleListSQL, utcTimestampSQL("cycle.start_date")) ||
		!strings.Contains(workspaceCycleListSQL, userTimestampSQL("cycle.start_date")) {
		t.Fatal("workspace cycle dates must use requester timezone")
	}
	for _, column := range []string{
		"cycle_issue.created_at", "cycle_issue.updated_at", "cycle_issue.deleted_at",
		"issue.created_at", "issue.updated_at", "issue.deleted_at", "issue.completed_at",
	} {
		if !strings.Contains(cycleIssueDetailSQL, userTimestampSQL(column)) {
			t.Errorf("cycle issue detail timestamp %s does not use requester timezone", column)
		}
	}
	if strings.Contains(cycleIssueDetailSQL, "cycle_issue.deleted_at IS NULL AND issue.deleted_at IS NULL") ||
		strings.Contains(cycleIssueDetailSQL, "assignee.issue_id=issue.id AND assignee.deleted_at IS NULL") ||
		!strings.Contains(cycleIssueDetailSQL, "live_label.deleted_at IS NULL") ||
		!strings.Contains(cycleIssueDetailSQL, "replace(encode(issue.description_binary,'base64'),E'\\n','')") {
		t.Fatal("cycle issue detail does not match source manager and BinaryField serialization semantics")
	}
}

func TestTimestampSQLMatchesDjangoFractionAndTimezoneForms(t *testing.T) {
	project := projectTimestampSQL("cycle.start_date")
	for _, fragment := range []string{"date_trunc('second'", "THEN '' ELSE '.' || to_char", "AT TIME ZONE project.timezone"} {
		if !strings.Contains(project, fragment) {
			t.Errorf("project timestamp missing %q: %s", fragment, project)
		}
	}
	utc := utcTimestampSQL("cycle.start_date")
	for _, fragment := range []string{"AT TIME ZONE 'UTC'", "date_trunc('second'", "|| 'Z'"} {
		if !strings.Contains(utc, fragment) {
			t.Errorf("UTC timestamp missing %q: %s", fragment, utc)
		}
	}
}

func TestIssueListPaginatesEachOuterGroupAndIncludesOnlyExistingSubgroups(t *testing.T) {
	results := `{
		"started":{"total_results":3,"results":{
			"high":{"results":[{"id":"00000000-0000-4000-8000-000000000001"}],"total_results":2},
			"low":{"results":[{"id":"00000000-0000-4000-8000-000000000003"}],"total_results":1}}},
		"completed":{"total_results":1,"results":{
			"urgent":{"results":[{"id":"00000000-0000-4000-8000-000000000004"}],"total_results":1}}},
		"backlog":{"total_results":0,"results":{}}
	}`
	database := &databaseStub{rows: []pgx.Row{
		boolRow(true), groupedPageRow(results, 4, 3, 3, true),
	}}
	repository, _ := NewRepository(database)
	data, err := repository.Issues(context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(), IssueListOptions{
		GroupBy: "state__group", SubGroupBy: "priority", PerPage: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(database.queries) != 2 {
		t.Fatalf("grouped request issued %d queries, want authorization plus one window query", len(database.queries))
	}
	for _, fragment := range []string{
		"row_number() OVER (PARTITION BY group_key,subgroup_key ORDER BY position)",
		"SELECT DISTINCT group_key,subgroup_key FROM expanded",
		"count(DISTINCT expanded.issue_id)",
		"CASE WHEN EXISTS(SELECT 1 FROM paged)",
	} {
		if !strings.Contains(database.queries[1], fragment) {
			t.Errorf("grouped window query missing %q", fragment)
		}
	}
	if strings.Contains(database.queries[1], "%!") {
		t.Fatalf("grouped SQL format mismatch: %s", database.queries[1])
	}
	if len(database.args[1]) != 6 || database.args[1][4] != float64(0) || database.args[1][5] != float64(2) {
		t.Fatalf("grouped SQL args=%v, want lower bound 0 and upper bound 2", database.args[1])
	}
	var envelope struct {
		Count      int `json:"count"`
		TotalPages int `json:"total_pages"`
		Results    map[string]struct {
			Total   int `json:"total_results"`
			Results map[string]struct {
				Total int              `json:"total_results"`
				Rows  []map[string]any `json:"results"`
			} `json:"results"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Results["started"].Total != 3 ||
		envelope.Results["started"].Results["high"].Total != 2 ||
		len(envelope.Results["started"].Results["high"].Rows) != 1 ||
		len(envelope.Results["started"].Results) != 2 {
		t.Fatalf("started group=%s", data)
	}
	if envelope.Results["completed"].Results["urgent"].Total != 1 ||
		len(envelope.Results["completed"].Results) != 1 {
		t.Fatalf("completed group=%s", data)
	}
	if envelope.Results["backlog"].Total != 0 || len(envelope.Results["backlog"].Results) != 0 {
		t.Fatalf("empty outer group=%s", data)
	}
	if envelope.Count != 3 {
		t.Fatalf("count=%d, want rows selected across group windows", envelope.Count)
	}
	if envelope.TotalPages != 3 {
		t.Fatalf("total_pages=%d, want largest outer-group page count", envelope.TotalPages)
	}
}

func TestGroupedIssueFollowUpUsesCursorLimitForOffsetAndWindow(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{
		boolRow(true), groupedPageRow(`{}`, 0, 0, 0, false),
	}}
	repository, _ := NewRepository(database)
	data, err := repository.Issues(
		context.Background(),
		uuid.New(),
		"alpha",
		uuid.New(),
		uuid.New(),
		IssueListOptions{GroupBy: "priority", Cursor: "1:1:0"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(database.args) != 2 || len(database.args[1]) != 6 ||
		database.args[1][4] != float64(1) || database.args[1][5] != float64(3) {
		t.Fatalf("grouped SQL args=%v, want lower bound 1 and upper bound 3", database.args)
	}
	var envelope struct {
		NextCursor string `json:"next_cursor"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.NextCursor != "1000:2:0" {
		t.Fatalf("next_cursor=%q, want request limit retained", envelope.NextCursor)
	}
}

func TestGroupedPreviousCursorKeepsSQLNextPageResult(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{
		boolRow(true), groupedPageRow(`{}`, 3, 2, 2, true),
	}}
	repository, _ := NewRepository(database)
	data, err := repository.Issues(
		context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{
			GroupBy: "priority", PerPage: 25, PerPageSet: true,
			Cursor: "1:1:1", CursorSet: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		NextPageResults bool `json:"next_page_results"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.NextPageResults {
		t.Fatalf("grouped previous cursor discarded SQL next-page result: %s", data)
	}
}

func TestGroupedIssueEmptyPageHasNoBucketsOrTotalPages(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{
		boolRow(true), groupedPageRow(`{}`, 7, 0, 0, false),
	}}
	repository, _ := NewRepository(database)
	data, err := repository.Issues(
		context.Background(),
		uuid.New(),
		"alpha",
		uuid.New(),
		uuid.New(),
		IssueListOptions{GroupBy: "priority", PerPage: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Count        int                       `json:"count"`
		TotalCount   int                       `json:"total_count"`
		TotalPages   int                       `json:"total_pages"`
		TotalResults int                       `json:"total_results"`
		Results      map[string]map[string]any `json:"results"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Results) != 0 || envelope.TotalPages != 0 {
		t.Fatalf("empty page must not seed buckets or pages: %s", data)
	}
	if envelope.Count != 0 || envelope.TotalCount != 7 || envelope.TotalResults != 7 {
		t.Fatalf("empty page must report zero selected rows and preserve expanded queryset totals: %s", data)
	}
}

func TestGroupedIssueSQLUsesPageGroupsForManyToManyAndRewritesMappedIDs(t *testing.T) {
	query, _, _ := groupedIssueListQuery(
		uuid.New(),
		"alpha",
		uuid.New(),
		uuid.New(),
		IssueListOptions{GroupBy: "labels__id"}, 30, 30, 0,
	)
	for _, fragment := range []string{
		"SELECT DISTINCT group_key,'__none__'::text subgroup_key FROM paged",
		"jsonb_build_object('label_ids'",
		"selected.group_key",
		"jsonb_agg(DISTINCT selected.group_key)",
		"live_label.deleted_at IS NULL",
		"SELECT DISTINCT ON (processed.issue_id)",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("many-to-many grouped SQL missing %q", fragment)
		}
	}
	if strings.Contains(query, "%!") {
		t.Fatalf("grouped SQL format mismatch: %s", query)
	}
	subgrouped, _, _ := groupedIssueListQuery(
		uuid.New(),
		"alpha",
		uuid.New(),
		uuid.New(),
		IssueListOptions{GroupBy: "labels__id", SubGroupBy: "priority"}, 30, 30, 0,
	)
	if strings.Contains(subgrouped, "SELECT DISTINCT ON (processed.issue_id)") {
		t.Fatal("subgrouped paginator must preserve duplicate source rows")
	}
	if !strings.Contains(subgrouped, "SELECT false AND EXISTS") {
		t.Fatal("M2M subgrouping must not inherit scalar paginator failure")
	}
	scalarSubgrouped, _, _ := groupedIssueListQuery(
		uuid.New(),
		"alpha",
		uuid.New(),
		uuid.New(),
		IssueListOptions{GroupBy: "state_id", SubGroupBy: "priority"}, 30, 30, 0,
	)
	for _, fragment := range []string{
		"SELECT true AND EXISTS",
		"NOT (universes.group_values ? paged.group_key)",
		"(SELECT missing_group_key FROM source_guard)",
	} {
		if !strings.Contains(scalarSubgrouped, fragment) {
			t.Fatalf("scalar subgroup source-failure parity missing %q", fragment)
		}
	}
}

func TestModuleGroupingUsesRelationIDsButOtherPayloadsOmitArchivedModules(t *testing.T) {
	query, _, _ := groupedIssueListQuery(
		uuid.New(),
		"alpha",
		uuid.New(),
		uuid.New(),
		IssueListOptions{GroupBy: "issue_module__module_id"}, 30, 30, 0,
	)
	for _, fragment := range []string{
		"grouped_module.issue_id=matching.issue_id",
		"projected_module.archived_at IS NULL",
		"jsonb_build_object('module_ids'",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("module grouped SQL missing %q", fragment)
		}
	}
	if projection := issueListRelationProjection(IssueListOptions{}); !strings.Contains(projection, "projected_module.deleted_at IS NULL") || !strings.Contains(projection, "projected_module.archived_at IS NULL") {
		t.Fatalf("ordinary issue payloads must omit deleted and archived module IDs: %s", projection)
	}
}

func TestGroupedTotalsApplyDjangoIntakeCountFilter(t *testing.T) {
	grouped := issueBucketTotalSQL(IssueListOptions{GroupBy: "priority"})
	if !strings.Contains(grouped, "FILTER (WHERE expanded.countable)") ||
		!strings.Contains(grouped, "THEN GREATEST") {
		t.Fatalf("grouped total must replace an existing zero filtered count with one: %s", grouped)
	}
	subgrouped := issueBucketTotalSQL(IssueListOptions{GroupBy: "priority", SubGroupBy: "state_id"})
	if !strings.Contains(subgrouped, "FILTER (WHERE expanded.countable)") ||
		strings.Contains(subgrouped, "GREATEST") {
		t.Fatalf("subgroup total must preserve a zero filtered count: %s", subgrouped)
	}
	query, _, _ := groupedIssueListQuery(
		uuid.New(),
		"alpha",
		uuid.New(),
		uuid.New(),
		IssueListOptions{GroupBy: "priority"}, 30, 30, 0,
	)
	for _, fragment := range []string{
		"count_intake.status IN (-1,1,2)",
		"max(group_total.countable_results)",
		"(SELECT count(*)::int FROM expanded)",
		"(SELECT count(*)::int FROM paged)",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("grouped intake-count SQL missing %q", fragment)
		}
	}
	if strings.Contains(query, "count_intake.deleted_at") {
		t.Fatalf("Django count filter includes soft-deleted intake relations: %s", query)
	}
}

func TestIssueGroupUniversesMatchDjangoHelpers(t *testing.T) {
	if universe := issueGroupUniverseSQL("priority"); universe != `'["low","medium","high","urgent","none"]'::json` {
		t.Fatalf("priority universe=%s", universe)
	}

	assignees := issueGroupUniverseSQL("assignees__id")
	if strings.Contains(assignees, "SELECT 'None'") ||
		!strings.Contains(assignees, "workspace.slug=$2") ||
		!strings.Contains(assignees, "project_member.project_id=$3") {
		t.Fatalf("assignee universe=%s", assignees)
	}
	projects := issueGroupUniverseSQL("project_id")
	if !strings.Contains(projects, "workspace.slug=$2") || strings.Contains(projects, "project.id=$3") {
		t.Fatalf("project universe=%s", projects)
	}
	states := issueGroupUniverseSQL("state_id")
	if !strings.Contains(states, `state."group"<>'triage'`) {
		t.Fatalf("state universe must apply StateManager group exclusion: %s", states)
	}
}
func TestComplexIssueFilterRendersNestedLogicAndBackendErrors(t *testing.T) {
	priority := "urgent"
	stateID := uuid.New()
	args := []any{uuid.New(), "demo", uuid.New(), uuid.New()}
	clause, err := buildIssueComplexFilter(
		`{"or":[{"priority":"`+priority+`"},{"not":{"state_id":"`+stateID.String()+`"}}]}`,
		&args,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"issue.priority =", " OR ", "NOT (", "issue.state_id::text"} {
		if !strings.Contains(clause, fragment) {
			t.Errorf("complex filter missing %q: %s", fragment, clause)
		}
	}
	for _, test := range []struct {
		filter string
		want   string
	}{
		{`{"is_archived":"2"}`, "issue.archived_at IS NOT NULL"},
		{`{"is_archived":"3"}`, "issue.archived_at IS NULL"},
		{`{"is_archived":"1"}`, "TRUE"},
		{`{"is_archived":1}`, "TRUE"},
		{`{"is_draft":"unknown"}`, "issue.is_draft IS NULL"},
	} {
		booleanArgs := make([]any, 0)
		got, booleanErr := buildIssueComplexFilter(test.filter, &booleanArgs)
		if booleanErr != nil || got != test.want {
			t.Errorf("filter=%s clause=%q err=%v, want %q", test.filter, got, booleanErr, test.want)
		}
	}
	for _, filter := range []string{
		`{"state_id__in":null}`,
		`{"project_id__in":""}`,
		`{"created_by_id__in":null}`,
		`{"state_group__in":""}`,
		`{"priority__in":null}`,
	} {
		emptyArgs := make([]any, 0)
		got, emptyErr := buildIssueComplexFilter(filter, &emptyArgs)
		if emptyErr != nil || got != "FALSE" {
			t.Errorf("empty direct __in filter=%s clause=%q err=%v", filter, got, emptyErr)
		}
	}
	relationArgs := make([]any, 0)
	relationEmpty, relationErr := buildIssueComplexFilter(`{"label_id__in":null}`, &relationArgs)
	if relationErr != nil || relationEmpty != "TRUE" {
		t.Fatalf("empty relation __in clause=%q err=%v", relationEmpty, relationErr)
	}
	dateArgs := make([]any, 0)
	dateClause, dateErr := buildIssueComplexFilter(`{"created_at__exact":" 07/31/2026 "}`, &dateArgs)
	if dateErr != nil || !strings.Contains(dateClause, "issue.created_at") ||
		len(dateArgs) != 1 || dateArgs[0] != "2026-07-31" {
		t.Fatalf("localized rich date clause=%q args=%v err=%v", dateClause, dateArgs, dateErr)
	}
	_, err = buildIssueComplexFilter(`{"unknown":"value"}`, &args)
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

func TestIssueQueriesApplyComplexFilterBeforeLegacyFilters(t *testing.T) {
	query, args, err := issueListQuery(
		uuid.New(),
		"alpha",
		uuid.New(),
		uuid.New(),
		IssueListOptions{
			ComplexFilter: `{"priority":"urgent"}`,
			Filters:       map[string]string{"state_group": "started"},
		},
		30,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "issue.priority = $5") ||
		!strings.Contains(query, `state."group" = ANY($6::text[])`) {
		t.Fatalf("issue SQL did not apply complex then legacy filters: %s", query)
	}
	if len(args) != 8 {
		t.Fatalf("issue args=%v", args)
	}
}

func TestIssueRelationFiltersPreserveDjangoJoinMultiplicity(t *testing.T) {
	labelA, labelB := uuid.New(), uuid.New()
	query, args, err := issueListQuery(
		uuid.New(),
		"alpha",
		uuid.New(),
		uuid.New(),
		IssueListOptions{
			Filters: map[string]string{
				"labels":        labelA.String() + "," + labelB.String(),
				"intake_status": "-2",
				"inbox_status":  "1,2",
			},
		},
		30,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"JOIN issue_labels legacy_label",
		"legacy_label.label_id=ANY($5::uuid[])",
		"JOIN intake_issues legacy_intake",
		"legacy_intake.status::text=ANY($6::text[])",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("legacy multiplicity SQL missing %q: %s", fragment, query)
		}
	}
	if got := args[5].([]string); len(got) != 2 || got[0] != "1" {
		t.Fatalf("inbox_status must replace intake_status, got %v", got)
	}

	fallbackArgs := make([]any, 0)
	fallbackJoins := issueLegacyRelationJoins(map[string]string{
		"intake_status": "1",
		"inbox_status":  "null",
	}, &fallbackArgs)
	if len(fallbackJoins) != 1 || len(fallbackArgs) != 1 {
		t.Fatalf("empty inbox_status must preserve intake filter: joins=%v args=%v", fallbackJoins, fallbackArgs)
	}
	if got := fallbackArgs[0].([]string); len(got) != 1 || got[0] != "1" {
		t.Fatalf("empty inbox_status replaced intake_status: %v", got)
	}

	richQuery, _, err := issueListQuery(
		uuid.New(),
		"alpha",
		uuid.New(),
		uuid.New(),
		IssueListOptions{ComplexFilter: `{"and":[{"label_id__in":"` + labelA.String() + `,` + labelB.String() + `"},{"or":[{"priority":"high"},{"priority":"low"}]}]}`},
		30,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(richQuery, "JOIN issue_labels complex_join_label_id") {
		t.Fatalf("conjunctive rich relation filter must preserve join multiplicity beside nested logic: %s", richQuery)
	}

	orQuery, _, err := issueListQuery(
		uuid.New(),
		"alpha",
		uuid.New(),
		uuid.New(),
		IssueListOptions{ComplexFilter: `{"or":[{"label_id__in":"` + labelA.String() + `,` + labelB.String() + `"},{"priority":"high"}]}`},
		30,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"LEFT JOIN issue_labels complex_join_label_id",
		"complex_join_label_id.label_id::text",
	} {
		if !strings.Contains(orQuery, fragment) {
			t.Fatalf("OR rich relation filter missing %q: %s", fragment, orQuery)
		}
	}
}
func TestIssueListRejectsInvalidCursorAndGrouping(t *testing.T) {
	repository, _ := NewRepository(&databaseStub{rows: []pgx.Row{
		boolRow(true), boolRow(true), boolRow(true), boolRow(true),
	}})
	_, err := repository.Issues(context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(), IssueListOptions{Cursor: "bad"})
	if !errors.Is(err, ErrInvalidPagination) {
		t.Fatalf("cursor err=%v", err)
	}
	_, err = repository.Issues(context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(), IssueListOptions{GroupBy: "priority", SubGroupBy: "priority"})
	if !errors.Is(err, ErrInvalidGrouping) {
		t.Fatalf("grouping err=%v", err)
	}
	for _, options := range []IssueListOptions{{GroupBy: "bogus"}, {GroupBy: "priority", SubGroupBy: "bogus"}} {
		_, err = repository.Issues(context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(), options)
		var invalid *InvalidGroupingFieldError

		if !errors.As(err, &invalid) {
			t.Fatalf("options=%+v err=%T %v", options, err, err)
		}
	}
	missingRepository, _ := NewRepository(&databaseStub{rows: []pgx.Row{missingGroupedPageRow()}})
	_, err = missingRepository.groupedIssues(
		context.Background(),
		uuid.New(),
		"alpha",
		uuid.New(),
		uuid.New(),
		IssueListOptions{GroupBy: "state_id", SubGroupBy: "priority"},
		30,
		30,
		30,
		0,
		false,
	)
	if !errors.Is(err, ErrMissingGroupKey) {
		t.Fatalf("missing group key err=%v", err)
	}
}

func TestStandaloneManyToManySubgroupPreservesScalarProjectionJoinAndMultiplicity(t *testing.T) {
	for _, test := range []struct {
		group  string
		table  string
		column string
	}{
		{"labels__id", "issue_labels", "label_id"},
		{"assignees__id", "issue_assignees", "assignee_id"},
		{"issue_module__module_id", "module_issues", "module_id"},
	} {
		query, args, err := issueListQuery(
			uuid.New(), "alpha", uuid.New(), uuid.New(),
			IssueListOptions{SubGroupBy: test.group}, 1, 0,
		)
		if err != nil {
			t.Fatal(err)
		}
		joinPrefix := "LEFT JOIN " + test.table + " standalone_subgroup ON standalone_subgroup.issue_id=issue.id"
		if test.group == "labels__id" {
			joinPrefix = "LEFT JOIN issue_labels standalone_subgroup ON standalone_subgroup.issue_id = issue.id"
		}
		for _, fragment := range []string{
			joinPrefix,
			"(standalone_subgroup.issue_id IS NULL OR standalone_subgroup.deleted_at IS NULL)",
			",standalone_subgroup." + test.column + " standalone_subgroup_id",
			"'" + test.group + "',issue.standalone_subgroup_id",
			"row_number() OVER",
			"LIMIT $5 OFFSET $6",
			"(SELECT count(*) FROM filtered)",
		} {
			if !strings.Contains(query, fragment) {
				t.Errorf("group=%s query missing %q", test.group, fragment)
			}
		}
		if strings.Contains(query, "'"+groupFields[test.group]+"',COALESCE") {
			t.Errorf("group=%s retained aggregate array projection", test.group)
		}
		if strings.Contains(query, "SELECT DISTINCT issue.*") {
			t.Errorf("group=%s removed relation multiplicity", test.group)
		}
		if len(args) != 6 || args[4] != 1 || args[5] != 0 {
			t.Errorf("group=%s pagination args=%v", test.group, args)
		}
	}

	database := &databaseStub{rows: []pgx.Row{
		boolRow(true), pageRow(`[]`, 0),
	}}
	repository, _ := NewRepository(database)
	data, err := repository.Issues(
		context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{SubGroupBy: "labels__id"},
	)
	if err != nil {
		t.Fatal(err)
	}
	var envelope map[string]any
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope["grouped_by"] != nil || envelope["sub_grouped_by"] != nil {
		t.Fatalf("standalone subgroup changed envelope grouping: %s", data)
	}
}

func TestDynamicGroupingFieldsPreserveDjangoSourceFailure(t *testing.T) {
	for _, group := range []string{"created_by", "start_date", "target_date"} {
		database := &databaseStub{rows: []pgx.Row{boolRow(true)}}
		repository, _ := NewRepository(database)
		_, err := repository.Issues(
			context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(),
			IssueListOptions{GroupBy: group},
		)
		if !errors.Is(err, ErrDynamicGrouping) {
			t.Fatalf("group=%s err=%v", group, err)
		}
		if len(database.queries) != 1 {
			t.Fatalf("group=%s queries=%d", group, len(database.queries))
		}
	}
}

func TestStandaloneSubgroupUsesDistinctTotalsAndExpandedNextPage(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{
		boolRow(true), pageRowWithExpanded(`[{"labels__id":"00000000-0000-4000-8000-000000000001"}]`, 1, 2),
	}}
	repository, _ := NewRepository(database)
	data, err := repository.Issues(
		context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{
			SubGroupBy: "labels__id", PerPage: 1, PerPageSet: true,
			Cursor: "1:0:1", CursorSet: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		TotalCount      int  `json:"total_count"`
		TotalResults    int  `json:"total_results"`
		TotalPages      int  `json:"total_pages"`
		NextPageResults bool `json:"next_page_results"`
		Count           int  `json:"count"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.TotalCount != 1 || envelope.TotalResults != 1 || envelope.TotalPages != 1 ||
		envelope.Count != 1 || !envelope.NextPageResults {
		t.Fatalf("standalone subgroup envelope=%s", data)
	}
	query := database.queries[1]
	if !strings.Contains(query, "base_total AS") ||
		!strings.Contains(query, "SELECT count(DISTINCT id)::int total FROM base") ||
		!strings.Contains(query, "(SELECT count(*) FROM filtered)") {
		t.Fatalf("standalone subgroup totals query=%s", query)
	}
}

func TestStandaloneDeletedOnlyAssigneeRelationIsExcludedAfterBaseTotal(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{
		boolRow(true), pageRowWithExpanded(`[]`, 1, 0),
	}}
	repository, _ := NewRepository(database)
	data, err := repository.Issues(
		context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{SubGroupBy: "assignees__id", PerPage: 1, PerPageSet: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		TotalCount      int   `json:"total_count"`
		TotalResults    int   `json:"total_results"`
		TotalPages      int   `json:"total_pages"`
		NextPageResults bool  `json:"next_page_results"`
		Results         []any `json:"results"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.TotalCount != 1 || envelope.TotalResults != 1 || envelope.TotalPages != 1 ||
		envelope.NextPageResults || len(envelope.Results) != 0 {
		t.Fatalf("deleted-only standalone subgroup envelope=%s", data)
	}
	query := database.queries[1]
	baseTotal := strings.Index(query, "base_total AS")
	deletedFilter := strings.Index(query, "standalone_subgroup.deleted_at IS NULL")
	if baseTotal < 0 || deletedFilter < 0 || baseTotal >= deletedFilter {
		t.Fatalf("deleted relation filter must follow base total: %s", query)
	}
}
func TestCycleDistributionsAndBurndownsScopeSourceIssueQueries(t *testing.T) {
	scope := "issue.workspace_id=workspace.id AND issue.project_id=project.id"
	for name, query := range map[string]string{
		"detail distribution":    issueDistributionSQL,
		"analytics distribution": analyticsIssueDistributionSQL,
		"estimate distribution":  estimateDistributionSQL,
	} {
		if strings.Count(query, scope) < 2 || strings.Count(query, "project.archived_at IS NULL") < 2 {
			t.Errorf("%s issue scope=%d active-project scope=%d", name, strings.Count(query, scope), strings.Count(query, "project.archived_at IS NULL"))
		}
	}
	completedScope := "completed_issue.workspace_id=workspace.id AND completed_issue.project_id=project.id"
	if !strings.Contains(issueBurndownSQL, completedScope+" AND project.archived_at IS NULL") ||
		strings.Contains(issueBurndownSQL, "total_issue.workspace_id=workspace.id") ||
		strings.Contains(issueBurndownSQL, "count(DISTINCT completed_issue.id)") {
		t.Fatal("issue burndown must scope active-project completions, preserve bridge multiplicity, and retain the cycle annotation baseline")
	}
	if !strings.Contains(estimateBurndownSQL, completedScope+" AND project.archived_at IS NULL") ||
		!strings.Contains(estimateBurndownSQL, "total_issue.workspace_id=workspace.id AND total_issue.project_id=project.id AND project.archived_at IS NULL") {
		t.Fatal("estimate burndown must scope both totals and completions through active projects")
	}
	if !strings.Contains(archivedCycleDetailSQL, "'sub_issues',(SELECT count(*) FROM cycle_issues ci JOIN issues issue ON issue.id=ci.issue_id AND "+scope) {
		t.Fatal("archived subissue lookup is not project/workspace scoped")
	}
}

func TestProgressAndAnalyticsQueriesContainAggregatesAndBurndown(t *testing.T) {
	analytics := `{"assignees":[],"labels":[],"completion_chart":{}}`
	progress := `{"backlog_estimate_points":0,"total_estimate_points":0,"backlog_issues":0,"total_issues":0}`
	database := &databaseStub{rows: []pgx.Row{
		boolRow(true), analyticsRow(true, analytics), boolRow(true), jsonRow(progress),
	}}
	repository, _ := NewRepository(database)
	userID, projectID, cycleID := uuid.New(), uuid.New(), uuid.New()
	if _, err := repository.Analytics(context.Background(), userID, "alpha", projectID, cycleID, "points"); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Progress(context.Background(), userID, "alpha", projectID, cycleID); err != nil {
		t.Fatal(err)
	}
	analyticsQuery := database.queries[1]
	for _, fragment := range []string{
		"generate_series", "json_object_agg", "sum(estimate_point.value::float8)", "completed_issue.completed_at", "requester_user.user_timezone", "$5='points'",
		"estimate.id=project.estimate_id", "count(assignee.assignee_id) total_issues", "count(issue_label.label_id) total_issues",
	} {
		if !strings.Contains(analyticsQuery, fragment) {
			t.Errorf("analytics query missing %q", fragment)
		}
	}
	if strings.Contains(analyticsIssueDistributionSQL, "first_name") || strings.Contains(analyticsIssueDistributionSQL, "last_name") {
		t.Error("live issue analytics must use the dedicated public distribution shape")
	}
	progressQuery := database.queries[3]
	for _, fragment := range []string{"backlog_estimate_points", "state.\"group\"='completed'", "progress_snapshot", "count(issue.id)", "AND project.archived_at IS NULL"} {
		if !strings.Contains(progressQuery, fragment) {
			t.Errorf("progress query missing %q", fragment)
		}
	}
}

func TestPaginationDistinguishesOmittedEmptyZeroNegativeAndFloatValues(t *testing.T) {
	limit, cursorLimit, windowLimit, page, previous, err := pagination(0, false, "", false)
	if err != nil || limit != 1000 || cursorLimit != 1000 || windowLimit != 1000 || page != 0 || previous {
		t.Fatalf("omitted pagination=(%v,%v,%v,%v,%v,%v)", limit, cursorLimit, windowLimit, page, previous, err)
	}
	limit, cursorLimit, windowLimit, page, previous, err = pagination(0, true, "", false)
	if err != nil || limit != 0 || cursorLimit != 0 || windowLimit != 0 || page != 0 || previous {
		t.Fatalf("explicit zero pagination=(%v,%v,%v,%v,%v,%v)", limit, cursorLimit, windowLimit, page, previous, err)
	}
	if _, _, _, _, _, err := pagination(25, true, "", true); !errors.Is(err, ErrInvalidPagination) {
		t.Fatalf("empty cursor err=%v", err)
	}
	limit, cursorLimit, windowLimit, page, previous, err = pagination(25, true, "0:1:0", true)
	if err != nil || limit != 25 || cursorLimit != 0 || windowLimit != 25 || page != 1 || previous {
		t.Fatalf("zero cursor value pagination=(%v,%v,%v,%v,%v,%v)", limit, cursorLimit, windowLimit, page, previous, err)
	}
	limit, cursorLimit, windowLimit, page, previous, err = pagination(25, true, "1.5:0:0", true)
	if err != nil || limit != 25 || cursorLimit != 1.5 || windowLimit != 1.5 || page != 0 || previous {
		t.Fatalf("float cursor value pagination=(%v,%v,%v,%v,%v,%v)", limit, cursorLimit, windowLimit, page, previous, err)
	}
	limit, cursorLimit, windowLimit, page, previous, err = pagination(-1, true, "", false)
	if err != nil || limit != -1 || cursorLimit != -1 || windowLimit != -1 || page != 0 || previous {
		t.Fatalf("negative per_page pagination=(%v,%v,%v,%v,%v,%v)", limit, cursorLimit, windowLimit, page, previous, err)
	}
	_, args, err := groupedIssueListQuery(
		uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{GroupBy: "priority", Cursor: "0:1:0"}, 0, 25, 1,
	)
	if err != nil || args[len(args)-2] != float64(0) || args[len(args)-1] != float64(26) {
		t.Fatalf("zero cursor grouped bounds=%v err=%v", args, err)
	}
	_, args, err = groupedIssueListQuery(
		uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{GroupBy: "priority", Cursor: "1.5:1:0"}, 1.5, 1.5, 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if args[len(args)-2] != float64(1) || args[len(args)-1] != float64(4) {
		t.Fatalf("fractional grouped cursor bounds=%v, want complete bounds 1 and 4", args)
	}
	_, args, err = groupedIssueListQuery(
		uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{GroupBy: "priority", Cursor: "1.2:0:0"}, 1.2, 1.2, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if args[len(args)-2] != float64(0) || args[len(args)-1] != float64(3) {
		t.Fatalf("fractional grouped cursor upper bound=%v, want ceiling bound 3", args)
	}
	_, _, _, _, previous, err = pagination(25, true, "1:0:2", true)
	if err != nil || !previous {
		t.Fatalf("truthy is_prev previous=%v err=%v", previous, err)
	}
	limit, cursorLimit, windowLimit, page, previous, err = pagination(25, true, "1:-1:0", true)
	if err != nil || limit != 25 || cursorLimit != 1 || windowLimit != 1 || page != -1 || previous {
		t.Fatalf("negative page parsing=(%v,%v,%v,%v,%v,%v)", limit, cursorLimit, windowLimit, page, previous, err)
	}
	limit, cursorLimit, windowLimit, page, previous, err = pagination(25, true, "-1:1:0", true)
	if err != nil || limit != 25 || cursorLimit != -1 || windowLimit != -1 || page != 1 || previous {
		t.Fatalf("negative cursor value pagination=(%v,%v,%v,%v,%v,%v)", limit, cursorLimit, windowLimit, page, previous, err)
	}
	limit, cursorLimit, windowLimit, page, previous, err = pagination(
		25, true, "\u00a02_٥\u00a0:\u00a0٠\u00a0:٩_٩_٩_٩_٩_٩_٩_٩_٩_٩_٩_٩_٩_٩_٩_٩_٩_٩_٩_٩", true,
	)
	if err != nil || limit != 25 || cursorLimit != 25 || windowLimit != 25 || page != 0 || !previous {
		t.Fatalf("Python integer cursor parsing=(%v,%v,%v,%v,%v,%v)", limit, cursorLimit, windowLimit, page, previous, err)
	}
	limit, cursorLimit, windowLimit, page, previous, err = pagination(25, true, "1.e309:0:0", true)
	if err != nil || limit != 25 || !math.IsInf(cursorLimit, 1) || !math.IsInf(windowLimit, 1) || page != 0 || previous {
		t.Fatalf("overflowing float cursor parsing=(%v,%v,%v,%v,%v,%v)", limit, cursorLimit, windowLimit, page, previous, err)
	}
}

func TestUngroupedOffsetOverflowUsesDjangoHTML500Sentinel(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	if _, ok := checkedIntProduct(maxInt, 3); ok {
		t.Fatal("checked product accepted overflowing native int multiplication")
	}
	database := &databaseStub{rows: []pgx.Row{boolRow(true)}}
	repository, _ := NewRepository(database)
	_, err := repository.Issues(
		context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{
			PerPage: 3, PerPageSet: true,
			Cursor: "3:" + strconv.Itoa(maxInt) + ":0", CursorSet: true,
		},
	)
	if !errors.Is(err, ErrPaginationOverflow) {
		t.Fatalf("overflow err=%v", err)
	}
	if len(database.queries) != 1 {
		t.Fatalf("overflow queried issues: %d queries", len(database.queries))
	}
}

func TestGroupedHugeFloatBoundsRemainRepresentable(t *testing.T) {
	_, args, err := groupedIssueListQuery(
		uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{GroupBy: "priority", Cursor: "10000000000000000000.0:1:0"},
		1e19, 1e19, 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	lower, lowerOK := args[len(args)-2].(float64)
	upper, upperOK := args[len(args)-1].(float64)
	if !lowerOK || !upperOK || lower != 1e19 || upper != 2e19 {
		t.Fatalf("huge grouped bounds=(%T %v,%T %v)", args[len(args)-2], args[len(args)-2], args[len(args)-1], args[len(args)-1])
	}
	if lower <= float64(int64(^uint64(0)>>1)) {
		t.Fatalf("test lower bound %v does not exercise int64 overflow", lower)
	}
}

func TestNegativeCursorValueSucceedsUngroupedAndFailsGrouped(t *testing.T) {
	ungroupedDatabase := &databaseStub{rows: []pgx.Row{
		boolRow(true), pageRow(`[]`, 0),
	}}
	ungroupedRepository, _ := NewRepository(ungroupedDatabase)
	_, err := ungroupedRepository.Issues(
		context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{PerPage: 25, PerPageSet: true, Cursor: "-1:1:0", CursorSet: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(ungroupedDatabase.args) != 2 ||
		ungroupedDatabase.args[1][4] != 25 ||
		ungroupedDatabase.args[1][5] != 25 {
		t.Fatalf("ungrouped cursor must use request per_page: %v", ungroupedDatabase.args)
	}

	groupedDatabase := &databaseStub{rows: []pgx.Row{boolRow(true)}}
	groupedRepository, _ := NewRepository(groupedDatabase)
	_, err = groupedRepository.Issues(
		context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{GroupBy: "priority", PerPage: 25, PerPageSet: true, Cursor: "-1:1:0", CursorSet: true},
	)
	if !errors.Is(err, ErrPaginationParsing) {
		t.Fatalf("grouped negative cursor err=%v", err)
	}
	if len(groupedDatabase.queries) != 1 {
		t.Fatalf("grouped negative cursor queried issues: %d queries", len(groupedDatabase.queries))
	}
}

func TestGroupedSignedCursorValidatesComputedOffset(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{
		boolRow(true), groupedPageRow(`{}`, 0, 0, 0, false),
	}}
	repository, _ := NewRepository(database)
	data, err := repository.Issues(
		context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{
			GroupBy: "priority", PerPage: 25, PerPageSet: true,
			Cursor: "-1:-1:0", CursorSet: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(database.args) != 2 ||
		database.args[1][4] != float64(1) ||
		database.args[1][5] != float64(1) {
		t.Fatalf("grouped signed cursor bounds=%v", database.args)
	}
	var envelope struct {
		Results map[string]any `json:"results"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Results) != 0 {
		t.Fatalf("grouped signed cursor results=%s", data)
	}
}

func TestUngroupedPreviousCursorWithDifferentValueUsesDjangoHTML500Sentinel(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true)}}
	repository, _ := NewRepository(database)
	_, err := repository.Issues(
		context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{
			PerPage: 25, PerPageSet: true,
			Cursor: "1:1:1", CursorSet: true,
		},
	)
	if !errors.Is(err, ErrPreviousPageSlice) {
		t.Fatalf("ungrouped previous cursor err=%v", err)
	}
	if len(database.queries) != 1 {
		t.Fatalf("previous cursor queried issues: %d queries", len(database.queries))
	}
}

func TestArbitraryPreviousIntegerAndInfiniteCursorServeUngroupedFirstPage(t *testing.T) {
	for _, cursor := range []string{
		"1000:0:99999999999999999999",
		"1.e309:0:0",
	} {
		database := &databaseStub{rows: []pgx.Row{
			boolRow(true), pageRow(`[]`, 0),
		}}
		repository, _ := NewRepository(database)
		_, err := repository.Issues(
			context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(),
			IssueListOptions{PerPage: 1000, PerPageSet: true, Cursor: cursor, CursorSet: true},
		)
		if err != nil {
			t.Fatalf("cursor=%q err=%v", cursor, err)
		}
		if len(database.queries) != 2 {
			t.Fatalf("cursor=%q queries=%d", cursor, len(database.queries))
		}
	}
}
func TestNegativePerPageMatchesUngroupedFailureAndGroupedEmptyPage(t *testing.T) {
	ungroupedDatabase := &databaseStub{rows: []pgx.Row{boolRow(true)}}
	ungroupedRepository, _ := NewRepository(ungroupedDatabase)
	_, err := ungroupedRepository.Issues(
		context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{PerPage: -1, PerPageSet: true},
	)
	if !errors.Is(err, ErrNegativePageSize) {
		t.Fatalf("ungrouped negative per_page err=%v", err)
	}
	if len(ungroupedDatabase.queries) != 1 {
		t.Fatalf("ungrouped negative per_page queried issues: %d queries", len(ungroupedDatabase.queries))
	}

	signedDatabase := &databaseStub{rows: []pgx.Row{boolRow(true)}}
	signedRepository, _ := NewRepository(signedDatabase)
	_, err = signedRepository.Issues(
		context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{
			PerPage: -1, PerPageSet: true,
			Cursor: "-1:1:0", CursorSet: true,
		},
	)
	if !errors.Is(err, ErrPaginationParsing) {
		t.Fatalf("ungrouped negative computed offset err=%v", err)
	}
	if len(signedDatabase.queries) != 1 {
		t.Fatalf("ungrouped negative computed offset queried issues: %d queries", len(signedDatabase.queries))
	}

	groupedDatabase := &databaseStub{rows: []pgx.Row{
		boolRow(true), groupedPageRow(`{}`, 4, 0, 0, false),
	}}
	groupedRepository, _ := NewRepository(groupedDatabase)
	data, err := groupedRepository.Issues(
		context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{GroupBy: "priority", PerPage: -1, PerPageSet: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Results    map[string]any `json:"results"`
		TotalPages int            `json:"total_pages"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Results) != 0 || envelope.TotalPages != 0 {
		t.Fatalf("grouped negative per_page envelope=%s", data)
	}
	if len(groupedDatabase.args) != 2 ||
		groupedDatabase.args[1][4] != float64(0) ||
		groupedDatabase.args[1][5] != float64(0) {
		t.Fatalf("grouped negative per_page bounds=%v", groupedDatabase.args)
	}
}

func TestZeroPerPageMatchesUngroupedFailureAndGroupedEmptyPages(t *testing.T) {
	ungroupedDatabase := &databaseStub{rows: []pgx.Row{boolRow(true)}}
	ungroupedRepository, _ := NewRepository(ungroupedDatabase)
	_, err := ungroupedRepository.Issues(
		context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{PerPage: 0, PerPageSet: true},
	)
	if !errors.Is(err, ErrZeroPageSize) {
		t.Fatalf("ungrouped zero per_page err=%v", err)
	}
	if len(ungroupedDatabase.queries) != 1 {
		t.Fatalf("ungrouped zero per_page queried issues: %d queries", len(ungroupedDatabase.queries))
	}

	for _, options := range []IssueListOptions{
		{GroupBy: "priority", PerPage: 0, PerPageSet: true},
		{GroupBy: "priority", SubGroupBy: "state_id", PerPage: 0, PerPageSet: true},
	} {
		database := &databaseStub{rows: []pgx.Row{
			boolRow(true), groupedPageRow(`{}`, 4, 0, 0, false),
		}}
		repository, _ := NewRepository(database)
		data, err := repository.Issues(
			context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(), options,
		)
		if err != nil {
			t.Fatalf("options=%+v err=%v", options, err)
		}
		var envelope struct {
			Results    map[string]any `json:"results"`
			TotalPages int            `json:"total_pages"`
		}
		if err := json.Unmarshal(data, &envelope); err != nil {
			t.Fatal(err)
		}
		if len(envelope.Results) != 0 || envelope.TotalPages != 0 {
			t.Fatalf("options=%+v envelope=%s", options, data)
		}
		if len(database.args) != 2 ||
			database.args[1][4] != float64(0) ||
			database.args[1][5] != float64(1) {
			t.Fatalf("options=%+v bounds=%v", options, database.args)
		}
	}

	selectedDatabase := &databaseStub{rows: []pgx.Row{
		boolRow(true), groupedPageRow(`{"high":{"results":[]}}`, 4, 1, 0, false),
	}}
	selectedRepository, _ := NewRepository(selectedDatabase)
	_, err = selectedRepository.Issues(
		context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(),
		IssueListOptions{
			GroupBy: "priority", PerPage: 0, PerPageSet: true,
			Cursor: "1:0:0", CursorSet: true,
		},
	)
	if !errors.Is(err, ErrZeroPageSize) {
		t.Fatalf("grouped zero per_page with selected rows err=%v", err)
	}
}

func TestAnalyticsPassesExplicitEmptyTypeToShapeDispatch(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{
		boolRow(true), analyticsRow(true, `{"assignees":[],"labels":[],"completion_chart":{}}`),
	}}
	repository, _ := NewRepository(database)
	data, err := repository.Analytics(context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(), "")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"assignees":[],"labels":[],"completion_chart":{}}` {
		t.Fatalf("analytics=%s", data)
	}
	if len(database.args) != 2 || len(database.args[1]) != 5 || database.args[1][4] != "" {
		t.Fatalf("analytics args=%v", database.args)
	}
}

func TestAnalyticsMapsMissingCycleAndDates(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{
		boolRow(true), rowFunc(func(...any) error { return pgx.ErrNoRows }),
		boolRow(true), analyticsRow(false, `{}`),
	}}
	repository, _ := NewRepository(database)
	_, err := repository.Analytics(context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(), "issues")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing err=%v", err)
	}
	_, err = repository.Analytics(context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(), "issues")
	if !errors.Is(err, ErrCycleDatesMissing) {
		t.Fatalf("dates err=%v", err)
	}
}
