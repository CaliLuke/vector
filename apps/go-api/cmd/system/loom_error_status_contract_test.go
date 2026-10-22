package main

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"runtime"
	"strconv"
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

func TestMiddlewareStrippedDesignsDoNotRequireLoomErrorHeader(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller did not return this test file path")
	}
	moduleRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	designs := map[string]string{
		"activityreads": "activityreads/design/design.go", "apicyclereads": "apicyclereads/design/design.go",
		"apilabelreads": "apilabelreads/design/design.go", "apimodulereads": "apimodulereads/design/design.go",
		"apistatereads": "apistatereads/design/design.go", "apiuseridentity": "apiuseridentity/design/design.go",
		"cyclelifecycle": "cyclelifecycle/design/design.go", "modulelifecycle": "modulelifecycle/design/design.go",
		"invitationsmembership": "invitationsmembership/design/design.go", "notificationactions": "notificationactions/design/design.go",
		"notificationreads": "notificationreads/design/design.go", "project_label_writes": "project_label_writes/design/design.go",
		"projectmembershiplifecycle": "projectmembershiplifecycle/design/design.go",
		"ticket_collaboration":       "ticket_collaboration/design/design.go", "work_item_core": "work_item_core/design/design.go",
		"public_feedback": "design/public_feedback.go",
	}
	for design, relativePath := range designs {
		t.Run(design, func(t *testing.T) {
			path := filepath.Join(moduleRoot, relativePath)
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok || !isHeaderCall(call.Fun) {
					return true
				}
				for _, argument := range call.Args {
					literal, ok := argument.(*ast.BasicLit)
					if !ok || literal.Kind != token.STRING {
						continue
					}
					value, err := strconv.Unquote(literal.Value)
					if err == nil && strings.Contains(strings.ToLower(value), "loom-error") {
						t.Errorf("%s still maps stripped Loom-Error in %s", design, path)
					}
				}
				return true
			})
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

func isHeaderCall(expression ast.Expr) bool {
	switch function := expression.(type) {
	case *ast.Ident:
		return function.Name == "Header"
	case *ast.SelectorExpr:
		return function.Sel.Name == "Header"
	default:
		return false
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
