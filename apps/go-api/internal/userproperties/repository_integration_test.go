//go:build integration

package userproperties

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"uuid"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	workspaceID    = "00000000-0000-4000-8000-000000000101"
	projectID      = "00000000-0000-4000-8000-000000000102"
	otherProjectID = "00000000-0000-4000-8000-000000000103"
	cycleID        = "00000000-0000-4000-8000-000000000104"
	moduleID       = "00000000-0000-4000-8000-000000000105"
	userID         = "00000000-0000-4000-8000-000000000106"
	otherUserID    = "00000000-0000-4000-8000-000000000107"
)

func TestRepositoryFirstAccessAndRepeatUseDjangoDefaultsForEveryKind(t *testing.T) {
	ctx := context.Background()
	pool := propertyTestDB(t, ctx)
	repository, err := NewRepository(pool)
	if err != nil {
		t.Fatal(err)
	}
	user := uuid.MustParse(userID)
	project := uuid.MustParse(projectID)
	cycle := uuid.MustParse(cycleID)
	module := uuid.MustParse(moduleID)

	cases := []struct {
		name     string
		get      func() (json.RawMessage, error)
		table    string
		defaults map[string]any
	}{
		{
			"workspace", func() (json.RawMessage, error) { return repository.GetWorkspace(ctx, user, "demo") }, "workspace_user_properties",
			map[string]any{
				"filters":                  map[string]any{"priority": nil, "state": nil, "state_group": nil, "assignees": nil, "created_by": nil, "labels": nil, "start_date": nil, "target_date": nil, "subscriber": nil},
				"display_filters":          map[string]any{"display_filters": map[string]any{"group_by": nil, "order_by": "-created_at", "type": nil, "sub_issue": true, "show_empty_groups": true, "layout": "list", "calendar_date_range": ""}},
				"display_properties":       map[string]any{"display_properties": map[string]any{"assignee": true, "attachment_count": true, "created_on": true, "due_date": true, "estimate": true, "key": true, "labels": true, "link": true, "priority": true, "start_date": true, "state": true, "sub_issue_count": true, "updated_on": true}},
				"navigation_project_limit": float64(10), "navigation_control_preference": "ACCORDION",
			},
		},
		{"project", func() (json.RawMessage, error) { return repository.GetProject(ctx, user, "demo", project) }, "project_user_properties", map[string]any{"sort_order": float64(65535), "preferences": map[string]any{"pages": map[string]any{"block_display": true}, "navigation": map[string]any{"default_tab": "work_items", "hide_in_more_menu": []any{}}}}},
		{"cycle", func() (json.RawMessage, error) { return repository.GetCycle(ctx, user, "demo", project, cycle) }, "cycle_user_properties", nil},
		{"module", func() (json.RawMessage, error) { return repository.GetModule(ctx, user, "demo", project, module) }, "module_user_properties", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			first, getErr := tc.get()
			if getErr != nil {
				t.Fatalf("first get: %v", getErr)
			}
			second, getErr := tc.get()
			if getErr != nil {
				t.Fatalf("repeat get: %v", getErr)
			}
			firstFields, secondFields := propertyFields(t, first), propertyFields(t, second)
			if firstFields["id"] != secondFields["id"] {
				t.Fatalf("repeat get changed id: %v then %v", firstFields["id"], secondFields["id"])
			}
			if firstFields["created_at"] == nil || firstFields["updated_at"] == nil {
				t.Fatalf("timestamps missing from response: %s", first)
			}
			if firstFields["created_by"] != userID || firstFields["updated_by"] != nil {
				t.Fatalf("creation audit fields = created_by:%v updated_by:%v", firstFields["created_by"], firstFields["updated_by"])
			}
			if _, parseErr := time.Parse(time.RFC3339Nano, firstFields["created_at"].(string)); parseErr != nil {
				t.Fatalf("created_at is invalid: %v", parseErr)
			}
			for key, expected := range tc.defaults {
				if !jsonEqual(firstFields[key], expected) {
					t.Errorf("%s default = %#v, want %#v", key, firstFields[key], expected)
				}
			}
			var count int
			if scanErr := pool.QueryRow(ctx, "SELECT count(*) FROM "+tc.table).Scan(&count); scanErr != nil || count != 1 {
				t.Fatalf("%s rows = %d, err=%v", tc.table, count, scanErr)
			}
		})
	}
	first, err := repository.GetProject(ctx, user, "demo", project)
	if err != nil {
		t.Fatal(err)
	}
	conflicted, err := repository.createScoped(ctx, user, "demo", project, uuid.Nil(), "project")
	if err != nil {
		t.Fatalf("create against an existing property: %v", err)
	}
	if propertyFields(t, conflicted)["id"] != propertyFields(t, first)["id"] {
		t.Fatalf("existing-row conflict created a new property: %s then %s", first, conflicted)
	}
}

func TestRepositoryPatchesOnlyProvidedFieldsAndIsolatesUsers(t *testing.T) {
	ctx := context.Background()
	pool := propertyTestDB(t, ctx)
	repository, _ := NewRepository(pool)
	user := uuid.MustParse(userID)
	otherUser := uuid.MustParse(otherUserID)
	project := uuid.MustParse(projectID)
	cycle := uuid.MustParse(cycleID)
	module := uuid.MustParse(moduleID)

	projectBefore, err := repository.GetProject(ctx, user, "demo", project)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.PatchProject(ctx, user, "demo", project, map[string]json.RawMessage{
		"filters":     json.RawMessage(`{"priority":["high"]}`),
		"preferences": json.RawMessage(`{"pages":{"block_display":false}}`),
	})
	if err != nil {
		t.Fatalf("patch project: %v", err)
	}
	projectAfter, err := repository.GetProject(ctx, user, "demo", project)
	if err != nil {
		t.Fatal(err)
	}
	before, after := propertyFields(t, projectBefore), propertyFields(t, projectAfter)
	if !jsonEqual(after["filters"], map[string]any{"priority": []any{"high"}}) || !jsonEqual(after["preferences"], map[string]any{"pages": map[string]any{"block_display": false}}) {
		t.Fatalf("project patch fields = filters:%v preferences:%v", after["filters"], after["preferences"])
	}
	if after["created_by"] != userID || after["updated_by"] != userID {
		t.Fatalf("project audit fields = created_by:%v updated_by:%v", after["created_by"], after["updated_by"])
	}
	for _, key := range []string{"display_filters", "display_properties", "rich_filters", "sort_order"} {
		if !jsonEqual(after[key], before[key]) {
			t.Errorf("omitted project field %s changed: %v -> %v", key, before[key], after[key])
		}
	}

	if _, getWorkspaceErr := repository.GetWorkspace(ctx, user, "demo"); getWorkspaceErr != nil {
		t.Fatalf("create workspace properties before patch: %v", getWorkspaceErr)
	}
	workspaceBefore, err := repository.GetWorkspace(ctx, user, "demo")
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.PatchWorkspace(ctx, user, "demo", map[string]json.RawMessage{"navigation_project_limit": json.RawMessage(`25`)})
	if err != nil {
		t.Fatalf("patch workspace: %v", err)
	}
	workspace, err := repository.GetWorkspace(ctx, user, "demo")
	if err != nil || propertyFields(t, workspace)["navigation_project_limit"] != float64(25) {
		t.Fatalf("workspace patch = %s err=%v", workspace, err)
	}
	if propertyFields(t, workspace)["navigation_control_preference"] != propertyFields(t, workspaceBefore)["navigation_control_preference"] {
		t.Fatalf("workspace patch changed omitted navigation preference: %s", workspace)
	}

	cycleBefore, err := repository.GetCycle(ctx, user, "demo", project, cycle)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.PatchCycle(ctx, user, "demo", project, cycle, map[string]json.RawMessage{"display_filters": json.RawMessage(`{"layout":"board"}`)})
	if err != nil {
		t.Fatalf("patch cycle: %v", err)
	}
	cycleAfter, err := repository.GetCycle(ctx, user, "demo", project, cycle)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"filters", "display_properties", "rich_filters"} {
		if !jsonEqual(propertyFields(t, cycleAfter)[key], propertyFields(t, cycleBefore)[key]) {
			t.Errorf("cycle patch changed omitted %s: %s", key, cycleAfter)
		}
	}

	moduleBefore, err := repository.GetModule(ctx, user, "demo", project, module)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.PatchModule(ctx, user, "demo", project, module, map[string]json.RawMessage{"rich_filters": json.RawMessage(`{"and":[]}`)})
	if err != nil {
		t.Fatalf("patch module: %v", err)
	}
	moduleAfter, err := repository.GetModule(ctx, user, "demo", project, module)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"filters", "display_filters", "display_properties"} {
		if !jsonEqual(propertyFields(t, moduleAfter)[key], propertyFields(t, moduleBefore)[key]) {
			t.Errorf("module patch changed omitted %s: %s", key, moduleAfter)
		}
	}

	other, err := repository.GetProject(ctx, otherUser, "demo", project)
	if err != nil {
		t.Fatal(err)
	}
	if propertyFields(t, other)["id"] == propertyFields(t, projectAfter)["id"] || jsonEqual(propertyFields(t, other)["filters"], propertyFields(t, projectAfter)["filters"]) {
		t.Fatalf("project properties leaked across users: %s", other)
	}
	if _, err := repository.GetCycle(ctx, user, "demo", uuid.MustParse(otherProjectID), cycle); err == nil {
		t.Fatal("cycle was returned under a different project")
	}
}

func TestRepositoryConcurrentFirstAccessCreatesOneProjectProperty(t *testing.T) {
	ctx := context.Background()
	pool := propertyTestDB(t, ctx)
	repository, _ := NewRepository(pool)
	const calls = 12
	ids := make(chan string, calls)
	errs := make(chan error, calls)
	var wg sync.WaitGroup
	for range calls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body, err := repository.GetProject(ctx, uuid.MustParse(userID), "demo", uuid.MustParse(projectID))
			if err != nil {
				errs <- err
				return
			}
			ids <- propertyFields(t, body)["id"].(string)
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Errorf("concurrent get: %v", err)
	}
	var first string
	for id := range ids {
		if first == "" {
			first = id
		} else if id != first {
			t.Errorf("concurrent get created multiple rows: %s and %s", first, id)
		}
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM project_user_properties").Scan(&count); err != nil || count != 1 {
		t.Fatalf("project property rows = %d, err=%v", count, err)
	}
}

func propertyFields(t *testing.T, body json.RawMessage) map[string]any {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatalf("decode property %s: %v", body, err)
	}
	return fields
}

func jsonEqual(actual, expected any) bool {
	left, _ := json.Marshal(actual)
	right, _ := json.Marshal(expected)
	return string(left) == string(right)
}

func propertyTestDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE users (id uuid PRIMARY KEY);
INSERT INTO users VALUES ('`+userID+`'), ('`+otherUserID+`');
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL, deleted_at timestamptz);
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid NOT NULL REFERENCES workspaces(id), deleted_at timestamptz);
CREATE TABLE cycles (id uuid PRIMARY KEY, project_id uuid NOT NULL REFERENCES projects(id), deleted_at timestamptz);
CREATE TABLE modules (id uuid PRIMARY KEY, project_id uuid NOT NULL REFERENCES projects(id), deleted_at timestamptz);
INSERT INTO workspaces VALUES ('`+workspaceID+`','demo',NULL);
INSERT INTO projects VALUES ('`+projectID+`','`+workspaceID+`',NULL), ('`+otherProjectID+`','`+workspaceID+`',NULL);
INSERT INTO cycles VALUES ('`+cycleID+`','`+projectID+`',NULL);
INSERT INTO modules VALUES ('`+moduleID+`','`+projectID+`',NULL);
CREATE TABLE workspace_user_properties (
 id uuid PRIMARY KEY, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
 created_by_id uuid REFERENCES users(id), updated_by_id uuid REFERENCES users(id), deleted_at timestamptz,
 workspace_id uuid NOT NULL REFERENCES workspaces(id), user_id uuid NOT NULL REFERENCES users(id),
 filters jsonb NOT NULL, display_filters jsonb NOT NULL, display_properties jsonb NOT NULL, rich_filters jsonb NOT NULL,
 navigation_project_limit integer NOT NULL, navigation_control_preference varchar(25) NOT NULL);
CREATE UNIQUE INDEX workspace_user_properties_active_unique ON workspace_user_properties(workspace_id,user_id) WHERE deleted_at IS NULL;
CREATE TABLE project_user_properties (
 id uuid PRIMARY KEY, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
 created_by_id uuid REFERENCES users(id), updated_by_id uuid REFERENCES users(id), deleted_at timestamptz,
 workspace_id uuid NOT NULL REFERENCES workspaces(id), project_id uuid NOT NULL REFERENCES projects(id), user_id uuid NOT NULL REFERENCES users(id),
 filters jsonb NOT NULL, display_filters jsonb NOT NULL, display_properties jsonb NOT NULL, rich_filters jsonb NOT NULL,
 preferences jsonb NOT NULL, sort_order double precision NOT NULL);
CREATE UNIQUE INDEX project_user_properties_active_unique ON project_user_properties(project_id,user_id) WHERE deleted_at IS NULL;
CREATE TABLE cycle_user_properties (
 id uuid PRIMARY KEY, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
 created_by_id uuid REFERENCES users(id), updated_by_id uuid REFERENCES users(id), deleted_at timestamptz,
 workspace_id uuid NOT NULL REFERENCES workspaces(id), project_id uuid NOT NULL REFERENCES projects(id), cycle_id uuid NOT NULL REFERENCES cycles(id), user_id uuid NOT NULL REFERENCES users(id),
 filters jsonb NOT NULL, display_filters jsonb NOT NULL, display_properties jsonb NOT NULL, rich_filters jsonb NOT NULL);
CREATE UNIQUE INDEX cycle_user_properties_active_unique ON cycle_user_properties(cycle_id,user_id) WHERE deleted_at IS NULL;
CREATE TABLE module_user_properties (
 id uuid PRIMARY KEY, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
 created_by_id uuid REFERENCES users(id), updated_by_id uuid REFERENCES users(id), deleted_at timestamptz,
 workspace_id uuid NOT NULL REFERENCES workspaces(id), project_id uuid NOT NULL REFERENCES projects(id), module_id uuid NOT NULL REFERENCES modules(id), user_id uuid NOT NULL REFERENCES users(id),
 filters jsonb NOT NULL, display_filters jsonb NOT NULL, display_properties jsonb NOT NULL, rich_filters jsonb NOT NULL);
CREATE UNIQUE INDEX module_user_properties_active_unique ON module_user_properties(module_id,user_id) WHERE deleted_at IS NULL;`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Errorf("close postgres test container: %v", closeErr)
		}
	})
	pool, err := pgxpool.New(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
