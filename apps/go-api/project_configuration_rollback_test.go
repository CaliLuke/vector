package systemapi

import (
	"errors"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	store "github.com/CaliLuke/go-plane/internal/projectconfiguration"
	api "github.com/CaliLuke/go-plane/projectconfiguration/gen/project_configuration"
)

func TestProjectConfigurationMappingPreservesPrimaryFailureAfterRollbackFailure(t *testing.T) {
	for _, test := range []struct {
		name     string
		primary  error
		wantName string
		wantBody string
	}{
		{"validation", &store.HTTPError{Status: 400, Body: map[string]any{"name": "required"}}, "status_400", `{"name":"required"}`},
		{"authorization", &djangoapi.HTTPError{Status: 403, Body: map[string]any{"detail": "forbidden"}}, "status_403", `{"detail":"forbidden"}`},
		{"internal", errors.New("private query details"), "status_500", `{"detail":"Something went wrong please try again later"}`},
		{"mapped", serviceError(404, map[string]any{"detail": "Not found."}), "status_404", `{"detail":"Not found."}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &projectConfigurationService{}
			got := service.mapError(errors.Join(test.primary, errors.New("private rollback details")))
			var response *api.ProjectConfigurationErrorResponse
			if !errors.As(got, &response) {
				t.Fatalf("expected transport error, got %T: %v", got, got)
			}
			if response.Name != test.wantName || string(response.Body) != test.wantBody {
				t.Fatalf("response = %s %s, want %s %s", response.Name, response.Body, test.wantName, test.wantBody)
			}
			if strings.Contains(got.Error(), "private") {
				t.Fatalf("public error exposed database details: %v", got)
			}
		})
	}
}
