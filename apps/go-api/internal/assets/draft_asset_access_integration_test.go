//go:build integration

package assets

import (
	"context"
	"io"
	"strings"
	"testing"

	api "github.com/CaliLuke/go-plane/assetslifecycle/gen/plane_internal_django_migration_api"
	"github.com/google/uuid"
)

func TestDraftAssetsRemainPrivateThroughWorkspaceProjectAndLegacyAliases(t *testing.T) {
	f := newDraftAssetFixture(t)
	id := f.asset("DRAFT_ISSUE_DESCRIPTION", f.actor)
	f.exec(`UPDATE draft_issues SET project_id=$2 WHERE id=$1`, f.draft, f.project)
	f.exec(`UPDATE file_assets SET asset=$2 WHERE id=$1`, id, f.workspace.String()+"/image.png")
	if err := f.repository.BulkAssociateDraftAssets(f.ctx, f.workspace, f.actor, f.draft, []uuid.UUID{id}); err != nil {
		t.Fatal(err)
	}
	tasks := &publicTaskRecorder{}
	service := &LifecycleService{assets: &Service{repository: f.repository, tasks: tasks}}
	ctx := context.WithValue(f.ctx, assetUserKey{}, f.other)
	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{"workspace image", func() error { _, err := service.workspaceAssetRedirect(ctx, "drafts", id.String()); return err }},
		{"workspace download", func() error { _, err := service.workspaceDownload(ctx, "drafts", id.String()); return err }},
		{"project image", func() error {
			_, err := service.projectAssetRedirect(ctx, "drafts", f.project.String(), id.String())
			return err
		}},
		{"project download", func() error {
			_, err := service.projectDownload(ctx, "drafts", f.project.String(), id.String())
			return err
		}},
		{"workspace confirm", func() error { return service.patchWorkspaceAsset(ctx, "drafts", id.String(), nil) }},
		{"project confirm", func() error { return service.patchProjectAsset(ctx, "drafts", f.project.String(), id.String(), nil) }},
		{"workspace delete", func() error { return service.deleteWorkspaceAsset(ctx, "drafts", id.String()) }},
		{"project delete", func() error { return service.deleteProjectAsset(ctx, "drafts", f.project.String(), id.String()) }},
		{"restore", func() error { return service.restoreWorkspaceAsset(ctx, "drafts", id.String()) }},
		{"duplicate", func() error {
			_, err := service.duplicateWorkspaceAsset(ctx, "drafts", id.String(), io.NopCloser(strings.NewReader(`{"entity_type":"WORKSPACE_LOGO"}`)))
			return err
		}},
		{"legacy confirm", func() error { return service.confirmLegacyWorkspaceAssetUpload(ctx, "drafts", id.String(), nil) }},
		{"legacy delete", func() error {
			return service.deleteLegacyWorkspaceAssetByKey(ctx, f.workspace.String(), "image.png", true)
		}},
		{"legacy restore", func() error {
			return service.deleteLegacyWorkspaceAssetByKey(ctx, f.workspace.String(), "image.png", false)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) { assertScopeNotFound(t, tc.run(), tc.name) })
	}
	assertLegacyAssetKeyHidden(t, service, ctx, f.workspace, "image.png")
	exists, err := service.GetAPIAssetsV2WorkspacesBySlugCheckByAssetID(ctx, &api.GetAPIAssetsV2WorkspacesBySlugCheckByAssetIDPayload{Slug: "drafts", AssetID: id.String()})
	if err != nil || exists.Exists {
		t.Fatalf("private asset existence leaked: %+v %v", exists, err)
	}
	if _, lookupErr := service.GetAPIV1WorkspacesBySlugAssetsByAssetID(ctx, &api.GetAPIV1WorkspacesBySlugAssetsByAssetIDPayload{Slug: "drafts", AssetID: id.String()}); lookupErr == nil {
		t.Fatal("API v1 disclosed private draft asset")
	}
	if tasks.calls != 0 {
		t.Fatalf("denied operations queued %d effects", tasks.calls)
	}
	f.assertBound(id, true)
	f.exec(`UPDATE file_assets SET entity_type='ISSUE_DESCRIPTION' WHERE id=$1`, id)
	assertLegacyAssetKeyHidden(t, service, ctx, f.workspace, "image.png")
}

func assertLegacyAssetKeyHidden(t *testing.T, service *LifecycleService, ctx context.Context, workspace uuid.UUID, key string) {
	t.Helper()
	response, err := service.getLegacyWorkspaceAssetByKey(ctx, workspace.String(), key)
	if err != nil {
		t.Fatalf("legacy asset-key read error = %v, want filtered empty response", err)
	}
	if response == nil {
		t.Fatal("legacy asset-key read returned nil response")
	}
	if data, ok := response.AsGetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseData(); ok {
		if data.Status || len(data.Data) != 0 {
			t.Fatalf("private legacy asset data leaked: status=%t assets=%d", data.Status, len(data.Data))
		}
		return
	}
	filtered, ok := response.AsGetAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyResponseError()
	if !ok || filtered.Status || filtered.Error != "Asset key does not exist" {
		t.Fatalf("legacy asset-key response = %+v, want empty filtered response", response)
	}
}

func TestDraftOwnerCanManageImagesAfterProjectAccessIsRevoked(t *testing.T) {
	f := newDraftAssetFixture(t)
	f.exec(`UPDATE draft_issues SET project_id=$2 WHERE id=$1`, f.draft, f.project)
	f.exec(`DELETE FROM project_members WHERE member_id=$1`, f.actor)
	id := uuid.New()
	if err := f.repository.CreateWorkspaceEntityUpload(f.ctx, id, f.actor, f.workspace, uuid.Nil, "image.png", []byte(`{"name":"image.png"}`), 10, "DRAFT_ISSUE_DESCRIPTION", f.draft.String(), false); err != nil {
		t.Fatal(err)
	}
	storage, err := NewStorage(StorageConfig{AccessKeyID: "test", SecretAccessKey: "test", Bucket: "drafts-test", Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	service := &LifecycleService{assets: &Service{repository: f.repository, storage: storage, tasks: &publicTaskRecorder{}}}
	ctx := context.WithValue(f.ctx, assetUserKey{}, f.actor)
	if err := service.patchWorkspaceAsset(ctx, "drafts", id.String(), nil); err != nil {
		t.Fatal(err)
	}
	if url, err := service.workspaceAssetRedirect(ctx, "drafts", id.String()); err != nil || url == "" {
		t.Fatalf("owner image: %q %v", url, err)
	}
	if url, err := service.workspaceDownload(ctx, "drafts", id.String()); err != nil || url == "" {
		t.Fatalf("owner download: %q %v", url, err)
	}
	if err := service.deleteWorkspaceAsset(ctx, "drafts", id.String()); err != nil {
		t.Fatal(err)
	}
	if err := service.restoreWorkspaceAsset(ctx, "drafts", id.String()); err != nil {
		t.Fatal(err)
	}
	f.assertBound(id, true)
	f.exec(`UPDATE draft_issues SET deleted_at=now() WHERE id=$1`, f.draft)
	assertScopeNotFound(t, service.patchWorkspaceAsset(ctx, "drafts", id.String(), nil), "consumed draft confirmation")
	if url, err := service.workspaceAssetRedirect(ctx, "drafts", id.String()); err == nil || url != "" {
		t.Fatal("consumed draft image remained readable")
	}
}

func TestPublicAssetPathsNeverExposeOrMutatePrivateDraftAssets(t *testing.T) {
	f := newDraftAssetFixture(t)
	comment := uuid.New()
	f.exec(`INSERT INTO issue_comments VALUES($1,$2,$3,$4,NULL)`, comment, f.workspace, f.project, f.actor)
	f.exec(`UPDATE draft_issues SET project_id=$2 WHERE id=$1`, f.draft, f.project)
	id := f.asset("DRAFT_ISSUE_DESCRIPTION", f.actor)
	if err := f.repository.BulkAssociateDraftAssets(f.ctx, f.workspace, f.actor, f.draft, []uuid.UUID{id}); err != nil {
		t.Fatal(err)
	}
	// Older rows may carry an ordinary kind despite their private draft binding.
	for _, kind := range []string{"DRAFT_ISSUE_DESCRIPTION", "ISSUE_DESCRIPTION", "COMMENT_DESCRIPTION", "PROJECT_COVER", "WORKSPACE_LOGO"} {
		f.exec(`UPDATE file_assets SET entity_type=$2 WHERE id=$1`, id, kind)
		f.exec(`DELETE FROM workspace_members WHERE member_id=$1`, f.actor)
		_, err := f.repository.PublicProjectAsset(f.ctx, f.workspace, f.project, id, false, false)
		assertScopeNotFound(t, err, "public read")
		_, err = f.repository.PublicOwnedProjectAsset(f.ctx, f.actor, f.workspace, f.project, id, false)
		assertScopeNotFound(t, err, "public owner read")
		service := &LifecycleService{assets: &Service{repository: f.repository}}
		_, err = service.GetAPIAssetsV2StaticByAssetID(f.ctx, &api.GetAPIAssetsV2StaticByAssetIDPayload{AssetID: id.String()})
		if err == nil {
			t.Fatal("private asset exposed through static URL")
		}
		_, _, err = f.repository.PublicBulkAssetRows(f.ctx, f.actor, f.workspace, f.project, []uuid.UUID{id})
		assertScopeNotFound(t, err, "public bulk lookup")
		assertScopeNotFound(t, f.repository.BulkAssociatePublicAssetComments(f.ctx, f.actor, f.workspace, f.project, comment, []uuid.UUID{id}), "public bulk association")
		assertScopeNotFound(t, f.repository.UpdatePublicAsset(f.ctx, f.actor, f.workspace, f.project, id, []byte(`{}`)), "public confirm")
		assertScopeNotFound(t, f.repository.DeletePublicAsset(f.ctx, f.actor, f.workspace, f.project, id), "public delete")
		assertScopeNotFound(t, f.repository.RestorePublicAsset(f.ctx, f.actor, f.workspace, f.project, id), "public restore")
	}
	var unchanged bool
	if err := f.db.QueryRow(f.ctx, `SELECT deleted_at IS NULL AND NOT is_deleted AND draft_issue_id=$2 FROM file_assets WHERE id=$1`, id, f.draft).Scan(&unchanged); err != nil || !unchanged {
		t.Fatalf("public mutation changed private asset: %v %v", unchanged, err)
	}
}
