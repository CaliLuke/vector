//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/go-plane/internal/modulereads"
	"github.com/CaliLuke/go-plane/internal/workitems"
	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
)

func TestDisabledPropertyProjectionPreservesWorkItemReads(t *testing.T) {
	f := newProjectTypeFixture(t)
	repository, _ := seedDiscoveryItems(t, f)
	f.exec(`UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, f.workspace)
	for _, operation := range []string{"api_workspaces_by_slug_projects_by_project_id_issues", "api_workspaces_by_slug_projects_by_project_id_v2_issues", "api_workspaces_by_slug_issues"} {
		t.Run(operation, func(t *testing.T) {
			page := readPropertyPage(t, f, repository, operation, f.owner, map[string]string{"include_properties": "true", "per_page": "10"}, false)
			if page.Total != 3 || len(page.Results) != 3 {
				t.Fatalf("disabled projection changed page: %+v", page)
			}
			for _, row := range page.Results {
				if row.Values == nil || len(row.Values) != 0 || row.EvaluatedAt.IsZero() {
					t.Fatalf("disabled projection not explicitly empty: %+v", row)
				}
			}
			for _, filter := range []string{fmt.Sprintf(`{"type_id":"%s"}`, f.task), fmt.Sprintf(`{"customproperty_%s__exact":0}`, uuid.New().String())} {
				project := ""
				if operation != "api_workspaces_by_slug_issues" {
					project = f.project
				}
				_, err := repository.Execute(f.ctx, workitems.Request{Operation: operation, Method: "GET", ActorID: f.owner, Slug: "types", ProjectID: project, Params: map[string]string{"include_properties": "true", "filters": filter}}, nil, "")
				if err == nil {
					t.Fatalf("disabled filter accepted: %s", filter)
				}
			}
		})
	}
}

func TestDisabledPropertyProjectionPreservesRelationReads(t *testing.T) {
	t.Run("module", func(t *testing.T) {
		f := newModulePropertyFixture(t)
		f.exec(`UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, f.workspace)
		assertDisabledRelationProjection(t, f.projectTypeFixture, func(actor, filter string) (json.RawMessage, error) {
			page, err := f.repo.Issues(f.ctx, uuid.MustParse(actor), "types", uuid.MustParse(f.project), f.module, modulereads.IssueListOptions{Limit: 10, IncludeProperties: true, Filters: filter})
			return page.Items, err
		})
	})
	t.Run("cycle", func(t *testing.T) {
		f := newCyclePropertyFixture(t)
		f.exec(`UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, f.workspace)
		assertDisabledRelationProjection(t, f.projectTypeFixture, func(actor, filter string) (json.RawMessage, error) {
			raw, err := f.repo.Issues(f.ctx, uuid.MustParse(actor), "types", uuid.MustParse(f.project), f.cycle, cyclereads.IssueListOptions{PerPage: 10, IncludeProperties: true, ComplexFilter: filter})
			if err != nil {
				return nil, err
			}
			var envelope struct {
				Results json.RawMessage `json:"results"`
			}
			err = json.Unmarshal(raw, &envelope)
			return envelope.Results, err
		})
	})
}

func assertDisabledRelationProjection(t *testing.T, f projectTypeFixture, read func(string, string) (json.RawMessage, error)) {
	t.Helper()
	raw, err := read(f.owner, `{"priority":"none"}`)
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Values map[string]types.ProjectedValue `json:"property_values"`
		At     time.Time                       `json:"properties_evaluated_at"`
	}
	if err = json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("disabled projection changed relation rows: %s", raw)
	}
	for _, row := range rows {
		if row.Values == nil || len(row.Values) != 0 || row.At.IsZero() || row.At != rows[0].At {
			t.Fatalf("disabled relation values: %s", raw)
		}
	}
	for _, filter := range []string{fmt.Sprintf(`{"type_id":"%s"}`, f.task), fmt.Sprintf(`{"customproperty_%s__exact":0}`, uuid.New().String()), `{"unknown_field":"x"}`} {
		if _, err := read(f.owner, filter); err == nil {
			t.Fatalf("disabled/invalid filter accepted: %s", filter)
		}
	}
	if _, err := read(f.outsider, ""); err == nil {
		t.Fatal("disabled projection bypassed access")
	}
}
