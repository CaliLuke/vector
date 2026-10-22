package cyclereads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestCycleLifecycleAuthorizationRequiresProjectMemberRole(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(false)}}
	repository, err := NewLifecycleRepository(database, lifecycleEffectsStub{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.AuthorizeProject(context.Background(), "user", "workspace", "project"); err != ErrForbidden {
		t.Fatalf("guest project member should be denied, got %v", err)
	}
	query := database.queries[0]
	for _, fragment := range []string{"project_members", "pm.role IN (15,20)", "pm.is_active=true", "w.slug=$3"} {
		if !strings.Contains(query, fragment) {
			t.Errorf("authorization query missing %q: %s", fragment, query)
		}
	}
}

func TestCycleAppWorkspaceAdminProjectGuestWriteFallback(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true)}}
	repository, err := NewLifecycleRepository(database, lifecycleEffectsStub{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.AuthorizeAppProject(context.Background(), "admin", "workspace", "project"); err != nil {
		t.Fatalf("workspace admin with active guest project membership should pass app write permission: %v", err)
	}
	query := database.queries[0]
	for _, fragment := range []string{"project_members", "pm.role IN (15,20)", "workspace_members wm", "wm.role=20", "wm.is_active=true"} {
		if !strings.Contains(query, fragment) {
			t.Errorf("app write predicate missing %q: %s", fragment, query)
		}
	}
	if strings.Contains(query, "w.deleted_at") {
		t.Errorf("source's related workspace lookup does not filter deleted workspace rows: %s", query)
	}
}

func TestCycleAppCreatorDeletePredicateKeepsWorkspaceMembershipAndDefaultManagerScope(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true)}}
	repository, err := NewLifecycleRepository(database, lifecycleEffectsStub{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.AuthorizeAppDelete(context.Background(), "workspace", "project", "cycle", "creator"); err != nil {
		t.Fatalf("active workspace member who created the cycle should pass app creator permission: %v", err)
	}
	query := database.queries[0]
	for _, fragment := range []string{"workspace_members wm", "wm.is_active=true", "wm.deleted_at IS NULL", "c.created_by_id=$1", "c.deleted_at IS NULL", "pm.role=20"} {
		if !strings.Contains(query, fragment) {
			t.Errorf("app delete predicate missing %q: %s", fragment, query)
		}
	}
	if strings.Contains(query, "w.deleted_at") {
		t.Errorf("source's related workspace lookup does not filter deleted workspace rows: %s", query)
	}
}

func TestCycleLifecycleAPIPatchDuplicateExternalIDUsesRawExternalSource(t *testing.T) {
	for _, test := range []struct {
		name          string
		beforeSource  any
		requestSource any
		sourcePresent bool
		wantSource    any
	}{
		{name: "explicit blank", beforeSource: "tracker", requestSource: "", sourcePresent: true, wantSource: ""},
		{name: "explicit null", beforeSource: "tracker", requestSource: nil, sourcePresent: true, wantSource: nil},
		{name: "omitted nonnull fallback", beforeSource: "tracker", wantSource: "tracker"},
		{name: "omitted null fallback", beforeSource: nil, wantSource: nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			database := &databaseStub{rows: []pgx.Row{boolRow(true)}}
			repository, err := NewLifecycleRepository(database, lifecycleEffectsStub{}, "")
			if err != nil {
				t.Fatal(err)
			}
			before := map[string]any{"external_id": "CYC-1", "external_source": test.beforeSource}
			requestData := map[string]any{"external_id": "CYC-2"}
			if test.sourcePresent {
				requestData["external_source"] = test.requestSource
			}
			err = repository.checkAPIExternalIDConflict(context.Background(), "project", "workspace", "cycle-current", before, requestData)
			var httpError *djangoapi.HTTPError
			if !errors.As(err, &httpError) || httpError.Status != 409 {
				t.Fatalf("conflict error=%v, want HTTP 409", err)
			}
			if !strings.Contains(database.queries[0], "external_source IS NOT DISTINCT FROM $3") {
				t.Fatalf("query does not compare NULL and blank source values using SQL null-safe equality: %s", database.queries[0])
			}
			if got := database.args[0][2]; !reflect.DeepEqual(got, test.wantSource) {
				t.Fatalf("external source query arg=%#v, want raw/fallback %#v", got, test.wantSource)
			}
			if httpError.Body["id"] != "cycle-current" {
				t.Fatalf("conflict id=%#v, want current cycle", httpError.Body["id"])
			}
		})
	}
}

func TestCycleLifecycleAPIPatchExternalIDNumericCoercionKeepsRawComparison(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true)}}
	repository, err := NewLifecycleRepository(database, lifecycleEffectsStub{}, "")
	if err != nil {
		t.Fatal(err)
	}
	err = repository.checkAPIExternalIDConflict(
		context.Background(), "project", "workspace", "cycle-current",
		map[string]any{"external_id": "1", "external_source": "tracker"},
		map[string]any{"external_id": json.Number("1")},
	)
	var httpError *djangoapi.HTTPError
	if !errors.As(err, &httpError) || httpError.Status != 409 {
		t.Fatalf("numeric raw external ID conflict=%v, want HTTP 409", err)
	}
	if got := database.args[0][3]; got != "1" {
		t.Fatalf("ORM external ID query arg=%#v, want DRF CharField string coercion", got)
	}
	if strings.Contains(database.queries[0], "id<>") {
		t.Fatalf("source existence query must include current row when raw numeric input differs from stored string: %s", database.queries[0])
	}
}

type cycleLifecycleDeleteOrderDB struct {
	base   *databaseStub
	events *[]string
}

func (database *cycleLifecycleDeleteOrderDB) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	if strings.Contains(query, "UPDATE cycles SET deleted_at") {
		*database.events = append(*database.events, "persist-cycle-delete")
	}
	return database.base.QueryRow(ctx, query, args...)
}

func (database *cycleLifecycleDeleteOrderDB) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(query, "user_favorites") {
		*database.events = append(*database.events, "cleanup-favorites")
	}
	return database.base.Exec(ctx, query, args...)
}

type cycleLifecycleDeleteOrderEffects struct{ events *[]string }

func (effects cycleLifecycleDeleteOrderEffects) Task(_ context.Context, task string, _ []any, _ map[string]any) error {
	if strings.HasSuffix(task, ".issue_activity") {
		*effects.events = append(*effects.events, "issue-activity")
	}
	return nil
}

func (effects cycleLifecycleDeleteOrderEffects) SoftDelete(context.Context, string, string, string) error {
	*effects.events = append(*effects.events, "publish-soft-delete")
	return nil
}

func TestCycleLifecycleDeletePersistsBeforeSoftDeletePublication(t *testing.T) {
	createdBy := "creator"
	rows := []pgx.Row{
		rowFunc(func(destinations ...any) error {
			*destinations[0].(*string) = "cycle"
			*destinations[1].(*string) = "Cycle"
			*destinations[2].(**string) = &createdBy
			*destinations[3].(**string) = nil
			*destinations[4].(**time.Time) = nil
			*destinations[5].(**time.Time) = nil
			return nil
		}),
		rowFunc(func(destinations ...any) error {
			*destinations[0].(*[]byte) = []byte("[]")
			return nil
		}),
		rowFunc(func(destinations ...any) error {
			*destinations[0].(*string) = "cycle"
			return nil
		}),
	}
	events := []string{}
	database := &cycleLifecycleDeleteOrderDB{base: &databaseStub{rows: rows}, events: &events}
	repository, err := NewLifecycleRepository(database, cycleLifecycleDeleteOrderEffects{events: &events}, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Delete(context.Background(), "workspace", "project", "cycle", "actor", true); err != nil {
		t.Fatal(err)
	}
	want := []string{"issue-activity", "persist-cycle-delete", "publish-soft-delete", "cleanup-favorites"}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("delete event order=%s, want %s", fmt.Sprint(events), fmt.Sprint(want))
	}
}

func TestCycleReadAuthorizationAllowsActiveProjectMembersWithoutRoleGate(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true)}}
	repository, err := NewLifecycleRepository(database, lifecycleEffectsStub{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.AuthorizeReadProject(context.Background(), "guest", "workspace", "project"); err != nil {
		t.Fatalf("active guest should pass inherited read-only PUT permission: %v", err)
	}
	query := database.queries[0]
	for _, fragment := range []string{"pm.is_active=true", "p.deleted_at IS NULL", "p.archived_at IS NULL", "w.slug=$3"} {
		if !strings.Contains(query, fragment) {
			t.Errorf("read authorization query missing %q: %s", fragment, query)
		}
	}
	if strings.Contains(query, "pm.role") {
		t.Errorf("inherited PUT must not require a write role: %s", query)
	}
}

func TestCycleMutationProjectionUsesExistingReadAggregatesAndSeparatesPUT(t *testing.T) {
	for _, fragment := range []string{"raw_counts.cancelled", "'assignee_ids'", "'is_favorite'", "'created_by'", "'version'"} {
		if !strings.Contains(projectCycleMutationDetailSQL, fragment) {
			t.Errorf("mutation projection is missing read aggregate %q", fragment)
		}
	}
	if !strings.Contains(projectCycleMutationDetailSQL, "'cancelled_issues',counts.cancelled") {
		t.Fatal("PUT serializer projection does not include cancelled issue count")
	}
	database := &databaseStub{rows: []pgx.Row{jsonRow(`{"id":"cycle","sub_issues":2,"assignee_ids":[],"version":1,"created_by":"actor","cancelled_issues":3}`), jsonRow(`{"id":"cycle","sub_issues":2,"assignee_ids":[],"version":1,"created_by":"actor","cancelled_issues":3}`)}}
	repository, err := NewLifecycleRepository(database, lifecycleEffectsStub{}, "")
	if err != nil {
		t.Fatal(err)
	}
	annotated, err := repository.appCycleMutationProjection(context.Background(), "actor", "workspace", "project", "cycle", false)
	if err != nil {
		t.Fatal(err)
	}
	put, err := repository.appCycleMutationProjection(context.Background(), "actor", "workspace", "project", "cycle", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := annotated["version"]; !ok {
		t.Errorf("create/PATCH projection omitted version: %#v", annotated)
	}
	if _, ok := annotated["cancelled_issues"]; ok {
		t.Errorf("create/PATCH projection included PUT-only cancelled count: %#v", annotated)
	}
	if _, ok := put["cancelled_issues"]; !ok {
		t.Errorf("PUT projection omitted cancelled count: %#v", put)
	}
	for _, projection := range []map[string]any{annotated, put} {
		if _, ok := projection["sub_issues"]; ok {
			t.Errorf("serializer projection leaked read-detail field: %#v", projection)
		}
	}
}

func TestCycleCurrentInstanceKeepsUTCDatesWhileAppResponseUsesProjectTimezone(t *testing.T) {
	if !strings.Contains(projectCycleMutationDetailSQL, "AT TIME ZONE project.timezone") {
		t.Fatal("app mutation response no longer serializes dates in the project timezone")
	}
	if !strings.Contains(utcTimestampSQL("c.start_date"), "'Z'") {
		t.Fatal("current-instance serializer no longer emits UTC Z timestamps")
	}
	encoded, err := pythonCycleCurrentInstance(map[string]any{
		"id": "cycle", "workspace_id": "workspace", "project_id": "project", "name": "Cycle",
		"start_date": "2099-03-08T08:00:01Z", "end_date": "2099-03-22T07:59:00Z",
		"owned_by_id": "owner", "view_props": map[string]any{}, "sort_order": float64(1),
		"external_source": nil, "external_id": nil, "progress_snapshot": map[string]any{}, "logo_props": map[string]any{},
		"is_favorite": false, "total_issues": 0, "cancelled_issues": 0, "completed_issues": 0, "status": "CURRENT",
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	var current map[string]any
	if err := json.Unmarshal([]byte(encoded), &current); err != nil {
		t.Fatal(err)
	}
	if current["start_date"] != "2099-03-08T08:00:01Z" || current["end_date"] != "2099-03-22T07:59:00Z" {
		t.Fatalf("current instance dates = %#v; expected UTC timestamps", current)
	}
}

func TestCycleTimestampProjectionNormalizesFractionalSerializerDates(t *testing.T) {
	projection := cycleTimestampJSONFields("cycles")
	for _, field := range []string{"created_at", "updated_at", "deleted_at", "archived_at", "start_date", "end_date"} {
		if !strings.Contains(projection, "'"+field+"',") {
			t.Errorf("cycle timestamp projection omits %q: %s", field, projection)
		}
	}
	updatedAt := utcTimestampSQL("cycles.updated_at")
	if !strings.Contains(updatedAt, "to_char") || !strings.Contains(updatedAt, "'US'") || !strings.Contains(updatedAt, "'Z'") {
		t.Fatalf("fractional cycle timestamp formatting changed: %s", updatedAt)
	}
	userProjection := cycleUserTimestampJSONFields("cycles")
	if !strings.Contains(userProjection, "requester_user.user_timezone") || !strings.Contains(userProjection, "'US'") {
		t.Fatalf("actor-timezone timestamp projection lost timezone or microsecond formatting: %s", userProjection)
	}
}

func TestParseCycleInputDatesUsesActorCalendarDayBeforeProjectNormalization(t *testing.T) {
	startText, endText := "2099-02-01T00:30:00Z", "2099-02-14T00:30:00Z"
	start, end, err := parseCycleInputDates(&startText, &endText, "America/Los_Angeles", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if got := start.Format(time.RFC3339); got != "2099-01-31T00:00:01Z" {
		t.Errorf("normalized start = %s, want actor-local Jan 31 projected to UTC", got)
	}
	if got := end.Format(time.RFC3339); got != "2099-02-13T23:59:00Z" {
		t.Errorf("normalized end = %s, want actor-local Feb 13 projected to UTC", got)
	}
}

func TestParseCycleInputDatesRejectsReversedInstantsOnSameDay(t *testing.T) {
	startText, endText := "2099-02-01T15:00:00Z", "2099-02-01T14:00:00Z"
	_, _, err := parseCycleInputDates(&startText, &endText, "UTC", "UTC")
	var httpError *djangoapi.HTTPError
	if !errors.As(err, &httpError) || httpError.Status != 400 {
		t.Fatalf("reversed same-day pair error = %v, want source-compatible 400", err)
	}
}

type lifecycleEffectsStub struct{}

func (lifecycleEffectsStub) Task(context.Context, string, []any, map[string]any) error { return nil }
func (lifecycleEffectsStub) SoftDelete(context.Context, string, string, string) error  { return nil }

func TestParseCycleDateMatchesCycleDayBoundaries(t *testing.T) {
	day := "2099-02-01"
	start, err := parseCycleDate(&day, "UTC", true)
	if err != nil {
		t.Fatal(err)
	}
	end, err := parseCycleDate(&day, "UTC", false)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2099, time.February, 1, 0, 0, 1, 0, time.UTC); !start.Equal(want) {
		t.Fatalf("start boundary = %s, want %s", start, want)
	}
	if want := time.Date(2099, time.February, 1, 23, 59, 0, 0, time.UTC); !end.Equal(want) {
		t.Fatalf("end boundary = %s, want %s", end, want)
	}
}

func TestParseCycleDateTimeKeepsSingleDateAtMidnight(t *testing.T) {
	day := "2099-03-21"
	parsed, err := parseCycleDateTime(day, "America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2099, time.March, 21, 0, 0, 0, 0, location)
	if !parsed.Equal(want) {
		t.Fatalf("single date = %s, want %s", parsed, want)
	}
}

func TestParseCycleDateTimeUsesDjangoDateTimeFormats(t *testing.T) {
	for _, test := range []struct {
		input string
		zone  string
		want  string
	}{
		{input: "2026-1-15 4:5", zone: "America/Los_Angeles", want: "2026-01-15T04:05:00-08:00"},
		{input: "2026-1-15T4:5:6+0530", zone: "UTC", want: "2026-01-14T22:35:06Z"},
		{input: "2026-01-15", zone: "America/Los_Angeles", want: "2026-01-15T00:00:00-08:00"},
	} {
		parsed, err := parseCycleDateTime(test.input, test.zone)
		if err != nil || parsed.Format(time.RFC3339) != test.want {
			t.Errorf("parseCycleDateTime(%q, %q) = %v, %v; want %s", test.input, test.zone, parsed, err, test.want)
		}
	}
}

func TestCycleStatusDistinguishesDraftUpcomingCurrentAndCompleted(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	before, after := now.Add(-time.Hour), now.Add(time.Hour)
	for _, test := range []struct {
		name       string
		start, end *time.Time
		want       string
	}{
		{name: "draft", want: "DRAFT"},
		{name: "upcoming", start: &after, end: &after, want: "UPCOMING"},
		{name: "current", start: &before, end: &after, want: "CURRENT"},
		{name: "completed", start: &before, end: &before, want: "COMPLETED"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := cycleStatus(test.start, test.end, now); got != test.want {
				t.Fatalf("cycleStatus() = %q, want %q", got, test.want)
			}
		})
	}
}
