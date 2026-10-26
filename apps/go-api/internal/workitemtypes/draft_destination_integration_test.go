//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
)

func publicationTarget(t *testing.T, f projectTypeFixture) string {
	t.Helper()
	target := uuid.New().String()
	f.addProject(target)
	f.exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,15,true,now(),now(),'{}','{}','{}',0)`, f.workspace, target, f.member)
	return target
}

func TestDraftPublicationChoosesDestinationWithoutSavingDraftFirst(t *testing.T) {
	for _, source := range []string{"projectless", "revoked", "deleted"} {
		t.Run(source, func(t *testing.T) {
			f := newProjectTypeFixture(t)
			repo, draft, _ := seedPublicationDraft(t, f)
			target := publicationTarget(t, f)
			if source == "projectless" {
				f.exec(`UPDATE draft_issues SET project_id=NULL WHERE id=$1`, draft)
			}
			if source == "revoked" {
				f.exec(`UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
			}
			if source == "deleted" {
				f.exec(`UPDATE projects SET deleted_at=now() WHERE id=$1`, f.project)
			}
			var sourceProject any = f.project
			if source == "projectless" {
				sourceProject = nil
			}
			asset := publicationAsset(f, draft, sourceProject)
			schema, err := f.repo.ItemSchema(f.ctx, f.member, "types", types.ItemSchemaInput{Mode: "publish", ProjectID: target, DraftID: draft})
			if err != nil || schema.TypeID != f.task {
				t.Fatalf("destination schema: %+v %v", schema, err)
			}
			body, _ := json.Marshal(map[string]any{"project_id": strings.ToUpper(target), "label_ids": []string{}, "assignee_ids": []string{}})
			result, err := publishDraftForTest(f, repo, draft, f.member, string(body))
			if err != nil {
				t.Fatal(err)
			}
			issue := resultIssueID(t, result)
			var project, assetProject, assetIssue, assetIdentifier, entityType string
			var consumed, detached bool
			err = f.db.QueryRow(f.ctx, `SELECT i.project_id::text,a.project_id::text,a.issue_id::text,a.entity_identifier,a.entity_type,a.draft_issue_id IS NULL,d.deleted_at IS NOT NULL FROM issues i,file_assets a,draft_issues d WHERE i.id=$1 AND a.id=$2 AND d.id=$3`, issue, asset, draft).Scan(&project, &assetProject, &assetIssue, &assetIdentifier, &entityType, &detached, &consumed)
			if err != nil || project != target || assetProject != target || assetIssue != issue || assetIdentifier != issue || entityType != "ISSUE_DESCRIPTION" || !detached || !consumed {
				t.Fatalf("destination publication incomplete: project=%s asset=%s/%s/%s/%s detached=%v consumed=%v %v", project, assetProject, assetIssue, assetIdentifier, entityType, detached, consumed, err)
			}
			retry, err := publishDraftForTest(f, repo, draft, f.member, `{"project_id":"invalid-retry-target"}`)
			if err != nil || resultIssueID(t, retry) != issue {
				t.Fatalf("retry changed destination: %+v %v", retry, err)
			}
		})
	}
}

func TestDraftDestinationErrorsPreserveOriginalDraft(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	target := publicationTarget(t, f)
	for _, body := range []string{
		`{"project_id":null}`, `{"project_id":""}`, `{"project_id":"bad"}`,
		`{"project_id":"` + target + `","project":"` + f.project + `"}`,
		`{"project_id":"` + target + `"}`,
	} {
		_, err := publishDraftForTest(f, repo, draft, f.member, body)
		var failure *djangoapi.HTTPError
		if !errors.As(err, &failure) || failure.Status != 400 {
			t.Fatalf("invalid destination or old relation accepted: %s %v", body, err)
		}
	}
	f.exec(`UPDATE project_members SET is_active=false WHERE project_id=$1`, target)
	_, err := publishDraftForTest(f, repo, draft, f.member, `{"project_id":"`+target+`","label_ids":[],"assignee_ids":[]}`)
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 403 {
		t.Fatalf("denied destination accepted: %v", err)
	}
	var original string
	var live bool
	var issues, receipts int
	err = f.db.QueryRow(f.ctx, `SELECT project_id::text,deleted_at IS NULL,(SELECT count(*) FROM issues),(SELECT count(*) FROM work_item_draft_publications) FROM draft_issues WHERE id=$1`, draft).Scan(&original, &live, &issues, &receipts)
	if err != nil || original != f.project || !live || issues != 0 || receipts != 0 {
		t.Fatalf("invalid destination mutated draft: %s %v %d %d %v", original, live, issues, receipts, err)
	}
}

func TestDraftPublicationCanClearAmbiguousSavedCycles(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	for range 2 {
		cycle := uuid.New().String()
		f.exec(`INSERT INTO cycles(id,name,description,owned_by_id,project_id,workspace_id,view_props,sort_order,progress_snapshot,logo_props,timezone,version,created_at,updated_at) VALUES($1,'Cycle','',$2,$3,$4,'{}',0,'{}','{}','UTC',1,now(),now())`, cycle, f.member, f.project, f.workspace)
		f.exec(`INSERT INTO draft_issue_cycles(id,draft_issue_id,cycle_id,project_id,workspace_id,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,now(),now())`, draft, cycle, f.project, f.workspace)
	}
	result, err := publishDraftForTest(f, repo, draft, f.member, `{"cycle_id":null}`)
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FROM cycle_issues WHERE issue_id=$1`, resultIssueID(t, result)).Scan(&count); err != nil || count != 0 {
		t.Fatalf("explicit clear ignored: %d %v", count, err)
	}
}

func publicationAsset(f projectTypeFixture, draft string, project any) string {
	asset := uuid.New().String()
	f.exec(`INSERT INTO file_assets(id,created_at,updated_at,attributes,asset,workspace_id,project_id,draft_issue_id,entity_identifier,entity_type,size,is_uploaded,is_deleted,is_archived,created_by_id) VALUES($1,now(),now(),'{}','draft-image',$2,$3,$4::uuid,$4::uuid::text,'DRAFT_ISSUE_DESCRIPTION',1,true,false,false,$5)`, asset, f.workspace, project, draft, f.member)
	return asset
}
