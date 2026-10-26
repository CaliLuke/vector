//go:build integration

package workitemtypes_test

import (
	"errors"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
)

func TestDraftPublicationAndDeletionRespectPrivacyAndCurrentProjectAccess(t *testing.T) {
	f := newProjectTypeFixture(t)
	draft := uuid.New().String()
	f.exec(`INSERT INTO draft_issues(id,workspace_id,project_id,type_id,name,description_json,description_html,priority,sort_order,created_at,updated_at,created_by_id) VALUES($1,$2,$3,$4,'Private work','{}','<p>Private text</p>','none',0,now(),now(),$5)`, draft, f.workspace, f.project, f.task, f.member)
	repo, err := workitems.NewRepository(f.db, typeTestPublisher{}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(actor, method, operation string) error {
		_, failure := repo.Execute(f.ctx, workitems.Request{Operation: operation, Method: method, ActorID: actor, Slug: "types", Timezone: "UTC", Params: map[string]string{"pk": draft, "draft_id": draft}}, strings.NewReader(`{"name":"Published"}`), "application/json")
		return failure
	}
	assertStatus := func(err error, status int) {
		t.Helper()
		var failure *djangoapi.HTTPError
		if !errors.As(err, &failure) || failure.Status != status {
			t.Fatalf("want HTTP %d, got %v", status, err)
		}
	}
	const publish = "api_workspaces_by_slug_draft_issues_by_draft_id_draft_to_issue"
	const edit = "api_workspaces_by_slug_draft_issues_by_pk"
	assertStatus(invoke(f.owner, "POST", publish), 404)
	assertStatus(invoke(f.owner, "DELETE", edit), 404)
	f.exec(`UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	assertStatus(invoke(f.member, "POST", publish), 403)
	var drafts, issues int
	if err = f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM draft_issues WHERE id=$1 AND deleted_at IS NULL),(SELECT count(*) FROM issues WHERE workspace_id=$2)`, draft, f.workspace).Scan(&drafts, &issues); err != nil {
		t.Fatal(err)
	}
	if drafts != 1 || issues != 0 {
		t.Fatalf("denied publication/deletion changed data: drafts=%d issues=%d", drafts, issues)
	}
	// Private ownership also remains usable for guest members who cannot publish.
	f.exec(`UPDATE workspace_members SET role=5 WHERE workspace_id=$1 AND member_id=$2`, f.workspace, f.member)
	f.exec(`UPDATE project_members SET role=5,is_active=true WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	if err = invoke(f.member, "PATCH", edit); err != nil {
		t.Fatalf("owner could not edit private draft: %v", err)
	}
	var name string
	if err = f.db.QueryRow(f.ctx, `SELECT name FROM draft_issues WHERE id=$1`, draft).Scan(&name); err != nil || name != "Published" {
		t.Fatalf("owner edit not durable: %q %v", name, err)
	}
	assertStatus(invoke(f.member, "POST", publish), 403)
	if err = invoke(f.member, "DELETE", edit); err != nil {
		t.Fatalf("owner could not delete private draft: %v", err)
	}
	var deleted bool
	if err = f.db.QueryRow(f.ctx, `SELECT deleted_at IS NOT NULL FROM draft_issues WHERE id=$1`, draft).Scan(&deleted); err != nil || !deleted {
		t.Fatalf("owner deletion not durable: %t %v", deleted, err)
	}

}
