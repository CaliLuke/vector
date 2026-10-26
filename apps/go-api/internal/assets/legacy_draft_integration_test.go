//go:build integration

package assets

import (
	"testing"

	"github.com/google/uuid"
)

func TestLegacyDraftUploadsEnforceOwnerScopeAndExclusiveBinding(t *testing.T) {
	f := newDraftAssetFixture(t)
	for _, tc := range []struct {
		name             string
		actor, workspace uuid.UUID
		extra            string
	}{
		{"foreign owner", f.other, f.workspace, ""},
		{"foreign workspace", f.actor, uuid.New(), ""},
		{"issue binding", f.actor, f.workspace, "issue"},
		{"comment binding", f.actor, f.workspace, "comment"},
		{"page binding", f.actor, f.workspace, "page"},
		{"user binding", f.actor, f.workspace, "user"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fields := map[string]any{"workspace": tc.workspace, "draft_issue": f.draft, "is_uploaded": true}
			if tc.extra != "" {
				fields[tc.extra] = uuid.New()
			}
			_, err := f.repository.CreateLegacyFileAsset(f.ctx, uuid.New(), tc.actor, fields, "image.png")
			assertScopeNotFound(t, err, tc.name)
		})
	}
	fields := map[string]any{"workspace": f.workspace, "draft_issue": f.draft, "is_uploaded": true, "project": uuid.New(), "entity_type": "PROJECT_COVER"}
	assertScopeNotFound(t, f.repository.CreateLegacyIssueAttachment(f.ctx, uuid.New(), f.actor, f.workspace, f.project, uuid.New(), fields, "image.png", "", nil), "issue attachment with draft binding")
	id := uuid.New()
	if _, err := f.repository.CreateLegacyFileAsset(f.ctx, id, f.actor, fields, "image.png"); err != nil {
		t.Fatal(err)
	}
	f.assertBound(id, true)
	f.exec(`UPDATE workspace_members SET is_active=false WHERE member_id=$1`, f.actor)
	_, err := f.repository.CreateLegacyFileAsset(f.ctx, uuid.New(), f.actor, fields, "image.png")
	assertScopeNotFound(t, err, "revoked workspace")
	f.exec(`UPDATE workspace_members SET is_active=true WHERE member_id=$1`, f.actor)
	f.exec(`UPDATE draft_issues SET deleted_at=now() WHERE id=$1`, f.draft)
	_, err = f.repository.CreateLegacyFileAsset(f.ctx, uuid.New(), f.actor, fields, "image.png")
	assertScopeNotFound(t, err, "consumed draft")
	var count int
	if err := f.db.QueryRow(f.ctx, `SELECT count(*) FROM file_assets`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("unauthorized insert: %d %v", count, err)
	}
}
