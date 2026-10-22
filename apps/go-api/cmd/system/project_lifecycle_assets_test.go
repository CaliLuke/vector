package main

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
)

type projectLifecycleAssetReaderFake struct {
	asset assets.FileAsset
	err   error
	scope []uuid.UUID
}

func (f *projectLifecycleAssetReaderFake) AssetByProjectAndID(_ context.Context, workspace, project, id uuid.UUID) (assets.FileAsset, error) {
	f.scope = []uuid.UUID{workspace, project, id}
	return f.asset, f.err
}

func TestProjectCoverAssignmentUsesAssetOwnerScopeAndRejectsDeletedOrWrongEntity(t *testing.T) {
	workspace, project, id := uuid.New(), uuid.New(), uuid.New()
	reader := &projectLifecycleAssetReaderFake{asset: assets.FileAsset{EntityType: "PROJECT_COVER"}}
	validate := projectLifecycleCoverValidator(reader)
	allowed, err := validate(context.Background(), workspace, project, id)
	if err != nil || !allowed || !slices.Equal(reader.scope, []uuid.UUID{workspace, project, id}) {
		t.Fatalf("cover scope=%v allowed=%v err=%v", reader.scope, allowed, err)
	}
	for _, asset := range []assets.FileAsset{{EntityType: "USER_COVER"}, {EntityType: "PROJECT_COVER", IsDeleted: true}} {
		reader.asset = asset
		if allowed, err = validate(context.Background(), workspace, project, id); err != nil || allowed {
			t.Fatalf("accepted unavailable cover %#v: %v", asset, err)
		}
	}
	reader.err = &djangoapi.HTTPError{Status: 404}
	if allowed, err = validate(context.Background(), workspace, project, id); err != nil || allowed {
		t.Fatalf("foreign asset accepted: %v", err)
	}
	reader.err = errors.New("database unavailable")
	if _, err = validate(context.Background(), workspace, project, id); err == nil {
		t.Fatal("database error became a validation rejection")
	}
}
