//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/cyclereads"
)

func seedDiscoveryCycle(t *testing.T, f projectTypeFixture) (uuid.UUID, []string) {
	t.Helper()
	_, ids := seedDiscoveryItems(t, f)
	cycle := uuid.New()
	f.exec(`INSERT INTO cycles(id,name,description,owned_by_id,project_id,workspace_id,view_props,sort_order,progress_snapshot,logo_props,timezone,version,created_at,updated_at) VALUES($1,'Discovery','',$2,$3,$4,'{}',0,'{}','{}','UTC',1,now(),now())`, cycle, f.owner, f.project, f.workspace)
	for _, id := range ids {
		f.exec(`INSERT INTO cycle_issues(id,issue_id,cycle_id,project_id,workspace_id,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,now(),now())`, id, cycle, f.project, f.workspace)
	}
	return cycle, ids
}

func TestCycleTypeDiscoveryPaginationAndGroupedCounts(t *testing.T) {
	f := newProjectTypeFixture(t)
	cycle, _ := seedDiscoveryCycle(t, f)
	repo, err := cyclereads.NewRepository(f.db)
	if err != nil {
		t.Fatal(err)
	}
	actor, project := uuid.MustParse(f.owner), uuid.MustParse(f.project)
	options := cyclereads.IssueListOptions{PerPage: 1, ComplexFilter: `{"AND":[{"type_id__in":["` + f.task + `"]},{"NOT":{"type_id":"` + f.bug + `"}}]}`}
	first, err := repo.Issues(f.ctx, actor, "types", project, cycle, options)
	if err != nil {
		t.Fatal(err)
	}
	options.Cursor = "1:1:0"
	second, err := repo.Issues(f.ctx, actor, "types", project, cycle, options)
	if err != nil {
		t.Fatal(err)
	}
	if cycleDiscoveryPage(t, first, 2, 1) == cycleDiscoveryPage(t, second, 2, 1) {
		t.Fatal("cycle pages repeated the same item")
	}
	options.Cursor, options.GroupBy = "", "priority"
	grouped, err := repo.Issues(f.ctx, actor, "types", project, cycle, options)
	if err != nil {
		t.Fatal(err)
	}
	var group struct {
		Total   int  `json:"total_results"`
		Next    bool `json:"next_page_results"`
		Results map[string]struct {
			Results []struct {
				TypeID string `json:"type_id"`
			} `json:"results"`
			Total int `json:"total_results"`
		} `json:"results"`
	}
	if err = json.Unmarshal(grouped, &group); err != nil || group.Total != 2 || !group.Next || len(group.Results["none"].Results) != 1 || group.Results["none"].Total != 2 || group.Results["none"].Results[0].TypeID != f.task {
		t.Fatalf("group=%s error=%v", grouped, err)
	}
}

func cycleDiscoveryPage(t *testing.T, raw json.RawMessage, total, rows int) string {
	t.Helper()
	var page struct {
		Total   int `json:"total_results"`
		Results []struct {
			ID     string `json:"id"`
			TypeID string `json:"type_id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &page); err != nil || page.Total != total || len(page.Results) != rows {
		t.Fatalf("page=%s error=%v", raw, err)
	}
	if rows == 0 {
		return ""
	}
	if page.Results[0].TypeID == "" {
		t.Fatal("cycle list omitted the type identity")
	}
	return page.Results[0].ID
}

func TestCycleTypeDiscoveryPreservesRelationMultiplicityAndNegation(t *testing.T) {
	f := newProjectTypeFixture(t)
	cycle, ids := seedDiscoveryCycle(t, f)
	labels := []string{uuid.New().String(), uuid.New().String()}
	for _, label := range labels {
		f.exec(`INSERT INTO labels(id,name,description,color,sort_order,project_id,workspace_id,created_at,updated_at) VALUES($1::uuid,$1::text,'','#123456',0,$2,$3,now(),now())`, label, f.project, f.workspace)
		f.exec(`INSERT INTO issue_labels(id,issue_id,label_id,project_id,workspace_id,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,now(),now())`, ids[0], label, f.project, f.workspace)
	}
	repo, err := cyclereads.NewRepository(f.db)
	if err != nil {
		t.Fatal(err)
	}
	labelFilter := `{"label_id__in":"` + labels[0] + `,` + labels[1] + `"}`
	for _, scenario := range []struct {
		filter      string
		total, rows int
	}{
		{`{"and":[{"type_id":"` + f.task + `"},` + labelFilter + `]}`, 1, 2},
		{`{"oR":[{"type_id":"` + f.bug + `"},` + labelFilter + `]}`, 2, 3},
		{`{"AND":[{"type_id":"` + f.task + `"},{"Not":` + labelFilter + `}]}`, 1, 1},
		{`{"type_id__in":[]}`, 0, 0},
	} {
		result, readErr := repo.Issues(f.ctx, uuid.MustParse(f.owner), "types", uuid.MustParse(f.project), cycle, cyclereads.IssueListOptions{PerPage: 20, ComplexFilter: scenario.filter})
		if readErr != nil {
			t.Fatal(readErr)
		}
		cycleDiscoveryPage(t, result, scenario.total, scenario.rows)
	}
	f.mutate(t, "pause", f.bug)
	f.mutate(t, "remove", f.bug)
	retained, err := repo.Issues(f.ctx, uuid.MustParse(f.owner), "types", uuid.MustParse(f.project), cycle, cyclereads.IssueListOptions{PerPage: 20, ComplexFilter: `{"NOT":{"type_id":"` + f.task + `"}}`})
	if err != nil {
		t.Fatal(err)
	}
	cycleDiscoveryPage(t, retained, 1, 1)
}

func TestCycleTypeDiscoveryRejectsInvalidAndUnauthorizedFilters(t *testing.T) {
	f := newProjectTypeFixture(t)
	cycle, _ := seedDiscoveryCycle(t, f)
	repo, err := cyclereads.NewRepository(f.db)
	if err != nil {
		t.Fatal(err)
	}
	for _, filter := range []string{`{"type_id":"` + uuid.New().String() + `"}`, `{"type_id__contains":"` + f.task + `"}`, `{"type_id":"` + f.task + `"} trailing`} {
		_, err = repo.Issues(f.ctx, uuid.MustParse(f.owner), "types", uuid.MustParse(f.project), cycle, cyclereads.IssueListOptions{ComplexFilter: filter})
		var invalid *cyclereads.FilterError
		if !errors.As(err, &invalid) {
			t.Fatalf("expected cycle filter error: %v", err)
		}
	}
	options := cyclereads.IssueListOptions{ComplexFilter: `{"type_id":"` + f.task + `"}`}
	f.exec(`UPDATE project_members SET role=5 WHERE member_id=$1`, f.member)
	for _, actor := range []string{f.outsider, f.member} {
		_, err = repo.Issues(f.ctx, uuid.MustParse(actor), "types", uuid.MustParse(f.project), cycle, options)
		if !errors.Is(err, cyclereads.ErrForbidden) {
			t.Fatalf("expected forbidden: %v", err)
		}
	}
	f.exec(`UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, f.workspace)
	_, err = repo.Issues(f.ctx, uuid.MustParse(f.owner), "types", uuid.MustParse(f.project), cycle, options)
	var invalid *cyclereads.FilterError
	if !errors.As(err, &invalid) || invalid.Code != "invalid_filter_reference" {
		t.Fatalf("disabled feature: %v", err)
	}
}
