package systemapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/CaliLuke/go-plane/cyclelifecycle/gen/cycle_lifecycle_api"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/google/uuid"
)

type cycleJSONInputRepositorySpy struct {
	cycleLifecycleRepositoryStub
	patchCalls int
}

func (spy *cycleJSONInputRepositorySpy) Patch(ctx context.Context, slug, projectID, cycleID, actor string, apiV1 bool, requested, activity map[string]any) (map[string]any, error) {
	spy.patchCalls++
	return spy.cycleLifecycleRepositoryStub.Patch(ctx, slug, projectID, cycleID, actor, apiV1, requested, activity)
}

func cycleJSONInputPatch(t *testing.T, service *cycleLifecycleService, contentType string, body []byte) (map[string]any, error) {
	t.Helper()

	policy, err := loomhttp.NewRequestMetadataPolicy([]string{"Content-Type"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	var patchErr error
	handler := loomhttp.RequestMetadataMiddleware(policy)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		result, patchErr = service.patch(request.Context(), "workspace", "project", "cycle", io.NopCloser(bytes.NewReader(body)), false)
	}))
	request := httptest.NewRequest(http.MethodPatch, "/api/workspaces/workspace/projects/project/cycles/cycle/", nil)
	request.Header.Set("Content-Type", contentType)
	request = request.WithContext(context.WithValue(request.Context(), cycleLifecyclePrincipalKey{}, cycleLifecyclePrincipal{userID: uuid.MustParse("00000000-0000-4000-8000-000000000303")}))
	handler.ServeHTTP(httptest.NewRecorder(), request)
	return result, patchErr
}

func TestCyclePatchValidatesUTF8JSONBeforePersistenceAndPreservesUTF8Bytes(t *testing.T) {
	invalidUTF8 := []byte(`{"name":"`)
	invalidUTF8 = append(invalidUTF8, 0xff)
	invalidUTF8 = append(invalidUTF8, []byte(`"}`)...)

	tests := []struct {
		name        string
		contentType string
		body        []byte
		wantStatus  string
		wantName    string
	}{
		{
			name:        "invalid UTF-8 body is rejected before persistence and activity",
			contentType: "application/json",
			body:        invalidUTF8,
			wantStatus:  "status_400",
		},
		{
			name:        "conflicting duplicate charset is rejected on empty body",
			contentType: "application/json; charset=utf-8; charset=iso-8859-1",
			body:        []byte{},
			wantStatus:  "status_415",
		},
		{
			name:        "valid UTF-8 remains UTF-8 under legacy charset label",
			contentType: "application/json; charset=iso-8859-1",
			body:        []byte(`{"name":"café"}`),
			wantName:    "café",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &cycleJSONInputRepositorySpy{cycleLifecycleRepositoryStub: cycleLifecycleRepositoryStub{result: map[string]any{}}}
			service := &cycleLifecycleService{repo: repository}
			_, err := cycleJSONInputPatch(t, service, test.contentType, test.body)

			if test.wantStatus != "" {
				var response *api.CycleLifecycleError
				if !errors.As(err, &response) || response.Kind != test.wantStatus {
					t.Fatalf("PATCH error = %T %v, want %s", err, err, test.wantStatus)
				}
				if repository.patchCalls != 0 || repository.requested != nil || repository.activity != nil {
					t.Fatalf("invalid JSON input reached persistence/activity: calls=%d requested=%#v activity=%#v", repository.patchCalls, repository.requested, repository.activity)
				}
				return
			}

			if err != nil {
				t.Fatalf("valid UTF-8 JSON was rejected for its charset label: %v", err)
			}
			if repository.patchCalls != 1 || repository.requested["name"] != test.wantName || repository.activity["name"] != test.wantName {
				t.Fatalf("UTF-8 name was not preserved at the cycle boundary: calls=%d requested=%#v activity=%#v", repository.patchCalls, repository.requested, repository.activity)
			}
		})
	}
}
