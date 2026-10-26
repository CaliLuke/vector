//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/google/uuid"
)

func propertyRestoreFixture(t *testing.T) (projectTypeFixture, *workitems.Repository, workitems.Request, workitems.ItemEditInput, string) {
	t.Helper()
	f, repo, q, input, field := reviewedEditFixture(t)
	version := uuid.NewString()
	f.exec(`INSERT INTO work_item_property_versions(id,workspace_id,issue_id,property_id,value,search_text,property_title,source,actor_id) VALUES($1,$2,$3,$4,'"<p>Earlier value</p>"','Earlier value','Old details','change',$5)`, version, f.workspace, input.ItemID, field, f.member)
	input.Changes = map[string]any{}
	input.Restore = &workitems.ItemPropertyRestore{PropertyID: field, VersionID: version}
	return f, repo, q, input, field
}

func reviewPropertyRestore(t *testing.T, f projectTypeFixture, repo *workitems.Repository, q workitems.Request, input *workitems.ItemEditInput) {
	t.Helper()
	preview, err := repo.PreviewItemChange(f.ctx, q, *input)
	if err != nil {
		t.Fatal(err)
	}
	input.ReviewID = preview.Body.(workitems.ItemEditReview).ID
}

func assertRestoreValue(t *testing.T, f projectTypeFixture, input workitems.ItemEditInput, expected any) {
	t.Helper()
	var raw []byte
	if err := f.db.QueryRow(f.ctx, `SELECT custom_properties->$2 FROM issues WHERE id=$1`, input.ItemID, input.Restore.PropertyID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var actual any
	if err := json.Unmarshal(raw, &actual); err != nil || actual != expected {
		t.Fatalf("restored=%v want=%v: %v", actual, expected, err)
	}
}

func TestPropertyRestorePreviewCommitAndExactRetry(t *testing.T) {
	f, repo, q, input, _ := propertyRestoreFixture(t)
	reviewPropertyRestore(t, f, repo, q, &input)
	assertRestoreValue(t, f, input, "<p>Preserve this value</p>")
	assertHistoryCounts(t, f, input.ItemID, 1, 0, 0)
	var bound bool
	if err := f.db.QueryRow(f.ctx, `SELECT command->'changes'='{}'::jsonb AND command->'restore'->>'version_id'=$2 FROM work_item_schema_reviews WHERE id=$1`, input.ReviewID, input.Restore.VersionID).Scan(&bound); err != nil || !bound {
		t.Fatalf("receipt lost immutable restore command: %v %v", bound, err)
	}
	if _, err := repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatal(err)
	}
	assertRestoreValue(t, f, input, "<p>Earlier value</p>")
	assertHistoryCounts(t, f, input.ItemID, 3, 1, 0)
	var restored, original, activity bool
	err := f.db.QueryRow(f.ctx, `SELECT
 (SELECT count(*)=1 FROM work_item_property_versions WHERE issue_id=$1 AND source='restore' AND actor_id=$2 AND value='"<p>Earlier value</p>"'),
 (SELECT value='"<p>Earlier value</p>"' AND source='change' FROM work_item_property_versions WHERE id=$3),
 (SELECT count(*)=1 FROM issue_activities WHERE issue_id=$1 AND field='custom_property' AND verb='restored')`, input.ItemID, q.ActorID, input.Restore.VersionID).Scan(&restored, &original, &activity)
	if err != nil || !restored || !original || !activity {
		t.Fatalf("restore provenance=%v original=%v activity=%v: %v", restored, original, activity, err)
	}
	f.exec(`UPDATE issues SET name='Newer title' WHERE id=$1`, input.ItemID)
	f.exec(`UPDATE work_item_type_settings SET revision=revision+1 WHERE workspace_id=$1`, f.workspace)
	if _, err = repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatal(err)
	}
	assertHistoryCounts(t, f, input.ItemID, 3, 1, 0)
	input.Restore.VersionID = uuid.NewString()
	_, err = repo.CommitItemChange(f.ctx, q, input)
	requireEditStatus(t, err, 409)
}

func TestPropertyRestoreRejectsCurrentCanonicalValue(t *testing.T) {
	f, repo, q, input, _ := propertyRestoreFixture(t)
	f.exec(`UPDATE work_item_property_versions SET value=to_jsonb('<p>Preserve this value</p><script>bad()</script>'::text) WHERE id=$1`, input.Restore.VersionID)
	_, err := repo.PreviewItemChange(f.ctx, q, input)
	requireEditStatus(t, err, 409)
	assertHistoryCounts(t, f, input.ItemID, 1, 0, 0)
}

func TestPropertyRestoreSanitizesHistoricalHTML(t *testing.T) {
	f, repo, q, input, _ := propertyRestoreFixture(t)
	f.exec(`UPDATE work_item_property_versions SET value=to_jsonb('<p>Earlier value</p><script>bad()</script>'::text) WHERE id=$1`, input.Restore.VersionID)
	reviewPropertyRestore(t, f, repo, q, &input)
	if _, err := repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatal(err)
	}
	assertRestoreValue(t, f, input, "<p>Earlier value</p>")
}

func TestPropertyRestoreAllowsClearUnlessCurrentlyRequired(t *testing.T) {
	for _, required := range []bool{false, true} {
		t.Run(map[bool]string{false: "optional", true: "required"}[required], func(t *testing.T) {
			f, repo, q, input, field := propertyRestoreFixture(t)
			f.exec(`UPDATE work_item_property_versions SET value='null'::jsonb WHERE id=$1`, input.Restore.VersionID)
			f.exec(`UPDATE work_item_properties SET definition=jsonb_set(definition,'{configuration,required}',to_jsonb($2::bool)) WHERE id=$1`, field, required)
			preview, err := repo.PreviewItemChange(f.ctx, q, input)
			if required {
				requireEditStatus(t, err, 400)
				assertHistoryCounts(t, f, input.ItemID, 1, 0, 0)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			input.ReviewID = preview.Body.(workitems.ItemEditReview).ID
			if _, err = repo.CommitItemChange(f.ctx, q, input); err != nil {
				t.Fatal(err)
			}
			assertRestoreValue(t, f, input, nil)
			assertHistoryCounts(t, f, input.ItemID, 3, 1, 0)
		})
	}
}

func TestPropertyRestoreRejectsUnavailableCurrentField(t *testing.T) {
	for _, change := range []string{"detached", "deleted", "inactive", "context inactive", "non-rich"} {
		t.Run(change, func(t *testing.T) {
			f, repo, q, input, field := propertyRestoreFixture(t)
			switch change {
			case "detached":
				f.exec(`DELETE FROM work_item_type_properties WHERE property_id=$1`, field)
			case "deleted":
				f.exec(`DELETE FROM work_item_properties WHERE id=$1`, field)
			case "context inactive":
				contexts, _ := json.Marshal([]map[string]any{{"id": uuid.NewString(), "name": "Inactive here", "project_ids": []string{f.project}, "type_ids": []string{f.task}, "override_keys": []string{"active"}, "configuration": map[string]any{"active": false}}})
				f.exec(`UPDATE work_item_properties SET definition=jsonb_set(definition,'{contexts}',$2::jsonb) WHERE id=$1`, field, contexts)
			case "inactive":
				f.exec(`UPDATE work_item_properties SET definition=jsonb_set(definition,'{configuration,active}','false') WHERE id=$1`, field)
			case "non-rich":
				f.exec(`UPDATE work_item_properties SET definition=jsonb_set(definition,'{kind}','"Text"') WHERE id=$1`, field)
			}
			_, err := repo.PreviewItemChange(f.ctx, q, input)
			requireEditStatus(t, err, 409)
			assertHistoryCounts(t, f, input.ItemID, 1, 0, 0)
		})
	}
}

func TestPropertyRestoreRejectsForeignRevisionScope(t *testing.T) {
	for _, scope := range []string{"workspace", "project", "item", "property", "version"} {
		t.Run(scope, func(t *testing.T) {
			f, repo, q, input, _ := propertyRestoreFixture(t)
			switch scope {
			case "workspace":
				other := uuid.NewString()
				f.exec(`INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Other',$2,'other','UTC','',now(),now())`, other, f.owner)
				f.exec(`UPDATE work_item_property_versions SET workspace_id=$2 WHERE id=$1`, input.Restore.VersionID, other)
			case "project":
				input.ProjectID = uuid.NewString()
			case "item":
				f.exec(`UPDATE work_item_property_versions SET issue_id=$2 WHERE id=$1`, input.Restore.VersionID, addHierarchyItem(t, f, f.task, ""))
			case "property":
				f.exec(`UPDATE work_item_property_versions SET property_id=$2 WHERE id=$1`, input.Restore.VersionID, uuid.NewString())
			case "version":
				input.Restore.VersionID = uuid.NewString()
			}
			_, err := repo.PreviewItemChange(f.ctx, q, input)
			if err == nil {
				t.Fatal("foreign revision scope accepted")
			}
		})
	}
}

func TestPropertyRestoreRejectsStaleSourceAndSchema(t *testing.T) {
	for _, change := range []string{"source", "schema"} {
		t.Run(change, func(t *testing.T) {
			f, repo, q, input, _ := propertyRestoreFixture(t)
			reviewPropertyRestore(t, f, repo, q, &input)
			if change == "source" {
				f.exec(`UPDATE issues SET name='Newer title' WHERE id=$1`, input.ItemID)
			} else {
				f.exec(`UPDATE work_item_type_settings SET revision=revision+1 WHERE workspace_id=$1`, f.workspace)
			}
			_, err := repo.CommitItemChange(f.ctx, q, input)
			requireEditStatus(t, err, 409)
			assertRestoreValue(t, f, input, "<p>Preserve this value</p>")
			assertHistoryCounts(t, f, input.ItemID, 1, 0, 0)
		})
	}
}

func TestPropertyRestoreRollsBackLateFailure(t *testing.T) {
	f, repo, q, input, _ := propertyRestoreFixture(t)
	reviewPropertyRestore(t, f, repo, q, &input)
	f.exec(`CREATE FUNCTION reject_restore_effect() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'late effect failure'; END $$; CREATE TRIGGER reject_restore_effect BEFORE INSERT ON work_item_effects FOR EACH ROW EXECUTE FUNCTION reject_restore_effect()`)
	if _, err := repo.CommitItemChange(f.ctx, q, input); err == nil {
		t.Fatal("expected rejected effect")
	}
	assertRestoreValue(t, f, input, "<p>Preserve this value</p>")
	assertHistoryCounts(t, f, input.ItemID, 1, 0, 0)
	f.exec(`DROP TRIGGER reject_restore_effect ON work_item_effects`)
	if _, err := repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatal(err)
	}
	assertHistoryCounts(t, f, input.ItemID, 3, 1, 0)
}

func TestPropertyRestoreRejectsMissingHistoricalAsset(t *testing.T) {
	f, repo, q, input, _ := propertyRestoreFixture(t)
	value := `<p>Earlier image</p><img src="` + uuid.NewString() + `">`
	f.exec(`UPDATE work_item_property_versions SET value=to_jsonb($2::text) WHERE id=$1`, input.Restore.VersionID, value)
	_, err := repo.PreviewItemChange(f.ctx, q, input)
	requireEditStatus(t, err, 400)
	detail, _ := json.Marshal(err)
	if !strings.Contains(string(detail), "image") {
		t.Fatal(err)
	}
	assertHistoryCounts(t, f, input.ItemID, 1, 0, 0)
}

func TestPropertyRestoreRejectsCommittedRetryAfterProjectRemoval(t *testing.T) {
	f, repo, q, input, _ := propertyRestoreFixture(t)
	f.exec(`UPDATE issues SET created_by_id=$2 WHERE id=$1`, input.ItemID, q.ActorID)
	setEditSourceVersion(t, f, repo, q, &input)
	reviewPropertyRestore(t, f, repo, q, &input)
	if _, err := repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatal(err)
	}
	f.exec(`UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, f.project, q.ActorID)
	result, err := repo.CommitItemChange(f.ctx, q, input)
	if err == nil || result.Body != nil {
		t.Fatalf("revoked creator received committed projection: %+v %v", result, err)
	}
	assertHistoryCounts(t, f, input.ItemID, 3, 1, 0)
}

func TestPropertyRestoreRejectsRestrictedGuestAndRevokedMembership(t *testing.T) {
	for _, denial := range []string{"guest", "workspace", "project"} {
		t.Run(denial, func(t *testing.T) {
			f, repo, q, input, _ := propertyRestoreFixture(t)
			switch denial {
			case "guest":
				f.exec(`UPDATE project_members SET role=5 WHERE member_id=$1`, q.ActorID)
				f.exec(`UPDATE issues SET created_by_id=NULL; UPDATE projects SET guest_view_all_features=false`)
			case "workspace":
				f.exec(`UPDATE workspace_members SET is_active=false WHERE member_id=$1`, q.ActorID)
			case "project":
				f.exec(`UPDATE project_members SET is_active=false WHERE member_id=$1`, q.ActorID)
			}
			if _, err := repo.PreviewItemChange(f.ctx, q, input); err == nil {
				t.Fatal("denied actor restored content")
			}
			assertHistoryCounts(t, f, input.ItemID, 1, 0, 0)
		})
	}
}
