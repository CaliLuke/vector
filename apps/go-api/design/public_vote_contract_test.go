package design

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/pb33f/libopenapi"
	libvalidator "github.com/pb33f/libopenapi-validator"
)

const (
	publicFeedbackOpenAPI = "../gen/http/openapi.json"
)

func TestPublicVoteResponsesUsePersistedIntegerSchema(t *testing.T) {
	validator := publicFeedbackValidator(t)
	defer validator.Release()
	for _, method := range []string{http.MethodPost, http.MethodGet} {
		for _, vote := range []any{1, true} {
			item := map[string]any{"actor": "00000000-0000-4000-8000-000000000001", "issue": "00000000-0000-4000-8000-000000000002", "project": "00000000-0000-4000-8000-000000000003", "workspace": "00000000-0000-4000-8000-000000000004", "actor_detail": map[string]any{}, "vote": vote}
			var body any = item
			status := http.StatusCreated
			if method == http.MethodGet {
				body = []any{item}
				status = http.StatusOK
			}
			request := httptest.NewRequest(method, "http://vector.test/api/public/anchor/test-board/issues/00000000-0000-4000-8000-000000000002/votes/", nil)
			recorder := httptest.NewRecorder()
			recorder.Header().Set("Content-Type", "application/json")
			recorder.WriteHeader(status)
			if err := json.NewEncoder(recorder).Encode(body); err != nil {
				t.Fatal(err)
			}
			response := recorder.Result()
			t.Cleanup(func() {
				if err := response.Body.Close(); err != nil {
					t.Error(err)
				}
			})
			valid, validationErrors := validator.ValidateHttpResponse(request, response)
			_, integer := vote.(int)
			if valid != integer {
				t.Fatalf("%s vote=%v valid=%v: %v", method, vote, valid, validationErrors)
			}
		}
	}
}

func publicFeedbackValidator(t *testing.T) libvalidator.Validator {
	t.Helper()
	path := publicFeedbackOpenAPI
	if override := os.Getenv("PUBLIC_FEEDBACK_OPENAPI_PATH"); override != "" {
		path = override
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read OpenAPI document %s: %v", path, err)
	}
	document, err := libopenapi.NewDocument(data)
	if err != nil {
		t.Fatalf("parse OpenAPI document %s: %v", path, err)
	}
	validator, errs := libvalidator.NewValidator(document)
	if len(errs) != 0 {
		t.Fatalf("build OpenAPI validator: %v", errs)
	}
	return validator
}
