package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	assetslifecyclesvr "github.com/CaliLuke/go-plane/assetslifecycle/gen/http/plane_internal_django_migration_api/server"
	assetsapi "github.com/CaliLuke/go-plane/assetslifecycle/gen/plane_internal_django_migration_api"
	loomhttp "github.com/CaliLuke/loom/http"
)

type assetRedirectFixture struct {
	Response struct {
		Status  int `json:"status"`
		Headers struct {
			ContentType string `json:"content-type"`
		} `json:"headers"`
	} `json:"response"`
}

func TestAssetsLifecycleRedirectEncodersPreserveSourceContentType(t *testing.T) {
	fixtureDir := filepath.Join(assetsTransportRepoRoot(t), "apps", "go-api", "internal", "parity", "testdata", "fixtures", "assets_lifecycle")
	encode := func(context.Context, http.ResponseWriter) loomhttp.Encoder { return nil }
	location := "https://signed.example.test/assets-lifecycle/file.png"
	cases := []struct {
		name   string
		encode func(context.Context, http.ResponseWriter, any) error
		result func(*string) any
	}{
		{"v2-static-avatar-download-anonymous.json", assetslifecyclesvr.EncodeGetAPIAssetsV2StaticByAssetIDResponse(encode), func(contentType *string) any {
			return &assetsapi.GetAPIAssetsV2StaticByAssetIDResult{Location: &location, ContentType: contentType}
		}},
		{"v2-workspace-download-project-scope-gap.json", assetslifecyclesvr.EncodeGetAPIAssetsV2WorkspacesBySlugDownloadByAssetIDResponse(encode), func(contentType *string) any {
			return &assetsapi.GetAPIAssetsV2WorkspacesBySlugDownloadByAssetIDResult{Location: &location, ContentType: contentType}
		}},
		{"v2-project-asset-download-redirect.json", assetslifecyclesvr.EncodeGetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDDownloadByAssetIDResponse(encode), func(contentType *string) any {
			return &assetsapi.GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDDownloadByAssetIDResult{Location: &location, ContentType: contentType}
		}},
		{"v2-attachment-detail-download-redirect.json", assetslifecyclesvr.EncodeGetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsByPkResponse(encode), func(contentType *string) any {
			return &assetsapi.GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsByPkResult{Location: &location, ContentType: contentType}
		}},
		{"v2-project-asset-detail-download-redirect.json", assetslifecyclesvr.EncodeGetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDByPkResponse(encode), func(contentType *string) any {
			return &assetsapi.GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDByPkResult{Location: &location, ContentType: contentType}
		}},
		{"v2-workspace-project-asset-detail-download-redirect.json", assetslifecyclesvr.EncodeGetAPIAssetsV2WorkspacesBySlugByAssetIDResponse(encode), func(contentType *string) any {
			return &assetsapi.GetAPIAssetsV2WorkspacesBySlugByAssetIDResult{Location: &location, ContentType: contentType}
		}},
		{"public-anchor-asset-get-anonymous-redirect.json", assetslifecyclesvr.EncodeGetAPIPublicAssetsV2AnchorByAnchorByPkResponse(encode), func(contentType *string) any {
			return &assetsapi.GetAPIPublicAssetsV2AnchorByAnchorByPkResult{Location: &location, ContentType: contentType}
		}},
		{"api-issue-attachment-download-redirect.json", assetslifecyclesvr.EncodeGetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsByPkResponse(encode), func(contentType *string) any {
			return &assetsapi.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsByPkResult{Location: &location, ContentType: contentType}
		}},
		{"api-work-item-attachment-download-redirect.json", assetslifecyclesvr.EncodeGetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDAttachmentsByPkResponse(encode), func(contentType *string) any {
			return &assetsapi.GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDAttachmentsByPkResult{Location: &location, ContentType: contentType}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(fixtureDir, tc.name))
			if err != nil {
				t.Fatal(err)
			}
			var fixture assetRedirectFixture
			if err := json.Unmarshal(data, &fixture); err != nil {
				t.Fatal(err)
			}
			if fixture.Response.Status != http.StatusFound || fixture.Response.Headers.ContentType == "" {
				t.Fatalf("fixture response status=%d content-type=%q; want 302 with a source content type", fixture.Response.Status, fixture.Response.Headers.ContentType)
			}
			contentType := fixture.Response.Headers.ContentType
			response := httptest.NewRecorder()
			if err := tc.encode(context.Background(), response, tc.result(&contentType)); err != nil {
				t.Fatal(err)
			}
			if response.Code != fixture.Response.Status || response.Header().Get("Content-Type") != contentType {
				t.Fatalf("encoded status=%d content-type=%q; want %d and %q", response.Code, response.Header().Get("Content-Type"), fixture.Response.Status, contentType)
			}
		})
	}
}
