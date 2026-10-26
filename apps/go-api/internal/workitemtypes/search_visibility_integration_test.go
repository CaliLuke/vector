//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"slices"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/CaliLuke/go-plane/internal/workspacesearch"
)

func TestWorkspaceItemSearchUsesCurrentItemVisibility(t *testing.T) {
	t.Run("title", func(t *testing.T) { testWorkspaceSearchVisibility(t, false) })
	t.Run("rich text only", func(t *testing.T) { testWorkspaceSearchVisibility(t, true) })
}
func testWorkspaceSearchVisibility(t *testing.T, richText bool) {
	f := newProjectTypeFixture(t)
	body := `{"name":"Search acceptance"}`
	if richText {
		field := uuid.New().String()
		attachValueProperties(t, f, property.Definition{ID: field, Title: "Notes", Kind: property.RichText, Configuration: property.Configuration{Active: true}})
		raw, _ := json.Marshal(map[string]any{"name": "Ordinary title", "custom_properties": map[string]any{field: "<p>Search acceptance</p>"}})
		body = string(raw)
	}
	writer, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 2)
	for n := range ids {
		result, writeErr := writeValueItem(f, writer, "POST", "", body)
		if writeErr != nil {
			t.Fatal(writeErr)
		}
		ids[n] = resultIssueID(t, result)
	}
	own, other := ids[0], ids[1]
	f.exec(`UPDATE issues SET created_by_id=$2,created_at=now()+interval '1 minute' WHERE id=$1`, other, f.owner)
	intake := uuid.New().String()
	f.exec(`INSERT INTO intakes(id,workspace_id,project_id,name,description,is_default,view_props,logo_props,created_at,updated_at) VALUES($1,$2,$3,'Intake','',true,'{}','{}',now(),now())`, intake, f.workspace, f.project)
	for _, id := range ids {
		f.exec(`INSERT INTO intake_issues(id,workspace_id,project_id,intake_id,issue_id,status,extra,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,0,'{}',now(),now())`, f.workspace, f.project, intake, id)
	}
	reader, err := workspacesearch.NewRepository(f.db)
	if err != nil {
		t.Fatal(err)
	}
	check := func(t *testing.T, want ...string) {
		t.Helper()
		for _, entity := range []bool{false, true} {
			for _, project := range []string{"", f.project} {
				var raw json.RawMessage
				if entity {
					raw, err = reader.Entity(f.ctx, uuid.MustParse(f.member), "types", "Search acceptance", "issue", project, 100)
				} else {
					workspace := "true"
					if project != "" {
						workspace = "false"
					}
					raw, err = reader.Global(f.ctx, uuid.MustParse(f.member), "types", "Search acceptance", "issue,intake", project, workspace)
				}
				if err != nil {
					t.Fatal(err)
				}
				var result map[string]json.RawMessage
				if err = json.Unmarshal(raw, &result); err != nil {
					t.Fatal(err)
				}
				if !entity {
					if err = json.Unmarshal(result["results"], &result); err != nil {
						t.Fatal(err)
					}
				}
				kinds := []string{"issue"}
				if !entity {
					kinds = append(kinds, "intake")
				}
				for _, kind := range kinds {
					var rows []struct {
						ID string `json:"id"`
					}
					if err = json.Unmarshal(result[kind], &rows); err != nil {
						t.Fatal(err)
					}
					got := make([]string, len(rows))
					for n, row := range rows {
						got[n] = row.ID
					}
					slices.Sort(got)
					expected := slices.Clone(want)
					slices.Sort(expected)
					if !slices.Equal(got, expected) {
						t.Fatalf("entity=%v project=%q kind=%s: got %v want %v", entity, project, kind, got, expected)
					}
				}
			}
		}
	}
	check(t, own, other)
	f.exec(`UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	check(t, own)
	// The inaccessible newer item must not consume the result limit.
	raw, err := reader.Entity(f.ctx, uuid.MustParse(f.member), "types", "Search acceptance", "issue", f.project, 1)
	if err != nil {
		t.Fatal(err)
	}
	var limited struct {
		Issues []struct {
			ID string `json:"id"`
		} `json:"issue"`
	}
	if err = json.Unmarshal(raw, &limited); err != nil || len(limited.Issues) != 1 || limited.Issues[0].ID != own {
		t.Fatalf("visibility before limit: %s, %v", raw, err)
	}
	f.exec(`UPDATE projects SET guest_view_all_features=true WHERE id=$1`, f.project)
	check(t, own, other)
	for _, change := range []struct{ name, disable, restore string }{
		{"workspace inactive", `UPDATE workspace_members SET is_active=false WHERE workspace_id=$1 AND member_id=$2`, `UPDATE workspace_members SET is_active=true WHERE workspace_id=$1 AND member_id=$2`},
		{"workspace membership deleted", `UPDATE workspace_members SET deleted_at=now() WHERE workspace_id=$1 AND member_id=$2`, `UPDATE workspace_members SET deleted_at=NULL WHERE workspace_id=$1 AND member_id=$2`},
		{"project inactive", `UPDATE project_members SET is_active=false WHERE workspace_id=$1 AND member_id=$2`, `UPDATE project_members SET is_active=true WHERE workspace_id=$1 AND member_id=$2`},
		{"project membership deleted", `UPDATE project_members SET deleted_at=now() WHERE workspace_id=$1 AND member_id=$2`, `UPDATE project_members SET deleted_at=NULL WHERE workspace_id=$1 AND member_id=$2`},
	} {
		t.Run(change.name, func(t *testing.T) {
			f.exec(change.disable, f.workspace, f.member)
			check(t)
			f.exec(change.restore, f.workspace, f.member)
		})
	}
	otherProject, otherWorkspace := uuid.New().String(), uuid.New().String()
	f.addProject(otherProject)
	f.exec(`INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Other',$2,'other-search','UTC','',now(),now())`, otherWorkspace, f.owner)
	for _, change := range []struct {
		name, sql string
		value     any
	}{
		{"deleted intake relation", `UPDATE intake_issues SET deleted_at=now() WHERE issue_id=$1`, nil},
		{"mismatched intake project", `UPDATE intake_issues SET project_id=$2 WHERE issue_id=$1`, otherProject},
		{"mismatched intake workspace", `UPDATE intake_issues SET workspace_id=$2 WHERE issue_id=$1`, otherWorkspace},
	} {
		t.Run(change.name, func(t *testing.T) {
			if change.value == nil {
				f.exec(change.sql, own)
			} else {
				f.exec(change.sql, own, change.value)
			}
			raw, err := reader.Global(f.ctx, uuid.MustParse(f.member), "types", "Search acceptance", "issue,intake", "", "true")
			if err != nil {
				t.Fatal(err)
			}
			var result struct {
				Results map[string][]struct {
					ID string `json:"id"`
				} `json:"results"`
			}
			if err = json.Unmarshal(raw, &result); err != nil {
				t.Fatal(err)
			}
			if len(result.Results["issue"]) != 2 || len(result.Results["intake"]) != 1 || result.Results["intake"][0].ID != other {
				t.Fatalf("invalid intake relation leaked or hid the ordinary item: %s", raw)
			}
			f.exec(`UPDATE intake_issues SET deleted_at=NULL,workspace_id=$2,project_id=$3 WHERE issue_id=$1`, own, f.workspace, f.project)
		})
	}

	f.exec(`UPDATE issues SET is_draft=true WHERE id=$1`, own)
	f.exec(`UPDATE issues SET deleted_at=now() WHERE id=$1`, other)
	check(t)
}
