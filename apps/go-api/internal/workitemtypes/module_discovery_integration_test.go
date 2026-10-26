//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/modulereads"
)

func seedDiscoveryModule(t *testing.T, f projectTypeFixture) uuid.UUID {
	t.Helper()
	_, ids := seedDiscoveryItems(t, f)
	module := uuid.New()
	f.exec(`INSERT INTO modules(id,name,description,status,project_id,workspace_id,view_props,sort_order,logo_props,created_at,updated_at) VALUES($1,'Discovery','','planned',$2,$3,'{}',0,'{}',now(),now())`, module, f.project, f.workspace)
	for _, id := range ids {
		f.exec(`INSERT INTO module_issues(id,issue_id,module_id,project_id,workspace_id,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,now(),now())`, id, module, f.project, f.workspace)
	}
	return module
}

func TestModuleTypeDiscoveryFiltersBeforePaginationAndGrouping(t *testing.T) {
	f := newProjectTypeFixture(t)
	module := seedDiscoveryModule(t, f)
	repo, err := modulereads.NewRepository(f.db)
	if err != nil {
		t.Fatal(err)
	}
	actor, project := uuid.MustParse(f.owner), uuid.MustParse(f.project)
	options := modulereads.IssueListOptions{Limit: 1, Filters: `{"type_id__in":["` + f.task + `"]}`}
	first, err := repo.Issues(f.ctx, actor, "types", project, module, options)
	if err != nil {
		t.Fatal(err)
	}
	options.Page = 1
	second, err := repo.Issues(f.ctx, actor, "types", project, module, options)
	if err != nil {
		t.Fatal(err)
	}
	firstID := moduleDiscoveryItem(t, first, 2)
	if firstID == moduleDiscoveryItem(t, second, 2) {
		t.Fatal("module pages repeated the same item")
	}
	options.Page, options.GroupBy = 0, "priority"
	grouped, err := repo.Issues(f.ctx, actor, "types", project, module, options)
	if err != nil || grouped.Total != 2 || grouped.GroupCounts["none"] != 2 || !grouped.GroupedHasNext {
		t.Fatalf("grouped=%+v error=%v", grouped, err)
	}
	f.mutate(t, "pause", f.bug)
	f.mutate(t, "remove", f.bug)
	options.GroupBy, options.Filters = "", `{"not":{"type_id":"`+f.task+`"}}`
	retained, err := repo.Issues(f.ctx, actor, "types", project, module, options)
	if err != nil {
		t.Fatal(err)
	}
	moduleDiscoveryItem(t, retained, 1)
}

func moduleDiscoveryItem(t *testing.T, page modulereads.IssuePage, total int) string {
	t.Helper()
	var items []struct {
		ID     string `json:"id"`
		TypeID string `json:"type_id"`
	}
	if err := json.Unmarshal(page.Items, &items); err != nil || len(items) != 1 || page.Total != total {
		t.Fatalf("page=%+v error=%v", page, err)
	}
	if items[0].TypeID == "" {
		t.Fatal("module list omitted the type identity")
	}
	return items[0].ID
}

func TestModuleTypeDiscoveryRejectsInvalidAndUnauthorizedFilters(t *testing.T) {
	f := newProjectTypeFixture(t)
	module := seedDiscoveryModule(t, f)
	repo, err := modulereads.NewRepository(f.db)
	if err != nil {
		t.Fatal(err)
	}
	options := modulereads.IssueListOptions{Limit: 10, Filters: `{"type_id":"` + uuid.New().String() + `"}`}
	_, err = repo.Issues(f.ctx, uuid.MustParse(f.owner), "types", uuid.MustParse(f.project), module, options)
	var filterError *issuequery.FilterError
	if !errors.As(err, &filterError) || filterError.Code != "invalid_filter_reference" {
		t.Fatalf("unknown selection: %v", err)
	}
	for _, actor := range []string{f.outsider, f.member} {
		f.exec(`UPDATE project_members SET role=5 WHERE member_id=$1`, f.member)
		_, err = repo.Issues(f.ctx, uuid.MustParse(actor), "types", uuid.MustParse(f.project), module, options)
		if !errors.Is(err, modulereads.ErrForbidden) {
			t.Fatalf("actor %s: %v", actor, err)
		}
	}
	options.Filters = `{"type_id":"` + f.task + `"}`
	f.exec(`UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, f.workspace)
	_, err = repo.Issues(f.ctx, uuid.MustParse(f.owner), "types", uuid.MustParse(f.project), module, options)
	if !errors.As(err, &filterError) || filterError.Code != "invalid_filter_reference" {
		t.Fatalf("disabled feature: %v", err)
	}
}
