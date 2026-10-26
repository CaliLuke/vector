package assets

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
)

const (
	StorageObjectDeleteWorkflowName = "plane.assets.delete_storage_object"
	storageObjectDeleteActivityName = "plane.assets.delete_storage_object_key"
)

// StorageObjectDeleteInput is an immutable object identity captured by the
// database transaction that removed the asset. A later row update cannot
// redirect a retry to a replacement object's key.
type StorageObjectDeleteInput struct {
	AssetID   string `json:"asset_id"`
	ObjectKey string `json:"object_key"`
	EventID   string `json:"event_id"`
}

type storageObjectDeleteGuard interface {
	ObjectKeyInUse(context.Context, uuid.UUID, string) (bool, error)
}

type storageObjectDeleter interface {
	Delete(context.Context, string) error
}

func deleteStorageObjectIfUnused(ctx context.Context, input StorageObjectDeleteInput, guard storageObjectDeleteGuard, storage storageObjectDeleter) error {
	assetID, err := uuid.Parse(input.AssetID)
	if err != nil {
		return fmt.Errorf("parse FileAsset ID for object deletion: %w", err)
	}
	if _, parseErr := uuid.Parse(input.EventID); parseErr != nil {
		return fmt.Errorf("parse FileAsset deletion event ID: %w", parseErr)
	}
	if input.ObjectKey == "" {
		return errors.New("FileAsset object deletion requires an object key")
	}
	inUse, err := guard.ObjectKeyInUse(ctx, assetID, input.ObjectKey)
	if err != nil {
		return fmt.Errorf("check FileAsset object key before deletion: %w", err)
	}
	if inUse {
		return nil
	}
	if err := storage.Delete(ctx, input.ObjectKey); err != nil {
		return fmt.Errorf("delete FileAsset object %q: %w", input.ObjectKey, err)
	}
	return nil
}

// RegisterStorageObjectDeleteWorkflow registers retryable deletion against
// the exact captured key. DeleteObject is idempotent for an already-absent key.
func RegisterStorageObjectDeleteWorkflow(app *selvedge.App, guard storageObjectDeleteGuard, storage storageObjectDeleter, policy *retry.Policy) (*selvedge.WorkflowFunc[StorageObjectDeleteInput, struct{}], error) {
	if app == nil || guard == nil || storage == nil {
		return nil, errors.New("FileAsset deletion app, guard, and storage are required")
	}
	if policy == nil {
		policy = retry.Exponential(8, time.Second, time.Minute, 2)
	}
	policyCopy := *policy
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	activity := execution.Activity[StorageObjectDeleteInput, struct{}]{Name: storageObjectDeleteActivityName}
	if err := executionSelvedge.Register(activities, activity, func(ctx context.Context, input StorageObjectDeleteInput) (struct{}, error) {
		return struct{}{}, deleteStorageObjectIfUnused(ctx, input, guard, storage)
	}, selvedge.ActivityOptions{StartToCloseTimeout: 30 * time.Second, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register FileAsset object deletion activity: %w", err)
	}
	workflow := selvedge.DefineWorkflow(StorageObjectDeleteWorkflowName, func(ctx *selvedge.WorkflowContext, input StorageObjectDeleteInput) (struct{}, error) {
		if _, err := uuid.Parse(input.EventID); err != nil {
			return struct{}{}, fmt.Errorf("parse FileAsset deletion event ID: %w", err)
		}
		return activity.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
	})
	selvedge.RegisterWorkflow(app, workflow)
	return workflow, nil
}
