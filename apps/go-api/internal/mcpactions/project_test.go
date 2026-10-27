package mcpactions_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	projects "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	"github.com/stretchr/testify/require"
)

type projectReadStub struct {
	page, detail      json.RawMessage
	err               error
	options           projects.APIProjectListOptions
	expectedWorkspace *uuid.UUID
	reads             int
}

func newProjectReadStub() *projectReadStub {
	return &projectReadStub{page: json.RawMessage(projectPageJSON), detail: json.RawMessage(strings.Replace(projectDetailJSON, `"sort_order":null,`, "", 1))}
}

func (s *projectReadStub) APIProjects(_ context.Context, _ uuid.UUID, _ string, options projects.APIProjectListOptions) (json.RawMessage, error) {
	s.reads++
	s.options = options
	s.expectedWorkspace = options.ExpectedWorkspaceID
	return s.page, s.err
}
func (s *projectReadStub) APIProject(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, options projects.APIProjectDetailOptions) (json.RawMessage, error) {
	s.reads++
	s.expectedWorkspace = options.ExpectedWorkspaceID
	return s.detail, s.err
}
func projectCall(input string) map[string]any {
	return map[string]any{"name": "project", "arguments": json.RawMessage(input)}
}

func TestProjectReadsPreservePinnedSDKResults(t *testing.T) {
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, test := range []struct{ name, input, want string }{
			{"list", `{"action":"list"}`, projectPageJSON},
			{"zero page size", `{"action":"list","per_page":0,"cursor":""}`, projectPageJSON},
			{"retrieve", `{"action":"retrieve","project_id":"30000000-0000-4000-8000-000000000001"}`, projectDetailJSON},
		} {
			t.Run(version+"/"+test.name, func(t *testing.T) {
				f := newFixture(t)
				result := rpcVersion(t, f.handler(t, nil), "test-one", "tools/call", projectCall(test.input), version)
				require.False(t, result.IsError, result.Content)
				require.JSONEq(t, test.want, string(result.StructuredContent))
				require.NotNil(t, f.projectReads.expectedWorkspace)
				require.Equal(t, "20000000-0000-4000-8000-000000000001", f.projectReads.expectedWorkspace.String())
			})
		}
	}
}

func TestProjectListUsesSDKDefaultsAndIncludesArchivedProjects(t *testing.T) {
	f := newFixture(t)
	result := rpc(t, f.handler(t, nil), "test-one", "tools/call", projectCall(`{"action":"list","per_page":0,"cursor":""}`))
	require.False(t, result.IsError, result.Content)
	require.Equal(t, "100", *f.projectReads.options.PerPage)
	require.True(t, f.projectReads.options.IncludeArchived)
	require.Nil(t, f.projectReads.options.Cursor)
	result = rpc(t, f.handler(t, nil), "test-one", "tools/call", projectCall(`{"action":"list","per_page":2,"cursor":"2:1:0","order_by":"-name"}`))
	require.False(t, result.IsError, result.Content)
	require.Equal(t, "2", *f.projectReads.options.PerPage)
	require.Equal(t, "2:1:0", *f.projectReads.options.Cursor)
	require.Equal(t, "-name", *f.projectReads.options.OrderBy)
}

func TestProjectGeneratedValidationRejectsInvalidActionInputs(t *testing.T) {
	for _, input := range []string{
		`{"action":"retrieve"}`, `{"action":"retrieve","project_id":"invalid"}`,
		`{"action":"retrieve","project_id":"30000000-0000-4000-8000-000000000001","cursor":"100:1:0"}`,
		`{"action":"create","name":"New"}`, `{"action":"list","per_page":-1}`,
		`{"action":"list","per_page":1001}`, `{"action":"list","per_page":null}`,
	} {
		t.Run(input, func(t *testing.T) {
			f := newFixture(t)
			result := rpc(t, f.handler(t, nil), "test-one", "tools/call", projectCall(input))
			require.True(t, result.IsError)
			require.Zero(t, f.projectReads.reads)
		})
	}
}

func TestProjectApplicationFailuresAreSafe(t *testing.T) {
	for _, input := range []string{`{"action":"list"}`, `{"action":"retrieve","project_id":"30000000-0000-4000-8000-000000000001"}`} {
		for _, test := range []struct {
			name, code string
			err        error
			body       string
		}{
			{"revoked", "forbidden", projects.ErrForbidden, ""},
			{"hidden", "not_found", projects.ErrNotFound, ""},
			{"bad query", "invalid_params", projects.ErrBadRequest, ""},
			{"bad cursor", "invalid_params", &djangoapi.HTTPError{Status: 400}, ""},
			{"cursor overflow", "internal_error", &djangoapi.HTTPError{Status: 500}, ""},
			{"database unavailable", "internal_error", errors.New("private-database-error"), ""},
			{"invalid JSON", "internal_error", nil, "invalid-private-json"},
			{"null JSON", "internal_error", nil, "null"},
		} {
			t.Run(input+"/"+test.name, func(t *testing.T) {
				f := newFixture(t)
				f.projectReads.err = test.err
				f.projectReads.page = json.RawMessage(test.body)
				f.projectReads.detail = json.RawMessage(test.body)
				result := rpc(t, f.handler(t, nil), "test-one", "tools/call", projectCall(input))
				require.True(t, result.IsError)
				require.Empty(t, result.StructuredContent)
				require.Contains(t, result.Content[0].Text, "["+test.code+"]")
				require.NotContains(t, result.Content[0].Text, "private")
			})
		}
	}
}

// Defaults and fields from pinned plane-sdk 0.3.1 models/projects.py.
const projectPageJSON = `{"grouped_by":null,"sub_grouped_by":null,"total_count":1,"next_cursor":"100:1:0","prev_cursor":"100:-1:1","next_page_results":false,"prev_page_results":false,"count":1,"total_pages":1,"total_results":1,"extra_stats":null,"results":[{"id":"30000000-0000-4000-8000-000000000001","identifier":"SDK","name":"SDK project","cover_image":null,"icon_prop":{"name":"folder","color":"#444"},"emoji":null,"description":null,"cover_image_url":null,"archived_at":null}]}`

// Defaults and fields from pinned plane-sdk 0.3.1 models/projects.py.
const projectDetailJSON = `{"id":"30000000-0000-4000-8000-000000000001","total_members":1,"total_cycles":null,"total_modules":null,"is_member":true,"sort_order":null,"member_role":null,"is_deployed":null,"cover_image_url":null,"created_at":null,"updated_at":null,"deleted_at":null,"name":"SDK project","description":null,"description_text":null,"description_html":"<p>Project</p>","network":2,"identifier":"SDK","emoji":null,"icon_prop":{"name":"folder","color":"#444"},"module_view":null,"cycle_view":null,"issue_views_view":null,"page_view":null,"intake_view":null,"is_time_tracking_enabled":null,"is_issue_type_enabled":null,"guest_view_all_features":null,"cover_image":null,"archive_in":null,"close_in":null,"logo_props":null,"archived_at":null,"timezone":null,"external_source":null,"external_id":null,"created_by":null,"updated_by":null,"workspace":null,"default_assignee":null,"project_lead":null,"cover_image_asset":null,"estimate":null,"default_state":null}`

func TestDirectProjectCallsRequireIdentityAndValidInputs(t *testing.T) {
	f := newFixture(t)
	input := api.NewProjectRequestProjectList(&api.ProjectList{Action: "list"})
	_, err := f.service.Project(t.Context(), &input)
	require.Error(t, err)
	request := httptest.NewRequest(http.MethodPost, "https://vector.example/mcp/api-key", nil)
	request.Header.Set("Authorization", "Bearer test-one")
	request.Header.Set("X-Workspace-Slug", "workspace-one")
	ctx, err := f.auth.Authenticate(t.Context(), request)
	require.NoError(t, err)
	for _, input := range []*api.ProjectRequest{
		nil, {},
		new(api.NewProjectRequestProjectList(&api.ProjectList{Action: "bad"})),
		new(api.NewProjectRequestProjectList(&api.ProjectList{Action: "list", PerPage: new(-1)})),
		new(api.NewProjectRequestProjectList(&api.ProjectList{Action: "list", PerPage: new(1001)})),
		new(api.NewProjectRequestProjectRetrieve(&api.ProjectRetrieve{Action: "retrieve", ProjectID: "bad"})),
	} {
		_, err = f.service.Project(ctx, input)
		require.Error(t, err)
	}
	require.Zero(t, f.projectReads.reads)
}

func TestProjectDiscoveryOmitsWritesWhenTheyAreUnavailable(t *testing.T) {
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		t.Run(version, func(t *testing.T) {
			f := newFixture(t)
			f.projectWrites.visibility = projects.ProjectWriteVisibility{}
			discovery := rpcVersion(t, f.handler(t, nil), "test-one", "tools/list", map[string]any{}, version)
			require.Len(t, discovery.Tools, 8)
			project := discoveredTool(t, discovery, "project")
			require.Equal(t, "project", project.Name)
			require.False(t, project.Annotations["readOnlyHint"])
			require.False(t, project.Annotations["destructiveHint"])
			require.False(t, project.Annotations["openWorldHint"])
			require.JSONEq(t, `{"anyOf":[{"type":"string","enum":["list"]},{"type":"string","enum":["retrieve"]}]}`, string(project.InputSchema.Properties["action"]))
			require.Len(t, project.InputSchema.OneOf, 2)
			for _, branch := range project.InputSchema.OneOf {
				var schema struct{ AdditionalProperties *bool }
				require.NoError(t, json.Unmarshal(branch, &schema))
				require.NotNil(t, schema.AdditionalProperties)
				require.False(t, *schema.AdditionalProperties)
			}
		})
	}
}
