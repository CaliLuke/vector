package projectlabelreads

import (
	"context"
	"errors"
	"strings"
	"testing"

	"uuid"

	"github.com/jackc/pgx/v5"
)

type rowFunc func(...any) error

func (row rowFunc) Scan(destinations ...any) error { return row(destinations...) }

type databaseStub struct {
	queries []string
	args    [][]any
	rows    []pgx.Row
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

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) returned nil error")
	}
}

func TestAuthorizeWorkspaceRequiresActiveNonDeletedMembership(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true)}}
	repository, _ := NewRepository(database)
	userID := uuid.New()
	if err := repository.AuthorizeWorkspace(context.Background(), userID, "demo"); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"workspace_member.is_active = true", "workspace_member.deleted_at IS NULL", "workspace.slug = $2",
	} {
		if !strings.Contains(database.queries[0], fragment) {
			t.Errorf("authorization query missing %q", fragment)
		}
	}
}

func TestAuthorizeWorkspaceMapsMissingMembershipToForbidden(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(false)}}
	repository, _ := NewRepository(database)
	if err := repository.AuthorizeWorkspace(context.Background(), uuid.New(), "demo"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("err=%v", err)
	}
}

func TestListScopesAndOrdersLabelsWhilePreservingInactiveProjectMembership(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true), jsonRow(`[]`)}}
	repository, _ := NewRepository(database)
	data, err := repository.List(context.Background(), uuid.New(), "demo", uuid.New())
	if err != nil || string(data) != `[]` {
		t.Fatalf("data=%s err=%v", data, err)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"label.deleted_at IS NULL", "workspace.slug = $2", "project.id = $3",
		"project_member.member_id = $1", "ORDER BY label.sort_order",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("list query missing %q", fragment)
		}
	}
	if strings.Contains(query, "project_member.is_active") || strings.Contains(query, "project_member.deleted_at") {
		t.Fatalf("list query must preserve Django reverse-relation membership behavior: %s", query)
	}
}

func TestDetailMapsUnavailableLabelToNotFound(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true), rowFunc(func(...any) error { return pgx.ErrNoRows })}}
	repository, _ := NewRepository(database)
	_, err := repository.Detail(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(database.queries[1], "label.id = $4") {
		t.Fatalf("detail query=%s", database.queries[1])
	}
}
