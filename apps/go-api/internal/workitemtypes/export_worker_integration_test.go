//go:build integration

package workitemtypes_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type exportArtifact struct{ data []byte }

func (a *exportArtifact) Put(_ context.Context, _, _ string, data []byte) error {
	a.data = bytes.Clone(data)
	return nil
}
func (*exportArtifact) PresignGet(context.Context, string, string, *string, string, string) (string, error) {
	return "https://example.test/private-export", nil
}

func exportWorkerFixture(t *testing.T, f projectTypeFixture, custom bool) (backgroundtasks.IssueExportInput, *exportArtifact, func(backgroundtasks.IssueExportInput) error) {
	t.Helper()
	pool, err := pgxpool.New(f.ctx, f.db.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	input := backgroundtasks.IssueExportInput{Provider: "json", WorkspaceID: f.workspace, ProjectIDs: []string{f.project}, TokenID: uuid.NewString(), Slug: "types", IncludeCustomProperties: custom}
	f.exec(`INSERT INTO exporters(id,created_at,updated_at,workspace_id,project,provider,status,reason,key,token,initiated_by_id,type,rich_filters) VALUES(gen_random_uuid(),now(),now(),$1,$2::uuid[],'json','queued','','',$3,$4,'issue_exports',jsonb_build_object('include_custom_properties',$5::boolean))`, f.workspace, input.ProjectIDs, input.TokenID, f.owner, custom)
	artifact := &exportArtifact{}
	store, err := backgroundtasks.NewPostgresIssueExportStore(pool, artifact)
	if err != nil {
		t.Fatal(err)
	}
	return input, artifact, func(input backgroundtasks.IssueExportInput) error { return store.Generate(f.ctx, input) }
}

func exportJSONFiles(t *testing.T, artifact *exportArtifact) [][]map[string]any {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(artifact.data), int64(len(artifact.data)))
	if err != nil {
		t.Fatal(err)
	}
	files := [][]map[string]any{}
	for _, entry := range archive.File {
		reader, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		testsupport.Close(t, reader)
		if err != nil {
			t.Fatal(err)
		}
		var rows []map[string]any
		if err = json.Unmarshal(data, &rows); err != nil {
			t.Fatal(err)
		}
		files = append(files, rows)
	}
	return files
}

func TestCustomExportWorkerUsesOneSnapshotAndPersistsPrivateScope(t *testing.T) {
	f := newProjectTypeFixture(t)
	_, ids := seedDiscoveryItems(t, f)
	flag := uuid.NewString()
	attachValueProperties(t, f, property.Definition{ID: flag, Title: "Ready", Kind: property.Boolean, Configuration: property.Configuration{Active: true}})
	f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($2::text,false) WHERE id=$1`, ids[0], flag)
	second := uuid.NewString()
	f.addProject(second)
	f.exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,20,true,now(),now(),'{}','{}','{}',0)`, f.workspace, second, f.owner)
	extra := uuid.NewString()
	f.exec(`INSERT INTO issues(id,project_id,workspace_id,created_by_id,type_id,name,description_json,description_html,priority,sequence_id,sort_order,is_draft,created_at,updated_at) VALUES($1,$2,$3,$4,$5,'Other project','{}','<p></p>','none',1,0,false,now(),now())`, extra, second, f.workspace, f.owner, f.task)
	input, artifact, generate := exportWorkerFixture(t, f, true)
	input.ProjectIDs = append(input.ProjectIDs, second)
	input.Multiple = true
	f.exec(`UPDATE exporters SET project=$2::uuid[] WHERE token=$1`, input.TokenID, input.ProjectIDs)
	if err := generate(input); err != nil {
		t.Fatal(err)
	}
	files := exportJSONFiles(t, artifact)
	if len(files) != 2 || len(files[0]) != 3 || len(files[1]) != 1 {
		t.Fatalf("partitioned files: %+v", files)
	}
	if files[0][0]["customproperty_"+flag] != false {
		t.Fatalf("false lost: %+v", files[0][0])
	}
	if files[0][0]["properties_evaluated_at"] != files[1][0]["properties_evaluated_at"] {
		t.Fatal("project files use different evaluation instants")
	}
	if _, leaked := files[0][0]["project_id"]; leaked {
		t.Fatal("internal partition identity leaked")
	}
	var status string
	var references []string
	if err := f.db.QueryRow(f.ctx, `SELECT status,rich_filters->'exported_item_ids' FROM exporters WHERE token=$1`, input.TokenID).Scan(&status, &references); err != nil {
		t.Fatal(err)
	}
	if status != "completed" || len(references) != 4 {
		t.Fatalf("history scope: %s %+v", status, references)
	}
}

func TestCustomExportWorkerRejectsChangedAuthorizationAndScope(t *testing.T) {
	for _, scenario := range []string{"demoted", "revoked project", "changed scope", "missing opt in", "downgraded input"} {
		t.Run(scenario, func(t *testing.T) {
			f := newProjectTypeFixture(t)
			seedDiscoveryItems(t, f)
			input, artifact, generate := exportWorkerFixture(t, f, true)
			switch scenario {
			case "demoted":
				f.exec(`UPDATE workspace_members SET role=5 WHERE workspace_id=$1 AND member_id=$2`, f.workspace, f.owner)
			case "revoked project":
				f.exec(`UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, f.project, f.owner)
			case "changed scope":
				input.ProjectIDs = []string{uuid.NewString()}
			case "missing opt in":
				f.exec(`UPDATE exporters SET rich_filters='{}' WHERE token=$1`, input.TokenID)
			case "downgraded input":
				input.IncludeCustomProperties = false
			}
			if err := generate(input); err == nil {
				t.Fatal("unauthorized export succeeded")
			}
			if len(artifact.data) != 0 {
				t.Fatal("unauthorized artifact uploaded")
			}
		})
	}
}

func TestStandardExportKeepsOriginalJSONFields(t *testing.T) {
	f := newProjectTypeFixture(t)
	seedDiscoveryItems(t, f)
	input, artifact, generate := exportWorkerFixture(t, f, false)
	if err := generate(input); err != nil {
		t.Fatal(err)
	}
	files := exportJSONFiles(t, artifact)
	if len(files) != 1 || len(files[0]) != 3 {
		t.Fatalf("standard rows: %+v", files)
	}
	row := files[0][0]
	if _, exists := row["type_id"]; exists {
		t.Fatal("standard export gained custom type field")
	}
	if _, exists := row["property_values"]; exists {
		t.Fatal("standard export gained custom values")
	}
	if row["name"] != "Discovery 0" || len(row) != 28 {
		t.Fatalf("standard format changed: %+v", row)
	}
}
