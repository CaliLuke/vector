package workspacerecentvisits

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"uuid"
)

type recentVisitRow func(...any) error

func (r recentVisitRow) Scan(dest ...any) error { return r(dest...) }

type recentVisitDB struct {
	queries []string
	args    [][]any
	rows    []pgx.Row
}

func (d *recentVisitDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	d.queries = append(d.queries, query)
	d.args = append(d.args, args)
	if len(d.rows) == 0 {
		return recentVisitRow(func(...any) error { return errors.New("unexpected query") })
	}
	row := d.rows[0]
	d.rows = d.rows[1:]
	return row
}

func (d *recentVisitDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func recentVisitJSONRow(value string) pgx.Row {
	return recentVisitRow(func(dest ...any) error {
		*dest[0].(*[]byte) = []byte(value)
		return nil
	})
}

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) returned nil error")
	}
}

func TestListScopesFiltersOrdersAndCapsVisits(t *testing.T) {
	db := &recentVisitDB{rows: []pgx.Row{
		recentVisitRow(func(dest ...any) error { *dest[0].(*bool) = true; return nil }),
		recentVisitJSONRow(`[]`),
	}}
	repository, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	user := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	entityName := "issue"
	if ok, err := repository.Authorized(context.Background(), user, "demo"); err != nil || !ok {
		t.Fatalf("Authorized() = %v, %v", ok, err)
	}
	if _, err := repository.List(context.Background(), user, "demo", &entityName); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(db.queries, "\n")
	for _, want := range []string{
		"wm.role IN (5,15,20)",
		"v.user_id=$1",
		"v.entity_name=COALESCE($3,v.entity_name)",
		"v.entity_name IN ('issue','page','project')",
		"ORDER BY v.created_at DESC LIMIT 20",
		"json_agg(visit ORDER BY created_at DESC)",
		"issue_assignees",
		"project_pages",
		"project_members",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("query missing %q", want)
		}
	}
	if len(db.args) != 2 || len(db.args[1]) != 3 || db.args[1][2] != entityName {
		t.Fatalf("List args = %#v", db.args)
	}
}

func TestRecordUpdatesOrCreatesAndCapsPerUserWorkspace(t *testing.T) {
	db := &recentVisitDB{rows: []pgx.Row{
		recentVisitRow(func(dest ...any) error {
			*dest[0].(*bool) = true
			*dest[1].(*int) = 1
			return nil
		}),
	}}
	repository, _ := NewRepository(db)
	user := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	entityID := uuid.MustParse("00000000-0000-4000-8000-000000000020")
	projectID := uuid.MustParse("00000000-0000-4000-8000-000000000030")
	if err := repository.Record(context.Background(), user, "demo", "project", entityID, projectID); err != nil {
		t.Fatal(err)
	}
	if len(db.queries) != 1 {
		t.Fatalf("Record query count = %d, want one batched statement", len(db.queries))
	}
	query := db.queries[0]
	for _, want := range []string{
		"FOR UPDATE",
		"UPDATE user_recent_visits visit",
		"SET visited_at=NOW()",
		"INSERT INTO user_recent_visits",
		"WHERE NOT EXISTS (SELECT 1 FROM updated)",
		"visit.user_id=$1",
		"candidate.workspace_id=target.workspace_id",
		"row_number() OVER (ORDER BY candidate.created_at DESC",
		"ranked.position>20",
		"SET deleted_at=NOW()",
		"candidate.project_id=$5",
		"target.workspace_id, $5",
	} {
		if !strings.Contains(query, want) {
			t.Errorf("Record query missing %q", want)
		}
	}
	for _, forbidden := range []string{"FROM cycles", "FROM modules", "FROM project_pages"} {
		if strings.Contains(query, forbidden) {
			t.Errorf("Record must use the caller's exact project ID, found %q", forbidden)
		}
	}
	if len(db.args[0]) != 6 || db.args[0][0] != user || db.args[0][1] != "demo" || db.args[0][2] != "project" || db.args[0][3] != entityID || db.args[0][4] != projectID {
		t.Fatalf("Record args = %#v", db.args[0])
	}
	if _, ok := db.args[0][5].(uuid.UUID); !ok {
		t.Fatalf("Record generated id = %#v, want UUID", db.args[0][5])
	}
}

func TestRecordReportsMissingWorkspace(t *testing.T) {
	db := &recentVisitDB{rows: []pgx.Row{
		recentVisitRow(func(dest ...any) error {
			*dest[0].(*bool) = false
			*dest[1].(*int) = 0
			return nil
		}),
	}}
	repository, _ := NewRepository(db)
	err := repository.Record(context.Background(), uuid.New(), "missing", "cycle", uuid.New(), uuid.New())
	if err == nil || !strings.Contains(err.Error(), `workspace "missing" not found`) {
		t.Fatalf("Record missing workspace error = %v", err)
	}
}
