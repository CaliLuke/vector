package backgroundtasks

import (
	"context"
	"errors"
	"fmt"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/selvedge"
	"github.com/google/uuid"
)

// FileAssetStorageDeleteDispatcher admits one immutable object-key deletion
// captured with its FileAsset tombstone transaction.
type FileAssetStorageDeleteDispatcher struct {
	app      *selvedge.App
	workflow *selvedge.WorkflowFunc[assets.StorageObjectDeleteInput, struct{}]
}

func NewFileAssetStorageDeleteDispatcher(app *selvedge.App, workflow *selvedge.WorkflowFunc[assets.StorageObjectDeleteInput, struct{}]) (*FileAssetStorageDeleteDispatcher, error) {
	if app == nil || workflow == nil {
		return nil, errors.New("workflow app and FileAsset storage deletion workflow are required")
	}
	return &FileAssetStorageDeleteDispatcher{app: app, workflow: workflow}, nil
}

func (dispatcher *FileAssetStorageDeleteDispatcher) Admit(ctx context.Context, assetID, objectKey, eventID string) error {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.workflow == nil {
		return errors.New("durable FileAsset storage deletion dispatcher is not configured")
	}
	parsedAssetID, err := uuid.Parse(assetID)
	if err != nil {
		return fmt.Errorf("parse FileAsset ID for deletion: %w", err)
	}
	if _, err := uuid.Parse(eventID); err != nil {
		return fmt.Errorf("parse FileAsset deletion event ID: %w", err)
	}
	if objectKey == "" {
		return errors.New("FileAsset deletion object key is required")
	}
	input := assets.StorageObjectDeleteInput{AssetID: parsedAssetID.String(), ObjectKey: objectKey, EventID: eventID}
	instanceID := fmt.Sprintf("plane-assets-delete:%s:%s", parsedAssetID, eventID)
	if _, err := admitOrReconcile(ctx, dispatcher.app, dispatcher.workflow, instanceID, input); err != nil {
		return fmt.Errorf("admit FileAsset storage deletion workflow: %w", err)
	}
	return nil
}
