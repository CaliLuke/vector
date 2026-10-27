//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workspaceexports"
	"github.com/google/uuid"
)

func TestCustomExportHistoryRechecksCrossProjectReferences(t *testing.T) {
	for _, kind := range []string{"outgoing", "incoming", "child", "parent"} {
		t.Run(kind, func(t *testing.T) {
			f := newProjectTypeFixture(t)
			_, ids := seedDiscoveryItems(t, f)
			project := uuid.NewString()
			f.addProject(project)
			f.exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,15,true,now(),now(),'{}','{}','{}',0)`, f.workspace, project, f.owner)
			reference := addHierarchyItem(t, f, f.task, "")
			f.exec(`UPDATE issues SET project_id=$2 WHERE id=$1`, reference, project)
			connectExportReference(f, kind, ids[0], reference, project)
			input, artifact, generate := exportWorkerFixture(t, f, true)
			if err := generate(input); err != nil {
				t.Fatal(err)
			}
			files := exportJSONFiles(t, artifact)
			if len(files) != 1 || len(files[0]) != 3 {
				t.Fatalf("unselected reference became exported row: %+v", files)
			}
			assertExportReference(t, kind, files[0][0])
			var references []string
			if err := f.db.QueryRow(f.ctx, `SELECT rich_filters->'exported_item_ids' FROM exporters WHERE token=$1`, input.TokenID).Scan(&references); err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(references, reference) {
				t.Fatalf("%s missing from authorization metadata: %+v", kind, references)
			}
			assertCustomExportHistoryCount(t, f, 1)
			f.exec(`UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, project, f.owner)
			assertCustomExportHistoryCount(t, f, 0)
		})
	}
}

func connectExportReference(f projectTypeFixture, kind, issue, reference, project string) {
	switch kind {
	case "child":
		f.exec(`UPDATE issues SET parent_id=$2 WHERE id=$1`, reference, issue)
	case "parent":
		f.exec(`UPDATE issues SET parent_id=$2 WHERE id=$1`, issue, reference)
	case "outgoing":
		f.exec(`INSERT INTO issue_relations(id,workspace_id,project_id,issue_id,related_issue_id,relation_type,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,'relates_to',now(),now())`, f.workspace, f.project, issue, reference)
	case "incoming":
		f.exec(`INSERT INTO issue_relations(id,workspace_id,project_id,issue_id,related_issue_id,relation_type,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,'relates_to',now(),now())`, f.workspace, project, reference, issue)
	}
}

func assertExportReference(t *testing.T, kind string, row map[string]any) {
	t.Helper()
	switch kind {
	case "child":
		if row["sub_issues_count"] != float64(1) {
			t.Fatalf("child did not contribute to export: %+v", row)
		}
	case "parent":
		if row["parent"] == "" {
			t.Fatalf("parent did not contribute to export: %+v", row)
		}
	default:
		relations, ok := row["relations"].([]any)
		if !ok || len(relations) != 1 {
			t.Fatalf("relation did not contribute to export: %+v", row)
		}
		if relations[0].(map[string]any)["direction"] != kind {
			t.Fatalf("unexpected direction: %+v", relations)
		}
	}
}

func assertCustomExportHistoryCount(t *testing.T, f projectTypeFixture, want int) {
	t.Helper()
	repo, err := workspaceexports.NewRepository(f.db)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := repo.History(f.ctx, f.owner, "types", djangoapi.Query{Params: map[string]string{"per_page": "20", "cursor": "20:1:0"}})
	if err != nil {
		t.Fatal(err)
	}
	var page struct {
		Total   int               `json:"total_results"`
		Results []json.RawMessage `json:"results"`
	}
	if err = json.Unmarshal(raw, &page); err != nil {
		t.Fatal(err)
	}
	if page.Total != want || len(page.Results) != want {
		t.Fatalf("history count want %d: %s", want, raw)
	}
}
