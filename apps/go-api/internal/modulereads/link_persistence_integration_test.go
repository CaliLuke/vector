//go:build integration

package modulereads

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

const linkWorkspace = "20000000-0000-4000-8000-000000000001"
const linkProject = "20000000-0000-4000-8000-000000000002"
const linkActor = "20000000-0000-4000-8000-000000000003"
const linkModule = "20000000-0000-4000-8000-000000000004"
const moduleLinkSchema = `
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text,deleted_at timestamptz);
CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid,deleted_at timestamptz,archived_at timestamptz);
CREATE TABLE users(id uuid PRIMARY KEY,user_timezone text);
CREATE TABLE modules(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,deleted_at timestamptz);
CREATE TABLE project_members(project_id uuid,workspace_id uuid,member_id uuid,is_active bool,deleted_at timestamptz,role int);
CREATE TABLE module_links(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,module_id uuid,created_by_id uuid,updated_by_id uuid,title text,url text,metadata jsonb,created_at timestamptz,updated_at timestamptz,deleted_at timestamptz);
INSERT INTO workspaces VALUES('20000000-0000-4000-8000-000000000001','links',NULL);
INSERT INTO projects VALUES('20000000-0000-4000-8000-000000000002','20000000-0000-4000-8000-000000000001',NULL,NULL);
INSERT INTO users VALUES('20000000-0000-4000-8000-000000000003','UTC');
INSERT INTO modules VALUES('20000000-0000-4000-8000-000000000004','20000000-0000-4000-8000-000000000002','20000000-0000-4000-8000-000000000001',NULL);
INSERT INTO project_members VALUES('20000000-0000-4000-8000-000000000002','20000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000003',true,NULL,20);
`

type linkEffects struct{ removed []string }

func (*linkEffects) Task(context.Context, string, []any, map[string]any) error { return nil }
func (effects *linkEffects) SoftDelete(_ context.Context, app, model, id string) error {
	effects.removed = append(effects.removed, app+"/"+model+"/"+id)
	return nil
}

func TestModuleLinkEditsPreserveOmittedMetadataAndAdmitScopedDeletion(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, moduleLinkSchema)
	effects := &linkEffects{}
	repository, err := NewLifecycleRepository(connection, effects)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := repository.CreateLink(ctx, linkActor, "links", linkProject, linkModule, map[string]any{"title": "Documentation", "url": "https://example.test/docs", "metadata": map[string]any{"source": "manual"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var created map[string]any
	if err = json.Unmarshal(raw, &created); err != nil {
		t.Fatal(err)
	}
	id, ok := created["id"].(string)
	if !ok || id == "" || created["project"] != linkProject || created["workspace"] != linkWorkspace || created["created_by"] != linkActor {
		t.Fatalf("created link = %#v", created)
	}
	raw, err = repository.UpdateLink(ctx, linkActor, "links", linkProject, linkModule, id, map[string]any{"title": "Updated documentation", "url": "https://example.test/docs"}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	var updated struct {
		Title     string
		URL       string
		Metadata  map[string]string
		UpdatedBy string `json:"updated_by"`
	}
	if err = json.Unmarshal(raw, &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Updated documentation" || updated.URL != "https://example.test/docs" || updated.Metadata["source"] != "manual" || updated.UpdatedBy != linkActor {
		t.Fatalf("updated link = %#v", updated)
	}
	if err = repository.DeleteLink(ctx, linkActor, "links", linkProject, linkModule, id); err != nil {
		t.Fatal(err)
	}
	var deleted bool
	if err = connection.QueryRow(ctx, `SELECT deleted_at IS NOT NULL FROM module_links WHERE id=$1 AND project_id=$2 AND workspace_id=$3`, id, linkProject, linkWorkspace).Scan(&deleted); err != nil {
		t.Fatal(err)
	}
	if !deleted || len(effects.removed) != 1 || effects.removed[0] != "db/modulelink/"+id {
		t.Fatalf("deletion = %t, effects = %#v", deleted, effects.removed)
	}
}

func TestModuleLinkDuplicateURLLeavesOnlyTheOriginalRow(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, moduleLinkSchema)
	repository, err := NewLifecycleRepository(connection, &linkEffects{})
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]any{"title": "Original", "url": "https://example.test/docs"}
	if _, err = repository.CreateLink(ctx, linkActor, "links", linkProject, linkModule, fields, nil); err != nil {
		t.Fatal(err)
	}
	_, err = repository.CreateLink(ctx, linkActor, "links", linkProject, linkModule, fields, nil)
	var invalid *ModuleValidationError
	if !errors.As(err, &invalid) {
		t.Fatalf("duplicate error = %v", err)
	}
	var count int
	if err = connection.QueryRow(ctx, `SELECT count(*) FROM module_links`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("duplicate URL created %d rows", count)
	}
}
