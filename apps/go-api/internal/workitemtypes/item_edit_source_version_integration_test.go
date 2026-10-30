//go:build integration

package workitemtypes_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitems"
)

func setEditSourceVersion(t *testing.T, f projectTypeFixture, repo *workitems.Repository, q workitems.Request, input *workitems.ItemEditInput) workitems.ItemEditSource {
	t.Helper()
	mode := input.Mode
	if mode == "publish" {
		mode = "draft"
	}
	result, err := repo.ReadItemEditSource(f.ctx, q, mode, input.ItemID, input.ProjectID)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	source := result.Body.(workitems.ItemEditSource)
	input.ExpectedSourceVersion = source.SourceVersion
	return source
}
func requireSourceConflict(t *testing.T, err error, reviewed bool) {
	t.Helper()
	var conflict *djangoapi.HTTPError
	if !errors.As(err, &conflict) || conflict.Status != 409 || conflict.Body["source_changed"] != true || (conflict.Body["review_again"] == true) != reviewed {
		t.Fatalf("expected source conflict, got %v", err)
	}
}
func TestItemEditSourceRejectsChangesWithoutTimestampBumps(t *testing.T) {
	for _, change := range []string{"title", "date", "assignee"} {
		t.Run(change, func(t *testing.T) {
			f, repo, q, input, _ := reviewedEditFixture(t)
			source := setEditSourceVersion(t, f, repo, q, &input)
			if source.SourceVersion == "" {
				t.Fatal("missing source token")
			}
			switch change {
			case "title":
				f.exec(`UPDATE issues SET name='Changed by someone else' WHERE id=$1`, input.ItemID)
			case "date":
				f.exec(`UPDATE issues SET target_date='2027-01-01' WHERE id=$1`, input.ItemID)
			case "assignee":
				f.exec(`INSERT INTO issue_assignees(id,workspace_id,project_id,issue_id,assignee_id,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,now(),now())`, f.workspace, f.project, input.ItemID, f.member)
			}
			_, err := repo.PreviewItemChange(f.ctx, q, input)
			requireSourceConflict(t, err, false)
			input.ExpectedSourceVersion = ""
			_, err = repo.PreviewItemChange(f.ctx, q, input)
			requireEditStatus(t, err, 400)
			latest := setEditSourceVersion(t, f, repo, q, &input)
			if latest.SourceVersion == source.SourceVersion {
				t.Fatal("source mutation did not change version")
			}
			if _, err = repo.PreviewItemChange(f.ctx, q, input); err != nil {
				t.Fatalf("%#v", err)
			}
		})
	}
}
func TestItemEditSourceConflictAndCommittedRetry(t *testing.T) {
	f, repo, q, input, _ := reviewedEditFixture(t)
	setEditSourceVersion(t, f, repo, q, &input)
	result, err := repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	input.ReviewID = result.Body.(workitems.ItemEditReview).ID
	f.exec(`UPDATE issues SET name='Changed before commit' WHERE id=$1`, input.ItemID)
	_, err = repo.CommitItemChange(f.ctx, q, input)
	requireSourceConflict(t, err, true)
	setEditSourceVersion(t, f, repo, q, &input)
	result, err = repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	input.ReviewID = result.Body.(workitems.ItemEditReview).ID
	if _, err = repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatalf("%#v", err)
	}
	f.exec(`UPDATE issues SET name='Later edit' WHERE id=$1`, input.ItemID)
	// The authorized exact receipt wins over stale token/revision/expiry checks.
	f.exec(`UPDATE work_item_schema_reviews SET created_at=now()-interval '40 minutes' WHERE id=$1`, input.ReviewID)
	f.exec(`UPDATE work_item_type_settings SET revision=revision+1 WHERE workspace_id=$1`, f.workspace)
	if _, err = repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatalf("%#v", err)
	}
	var title string
	if err = f.db.QueryRow(f.ctx, `SELECT name FROM issues WHERE id=$1`, input.ItemID).Scan(&title); err != nil || title != "Later edit" {
		t.Fatalf("retry overwrote newer item: %s %v", title, err)
	}
	// Existing committed receipts issued before source versions remain retryable.
	f.exec(`UPDATE work_item_schema_reviews SET command=command-'expected_source_version' WHERE id=$1`, input.ReviewID)
	input.ExpectedSourceVersion = ""
	if _, err = repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatalf("%#v", err)
	}
}
func TestItemEditSourcePrivateDraftAuthorizationAndPublication(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	q := workitems.Request{ActorID: f.member, Slug: "types", Timezone: "UTC"}
	input := workitems.ItemEditInput{Mode: "draft", ItemID: draft, ProjectID: f.project, Revision: f.catalog(t).Revision, Changes: map[string]any{"name": "Reviewed draft"}}
	setEditSourceVersion(t, f, repo, q, &input)
	denied := q
	denied.ActorID = f.owner
	_, err := repo.ReadItemEditSource(f.ctx, denied, "draft", draft, "")
	requireEditStatus(t, err, 404)
	f.exec(`UPDATE draft_issues SET name='Concurrent draft' WHERE id=$1`, draft)
	_, err = repo.PreviewItemChange(f.ctx, q, input)
	requireSourceConflict(t, err, false)
	setEditSourceVersion(t, f, repo, q, &input)
	input.ProjectID = ""
	input.Changes["state_id"] = nil
	input.Changes["assignee_ids"] = []any{}
	input.Changes["label_ids"] = []any{}
	input.Changes["cycle_id"] = nil
	input.Changes["module_ids"] = []any{}
	input.Changes["estimate_point"] = nil
	review, err := repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	input.ReviewID = review.Body.(workitems.ItemEditReview).ID
	if _, err = repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatalf("%#v", err)
	}
	input.Mode = "publish"
	input.ProjectID = f.project
	setEditSourceVersion(t, f, repo, q, &input)
	review, err = repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	input.ReviewID = review.Body.(workitems.ItemEditReview).ID
	if _, err = repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatalf("%#v", err)
	}
}

func TestItemEditSourceReadDoesNotWaitForConcurrentWriter(t *testing.T) {
	f, repo, q, input, field := reviewedEditFixture(t)
	before := setEditSourceVersion(t, f, repo, q, &input)
	repeated := setEditSourceVersion(t, f, repo, q, &input)
	if before.SourceVersion != repeated.SourceVersion {
		t.Fatal("unchanged source changed version")
	}
	writer, err := pgx.Connect(f.ctx, f.db.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.CloseContext(t, writer, context.WithoutCancel(f.ctx)) }()
	tx, err := writer.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.Rollback(t, tx, f.ctx) }()
	if _, err = tx.Exec(f.ctx, `SELECT work_item_schema_lock($1::uuid)`, f.workspace); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(f.ctx, `UPDATE issues SET name='Concurrent title',description_html='<p>Concurrent description</p>',custom_properties=jsonb_build_object($2::text,'<p>Concurrent value</p>') WHERE id=$1`, input.ItemID, field); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(f.ctx, `INSERT INTO issue_assignees(id,workspace_id,project_id,issue_id,assignee_id,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,now(),now())`, f.workspace, f.project, input.ItemID, f.member); err != nil {
		t.Fatal(err)
	}
	deadline, cancel := context.WithTimeout(f.ctx, 3*time.Second)
	defer cancel()
	during, err := repo.ReadItemEditSource(deadline, q, "edit", input.ItemID, input.ProjectID)
	if err != nil {
		t.Fatalf("read blocked behind writer: %v", err)
	}
	if !reflect.DeepEqual(during.Body, before) {
		t.Fatalf("read mixed uncommitted values: %+v", during.Body)
	}
	if err = tx.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	_, err = repo.PreviewItemChange(f.ctx, q, input)
	requireSourceConflict(t, err, false)
	after := setEditSourceVersion(t, f, repo, q, &input)
	item := after.Item.(map[string]any)
	if after.SourceVersion == before.SourceVersion || item["name"] != "Concurrent title" || item["description_html"] != "<p>Concurrent description</p>" {
		t.Fatalf("incorrect canonical source: %+v", after)
	}
	values := item["custom_properties"].(map[string]any)
	if values[field] != "<p>Concurrent value</p>" {
		t.Fatalf("incorrect canonical custom values: %+v", values)
	}
	if !reflect.DeepEqual(item["assignee_ids"], []any{f.member}) {
		t.Fatalf("incorrect canonical relationships: %#v", item["assignee_ids"])
	}
}
func TestItemEditSourceReadRejectsRevokedWorkspaceAccess(t *testing.T) {
	f, repo, q, input, _ := reviewedEditFixture(t)
	setEditSourceVersion(t, f, repo, q, &input)
	f.exec(`UPDATE workspace_members SET is_active=false WHERE workspace_id=$1 AND member_id=$2`, f.workspace, q.ActorID)
	_, err := repo.ReadItemEditSource(f.ctx, q, "edit", input.ItemID, input.ProjectID)
	requireEditStatus(t, err, 403)
}

func TestItemEditSourceReadRejectsDeletedProject(t *testing.T) {
	f, repo, q, input, _ := reviewedEditFixture(t)
	f.exec(`UPDATE projects SET deleted_at=now() WHERE id=$1`, f.project)
	_, err := repo.ReadItemEditSource(f.ctx, q, "edit", input.ItemID, input.ProjectID)
	requireEditStatus(t, err, 404)
}
