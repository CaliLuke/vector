package systemapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	api "github.com/CaliLuke/go-plane/customviewcreation/gen/custom_view_creation"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestCustomViewCreationRejectsInvalidSortOrderBeforeUpdate(t *testing.T) {
	for _, value := range []string{`"NaN"`, `"Infinity"`, `"-Infinity"`, `1e309`, `"-1e309"`, `null`, `"invalid"`} {
		t.Run(value, func(t *testing.T) {
			repository := &customViewCreationRepositoryStub{}
			service, err := NewCustomViewCreation(repository, customViewCreationSessionsStub{})
			if err != nil {
				t.Fatal(err)
			}
			err = callCustomViewProjectJSON(t, service, `{"sort_order":`+value+`}`)
			status, response := customViewCreationProtocolError(err)
			if status != 400 {
				t.Fatalf("error=%v, status=%d, want 400", err, status)
			}
			encoded, marshalErr := json.Marshal(response)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if string(encoded) != `{"sort_order":["A valid number is required."]}` {
				t.Fatalf("validation body=%s", encoded)
			}
			if len(repository.calls) != 1 || repository.calls[0] != "project-target" {
				t.Fatalf("invalid sort order reached mutation: %v", repository.calls)
			}
		})
	}
}

func TestCustomViewCreationAcceptsFiniteSortOrder(t *testing.T) {
	repository := &customViewCreationRepositoryStub{}
	service, err := NewCustomViewCreation(repository, customViewCreationSessionsStub{})
	if err != nil {
		t.Fatal(err)
	}
	err = callCustomViewProjectJSON(t, service, `{"sort_order":123.25}`)
	if err != nil {
		t.Fatal(err)
	}
	if !repository.changes.HasSortOrder || repository.changes.SortOrder != 123.25 {
		t.Fatalf("finite changes=%+v", repository.changes)
	}
	if len(repository.calls) != 2 || repository.calls[1] != "project-update" {
		t.Fatalf("finite order did not reach mutation: %v", repository.calls)
	}
}

func callCustomViewProjectJSON(t *testing.T, service api.Service, body string) error {
	t.Helper()
	policy, err := loomhttp.NewRequestMetadataPolicy([]string{"Content-Type"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var requestErr error
	handler := loomhttp.RequestMetadataMiddleware(policy)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		requestErr = service.CreateProjectView(request.Context(), &api.CreateProjectViewPayload{Slug: "demo", ProjectID: "00000000-0000-4000-8000-000000000100"}, request.Body)
	}))
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)).WithContext(customViewContext("UTC"))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(httptest.NewRecorder(), request)
	return requestErr
}
