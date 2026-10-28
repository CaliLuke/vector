package assets

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

// This test is an opt-in real-service proof against a disposable S3-compatible
// test bucket. It covers RustFS POST-policy enforcement and object operations.
func TestStorageRustFSPresignedPostGetCopyHeadDelete(t *testing.T) {
	endpoint := os.Getenv("PLANE_ASSETS_STORAGE_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("set PLANE_ASSETS_STORAGE_TEST_ENDPOINT to run against isolated RustFS")
	}
	config := StorageConfig{
		AccessKeyID:     os.Getenv("PLANE_ASSETS_STORAGE_TEST_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("PLANE_ASSETS_STORAGE_TEST_SECRET_ACCESS_KEY"),
		Bucket:          os.Getenv("PLANE_ASSETS_STORAGE_TEST_BUCKET"),
		Region:          "us-east-1",
		Endpoint:        endpoint,
		PublicEndpoint:  endpoint,
		PathStyle:       true,
		SignedURLTTL:    5 * time.Minute,
	}
	if config.Bucket == "" {
		config.Bucket = "plane-rustfs-sdk-spike"
	}
	if config.AccessKeyID == "" || config.SecretAccessKey == "" {
		t.Fatal("RustFS fixture access key and secret are required")
	}
	for _, region := range []string{"us-east-1", ""} {
		t.Run("region="+region, func(t *testing.T) {
			config.Region = region
			checkStorageRustFS(t, config)
		})
	}
}

func checkStorageRustFS(t *testing.T, config StorageConfig) {
	t.Helper()
	storage, err := NewStorage(config)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_, err = storage.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(storage.bucket)})
	if err != nil && !strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") && !strings.Contains(err.Error(), "BucketAlreadyExists") {
		t.Fatalf("create own test bucket: %v", err)
	}

	key := "storage-test/" + randomHex32() + ".bin"
	copyKey := key + ".copy"
	badSignatureKey := key + ".bad-signature"
	keyEscape := key + ".key-escape"
	escapedKey := key + ".escaped"
	wrongContentTypeKey := key + ".wrong-content-type"
	oversizeKey := key + ".oversize"
	expiredKey := key + ".expired"
	t.Cleanup(func() {
		for _, cleanupKey := range []string{key, copyKey, badSignatureKey, keyEscape, escapedKey, wrongContentTypeKey, oversizeKey, expiredKey} {
			if deleteErr := storage.Delete(context.Background(), cleanupKey); deleteErr != nil {
				t.Errorf("clean test object: %v", deleteErr)
			}
		}
	})
	content := []byte{0x00, 0x01, 0xff, '\n'}
	// S3 evaluates content-length-range over the full multipart form payload,
	// which includes form fields and boundaries as well as the file bytes.
	postURL, values, err := storage.PresignPost(ctx, key, "application/octet-stream", 4096, "http", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	status, responseBody, responseOrigin := postStorageForm(t, ctx, postURL, values, content, "http://localhost:8000")
	if status != http.StatusNoContent && status != http.StatusCreated {
		t.Fatalf("presigned POST status=%d body=%s", status, responseBody)
	}
	if responseOrigin != "http://localhost:8000" {
		t.Fatalf("presigned POST allow-origin=%q, want %q", responseOrigin, "http://localhost:8000")
	}

	metadata, err := storage.Head(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if aws.ToInt64(metadata.ContentLength) != int64(len(content)) {
		t.Fatalf("HeadObject content length=%d", aws.ToInt64(metadata.ContentLength))
	}
	filename := "fixture.bin"
	getURL, err := storage.PresignGet(ctx, key, "attachment", &filename, "http", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(getURL)
	if err != nil || parsed.Host == "" || parsed.Query().Get("X-Amz-Signature") == "" {
		t.Fatalf("GET URL is not signed: %q (%v)", getURL, err)
	}
	getResponse, err := http.Get(getURL)
	if err != nil {
		t.Fatal(err)
	}
	got, readErr := io.ReadAll(getResponse.Body)
	_ = getResponse.Body.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if getResponse.StatusCode != http.StatusOK || !bytes.Equal(got, content) {
		t.Fatalf("signed GET status=%d body=%v", getResponse.StatusCode, got)
	}

	checkStoragePostCORS(t, ctx, postURL, "http://localhost:8000")
	checkStoragePostRejections(t, ctx, storage, config, content, badSignatureKey, keyEscape, escapedKey, wrongContentTypeKey, oversizeKey, expiredKey)

	if err := storage.Copy(ctx, key, copyKey); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.Head(ctx, copyKey); err != nil {
		t.Fatalf("HeadObject copied key: %v", err)
	}
	if err := storage.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if err := storage.Delete(ctx, copyKey); err != nil {
		t.Fatal(err)
	}
}

func checkStoragePostCORS(t *testing.T, ctx context.Context, postURL, origin string) {
	t.Helper()
	request, err := http.NewRequestWithContext(ctx, http.MethodOptions, postURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", origin)
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "content-type")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("POST CORS preflight status=%d body=%s", response.StatusCode, body)
	}
	if response.Header.Get("Access-Control-Allow-Origin") != origin {
		t.Fatalf("POST CORS allow-origin=%q, want %q", response.Header.Get("Access-Control-Allow-Origin"), origin)
	}
	if !strings.Contains(strings.ToUpper(response.Header.Get("Access-Control-Allow-Methods")), http.MethodPost) {
		t.Fatalf("POST CORS allow-methods=%q does not allow POST", response.Header.Get("Access-Control-Allow-Methods"))
	}
	allowedHeaders := strings.ToLower(response.Header.Get("Access-Control-Allow-Headers"))
	if !strings.Contains(allowedHeaders, "content-type") && !strings.Contains(allowedHeaders, "*") {
		t.Fatalf("POST CORS allow-headers=%q does not allow Content-Type", response.Header.Get("Access-Control-Allow-Headers"))
	}
}

func checkStoragePostRejections(t *testing.T, ctx context.Context, storage *Storage, config StorageConfig, content []byte, badSignatureKey, keyEscape, escapedKey, wrongContentTypeKey, oversizeKey, expiredKey string) {
	t.Helper()
	type invalidPost struct {
		name      string
		objectKey string
		otherKey  string
		mutate    func(map[string]string)
		body      []byte
		ttl       time.Duration
	}
	invalid := []invalidPost{
		{name: "tampered signature", objectKey: badSignatureKey, mutate: func(fields map[string]string) { fields["x-amz-signature"] = strings.Repeat("0", 64) }, body: content},
		{name: "key escape", objectKey: keyEscape, otherKey: escapedKey, mutate: func(fields map[string]string) { fields["key"] = escapedKey }, body: content},
		{name: "wrong content type", objectKey: wrongContentTypeKey, mutate: func(fields map[string]string) { fields["Content-Type"] = "text/plain" }, body: content},
		{name: "oversize", objectKey: oversizeKey, body: bytes.Repeat([]byte("x"), 8192)},
		{name: "expired policy", objectKey: expiredKey, body: content, ttl: time.Second},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			postStorage := storage
			if test.ttl > 0 {
				shortConfig := config
				shortConfig.SignedURLTTL = test.ttl
				var err error
				postStorage, err = NewStorage(shortConfig)
				if err != nil {
					t.Fatal(err)
				}
			}
			postURL, fields, err := postStorage.PresignPost(ctx, test.objectKey, "application/octet-stream", 4096, "http", "127.0.0.1")
			if err != nil {
				t.Fatal(err)
			}
			if test.mutate != nil {
				test.mutate(fields)
			}
			if test.ttl > 0 {
				time.Sleep(test.ttl + 250*time.Millisecond)
			}
			status, body, _ := postStorageForm(t, ctx, postURL, fields, test.body, "http://localhost:8000")
			if status < http.StatusBadRequest || status >= http.StatusInternalServerError {
				t.Fatalf("invalid presigned POST status=%d body=%s", status, body)
			}
			for _, objectKey := range []string{test.objectKey, test.otherKey} {
				if objectKey == "" {
					continue
				}
				if _, err := storage.Head(ctx, objectKey); !isS3ObjectNotFound(err) {
					t.Fatalf("invalid presigned POST object %q is not absent: %v", objectKey, err)
				}
			}
		})
	}
}

func postStorageForm(t *testing.T, ctx context.Context, postURL string, values map[string]string, content []byte, origin string) (int, string, string) {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for field, value := range values {
		if err := form.WriteField(field, value); err != nil {
			t.Fatal(err)
		}
	}
	file, err := form.CreateFormFile("file", "fixture.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, writeErr := file.Write(content); writeErr != nil {
		t.Fatal(writeErr)
	}
	if closeErr := form.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, postURL, &body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", form.FormDataContentType())
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(responseBody), response.Header.Get("Access-Control-Allow-Origin")
}

func isS3ObjectNotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiError smithy.APIError
	if !errors.As(err, &apiError) {
		return false
	}
	switch apiError.ErrorCode() {
	case "NotFound", "NoSuchKey", "NoSuchObject", "404":
		return true
	default:
		return false
	}
}
