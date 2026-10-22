package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	api "github.com/CaliLuke/go-plane/gen/project_membership_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	"github.com/CaliLuke/loom/security"
	"uuid"
)

const projectMembershipFixture = `{
  "id":"00000000-0000-4000-8000-000000000001",
  "created_at":"2026-09-01T00:00:00Z",
  "updated_at":"2026-09-01T00:00:00Z",
  "created_by":null,
  "updated_by":null,
  "deleted_at":null,
  "comment":null,
  "role":15,
  "view_props":{},
  "default_props":{},
  "preferences":{},
  "sort_order":65535,
  "is_active":true,
  "member":{"id":"00000000-0000-4000-8000-000000000002","first_name":"Project","last_name":"Member","avatar":"","avatar_url":null,"is_bot":false,"display_name":"Project Member"},
  "project":{"id":"00000000-0000-4000-8000-000000000003","identifier":"MEM","name":"Membership Project","cover_image":null,"cover_image_url":null,"logo_props":{},"description":"Membership read contract"},
  "workspace":{"id":"00000000-0000-4000-8000-000000000004","name":"Membership","slug":"membership","logo_url":null}
}`

type projectMembershipReadsRepositoryStub struct {
	membership              json.RawMessage
	roles                   json.RawMessage
	workspaceProjectMembers json.RawMessage
	projectMember           json.RawMessage
	err                     error
}

func (repository *projectMembershipReadsRepositoryStub) AuthorizeWorkspace(
	context.Context, uuid.UUID, string,
) error {
	return repository.err
}

func (repository *projectMembershipReadsRepositoryStub) Membership(
	context.Context,
	uuid.UUID,
	string,
	uuid.UUID,
) (json.RawMessage, error) {
	return repository.membership, repository.err
}

func (repository *projectMembershipReadsRepositoryStub) Roles(
	context.Context,
	uuid.UUID,
	string,
) (json.RawMessage, error) {
	return repository.roles, repository.err
}

func (repository *projectMembershipReadsRepositoryStub) WorkspaceMembers(
	context.Context, uuid.UUID, string, []string,
) (json.RawMessage, error) {
	return json.RawMessage(`[]`), repository.err
}

func (repository *projectMembershipReadsRepositoryStub) WorkspaceMember(
	context.Context, uuid.UUID, string, uuid.UUID, []string,
) (json.RawMessage, error) {
	return json.RawMessage(`{}`), repository.err
}

func (repository *projectMembershipReadsRepositoryStub) WorkspaceProjectMembers(
	context.Context, uuid.UUID, string,
) (json.RawMessage, error) {
	if repository.workspaceProjectMembers != nil {
		return repository.workspaceProjectMembers, repository.err
	}
	return json.RawMessage(`{}`), repository.err
}

func (repository *projectMembershipReadsRepositoryStub) ProjectMembers(
	context.Context, uuid.UUID, string, uuid.UUID,
) (json.RawMessage, error) {
	return json.RawMessage(`[]`), repository.err
}

func (repository *projectMembershipReadsRepositoryStub) ProjectMember(
	context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID,
) (json.RawMessage, error) {
	if repository.projectMember != nil {
		return repository.projectMember, repository.err
	}
	return json.RawMessage(`{
		"id":"00000000-0000-4000-8000-000000000004",
		"role":15,
		"member":"00000000-0000-4000-8000-000000000005",
		"project":"00000000-0000-4000-8000-000000000003",
		"original_role":15,
		"created_at":"2026-09-01T00:00:00Z"
	}`), repository.err
}

func (repository *projectMembershipReadsRepositoryStub) Preference(
	context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID,
) (json.RawMessage, error) {
	return json.RawMessage(`{}`), repository.err
}

type projectMembershipReadsSessionsStub struct{ err error }

func (sessions projectMembershipReadsSessionsStub) Validate(
	context.Context,
	string,
) (djangosession.Principal, error) {
	if sessions.err != nil {
		return djangosession.Principal{}, sessions.err
	}
	return djangosession.Principal{UserID: "00000000-0000-4000-8000-000000000010"}, nil
}

func TestNewProjectMembershipReadsRequiresDependencies(t *testing.T) {
	if _, err := NewProjectMembershipReads(nil, projectMembershipReadsSessionsStub{}); err == nil {
		t.Fatal("missing repository returned nil error")
	}
	if _, err := NewProjectMembershipReads(&projectMembershipReadsRepositoryStub{}, nil); err == nil {
		t.Fatal("missing session validator returned nil error")
	}
}

func TestProjectMembershipReadsAuthenticatesAndDecodesResults(t *testing.T) {
	serviceValue, err := NewProjectMembershipReads(
		&projectMembershipReadsRepositoryStub{
			membership: json.RawMessage(projectMembershipFixture),
			roles:      json.RawMessage(`{"00000000-0000-4000-8000-000000000003":15}`),
		},
		projectMembershipReadsSessionsStub{},
	)
	if err != nil {
		t.Fatal(err)
	}
	service := serviceValue.(*projectMembershipReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "session", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	membership, err := service.Membership(ctx, &api.MembershipPayload{
		Slug:      "membership",
		ProjectID: "00000000-0000-4000-8000-000000000003",
	})
	if err != nil || membership.Role != 15 || membership.Member.DisplayName != "Project Member" {
		t.Fatalf("membership=%+v err=%v", membership, err)
	}
	roles, err := service.Roles(ctx, &api.RolesPayload{Slug: "membership"})
	if err != nil || roles["00000000-0000-4000-8000-000000000003"] != 15 {
		t.Fatalf("roles=%+v err=%v", roles, err)
	}
}

func TestProjectMembershipReadsMapsAuthenticationAndRepositoryErrors(t *testing.T) {
	serviceValue, _ := NewProjectMembershipReads(
		&projectMembershipReadsRepositoryStub{err: store.ErrNotFound},
		projectMembershipReadsSessionsStub{err: errors.New("invalid")},
	)
	service := serviceValue.(*projectMembershipReadsService)
	if _, err := service.APIKeyAuth(context.Background(), "bad", &security.APIKeyScheme{}); err == nil {
		t.Fatal("invalid session returned nil error")
	}

	ctx := context.WithValue(
		context.Background(),
		projectMembershipReadsPrincipal{},
		uuid.MustParse("00000000-0000-4000-8000-000000000010"),
	)
	_, err := service.Membership(ctx, &api.MembershipPayload{
		Slug:      "membership",
		ProjectID: "00000000-0000-4000-8000-000000000003",
	})
	var notFound *api.ProjectMembershipReadsNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("err=%T %v", err, err)
	}

	service.repository = &projectMembershipReadsRepositoryStub{err: store.ErrForbidden}
	_, err = service.Roles(ctx, &api.RolesPayload{Slug: "membership"})
	var forbidden *api.ProjectMembershipReadsForbidden
	if !errors.As(err, &forbidden) {
		t.Fatalf("err=%T %v", err, err)
	}

	_, err = service.WorkspaceMembers(ctx, &api.WorkspaceMembersPayload{Slug: "membership"})
	var permissionDenied *api.ProjectMembershipReadsPermissionDenied
	if !errors.As(err, &permissionDenied) || permissionDenied.Message != "You don't have the required permissions." {
		t.Fatalf("err=%T %v", err, err)
	}

	_, err = service.WorkspaceProjectMembers(ctx, &api.WorkspaceProjectMembersPayload{Slug: "membership"})
	var aggregateForbidden *api.ProjectMembershipReadsForbidden
	if !errors.As(err, &aggregateForbidden) || aggregateForbidden.Detail != "You do not have permission to perform this action." {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestMemberDirectorySearchTermsMatchDRFSplitting(t *testing.T) {
	for _, test := range []struct {
		name  string
		value string
		want  []string
	}{
		{name: "whitespace", value: "Alex Smith", want: []string{"Alex", "Smith"}},
		{name: "commas", value: "Alex,Smith,,", want: []string{"Alex", "Smith"}},
		{name: "quoted phrase", value: `"Alex Smith" Jones`, want: []string{"Alex Smith", "Jones"}},
		{name: "escaped quote", value: `"Alex \"Ace\" Smith"`, want: []string{`Alex "Ace" Smith`}},
		{name: "unclosed quote", value: `"Alex Smith`, want: []string{`"Alex`, "Smith"}},
		{name: "unicode whitespace", value: "Alex\u00a0Smith", want: []string{"Alex", "Smith"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := memberDirectorySearchTerms(test.value)
			if len(got) != len(test.want) {
				t.Fatalf("got=%q want=%q", got, test.want)
			}
			for index := range got {
				if got[index] != test.want[index] {
					t.Fatalf("got=%q want=%q", got, test.want)
				}
			}
		})
	}
}

func TestMemberDirectorySearchPatternsEscapeLikeMetacharacters(t *testing.T) {
	got := memberDirectorySearchPatterns(`100% under_score back\slash`)
	want := []string{`100\%`, `under\_score`, `back\\slash`}
	if len(got) != len(want) {
		t.Fatalf("got=%q want=%q", got, want)
	}
	for index := range got {
		if got[index] != want[index] {
			t.Fatalf("got=%q want=%q", got, want)
		}
	}
}

func TestMemberDirectoryPreservesArbitraryJSONNumbersInTheHTTPResponse(t *testing.T) {
	data := json.RawMessage(`[{
		"id":"00000000-0000-4000-8000-000000000001",
		"created_at":"2026-09-01T00:00:00Z",
		"updated_at":"2026-09-01T00:00:00Z",
		"created_by":null,
		"updated_by":null,
		"deleted_at":null,
		"workspace":"00000000-0000-4000-8000-000000000002",
		"member":{
			"id":"00000000-0000-4000-8000-000000000003",
			"first_name":"Large",
			"last_name":"Number",
			"avatar":"",
			"avatar_url":null,
			"is_bot":false,
			"display_name":"Large Number"
		},
		"role":15,
		"is_active":true,
		"company_role":"Engineer",
		"view_props":{"integer":9007199254740993,"decimal":0.1234567890123456789},
		"default_props":{},
		"issue_props":{},
		"explored_features":{},
		"getting_started_checklist":{},
		"tips":{}
	}]`)
	capture := &ProjectMembershipReadsResponseCapture{}
	ctx := WithProjectMembershipReadsResponseCapture(context.Background(), capture)
	result, err := decodeMemberDirectory[[]*api.WorkspaceMembershipDirectory](ctx, data, nil, "")
	if err != nil || len(result) != 1 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if string(capture.Data) != string(data) {
		t.Fatalf("captured JSON changed:\n got %s\nwant %s", capture.Data, data)
	}
	encoded, err := json.Marshal(result)
	if err != nil || !strings.Contains(string(encoded), `"company_role":"Engineer"`) {
		t.Fatalf("company_role did not decode as nullable text: %s err=%v", encoded, err)
	}
}

func TestProjectMembershipDirectoryDecodesDynamicResponses(t *testing.T) {
	serviceValue, _ := NewProjectMembershipReads(
		&projectMembershipReadsRepositoryStub{},
		projectMembershipReadsSessionsStub{},
	)
	service := serviceValue.(*projectMembershipReadsService)
	ctx := context.WithValue(
		context.Background(),
		projectMembershipReadsPrincipal{},
		uuid.MustParse("00000000-0000-4000-8000-000000000010"),
	)
	result, err := service.ProjectMember(ctx, &api.ProjectMemberPayload{
		Slug:      "membership",
		ProjectID: "00000000-0000-4000-8000-000000000003",
		Pk:        "00000000-0000-4000-8000-000000000004",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	guest, ok := result.AsProjectMembershipGuestDirectory()
	if !ok || guest == nil || guest.ID == "" {
		t.Fatalf("result=%T %#v", result, result)
	}
}

func TestProjectMembershipDirectoryDecodesFullDetailWithoutOriginalRole(t *testing.T) {
	fullDetail := json.RawMessage(`{
		"id":"00000000-0000-4000-8000-000000000004",
		"role":15,
		"member":{
			"id":"00000000-0000-4000-8000-000000000005",
			"first_name":"Full",
			"last_name":"Member",
			"avatar":"",
			"avatar_url":null,
			"is_bot":false,
			"display_name":"Full Member",
			"email":""
		},
		"project":{
			"id":"00000000-0000-4000-8000-000000000003",
			"identifier":"MEM",
			"name":"Membership Project",
			"cover_image":null,
			"cover_image_url":null,
			"logo_props":{},
			"description":"Membership read contract"
		},
		"created_at":"2026-09-01T00:00:00Z",
		"updated_at":"2026-09-01T00:00:00Z",
		"created_by":null,
		"updated_by":null,
		"deleted_at":null,
		"comment":null,
		"view_props":"compact",
		"default_props":[1,2],
		"preferences":{"integer":9007199254740993},
		"sort_order":65535,
		"is_active":true,
		"workspace":{
			"id":"00000000-0000-4000-8000-000000000006",
			"name":"Membership",
			"slug":"membership",
			"logo_url":null
		}
	}`)
	serviceValue, _ := NewProjectMembershipReads(
		&projectMembershipReadsRepositoryStub{projectMember: fullDetail},
		projectMembershipReadsSessionsStub{},
	)
	service := serviceValue.(*projectMembershipReadsService)
	capture := &ProjectMembershipReadsResponseCapture{}
	ctx := WithProjectMembershipReadsResponseCapture(context.WithValue(
		context.Background(),
		projectMembershipReadsPrincipal{},
		uuid.MustParse("00000000-0000-4000-8000-000000000010"),
	), capture)
	result, err := service.ProjectMember(ctx, &api.ProjectMemberPayload{
		Slug:      "membership",
		ProjectID: "00000000-0000-4000-8000-000000000003",
		Pk:        "00000000-0000-4000-8000-000000000004",
	})
	if err != nil {
		t.Fatal(err)
	}
	full, ok := result.AsProjectMembershipFullDirectory()
	if !ok || full == nil || full.Member.DisplayName != "Full Member" {
		t.Fatalf("result=%T %#v", result, result)
	}
	if string(capture.Data) != string(fullDetail) {
		t.Fatalf("captured full-detail JSON changed:\n got %s\nwant %s", capture.Data, fullDetail)
	}
}

func TestWorkspaceProjectMembersDecodesNullMemberAndUnknownRole(t *testing.T) {
	serviceValue, _ := NewProjectMembershipReads(
		&projectMembershipReadsRepositoryStub{workspaceProjectMembers: json.RawMessage(`{
			"00000000-0000-4000-8000-000000000003":[{
				"id":"00000000-0000-4000-8000-000000000004",
				"role":10,
				"member":null,
				"original_role":10,
				"created_at":"2026-09-01T00:00:00Z"
			}]
		}`)},
		projectMembershipReadsSessionsStub{},
	)
	service := serviceValue.(*projectMembershipReadsService)
	ctx := context.WithValue(
		context.Background(),
		projectMembershipReadsPrincipal{},
		uuid.MustParse("00000000-0000-4000-8000-000000000010"),
	)
	result, err := service.WorkspaceProjectMembers(ctx, &api.WorkspaceProjectMembersPayload{Slug: "membership"})
	if err != nil || len(result["00000000-0000-4000-8000-000000000003"]) != 1 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestProjectMemberOpenAPIModelsGuestAndFullResponses(t *testing.T) {
	type schema struct {
		Ref        string            `json:"$ref"`
		Type       string            `json:"type"`
		OneOf      []schema          `json:"oneOf"`
		Properties map[string]schema `json:"properties"`
		Required   []string          `json:"required"`
	}
	var document struct {
		Paths map[string]struct {
			Get struct {
				Responses map[string]struct {
					Content map[string]struct {
						Schema schema `json:"schema"`
					} `json:"content"`
				} `json:"responses"`
			} `json:"get"`
		} `json:"paths"`
		Components struct {
			Schemas map[string]schema `json:"schemas"`
		} `json:"components"`
	}
	data, err := os.ReadFile("gen/http/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	responseSchema := document.Paths["/api/workspaces/{slug}/projects/{project_id}/members/{pk}/"].Get.Responses["200"].Content["application/json"].Schema
	want := map[string]bool{
		"#/components/schemas/ProjectMembershipFullDirectory":  false,
		"#/components/schemas/ProjectMembershipGuestDirectory": false,
	}
	for _, branch := range responseSchema.OneOf {
		if _, exists := want[branch.Ref]; exists {
			want[branch.Ref] = true
		}
	}
	for reference, found := range want {
		if !found {
			t.Errorf("project-member OpenAPI missing %s: %#v", reference, responseSchema)
		}
	}

	fullSchema := document.Components.Schemas["ProjectMembershipFullDirectory"]
	required := make(map[string]bool, len(fullSchema.Required))
	for _, name := range fullSchema.Required {
		required[name] = true
	}
	for _, name := range []string{
		"id", "role", "member", "project", "created_at", "updated_at", "created_by", "updated_by",
		"deleted_at", "comment", "view_props", "default_props", "preferences", "sort_order", "is_active", "workspace",
	} {
		if !required[name] {
			t.Errorf("full project-member OpenAPI does not require %q: %#v", name, fullSchema.Required)
		}
	}
	for _, name := range []string{"view_props", "default_props", "preferences"} {
		property := fullSchema.Properties[name]
		if property.Type != "" || property.Ref != "" || len(property.OneOf) != 0 {
			t.Errorf("full project-member %s is not arbitrary JSON: %#v", name, property)
		}
	}
}
