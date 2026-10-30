package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	activityreadsserver "github.com/CaliLuke/go-plane/activityreads/gen/http/plane_internal_django_migration_api/server"
	cyclelifecycleserver "github.com/CaliLuke/go-plane/cyclelifecycle/gen/http/cycle_lifecycle_api/server"
	publicfeedbackclient "github.com/CaliLuke/go-plane/gen/http/public_feedback/client"
	publicfeedbackserver "github.com/CaliLuke/go-plane/gen/http/public_feedback/server"
	publicfeedback "github.com/CaliLuke/go-plane/gen/public_feedback"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	invitationsmembershipserver "github.com/CaliLuke/go-plane/invitationsmembership/gen/http/invitations_membership/server"
	modulelifecycleclient "github.com/CaliLuke/go-plane/modulelifecycle/gen/http/module_lifecycle_api/client"
	modulelifecycleserver "github.com/CaliLuke/go-plane/modulelifecycle/gen/http/module_lifecycle_api/server"
	modulelifecycleapi "github.com/CaliLuke/go-plane/modulelifecycle/gen/module_lifecycle_api"
	notificationactionsserver "github.com/CaliLuke/go-plane/notificationactions/gen/http/notification_actions/server"
	notificationreadsserver "github.com/CaliLuke/go-plane/notificationreads/gen/http/notifications/server"
	projectlabelwritesserver "github.com/CaliLuke/go-plane/project_label_writes/gen/http/project_label_writes/server"
	projectmembershiplifecycleserver "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/http/project_membership_lifecycle/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

func TestGeneratedErrorResponseContractsUseStatusIdentity(t *testing.T) {
	services := []struct {
		name  string
		cases func() []loomhttp.ResponseContractCase
	}{
		{name: "activityreads", cases: activityreadsserver.ResponseContractCases},
		{name: "cyclelifecycle", cases: cyclelifecycleserver.ResponseContractCases},
		{name: "invitationsmembership", cases: invitationsmembershipserver.ResponseContractCases},
		{name: "modulelifecycle", cases: modulelifecycleserver.ResponseContractCases},
		{name: "notificationactions", cases: notificationactionsserver.ResponseContractCases},
		{name: "notificationreads", cases: notificationreadsserver.ResponseContractCases},
		{name: "project_label_writes", cases: projectlabelwritesserver.ResponseContractCases},
		{name: "projectmembershiplifecycle", cases: projectmembershiplifecycleserver.ResponseContractCases},
		{name: "public_feedback", cases: publicfeedbackserver.ResponseContractCases},
	}
	for _, service := range services {
		t.Run(service.name, func(t *testing.T) {
			var errorCases int
			for _, contract := range service.cases() {
				if contract.Kind != loomhttp.ResponseContractError {
					continue
				}
				errorCases++
				if contract.ErrorName == "" || !contract.ErrorNameFromStatus {
					t.Errorf("error response contract %q = %+v, want nonempty status-derived identity", contract.ID, contract)
				}
				for _, header := range contract.RequiredHeaders {
					if strings.EqualFold(header, "Loom-Error") {
						t.Errorf("error response contract %q still requires stripped Loom-Error header", contract.ID)
					}
				}
			}
			if errorCases == 0 {
				t.Fatal("generated response contract manifest contains no error cases")
			}
		})
	}
}

func TestGeneratedResponsesDoNotPublishStrippedErrorHeader(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller did not return this test file path")
	}
	moduleRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	documents := map[string]string{
		"activityreads": "activityreads/gen/http/openapi.json", "apicyclereads": "apicyclereads/gen/http/openapi.json",
		"apilabelreads": "apilabelreads/gen/http/openapi.json", "apimodulereads": "apimodulereads/gen/http/openapi.json",
		"apistatereads": "apistatereads/gen/http/openapi.json", "apiuseridentity": "apiuseridentity/gen/http/openapi.json",
		"cyclelifecycle": "cyclelifecycle/gen/http/openapi.json", "modulelifecycle": "modulelifecycle/gen/http/openapi.json",
		"invitationsmembership": "invitationsmembership/gen/http/openapi.json", "notificationactions": "notificationactions/gen/http/openapi.json",
		"notificationreads": "notificationreads/gen/http/openapi.json", "project_label_writes": "project_label_writes/gen/http/openapi.json",
		"projectmembershiplifecycle": "projectmembershiplifecycle/gen/http/openapi.json",
		"ticket_collaboration":       "ticket_collaboration/gen/http/openapi.json", "work_item_core": "work_item_core/gen/http/openapi.json",
		"public_feedback": "gen/http/openapi.json",
	}
	for name, path := range documents {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(moduleRoot, path))
			if err != nil {
				t.Fatal(err)
			}
			type response struct {
				Ref     string                     `json:"$ref"`
				Headers map[string]json.RawMessage `json:"headers"`
			}
			var document struct {
				Paths      map[string]map[string]json.RawMessage `json:"paths"`
				Components struct {
					Responses map[string]response `json:"responses"`
				} `json:"components"`
			}
			if err := json.Unmarshal(data, &document); err != nil {
				t.Fatal(err)
			}
			checked := 0
			for path, item := range document.Paths {
				for _, method := range []string{"get", "post", "put", "patch", "delete", "options", "head", "trace"} {
					raw, exists := item[method]
					if !exists {
						continue
					}
					var operation struct {
						Responses map[string]response `json:"responses"`
					}
					if err := json.Unmarshal(raw, &operation); err != nil {
						t.Fatal(err)
					}
					for status, value := range operation.Responses {
						seen := map[string]bool{}
						for value.Ref != "" {
							ref := value.Ref
							const prefix = "#/components/responses/"
							if !strings.HasPrefix(ref, prefix) || seen[ref] {
								t.Fatalf("unsupported or cyclic response reference %q", ref)
							}
							seen[ref] = true
							var found bool
							value, found = document.Components.Responses[strings.TrimPrefix(ref, prefix)]
							if !found {
								t.Fatalf("missing response reference %q", ref)
							}
						}
						checked++
						for header := range value.Headers {
							if strings.EqualFold(header, "Loom-Error") {
								t.Errorf("%s %s status %s publishes stripped error header", method, path, status)
							}
						}
					}
				}
			}
			if checked == 0 {
				t.Fatal("no generated responses checked")
			}
		})
	}
}

func TestPublicFeedbackGeneratedClientDecodesKindFromStatusWithoutHeader(t *testing.T) {
	contractID := "public_feedback.vote_delete.error.status_400.400"
	var contract *loomhttp.ResponseContractCase
	for _, candidate := range publicfeedbackserver.VoteDeleteResponseContractCases() {
		if candidate.ID == contractID {
			contract = &candidate
			break
		}
	}
	if contract == nil || !contract.ErrorNameFromStatus {
		t.Fatalf("public-feedback generated response contract = %+v, want status-derived identity", contract)
	}

	items := &publicFeedbackMiddlewareItems{
		voteDeleteErr: &djangoapi.HTTPError{
			Status: http.StatusBadRequest,
			Body:   map[string]any{"error": "The required object does not exist."},
		},
	}
	session := &publicProjectMetadataThrottleStub{allowed: true, userID: "00000000-0000-4000-8000-000000000004"}
	handler := publicFeedbackProductionHandler(session, items)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	capture := &apiV1ResponseCaptureDoer{doer: server.Client(), contract: contract}
	client := publicfeedbackclient.NewClient(serverURL.Scheme, serverURL.Host, capture, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, true)
	_, err = client.API.VoteDelete()(context.Background(), &publicfeedback.VoteDeletePayload{
		Anchor:  "public-feedback-test",
		IssueID: "00000000-0000-4000-8000-000000000109",
	})
	var response *publicfeedback.PublicFeedbackError
	if !errors.As(err, &response) {
		t.Fatalf("generated client error = %T %#v, want typed public feedback error", err, err)
	}
	if response.Kind != "status_400" {
		t.Fatalf("generated kind = %q, want status-derived status_400", response.Kind)
	}
	if !loom.JSONValueEqual(response.Body, map[string]any{"error": "The required object does not exist."}) {
		t.Fatalf("generated error body = %s", response.Body)
	}
	if capture.headers.Get("Loom-Error") != "" {
		t.Fatalf("Django-compatible response unexpectedly includes Loom-Error: %v", capture.headers)
	}
	if capture.contractErr != nil {
		t.Fatalf("generated public-feedback response contract: %v", capture.contractErr)
	}
}

func TestAPIV1CompatibilityGeneratedClientDecodesKindErrorFromStatus(t *testing.T) {
	mux := loomhttp.NewMuxer()
	generated := modulelifecycleserver.New(
		modulelifecycleapi.NewEndpoints(apiV1ModuleKindErrorService{}),
		mux,
		loomhttp.RequestDecoder,
		loomhttp.ResponseEncoder,
		nil,
		nil,
	)
	generated.Mount(mux)
	handler := apiV1CompatibilityMiddleware(apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit})(mux)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	var contract *loomhttp.ResponseContractCase
	for _, candidate := range modulelifecycleserver.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPkResponseContractCases() {
		if candidate.ID == "ModuleLifecycleAPI.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPk.error.status_400.400" {
			contract = &candidate
			break
		}
	}
	if contract == nil || !contract.ErrorNameFromStatus {
		t.Fatalf("generated module status 400 response contract = %+v, want explicit status-derived identity", contract)
	}
	capture := &apiV1ResponseCaptureDoer{doer: server.Client(), contract: contract}
	client := modulelifecycleclient.NewClient(serverURL.Scheme, serverURL.Host, capture, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, true)
	_, err = client.API.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPk()(context.Background(), &modulelifecycleapi.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPkPayload{
		Slug: "acme", ProjectID: "11111111-1111-4111-8111-111111111111", Pk: "22222222-2222-4222-8222-222222222222",
	})
	var response *modulelifecycleapi.ModuleLifecycleError
	if !errors.As(err, &response) {
		t.Fatalf("generated client error = %T %#v, want typed kind error", err, err)
	}
	if response.Kind != "status_400" {
		t.Fatalf("generated error kind = %q, want status-derived status_400", response.Kind)
	}
	if !loom.JSONValueEqual(response.Body, map[string]any{"detail": "This module cannot be deleted."}) {
		t.Fatalf("module error body = %s", response.Body)
	}
	if capture.headers.Get("Loom-Error") != "" {
		t.Fatalf("Django-compatible response unexpectedly includes Loom-Error: %v", capture.headers)
	}
	if capture.contractErr != nil {
		t.Fatalf("generated response contract: %v", capture.contractErr)
	}
}

type apiV1ModuleKindErrorService struct {
	modulelifecycleapi.Service
}

func (apiV1ModuleKindErrorService) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, nil
}

func (apiV1ModuleKindErrorService) DeleteAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPk(context.Context, *modulelifecycleapi.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPkPayload) error {
	return &modulelifecycleapi.ModuleLifecycleError{
		Kind: "status_400",
		Body: loom.MustJSONValueFrom(map[string]string{"detail": "This module cannot be deleted."}),
	}
}
