package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	assetsapi "github.com/CaliLuke/go-plane/assetslifecycle/gen/assets_lifecycle"
	assetslifecyclesvr "github.com/CaliLuke/go-plane/assetslifecycle/gen/http/assets_lifecycle/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestAssetsLifecycleRedirectEncodersPreserveContentType(t *testing.T) {
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
			contentType := "text/html; charset=utf-8"
			response := httptest.NewRecorder()
			if err := tc.encode(context.Background(), response, tc.result(&contentType)); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusFound || response.Header().Get("Content-Type") != contentType {
				t.Fatalf("encoded status=%d content-type=%q; want %d and %q", response.Code, response.Header().Get("Content-Type"), http.StatusFound, contentType)
			}
		})
	}
}
