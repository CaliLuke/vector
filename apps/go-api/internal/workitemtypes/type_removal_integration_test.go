//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"uuid"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func replacementCommand(t *testing.T, f projectTypeFixture) types.TypeRemovalInput {
	t.Helper()
	catalog, err := f.repo.Catalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	return types.TypeRemovalInput{Revision: catalog.Revision, TypeID: f.bug, ReplacementID: f.task}
}

func TestTypeRemovalCompactsHierarchyAndCarriesOrResetsDefault(t *testing.T) {
	for _, test := range []struct {
		name                          string
		defaultLevel, expectedDefault int
		remove                        func(*testing.T, projectTypeFixture)
	}{
		{"reviewed reset", 1, 0, removeReviewedType},
		{"reviewed carry", 2, 1, removeReviewedType},
		{"unused reset", 1, 0, removeUnusedType},
		{"unused carry", 2, 1, removeUnusedType},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newProjectTypeFixture(t)
			catalog, err := f.repo.Catalog(f.ctx, f.owner, "types")
			if err != nil {
				t.Fatal(err)
			}
			catalog, err = f.repo.Mutate(f.ctx, f.owner, "types", "create", "", types.Input{Revision: catalog.Revision, Name: "Epic", Icon: "epic", Color: "#123456"})
			if err != nil {
				t.Fatal(err)
			}
			epic := catalog.Types[len(catalog.Types)-1].ID
			hierarchy := types.HierarchyInput{Revision: catalog.Revision, DefaultLevel: test.defaultLevel, Placements: []types.HierarchyPlacement{{TypeID: f.task, Level: 0}, {TypeID: f.bug, Level: 1}, {TypeID: epic, Level: 2}}}
			review, err := f.repo.PreviewHierarchy(f.ctx, f.owner, "types", hierarchy)
			if err != nil {
				t.Fatal(err)
			}
			hierarchy.ReviewID = review.ID
			if _, err = f.repo.CommitHierarchy(f.ctx, f.owner, "types", hierarchy); err != nil {
				t.Fatal(err)
			}
			test.remove(t, f)
			var level, chosenDefault int
			if err = f.db.QueryRow(f.ctx, `SELECT t.level::integer,s.default_level FROM issue_types t JOIN work_item_type_settings s ON s.workspace_id=t.workspace_id WHERE t.id=$1`, epic).Scan(&level, &chosenDefault); err != nil {
				t.Fatal(err)
			}
			if level != 1 || chosenDefault != test.expectedDefault {
				t.Fatalf("hierarchy not normalized: epic=%d default=%d", level, chosenDefault)
			}
		})
	}
}

func TestTypeRemovalCannotUnlinkChildrenToMakeReplacementValid(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	in := hierarchyInput(t, f, 1)
	review, err := f.repo.PreviewHierarchy(f.ctx, f.owner, "types", in)
	if err != nil {
		t.Fatal(err)
	}
	in.ReviewID = review.ID
	if _, err = f.repo.CommitHierarchy(f.ctx, f.owner, "types", in); err != nil {
		t.Fatal(err)
	}
	source := f
	source.task = f.bug
	repo, draft, _ := seedPublicationDraft(t, source)
	result, err := publishDraftForTest(source, repo, draft, f.member, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	parent := resultIssueID(t, result)
	result, err = writeValueItem(f, repo, "POST", "", `{"name":"Child","type_id":"`+f.task+`","parent_id":"`+parent+`"}`)
	if err != nil {
		t.Fatal(err)
	}
	child := resultIssueID(t, result)
	removal, err := f.repo.PreviewTypeRemoval(f.ctx, f.owner, "types", replacementCommand(t, f))
	if err != nil || removal.ID != "" || removal.BlockedReason == "" {
		t.Fatalf("invalid hierarchy accepted: %+v %v", removal, err)
	}
	var keptParent string
	if err = f.db.QueryRow(f.ctx, `SELECT parent_id::text FROM issues WHERE id=$1`, child).Scan(&keptParent); err != nil || keptParent != parent {
		t.Fatalf("preview unlinked child: %s %v", keptParent, err)
	}
}
func TestTypeRemovalPreservesHiddenHistoricalItemsAndRetriesOnce(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	source := f
	source.task = f.bug
	rich, release, releaseID := uuid.New().String(), uuid.New().String(), uuid.New().String()
	attachValueProperties(t, source,
		property.Definition{ID: rich, Title: "Notes", Kind: property.RichText, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: release, Title: "Delivery", Kind: property.Release, Configuration: property.Configuration{Active: true}},
	)
	f.exec(`INSERT INTO work_item_releases(id,workspace_id,project_id,name) VALUES($1,$2,$3,'Private release')`, releaseID, f.workspace, f.project)
	ids := []string{}
	for _, historical := range []string{"current", "archived", "trashed"} {
		repo, draft, label := seedPublicationDraft(t, source)
		f.exec(`UPDATE labels SET name=name||id::text WHERE id=$1`, label)
		f.exec(`UPDATE draft_issues SET custom_properties=jsonb_build_object($2::text,'<p>Private notes</p>',$3::text,$4::text) WHERE id=$1`, draft, rich, release, releaseID)
		result, err := publishDraftForTest(source, repo, draft, f.member, `{}`)
		if err != nil {
			t.Fatal(err)
		}
		id := resultIssueID(t, result)
		ids = append(ids, id)
		if historical == "archived" {
			f.exec(`UPDATE issues SET archived_at=now() WHERE id=$1`, id)
		}
		if historical == "trashed" {
			f.exec(`UPDATE issues SET deleted_at=now() WHERE id=$1`, id)
		}
	}
	_, privateDraft, _ := seedPublicationDraft(t, source)
	f.exec(`UPDATE draft_issues SET project_id=NULL,custom_properties=jsonb_build_object($2::text,'<p>Projectless private notes</p>') WHERE id=$1`, privateDraft, rich)
	f.exec(`UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, f.project, f.owner)
	in := replacementCommand(t, f)
	var versionsBefore int
	if err := f.db.QueryRow(f.ctx, `SELECT count(*) FROM work_item_property_versions WHERE property_id=$1`, rich).Scan(&versionsBefore); err != nil {
		t.Fatal(err)
	}
	review, err := f.repo.PreviewTypeRemoval(f.ctx, f.owner, "types", in)
	if err != nil || review.ID == "" || review.Items != 3 || review.Drafts != 4 || review.Preserved != 7 || review.InvalidItems != 0 {
		t.Fatalf("review: %+v %v", review, err)
	}
	raw, err := json.Marshal(review)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "Private") || strings.Contains(string(raw), privateDraft) || strings.Contains(string(raw), ids[0]) {
		t.Fatalf("private preview data: %s", raw)
	}
	var original int
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FROM issues WHERE type_id=$1`, f.bug).Scan(&original); err != nil || original != 3 {
		t.Fatalf("preview wrote source: %d %v", original, err)
	}
	in.ReviewID = review.ID
	if _, err = f.repo.CommitTypeRemoval(f.ctx, f.owner, "types", in); err != nil {
		t.Fatal(err)
	}
	f.exec(`UPDATE work_item_schema_reviews SET created_at=now()-interval '31 minutes' WHERE id=$1`, review.ID)
	if _, err = f.repo.CommitTypeRemoval(f.ctx, f.owner, "types", in); err != nil {
		t.Fatal(err)
	}
	var migrated, activity, versions, leftovers int
	if err = f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM issues WHERE type_id=$1 AND description_html LIKE '%Private release%' AND description_html LIKE '%Private notes%'),(SELECT count(*) FROM issue_activities WHERE field='type'),(SELECT count(*) FROM work_item_property_versions WHERE property_id=$2),(SELECT count(*) FROM draft_issues WHERE type_id=$3)`, f.task, rich, f.bug).Scan(&migrated, &activity, &versions, &leftovers); err != nil {
		t.Fatal(err)
	}
	if migrated != 3 || activity != 3 || versions != versionsBefore+3 || leftovers != 0 {
		t.Fatalf("migration/retry effects: %d %d %d %d", migrated, activity, versions, leftovers)
	}
}

func TestTypeRemovalBlocksInvalidDestinationAndStaleReview(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	source := f
	source.task = f.bug
	repo, draft, _ := seedPublicationDraft(t, source)
	result, err := publishDraftForTest(source, repo, draft, f.member, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	item := resultIssueID(t, result)
	in := replacementCommand(t, f)
	review, err := f.repo.PreviewTypeRemoval(f.ctx, f.owner, "types", in)
	if err != nil || review.ID == "" {
		t.Fatalf("preview: %+v %v", review, err)
	}
	in.ReviewID = review.ID
	f.exec(`UPDATE issues SET description_html='<p>Concurrent edit</p>' WHERE id=$1`, item)
	_, err = f.repo.CommitTypeRemoval(f.ctx, f.owner, "types", in)
	requireTypeStatus(t, err, 409)
	var conflict *types.Error
	if !errors.As(err, &conflict) || !conflict.ReviewAgain {
		t.Fatalf("uncommitted conflict did not permit fresh review: %v", err)
	}
	review, err = f.repo.PreviewTypeRemoval(f.ctx, f.owner, "types", in)
	if err != nil {
		t.Fatal(err)
	}
	in.ReviewID = review.ID
	f.exec(`UPDATE work_item_schema_reviews SET created_at=now()-interval '31 minutes' WHERE id=$1`, review.ID)
	_, err = f.repo.CommitTypeRemoval(f.ctx, f.owner, "types", in)
	if !errors.As(err, &conflict) || !conflict.ReviewAgain {
		t.Fatalf("expired uncommitted review could not be refreshed: %v", err)
	}
	_, err = f.repo.CommitTypeRemoval(f.ctx, f.member, "types", in)
	if !errors.As(err, &conflict) || conflict.ReviewAgain {
		t.Fatalf("unauthorized request received recovery proof: %v", err)
	}
	required := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: required, Title: "Required effort", Kind: property.Number, Configuration: property.Configuration{Active: true, Required: true}})
	in = replacementCommand(t, f)
	review, err = f.repo.PreviewTypeRemoval(f.ctx, f.owner, "types", in)
	if err != nil || review.ID != "" || review.InvalidItems != 1 || review.BlockedReason == "" {
		t.Fatalf("invalid replacement review: %+v %v", review, err)
	}
	var typeID string
	if err = f.db.QueryRow(f.ctx, `SELECT type_id::text FROM issues WHERE id=$1`, item).Scan(&typeID); err != nil || typeID != f.bug {
		t.Fatalf("blocked replacement changed item: %s %v", typeID, err)
	}
	in.TypeID, in.ReplacementID = f.task, f.bug
	_, err = f.repo.PreviewTypeRemoval(f.ctx, f.owner, "types", in)
	requireTypeStatus(t, err, 422)
	_, err = f.repo.PreviewTypeRemoval(f.ctx, f.member, "types", replacementCommand(t, f))
	requireTypeStatus(t, err, 403)
}

func removeReviewedType(t *testing.T, f projectTypeFixture) {
	t.Helper()
	in := replacementCommand(t, f)
	review, err := f.repo.PreviewTypeRemoval(f.ctx, f.owner, "types", in)
	if err != nil || review.ID == "" {
		t.Fatalf("preview: %+v %v", review, err)
	}
	in.ReviewID = review.ID
	if _, err = f.repo.CommitTypeRemoval(f.ctx, f.owner, "types", in); err != nil {
		t.Fatal(err)
	}
}
func removeUnusedType(t *testing.T, f projectTypeFixture) {
	t.Helper()
	in := replacementCommand(t, f)
	if _, err := f.repo.Mutate(f.ctx, f.owner, "types", "delete", f.bug, types.Input{Revision: in.Revision}); err != nil {
		t.Fatal(err)
	}
}
func TestUnusedTypeRemovalKeepsHierarchyDisabled(t *testing.T) {
	f := newProjectTypeFixture(t)
	removeUnusedType(t, f)
	catalog, err := f.repo.HierarchyCatalog(f.ctx, f.owner, "types")
	if err != nil || catalog.HierarchyEnabled || len(catalog.Types) != 1 || catalog.DefaultLevel != 0 {
		t.Fatalf("deletion enabled hierarchy or retained deleted type: %+v %v", catalog, err)
	}
}
