//go:build integration

package assets

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestAttachmentCollectionChecksCurrentItemVisibility(t *testing.T) {
	f, issue, asset := newPublishedAttachmentFixture(t)
	service := &LifecycleService{assets: &Service{repository: f.repository}}
	ctx := context.WithValue(f.ctx, assetUserKey{}, f.actor)

	rows, err := service.listIssueAttachments(ctx, "drafts", f.project.String(), issue.String())
	if err != nil || len(rows) != 1 || (rows[0].ID == nil || *rows[0].ID != asset.String()) {
		t.Fatalf("visible collection = %+v, %v; want the uploaded attachment", rows, err)
	}
	for _, tc := range []struct {
		name    string
		change  string
		visible bool
	}{
		{"hidden guest item", `UPDATE project_members SET role=5`, false},
		{"guest creator", `UPDATE project_members SET role=5; UPDATE issues SET created_by_id=(SELECT member_id FROM workspace_members WHERE role=15)`, true},
		{"guest view all", `UPDATE project_members SET role=5; UPDATE projects SET guest_view_all_features=true`, true},
		{"revoked project membership", `UPDATE project_members SET is_active=false`, false},
		{"revoked workspace membership", `UPDATE workspace_members SET is_active=false`, false},
		{"deleted project membership", `UPDATE project_members SET deleted_at=now()`, false},
		{"deleted workspace membership", `UPDATE workspace_members SET deleted_at=now()`, false},
		{"deleted item", `UPDATE issues SET deleted_at=now()`, false},
		{"draft item", `UPDATE issues SET is_draft=true`, false},
		{"deleted project", `UPDATE projects SET deleted_at=now()`, false},
		{"deleted workspace", `UPDATE workspaces SET deleted_at=now()`, false},
		{"item moved to another project", `UPDATE issues SET project_id='00000000-0000-4000-8000-000000000099'`, false},
		{"item moved to another workspace", `UPDATE issues SET workspace_id='00000000-0000-4000-8000-000000000099'`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := f
			f.t = t
			tx, beginErr := f.db.Begin(f.ctx)
			if beginErr != nil {
				t.Fatal(beginErr)
			}
			defer func() { _ = tx.Rollback(context.WithoutCancel(f.ctx)) }()
			f.repository = &PostgresRepository{database: tx}
			f.exec(tc.change)
			// This raw snapshot is also used for write activity. It demonstrates
			// why project/path filtering alone cannot authorize collection reads.
			raw, rawErr := f.repository.IssueAttachments(ctx, f.workspace, f.project, issue, true)
			if rawErr != nil || len(raw) != 1 {
				t.Fatalf("raw attachment snapshot = %+v, %v", raw, rawErr)
			}
			found, allowed, accessErr := f.repository.IssueAccess(ctx, f.actor, f.workspace, f.project, issue)
			if accessErr != nil || allowed != tc.visible {
				t.Fatalf("API preflight found=%v allowed=%v: %v", found, allowed, accessErr)
			}
			detail, metadata, detailErr := f.repository.IssueAttachment(ctx, f.actor, f.workspace, f.project, issue, asset)
			if tc.visible {
				if detailErr != nil || detail.ID != asset.String() || metadata["id"] != asset.String() || detail.Asset != "image.png" {
					t.Fatalf("attachment detail = %+v, %+v, %v", detail, metadata, detailErr)
				}
			} else {
				assertScopeNotFound(t, detailErr, "read hidden attachment detail")
				if detail.ID != "" || metadata != nil {
					t.Fatalf("hidden detail leaked: %+v, %+v", detail, metadata)
				}
			}
			visible, readErr := f.repository.VisibleIssueAttachments(ctx, f.actor, f.workspace, f.project, issue, true)
			if tc.visible {
				if readErr != nil || len(visible) != 1 || visible[0]["id"] != asset.String() {
					t.Fatalf("visible attachment = %+v, %v", visible, readErr)
				}
			} else {
				assertScopeNotFound(t, readErr, "read hidden attachment metadata")
				if len(visible) != 0 {
					t.Fatalf("hidden attachment metadata leaked: %+v", visible)
				}
			}
			duplicateID, duplicateFound, duplicateErr := f.repository.IssueAttachmentExternal(ctx, f.actor, f.workspace, f.project, issue, "private-id", "tracker")
			if duplicateErr != nil || duplicateFound != tc.visible || tc.visible && duplicateID != asset {
				t.Fatalf("external-ID duplicate lookup = %s, %v, %v; visible=%v", duplicateID, duplicateFound, duplicateErr, tc.visible)
			}
			assertAttachmentWritesRespectVisibility(t, f, issue, asset, tc.visible)
		})
	}
	f.exec(`DELETE FROM file_assets`)
	empty, err := f.repository.VisibleIssueAttachments(ctx, f.actor, f.workspace, f.project, issue, true)
	if err != nil || len(empty) != 0 {
		t.Fatalf("visible item without attachments = %+v, %v", empty, err)
	}
}

func newPublishedAttachmentFixture(t *testing.T) (draftAssetFixture, uuid.UUID, uuid.UUID) {
	t.Helper()
	f := newDraftAssetFixture(t)
	f.exec(`INSERT INTO projects(id,workspace_id) VALUES($1,$2)`, f.project, f.workspace)
	issue := uuid.New()
	f.exec(`INSERT INTO issues(id,workspace_id,project_id,created_by_id) VALUES($1,$2,$3,$4)`, issue, f.workspace, f.project, f.other)
	asset := f.asset("ISSUE_ATTACHMENT", f.other)
	f.exec(`UPDATE file_assets SET project_id=$2,issue_id=$3,external_id='private-id',external_source='tracker' WHERE id=$1`, asset, f.project, issue)

	return f, issue, asset
}

func assertAttachmentWritesRespectVisibility(t *testing.T, f draftAssetFixture, issue, asset uuid.UUID, visible bool) {
	t.Helper()
	for _, operation := range []struct {
		name           string
		wantEffects    int
		run            func() error
		assertAccepted func()
	}{
		{"confirm", 2, func() error {
			return f.repository.UpdateIssueAttachmentUploaded(f.ctx, f.actor, f.workspace, f.project, issue, asset, map[string]any{"type": "attachment.activity.created"}, true)
		}, func() { assertAttachmentState(t, f.db, asset.String(), true, false) }},
		{"delete", 1, func() error {
			return f.repository.DeleteIssueAttachment(f.ctx, f.actor, f.workspace, f.project, issue, asset, map[string]any{"type": "attachment.activity.deleted"})
		}, func() { assertAttachmentState(t, f.db, asset.String(), false, true) }},
		{"legacy delete", 2, func() error {
			return f.repository.DeleteLegacyIssueAttachment(f.ctx, f.actor, f.workspace, f.project, issue, asset, nil)
		}, func() {
			var deleted bool
			if err := f.db.QueryRow(f.ctx, `SELECT deleted_at IS NOT NULL AND asset='' FROM file_assets WHERE id=$1`, asset).Scan(&deleted); err != nil || !deleted {
				t.Fatalf("legacy delete: deleted=%v err=%v", deleted, err)
			}
		}},
	} {
		f.exec(`SAVEPOINT attachment_write`)
		f.exec(`UPDATE file_assets SET is_uploaded=false WHERE id=$1`, asset)
		err := operation.run()
		if visible {
			if err != nil {
				t.Fatalf("%s visible attachment: %v", operation.name, err)
			}
			operation.assertAccepted()
		} else {
			assertScopeNotFound(t, err, operation.name+" hidden attachment")
			assertAttachmentState(t, f.db, asset.String(), false, false)
		}
		var effects int
		if err := f.db.QueryRow(f.ctx, `SELECT count(*) FROM work_item_effects`).Scan(&effects); err != nil {
			t.Fatal(err)
		}
		wantEffects := 0
		if visible {
			wantEffects = operation.wantEffects
		}
		if effects != wantEffects {
			t.Fatalf("%s work-item effects=%d want %d", operation.name, effects, wantEffects)
		}
		f.exec(`ROLLBACK TO SAVEPOINT attachment_write`)
		f.exec(`RELEASE SAVEPOINT attachment_write`)
	}
	for _, legacy := range []bool{false, true} {
		id := uuid.New()
		var err error
		if legacy {
			err = f.repository.CreateLegacyIssueAttachment(f.ctx, id, f.actor, f.workspace, f.project, issue, map[string]any{}, "new.png", "/attachment", nil)
		} else {
			err = f.repository.CreateIssueAttachment(f.ctx, id, f.actor, f.workspace, f.project, issue, "new.png", []byte(`{"name":"new.png"}`), 12, "", "")
		}
		var exists bool
		if readErr := f.db.QueryRow(f.ctx, `SELECT EXISTS(SELECT 1 FROM file_assets WHERE id=$1)`, id).Scan(&exists); readErr != nil {
			t.Fatal(readErr)
		}
		if exists != visible {
			t.Fatalf("legacy=%v created=%v want %v", legacy, exists, visible)
		}
		if visible {
			if err != nil {
				t.Fatalf("create visible attachment legacy=%v: %v", legacy, err)
			}
		} else {
			assertScopeNotFound(t, err, "create hidden attachment")
		}
	}
}

func TestAttachmentConfirmationRollsBackWhenEffectAdmissionCannotBeRecorded(t *testing.T) {
	f, issue, asset := newPublishedAttachmentFixture(t)
	f.exec(`UPDATE file_assets SET is_uploaded=false WHERE id=$1`, asset)
	f.exec(`CREATE FUNCTION reject_attachment_effect() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.kind='activity' THEN RAISE EXCEPTION 'injected outbox failure'; END IF; RETURN NEW; END $$;
CREATE TRIGGER reject_attachment_effect BEFORE INSERT ON work_item_effects FOR EACH ROW EXECUTE FUNCTION reject_attachment_effect()`)
	err := f.repository.UpdateIssueAttachmentUploaded(f.ctx, f.actor, f.workspace, f.project, issue, asset, map[string]any{"type": "attachment.activity.created"}, true)
	if err == nil {
		t.Fatal("confirmation committed without durable activity admission")
	}
	assertAttachmentState(t, f.db, asset.String(), false, false)
	var effects int
	if err := f.db.QueryRow(f.ctx, `SELECT count(*) FROM work_item_effects`).Scan(&effects); err != nil || effects != 0 {
		t.Fatalf("failed confirmation effects=%d error=%v; want no committed effects", effects, err)
	}
}
