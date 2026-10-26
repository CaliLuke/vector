//go:build integration

package workitemtypes_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
)

type typeTestPublisher struct{}

func (typeTestPublisher) Task(context.Context, string, []any, map[string]any) error { return nil }
func (typeTestPublisher) SoftDelete(context.Context, string, string, string) error  { return nil }

func TestUnavailableTypeRejectsWholeIssueEdit(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	repo, err := workitems.NewRepository(f.db, typeTestPublisher{}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	execute := func(method, id, body string) (workitems.Result, error) {
		q := workitems.Request{Operation: "api_v1_workspaces_by_slug_projects_by_project_id_issues_by_pk", Method: method, ActorID: f.owner, Slug: "types", ProjectID: f.project, IssueID: id, API: true, Timezone: "UTC", Params: map[string]string{}}
		return repo.Execute(f.ctx, q, strings.NewReader(body), "application/json")
	}
	result, err := execute("POST", "", `{"name":"Original"}`)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(result.Body)
	if err != nil {
		t.Fatal(err)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err = json.Unmarshal(data, &created); err != nil || created.ID == "" {
		t.Fatalf("creation failed: %s %v", data, err)
	}
	label := uuid.New().String()
	f.exec(`INSERT INTO labels(id,name,description,color,sort_order,project_id,workspace_id,created_at,updated_at) VALUES($1,'Keep label','','#123456',0,$2,$3,now(),now())`, label, f.project, f.workspace)
	f.exec(`INSERT INTO issue_labels(id,issue_id,label_id,project_id,workspace_id,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,now(),now())`, created.ID, label, f.project, f.workspace)
	f.exec(`INSERT INTO issue_assignees(id,issue_id,assignee_id,project_id,workspace_id,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,now(),now())`, created.ID, f.owner, f.project, f.workspace)
	assertUnchanged := func(t *testing.T) {
		t.Helper()
		var name, typeID string
		var labels, assignees int
		if e := f.db.QueryRow(f.ctx, `SELECT name,type_id::text,(SELECT count(*) FROM issue_labels WHERE issue_id=$1 AND deleted_at IS NULL),(SELECT count(*) FROM issue_assignees WHERE issue_id=$1 AND deleted_at IS NULL) FROM issues WHERE id=$1`, created.ID).Scan(&name, &typeID, &labels, &assignees); e != nil {
			t.Fatal(e)
		}
		if name != "Original" || typeID != f.task || labels != 1 || assignees != 1 {
			t.Fatalf("rejected edit partially persisted: %s %s labels=%d assignees=%d", name, typeID, labels, assignees)
		}
	}
	f.mutate(t, "pause", f.bug)
	t.Run("paused type", func(t *testing.T) {
		_, writeErr := execute("PATCH", created.ID, `{"name":"Must not save","type_id":"`+f.bug+`","labels":[],"assignees":[]}`)
		var constraint *djangoapi.HTTPError
		if !errors.As(writeErr, &constraint) || constraint.Status != 400 {
			t.Fatalf("want availability validation rejection, got %v", writeErr)
		}
		assertUnchanged(t)
	})
	f.mutate(t, "remove", f.bug)
	t.Run("removed type", func(t *testing.T) {
		_, writeErr := execute("PATCH", created.ID, `{"name":"Must not save","type_id":"`+f.bug+`","labels":[],"assignees":[]}`)
		var constraint *djangoapi.HTTPError
		if !errors.As(writeErr, &constraint) || constraint.Status != 400 {
			t.Fatalf("want availability validation rejection, got %v", writeErr)
		}
		assertUnchanged(t)
	})
	f.mutate(t, "import", f.bug)
	_, err = execute("PATCH", created.ID, `{"type_id":"`+f.bug+`"}`)
	if err != nil {
		t.Fatal(err)
	}
	f.mutate(t, "remove", f.bug)
	_, err = execute("PATCH", created.ID, `{"name":"Existing type stays editable"}`)
	if err != nil {
		t.Fatal(err)
	}
	var name, typeID string
	if err = f.db.QueryRow(f.ctx, `SELECT name,type_id::text FROM issues WHERE id=$1`, created.ID).Scan(&name, &typeID); err != nil {
		t.Fatal(err)
	}
	if name != "Existing type stays editable" || typeID != f.bug {
		t.Fatalf("existing item changed identity: %s %s", name, typeID)
	}
}
