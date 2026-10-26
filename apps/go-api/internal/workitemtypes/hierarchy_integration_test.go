//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"strings"
	"testing"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
)

func hierarchyInput(t *testing.T, f projectTypeFixture, bugLevel int) types.HierarchyInput {
	t.Helper()
	catalog, err := f.repo.HierarchyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	return types.HierarchyInput{Revision: catalog.Revision, Placements: []types.HierarchyPlacement{{TypeID: f.task, Level: 0}, {TypeID: f.bug, Level: bugLevel}}}
}
func TestHierarchyFirstSaveCountsPrivateUntitledDraftAndRetriesOnce(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	parentResult, err := writeValueItem(f, repo, "POST", "", `{"name":"Parent"}`)
	if err != nil {
		t.Fatal(err)
	}
	parent := resultIssueID(t, parentResult)
	childResult, err := writeValueItem(f, repo, "POST", "", `{"name":"Child","parent_id":"`+parent+`"}`)
	if err != nil {
		t.Fatal(err)
	}
	child := resultIssueID(t, childResult)
	f.exec(`UPDATE draft_issues SET name=NULL,project_id=NULL,parent_id=$2 WHERE id=$1`, draft, parent)
	input := hierarchyInput(t, f, 0)
	review, err := f.repo.PreviewHierarchy(f.ctx, f.owner, "types", input)
	if err != nil {
		t.Fatal(err)
	}
	if review.Links != 2 || review.RestrictedLinks != 1 || len(review.Pairs) != 1 || review.Pairs[0].ChildID != child {
		t.Fatalf("incorrect visible/private impact: %+v", review)
	}
	raw, _ := json.Marshal(review)
	if strings.Contains(string(raw), draft) {
		t.Fatal("private draft identity exposed")
	}
	before, err := f.repo.HierarchyCatalog(f.ctx, f.owner, "types")
	if err != nil || before.HierarchyEnabled || before.Revision != input.Revision {
		t.Fatalf("preview changed hierarchy: %+v %v", before, err)
	}
	input.ReviewID = review.ID
	input.Confirm = "Confirm"
	_, err = f.repo.CommitHierarchy(f.ctx, f.owner, "types", input)
	requireTypeStatus(t, err, 422)
	input.Confirm = "confirm"
	saved, err := f.repo.CommitHierarchy(f.ctx, f.owner, "types", input)
	if err != nil || !saved.HierarchyEnabled || saved.Revision != input.Revision+1 {
		t.Fatalf("first save: %+v %v", saved, err)
	}
	if _, err = f.repo.CommitHierarchy(f.ctx, f.owner, "types", input); err != nil {
		t.Fatalf("exact retry: %v", err)
	}
	var links, activities int
	err = f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM issues WHERE id=$1 AND parent_id IS NOT NULL)+(SELECT count(*) FROM draft_issues WHERE id=$2 AND parent_id IS NOT NULL),(SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field='parent')`, child, draft).Scan(&links, &activities)
	if err != nil || links != 0 || activities != 1 {
		t.Fatalf("links=%d activities=%d err=%v", links, activities, err)
	}
	// A later catalog revision must not make an already committed receipt repeat
	// its effects, reject its old placements, or overwrite the new configuration.
	_, err = f.repo.Mutate(f.ctx, f.owner, "types", "create", "", types.Input{Revision: saved.Revision, Name: "Later", Icon: "task", Color: "#123456"})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := f.repo.CommitHierarchy(f.ctx, f.owner, "types", input)
	if err != nil || len(replay.Types) != 3 || replay.Revision != saved.Revision+1 {
		t.Fatalf("later retry changed configuration: %+v %v", replay, err)
	}
}

func TestHierarchyReviewBindsHiddenGraphAndExactConfiguration(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	result, err := writeValueItem(f, repo, "POST", "", `{"name":"Parent"}`)
	if err != nil {
		t.Fatal(err)
	}
	parent := resultIssueID(t, result)
	input := hierarchyInput(t, f, 1)
	review, err := f.repo.PreviewHierarchy(f.ctx, f.owner, "types", input)
	if err != nil {
		t.Fatal(err)
	}
	input.ReviewID = review.ID
	_, err = f.repo.CommitHierarchy(f.ctx, f.member, "types", input)
	requireTypeStatus(t, err, 403)
	changed := input
	changed.DefaultLevel = 1
	_, err = f.repo.CommitHierarchy(f.ctx, f.owner, "types", changed)
	requireTypeStatus(t, err, 409)
	f.exec(`UPDATE draft_issues SET parent_id=$2 WHERE id=$1`, draft, parent)
	_, err = f.repo.CommitHierarchy(f.ctx, f.owner, "types", input)
	requireTypeStatus(t, err, 409)
	var enabled bool
	if err = f.db.QueryRow(f.ctx, `SELECT hierarchy_enabled FROM work_item_type_settings WHERE workspace_id=$1`, f.workspace).Scan(&enabled); err != nil || enabled {
		t.Fatalf("stale review activated hierarchy: %t %v", enabled, err)
	}
	review, err = f.repo.PreviewHierarchy(f.ctx, f.owner, "types", input)
	if err != nil {
		t.Fatal(err)
	}
	input.ReviewID = review.ID
	input.Confirm = "confirm"
	f.exec(`UPDATE work_item_schema_reviews SET created_at=clock_timestamp()-interval '31 minutes' WHERE id=$1`, review.ID)
	_, err = f.repo.CommitHierarchy(f.ctx, f.owner, "types", input)
	requireTypeStatus(t, err, 409)
}
