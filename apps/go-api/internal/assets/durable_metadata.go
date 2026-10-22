package assets

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	awsretry "github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	StorageMetadataWorkflowName    = "plane.assets.refresh_storage_metadata"
	StorageMetadataActivityName    = "plane.assets.read_storage_metadata"
	storageMetadataActivityTimeout = 30 * time.Second
)

var ErrPermanentStorageMetadata = errors.New("permanent storage metadata failure")

// StorageMetadataWorkflowInput identifies an asset and its stable upload/event
// generation. Reusing EventID is idempotent at the workflow admission boundary.
type StorageMetadataWorkflowInput struct {
	AssetID string `json:"asset_id"`
	EventID string `json:"event_id"`
}

type storageMetadataStore interface {
	AssetKey(context.Context, uuid.UUID) (string, bool, error)
	SaveMetadataForKey(context.Context, uuid.UUID, string, any) error
}

type objectMetadataReader interface {
	Head(context.Context, string) (*s3.HeadObjectOutput, error)
}

// RegisterStorageMetadataWorkflow registers the typed metadata activity and native
// workflow before App.Start. Selvedge owns retries; callers can supply a native
// policy for the isolated prototype harness.
func RegisterStorageMetadataWorkflow(app *selvedge.App, repository storageMetadataStore, storage objectMetadataReader, policy *retry.Policy) (*selvedge.WorkflowFunc[StorageMetadataWorkflowInput, struct{}], error) {
	return registerStorageMetadataWorkflow(app, repository, storage, policy, storageMetadataActivityTimeout)
}

func registerStorageMetadataWorkflow(app *selvedge.App, repository storageMetadataStore, storage objectMetadataReader, policy *retry.Policy, activityTimeout time.Duration) (*selvedge.WorkflowFunc[StorageMetadataWorkflowInput, struct{}], error) {
	if repository == nil || storage == nil {
		return nil, errors.New("FileAsset metadata repository and storage are required")
	}
	if activityTimeout <= 0 {
		return nil, errors.New("FileAsset metadata activity timeout must be positive")
	}
	if policy == nil {
		policy = retry.Exponential(8, time.Second, time.Minute, 2)
	}
	policyCopy := *policy
	policyCopy.NonRetryableErrors = append(append([]error(nil), policy.NonRetryableErrors...), ErrPermanentStorageMetadata)
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	activity := execution.Activity[StorageMetadataWorkflowInput, struct{}]{Name: StorageMetadataActivityName}
	err = executionSelvedge.Register(activities, activity, func(ctx context.Context, input StorageMetadataWorkflowInput) (struct{}, error) {
		assetID, err := uuid.Parse(input.AssetID)
		if err != nil {
			return struct{}{}, fmt.Errorf("%w: parse FileAsset ID: %v", ErrPermanentStorageMetadata, err)
		}
		key, found, err := repository.AssetKey(ctx, assetID)
		if err != nil {
			return struct{}{}, classifyStorageMetadataError(err)
		}
		if !found {
			return struct{}{}, nil
		}
		head, err := storage.Head(ctx, key)
		if isObjectNotFound(err) {
			// Django's S3 helper converts ClientError into None and saves it.
			if saveErr := repository.SaveMetadataForKey(ctx, assetID, key, nil); saveErr != nil {
				return struct{}{}, classifyStorageMetadataError(saveErr)
			}
			return struct{}{}, nil
		}
		if err != nil {
			return struct{}{}, classifyStorageMetadataError(fmt.Errorf("head FileAsset object %q: %w", key, err))
		}
		if err := repository.SaveMetadataForKey(ctx, assetID, key, metadataFromHead(head)); err != nil {
			return struct{}{}, classifyStorageMetadataError(err)
		}
		return struct{}{}, nil
	}, selvedge.ActivityOptions{StartToCloseTimeout: activityTimeout, RetryPolicy: &policyCopy})
	if err != nil {
		return nil, fmt.Errorf("register FileAsset metadata activity: %w", err)
	}
	workflow := selvedge.DefineWorkflow(StorageMetadataWorkflowName, func(ctx *selvedge.WorkflowContext, input StorageMetadataWorkflowInput) (struct{}, error) {
		return activity.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
	})
	selvedge.RegisterWorkflow(app, workflow)
	return workflow, nil
}

func metadataFromHead(head *s3.HeadObjectOutput) map[string]any {
	var contentType, contentLength, lastModified, etag any
	if head.ContentType != nil {
		contentType = *head.ContentType
	}
	if head.ContentLength != nil {
		contentLength = *head.ContentLength
	}
	if head.LastModified != nil {
		// Python datetime.isoformat() keeps the numeric UTC offset (+00:00).
		modified := head.LastModified.Truncate(time.Microsecond)
		layout := "2006-01-02T15:04:05-07:00"
		if modified.Nanosecond() != 0 {
			layout = "2006-01-02T15:04:05.000000-07:00"
		}
		lastModified = modified.Format(layout)
	}
	if head.ETag != nil {
		etag = *head.ETag
	}
	metadata := head.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}
	return map[string]any{
		"ContentType":   contentType,
		"ContentLength": contentLength,
		"LastModified":  lastModified,
		"ETag":          etag,
		"Metadata":      metadata,
	}
}

func isObjectNotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		switch apiError.ErrorCode() {
		case "NoSuchKey", "NotFound", "NoSuchObject":
			return true
		}
	}
	var responseError *smithyhttp.ResponseError
	return errors.As(err, &responseError) && responseError.HTTPStatusCode() == http.StatusNotFound
}

func classifyStorageMetadataError(err error) error {
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) {
		// PostgreSQL connection, transaction rollback, resource exhaustion, and
		// shutdown classes can recover. Other server-reported SQL failures need
		// operator or data correction and should not be retried by the worker.
		if len(databaseError.Code) < 2 {
			return fmt.Errorf("%w: %v", ErrPermanentStorageMetadata, err)
		}
		switch databaseError.Code[:2] {
		case "08", "40", "53", "57":
		default:
			return fmt.Errorf("%w: %w", ErrPermanentStorageMetadata, err)
		}
	}
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		if awsretry.NewStandard().IsErrorRetryable(err) || apiError.ErrorFault() == smithy.FaultServer {
			return err
		}
		return fmt.Errorf("%w: %w", ErrPermanentStorageMetadata, err)
	}
	return err
}

var _ storageMetadataStore = (*PostgresRepository)(nil)
var _ objectMetadataReader = (*Storage)(nil)
