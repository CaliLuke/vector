//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

type cyclePropertyFixture struct {
	projectTypeFixture
	cycle              uuid.UUID
	ids                []string
	field, calculation string
	repo               *cyclereads.Repository
}

func newCyclePropertyFixture(t *testing.T) cyclePropertyFixture {
	t.Helper()
	f := newProjectTypeFixture(t)
	cycle, ids := seedDiscoveryCycle(t, f)
	repo, err := cyclereads.NewRepository(f.db)
	if err != nil {
		t.Fatal(err)
	}
	field, calculation := uuid.New().String(), uuid.New().String()
	attachValueProperties(t, f,
		property.Definition{ID: field, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: calculation, Title: "Double", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{effort}} * 2", Result: "Number"}},
	)
	f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($1::text,CASE WHEN name='Discovery 1' THEN 9 ELSE 2 END),created_at='2026-01-01'::timestamptz + sequence_id * interval '1 day'`, field)
	return cyclePropertyFixture{f, cycle, ids, field, calculation, repo}
}

func (f cyclePropertyFixture) read(t *testing.T, options cyclereads.IssueListOptions) json.RawMessage {
	t.Helper()
	result, err := f.repo.Issues(f.ctx, uuid.MustParse(f.owner), "types", uuid.MustParse(f.project), f.cycle, options)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestCyclePropertyFiltersPrecedeCountsPagesAndGroups(t *testing.T) {
	f := newCyclePropertyFixture(t)
	options := cyclereads.IssueListOptions{PerPage: 1, OrderBy: "created_at"}
	if id := cycleDiscoveryPage(t, f.read(t, options), 3, 1); id != f.ids[0] {
		t.Fatalf("first item: %s", id)
	}
	options.ComplexFilter = fmt.Sprintf(`{"customproperty_%s__gte":8}`, f.field)
	if id := cycleDiscoveryPage(t, f.read(t, options), 1, 1); id != f.ids[1] {
		t.Fatalf("match beyond first page lost: %s", id)
	}
	options.ComplexFilter = fmt.Sprintf(`{"customproperty_%s__exact":18}`, f.calculation)
	cycleDiscoveryPage(t, f.read(t, options), 1, 1)
	options.ComplexFilter = fmt.Sprintf(`{"or":[{"customproperty_%s__exact":2},{"customproperty_%s__exact":9}]}`, f.field, f.field)
	cycleDiscoveryPage(t, f.read(t, options), 2, 1)
	options.ComplexFilter = fmt.Sprintf(`{"customproperty_%s__gte":0}`, f.field)
	first := cycleDiscoveryPage(t, f.read(t, options), 2, 1)
	options.Cursor = "1:1:0"
	if second := cycleDiscoveryPage(t, f.read(t, options), 2, 1); first == second {
		t.Fatal("filtered pages repeated an item")
	}
	options.Cursor, options.GroupBy = "", "priority"
	var grouped struct {
		Total   int  `json:"total_results"`
		Next    bool `json:"next_page_results"`
		Results map[string]struct {
			Total   int               `json:"total_results"`
			Results []json.RawMessage `json:"results"`
		} `json:"results"`
	}
	raw := f.read(t, options)
	if err := json.Unmarshal(raw, &grouped); err != nil || grouped.Total != 2 || !grouped.Next || grouped.Results["none"].Total != 2 || len(grouped.Results["none"].Results) != 1 {
		t.Fatalf("filtered groups: %s %v", raw, err)
	}
}

func TestCyclePropertyFiltersPreserveNullableNegationAndRelationRows(t *testing.T) {
	f := newCyclePropertyFixture(t)
	condition := fmt.Sprintf(`{"customproperty_%s__gte":8}`, f.field)
	options := cyclereads.IssueListOptions{PerPage: 20, ComplexFilter: `{"not":` + condition + `}`}
	if id := cycleDiscoveryPage(t, f.read(t, options), 1, 1); id != f.ids[0] {
		t.Fatalf("unknown included by negation: %s", id)
	}
	options.ComplexFilter = `{"not":{"or":[` + condition + `,{"target_date":"2026-01-01"}]}}`
	cycleDiscoveryPage(t, f.read(t, options), 0, 0)
	f.exec(`UPDATE issues SET target_date='2026-02-01' WHERE id=$1`, f.ids[0])
	cycleDiscoveryPage(t, f.read(t, options), 1, 1)
	labels := []string{uuid.New().String(), uuid.New().String()}
	for _, label := range labels {
		f.exec(`INSERT INTO labels(id,name,description,color,sort_order,project_id,workspace_id,created_at,updated_at) VALUES($1::uuid,$1::text,'','#123456',0,$2,$3,now(),now())`, label, f.project, f.workspace)
		f.exec(`INSERT INTO issue_labels(id,issue_id,label_id,project_id,workspace_id,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,now(),now())`, f.ids[1], label, f.project, f.workspace)
	}
	options.ComplexFilter = `{"and":[` + condition + `,{"label_id__in":"` + labels[0] + `,` + labels[1] + `"}]}`
	cycleDiscoveryPage(t, f.read(t, options), 1, 2)
	f.exec(`UPDATE issues SET priority='urgent' WHERE id=$1`, f.ids[2])
	options.ComplexFilter = `{"or":[` + condition + `,{"priority":"urgent"}]}`
	cycleDiscoveryPage(t, f.read(t, options), 2, 2)
	options.Filters = map[string]string{"priority": "urgent"}
	cycleDiscoveryPage(t, f.read(t, options), 1, 1)
	f.exec(`UPDATE cycle_issues SET deleted_at=now() WHERE issue_id=$1`, f.ids[2])
	cycleDiscoveryPage(t, f.read(t, options), 0, 0)
	// The same surrounding scope must hold without any custom-property leaf.
	options.ComplexFilter = `{"or":[{"priority":"low"},{"priority":"urgent"}]}`
	cycleDiscoveryPage(t, f.read(t, options), 0, 0)
	foreignWorkspace, foreignProject := uuid.New().String(), uuid.New().String()
	f.exec(`INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Elsewhere',$2,'elsewhere','UTC','',now(),now())`, foreignWorkspace, f.owner)
	f.exec(`INSERT INTO projects(id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at) VALUES($1,'Foreign','',0,$2,'FOREIGN',false,true,false,true,false,false,false,false,0,0,'{}','UTC',now(),now())`, foreignProject, foreignWorkspace)
	foreignItem, foreignCycle := uuid.New().String(), uuid.New().String()
	f.exec(`INSERT INTO issues(id,workspace_id,project_id,name,description_json,description_html,priority,sort_order,sequence_id,is_draft,created_at,updated_at,created_by_id) VALUES($1,$2,$3,'Foreign item','{}','','urgent',0,1,false,now(),now(),$4)`, foreignItem, foreignWorkspace, foreignProject, f.owner)
	f.exec(`INSERT INTO cycles(id,name,description,owned_by_id,project_id,workspace_id,view_props,sort_order,progress_snapshot,logo_props,timezone,version,created_at,updated_at) VALUES($1,'Foreign','',$2,$3,$4,'{}',0,'{}','{}','UTC',1,now(),now())`, foreignCycle, f.owner, foreignProject, foreignWorkspace)
	f.exec(`INSERT INTO cycle_issues(id,issue_id,cycle_id,project_id,workspace_id,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,now(),now())`, foreignItem, foreignCycle, foreignProject, foreignWorkspace)
	options.Filters = nil
	for _, scenario := range []struct {
		filter string
		want   int
	}{
		{`{"or":[` + condition + `,{"priority":"urgent"}]}`, 1},
		{`{"or":[{"priority":"low"},{"priority":"urgent"}]}`, 0},
	} {
		options.ComplexFilter = scenario.filter
		cycleDiscoveryPage(t, f.read(t, options), scenario.want, scenario.want)
		options.GroupBy = "priority"
		var grouped struct {
			Total int `json:"total_results"`
		}
		if err := json.Unmarshal(f.read(t, options), &grouped); err != nil || grouped.Total != scenario.want {
			t.Fatalf("foreign workspace escaped grouped scope: %+v %v", grouped, err)
		}
		options.GroupBy = ""
	}
}

func TestCyclePropertyFiltersAuthorizeBeforeValidatingReferences(t *testing.T) {
	f := newCyclePropertyFixture(t)
	options := cyclereads.IssueListOptions{PerPage: 10, ComplexFilter: fmt.Sprintf(`{"customproperty_%s__exact":4}`, uuid.New().String())}
	project := uuid.MustParse(f.project)
	f.exec(`UPDATE project_members SET role=5 WHERE member_id=$1`, f.member)
	for _, actor := range []string{f.member, f.outsider} {
		_, err := f.repo.Issues(f.ctx, uuid.MustParse(actor), "types", project, f.cycle, options)
		if !errors.Is(err, cyclereads.ErrForbidden) {
			t.Fatalf("actor not denied before validation: %v", err)
		}
	}
	_, err := f.repo.Issues(f.ctx, uuid.MustParse(f.owner), "types", project, f.cycle, options)
	var invalid *cyclereads.FilterError
	if !errors.As(err, &invalid) || invalid.Code != "invalid_filter_reference" {
		t.Fatalf("invalid property: %v", err)
	}
	options.ComplexFilter = fmt.Sprintf(`{"customproperty_%s__gte":0}`, f.field)
	f.exec(`UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, f.workspace)
	_, err = f.repo.Issues(f.ctx, uuid.MustParse(f.owner), "types", project, f.cycle, options)
	if !errors.As(err, &invalid) || invalid.Code != "invalid_filter_reference" {
		t.Fatalf("disabled property accepted: %v", err)
	}
}
