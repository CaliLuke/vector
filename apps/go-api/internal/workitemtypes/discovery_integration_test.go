//go:build integration

package workitemtypes_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/workitems"
	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestTypeDiscoveryCatalogScopeAndRetainedIdentities(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.exec(`UPDATE issue_types SET is_active=false WHERE id=$1`, f.bug)
	for _, project := range []string{"", f.project} {
		catalog, err := f.repo.TypeIdentities(f.ctx, f.member, "types", project)
		if err != nil || !catalog.Enabled || len(catalog.Types) != 2 || !catalog.TypeIDs()[f.bug] {
			t.Fatalf("catalog=%+v error=%v", catalog, err)
		}
		for _, identity := range catalog.Types {
			if identity.ID == f.bug && identity.Active {
				t.Fatal("paused identity became active")
			}
		}
		raw, err := json.Marshal(catalog)
		if err != nil {
			t.Fatal(err)
		}
		for _, private := range []string{"usage", "project_count", "can_manage", "description"} {
			if strings.Contains(string(raw), `"`+private+`"`) {
				t.Fatalf("discovery exposed %s: %s", private, raw)
			}
		}
	}
	for _, scope := range [][3]string{{f.outsider, "types", ""}, {f.member, "missing", ""}, {f.member, "types", uuid.NewString()}} {
		_, err := f.repo.TypeIdentities(f.ctx, scope[0], scope[1], scope[2])
		requireTypeStatus(t, err, 403)
	}
	f.exec(`UPDATE projects SET deleted_at=now() WHERE id=$1`, f.project)
	_, err := f.repo.TypeIdentities(f.ctx, f.member, "types", f.project)
	requireTypeStatus(t, err, 403)
	f.exec(`UPDATE workspace_members SET is_active=false WHERE workspace_id=$1 AND member_id=$2`, f.workspace, f.member)
	_, err = f.repo.TypeIdentities(f.ctx, f.member, "types", "")
	requireTypeStatus(t, err, 403)
}

func TestTypeDiscoveryRejectsOverflowAndReturnsDisabledEmptyCatalog(t *testing.T) {
	f := newTypeFixture(t)
	repo := types.NewRepository(f.db)
	before, err := repo.TypeIdentities(f.ctx, f.member, "types", "")
	if err != nil || before.Enabled || len(before.Types) != 0 {
		t.Fatalf("disabled catalog=%+v error=%v", before, err)
	}
	f.exec(`INSERT INTO issue_types(id,workspace_id,name,description,logo_props,is_active,is_default,mandatory,level,is_epic,created_at,updated_at) SELECT gen_random_uuid(),$1,'Type '||n,'','{}',false,false,false,0,false,now(),now() FROM generate_series(1,1001) n`, f.workspace)
	catalog, err := repo.TypeIdentities(f.ctx, f.member, "types", "")
	requireTypeStatus(t, err, 422)
	if len(catalog.Types) != 0 {
		t.Fatal("overflow returned a truncated catalog")
	}
}

func TestTypeDiscoveryCatalogAndCallerReadUseOneSnapshot(t *testing.T) {
	f := newProjectTypeFixture(t)
	concurrent, err := pgx.ConnectConfig(f.ctx, f.db.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := concurrent.Close(context.WithoutCancel(f.ctx)); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	before := f.catalog(t)
	_, err = types.WithTypeDiscoveryRead(f.ctx, f.db, f.member, "types", f.project, func(tx pgx.Tx, catalog types.TypeIdentityCatalog) (bool, error) {
		if catalog.Revision != before.Revision {
			t.Fatal("unexpected initial revision")
		}
		_, writeErr := concurrent.Exec(f.ctx, `UPDATE issue_types SET name='Renamed concurrently' WHERE id=$1;`, f.bug)
		if writeErr != nil {
			return false, writeErr
		}
		var name string
		if readErr := tx.QueryRow(f.ctx, `SELECT name FROM issue_types WHERE id=$1`, f.bug).Scan(&name); readErr != nil {
			return false, readErr
		}
		if name != "Bug" {
			t.Fatalf("mixed discovery snapshots: %s", name)
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	after, err := f.repo.TypeIdentities(f.ctx, f.member, "types", f.project)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range after.Types {
		if item.Name == "Renamed concurrently" {
			found = true
		}
	}
	if !found {
		t.Fatal("new request did not see committed catalog change")
	}
}

func seedDiscoveryItems(t *testing.T, f projectTypeFixture) (*workitems.Repository, []string) {
	t.Helper()
	f.mutate(t, "import", f.bug)
	repo, err := workitems.NewRepository(f.db, typeTestPublisher{}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for index, typeID := range []string{f.task, f.task, f.bug} {
		result, createErr := repo.Execute(f.ctx, workitems.Request{Operation: "api_v1_workspaces_by_slug_projects_by_project_id_issues", Method: "POST", ActorID: f.owner, Slug: "types", ProjectID: f.project, API: true, Timezone: "UTC", Params: map[string]string{}}, strings.NewReader(fmt.Sprintf(`{"name":"Discovery %d","type_id":"%s"}`, index, typeID)), "application/json")
		if createErr != nil {
			t.Fatal(createErr)
		}
		ids = append(ids, resultIssueID(t, result))
	}
	return repo, ids
}

func typeDiscoveryList(f projectTypeFixture, repo *workitems.Repository, operation, project, actor, filter, cursor string, api bool) (workitems.Result, error) {
	return repo.Execute(f.ctx, workitems.Request{Operation: operation, Method: "GET", ActorID: actor, Slug: "types", ProjectID: project, API: api, Timezone: "UTC", Params: map[string]string{"filters": filter, "per_page": "1", "cursor": cursor, "user_id": f.owner}, Present: map[string]bool{"filters": true, "per_page": true, "cursor": cursor != ""}}, nil, "")
}

func assertDiscoveryPage(t *testing.T, result workitems.Result, total int) string {
	t.Helper()
	raw, err := json.Marshal(result.Body)
	if err != nil {
		t.Fatal(err)
	}
	var page struct {
		Total   int `json:"total_results"`
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	if err = json.Unmarshal(raw, &page); err != nil || page.Total != total || len(page.Results) != 1 {
		t.Fatalf("page=%s error=%v", raw, err)
	}
	return page.Results[0].ID
}

func TestAuthenticatedTypeFilterListsAndPagination(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, _ := seedDiscoveryItems(t, f)
	filter := `{"type_id__exact":"` + f.task + `"}`
	for _, operation := range []string{"api_workspaces_by_slug_projects_by_project_id_issues", "api_workspaces_by_slug_projects_by_project_id_v2_issues", "api_workspaces_by_slug_user_issues_by_user_id", "api_workspaces_by_slug_issues"} {
		project := f.project
		if !strings.Contains(operation, "_projects_") {
			project = ""
		}
		first, err := typeDiscoveryList(f, repo, operation, project, f.owner, filter, "1:0:0", false)
		if err != nil {
			t.Fatal(err)
		}
		second, err := typeDiscoveryList(f, repo, operation, project, f.owner, filter, "1:1:0", false)
		if err != nil {
			t.Fatal(err)
		}
		if assertDiscoveryPage(t, first, 2) == assertDiscoveryPage(t, second, 2) {
			t.Fatal("pagination returned the same item twice")
		}
	}
	f.mutate(t, "pause", f.bug)
	f.mutate(t, "remove", f.bug)
	result, err := typeDiscoveryList(f, repo, "api_workspaces_by_slug_projects_by_project_id_issues", f.project, f.owner, `{"not":{"type_id__in":["`+f.task+`"]}}`, "1:0:0", false)
	if err != nil {
		t.Fatal(err)
	}
	assertDiscoveryPage(t, result, 1)
}

func TestAuthenticatedTypeFilterRejectsUnknownAndUnauthorizedSelections(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, _ := seedDiscoveryItems(t, f)
	operation := "api_workspaces_by_slug_projects_by_project_id_issues"
	filter := `{"type_id__exact":"` + uuid.NewString() + `"}`
	_, err := typeDiscoveryList(f, repo, operation, f.project, f.owner, filter, "", false)
	var invalid *issuequery.FilterError
	if !errors.As(err, &invalid) || invalid.Code != "invalid_filter_reference" {
		t.Fatalf("unknown selection: %v", err)
	}
	_, err = typeDiscoveryList(f, repo, operation, f.project, f.outsider, filter, "", false)
	var denied *djangoapi.HTTPError
	if !errors.As(err, &denied) || denied.Status != 403 {
		t.Fatalf("access mapping: %v", err)
	}
	_, err = typeDiscoveryList(f, repo, "api_v1_workspaces_by_slug_projects_by_project_id_issues", f.project, f.owner, filter, "", true)
	if !errors.As(err, &denied) || denied.Status != 400 {
		t.Fatalf("API filter contract changed: %v", err)
	}
	f.exec(`UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, f.workspace)
	_, err = typeDiscoveryList(f, repo, operation, f.project, f.owner, `{"type_id__exact":"`+f.task+`"}`, "", false)
	if !errors.As(err, &denied) || denied.Status != 400 {
		t.Fatalf("disabled filter silently broadened: %v", err)
	}
}
