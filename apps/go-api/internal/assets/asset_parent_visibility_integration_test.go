//go:build integration

package assets

import (
	"testing"

	"github.com/google/uuid"
)

func TestGenericAssetReadsAndMutationsRequireVisibleIssueCommentOrPage(t *testing.T) {
	f := newDraftAssetFixture(t)
	f.exec(`INSERT INTO projects(id,workspace_id) VALUES($1,$2)`, f.project, f.workspace)
	issue, comment, page := uuid.New(), uuid.New(), uuid.New()
	otherProject := uuid.New()
	f.exec(`INSERT INTO projects(id,workspace_id) VALUES($1,$2)`, otherProject, f.workspace)
	f.exec(`INSERT INTO issues(id,workspace_id,project_id,created_by_id) VALUES($1,$2,$3,$4)`, issue, f.workspace, f.project, f.other)
	f.exec(`INSERT INTO issue_comments(id,workspace_id,project_id,issue_id) VALUES($1,$2,$3,$4)`, comment, f.workspace, f.project, issue)
	f.exec(`INSERT INTO pages(id,workspace_id,owned_by_id,access) VALUES($1,$2,$3,0)`, page, f.workspace, f.other)
	f.exec(`INSERT INTO project_pages(page_id,project_id) VALUES($1,$2)`, page, f.project)

	issueAsset, commentAsset, pageAsset := f.asset("ISSUE_DESCRIPTION", f.actor), f.asset("COMMENT_DESCRIPTION", f.actor), f.asset("PAGE_DESCRIPTION", f.actor)
	f.exec(`UPDATE file_assets SET project_id=$2,issue_id=$3 WHERE id=$1`, issueAsset, f.project, issue)
	f.exec(`UPDATE file_assets SET project_id=$2,comment_id=$3 WHERE id=$1`, commentAsset, f.project, comment)
	f.exec(`UPDATE file_assets SET project_id=$2,page_id=$3 WHERE id=$1`, pageAsset, f.project, page)
	commentUpload := f.asset("COMMENT_DESCRIPTION", f.actor)
	f.exec(`UPDATE file_assets SET project_id=$2 WHERE id=$1`, commentUpload, f.project)
	if err := f.repository.BulkAssociateProjectAssets(f.ctx, f.workspace, f.project, f.actor, comment, []uuid.UUID{commentUpload}); err != nil {
		t.Fatalf("associate upload with visible comment: %v", err)
	}
	var associatedComment uuid.UUID
	if err := f.db.QueryRow(f.ctx, `SELECT comment_id FROM file_assets WHERE id=$1`, commentUpload).Scan(&associatedComment); err != nil || associatedComment != comment {
		t.Fatalf("comment upload association = %s, %v; want %s", associatedComment, err, comment)
	}

	for _, assetID := range []uuid.UUID{issueAsset, commentAsset, pageAsset} {
		visible, err := f.repository.AssetParentVisible(f.ctx, f.actor, f.workspace, assetID)
		if err != nil || !visible {
			t.Fatalf("member parent visibility for %s = %v, %v; want visible", assetID, visible, err)
		}
	}

	// A page may link to several projects. Its asset's project context must be
	// checked against that exact link, not another linked project where the
	// actor happens to have membership.
	f.exec(`INSERT INTO project_pages(page_id,project_id) VALUES($1,$2)`, page, otherProject)
	f.exec(`INSERT INTO project_members(workspace_id,project_id,member_id,role,is_active,deleted_at) VALUES($1,$2,$3,15,true,NULL)`, f.workspace, otherProject, f.actor)
	f.exec(`UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, f.project, f.actor)
	visible, err := f.repository.AssetParentVisible(f.ctx, f.actor, f.workspace, pageAsset)
	if err != nil || visible {
		t.Fatalf("page asset borrowed visibility from another project link = %v, %v; want hidden", visible, err)
	}
	f.exec(`UPDATE project_members SET is_active=true WHERE project_id=$1 AND member_id=$2`, f.project, f.actor)

	// Guests can still read an issue created by them and pages the route exposes,
	// while a private page remains unavailable until the actor owns it.
	f.exec(`UPDATE project_members SET role=5 WHERE member_id=$1`, f.actor)
	visible, err = f.repository.AssetParentVisible(f.ctx, f.actor, f.workspace, issueAsset)
	if err != nil || visible {
		t.Fatalf("guest read of another member's issue = %v, %v; want hidden", visible, err)
	}
	visible, err = f.repository.AssetParentVisible(f.ctx, f.actor, f.workspace, commentAsset)
	if err != nil || visible {
		t.Fatalf("guest read of another member's comment = %v, %v; want hidden", visible, err)
	}
	f.exec(`UPDATE pages SET access=1 WHERE id=$1`, page)
	visible, err = f.repository.AssetParentVisible(f.ctx, f.actor, f.workspace, pageAsset)
	if err != nil || visible {
		t.Fatalf("guest read of private page asset = %v, %v; want hidden", visible, err)
	}
	f.exec(`UPDATE pages SET owned_by_id=$2 WHERE id=$1`, page, f.actor)
	visible, err = f.repository.AssetParentVisible(f.ctx, f.actor, f.workspace, pageAsset)
	if err != nil || !visible {
		t.Fatalf("page owner access = %v, %v; want visible", visible, err)
	}

	// The endpoint statement itself still rejects a stale/hidden target after a
	// service-side preflight, and does not insert an asset row.
	newAsset := uuid.New()
	err = f.repository.CreateWorkspaceEntityUpload(f.ctx, newAsset, f.actor, f.workspace, f.project, "hidden.png", []byte(`{"name":"hidden.png"}`), 12, "ISSUE_DESCRIPTION", issue.String(), true)
	assertScopeNotFound(t, err, "create asset for hidden issue")
	var inserted bool
	if scanErr := f.db.QueryRow(f.ctx, `SELECT EXISTS(SELECT 1 FROM file_assets WHERE id=$1)`, newAsset).Scan(&inserted); scanErr != nil || inserted {
		t.Fatalf("hidden parent upload inserted=%v err=%v", inserted, scanErr)
	}
	assertScopeNotFound(t, f.repository.UpdateWorkspaceAssetUploaded(f.ctx, f.actor, f.workspace, issueAsset, true, false), "confirm asset for hidden issue")
	rows, err := f.repository.WorkspaceAssetsByKey(f.ctx, "image.png", f.workspace, f.actor)
	if err != nil || len(rows) != 1 || rows[0]["id"] != pageAsset.String() {
		t.Fatalf("legacy key listing for hidden parents = %#v, %v; want only visible page asset", rows, err)
	}
}
