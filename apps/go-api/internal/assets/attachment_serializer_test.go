package assets

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestAttachmentSerializerVariantsPreserveSourceAssetURLAndFloat(t *testing.T) {
	row := map[string]any{
		"id": "asset-id", "entity_type": "ISSUE_ATTACHMENT", "project": "project-id",
		"issue": "issue-id", "size": json.Number("68"),
	}
	app := decorateIssueAttachmentRows([]map[string]any{row}, "demo", false)[0]
	if got, want := app["asset_url"], "/api/assets/v2/workspaces/demo/projects/project-id/issues/issue-id/attachments/asset-id/"; got != want {
		t.Fatalf("app asset_url = %#v, want %q", got, want)
	}
	if got, want := app["size"], json.Number("68.0"); got != want {
		t.Fatalf("app size = %#v, want %v", got, want)
	}

	api := decorateIssueAttachmentRows([]map[string]any{row}, "demo", true)[0]
	if _, ok := api["asset_url"]; ok {
		t.Fatalf("API serializer unexpectedly includes asset_url: %#v", api)
	}
	if got, want := api["size"], json.Number("68.0"); got != want {
		t.Fatalf("API size = %#v, want %v", got, want)
	}
}

func TestLegacyFileAssetURLUsesRequestPathAndStorageKey(t *testing.T) {
	ctx := WithRequestOrigin(context.Background(), "http", "testserver")
	user := legacyFileAssetURL(ctx, "/api/users/file-assets/name.png/", "user-object/name.png")
	if want := "http://testserver/api/users/file-assets/name.png/user-object/name.png"; user != want {
		t.Fatalf("user FileField URL = %q, want %q", user, want)
	}
	workspace := legacyFileAssetURL(ctx, "/api/workspaces/file-assets/workspace-id/name.png/", "workspace-id/name.png")
	if want := "http://testserver/api/workspaces/file-assets/workspace-id/name.png/workspace-id/name.png"; workspace != want {
		t.Fatalf("workspace FileField URL = %q, want %q", workspace, want)
	}
}

func TestUploadRequestUsesV2AndDuplicateEntityInputKeys(t *testing.T) {
	for _, test := range []struct {
		name   string
		fields map[string]any
		want   string
	}{
		{name: "V2 upload", fields: map[string]any{"entity_identifier": "issue-id"}, want: "issue-id"},
		{name: "duplicate", fields: map[string]any{"entity_id": "issue-id"}, want: "issue-id"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := uploadRequestFromFields(test.fields)
			if got := stringValue(request.EntityID); got != test.want {
				t.Fatalf("entity ID = %q, want %q", got, test.want)
			}
		})
	}
}

func TestProjectV2AssetURLUsesPathProject(t *testing.T) {
	projectID := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	got := v2ProjectAssetURL("demo", projectID, "asset-id", "ISSUE_DESCRIPTION", "issue-id")
	want := "/api/assets/v2/workspaces/demo/projects/11111111-1111-4111-8111-111111111111/asset-id/"
	if got != want {
		t.Fatalf("project asset URL = %q, want %q", got, want)
	}
}
