//go:build integration

package workitemtypes_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestTypeDiscoveryV2InvalidFiltersCannotBroadenResults(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, _ := seedDiscoveryItems(t, f)
	leaf := `{"type_id__exact":"` + f.task + `"}`
	for _, filter := range []string{leaf + " trailing", `{"type_id__gt":"` + f.task + `"}`, `{"not":{"not":{"not":{"not":{"not":` + leaf + `}}}}}`, `{"priority__invalid":"high"}`} {
		_, err := typeDiscoveryList(f, repo, "api_workspaces_by_slug_projects_by_project_id_v2_issues", f.project, f.owner, filter, "1:0:0", false)
		var invalid *issuequery.FilterError
		if !errors.As(err, &invalid) {
			t.Fatalf("invalid v2 filter accepted: %s, %v", filter, err)
		}
	}
}

func TestTypeDiscoveryRejectsForeignTypesAndProjectScope(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, _ := seedDiscoveryItems(t, f)
	foreignWorkspace, foreignType, foreignProject := uuid.NewString(), uuid.NewString(), uuid.NewString()
	f.exec(`INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Elsewhere',$2,'elsewhere','UTC','',now(),now())`, foreignWorkspace, f.owner)
	f.exec(`INSERT INTO issue_types(id,workspace_id,name,description,logo_props,is_active,is_default,mandatory,level,is_epic,created_at,updated_at) VALUES($1,$2,'Private foreign type','','{}',true,false,false,0,false,now(),now())`, foreignType, foreignWorkspace)
	f.addProject(foreignProject)
	f.exec(`UPDATE projects SET workspace_id=$2 WHERE id=$1`, foreignProject, foreignWorkspace)
	// Even an inconsistent retained membership cannot authorize a foreign project.
	f.exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,20,true,now(),now(),'{}','{}','{}',0)`, f.workspace, foreignProject, f.owner)
	_, err := f.repo.TypeIdentities(f.ctx, f.owner, "types", foreignProject)
	requireTypeStatus(t, err, 403)
	catalog, err := f.repo.TypeIdentities(f.ctx, f.owner, "types", "")
	if err != nil {
		t.Fatal(err)
	}
	if catalog.TypeIDs()[foreignType] {
		t.Fatal("foreign identity appeared in catalog")
	}
	_, err = typeDiscoveryList(f, repo, "api_workspaces_by_slug_projects_by_project_id_issues", f.project, f.owner, `{"type_id__exact":"`+foreignType+`"}`, "1:0:0", false)
	var invalid *issuequery.FilterError
	if !errors.As(err, &invalid) || invalid.Code != "invalid_filter_reference" {
		t.Fatalf("foreign predicate accepted: %v", err)
	}
}

func TestTypeDiscoveryPreservesGuestVisibilityAndArchivePermissions(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	f.exec(`UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	f.exec(`UPDATE issues SET created_by_id=$2 WHERE id=$1`, ids[0], f.member)
	filter := `{"type_id__exact":"` + f.task + `"}`
	result, err := typeDiscoveryList(f, repo, "api_workspaces_by_slug_projects_by_project_id_issues", f.project, f.member, filter, "1:0:0", false)
	if err != nil {
		t.Fatal(err)
	}
	if assertDiscoveryPage(t, result, 1) != ids[0] {
		t.Fatal("guest saw another creator's item")
	}
	f.exec(`UPDATE issues SET archived_at=now() WHERE id=$1`, ids[0])
	_, err = typeDiscoveryList(f, repo, "api_workspaces_by_slug_projects_by_project_id_archived_issues", f.project, f.member, filter, "1:0:0", false)
	var denied *djangoapi.HTTPError
	if !errors.As(err, &denied) || denied.Status != 403 {
		t.Fatalf("guest archive restriction lost: %v", err)
	}
	result, err = typeDiscoveryList(f, repo, "api_workspaces_by_slug_projects_by_project_id_archived_issues", f.project, f.owner, filter, "1:0:0", false)
	if err != nil {
		t.Fatal(err)
	}
	assertDiscoveryPage(t, result, 1)
}

func TestTypeDiscoveryWorkspaceListsExcludeDeletedProjects(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, _ := seedDiscoveryItems(t, f)
	f.exec(`UPDATE projects SET deleted_at=now() WHERE id=$1`, f.project)
	for _, filter := range []string{"", `{"type_id__exact":"` + f.task + `"}`} {
		for _, operation := range []string{"api_workspaces_by_slug_issues", "api_workspaces_by_slug_user_issues_by_user_id"} {
			result, err := typeDiscoveryList(f, repo, operation, "", f.owner, filter, "1:0:0", false)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(result.Body)
			if err != nil {
				t.Fatal(err)
			}
			var page struct {
				Total   int   `json:"total_results"`
				Results []any `json:"results"`
			}
			if err = json.Unmarshal(raw, &page); err != nil || page.Total != 0 || len(page.Results) != 0 {
				t.Fatalf("deleted project leaked: %s %v", raw, err)
			}
		}
	}
}

type discoveryCountDatabase struct {
	*pgx.Conn
	afterCount func() error
}

func (db discoveryCountDatabase) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := db.Conn.Begin(ctx)
	return discoveryCountTransaction{Tx: tx, afterCount: db.afterCount}, err
}

type discoveryCountTransaction struct {
	pgx.Tx
	afterCount func() error
}

func (tx discoveryCountTransaction) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	row := tx.Tx.QueryRow(ctx, sql, args...)
	if strings.HasPrefix(sql, "SELECT COUNT(*) FROM (") {
		return interleavedSchemaRow{Row: row, afterRead: tx.afterCount}
	}
	return row
}
func TestTypeDiscoveryRowsAndCountShareSnapshotDuringConcurrentDeletion(t *testing.T) {
	f := newProjectTypeFixture(t)
	_, ids := seedDiscoveryItems(t, f)
	concurrent, err := pgx.ConnectConfig(f.ctx, f.db.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := concurrent.Close(context.WithoutCancel(f.ctx)); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	changed := false
	db := discoveryCountDatabase{Conn: f.db, afterCount: func() error {
		if changed {
			return nil
		}
		changed = true
		_, writeErr := concurrent.Exec(f.ctx, `UPDATE issues SET deleted_at=now() WHERE id=$1`, ids[1])
		return writeErr
	}}
	repo, err := workitems.NewRepository(db, typeTestPublisher{}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	filter := `{"type_id__exact":"` + f.task + `"}`
	page, err := typeDiscoveryList(f, repo, "api_workspaces_by_slug_projects_by_project_id_issues", f.project, f.owner, filter, "1:0:0", false)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || assertDiscoveryPage(t, page, 2) != ids[1] {
		t.Fatal("count and rows used different snapshots")
	}
	next, err := typeDiscoveryList(f, repo, "api_workspaces_by_slug_projects_by_project_id_issues", f.project, f.owner, filter, "1:0:0", false)
	if err != nil {
		t.Fatal(err)
	}
	if assertDiscoveryPage(t, next, 1) != ids[0] {
		t.Fatal("new request did not see deletion")
	}
}
