package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/selvedge"
	"github.com/google/uuid"
)

const fileAssetMetadataTask = "plane.bgtasks.storage_metadata_task.get_asset_object_metadata"

// FileAssetMetadataDispatcher admits the existing durable FileAsset metadata
// workflow using an event identity supplied by the upload producer.
type FileAssetMetadataDispatcher struct {
	app      *selvedge.App
	workflow *selvedge.WorkflowFunc[assets.StorageMetadataWorkflowInput, struct{}]
}

func NewFileAssetMetadataDispatcher(app *selvedge.App, workflow *selvedge.WorkflowFunc[assets.StorageMetadataWorkflowInput, struct{}]) (*FileAssetMetadataDispatcher, error) {
	if app == nil || workflow == nil {
		return nil, errors.New("workflow app and FileAsset metadata workflow are required")
	}
	return &FileAssetMetadataDispatcher{app: app, workflow: workflow}, nil
}

// Admit starts one metadata refresh. The event ID must be stable for a single
// object upload so framework retries and duplicate producer submissions map
// to the same durable run.
func (dispatcher *FileAssetMetadataDispatcher) Admit(ctx context.Context, assetID, eventID string) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.workflow == nil {
		return "", errors.New("durable FileAsset metadata dispatcher is not configured")
	}
	parsedAssetID, err := uuid.Parse(assetID)
	if err != nil {
		return "", fmt.Errorf("parse FileAsset ID: %w", err)
	}
	if eventID == "" {
		return "", errors.New("FileAsset metadata event ID is required")
	}
	instanceID := fmt.Sprintf("plane-assets-metadata:%s:%s", parsedAssetID, eventID)
	input := assets.StorageMetadataWorkflowInput{AssetID: parsedAssetID.String(), EventID: eventID}
	startedID, err := admitOrReconcile(ctx, dispatcher.app, dispatcher.workflow, instanceID, input)
	if err != nil {
		return "", fmt.Errorf("admit FileAsset metadata workflow: %w", err)
	}
	return startedID, nil
}

// FileAssetMetadataTaskPublisher routes the existing Celery task name to the
// registered durable workflow while leaving unrelated task names to the
// configured publisher.
type FileAssetMetadataTaskPublisher struct {
	fallback   TaskFallback
	dispatcher *FileAssetMetadataDispatcher
}

func NewFileAssetMetadataTaskPublisher(fallback TaskFallback, dispatcher *FileAssetMetadataDispatcher) (*FileAssetMetadataTaskPublisher, error) {
	if fallback == nil || dispatcher == nil {
		return nil, errors.New("task fallback and FileAsset metadata dispatcher are required")
	}
	return &FileAssetMetadataTaskPublisher{fallback: fallback, dispatcher: dispatcher}, nil
}

func (publisher *FileAssetMetadataTaskPublisher) Task(ctx context.Context, name string, args []any, kwargs map[string]any) error {
	if publisher == nil || publisher.fallback == nil || publisher.dispatcher == nil {
		return errors.New("FileAsset metadata task publisher is not configured")
	}
	if name != fileAssetMetadataTask {
		return publisher.fallback.Task(ctx, name, args, kwargs)
	}
	assetID, err := fileAssetMetadataID(args, kwargs)
	if err != nil {
		return err
	}
	if _, err := publisher.dispatcher.Admit(ctx, assetID, uuid.NewString()); err != nil {
		return fmt.Errorf("admit FileAsset storage metadata refresh: %w", err)
	}
	return nil
}

func (publisher *FileAssetMetadataTaskPublisher) SoftDelete(ctx context.Context, table, id, field string) error {
	if publisher == nil || publisher.fallback == nil {
		return errors.New("FileAsset metadata task publisher is not configured")
	}
	return publisher.fallback.SoftDelete(ctx, table, id, field)
}

func fileAssetMetadataID(args []any, kwargs map[string]any) (string, error) {
	if len(args) == 1 && len(kwargs) == 0 {
		raw, err := json.Marshal(args[0])
		if err != nil {
			return "", fmt.Errorf("encode FileAsset metadata task ID: %w", err)
		}
		var id string
		if err := json.Unmarshal(raw, &id); err != nil {
			return "", errors.New("FileAsset metadata task ID must be a string")
		}
		return id, nil
	}
	if len(args) != 0 {
		return "", errors.New("FileAsset metadata task accepts one positional ID or asset_id keyword")
	}
	if len(kwargs) != 1 {
		return "", errors.New("FileAsset metadata task requires only asset_id")
	}
	raw, ok := kwargs["asset_id"]
	if !ok {
		return "", errors.New("FileAsset metadata task requires asset_id")
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return "", fmt.Errorf("encode FileAsset metadata task ID: %w", err)
	}
	var id string
	if err := json.Unmarshal(encoded, &id); err != nil {
		return "", errors.New("FileAsset metadata task ID must be a string")
	}
	return id, nil
}
