//go:build integration

package workitemtypes_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
)

const draftPublishOperation = "api_workspaces_by_slug_draft_issues_by_draft_id_draft_to_issue"

type noRequestEffects struct{ t *testing.T }

func (p noRequestEffects) Task(context.Context, string, []any, map[string]any) error {
	p.t.Error("effect admitted before publication committed")
	return errors.New("unexpected synchronous effect")
}
func (p noRequestEffects) SoftDelete(context.Context, string, string, string) error {
	p.t.Error("deletion admitted before publication committed")
	return errors.New("unexpected synchronous deletion")
}

func seedPublicationDraft(t *testing.T, f projectTypeFixture) (*workitems.Repository, string, string) {
	t.Helper()
	draft, label := uuid.New().String(), uuid.New().String()
	f.exec(`INSERT INTO draft_issues(id,workspace_id,project_id,type_id,name,description_json,description_html,priority,sort_order,created_at,updated_at,created_by_id) VALUES($1,$2,$3,$4,'Saved title','{}','<p>Saved description</p>','high',0,now(),now(),$5)`, draft, f.workspace, f.project, f.task, f.member)
	f.exec(`INSERT INTO labels(id,name,description,color,sort_order,project_id,workspace_id,created_at,updated_at) VALUES($1,'Saved label','','#123456',0,$2,$3,now(),now())`, label, f.project, f.workspace)
	f.exec(`INSERT INTO draft_issue_labels(id,draft_issue_id,label_id,project_id,workspace_id,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,now(),now())`, draft, label, f.project, f.workspace)
	f.exec(`INSERT INTO draft_issue_assignees(id,draft_issue_id,assignee_id,project_id,workspace_id,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,now(),now())`, draft, f.member, f.project, f.workspace)
	repo, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	return repo, draft, label
}

func publishDraftForTest(f projectTypeFixture, repo *workitems.Repository, draft, actor, body string) (workitems.Result, error) {
	return repo.Execute(f.ctx, workitems.Request{Operation: draftPublishOperation, Method: "POST", ActorID: actor, Slug: "types", Timezone: "UTC", Params: map[string]string{"draft_id": draft}}, strings.NewReader(body), "application/json")
}

func resultIssueID(t *testing.T, result workitems.Result) string {
	t.Helper()
	raw, err := json.Marshal(result.Body)
	if err != nil {
		t.Fatal(err)
	}
	var item struct{ ID string }
	if err = json.Unmarshal(raw, &item); err != nil || item.ID == "" {
		t.Fatalf("missing issue: %s %v", raw, err)
	}
	return item.ID
}

func TestDraftPublicationRetainsSavedDataAndRetriesAfterCleanup(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, label := seedPublicationDraft(t, f)
	result, err := publishDraftForTest(f, repo, draft, f.member, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	id := resultIssueID(t, result)
	var name, description, priority, typeID string
	var labels, assignees, effects int
	var consumed bool
	err = f.db.QueryRow(f.ctx, `SELECT name,description_html,priority,type_id::text,(SELECT count(*) FROM issue_labels WHERE issue_id=i.id AND label_id=$2 AND deleted_at IS NULL),(SELECT count(*) FROM issue_assignees WHERE issue_id=i.id AND assignee_id=$3 AND deleted_at IS NULL),(SELECT deleted_at IS NOT NULL FROM draft_issues WHERE id=$4),(SELECT count(*) FROM work_item_effects WHERE admitted_at IS NULL) FROM issues i WHERE id=$1`, id, label, f.member, draft).Scan(&name, &description, &priority, &typeID, &labels, &assignees, &consumed, &effects)
	if err != nil {
		t.Fatal(err)
	}
	if name != "Saved title" || description != "<p>Saved description</p>" || priority != "high" || typeID != f.task || labels != 1 || assignees != 1 || !consumed || effects != 2 {
		t.Fatalf("saved publication data lost: name=%s html=%s priority=%s type=%s labels=%d assignees=%d consumed=%v effects=%d", name, description, priority, typeID, labels, assignees, consumed, effects)
	}
	f.exec(`DELETE FROM draft_issue_labels WHERE draft_issue_id=$1`, draft)
	f.exec(`DELETE FROM draft_issue_assignees WHERE draft_issue_id=$1`, draft)
	f.exec(`DELETE FROM draft_issues WHERE id=$1`, draft)
	retry, err := publishDraftForTest(f, repo, draft, f.member, `{"name":"Must not replace published title"}`)
	if err != nil || resultIssueID(t, retry) != id {
		t.Fatalf("publication retry not idempotent: %+v %v", retry, err)
	}
	retryFields, ok := retry.Body.(map[string]any)
	if !ok || !reflect.DeepEqual(retryFields["assignee_ids"], []any{f.member}) || !reflect.DeepEqual(retryFields["label_ids"], []any{label}) {
		t.Fatalf("retry lost stored relationship IDs: %#v", retry.Body)
	}
	var issues, remainingEffects int
	if err = f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM issues),(SELECT count(*) FROM work_item_effects)`).Scan(&issues, &remainingEffects); err != nil || issues != 1 || remainingEffects != effects {
		t.Fatalf("retry repeated durable work: issues=%d effects=%d %v", issues, remainingEffects, err)
	}
	_, err = publishDraftForTest(f, repo, draft, f.owner, `{}`)
	var denied *djangoapi.HTTPError
	if !errors.As(err, &denied) || denied.Status != 404 {
		t.Fatalf("another owner obtained published draft: %v", err)
	}
	f.exec(`UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	_, err = publishDraftForTest(f, repo, draft, f.member, `{}`)
	if !errors.As(err, &denied) || denied.Status != 403 {
		t.Fatalf("retry ignored revoked access: %v", err)
	}
}

func TestDraftPublicationAcceptsFormOverridesAlongsideSavedTypedData(t *testing.T) {
	for _, indexed := range []bool{false, true} {
		t.Run(fmt.Sprintf("indexed=%v", indexed), func(t *testing.T) {
			f := newProjectTypeFixture(t)
			repo, draft, _ := seedPublicationDraft(t, f)
			body := url.Values{"name": {"Form publication"}}
			if indexed {
				body["assignee_ids[0]"], body["assignee_ids[1]"] = []string{f.member}, []string{f.owner}
			} else {
				body["assignee_ids"] = []string{f.member, f.owner}
			}
			result, err := repo.Execute(f.ctx, workitems.Request{Operation: draftPublishOperation, Method: "POST", ActorID: f.member, Slug: "types", Timezone: "UTC", Params: map[string]string{"draft_id": draft}}, strings.NewReader(body.Encode()), "application/x-www-form-urlencoded")
			if err != nil {
				t.Fatal(err)
			}
			var count int
			var name, description string
			if err = f.db.QueryRow(f.ctx, `SELECT name,description_html,(SELECT count(*) FROM issue_assignees WHERE issue_id=i.id AND deleted_at IS NULL) FROM issues i WHERE id=$1`, resultIssueID(t, result)).Scan(&name, &description, &count); err != nil || name != "Form publication" || description != "<p>Saved description</p>" || count != 2 {
				t.Fatalf("form publication lost values: %q %q %d %v", name, description, count, err)
			}

		})
	}
}

func TestDraftPublicationRollsBackEveryWriteOnAssetFailure(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	target := publicationTarget(t, f)
	asset := publicationAsset(f, draft, f.project)
	body := `{"project_id":"` + target + `","label_ids":[],"assignee_ids":[]}`
	f.exec(`CREATE FUNCTION reject_publication_assets() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected asset failure'; END $$; CREATE TRIGGER reject_publication_assets BEFORE UPDATE ON file_assets FOR EACH STATEMENT EXECUTE FUNCTION reject_publication_assets()`)
	if _, err := publishDraftForTest(f, repo, draft, f.member, body); err == nil || !strings.Contains(err.Error(), "injected asset failure") {
		t.Fatalf("expected late asset failure: %v", err)
	}
	var issues, sequences, labels, assignees, effects, receipts int
	var live bool
	err := f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM issues),(SELECT count(*) FROM issue_sequences),(SELECT count(*) FROM issue_labels),(SELECT count(*) FROM issue_assignees),(SELECT count(*) FROM work_item_effects),(SELECT count(*) FROM work_item_draft_publications),(SELECT deleted_at IS NULL FROM draft_issues WHERE id=$1)`, draft).Scan(&issues, &sequences, &labels, &assignees, &effects, &receipts, &live)
	if err != nil || issues+sequences+labels+assignees+effects+receipts != 0 || !live {
		t.Fatalf("publication partially committed: %d %d %d %d %d %d live=%v %v", issues, sequences, labels, assignees, effects, receipts, live, err)
	}
	var assetProject, draftProject, assetDraft string
	var unattached bool
	err = f.db.QueryRow(f.ctx, `SELECT a.project_id::text,d.project_id::text,a.draft_issue_id::text,a.issue_id IS NULL FROM file_assets a,draft_issues d WHERE a.id=$1 AND d.id=$2`, asset, draft).Scan(&assetProject, &draftProject, &assetDraft, &unattached)
	if err != nil || assetProject != f.project || draftProject != f.project || assetDraft != draft || !unattached {
		t.Fatalf("failed publication changed ownership: %s/%s/%s %v %v", assetProject, draftProject, assetDraft, unattached, err)
	}
	f.exec(`DROP TRIGGER reject_publication_assets ON file_assets; DROP FUNCTION reject_publication_assets()`)
	if _, err = publishDraftForTest(f, repo, draft, f.member, body); err != nil {
		t.Fatalf("retained draft cannot be retried: %v", err)
	}
}

func TestDraftPublicationValidatesSavedRelationsAndHonorsExplicitClearing(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, label := seedPublicationDraft(t, f)
	f.exec(`UPDATE labels SET deleted_at=now() WHERE id=$1`, label)
	_, err := publishDraftForTest(f, repo, draft, f.member, `{}`)
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 400 {
		t.Fatalf("removed saved relationship silently discarded: %v", err)
	}
	result, err := publishDraftForTest(f, repo, draft, f.member, `{"label_ids":[],"assignee_ids":[],"name":"Edited at publication"}`)
	if err != nil {
		t.Fatal(err)
	}
	id := resultIssueID(t, result)
	var name string
	var relations int
	if err = f.db.QueryRow(f.ctx, `SELECT name,(SELECT count(*) FROM issue_labels WHERE issue_id=$1)+(SELECT count(*) FROM issue_assignees WHERE issue_id=$1) FROM issues WHERE id=$1`, id).Scan(&name, &relations); err != nil || name != "Edited at publication" || relations != 0 {
		t.Fatalf("explicit overrides lost: %q relations=%d %v", name, relations, err)
	}
}
