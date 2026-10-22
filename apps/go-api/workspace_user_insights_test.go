package systemapi

import (
	"context"
	"encoding/json"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	store "github.com/CaliLuke/go-plane/internal/workspaceuserinsights"
	api "github.com/CaliLuke/go-plane/workspaceuserinsights/gen/workspace_user_insights"
	"github.com/CaliLuke/loom/security"
)

func TestWorkspaceActivityCSVMatchesPythonDialect(t *testing.T) {
	rows := []store.ActivityCSVRow{{Actor: `Zoë "A"`, IssueID: "P-1", Project: "=SUM(A1)", CreatedAt: time.Date(2026, 10, 22, 1, 0, 0, 120000000, time.UTC), Verb: "updated", Field: "description", OldValue: "line one\nline two", NewValue: "@formula"}}
	got, err := encodeWorkspaceActivityCSV(rows)
	if err != nil {
		t.Fatal(err)
	}
	// Oracle generated with Python csv.writer(quoting=csv.QUOTE_ALL), after
	// applying plane.utils.csv_utils.sanitize_csv_row to the data row.
	want := "\"Actor name\",\"Issue ID\",\"Project\",\"Created at\",\"Updated at\",\"Action\",\"Field\",\"Old value\",\"New value\"\r\n\"Zoë \"\"A\"\"\",\"P-1\",\"'=SUM(A1)\",\"2026-10-22 01:00:00.120000+00:00\",\"\",\"updated\",\"description\",\"line one\nline two\",\"'@formula\"\r\n"
	if got != want {
		t.Fatalf("CSV = %q, want %q", got, want)
	}
}

type workspaceUserInsightsSessionStub struct{ principal djangosession.Principal }

func (s workspaceUserInsightsSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return s.principal, nil
}

type workspaceUserInsightsRepositoryStub struct {
	authorizeErr error
	allowExport  bool
	activityErr  error
	calls        []string
	projects     []string
	query        djangoapi.Query
}

func (r *workspaceUserInsightsRepositoryStub) AuthorizeWorkspace(_ context.Context, _, _ string, write bool) error {
	if write {
		r.calls = append(r.calls, "authorize:write")
	} else {
		r.calls = append(r.calls, "authorize:safe")
	}
	return r.authorizeErr
}
func (*workspaceUserInsightsRepositoryStub) Profile(context.Context, string, string, string) (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}
func (r *workspaceUserInsightsRepositoryStub) Activity(_ context.Context, _, _, _ string, projects []string, query djangoapi.Query) (json.RawMessage, error) {
	r.projects, r.query = projects, query
	r.calls = append(r.calls, "activity")
	return json.RawMessage(`{}`), r.activityErr
}
func (*workspaceUserInsightsRepositoryStub) Stats(context.Context, string, string, string, map[string]string) (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}
func (*workspaceUserInsightsRepositoryStub) ExportActivity(context.Context, string, string, string, string) ([]store.ActivityCSVRow, error) {
	return nil, nil
}
func (r *workspaceUserInsightsRepositoryStub) CanExportActivity(context.Context, string, string) (bool, error) {
	r.calls = append(r.calls, "authorize:write")
	return r.allowExport, nil
}

func newWorkspaceUserInsightsTestService(t *testing.T, repo *workspaceUserInsightsRepositoryStub, timezone string) *workspaceUserInsightsService {
	t.Helper()
	service, err := NewWorkspaceUserInsights(repo, workspaceUserInsightsSessionStub{principal: djangosession.Principal{
		UserID: "00000000-0000-4000-8000-000000000001", UserTimezone: timezone,
	}})
	if err != nil {
		t.Fatal(err)
	}
	return service.(*workspaceUserInsightsService)
}

func TestWorkspaceActivityPermissionPrecedesTimezoneForProtocolMethods(t *testing.T) {
	for _, method := range []string{"GET", "HEAD", "OPTIONS", "PATCH", "PUT"} {
		t.Run(method, func(t *testing.T) {
			repo := &workspaceUserInsightsRepositoryStub{authorizeErr: store.ErrForbidden}
			service := newWorkspaceUserInsightsTestService(t, repo, "Invalid/Zone")
			status, body := service.AuthorizeSessionResourceProtocol(context.Background(), "session", SessionProtocolResource{
				Method: method, Pattern: "/api/workspaces/{slug}/user-activity/{user_id}/", Slug: "demo",
			})
			if status != 403 || body["detail"] != "You do not have permission to perform this action." {
				t.Fatalf("protocol authorization = (%d, %#v), want source permission denial", status, body)
			}
			want := "authorize:safe"
			if method == "PATCH" || method == "PUT" {
				want = "authorize:write"
			}
			if !reflect.DeepEqual(repo.calls, []string{want}) {
				t.Fatalf("permission calls = %v, want [%s] before timezone validation", repo.calls, want)
			}
		})
	}
}

func TestWorkspaceActivityResourceProtocolValidatesTimezoneAfterPermission(t *testing.T) {
	repo := &workspaceUserInsightsRepositoryStub{}
	service := newWorkspaceUserInsightsTestService(t, repo, "Invalid/Zone")
	status, body := service.AuthorizeSessionResourceProtocol(context.Background(), "session", SessionProtocolResource{
		Method: "GET", Pattern: "/api/workspaces/{slug}/user-activity/{user_id}/", Slug: "demo",
	})
	if status != 500 || body["error"] != "Something went wrong please try again later" {
		t.Fatalf("protocol authorization = (%d, %#v), want timezone failure after permission", status, body)
	}
	if !reflect.DeepEqual(repo.calls, []string{"authorize:safe"}) {
		t.Fatalf("permission calls = %v, want permission before timezone", repo.calls)
	}
}

func TestWorkspaceActivityUsesRepeatedProjectQueryValues(t *testing.T) {
	repo := &workspaceUserInsightsRepositoryStub{}
	service := newWorkspaceUserInsightsTestService(t, repo, "UTC")
	ctx, err := service.APIKeyAuth(context.Background(), "session", &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	if err != nil {
		t.Fatal(err)
	}
	ctx = WithReadServiceQueryPresence(ctx, url.Values{"project": {"00000000-0000-4000-8000-000000000101", "00000000-0000-4000-8000-000000000102"}})
	combined := "00000000-0000-4000-8000-000000000101,00000000-0000-4000-8000-000000000102"
	_, body, err := service.Activity(ctx, &api.ActivityPayload{Slug: "demo", UserID: "00000000-0000-4000-8000-000000000002", Project: &combined})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	if !reflect.DeepEqual(repo.projects, []string{"00000000-0000-4000-8000-000000000101", "00000000-0000-4000-8000-000000000102"}) {
		t.Fatalf("project filter = %#v, want repeated getlist values", repo.projects)
	}
	if !reflect.DeepEqual(repo.calls, []string{"authorize:safe", "activity"}) {
		t.Fatalf("repository calls = %v, want permission before activity", repo.calls)
	}
}

func TestWorkspaceUserInsightsActivityPaginationZeroPreservesApproved400(t *testing.T) {
	repo := &workspaceUserInsightsRepositoryStub{activityErr: &djangoapi.HTTPError{
		Status: 400, Body: map[string]any{"detail": "Error in parsing"},
	}}
	service := newWorkspaceUserInsightsTestService(t, repo, "UTC")
	ctx, err := service.APIKeyAuth(context.Background(), "session", &security.APIKeyScheme{Name: "PlaneSessionCookie"})
	if err != nil {
		t.Fatal(err)
	}
	perPage := "0"
	_, _, err = service.Activity(ctx, &api.ActivityPayload{Slug: "demo", UserID: "00000000-0000-4000-8000-000000000002", PerPage: &perPage})
	status, body := workspaceUserInsightsProtocolError(err)
	if status != 400 || body["detail"] != "Error in parsing" {
		t.Fatalf("pagination response = (%d, %#v), want approved HTTP 400 parsing detail", status, body)
	}
}
