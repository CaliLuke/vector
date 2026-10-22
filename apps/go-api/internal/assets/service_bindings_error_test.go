package assets

import (
	"encoding/json"
	"errors"
	"testing"

	api "github.com/CaliLuke/go-plane/assetslifecycle/gen/plane_internal_django_migration_api"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
)

func TestMapAssetsLifecycleErrorUsesOperationSpecificStatus(t *testing.T) {
	err := mapAssetsLifecycleError("PostAPIAssetsV2WorkspacesBySlug", &djangoapi.HTTPError{
		Status: 405,
		Body:   map[string]any{"detail": `Method "PATCH" not allowed.`},
	})
	var named interface{ LoomErrorName() string }
	if !errors.As(err, &named) {
		t.Fatalf("mapped error %T does not implement LoomErrorName", err)
	}
	if got, want := named.LoomErrorName(), "PostAPIAssetsV2WorkspacesBySlugStatus405"; got != want {
		t.Fatalf("LoomErrorName() = %q, want %q", got, want)
	}
	var response api.PostAPIAssetsV2WorkspacesBySlugStatus405
	if !errors.As(err, &response) {
		t.Fatalf("mapped error %T does not match operation response type", err)
	}
	var got string
	if err := json.Unmarshal(response["detail"], &got); err != nil {
		t.Fatalf("decode mapped detail: %v", err)
	}
	if got != `Method "PATCH" not allowed.` {
		t.Fatalf("mapped detail = %q", got)
	}
}

func TestMapAssetsLifecycleErrorFallsBackToOperationGeneric500(t *testing.T) {
	err := mapAssetsLifecycleError("GetAPIAssetsV2WorkspacesBySlug", errors.New("unexpected internal error"))
	var named interface{ LoomErrorName() string }
	if !errors.As(err, &named) {
		t.Fatalf("mapped error %T does not implement LoomErrorName", err)
	}
	if got, want := named.LoomErrorName(), "GetAPIAssetsV2WorkspacesBySlugStatus500"; got != want {
		t.Fatalf("LoomErrorName() = %q, want %q", got, want)
	}
}

func TestMapAssetsLifecycleErrorUsesValueReceiverResponse(t *testing.T) {
	err := mapAssetsLifecycleError("GetAPIV1WorkspacesBySlugAssetsByAssetID", &djangoapi.HTTPError{
		Status: 400,
		Body:   map[string]any{"error": "bad request"},
	})
	var response api.GetAPIV1WorkspacesBySlugAssetsByAssetIDStatus400
	if !errors.As(err, &response) {
		t.Fatalf("mapped error %T does not match value-receiver response type", err)
	}
	var named interface{ LoomErrorName() string }
	if !errors.As(err, &named) || named.LoomErrorName() != "GetAPIV1WorkspacesBySlugAssetsByAssetIDStatus400" {
		t.Fatalf("mapped error name = %v, want operation status 400", named)
	}
}
