package systemapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	api "github.com/CaliLuke/go-plane/workspaceexports/gen/workspace_exports"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestWorkspaceExportStringFieldsUseSharedPythonCompatibility(t *testing.T) {
	input, err := djangoinput.Parse(strings.NewReader(`{"x_axis":{"z":[true,null,"é"],"a":0},"y_axis":false,"segment":1.0}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	for field, want := range map[string]string{"x_axis": "{'z': [True, None, 'é'], 'a': 0}", "y_axis": "", "segment": "1.0", "missing": ""} {
		got, err := workspaceExportStringOrFalse(input, field)
		if err != nil || got != want {
			t.Errorf("%s = %q, %v; want %q", field, got, err, want)
		}
	}
}

type projectValuesRepository struct {
	projects      []any
	includeCustom bool
	called        bool
}

func (*projectValuesRepository) Authorized(context.Context, string, string) (bool, error) {
	return true, nil
}
func (*projectValuesRepository) UserEmail(context.Context, string) (string, error) { return "", nil }
func (repository *projectValuesRepository) CreateIssueExport(_ context.Context, _, _, _ string, projects []any, includeCustom bool) (json.RawMessage, string, string, []any, error) {
	repository.called = true
	repository.includeCustom = includeCustom
	repository.projects = append([]any(nil), projects...)
	if len(projects) == 0 {
		projects = []any{"default-project-id"}
	}
	return json.RawMessage(`{"message":"queued"}`), "workspace-id", "token-id", projects, nil
}
func (*projectValuesRepository) History(context.Context, string, string, djangoapi.Query) (json.RawMessage, error) {
	return nil, nil
}

type projectValuesSessions struct{}

func (projectValuesSessions) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{}, nil
}

type projectValuesPublisher struct {
	kwargs map[string]any
	calls  int
}

func (publisher *projectValuesPublisher) Task(_ context.Context, _ string, _ []any, kwargs map[string]any) error {
	publisher.calls++
	publisher.kwargs = kwargs
	return nil
}

type deniedWorkspaceExportsRepository struct {
	historyCalls int
	createCalls  int
}

func (*deniedWorkspaceExportsRepository) Authorized(context.Context, string, string) (bool, error) {
	return false, nil
}
func (*deniedWorkspaceExportsRepository) UserEmail(context.Context, string) (string, error) {
	return "", nil
}
func (repository *deniedWorkspaceExportsRepository) CreateIssueExport(context.Context, string, string, string, []any, bool) (json.RawMessage, string, string, []any, error) {
	repository.createCalls++
	return nil, "", "", nil, nil
}
func (repository *deniedWorkspaceExportsRepository) History(context.Context, string, string, djangoapi.Query) (json.RawMessage, error) {
	repository.historyCalls++
	return nil, nil
}

func TestWorkspaceExportProjectValuesRemainOriginalInTaskPayload(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want []any
	}{
		{name: "integer UUID", body: `{"provider":"csv","project":[123]}`, want: []any{json.Number("123")}},
		{name: "UUID string", body: `{"provider":"csv","project":["123e4567-e89b-12d3-a456-426614174000"]}`, want: []any{"123e4567-e89b-12d3-a456-426614174000"}},
		{name: "null UUID array member", body: `{"provider":"csv","project":[null]}`, want: []any{nil}},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &projectValuesRepository{}
			publisher := &projectValuesPublisher{}
			service, err := NewWorkspaceExports(repository, projectValuesSessions{}, publisher)
			if err != nil {
				t.Fatal(err)
			}
			ctx := workspaceExportTestRequestContext(t, test.body)
			ctx = context.WithValue(ctx, workspaceExportsPrincipalKey{}, workspaceExportsPrincipal{userID: "00000000-0000-4000-8000-000000000001"})
			_, responseBody, err := service.CreateIssueExport(ctx, &api.CreateIssueExportPayload{Slug: "demo"}, io.NopCloser(strings.NewReader(test.body)))
			if err != nil {
				t.Fatal(err)
			}
			defer testsupport.Close(t, responseBody)
			response, err := io.ReadAll(responseBody)
			if err != nil {
				t.Fatal(err)
			}
			if string(response) != `{"message":"Once the export is ready you will be able to download it"}` {
				t.Fatalf("create response = %s", response)
			}
			if !repository.called || !reflect.DeepEqual(repository.projects, test.want) {
				t.Fatalf("repository projects=%#v called=%v, want %#v", repository.projects, repository.called, test.want)
			}
			queued, ok := publisher.kwargs["project_ids"].([]any)
			if !ok || !reflect.DeepEqual(queued, test.want) {
				t.Fatalf("queued project_ids=%#v, want original %#v", publisher.kwargs["project_ids"], test.want)
			}
		})
	}
}

func TestWorkspaceExportProjectValuesUseSourceFallbackAndRejectNonArray(t *testing.T) {
	for _, test := range []struct {
		name    string
		body    string
		want    []any
		wantErr bool
	}{
		{name: "null means use defaults", body: `{"project":null}`},
		{name: "empty array means use defaults", body: `{"project":[]}`},
		{name: "false means use defaults", body: `{"project":false}`},
		{name: "malformed scalar rejected", body: `{"project":"123"}`, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			input, err := djangoinput.Parse(strings.NewReader(test.body), "application/json")
			if err != nil {
				t.Fatal(err)
			}
			got, err := workspaceExportProjectValues(input)
			if (err != nil) != test.wantErr || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("projects=%#v err=%v, want %#v err=%v", got, err, test.want, test.wantErr)
			}
		})
	}
}

func TestWorkspaceExportsRejectSoftDeletedMemberBeforeHistoryOrEffects(t *testing.T) {
	repository := &deniedWorkspaceExportsRepository{}
	publisher := &projectValuesPublisher{}
	service, err := NewWorkspaceExports(repository, projectValuesSessions{}, publisher)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), workspaceExportsPrincipalKey{}, workspaceExportsPrincipal{
		userID: "00000000-0000-4000-8000-000000000001",
		email:  "user@example.test",
	})

	_, _, historyErr := service.GetIssueExports(ctx, &api.GetIssueExportsPayload{
		Slug: "demo", PerPage: stringPointer("20"), Cursor: stringPointer("20:1:0"),
	})
	_, _, issueExportErr := service.CreateIssueExport(ctx, &api.CreateIssueExportPayload{Slug: "demo"}, io.NopCloser(strings.NewReader(`{"provider":"csv"}`)))
	_, _, analyticsErr := service.ExportAnalytics(ctx, &api.ExportAnalyticsPayload{Slug: "demo"}, io.NopCloser(strings.NewReader(`{"x_axis":"priority","y_axis":"issue_count"}`)))
	for name, callErr := range map[string]error{"history": historyErr, "issue export": issueExportErr, "analytics export": analyticsErr} {
		status, _ := workspaceExportsProtocolError(callErr)
		if status != http.StatusForbidden {
			t.Errorf("%s status = %d, want %d (error %v)", name, status, http.StatusForbidden, callErr)
		}
	}
	if repository.historyCalls != 0 || repository.createCalls != 0 {
		t.Fatalf("unauthorized repository calls: history=%d create=%d, want none", repository.historyCalls, repository.createCalls)
	}
	if publisher.calls != 0 {
		t.Fatalf("published %d task(s) for a soft-deleted member, want none", publisher.calls)
	}
}

func workspaceExportTestRequestContext(t *testing.T, body string) context.Context {
	t.Helper()
	policy, err := loomhttp.NewRequestMetadataPolicy([]string{"Content-Type"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/workspaces/demo/export-issues/", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	var result context.Context
	loomhttp.RequestMetadataMiddleware(policy)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		result = request.Context()
	})).ServeHTTP(httptest.NewRecorder(), request)
	if result == nil {
		t.Fatal("request metadata middleware did not capture context")
	}
	return result
}

func TestWorkspaceExportCustomPropertiesRequiresExplicitBoolean(t *testing.T) {
	for _, test := range []struct {
		name, body string
		want       bool
		status     int
	}{
		{name: "default stays standard", body: `{"provider":"csv"}`, status: 200},
		{name: "explicit opt in", body: `{"provider":"csv","include_custom_properties":true}`, want: true, status: 200},
		{name: "explicit opt out", body: `{"provider":"csv","include_custom_properties":false}`, status: 200},
		{name: "string is not consent", body: `{"provider":"csv","include_custom_properties":"true"}`, status: 400},
		{name: "null rejected", body: `{"provider":"csv","include_custom_properties":null}`, status: 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &projectValuesRepository{}
			publisher := &projectValuesPublisher{}
			service, err := NewWorkspaceExports(repository, projectValuesSessions{}, publisher)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.WithValue(workspaceExportTestRequestContext(t, test.body), workspaceExportsPrincipalKey{}, workspaceExportsPrincipal{userID: "00000000-0000-4000-8000-000000000001"})
			_, body, err := service.CreateIssueExport(ctx, &api.CreateIssueExportPayload{Slug: "demo"}, io.NopCloser(strings.NewReader(test.body)))
			if body != nil {
				defer func() {
					if closeErr := body.Close(); closeErr != nil {
						t.Error(closeErr)
					}
				}()
			}
			if test.status != 200 {
				status, _ := workspaceExportsProtocolError(err)
				if status != test.status || repository.called || publisher.calls != 0 {
					t.Fatalf("status=%d error=%v created=%v published=%d", status, err, repository.called, publisher.calls)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if repository.includeCustom != test.want || publisher.kwargs["include_custom_properties"] != test.want {
				t.Fatalf("stored=%v queued=%v want=%v", repository.includeCustom, publisher.kwargs["include_custom_properties"], test.want)
			}
		})
	}
}
