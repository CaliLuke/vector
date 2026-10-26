//go:build integration

package workitemtypes_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func reviewedEditFixture(t *testing.T) (projectTypeFixture, *workitems.Repository, workitems.Request, workitems.ItemEditInput, string) {
	t.Helper()
	f := newProjectTypeFixture(t)
	field := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: field, Title: "Old details", Kind: property.RichText, Configuration: property.Configuration{Active: true}})
	f.mutate(t, "import", f.bug)
	item := addHierarchyItem(t, f, f.task, "")
	values, _ := json.Marshal(map[string]any{field: "<p>Preserve this value</p>"})
	f.exec(`UPDATE issues SET description_html='<p>Before</p>',custom_properties=$2 WHERE id=$1`, item, values)
	repo, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatalf("%#v", err)
	}
	q := workitems.Request{ActorID: f.owner, Slug: "types", Timezone: "UTC"}
	input := workitems.ItemEditInput{Mode: "edit", ProjectID: f.project, ItemID: item, Revision: f.catalog(t).Revision, Changes: map[string]any{"type_id": f.bug, "description_html": "<p>Proposed description</p>", "name": "Edited name"}}
	setEditSourceVersion(t, f, repo, q, &input)
	return f, repo, q, input, field
}

func requireEditStatus(t *testing.T, err error, status int) {
	t.Helper()
	var denied *djangoapi.HTTPError
	if !errors.As(err, &denied) || denied.Status != status {
		t.Fatalf("want HTTP %d, got %v", status, err)
	}
}

func TestReviewedItemEditPreviewCommitAndExactRetry(t *testing.T) {
	f, repo, q, input, field := reviewedEditFixture(t)
	result, err := repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	review := result.Body.(workitems.ItemEditReview)
	if review.PreservedProperties != 1 || !strings.Contains(review.Description, "Preserve this value") || !strings.HasPrefix(review.Description, "<p>Proposed description</p>") {
		t.Fatalf("bad review: %+v", review)
	}
	var itemType, description, name string
	var effects, versions int
	check := func() {
		t.Helper()
		if err = f.db.QueryRow(f.ctx, `SELECT type_id::text,description_html,name,(SELECT count(*) FROM work_item_effects),(SELECT count(*) FROM work_item_property_versions WHERE issue_id=$1) FROM issues WHERE id=$1`, input.ItemID).Scan(&itemType, &description, &name, &effects, &versions); err != nil {
			t.Fatalf("%#v", err)
		}
	}
	check()
	if itemType != f.task || description != "<p>Before</p>" || effects != 0 || versions != 0 {
		t.Fatal("preview persisted its dry run")
	}
	input.ReviewID = review.ID
	result, err = repo.CommitItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	assertReviewedAppItem(t, result, input.ItemID, f.project, f.bug)
	check()
	if itemType != f.bug || description != review.Description || name != "Edited name" || effects == 0 || versions != 1 {
		t.Fatalf("incomplete commit: %s %s %s %d %d", itemType, description, name, effects, versions)
	}
	var value string
	if err = f.db.QueryRow(f.ctx, `SELECT value#>>'{}' FROM work_item_property_versions WHERE issue_id=$1 AND property_id=$2`, input.ItemID, field).Scan(&value); err != nil || value != "<p>Preserve this value</p>" {
		t.Fatalf("lost history: %s %v", value, err)
	}
	beforeEffects := effects
	if _, err = repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatalf("%#v", err)
	}
	check()
	if effects != beforeEffects || versions != 1 || strings.Count(description, "Preserve this value") != 1 {
		t.Fatal("retry duplicated effects or preservation")
	}
	input.Changes["name"] = "Different request"
	_, err = repo.CommitItemChange(f.ctx, q, input)
	requireEditStatus(t, err, 409)
}

func TestReviewedItemEditRejectsStaleSourceAndLateFailure(t *testing.T) {
	f, repo, q, input, _ := reviewedEditFixture(t)
	result, err := repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	input.ReviewID = result.Body.(workitems.ItemEditReview).ID
	f.exec(`UPDATE issues SET name='Concurrent name' WHERE id=$1`, input.ItemID)
	_, err = repo.CommitItemChange(f.ctx, q, input)
	requireEditStatus(t, err, 409)
	var name, itemType string
	if err = f.db.QueryRow(f.ctx, `SELECT name,type_id::text FROM issues WHERE id=$1`, input.ItemID).Scan(&name, &itemType); err != nil || name != "Concurrent name" || itemType != f.task {
		t.Fatalf("stale review overwrote source: %s %s %v", name, itemType, err)
	}
	setEditSourceVersion(t, f, repo, q, &input)
	result, err = repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	input.ReviewID = result.Body.(workitems.ItemEditReview).ID
	f.exec(`CREATE FUNCTION reject_reviewed_effect() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'reject effect'; END $$; CREATE TRIGGER reject_reviewed_effect BEFORE INSERT ON work_item_effects FOR EACH ROW EXECUTE FUNCTION reject_reviewed_effect()`)
	if _, err = repo.CommitItemChange(f.ctx, q, input); err == nil {
		t.Fatal("injected failure accepted")
	}
	f.exec(`DROP TRIGGER reject_reviewed_effect ON work_item_effects`)
	if _, err = repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatalf("%#v", err)
	}
}

func TestReviewedDraftMoveRevalidatesAndMovesRetainedRelations(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	target := publicationTarget(t, f)
	state := uuid.New().String()
	f.exec(`INSERT INTO states(id,name,description,slug,color,sequence,"group","default",is_triage,workspace_id,project_id,created_at,updated_at) VALUES($1,'Todo','','todo','#abc',1,'unstarted',true,false,$2,$3,now(),now())`, state, f.workspace, f.project)
	f.exec(`UPDATE draft_issues SET state_id=$2 WHERE id=$1`, draft, state)
	input := workitems.ItemEditInput{Mode: "draft", ProjectID: target, ItemID: draft, Revision: f.catalog(t).Revision, Changes: map[string]any{"label_ids": []string{}}}
	q := workitems.Request{ActorID: f.member, Slug: "types", Timezone: "UTC"}
	setEditSourceVersion(t, f, repo, q, &input)
	_, err := repo.PreviewItemChange(f.ctx, q, input)
	requireEditStatus(t, err, 400)
	input.Changes["state_id"] = nil
	result, err := repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	input.ReviewID = result.Body.(workitems.ItemEditReview).ID
	if _, err = repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatalf("%#v", err)
	}
	var project, relationProject string
	if err = f.db.QueryRow(f.ctx, `SELECT d.project_id::text,a.project_id::text FROM draft_issues d JOIN draft_issue_assignees a ON a.draft_issue_id=d.id AND a.deleted_at IS NULL WHERE d.id=$1`, draft).Scan(&project, &relationProject); err != nil || project != target || relationProject != target {
		t.Fatalf("retained relation scope wrong: %s %s %v", project, relationProject, err)
	}
}

func TestReviewedDraftPublicationTransfersOnlyReviewedAssets(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	retained := publicationAsset(f, draft, f.project)
	removed := publicationAsset(f, draft, f.project)
	attachment := publicationAsset(f, draft, f.project)
	f.exec(`UPDATE file_assets SET entity_type='DRAFT_ISSUE_ATTACHMENT' WHERE id=$1`, attachment)
	input := workitems.ItemEditInput{Mode: "publish", ProjectID: f.project, ItemID: draft, Revision: f.catalog(t).Revision, Changes: map[string]any{"description_html": "<p>Keep image</p><img src=\"" + retained + "\">"}}
	q := workitems.Request{ActorID: f.member, Slug: "types", Timezone: "UTC"}
	setEditSourceVersion(t, f, repo, q, &input)
	result, err := repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	input.ReviewID = result.Body.(workitems.ItemEditReview).ID
	// An attachment change after review must invalidate the whole operation.
	f.exec(`UPDATE file_assets SET size=2 WHERE id=$1`, attachment)
	_, err = repo.CommitItemChange(f.ctx, q, input)
	requireEditStatus(t, err, 409)
	setEditSourceVersion(t, f, repo, q, &input)
	result, err = repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	input.ReviewID = result.Body.(workitems.ItemEditReview).ID
	result, err = repo.CommitItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	issue := resultIssueID(t, result)
	assertReviewedAppItem(t, result, issue, f.project, f.task)
	if !reflect.DeepEqual(result.Body.(map[string]any)["assignee_ids"], []any{f.member}) {
		t.Fatal("publication response omitted saved assignees")
	}
	var transferred, private int
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FILTER(WHERE issue_id=$1),count(*) FILTER(WHERE id=$2 AND issue_id IS NULL AND draft_issue_id=$3) FROM file_assets WHERE id=ANY($4::uuid[])`, issue, removed, draft, []string{retained, removed, attachment}).Scan(&transferred, &private); err != nil || transferred != 2 || private != 1 {
		t.Fatalf("unreviewed transfer: %d %d %v", transferred, private, err)
	}
	if retry, retryErr := repo.CommitItemChange(f.ctx, q, input); retryErr != nil || resultIssueID(t, retry) != issue {
		t.Fatalf("publication retry failed: %+v %v", retry, retryErr)
	}
}

func TestReviewedDraftMoveUpdatesExistingImageProject(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	target := publicationTarget(t, f)
	asset := publicationAsset(f, draft, f.project)
	q := workitems.Request{ActorID: f.member, Slug: "types", Timezone: "UTC"}
	input := workitems.ItemEditInput{Mode: "draft", ProjectID: target, ItemID: draft, Revision: f.catalog(t).Revision, Changes: map[string]any{"description_html": "<img src=\"" + asset + "\">", "label_ids": []string{}}}
	setEditSourceVersion(t, f, repo, q, &input)
	result, err := repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	input.ReviewID = result.Body.(workitems.ItemEditReview).ID
	if _, err = repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatalf("%#v", err)
	}
	var project string
	if err = f.db.QueryRow(f.ctx, `SELECT project_id::text FROM file_assets WHERE id=$1`, asset).Scan(&project); err != nil || project != target {
		t.Fatalf("existing image kept old project: %s %v", project, err)
	}
}

func TestReviewedItemEditLocksExplicitLabelUntilCommit(t *testing.T) {
	f, _, q, input, _ := reviewedEditFixture(t)
	label := uuid.New().String()
	f.exec(`INSERT INTO labels(id,name,description,color,sort_order,project_id,workspace_id,created_at,updated_at) VALUES($1,'Selection','','#123456',0,$2,$3,now(),now())`, label, f.project, f.workspace)
	input.Changes["label_ids"] = []string{label}
	connection := hierarchyConnection(t, f)
	repo, err := workitems.NewRepository(connection, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatalf("%#v", err)
	}
	result, err := repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	input.ReviewID = result.Body.(workitems.ItemEditReview).ID
	f.exec(`CREATE FUNCTION pause_reviewed_item() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock(88310); RETURN NEW; END $$; CREATE TRIGGER pause_reviewed_item BEFORE UPDATE ON issues FOR EACH ROW EXECUTE FUNCTION pause_reviewed_item()`)
	lock, err := f.db.Begin(f.ctx)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	t.Cleanup(func() { _ = lock.Rollback(context.WithoutCancel(f.ctx)) })
	if _, err = lock.Exec(f.ctx, `SELECT pg_advisory_xact_lock(88310)`); err != nil {
		t.Fatalf("%#v", err)
	}
	done := make(chan error, 1)
	go func() { _, writeErr := repo.CommitItemChange(f.ctx, q, input); done <- writeErr }()
	waitForSchemaWriter(t, f, lock, int(connection.PgConn().PID()), done)
	remover := hierarchyConnection(t, f)
	if _, err = remover.Exec(f.ctx, `SET lock_timeout='100ms'`); err != nil {
		t.Fatalf("%#v", err)
	}
	_, err = remover.Exec(f.ctx, `UPDATE labels SET deleted_at=now() WHERE id=$1`, label)
	var blocked *pgconn.PgError
	if !errors.As(err, &blocked) || blocked.Code != "55P03" {
		t.Fatalf("reviewed selection was not locked: %v", err)
	}
	if err = lock.Commit(f.ctx); err != nil {
		t.Fatalf("%#v", err)
	}
	select {
	case err = <-done:
		if err != nil {
			t.Fatalf("%#v", err)
		}
	case <-f.ctx.Done():
		t.Fatal("reviewed write did not finish")
	}
	var retained int
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FROM issue_labels WHERE issue_id=$1 AND label_id=$2 AND deleted_at IS NULL`, input.ItemID, label).Scan(&retained); err != nil || retained != 1 {
		t.Fatalf("explicit selection was filtered: %d %v", retained, err)
	}
}

func TestReviewedPublicationRetainsEstimateAndRejectsForeignDestination(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	estimate, point := uuid.New().String(), uuid.New().String()
	f.exec(`INSERT INTO estimates(id,workspace_id,project_id,name,description,type,last_used,created_at,updated_at) VALUES($1,$2,$3,'Effort','','points',true,now(),now())`, estimate, f.workspace, f.project)
	f.exec(`INSERT INTO estimate_points(id,workspace_id,project_id,estimate_id,key,value,description,created_at,updated_at) VALUES($1,$2,$3,$4,1,'8','',now(),now())`, point, f.workspace, f.project, estimate)
	f.exec(`UPDATE draft_issues SET estimate_point_id=$2 WHERE id=$1`, draft, point)
	target := publicationTarget(t, f)
	q := workitems.Request{ActorID: f.member, Slug: "types", Timezone: "UTC"}
	input := workitems.ItemEditInput{Mode: "publish", ProjectID: target, ItemID: draft, Revision: f.catalog(t).Revision, Changes: map[string]any{"label_ids": []string{}}}
	setEditSourceVersion(t, f, repo, q, &input)
	_, err := repo.PreviewItemChange(f.ctx, q, input)
	requireEditStatus(t, err, 400)
	input.ProjectID = f.project
	result, err := repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	input.ReviewID = result.Body.(workitems.ItemEditReview).ID
	result, err = repo.CommitItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	var actual string
	if err = f.db.QueryRow(f.ctx, `SELECT estimate_point_id::text FROM issues WHERE id=$1`, resultIssueID(t, result)).Scan(&actual); err != nil || actual != point {
		t.Fatalf("omitted saved estimate lost: %s %v", actual, err)
	}
}

func assertReviewedAppItem(t *testing.T, result workitems.Result, id, project, itemType string) {
	t.Helper()
	body, ok := result.Body.(map[string]any)
	if !ok || body["id"] != id || body["project_id"] != project || body["type_id"] != itemType {
		t.Fatalf("reviewed save response is not an app item: %#v", result.Body)
	}
	for _, field := range []string{"custom_properties", "description_html", "state_id", "parent_id", "label_ids", "assignee_ids", "cycle_id", "module_ids", "is_subscribed"} {
		if _, present := body[field]; !present {
			t.Fatalf("app response missing %s: %#v", field, body)
		}
	}
}

func TestReviewedPublicationRetryMasksParentAfterAccessLoss(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	parent := addHierarchyItem(t, f, f.task, "")
	f.exec(`UPDATE draft_issues SET parent_id=$2 WHERE id=$1`, draft, parent)
	input := workitems.ItemEditInput{Mode: "publish", ProjectID: f.project, ItemID: draft, Revision: f.catalog(t).Revision, Changes: map[string]any{}}
	q := workitems.Request{ActorID: f.member, Slug: "types", Timezone: "UTC"}
	setEditSourceVersion(t, f, repo, q, &input)
	review, err := repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatal(err)
	}
	input.ReviewID = review.Body.(workitems.ItemEditReview).ID
	result, err := repo.CommitItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatal(err)
	}
	item := resultIssueID(t, result)
	privateProject := uuid.New().String()
	f.addProject(privateProject)
	f.exec(`UPDATE issues SET project_id=$2 WHERE id=$1`, parent, privateProject)
	retry, err := repo.CommitItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatal(err)
	}
	assertReviewedAppItem(t, retry, item, f.project, f.task)
	assertHierarchyPrivate(t, retry.Body, parent)
}
