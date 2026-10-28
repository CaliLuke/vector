package intakeinboxreads

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"uuid"

	"github.com/jackc/pgx/v5"
)

type repositoryRow func(...any) error

func (row repositoryRow) Scan(destinations ...any) error { return row(destinations...) }

type repositoryDatabaseStub struct {
	queries []string
	args    [][]any
	rows    []pgx.Row
}

func (database *repositoryDatabaseStub) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	database.queries = append(database.queries, query)
	database.args = append(database.args, args)
	if len(database.rows) == 0 {
		return repositoryRow(func(...any) error { return errors.New("unexpected query") })
	}
	row := database.rows[0]
	database.rows = database.rows[1:]
	return row
}

func repositoryBoolRow(value bool) pgx.Row {
	return repositoryRow(func(destinations ...any) error {
		*destinations[0].(*bool) = value
		return nil
	})
}

func repositoryJSONRow(value string) pgx.Row {
	return repositoryRow(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(value)
		return nil
	})
}

func repositoryIntakeContextRow(id uuid.UUID, guestRestricted bool) pgx.Row {
	return repositoryRow(func(destinations ...any) error {
		*destinations[0].(*uuid.UUID) = id
		*destinations[1].(*bool) = guestRestricted
		return nil
	})
}

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) returned nil error")
	}
}

func TestDraftIssueListScopesCreatorTenantFiltersAndProjection(t *testing.T) {
	stateID := uuid.New()
	database := &repositoryDatabaseStub{rows: []pgx.Row{repositoryBoolRow(true), repositoryJSONRow(`{"total_count":1,"results":[]}`)}}
	repository, _ := NewRepository(database)
	result, err := repository.DraftIssueList(context.Background(), uuid.New(), "demo", ListOptions{
		PerPage: "25", Cursor: "25:2:0", Filters: map[string]string{"state": stateID.String(), "priority": "high,urgent", "name": "login"},
	})
	if err != nil || string(result) != `{"total_count":1,"results":[]}` {
		t.Fatalf("result=%s err=%v", result, err)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"draft.created_by_id = $1", "workspace.slug = $2", "draft.deleted_at IS NULL", "workspace.deleted_at IS NULL",
		"draft.state_id = ANY", "draft.priority = ANY", "draft.name ILIKE", "ORDER BY draft.created_at DESC",
		"draft_issue_cycles", "draft_issue_modules", "draft_issue_labels", "draft_issue_assignees",
		"'description_html', draft.description_html", "'module_ids'", "'label_ids'", "'assignee_ids'",
		"'next_cursor'", "'prev_cursor'", "'total_pages'", "'extra_stats'",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("draft list query missing %q", fragment)
		}
	}
	if strings.Contains(query, "issue_mentions") || strings.Contains(query, "issue_subscribers") {
		t.Fatalf("draft list applied issue-only relations: %s", query)
	}
	if got := database.args[1][len(database.args[1])-2:]; got[0] != 25 || got[1] != 50 {
		t.Fatalf("pagination args=%#v", got)
	}
}

func TestDraftIssueListPreservesIncumbentBrokenSharedFilters(t *testing.T) {
	validID := uuid.New().String()
	tests := []ListOptions{
		{Filters: map[string]string{}, PresentFilters: map[string]bool{"labels": true}},
		{Filters: map[string]string{"assignees": validID}},
		{Filters: map[string]string{"mentions": validID}},
		{Filters: map[string]string{"logged_by": "None"}},
		{Filters: map[string]string{"intake_status": "1"}},
	}
	for _, options := range tests {
		database := &repositoryDatabaseStub{rows: []pgx.Row{repositoryBoolRow(true)}}
		repository, _ := NewRepository(database)
		if _, err := repository.DraftIssueList(context.Background(), uuid.New(), "demo", options); !errors.Is(err, ErrPaginationRuntime) {
			t.Errorf("options=%+v err=%v", options, err)
		}
		if len(database.queries) != 1 {
			t.Errorf("broken draft filter reached data query: %v", database.queries)
		}
	}
}

func TestDraftIssueDetailReturnsAuthorizedOwnersDraft(t *testing.T) {
	database := &repositoryDatabaseStub{rows: []pgx.Row{repositoryBoolRow(true), repositoryJSONRow(`{"id":"` + uuid.New().String() + `"}`)}}
	repository, _ := NewRepository(database)
	userID, draftID := uuid.New(), uuid.New()
	_, err := repository.DraftIssueDetail(context.Background(), userID, "demo", draftID)
	if err != nil {
		t.Fatal(err)
	}
	if len(database.queries) != 2 {
		t.Fatalf("draft detail queries=%v", database.queries)
	}

	if !strings.Contains(database.queries[1], "draft.created_by_id = $1") {
		t.Fatalf("draft lookup is not owner-scoped: %s", database.queries[1])
	}
}

func TestDraftIssueDetailDeniesAccessBeforeReadingContent(t *testing.T) {
	database := &repositoryDatabaseStub{rows: []pgx.Row{repositoryBoolRow(false)}}
	repository, _ := NewRepository(database)
	_, err := repository.DraftIssueDetail(context.Background(), uuid.New(), "demo", uuid.New())
	if !errors.Is(err, ErrForbidden) || len(database.queries) != 1 {
		t.Fatalf("err=%v queries=%d", err, len(database.queries))
	}
}

func TestDraftIssueDetailHandlesDeletionAfterAuthorization(t *testing.T) {
	database := &repositoryDatabaseStub{rows: []pgx.Row{
		repositoryBoolRow(true),
		repositoryRow(func(...any) error { return pgx.ErrNoRows }),
	}}
	repository, _ := NewRepository(database)
	_, err := repository.DraftIssueDetail(context.Background(), uuid.New(), "demo", uuid.New())
	if !errors.Is(err, ErrNotFound) || len(database.queries) != 2 {
		t.Fatalf("err=%v queries=%d", err, len(database.queries))
	}
	if !strings.Contains(database.queries[1], "draft.created_by_id = $1") {
		t.Fatalf("draft detail lookup is not owner-only: %s", database.queries[1])
	}
}

func TestIssueListAliasesPreserveEnvelopeFilteringOrderingAndGuestRestriction(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	intakeID := uuid.New()
	labelID := uuid.New()
	options := ListOptions{
		PerPage: "20", Cursor: "20:1:0", OrderBy: "issue__priority",
		Filters: map[string]string{"status": "-2,1", "priority": "high", "labels": labelID.String(), "sub_issue": "false"},
	}
	invoke := func(inbox bool) (*repositoryDatabaseStub, string) {
		database := &repositoryDatabaseStub{rows: []pgx.Row{repositoryBoolRow(true), repositoryIntakeContextRow(intakeID, true), repositoryJSONRow(`{"count":0,"results":[]}`)}}
		repository, _ := NewRepository(database)
		var result []byte
		var err error
		if inbox {
			result, err = repository.InboxIssueList(context.Background(), userID, "demo", projectID, options)
		} else {
			result, err = repository.IntakeIssueList(context.Background(), userID, "demo", projectID, options)
		}
		if err != nil || string(result) != `{"count":0,"results":[]}` {
			t.Fatalf("inbox=%v result=%s err=%v", inbox, result, err)
		}
		return database, database.queries[2]
	}
	inboxDatabase, inboxQuery := invoke(true)
	intakeDatabase, intakeQuery := invoke(false)
	if inboxQuery != intakeQuery {
		t.Fatal("inbox/intake issue aliases produced different repository queries")
	}
	for _, fragment := range []string{
		"intake_issue.created_by_id =", "issue.priority = ANY", "FROM issue_labels filtered_relation",
		"issue.parent_id IS NULL", "intake_issue.status = ANY", "ORDER BY issue.priority ASC",
		"'grouped_by'", "'total_count'", "'next_cursor'", "'results'", "'issue'", "'label_ids'",
	} {
		if !strings.Contains(inboxQuery, fragment) {
			t.Errorf("issue list query missing %q", fragment)
		}
	}
	for _, predicate := range []string{"AND issue.deleted_at IS NULL", "issue.is_draft = false"} {
		if strings.Contains(inboxQuery, predicate) {
			t.Errorf("issue list added related Issue manager predicate %q", predicate)
		}
	}
	if !strings.Contains(inboxDatabase.queries[0], "workspace.slug = $2") || !strings.Contains(intakeDatabase.queries[0], "project.id = $3") {
		t.Fatal("alias authorization is not tenant scoped")
	}
}

func TestIssueDetailAliasesUseIssueIDAndDetailedProjection(t *testing.T) {
	invoke := func(inbox bool) string {
		database := &repositoryDatabaseStub{rows: []pgx.Row{repositoryBoolRow(true), repositoryIntakeContextRow(uuid.New(), false), repositoryJSONRow(`{"id":"` + uuid.New().String() + `"}`)}}
		repository, _ := NewRepository(database)
		var err error
		if inbox {
			_, err = repository.InboxIssueDetail(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New())
		} else {
			_, err = repository.IntakeIssueDetail(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New())
		}
		if err != nil {
			t.Fatal(err)
		}
		return database.queries[2]
	}
	inboxQuery := invoke(true)
	intakeQuery := invoke(false)
	if inboxQuery != intakeQuery {
		t.Fatal("inbox/intake issue detail aliases produced different SQL")
	}
	for _, fragment := range []string{
		"intake_issue.issue_id = $3", "'duplicate_issue_detail'", "'description_html'", "'assignee_ids'", "'label_ids'",
		"'is_draft'", "intake_issue.deleted_at IS NULL",
	} {
		if !strings.Contains(inboxQuery, fragment) {
			t.Errorf("issue detail query missing %q", fragment)
		}
	}
	for _, predicate := range []string{"AND issue.deleted_at IS NULL", "issue.is_draft = false"} {
		if strings.Contains(inboxQuery, predicate) {
			t.Errorf("issue detail added related Issue manager predicate %q", predicate)
		}
	}
	if strings.Contains(inboxQuery, "'created_by', intake_issue.created_by_id") {
		t.Error("detail projection includes list-only top-level created_by")
	}
}

func TestIssueDetailMissingActiveIntakePreservesDjangoInternalFailure(t *testing.T) {
	for _, inbox := range []bool{true, false} {
		database := &repositoryDatabaseStub{rows: []pgx.Row{
			repositoryBoolRow(true),
			repositoryRow(func(...any) error { return pgx.ErrNoRows }),
		}}
		repository, _ := NewRepository(database)
		var err error
		if inbox {
			_, err = repository.InboxIssueDetail(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New())
		} else {
			_, err = repository.IntakeIssueDetail(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New())
		}
		if !errors.Is(err, ErrMissingIntake) || len(database.queries) != 2 {
			t.Fatalf("inbox=%v err=%v queries=%v", inbox, err, database.queries)
		}
	}
}

func TestIssueDetailReturnsGuestRestrictionOnlyAfterFindingRelationship(t *testing.T) {
	database := &repositoryDatabaseStub{rows: []pgx.Row{
		repositoryBoolRow(true),
		repositoryIntakeContextRow(uuid.New(), true),
		repositoryBoolRow(false),
	}}
	repository, _ := NewRepository(database)
	_, err := repository.InboxIssueDetail(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New())
	if !errors.Is(err, ErrGuestRestricted) {
		t.Fatalf("err=%v", err)
	}
	if len(database.queries) != 3 || !strings.Contains(database.queries[2], "intake_issue.created_by_id IS NOT DISTINCT FROM $4") {
		t.Fatalf("guest detail queries=%v", database.queries)
	}
}

func TestIntakeContainerAliasesPreserveListRoleAndAuthenticatedDetailBehavior(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	intakeID := uuid.New()
	list := func(inbox bool) *repositoryDatabaseStub {
		database := &repositoryDatabaseStub{rows: []pgx.Row{repositoryBoolRow(true), repositoryJSONRow(`{"name":"Inbox"}`)}}
		repository, _ := NewRepository(database)
		var err error
		if inbox {
			_, err = repository.InboxList(context.Background(), userID, "demo", projectID)
		} else {
			_, err = repository.IntakeList(context.Background(), userID, "demo", projectID)
		}
		if err != nil {
			t.Fatal(err)
		}
		return database
	}
	inboxList := list(true)
	intakeList := list(false)
	if inboxList.queries[1] != intakeList.queries[1] || !strings.Contains(inboxList.queries[1], "ORDER BY intake.name LIMIT 1") {
		t.Fatal("container list aliases do not share first-object behavior")
	}
	roles := inboxList.args[0][3].([]int)
	if len(roles) != 2 || roles[0] != 15 || roles[1] != 20 {
		t.Fatalf("container list roles=%v", roles)
	}
	detail := func(inbox bool) *repositoryDatabaseStub {
		database := &repositoryDatabaseStub{rows: []pgx.Row{repositoryJSONRow(`{"name":"Inbox"}`)}}
		repository, _ := NewRepository(database)
		var err error
		if inbox {
			_, err = repository.InboxDetail(context.Background(), userID, "demo", projectID, intakeID)
		} else {
			_, err = repository.IntakeDetail(context.Background(), userID, "demo", projectID, intakeID)
		}
		if err != nil {
			t.Fatal(err)
		}
		return database
	}
	inboxDetail := detail(true)
	intakeDetail := detail(false)
	if len(inboxDetail.queries) != 1 || inboxDetail.queries[0] != intakeDetail.queries[0] {
		t.Fatal("container detail aliases added membership reads or differ")
	}
	for _, fragment := range []string{"workspace.slug = $1", "project.id = $2", "intake.id = $3", "'pending_issue_count'", "'project_detail'", "'cover_image_url'"} {
		if !strings.Contains(inboxDetail.queries[0], fragment) {
			t.Errorf("container detail query missing %q", fragment)
		}
	}
}

func TestDescriptionVersionListAndDetailHaveDistinctProjections(t *testing.T) {
	listDatabase := &repositoryDatabaseStub{rows: []pgx.Row{repositoryBoolRow(true), repositoryBoolRow(false), repositoryJSONRow(`{"results":[]}`)}}
	listRepository, _ := NewRepository(listDatabase)
	cursor := "50:2:0"
	_, err := listRepository.DescriptionVersionList(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), &cursor)
	if err != nil {
		t.Fatal(err)
	}
	accessQuery := listDatabase.queries[1]
	if strings.Contains(accessQuery, "project.archived_at IS NULL") {
		t.Fatalf("description access excludes archived projects: %s", accessQuery)
	}
	for _, fragment := range []string{"workspace.slug = $2", "project.id = $3", "issue.id = $4", "workspace.deleted_at IS NULL", "project.deleted_at IS NULL", "issue.deleted_at IS NULL"} {
		if !strings.Contains(accessQuery, fragment) {
			t.Errorf("description access query missing %q", fragment)
		}
	}
	listQuery := listDatabase.queries[2]
	for _, field := range []string{"'last_saved_at'", "'owned_by'", "'created_at'", "'updated_at'", "'prev_cursor'", "'next_cursor'"} {
		if !strings.Contains(listQuery, field) {
			t.Errorf("description list query missing %q", field)
		}
	}
	for _, field := range []string{"'description_binary'", "'description_html'", "'description_json'", "'description_stripped'"} {
		if strings.Contains(listQuery, field) {
			t.Errorf("description list query exposes detail field %q", field)
		}
	}
	if got := listDatabase.args[2][3:]; len(got) != 3 || got[0] != 50 || got[1] != 100 || got[2] != 2 {
		t.Fatalf("description pagination args=%#v", got)
	}

	detailDatabase := &repositoryDatabaseStub{rows: []pgx.Row{repositoryBoolRow(true), repositoryBoolRow(false), repositoryJSONRow(`{"description_html":"<p>x</p>"}`)}}
	detailRepository, _ := NewRepository(detailDatabase)
	_, err = detailRepository.DescriptionVersionDetail(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"'description_binary'", "'description_html'", "'description_json'", "'description_stripped'"} {
		if !strings.Contains(detailDatabase.queries[2], field) {
			t.Errorf("description detail query missing %q", field)
		}
	}
}

func TestDescriptionVersionNegativePagePreservesCursorAndClampsQueryStart(t *testing.T) {
	database := &repositoryDatabaseStub{rows: []pgx.Row{
		repositoryBoolRow(true),
		repositoryBoolRow(false),
		repositoryJSONRow(`{"results":[]}`),
	}}
	repository, _ := NewRepository(database)
	cursor := "50:-2:0"
	if _, err := repository.DescriptionVersionList(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), &cursor); err != nil {
		t.Fatal(err)
	}
	args := database.args[2]
	if len(args) != 6 || args[3] != 50 || args[4] != 0 || args[5] != -2 {
		t.Fatalf("description negative-page args=%#v", args)
	}
	for _, fragment := range []string{"$6::numeric - 1", "$6::numeric + 1", "'cursor', concat($4::int, ':', $6::numeric"} {
		if !strings.Contains(database.queries[2], fragment) {
			t.Errorf("description cursor SQL missing %q", fragment)
		}
	}

	hugeDatabase := &repositoryDatabaseStub{rows: []pgx.Row{
		repositoryBoolRow(true),
		repositoryBoolRow(false),
		repositoryJSONRow(`{"cursor":"1000:-999999999999999999999999:0","results":[]}`),
	}}
	hugeRepository, _ := NewRepository(hugeDatabase)
	hugeCursor := "1000:-999999999999999999999999:0"
	if _, err := hugeRepository.DescriptionVersionList(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), &hugeCursor); err != nil {
		t.Fatal(err)
	}
	hugeArgs := hugeDatabase.args[2]
	if hugeArgs[4] != 0 || hugeArgs[5] != "-999999999999999999999999" {
		t.Fatalf("description huge negative-page args=%#v", hugeArgs)
	}

	hugePositiveDatabase := &repositoryDatabaseStub{rows: []pgx.Row{
		repositoryBoolRow(true),
		repositoryBoolRow(false),
		repositoryJSONRow(`{"cursor":"1000:999999999999999999999999:0","results":[]}`),
	}}
	hugePositiveRepository, _ := NewRepository(hugePositiveDatabase)
	hugePositiveCursor := "1000:999999999999999999999999:0"
	if _, err := hugePositiveRepository.DescriptionVersionList(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), &hugePositiveCursor); err != nil {
		t.Fatal(err)
	}
	hugePositiveArgs := hugePositiveDatabase.args[2]
	if hugePositiveArgs[4] != int(^uint(0)>>1) || hugePositiveArgs[5] != "999999999999999999999999" {
		t.Fatalf("description huge positive-page args=%#v", hugePositiveArgs)
	}
}

func TestIssueDateFiltersImplementRelativeDatesAndUserTimezone(t *testing.T) {
	database := &repositoryDatabaseStub{rows: []pgx.Row{repositoryBoolRow(true), repositoryJSONRow(`{"total_count":0,"results":[]}`)}}
	repository, _ := NewRepository(database)
	_, err := repository.DraftIssueList(context.Background(), uuid.New(), "demo", ListOptions{
		Timezone: "America/New_York",
		Filters: map[string]string{
			"created_at":   "1_weeks;before;fromnow",
			"updated_at":   "2_months;after;beforetoday",
			"completed_at": "2026-01-02;after",
			"start_date":   "2026-01",
			"target_date":  "2026-02-03;before",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"timezone(", ", draft.created_at)::date <=", ", draft.updated_at)::date >=",
		", draft.completed_at)::date >= ", "(draft.start_date)::text LIKE", "draft.target_date <= ",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("date-filter query missing %q: %s", fragment, query)
		}
	}
	if strings.Contains(query, ", draft.start_date)") || strings.Contains(query, ", draft.target_date)") {
		t.Fatalf("date-only filters were timezone shifted: %s", query)
	}
	args := database.args[1]
	for _, want := range []any{"America/New_York", "2026-01-02", "2026-01", "2026-02-03"} {
		if !repositoryArgsContain(args, want) {
			t.Errorf("date-filter args missing %#v: %#v", want, args)
		}
	}
	for _, forbidden := range []any{"1_weeks", "2_months", 7, -60} {
		if repositoryArgsContain(args, forbidden) {
			t.Errorf("relative date token or duration reached PostgreSQL: %#v", args)
		}
	}
}

func TestRelativeDateFilterUsesDjangoWeekAndMonthBoundaries(t *testing.T) {
	now := time.Date(2026, time.January, 15, 23, 59, 0, 0, time.UTC)
	builder := newWhereBuilder(uuid.New(), "demo")
	if err := addDateFilter(builder, "issue.created_at", "1_weeks;before;fromnow", "America/New_York", true, now); err != nil {
		t.Fatal(err)
	}
	if err := addDateFilter(builder, "issue.updated_at", "2_months;after;beforetoday", "America/New_York", true, now); err != nil {
		t.Fatal(err)
	}
	for _, want := range []any{"2026-01-22", "2025-11-16"} {
		if !repositoryArgsContain(builder.args, want) {
			t.Errorf("relative boundary missing %#v: %#v", want, builder.args)
		}
	}
	if len(builder.conditions) != 2 {
		t.Fatalf("relative conditions=%v", builder.conditions)
	}
}

func TestListFilterErrorsPrecedePaginationErrors(t *testing.T) {
	draftDatabase := &repositoryDatabaseStub{rows: []pgx.Row{repositoryBoolRow(true)}}
	draftRepository, _ := NewRepository(draftDatabase)
	_, err := draftRepository.DraftIssueList(context.Background(), uuid.New(), "demo", ListOptions{
		PerPage: "bad", PerPageSet: true, Filters: map[string]string{"estimate_point": "bad"},
	})
	if !errors.Is(err, ErrInvalidDetail) {
		t.Fatalf("draft combined filter/pagination err=%v", err)
	}
	if len(draftDatabase.queries) != 1 {
		t.Fatalf("draft filter error reached data query: %v", draftDatabase.queries)
	}

	intakeDatabase := &repositoryDatabaseStub{rows: []pgx.Row{
		repositoryBoolRow(true),
		repositoryIntakeContextRow(uuid.New(), false),
	}}
	intakeRepository, _ := NewRepository(intakeDatabase)
	_, err = intakeRepository.IntakeIssueList(context.Background(), uuid.New(), "demo", uuid.New(), ListOptions{
		PerPage: "bad", PerPageSet: true, StatusSet: true, Filters: map[string]string{"status": "bad"},
	})
	if !errors.Is(err, ErrPaginationRuntime) {
		t.Fatalf("intake combined status/pagination err=%v", err)
	}
	if len(intakeDatabase.queries) != 2 {
		t.Fatalf("intake status error reached data query: %v", intakeDatabase.queries)
	}

	missingIntakeDatabase := &repositoryDatabaseStub{rows: []pgx.Row{
		repositoryBoolRow(true),
		repositoryRow(func(...any) error { return pgx.ErrNoRows }),
	}}
	missingIntakeRepository, _ := NewRepository(missingIntakeDatabase)
	_, err = missingIntakeRepository.IntakeIssueList(context.Background(), uuid.New(), "demo", uuid.New(), ListOptions{
		Filters: map[string]string{"logged_by": uuid.New().String()},
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing intake did not precede broken filter: %v", err)
	}
	if len(missingIntakeDatabase.queries) != 2 {
		t.Fatalf("missing intake filter reached data query: %v", missingIntakeDatabase.queries)
	}
}

func TestPaginationErrorsPreserveDjangoParameterDetail(t *testing.T) {
	if _, err := parseOffsetPagination("invalid", true, "", false); !errors.Is(err, ErrInvalidPerPage) {
		t.Fatalf("per_page err=%v", err)
	}

	if _, err := parseOffsetPagination("1001", true, "", false); !errors.Is(err, ErrPerPageTooLarge) {
		t.Fatalf("per_page overflow err=%v", err)
	}
	if _, err := parseOffsetPagination("", false, "invalid", true); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("cursor err=%v", err)
	}
	if _, err := parseOffsetPagination("0", true, "", false); !errors.Is(err, ErrPaginationRuntime) {
		t.Fatalf("zero per_page err=%v", err)
	}
	if _, err := parseOffsetPagination("", false, "1e3:0:0", true); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("exponent cursor err=%v", err)
	}
	if _, err := parseOffsetPagination("", false, "10:-1:0", true); !errors.Is(err, ErrPaginationParsing) {
		t.Fatalf("negative-page cursor err=%v", err)
	}
	if pagination, err := parseOffsetPagination("", false, "1000:2:9", true); err != nil || pagination.page != 2 {
		t.Fatalf("matching truthy is_prev cursor pagination=%+v err=%v", pagination, err)
	}
	if _, err := parseOffsetPagination("", false, "10:2:1", true); !errors.Is(err, ErrPaginationRuntime) {
		t.Fatalf("mismatched truthy is_prev cursor err=%v", err)
	}
	if pagination, err := parseOffsetPagination("", false, "1.e999:0:0", true); err != nil || pagination.page != 0 {
		t.Fatalf("overflowing Python float cursor pagination=%+v err=%v", pagination, err)
	}
	if pagination, err := parseOffsetPagination(" 2_0 ", true, " 2_0 : 2 : 9 ", true); err != nil || pagination.limit != 20 || pagination.page != 2 {
		t.Fatalf("whitespace pagination=%+v err=%v", pagination, err)
	}
	if _, err := parseOffsetPagination("999999999999999999999999", true, "", false); !errors.Is(err, ErrPerPageTooLarge) {
		t.Fatalf("huge positive per_page err=%v", err)
	}
	if _, err := parseOffsetPagination("-999999999999999999999999", true, "", false); !errors.Is(err, ErrPaginationRuntime) {
		t.Fatalf("huge negative per_page err=%v", err)
	}
	if pagination, err := parseOffsetPagination("١", true, "١:٠:٠", true); err != nil || pagination.limit != 1 || pagination.page != 0 {
		t.Fatalf("Unicode cursor pagination=%+v err=%v", pagination, err)
	}
	if pagination, err := parseOffsetPagination("", false, "٩٩٩٩٩٩٩٩٩٩٩٩٩٩٩٩٩٩٩٩٩٩:٠:٠", true); err != nil || pagination.page != 0 {
		t.Fatalf("arbitrary ignored first cursor component pagination=%+v err=%v", pagination, err)
	}
	if pagination, err := parseOffsetPagination("", false, "١٠٠٠:٠:٩٩٩٩٩٩٩٩٩٩٩٩٩٩٩٩٩٩٩٩٩٩", true); err != nil || pagination.page != 0 {
		t.Fatalf("arbitrary truthy previous cursor component pagination=%+v err=%v", pagination, err)
	}
	if _, err := parseOffsetPagination("", false, "0x1.0p0:0:0", true); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("Go-only hex float cursor err=%v", err)
	}
	if _, err := parseOffsetPagination("", false, "1000:-999999999999999999999999:0", true); !errors.Is(err, ErrPaginationParsing) {
		t.Fatalf("huge negative cursor page err=%v", err)
	}
	if _, err := parseOffsetPagination("", false, "1000:999999999999999999999999:0", true); !errors.Is(err, ErrPaginationRuntime) {
		t.Fatalf("huge positive cursor page err=%v", err)
	}
	if _, err := parseOffsetPagination("1000", true, "1000:2305843009213693952:0", true); !errors.Is(err, ErrPaginationRuntime) {
		t.Fatalf("overflowing cursor offset err=%v", err)
	}
	invalidDescriptionCursor := "invalid"
	if _, _, err := parseDescriptionCursor(&invalidDescriptionCursor); !errors.Is(err, ErrInvalidDescriptionCursor) {
		t.Fatalf("description cursor err=%v", err)
	}
	negativePageCursor := "50:-2:0"
	pageSize, page, err := parseDescriptionCursor(&negativePageCursor)
	if err != nil || pageSize != 50 || page.Int64() != -2 {
		t.Fatalf("negative description cursor page_size=%d page=%s err=%v", pageSize, page, err)
	}
	whitespaceDescriptionCursor := " 50 : -2 : 0 "
	pageSize, page, err = parseDescriptionCursor(&whitespaceDescriptionCursor)
	if err != nil || pageSize != 50 || page.Int64() != -2 {
		t.Fatalf("whitespace description cursor page_size=%d page=%s err=%v", pageSize, page, err)
	}
	hugeDescriptionPageSize := "999999999999999999999999:0:0"
	pageSize, page, err = parseDescriptionCursor(&hugeDescriptionPageSize)
	if err != nil || pageSize != 1000 || page.Sign() != 0 {
		t.Fatalf("huge description page_size=%d page=%s err=%v", pageSize, page, err)
	}
	hugeNegativeDescriptionPageSize := "-999999999999999999999999:0:0"
	if _, _, parseDescriptionCursorErr := parseDescriptionCursor(&hugeNegativeDescriptionPageSize); !errors.Is(parseDescriptionCursorErr, ErrInvalidDescriptionCursor) {
		t.Fatalf("huge negative description page size err=%v", parseDescriptionCursorErr)
	}
	hugeIgnoredComponent := "1000:0:999999999999999999999999"
	pageSize, page, err = parseDescriptionCursor(&hugeIgnoredComponent)
	if err != nil || pageSize != 1000 || page.Sign() != 0 {
		t.Fatalf("huge ignored description component page_size=%d page=%s err=%v", pageSize, page, err)
	}
	hugeNegativeDescriptionPage := "1000:-999999999999999999999999:0"
	pageSize, page, err = parseDescriptionCursor(&hugeNegativeDescriptionPage)
	if err != nil || pageSize != 1000 || page.String() != "-999999999999999999999999" {
		t.Fatalf("huge negative description page_size=%d page=%s err=%v", pageSize, page, err)
	}
}
func TestUnicodeRelativeDatesAndEmptyCSVTokensPreserveDjangoFilters(t *testing.T) {
	if days, ok := relativeDateDays("١_weeks"); !ok || days != 7 {
		t.Fatalf("Unicode relative duration days=%d ok=%v", days, ok)
	}
	if values := csvStrings("high,"); len(values) != 0 {
		t.Fatalf("priority with empty token values=%v", values)
	}
	if values := csvStrings("completed,,started"); len(values) != 0 {
		t.Fatalf("state group with empty token values=%v", values)
	}
}

func TestRelativeDateFilterWithoutOffsetMatchesDjangoNoOp(t *testing.T) {
	builder := newWhereBuilder(uuid.New(), "demo")
	if err := addDateFilter(builder, "issue.created_at", "1_weeks;after", "UTC", true, time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(builder.conditions) != 0 || len(builder.args) != 2 {
		t.Fatalf("relative date without offset conditions=%v args=%v", builder.conditions, builder.args)
	}
}

func TestDjangoAbsoluteDateSpellingsCanonicalizeSQLBoundaries(t *testing.T) {
	for value, expected := range map[string]string{
		"2026-1-2": "2026-01-02", "20260102": "2026-01-02", "٢٠٢٦-١-٢": "2026-01-02",
		"2026-W01-1": "2025-12-29", "2026W011": "2025-12-29", "2026-W01": "2025-12-29", "2026W01": "2025-12-29",
	} {
		builder := newWhereBuilder(uuid.New(), "demo")
		if err := addDateFilter(builder, "issue.created_at", value+";after", "UTC", true, time.Now()); err != nil {
			t.Fatalf("value=%q err=%v", value, err)
		}
		if !repositoryArgsContain(builder.args, expected) {
			t.Fatalf("value=%q expected=%q args=%#v", value, expected, builder.args)
		}

	}
	for _, value := range []string{"0000-01-01", "2026-+1-+2", "2026-W54-1", "٢٠٢٦٠١٠٢", "٢٠٢٦-W٠١-١", "٢٠٢٦W٠١"} {
		if parsed, ok := parseDjangoDate(value); ok {
			t.Fatalf("invalid value=%q parsed=%q", value, parsed)
		}
	}
	if _, ok := relativeDateDays("+1_weeks"); ok {
		t.Fatal("signed relative duration was accepted")
	}
}
func TestDjangoContainsFiltersEscapeLikeMetacharacters(t *testing.T) {
	const raw = `%_\`
	const escaped = `\%\_\\`

	nameBuilder := newWhereBuilder(uuid.New(), "demo")
	if err := addIssueFilters(nameBuilder, "issue", map[string]string{"name": raw}, nil, false, "UTC"); err != nil {
		t.Fatal(err)
	}
	if nameBuilder.args[len(nameBuilder.args)-1] != escaped || !strings.Contains(nameBuilder.conditions[0], `ESCAPE '\'`) {
		t.Fatalf("name contains conditions=%v args=%#v", nameBuilder.conditions, nameBuilder.args)
	}

	dateBuilder := newWhereBuilder(uuid.New(), "demo")
	if err := addDateFilter(dateBuilder, "issue.created_at", raw, "UTC", true, time.Now()); err != nil {
		t.Fatal(err)
	}
	if dateBuilder.args[len(dateBuilder.args)-1] != escaped || !strings.Contains(dateBuilder.conditions[0], `ESCAPE '\'`) {
		t.Fatalf("date contains conditions=%v args=%#v", dateBuilder.conditions, dateBuilder.args)
	}
}

func TestDateFilterNullTokenMatchesDjangoContainsFilter(t *testing.T) {
	builder := newWhereBuilder(uuid.New(), "demo")
	if err := addDateFilter(builder, "issue.start_date", "null", "UTC", false, time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(builder.conditions) != 1 || !repositoryArgsContain(builder.args, "null") {
		t.Fatalf("date null filter conditions=%v args=%v", builder.conditions, builder.args)
	}
}

func TestStrictEstimatePointAndMalformedAbsoluteDateReturnInvalidDetail(t *testing.T) {
	for name, filters := range map[string]map[string]string{
		"estimate point": {"estimate_point": "not-a-uuid"},
		"absolute date":  {"created_at": "not-a-date;after"},
	} {
		t.Run(name, func(t *testing.T) {
			database := &repositoryDatabaseStub{rows: []pgx.Row{repositoryBoolRow(true)}}
			repository, _ := NewRepository(database)
			_, err := repository.DraftIssueList(context.Background(), uuid.New(), "demo", ListOptions{Filters: filters})
			if !errors.Is(err, ErrInvalidDetail) {
				t.Fatalf("err=%v", err)
			}
			if len(database.queries) != 1 {
				t.Fatalf("invalid filter executed data query: %v", database.queries)
			}
		})
	}
}

func TestRepeatedDateBoundsKeepOnlyLastValuePerDirection(t *testing.T) {
	builder := newWhereBuilder(uuid.New(), "demo")
	err := addDateFilter(builder, "issue.created_at", "2024-01-01;before,2025-01-01;before,2023-01-01;after,2023-06-01;after", "UTC", true, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(builder.conditions) != 2 {
		t.Fatalf("conditions=%v", builder.conditions)
	}
	for _, want := range []any{"2025-01-01", "2023-06-01"} {
		if !repositoryArgsContain(builder.args, want) {
			t.Errorf("missing last bound %q: %v", want, builder.args)
		}
	}
	for _, overwritten := range []any{"2024-01-01", "2023-01-01"} {
		if repositoryArgsContain(builder.args, overwritten) {
			t.Errorf("retained overwritten bound %q: %v", overwritten, builder.args)
		}
	}
}

func TestInboxStatusOverwritesIntakeStatus(t *testing.T) {
	builder := newWhereBuilder(uuid.New(), "demo")
	err := addIssueFilters(builder, "issue", map[string]string{"intake_status": "-2", "inbox_status": "2"}, nil, false, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if len(builder.conditions) != 1 {
		t.Fatalf("status conditions=%v", builder.conditions)
	}
	statuses, ok := builder.args[len(builder.args)-1].([]int)
	if !ok || len(statuses) != 1 || statuses[0] != 2 {
		t.Fatalf("status args=%#v", builder.args)
	}
}

func TestEmptyInboxStatusDoesNotEraseIntakeStatus(t *testing.T) {
	builder := newWhereBuilder(uuid.New(), "demo")
	err := addIssueFilters(builder, "issue", map[string]string{"intake_status": "-2", "inbox_status": "null"}, nil, false, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if len(builder.conditions) != 1 {
		t.Fatalf("status conditions=%v", builder.conditions)
	}
	statuses, ok := builder.args[len(builder.args)-1].([]int)
	if !ok || len(statuses) != 1 || statuses[0] != -2 {
		t.Fatalf("status args=%#v", builder.args)
	}
}

func TestUnrelatedModuleAndAssigneeProjectionSemanticsRemainUnchanged(t *testing.T) {
	if strings.Contains(draftIssueProjection, "module.deleted_at IS NULL") || !strings.Contains(draftIssueProjection, "module.archived_at IS NULL") {
		t.Fatalf("draft module target predicates=%s", draftIssueProjection)
	}
	if strings.Contains(draftIssueProjection, " AND member.deleted_at IS NULL") || !strings.Contains(draftIssueProjection, "member.is_active = true") {
		t.Fatalf("draft assignee membership predicates=%s", draftIssueProjection)
	}
}

func TestSoftDeletedDuplicateAndPendingRowsPreserveDjangoProjection(t *testing.T) {
	if strings.Contains(intakeIssueDetailSQL, "duplicate_issue.deleted_at IS NULL") {
		t.Fatalf("detail excludes soft-deleted duplicate target: %s", intakeIssueDetailSQL)
	}
	if strings.Contains(intakeProjection, "pending.deleted_at IS NULL") {
		t.Fatalf("pending count excludes soft-deleted intake rows: %s", intakeProjection)
	}
}

func TestTypeFilterOverwritesStateGroupWithoutExcludingSoftDeletedStateTarget(t *testing.T) {
	builder := newWhereBuilder(uuid.New(), "demo")
	if err := addIssueFilters(builder, "issue", map[string]string{"type": "active", "state_group": "completed"}, nil, false, "UTC"); err != nil {
		t.Fatal(err)
	}
	if len(builder.conditions) != 1 {
		t.Fatalf("type and state_group intersected instead of overwriting: %v", builder.conditions)
	}
	for _, condition := range builder.conditions {
		if strings.Contains(condition, "filtered_state.deleted_at IS NULL") {
			t.Fatalf("state filter excludes soft-deleted target: %s", condition)
		}
	}
	groups, ok := builder.args[len(builder.args)-1].([]string)
	if !ok || !reflect.DeepEqual(groups, []string{"unstarted", "started"}) {
		t.Fatalf("type filter groups=%#v", builder.args)
	}
}

func TestExplicitEmptyTypeAndRelationPresencePreserveDjangoFiltering(t *testing.T) {
	options := ListOptions{Filters: map[string]string{"type": ""}, PresentFilters: map[string]bool{"type": true}}
	if err := validateIntakeIssueFilters(options); err != nil {
		t.Fatal(err)
	}
	builder := newWhereBuilder(uuid.New(), "demo")
	if err := addIssueFilters(builder, "issue", options.Filters, options.PresentFilters, false, "UTC"); err != nil {
		t.Fatal(err)
	}
	if len(builder.conditions) != 1 || options.Filters["type"] != "all" {
		t.Fatalf("empty type conditions=%v filters=%v", builder.conditions, options.Filters)
	}

	builder = newWhereBuilder(uuid.New(), "demo")
	if err := addIssueFilters(builder, "issue", map[string]string{"mentions": uuid.New().String()}, nil, false, "UTC"); err != nil {
		t.Fatal(err)
	}
	if len(builder.conditions) != 1 || strings.Contains(builder.conditions[0], "deleted_at") {
		t.Fatalf("mention relation conditions=%v", builder.conditions)
	}

	builder = newWhereBuilder(uuid.New(), "demo")
	if err := addIssueFilters(builder, "issue", map[string]string{"subscriber": "None"}, map[string]bool{"subscriber": true}, false, "UTC"); err != nil {
		t.Fatal(err)
	}
	if len(builder.conditions) != 1 || !strings.Contains(builder.conditions[0], "deleted_at IS NULL") {
		t.Fatalf("subscriber None lost live through-row predicate: %v", builder.conditions)
	}
	if err := validateDraftIssueFilters(ListOptions{Filters: map[string]string{"mentions": "None"}}); err != nil {
		t.Fatalf("draft mentions None should be dropped by UUID validation: %v", err)
	}
}

func TestMalformedStatusFiltersPreserveDjangoInternalError(t *testing.T) {
	builder := newWhereBuilder(uuid.New(), "demo")
	if err := addIntakeStatuses(builder, "", true); !errors.Is(err, ErrPaginationRuntime) {
		t.Fatalf("explicit empty status err=%v", err)
	}
	if err := addIssueFilters(builder, "issue", map[string]string{"intake_status": "invalid"}, nil, false, "UTC"); !errors.Is(err, ErrPaginationRuntime) {
		t.Fatalf("invalid intake_status err=%v", err)
	}
	if strings.Contains(strings.Join(builder.conditions, " "), "filtered_intake.deleted_at") {
		t.Fatalf("intake status filters soft-delete related rows: %v", builder.conditions)
	}
	builder = newWhereBuilder(uuid.New(), "demo")
	if err := addIntakeStatuses(builder, " 1 ", true); err != nil {
		t.Fatal(err)
	}
	if statuses, ok := builder.args[len(builder.args)-1].([]int); !ok || !reflect.DeepEqual(statuses, []int{1}) {
		t.Fatalf("whitespace status args=%#v", builder.args)
	}
	builder = newWhereBuilder(uuid.New(), "demo")
	if err := addIssueFilters(builder, "issue", map[string]string{"intake_status": " 0_1 "}, nil, false, "UTC"); err != nil {
		t.Fatal(err)
	}
	if statuses, ok := builder.args[len(builder.args)-1].([]int); !ok || !reflect.DeepEqual(statuses, []int{1}) {
		t.Fatalf("whitespace intake_status args=%#v", builder.args)
	}
}

func TestProjectionOrderingAndCoverAssetURLsPreserveDjangoProperties(t *testing.T) {
	if !strings.Contains(draftIssueProjection, "ORDER BY relation.created_at DESC LIMIT 1") {
		t.Fatalf("draft cycle projection does not select newest relation: %s", draftIssueProjection)
	}
	for _, fragment := range []string{
		"WHEN asset.id IS NULL THEN NULLIF(project.cover_image, '')",
		"asset_workspace.id = asset.workspace_id",
		"COALESCE(asset.project_id::text, 'None')",
		"COALESCE(asset.issue_id::text, 'None')",
		"'ISSUE_ATTACHMENT'",
		"'DRAFT_ISSUE_DESCRIPTION'",
	} {
		if !strings.Contains(intakeProjection, fragment) {
			t.Errorf("cover URL projection missing %q", fragment)
		}
	}
	if strings.Contains(intakeProjection, "workspace.slug || '/projects/' || asset.project_id") {
		t.Fatalf("cover URL projection changed Django FileAsset property semantics: %s", intakeProjection)
	}
}

func TestIssueAndDescriptionQueriesKeepDjangoRelatedIssueManagerSemantics(t *testing.T) {
	for name, query := range map[string]string{
		"intake detail": intakeIssueDetailSQL,
		"guest access":  guestIntakeIssueAccessSQL,
	} {
		for _, predicate := range []string{"AND issue.deleted_at IS NULL", "issue.is_draft = false"} {
			if strings.Contains(query, predicate) {
				t.Errorf("%s unexpectedly contains %q", name, predicate)
			}
		}
	}
	for name, query := range map[string]string{
		"description access": descriptionIssueAccessSQL,
		"description list":   descriptionVersionListSQL,
		"description detail": descriptionVersionDetailSQL,
	} {
		if !strings.Contains(query, "AND issue.deleted_at IS NULL") {
			t.Errorf("%s lost explicit Issue.objects soft-delete predicate", name)
		}
		if strings.Contains(query, "issue.is_draft = false") {
			t.Errorf("%s unexpectedly excludes draft issues", name)
		}
	}
}

func repositoryArgsContain(args []any, want any) bool {
	for _, argument := range args {
		if argument == want {
			return true
		}
	}
	return false
}

func TestDescriptionVersionGuestRestrictionStopsBeforeVersionRead(t *testing.T) {
	database := &repositoryDatabaseStub{rows: []pgx.Row{repositoryBoolRow(true), repositoryBoolRow(true)}}
	repository, _ := NewRepository(database)
	_, err := repository.DescriptionVersionList(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), nil)
	if !errors.Is(err, ErrGuestRestricted) {
		t.Fatalf("err=%v", err)
	}
	if len(database.queries) != 2 || !strings.Contains(database.queries[1], "project.guest_view_all_features = false") {
		t.Fatalf("description guest queries=%v", database.queries)
	}
}

func TestAuthorizationDenialStopsBeforeDataRead(t *testing.T) {
	database := &repositoryDatabaseStub{rows: []pgx.Row{repositoryBoolRow(false)}}
	repository, _ := NewRepository(database)
	_, err := repository.InboxIssueList(context.Background(), uuid.New(), "other-workspace", uuid.New(), ListOptions{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err=%v", err)
	}
	if len(database.queries) != 1 {
		t.Fatalf("denied request executed %d queries", len(database.queries))
	}
	for _, fragment := range []string{"member.deleted_at IS NULL", "project.deleted_at IS NULL", "workspace.deleted_at IS NULL"} {
		if !strings.Contains(database.queries[0], fragment) {
			t.Errorf("authorization query missing %q", fragment)
		}
	}
}

func TestRepositoryBoundaryCannotWrite(t *testing.T) {
	queries := []string{workspaceAuthorizationSQL, draftIssueDetailAuthorizationSQL, projectAuthorizationSQL, projectOrCreatorAuthorizationSQL, descriptionIssueAccessSQL, guestIntakeIssueAccessSQL, intakeContextSQL, draftIssueListSQL, draftIssueDetailSQL, intakeIssueListSQL, intakeIssueDetailSQL, intakeContainerListSQL, intakeContainerDetailSQL, descriptionVersionListSQL, descriptionVersionDetailSQL}
	for _, query := range queries {
		upper := strings.ToUpper(query)
		for _, verb := range []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE "} {
			if strings.Contains(upper, verb) {
				t.Errorf("read boundary contains %s: %s", strings.TrimSpace(verb), query)
			}
		}
	}
}
