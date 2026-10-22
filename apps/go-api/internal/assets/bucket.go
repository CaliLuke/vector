package assets

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// EnsureBucket prepares object storage for a fresh installation. Permission and
// service failures are fatal: only a missing bucket permits a create attempt.
func (storage *Storage) EnsureBucket(ctx context.Context) error {
	input := &s3.HeadBucketInput{Bucket: aws.String(storage.bucket)}
	if _, err := storage.client.HeadBucket(ctx, input); err == nil {
		return nil
	} else {
		var response *smithyhttp.ResponseError
		var apiError smithy.APIError
		missing := errors.As(err, &response) && response.HTTPStatusCode() == http.StatusNotFound
		missing = missing || errors.As(err, &apiError) && (apiError.ErrorCode() == "NotFound" || apiError.ErrorCode() == "NoSuchBucket")
		if !missing {
			return fmt.Errorf("check application bucket: %w", err)
		}
	}
	create := &s3.CreateBucketInput{Bucket: aws.String(storage.bucket)}
	// AWS requires the location constraint outside us-east-1. Custom S3
	// endpoints retain Plane's existing create-without-location contract.
	if storage.config.Endpoint == "" && storage.config.Region != "us-east-1" {
		create.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(storage.config.Region),
		}
	}
	if _, err := storage.client.CreateBucket(ctx, create); err != nil {
		var owned *types.BucketAlreadyOwnedByYou
		if !errors.As(err, &owned) {
			return fmt.Errorf("create application bucket: %w", err)
		}
	}
	if _, err := storage.client.HeadBucket(ctx, input); err != nil {
		return fmt.Errorf("verify application bucket: %w", err)
	}
	return nil
}
