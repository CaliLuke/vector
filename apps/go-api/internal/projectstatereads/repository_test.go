package projectstatereads

import (
	"context"
	"encoding/json"
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

func TestAuthorizeProjectMirrorsAllowedRolesAndWorkspaceAdminOverride(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true)}}
	repository, _ := NewRepository(database)
	userID := uuid.New()
	projectID := uuid.New()
	if err := repository.AuthorizeProject(context.Background(), userID, "demo", projectID); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"project_member.role IN (5, 15, 20)", "workspace_member.role = 20",
		"project_member.is_active = true", "project_member.deleted_at IS NULL",
		"workspace.slug = $2", "project.id = $3",
	} {
		if !strings.Contains(database.queries[0], fragment) {
			t.Errorf("authorization query missing %q", fragment)
		}
	}
	if len(database.args[0]) != 3 || database.args[0][0] != userID || database.args[0][2] != projectID {
		t.Fatalf("authorization args=%#v", database.args[0])
	}
}

func TestAuthorizeProjectMapsMissingMembershipToForbidden(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(false)}}
	repository, _ := NewRepository(database)
	if err := repository.AuthorizeProject(context.Background(), uuid.New(), "demo", uuid.New()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("err=%v", err)
	}
}

func TestListScopesStatesAndCalculatesOrderWithinEachGroup(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true), jsonRow(`[]`)}}
	repository, _ := NewRepository(database)
	data, err := repository.List(context.Background(), uuid.New(), "demo", uuid.New())
	if err != nil || string(data) != `[]` {
		t.Fatalf("data=%s err=%v", data, err)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"project.archived_at IS NULL", "state.deleted_at IS NULL", `state."group" <> 'triage'`,
		"state.is_triage = false", "PARTITION BY state.\"group\"", "ORDER BY state.sequence",
		"project_member.is_active = true", "project_member.deleted_at IS NULL",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("list query missing %q", fragment)
		}
	}
}

func TestDetailHidesUnavailableStatesAsNotFound(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{rowFunc(func(...any) error { return pgx.ErrNoRows })}}
	repository, _ := NewRepository(database)
	_, err := repository.Detail(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
	for _, fragment := range []string{
		"project.archived_at IS NULL", `state."group" <> 'triage'`, "state.is_triage = false",
		"project_member.is_active = true", "state.id = $4",
	} {
		if !strings.Contains(database.queries[0], fragment) {
			t.Errorf("detail query missing %q", fragment)
		}
	}
	if strings.Contains(database.queries[0], "project_member.deleted_at IS NULL") {
		t.Fatalf("detail query must preserve Django's reverse-relation soft-delete behavior: %s", database.queries[0])
	}
}

func TestIntakeUsesTriageGroupWithoutArchivedOrIsTriageFilters(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true), jsonRow(`{"group":"triage"}`)}}
	repository, _ := NewRepository(database)
	data, err := repository.Intake(context.Background(), uuid.New(), "demo", uuid.New())
	if err != nil || string(data) != `{"group":"triage"}` {
		t.Fatalf("data=%s err=%v", data, err)
	}
	query := database.queries[1]
	if !strings.Contains(query, `state."group" = 'triage'`) || strings.Contains(query, "state.is_triage") || strings.Contains(query, "archived_at") {
		t.Fatalf("intake query does not mirror the triage manager: %s", query)
	}
	if len(database.args[1]) != 2 || database.args[1][0] != "demo" {
		t.Fatalf("intake args=%#v", database.args[1])
	}
}

func TestAPIV1ListKeepsSerializerFieldsPaginationAndProjectScope(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{jsonRow(`{"total_count":1,"has_next":false,"results":[{"id":"state-1","name":"Backlog","description":"plain text","color":"#fff","name_id":null,"sequence":10,"project_id":"project-1","workspace_id":"workspace-1"}]}`)}}
	repository, _ := NewRepository(database)
	request := APIV1Request{
		Operation: "list",
		Params:    map[string]string{"slug": "demo", "project_id": "project-1", "per_page": "1", "fields": "color,name,description,sequence", "expand": "name"},
		Present:   map[string]bool{"per_page": true, "fields": true, "expand": true},
	}
	result, err := repository.ReadAPIV1(context.Background(), "user-1", request)
	if err != nil {
		t.Fatal(err)
	}
	page := result.(map[string]any)
	row := page["results"].([]any)[0].(map[string]any)
	if row["color"] != "#fff" || row["name"] != nil || row["description"] != "plain text" || row["sequence"] != json.Number("10.0") || page["total_count"] != 1 {
		t.Fatalf("result=%#v", result)
	}
	query := database.queries[0]
	for _, fragment := range []string{
		"project.archived_at IS NULL", "state.deleted_at IS NULL", `state."group" <> 'triage'`,
		"state.is_triage=false", "project_member.is_active=true", "project_member.deleted_at IS NULL",
		"EXISTS (", "ORDER BY state.sequence", "LIMIT $4 OFFSET $5",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("API state query missing %q", fragment)
		}
	}
	if strings.Contains(query, "workspace.deleted_at") || strings.Contains(query, "project.deleted_at") {
		t.Fatalf("API state query must preserve related-object soft-delete visibility: %s", query)
	}
	if got := database.args[0]; len(got) != 5 || got[0] != "demo" || got[1] != "project-1" || got[2] != "user-1" || got[3] != 1 || got[4] != 0 {
		t.Fatalf("args=%#v", got)
	}
}

func TestAPIV1DetailUsesPathStateAndAppliesDynamicProjection(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{jsonRow(`{"total_count":1,"has_next":false,"results":[{"id":"state-1","name":"Backlog","color":"#fff","sequence":10,"project_id":"project-1","workspace_id":"workspace-1"}]}`)}}
	repository, _ := NewRepository(database)
	request := APIV1Request{
		Operation: "detail",
		Params:    map[string]string{"slug": "demo", "project_id": "project-1", "state_id": "path-state", "fields": "name,color", "expand": "name"},
		Present:   map[string]bool{"fields": true, "expand": true},
	}
	result, err := repository.ReadAPIV1(context.Background(), "user-1", request)
	if err != nil {
		t.Fatal(err)
	}
	row := result.(map[string]any)
	if len(row) != 2 || row["color"] != "#fff" || row["name"] != nil {
		t.Fatalf("result=%#v", result)
	}
	if len(database.args[0]) != 4 || database.args[0][3] != "path-state" {
		t.Fatalf("path state was not used: args=%#v", database.args[0])
	}
}
