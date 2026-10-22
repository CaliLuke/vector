package estimatereads

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"uuid"
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

func TestWorkspaceAuthorizationScopesActiveMembershipAndTenant(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true)}}
	repository, _ := NewRepository(database)
	userID := uuid.New()
	if err := repository.AuthorizeWorkspace(context.Background(), userID, "acme"); err != nil {
		t.Fatal(err)
	}
	query := database.queries[0]
	for _, fragment := range []string{
		"workspace_member.member_id = $1",
		"workspace.slug = $2",
		"workspace_member.is_active = true",
		"workspace_member.deleted_at IS NULL",
		"workspace.deleted_at IS NULL",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("workspace authorization missing %q", fragment)
		}
	}
	if database.args[0][0] != userID || database.args[0][1] != "acme" {
		t.Fatalf("args=%v", database.args[0])
	}
}

func TestProjectAuthorizationRequiresExactActiveProjectMembership(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true)}}
	repository, _ := NewRepository(database)
	projectID := uuid.New()
	if err := repository.AuthorizeProject(context.Background(), uuid.New(), "acme", projectID); err != nil {
		t.Fatal(err)
	}
	query := database.queries[0]
	for _, fragment := range []string{
		"project_member.member_id = $1",
		"workspace.slug = $2",
		"project.id = $3",
		"project_member.is_active = true",
		"project_member.deleted_at IS NULL",
		"project.deleted_at IS NULL",
		"workspace.deleted_at IS NULL",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("project authorization missing %q", fragment)
		}
	}
}

func TestAuthorizationRejectsMissingMembership(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(false), boolRow(false)}}
	repository, _ := NewRepository(database)
	if err := repository.AuthorizeWorkspace(context.Background(), uuid.New(), "acme"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("workspace err=%v", err)
	}
	if err := repository.AuthorizeProject(context.Background(), uuid.New(), "acme", uuid.New()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("project err=%v", err)
	}
}

func TestWorkspaceSelectedUsesSelectedProjectsInsteadOfAllEstimates(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true), jsonRow(`[]`)}}
	repository, _ := NewRepository(database)
	if _, err := repository.WorkspaceSelected(context.Background(), uuid.New(), "acme"); err != nil {
		t.Fatal(err)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"selected_project.estimate_id = estimate.id",
		"selected_project.workspace_id = workspace.id",
		"selected_project.deleted_at IS NULL",
		"estimate.deleted_at IS NULL",
		"ORDER BY estimate.name",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("workspace selected query missing %q", fragment)
		}
	}
	if strings.Contains(strings.ToLower(query), "search") {
		t.Fatalf("generated search parameter must remain a no-op: %s", query)
	}
}

func TestProjectListAndDetailUseDistinctScopedPaths(t *testing.T) {
	projectID := uuid.New()
	estimateID := uuid.New()
	database := &databaseStub{rows: []pgx.Row{
		boolRow(true), jsonRow(`[]`),
		boolRow(true), jsonRow(`{}`),
	}}
	repository, _ := NewRepository(database)
	if _, err := repository.ProjectList(context.Background(), uuid.New(), "acme", projectID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.ProjectDetail(context.Background(), uuid.New(), "acme", projectID, estimateID); err != nil {
		t.Fatal(err)
	}
	listQuery := database.queries[1]
	detailQuery := database.queries[3]
	if !strings.Contains(listQuery, "project.id = $3") || strings.Contains(listQuery, "estimate.id = $4") {
		t.Fatalf("list query=%s", listQuery)
	}
	if !strings.Contains(detailQuery, "project.id = $3") || !strings.Contains(detailQuery, "estimate.id = $4") {
		t.Fatalf("detail query=%s", detailQuery)
	}
	for _, query := range []string{listQuery, detailQuery} {
		for _, fragment := range []string{
			"point.deleted_at IS NULL",
			"point.workspace_id = estimate.workspace_id",
			"point.project_id = estimate.project_id",
			"ORDER BY point.value",
			"'created_by', point.created_by_id",
			"'updated_by', point.updated_by_id",
			"'deleted_at', point.deleted_at",
			"set_config('TimeZone'",
		} {
			if !strings.Contains(query, fragment) {
				t.Errorf("estimate query missing %q", fragment)
			}
		}
	}
}

func TestProjectDetailMapsUnavailableEstimateToNotFound(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{
		boolRow(true),
		rowFunc(func(...any) error { return pgx.ErrNoRows }),
	}}
	repository, _ := NewRepository(database)
	_, err := repository.ProjectDetail(context.Background(), uuid.New(), "acme", uuid.New(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}
