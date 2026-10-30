package assets

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
)

type numericAssetRepository struct {
	assetRepository
	writes int
}

func (repository *numericAssetRepository) CreateLegacyFileAsset(_ context.Context, _, _ uuid.UUID, _ map[string]any, _ string) (map[string]any, error) {
	repository.writes++
	return map[string]any{}, nil
}

func TestLegacyUploadValidatesSizeBeforeObjectOrDatabaseWrites(t *testing.T) {
	for _, test := range []struct {
		size     string
		accepted bool
	}{
		{"NaN", false}, {"Infinity", false}, {"-Infinity", false}, {"1e309", false}, {"37.5", true},
	} {
		t.Run(test.size, func(t *testing.T) {
			var requests atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			storage, err := NewStorage(StorageConfig{Bucket: "numeric-validation", Region: "us-east-1", Endpoint: server.URL, PathStyle: true, AccessKeyID: "test", SecretAccessKey: "test"})
			if err != nil {
				t.Fatal(err)
			}
			repository := &numericAssetRepository{}
			service := &LifecycleService{assets: &Service{repository: repository, storage: storage, fileLimit: 1024}}
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			if fieldErr := writer.WriteField("size", test.size); fieldErr != nil {
				t.Fatal(fieldErr)
			}
			part, err := writer.CreateFormFile("asset", "example.txt")
			if err != nil {
				t.Fatal(err)
			}
			if _, writeErr := part.Write([]byte("data")); writeErr != nil {
				t.Fatal(writeErr)
			}
			if closeErr := writer.Close(); closeErr != nil {
				t.Fatal(closeErr)
			}
			ctx := context.WithValue(t.Context(), assetUserKey{}, uuid.New())
			ctx = WithRequestContentType(ctx, writer.FormDataContentType())
			_, err = service.createLegacyUserAsset(ctx, io.NopCloser(bytes.NewReader(body.Bytes())))
			if test.accepted {
				if err != nil {
					t.Fatal(err)
				}
				if requests.Load() != 1 || repository.writes != 1 {
					t.Fatalf("finite upload object writes=%d row writes=%d, want one of each", requests.Load(), repository.writes)
				}
				return
			}
			var failure *djangoapi.HTTPError
			if !errors.As(err, &failure) || failure.Status != http.StatusBadRequest {
				t.Errorf("error=%v, want HTTP 400", err)
			}
			if requests.Load() != 0 || repository.writes != 0 {
				t.Errorf("invalid size caused object writes=%d row writes=%d", requests.Load(), repository.writes)
			}
		})
	}
}
