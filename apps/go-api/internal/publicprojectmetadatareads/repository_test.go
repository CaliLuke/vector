package publicprojectmetadatareads

import (
	"context"
	"encoding/json"
	"errors"
	"math"
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
	row := database.rows[0]
	database.rows = database.rows[1:]
	return row
}

func anchorRow(projectID *uuid.UUID) pgx.Row {
	return rowFunc(func(destinations ...any) error {
		*destinations[0].(*string) = "demo"
		*destinations[1].(**uuid.UUID) = projectID
		return nil
	})
}

func jsonRow(value string) pgx.Row {
	return jsonRows(value, 1)
}

func jsonRows(value string, matches int64) pgx.Row {
	return rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(value)
		if len(destinations) > 1 {
			*destinations[1].(*int64) = matches
		}
		return nil
	})
}

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) returned nil error")
	}
}

func TestPublicCataloguesResolveAnchorAndPreserveDjangoOrdering(t *testing.T) {
	for _, test := range []struct {
		name      string
		read      func(*Repository, context.Context, string) (json.RawMessage, error)
		fragments []string
	}{
		{"cycles", (*Repository).Cycles, []string{"cycle.deleted_at IS NULL", "ORDER BY cycle.created_at DESC", "cycle.project_id IS NOT DISTINCT FROM $2"}},
		{"labels", (*Repository).Labels, []string{"label.deleted_at IS NULL", "ORDER BY label.created_at DESC", "label.project_id IS NOT DISTINCT FROM $2"}},
		{"modules", (*Repository).Modules, []string{"module.deleted_at IS NULL", "ORDER BY module.created_at DESC", "module.project_id IS NOT DISTINCT FROM $2"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			projectID := uuid.New()
			database := &databaseStub{rows: []pgx.Row{anchorRow(&projectID), jsonRow(`[]`)}}
			repository, err := NewRepository(database)
			if err != nil {
				t.Fatal(err)
			}
			data, err := test.read(repository, context.Background(), "anchor")
			if err != nil || string(data) != `[]` {
				t.Fatalf("data=%s err=%v", data, err)
			}
			for _, fragment := range test.fragments {
				if !strings.Contains(database.queries[1], fragment) {
					t.Errorf("query missing %q", fragment)
				}
			}
		})
	}
}

func TestProjectBoardsReturnsOnlyLiveEnabledConsistentPublicProjects(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{jsonRow(`[]`)}}
	repository, _ := NewRepository(database)
	data, err := repository.ProjectBoards(context.Background(), "missing-workspace")
	if err != nil || string(data) != `[]` {
		t.Fatalf("data=%s err=%v", data, err)
	}
	if got := database.args[0]; len(got) != 1 || got[0] != "missing-workspace" {
		t.Fatalf("args=%v", got)
	}
	for _, fragment := range []string{
		"workspace.deleted_at IS NULL",
		"project.deleted_at IS NULL",
		"board.deleted_at IS NULL",
		"board.entity_name = 'project'",
		"board.is_disabled = FALSE",
		"board.entity_identifier = project.id",
		"board.workspace_id = workspace.id",
		"DISTINCT ON (project.id)",
		"'icon_prop', project.icon_prop",
	} {
		if !strings.Contains(database.queries[0], fragment) {
			t.Errorf("board directory query missing %q", fragment)
		}
	}
}

func TestResolvePublicBoardRequiresConsistentLiveEnabledProjectBoard(t *testing.T) {
	projectID := uuid.New()
	workspaceID := uuid.New()
	intakeID := uuid.New()
	database := &databaseStub{rows: []pgx.Row{rowFunc(func(destinations ...any) error {
		*destinations[0].(*string) = "demo"
		*destinations[1].(*uuid.UUID) = workspaceID
		*destinations[2].(*uuid.UUID) = projectID
		*destinations[3].(**uuid.UUID) = &intakeID
		*destinations[4].(*bool) = true
		return nil
	})}}
	repository, _ := NewRepository(database)
	board, err := repository.ResolvePublicBoard(context.Background(), "anchor")
	if err != nil || board.WorkspaceSlug != "demo" || board.WorkspaceID != workspaceID || board.ProjectID != projectID || board.IntakeID == nil || *board.IntakeID != intakeID || !board.VotesEnabled {
		t.Fatalf("board=%+v err=%v", board, err)
	}
	for _, fragment := range []string{
		"deploy_board.entity_name = 'project'",
		"deploy_board.deleted_at IS NULL",
		"deploy_board.is_disabled = FALSE",
		"project.id = deploy_board.entity_identifier",
		"project.id = deploy_board.project_id",
		"project.workspace_id = deploy_board.workspace_id",
		"project.archived_at IS NULL",
	} {
		if !strings.Contains(database.queries[0], fragment) {
			t.Errorf("board scope query missing %q", fragment)
		}
	}
}

func TestIssueDetailUsesPublicScopeAndReactionActorForAvatarURL(t *testing.T) {
	projectID, workspaceID := uuid.New(), uuid.New()
	response := `{"id":"issue","reaction_items":[{"reaction":"heart","actor_details":{"avatar_url":"/api/assets/v2/static/reaction-asset/"}}],"vote_items":null}`
	database := &databaseStub{rows: []pgx.Row{jsonRow(response)}}
	repository, _ := NewRepository(database)
	data, err := repository.IssueDetail(context.Background(), PublicBoard{ProjectID: projectID, WorkspaceID: workspaceID}, "issue-id")
	if err != nil || string(data) != response {
		t.Fatalf("data=%s err=%v", data, err)
	}
	if got := database.args[0]; len(got) != 3 || got[0] != "issue-id" || got[1] != projectID || got[2] != workspaceID {
		t.Fatalf("args=%v", got)
	}
	for _, fragment := range []string{
		"issue.deleted_at IS NULL",
		"issue.archived_at IS NULL",
		"issue.is_draft = FALSE",
		`state."group" IS DISTINCT FROM 'triage'`,
		"issue.project_id = $2::uuid",
		"issue.workspace_id = $3::uuid",
		"actor.avatar_asset_id",
		"reaction.actor_id",
		"vote.actor_id",
		"'reaction_items'",
		"'vote_items'",
	} {
		if !strings.Contains(database.queries[0], fragment) {
			t.Errorf("public issue detail query missing %q", fragment)
		}
	}
}

func TestMembersApplyDjangoManagerAndEndpointFilters(t *testing.T) {
	projectID := uuid.New()
	database := &databaseStub{rows: []pgx.Row{anchorRow(&projectID), jsonRow(`[]`)}}
	repository, _ := NewRepository(database)
	data, err := repository.Members(context.Background(), "anchor")
	if err != nil || string(data) != `[]` {
		t.Fatalf("data=%s err=%v", data, err)
	}
	for _, fragment := range []string{
		"LEFT JOIN users member",
		"project_member.deleted_at IS NULL",
		"project_member.is_active = TRUE",
		"ORDER BY project_member.created_at DESC",
		"project_member.project_id IS NOT DISTINCT FROM $2",
	} {
		if !strings.Contains(database.queries[1], fragment) {
			t.Errorf("members query missing %q", fragment)
		}
	}
}

func TestSettingsCarriesAuthenticatedUserToTimezoneLookup(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{jsonRow(`{"id":"00000000-0000-4000-8000-000000000001"}`)}}
	repository, _ := NewRepository(database)
	ctx := WithUserID(context.Background(), "00000000-0000-4000-8000-000000000099")
	if _, err := repository.Settings(ctx, "anchor"); err != nil {
		t.Fatal(err)
	}
	if database.args[0][3] != "00000000-0000-4000-8000-000000000099" {
		t.Fatalf("authenticated settings args=%#v", database.args[0])
	}
}

func TestMetaUsesPublishedEntityIdentifierAndDjangoManagerFilters(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{jsonRow(`{"id":"00000000-0000-4000-8000-000000000001"}`)}}
	repository, _ := NewRepository(database)
	data, err := repository.Meta(context.Background(), "anchor")
	if err != nil || string(data) != `{"id":"00000000-0000-4000-8000-000000000001"}` {
		t.Fatalf("data=%s err=%v", data, err)
	}
	query := database.queries[0]
	for _, fragment := range []string{
		"project.id = deploy_board.entity_identifier",
		"deploy_board.entity_name = 'project'",
		"deploy_board.deleted_at IS NULL",
		"project.deleted_at IS NULL",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("meta query missing %q", fragment)
		}
	}
}

func TestMetaMissingProjectMapsToNotPublished(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{rowFunc(func(...any) error { return pgx.ErrNoRows })}}
	repository, _ := NewRepository(database)
	_, err := repository.Meta(context.Background(), "missing")
	if !errors.Is(err, ErrProjectNotPublished) {
		t.Fatalf("err=%v", err)
	}
}

func TestSettingsUsesDjangoRelationsFiltersAndRendering(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{jsonRow(`{ "id" : "00000000-0000-4000-8000-000000000001", "project_details" : null }`)}}
	repository, _ := NewRepository(database)
	data, err := repository.Settings(context.Background(), "anchor")
	if err != nil || string(data) != `{"id":"00000000-0000-4000-8000-000000000001","project_details":null}` {
		t.Fatalf("data=%s err=%v", data, err)
	}
	query := database.queries[0]
	for _, fragment := range []string{
		"SELECT set_config('TimeZone', requested_timezone.name, true)",
		"SELECT user_timezone FROM users WHERE id = $4::uuid",
		"$1 = 'anchor' AND deploy_board.anchor = $2",
		"$1 = 'project' AND workspace.slug = $2 AND deploy_board.project_id = $3::uuid",
		"count(*) OVER ()",
		"deploy_board.entity_name = 'project'",
		"deploy_board.deleted_at IS NULL",
		"LEFT JOIN projects project ON project.id = deploy_board.project_id",
		"LEFT JOIN file_assets project_cover_asset",
		"LEFT JOIN file_assets workspace_logo_asset",
		"CASE WHEN project.id IS NULL THEN NULL",
		"project_cover_asset.entity_type = 'ISSUE_ATTACHMENT'",
		"workspace_logo_asset.entity_type IN",
		`to_char(deploy_board.created_at, 'YYYY-MM-DD"T"HH24:MI:SS')`,
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("settings query missing %q", fragment)
		}
	}
	if len(database.args[0]) != 4 || database.args[0][0] != "anchor" || database.args[0][1] != "anchor" || database.args[0][2] != uuid.Nil() || database.args[0][3] != "" {
		t.Fatalf("anonymous settings args=%#v", database.args[0])
	}
}

func TestProjectAnchorUsesWorkspaceAndProjectLookup(t *testing.T) {
	projectID := uuid.New()
	database := &databaseStub{rows: []pgx.Row{jsonRow(`{"id":"00000000-0000-4000-8000-000000000001"}`)}}
	repository, _ := NewRepository(database)
	if _, err := repository.ProjectAnchor(context.Background(), "demo", projectID); err != nil {
		t.Fatal(err)
	}
	if len(database.args[0]) != 4 || database.args[0][0] != "project" || database.args[0][1] != "demo" || database.args[0][2] != projectID || database.args[0][3] != "" {
		t.Fatalf("project anchor args=%#v", database.args[0])
	}
}

func TestProjectAnchorRejectsDuplicateDeployBoards(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{jsonRows(`{"id":"00000000-0000-4000-8000-000000000001"}`, 2)}}
	repository, _ := NewRepository(database)
	_, err := repository.ProjectAnchor(context.Background(), "demo", uuid.New())
	if err == nil || errors.Is(err, ErrRender) {
		t.Fatalf("duplicate project anchor err=%v", err)
	}
}

func TestSettingsQueryFailureIsAnEagerViewError(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{rowFunc(func(...any) error { return errors.New("database unavailable") })}}
	repository, _ := NewRepository(database)
	_, err := repository.Settings(context.Background(), "anchor")
	if err == nil || errors.Is(err, ErrRender) {
		t.Fatalf("settings query err=%v", err)
	}
}

func TestSettingsMissingObjectMapsToRequiredObject(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{rowFunc(func(...any) error { return pgx.ErrNoRows })}}
	repository, _ := NewRepository(database)
	_, err := repository.Settings(context.Background(), "missing")
	if !errors.Is(err, ErrRequiredObject) {
		t.Fatalf("err=%v", err)
	}
}

func TestStatesApplyManagerAndEndpointFilters(t *testing.T) {
	projectID := uuid.New()
	database := &databaseStub{rows: []pgx.Row{anchorRow(&projectID), jsonRow(`[]`)}}
	repository, _ := NewRepository(database)
	if _, err := repository.States(context.Background(), "anchor"); err != nil {
		t.Fatal(err)
	}
	query := database.queries[1]
	for _, fragment := range []string{`state."group" <> 'triage'`, "state.name <> 'Triage'", "state.deleted_at IS NULL", "state.project_id IS NOT DISTINCT FROM $2", "state.sequence::text", "ORDER BY state.sequence"} {
		if !strings.Contains(query, fragment) {
			t.Errorf("states query missing %q", fragment)
		}
	}
}

func TestStatesPreserveDjangoStringAndFloatEncoding(t *testing.T) {
	projectID := uuid.New()
	database := &databaseStub{rows: []pgx.Row{
		anchorRow(&projectID),
		jsonRow(`[{"name":"<&>","group":"backlog","color":"#000","id":"00000000-0000-4000-8000-000000000001","sequence":"100"}]`),
	}}
	repository, _ := NewRepository(database)
	data, err := repository.States(context.Background(), "anchor")
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"name":"<&>","group":"backlog","color":"#000","id":"00000000-0000-4000-8000-000000000001","sequence":100.0}]`
	if string(data) != want {
		t.Fatalf("states=%s want=%s", data, want)
	}
}

func TestPythonFloatJSONMatchesDjangoRenderingBoundaries(t *testing.T) {
	for _, testCase := range []struct {
		value float64
		want  string
	}{
		{0, "0.0"},
		{math.Copysign(0, -1), "-0.0"},
		{100, "100.0"},
		{1.25, "1.25"},
		{1e-4, "0.0001"},
		{1e-5, "1e-05"},
		{1e15, "1000000000000000.0"},
		{1.234e15, "1234000000000000.0"},
		{1e16, "1e+16"},
		{1e20, "1e+20"},
	} {
		got, err := pythonFloatJSON(testCase.value)
		if err != nil || string(got) != testCase.want {
			t.Errorf("pythonFloatJSON(%v) = %q, %v; want %q", testCase.value, got, err, testCase.want)
		}
	}
}

func TestPythonFloatJSONRejectsNonFiniteValues(t *testing.T) {
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := pythonFloatJSON(value); err == nil {
			t.Errorf("pythonFloatJSON(%v) returned nil error", value)
		}
	}
}

func TestMissingAnchorMapsToInvalidAnchor(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{rowFunc(func(...any) error { return pgx.ErrNoRows })}}
	repository, _ := NewRepository(database)
	_, err := repository.Labels(context.Background(), "missing")
	if !errors.Is(err, ErrInvalidAnchor) {
		t.Fatalf("err=%v", err)
	}
}

func TestMetadataQueryFailureIsMarkedAsRenderPhase(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{
		anchorRow(nil),
		rowFunc(func(...any) error { return errors.New("database unavailable") }),
	}}
	repository, _ := NewRepository(database)
	_, err := repository.Labels(context.Background(), "anchor")
	if !errors.Is(err, ErrRender) {
		t.Fatalf("err=%v", err)
	}
}
