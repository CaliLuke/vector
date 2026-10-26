//go:build integration

package workitemtypes_test

import (
	"errors"
	"fmt"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/modulereads"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

type modulePropertyFixture struct {
	projectTypeFixture
	module             uuid.UUID
	repo               *modulereads.Repository
	field, calculation string
	ids                []string
}

func newModulePropertyFixture(t *testing.T) modulePropertyFixture {
	t.Helper()
	f := newProjectTypeFixture(t)
	module := seedDiscoveryModule(t, f)
	repo, err := modulereads.NewRepository(f.db)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := f.db.Query(f.ctx, `SELECT id::text FROM issues ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	field, calculation := uuid.New().String(), uuid.New().String()
	attachValueProperties(t, f,
		property.Definition{ID: field, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: calculation, Title: "Double", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{effort}} * 2", Result: "Number"}},
	)
	f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($1::text,CASE WHEN name='Discovery 1' THEN 9 ELSE 2 END),created_at='2026-01-01'::timestamptz + sequence_id * interval '1 day'`, field)
	return modulePropertyFixture{projectTypeFixture: f, module: module, repo: repo, field: field, calculation: calculation, ids: ids}
}

func (f modulePropertyFixture) read(t *testing.T, options modulereads.IssueListOptions) modulereads.IssuePage {
	t.Helper()
	page, err := f.repo.Issues(f.ctx, uuid.MustParse(f.owner), "types", uuid.MustParse(f.project), f.module, options)
	if err != nil {
		t.Fatal(err)
	}
	return page
}

func TestModulePropertyFiltersPrecedeCountsPagesAndGroups(t *testing.T) {
	f := newModulePropertyFixture(t)
	options := modulereads.IssueListOptions{Limit: 1, OrderBy: "created_at"}
	if id := moduleDiscoveryItem(t, f.read(t, options), 3); id != f.ids[0] {
		t.Fatalf("unexpected first unfiltered item: %s", id)
	}
	options.Filters = fmt.Sprintf(`{"customproperty_%s__gte":8}`, f.field)
	if id := moduleDiscoveryItem(t, f.read(t, options), 1); id != f.ids[1] {
		t.Fatalf("match beyond first page was lost: %s", id)
	}
	options.Filters = fmt.Sprintf(`{"customproperty_%s__exact":18}`, f.calculation)
	if id := moduleDiscoveryItem(t, f.read(t, options), 1); id != f.ids[1] {
		t.Fatalf("formula filter disagrees: %s", id)
	}
	options.Filters = fmt.Sprintf(`{"customproperty_%s__gte":0}`, f.field)
	first := moduleDiscoveryItem(t, f.read(t, options), 2)
	options.Page = 1
	second := moduleDiscoveryItem(t, f.read(t, options), 2)
	if first == second {
		t.Fatal("filtered pages repeated an item")
	}
	options.Page, options.GroupBy = 0, "priority"
	grouped := f.read(t, options)
	if grouped.Total != 2 || grouped.GroupCounts["none"] != 2 || !grouped.GroupedHasNext {
		t.Fatalf("filtered groups: %+v", grouped)
	}
}

func TestModulePropertyFiltersKeepUnknownAndWholeDisjunction(t *testing.T) {
	f := newModulePropertyFixture(t)
	condition := fmt.Sprintf(`{"customproperty_%s__gte":8}`, f.field)
	options := modulereads.IssueListOptions{Limit: 1, Filters: `{"not":` + condition + `}`}
	if id := moduleDiscoveryItem(t, f.read(t, options), 1); id != f.ids[0] {
		t.Fatalf("negative property matched unattached type: %s", id)
	}
	f.exec(`UPDATE issues SET priority='urgent' WHERE id=$1`, f.ids[2])
	options.Filters = `{"or":[` + condition + `,{"priority":"urgent"}]}`
	options.OrderBy = "created_at"
	if id := moduleDiscoveryItem(t, f.read(t, options), 2); id != f.ids[1] {
		t.Fatalf("custom branch missing: %s", id)
	}
	options.Page = 1
	if id := moduleDiscoveryItem(t, f.read(t, options), 2); id != f.ids[2] {
		t.Fatalf("legacy branch missing: %s", id)
	}
	options.Page, options.Priority = 0, "urgent"
	moduleDiscoveryItem(t, f.read(t, options), 1)
	f.exec(`UPDATE module_issues SET deleted_at=now() WHERE issue_id=$1`, f.ids[2])
	page := f.read(t, options)
	if page.Total != 0 || string(page.Items) != "[]" {
		t.Fatalf("removed module relation entered candidate set: %+v", page)
	}
}

func TestModulePropertyFiltersRespectScopeAndFeatureState(t *testing.T) {
	f := newModulePropertyFixture(t)
	options := modulereads.IssueListOptions{Limit: 10, Filters: fmt.Sprintf(`{"customproperty_%s__exact":4}`, uuid.New().String())}
	project := uuid.MustParse(f.project)
	f.exec(`UPDATE project_members SET role=5 WHERE member_id=$1`, f.member)
	for _, actor := range []string{f.member, f.outsider} {
		_, err := f.repo.Issues(f.ctx, uuid.MustParse(actor), "types", project, f.module, options)
		if !errors.Is(err, modulereads.ErrForbidden) {
			t.Fatalf("actor %s was not denied before filter validation: %v", actor, err)
		}
	}
	_, err := f.repo.Issues(f.ctx, uuid.MustParse(f.owner), "types", project, f.module, options)
	var invalid *issuequery.FilterError
	if !errors.As(err, &invalid) || invalid.Code != "invalid_filter_reference" {
		t.Fatalf("invalid property: %v", err)
	}
	options.Filters = fmt.Sprintf(`{"customproperty_%s__gte":0}`, f.field)
	f.exec(`UPDATE project_members SET role=5 WHERE member_id=$1`, f.owner)
	if page := f.read(t, options); page.Total != 2 {
		t.Fatalf("workspace administrator exception lost: %+v", page)
	}
	f.exec(`UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, f.workspace)
	_, err = f.repo.Issues(f.ctx, uuid.MustParse(f.owner), "types", project, f.module, options)
	if !errors.As(err, &invalid) || invalid.Code != "invalid_filter_reference" {
		t.Fatalf("disabled property filter was accepted: %v", err)
	}
}
