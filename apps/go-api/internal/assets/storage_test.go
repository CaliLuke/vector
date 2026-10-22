package assets

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestSourceCompatibleCopySourceEscaping(t *testing.T) {
	got := encodeCopySourceKey("uploads/with space/雪!'*.png")
	want := "uploads/with%20space/%E9%9B%AA%21%27%2A.png"
	if got != want {
		t.Fatalf("copy-source escaped key = %q, want %q", got, want)
	}
}

func TestSourceCompatibleContentDispositionFilename(t *testing.T) {
	empty := ""
	if got := contentDisposition("inline", &empty); got != "inline" {
		t.Fatalf("empty filename disposition = %q, want inline", got)
	}
	filename := "folder/a b/雪!'*.png"
	if got, want := contentDisposition("attachment", &filename), "attachment; filename*=UTF-8''folder/a%20b/%E9%9B%AA%21%27%2A.png"; got != want {
		t.Fatalf("filename disposition = %q, want %q", got, want)
	}
	nilFilename := contentDisposition("inline", nil)
	if len(nilFilename) != len("inline; filename*=UTF-8''")+32 || nilFilename[:len("inline; filename*=UTF-8''")] != "inline; filename*=UTF-8''" {
		t.Fatalf("nil filename disposition = %q", nilFilename)
	}
}

func TestBotoCompatiblePostPresignerMatchesSourceFieldAndConditionContract(t *testing.T) {
	signingTime := time.Date(2026, 10, 18, 12, 34, 56, 0, time.UTC)
	conditions := []any{
		map[string]string{"bucket": "assets"},
		[]any{"content-length-range", int64(1), int64(1234)},
		map[string]string{"Content-Type": "image/png"},
		map[string]string{"key": "workspace/key.png"},
	}
	fields, err := (botoCompatiblePostPresigner{}).PresignPost(
		aws.Credentials{AccessKeyID: "access", SecretAccessKey: "secret"},
		"assets", "workspace/key.png", "us-east-1", "s3", signingTime, conditions,
		signingTime.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	if fields["x-amz-algorithm"] != "AWS4-HMAC-SHA256" || fields["X-Amz-Algorithm"] != "" || fields["key"] != "workspace/key.png" {
		t.Fatalf("POST form fields do not match boto casing: %#v", fields)
	}
	policyBytes, err := base64.StdEncoding.DecodeString(fields["policy"])
	if err != nil {
		t.Fatal(err)
	}
	var policy struct {
		Conditions []any `json:"conditions"`
	}
	if err := json.Unmarshal(policyBytes, &policy); err != nil {
		t.Fatal(err)
	}
	want := []any{
		map[string]any{"bucket": "assets"},
		[]any{"content-length-range", float64(1), float64(1234)},
		map[string]any{"Content-Type": "image/png"},
		map[string]any{"key": "workspace/key.png"},
		map[string]any{"bucket": "assets"},
		map[string]any{"key": "workspace/key.png"},
		map[string]any{"x-amz-algorithm": "AWS4-HMAC-SHA256"},
		map[string]any{"x-amz-credential": "access/20261018/us-east-1/s3/aws4_request"},
		map[string]any{"x-amz-date": "20261018T123456Z"},
	}
	if !reflect.DeepEqual(policy.Conditions, want) {
		t.Fatalf("POST policy conditions = %#v, want %#v", policy.Conditions, want)
	}
}

func TestStorageCustomEndpointPreservesEmptySourceSigningRegion(t *testing.T) {
	for _, variable := range []string{"PLANE_ASSETS_S3_REGION", "AWS_REGION"} {
		t.Setenv(variable, "")
	}
	t.Setenv("AWS_DEFAULT_REGION", "eu-west-1")
	t.Setenv("PLANE_ASSETS_S3_BUCKET", "uploads")
	t.Setenv("PLANE_ASSETS_S3_ACCESS_KEY_ID", "test")
	t.Setenv("PLANE_ASSETS_S3_SECRET_ACCESS_KEY", "test")
	t.Setenv("PLANE_ASSETS_S3_INTERNAL_ENDPOINT", "http://minio.example.test:9000")
	t.Setenv("PLANE_ASSETS_S3_PUBLIC_ENDPOINT", "http://minio.example.test:9000")
	config, err := StorageConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	storage, err := NewStorage(config)
	if err != nil {
		t.Fatal(err)
	}
	_, fields, err := storage.PresignPost(context.Background(), "probe.txt", "text/plain", 4096, "http", "localhost")
	if err != nil {
		t.Fatal(err)
	}
	assertEmptyRegion := func(credential string) {
		t.Helper()
		parts := strings.Split(credential, "/")
		if len(parts) != 5 || parts[2] != "" || parts[3] != "s3" {
			t.Fatalf("credential scope %q does not preserve source empty region", credential)
		}
	}
	assertEmptyRegion(fields["x-amz-credential"])
	filename := "probe.txt"
	signed, err := storage.PresignGet(context.Background(), "probe.txt", "attachment", &filename, "http", "localhost")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(signed)
	if err != nil {
		t.Fatal(err)
	}
	assertEmptyRegion(parsed.Query().Get("X-Amz-Credential"))
	if parsed.Host != "minio.example.test:9000" || parsed.Path != "/uploads/probe.txt" {
		t.Fatalf("unexpected custom endpoint: %s", parsed.Host+parsed.Path)
	}
}

func TestStorageAbsentRegionUsesProviderDefault(t *testing.T) {
	t.Setenv("AWS_REGION", "")
	if err := os.Unsetenv("AWS_REGION"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PLANE_ASSETS_S3_REGION", "")
	t.Setenv("PLANE_ASSETS_S3_BUCKET", "uploads")
	t.Setenv("PLANE_ASSETS_S3_INTERNAL_ENDPOINT", "http://minio.example.test:9000")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(t.TempDir(), "absent-config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "absent-credentials"))
	t.Setenv("AWS_PROFILE", "")
	for _, test := range []struct{ configured, expected string }{
		{"", "us-east-1"}, {"eu-west-1", "eu-west-1"},
	} {
		t.Run(test.expected, func(t *testing.T) {
			t.Setenv("AWS_DEFAULT_REGION", test.configured)
			config, err := StorageConfigFromEnv()
			if err != nil {
				t.Fatal(err)
			}
			if config.Region != test.expected {
				t.Fatalf("region = %q, want %q", config.Region, test.expected)
			}
		})
	}
}
