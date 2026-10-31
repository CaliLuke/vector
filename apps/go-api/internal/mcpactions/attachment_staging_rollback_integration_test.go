//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPAttachmentStagingAndDeletionRollBackWithStorage(t *testing.T) {
	h := attachmentRecoveryFixture(t)
	config, err := assets.StorageConfigFromEnv()
	require.NoError(t, err)
	client := s3.NewFromConfig(aws.Config{Region: config.Region, Credentials: credentials.NewStaticCredentialsProvider(config.AccessKeyID, config.SecretAccessKey, ""), HTTPClient: &http.Client{Timeout: 10 * time.Second}}, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(config.Endpoint)
		options.UsePathStyle = config.PathStyle
		options.RetryMaxAttempts = 1
	})
	runCatalogueFixtureMatrix(t, h.milestoneFixture, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		for _, scenario := range []struct {
			action string
			tables []string
		}{
			{"upload_from_url", []string{"file_assets", "file_asset_upload_intents", "work_item_effects", "mutation_receipts"}},
			{"delete", []string{"file_assets", "work_item_effects", "mutation_receipts"}},
		} {
			for _, table := range scenario.tables {
				t.Run(scenario.action+"/"+table, func(t *testing.T) {
					call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": f.item(t)})
					input := map[string]any{"action": scenario.action, "request_id": uuid.NewString()}
					if scenario.action == "delete" {
						created := milestoneObject(t, call("workitem_attachment", map[string]any{"action": "upload_from_url", "url": "https://files.example.test/staged.txt"}))
						input["attachment_id"] = created["id"]
					} else {
						input["url"] = "https://files.example.test/staged.txt"
					}
					snapshot := func() []byte { return attachmentSnapshotWithObjects(t, h, client, config.Bucket) }
					before := h.fetched.Load()
					assertCatalogueStorageFailure(t, f, call, "workitem_attachment", input, table, snapshot)
					if scenario.action == "delete" {
						require.Equal(t, before, h.fetched.Load())
					} else {
						require.Equal(t, before+2, h.fetched.Load(), "failed staging and retry fetch once each; committed replay does not fetch")
					}
				})
			}
		}
	})
}

func attachmentSnapshotWithObjects(t *testing.T, h attachmentRecoveryHarness, client *s3.Client, bucket string) []byte {
	t.Helper()
	objects, err := client.ListObjectsV2(t.Context(), &s3.ListObjectsV2Input{Bucket: aws.String(bucket)})
	require.NoError(t, err)
	require.False(t, aws.ToBool(objects.IsTruncated), "the storage snapshot must include every fixture object")
	keys := make([]string, 0, len(objects.Contents))
	for _, object := range objects.Contents {
		keys = append(keys, aws.ToString(object.Key))
	}
	slices.Sort(keys)
	data, err := json.Marshal(map[string]any{"database": json.RawMessage(attachmentStorageSnapshot(t, h.milestoneFixture)), "objects": keys})
	require.NoError(t, err)
	return data
}
