package workspaceprojectreads

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
)

func TestAPIProjectLiteSortOrderUsesProjectMemberAnnotation(t *testing.T) {
	user := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	order := "-sort_order"
	db := &workspaceProjectDatabase{rows: []pgx.Row{workspaceProjectRoleRow(20), workspaceProjectRow(func(dest ...any) error {
		*dest[0].(*[]byte) = []byte(`[]`)
		*dest[1].(*int) = 0
		*dest[2].(*bool) = true
		return nil
	})}}
	repo, _ := NewRepository(db)
	if _, err := repo.APIProjects(context.Background(), user, "space", APIProjectListOptions{OrderBy: &order}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(db.queries[1], "member.sort_order FROM project_members") || !strings.Contains(db.queries[1], "ORDER BY sort_order DESC") || strings.Contains(db.queries[1], "project_user_properties") {
		t.Fatalf("wrong picker ordering: %s", db.queries[1])
	}
}

func TestAPIProjectListUsesFullSerializerAndProjectVisibility(t *testing.T) {
	user := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	db := &workspaceProjectDatabase{rows: []pgx.Row{
		workspaceProjectRoleRow(20),
		workspaceProjectRow(func(dest ...any) error {
			*dest[0].(*[]byte) = []byte(`{"results":[]}`)
			*dest[1].(*bool) = true
			return nil
		}),
	}}
	repo, _ := NewRepository(db)
	body, err := repo.APIProjectList(context.Background(), user, "space", APIProjectListOptions{})
	if err != nil || string(body) != `{"results":[]}` {
		t.Fatalf("list full API projects: body=%s err=%v", body, err)
	}
	query := db.queries[1]
	for _, fragment := range []string{"'total_members'", "member_user.is_bot=false", "'total_cycles'", "'total_modules'", "'is_member'", "workspace.slug=$2", "$3 IN (5,15,20)", "project.network=2 OR EXISTS"} {
		if !strings.Contains(query, fragment) {
			t.Errorf("full API project list query is missing %q", fragment)
		}
	}
	if strings.Contains(query, "$3=20 OR") {
		t.Fatalf("workspace-admin role bypasses project visibility: %s", query)
	}
	if !strings.Contains(query, "'anchor','inbox_view','is_favorite','members','next_work_item_sequence'") {
		t.Error("API project list query does not remove non-serializer fields")
	}
	if !strings.Contains(query, "'sort_order', (SELECT member.sort_order FROM project_members member") {
		t.Error("API project list must preserve NULL sort_order for public projects without project membership")
	}
	if !strings.Contains(query, "ORDER BY sort_order ASC NULLS LAST, project_id ASC") || strings.Contains(query, "name ASC") {
		t.Fatalf("API project list must use the source sort key with deterministic primary-key tie order: %s", query)
	}
}

func TestAPIProjectListKeepsSourceSortPrimaryAndUsesStableIDTie(t *testing.T) {
	user := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	db := &workspaceProjectDatabase{rows: []pgx.Row{
		workspaceProjectRoleRow(20),
		workspaceProjectRow(func(dest ...any) error {
			*dest[0].(*[]byte) = []byte(`{"results":[]}`)
			*dest[1].(*bool) = true
			return nil
		}),
	}}
	repo, _ := NewRepository(db)
	if _, err := repo.APIProjectList(context.Background(), user, "space", APIProjectListOptions{}); err != nil {
		t.Fatal(err)
	}
	query := db.queries[1]
	if !strings.Contains(query, "ORDER BY sort_order ASC NULLS LAST, project_id ASC") || strings.Contains(query, "name ASC") {
		t.Fatalf("project list tie order must preserve source sort_order then use ID: %s", query)
	}
}

func TestAPIProjectDetailRestrictsWorkspaceAndVisibility(t *testing.T) {
	user := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	project := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	db := &workspaceProjectDatabase{rows: []pgx.Row{
		workspaceProjectRoleRow(20),
		workspaceProjectRow(func(dest ...any) error {
			*dest[0].(*[]byte) = []byte(`{"id":"00000000-0000-4000-8000-000000000003"}`)
			*dest[1].(*bool) = true
			return nil
		}),
	}}
	repo, _ := NewRepository(db)
	body, err := repo.APIProject(context.Background(), user, "space", project, APIProjectDetailOptions{})
	if err != nil || string(body) != `{"id":"00000000-0000-4000-8000-000000000003"}` {
		t.Fatalf("get API project: body=%s err=%v", body, err)
	}
	query := db.queries[1]
	for _, fragment := range []string{"workspace.slug=$2", "project.id=$3", "$4 IN (5,15,20)", "project.network=2 OR EXISTS", "member_user.is_bot=false", "'total_members'", "'total_cycles'", "'total_modules'"} {
		if !strings.Contains(query, fragment) {
			t.Errorf("full API project detail query is missing %q", fragment)
		}
	}
	if strings.Contains(query, "$4=20 OR") {
		t.Fatalf("workspace-admin role bypasses project visibility: %s", query)
	}
}

func TestAPIProjectListAndDetailVisibilityAllowsPublicProjectsForWorkspaceRoles(t *testing.T) {
	user := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	project := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	for _, role := range []int{5, 15, 20} {
		t.Run(fmt.Sprintf("role-%d", role), func(t *testing.T) {
			listDB := &workspaceProjectDatabase{rows: []pgx.Row{
				workspaceProjectRoleRow(role),
				workspaceProjectRow(func(dest ...any) error {
					*dest[0].(*[]byte) = []byte(`{"results":[]}`)
					*dest[1].(*bool) = true
					return nil
				}),
			}}
			listRepo, _ := NewRepository(listDB)
			if _, err := listRepo.APIProjectList(context.Background(), user, "space", APIProjectListOptions{}); err != nil {
				t.Fatal(err)
			}
			if listDB.args[1][2] != role || !strings.Contains(listDB.queries[1], "$3 IN (5,15,20)") || !strings.Contains(listDB.queries[1], "AND (project.network=2 OR EXISTS") {
				t.Fatalf("list visibility for workspace role %d: args=%#v query=%s", role, listDB.args[1], listDB.queries[1])
			}

			detailDB := &workspaceProjectDatabase{rows: []pgx.Row{
				workspaceProjectRoleRow(role),
				workspaceProjectRow(func(dest ...any) error {
					*dest[0].(*[]byte) = []byte(`{"id":"00000000-0000-4000-8000-000000000003"}`)
					*dest[1].(*bool) = true
					return nil
				}),
			}}
			detailRepo, _ := NewRepository(detailDB)
			if _, err := detailRepo.APIProject(context.Background(), user, "space", project, APIProjectDetailOptions{}); err != nil {
				t.Fatal(err)
			}
			if detailDB.args[1][3] != role || !strings.Contains(detailDB.queries[1], "$4 IN (5,15,20)") || !strings.Contains(detailDB.queries[1], "AND (project.network=2 OR EXISTS") {
				t.Fatalf("detail visibility for workspace role %d: args=%#v query=%s", role, detailDB.args[1], detailDB.queries[1])
			}
		})
	}
}

func TestAPIProjectProjectionFiltersFieldsAndExpandsWorkspace(t *testing.T) {
	for _, collection := range []bool{false, true} {
		t.Run(fmt.Sprint(collection), func(t *testing.T) {
			db := &workspaceProjectDatabase{rows: []pgx.Row{workspaceProjectRow(func(dest ...any) error {
				*dest[0].(*[]byte) = []byte(`{"id":"workspace-id","name":"Core Workspace","slug":"core"}`)
				return nil
			})}}
			repo, _ := NewRepository(db)
			input := json.RawMessage(`{"id":"project-id","name":"Alpha","workspace":"workspace-id","description":"omitted"}`)
			if collection {
				input = append(append([]byte(`{"total_results":1,"results":[`), input...), []byte(`]}`)...)
			}
			body, err := repo.projectProjection(context.Background(), input, "id,name,workspace", "workspace", collection)
			if err != nil {
				t.Fatal(err)
			}
			var actual map[string]any
			if err := json.Unmarshal(body, &actual); err != nil {
				t.Fatal(err)
			}
			if collection {
				if actual["total_results"] != float64(1) {
					t.Fatalf("pagination changed: %s", body)
				}
				actual = actual["results"].([]any)[0].(map[string]any)
			}
			if len(actual) != 3 || actual["id"] != "project-id" || actual["name"] != "Alpha" {
				t.Fatalf("wrong projected fields: %s", body)
			}
			workspace := actual["workspace"].(map[string]any)
			if workspace["slug"] != "core" || workspace["name"] != "Core Workspace" {
				t.Fatalf("missing workspace expansion: %s", body)
			}
		})
	}
}
