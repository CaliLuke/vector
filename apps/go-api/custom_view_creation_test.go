package systemapi

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	api "github.com/CaliLuke/go-plane/customviewcreation/gen/custom_view_creation"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/customviewcreation"
)

type customViewCreationSessionsStub struct{}

func (customViewCreationSessionsStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{}, errors.New("unexpected session lookup")
}

type customViewCreationRepositoryStub struct {
	calls     []string
	targetErr error
	changes   customviewcreation.ProjectViewChanges
}

func (r *customViewCreationRepositoryStub) ProjectTarget(context.Context, string, string, string) error {
	r.calls = append(r.calls, "project-target")
	return r.targetErr
}
func (r *customViewCreationRepositoryStub) WorkspaceTarget(context.Context, string, string) error {
	r.calls = append(r.calls, "workspace-target")
	return r.targetErr
}
func (r *customViewCreationRepositoryStub) UpdateWorkspace(context.Context, string, string, any) error {
	r.calls = append(r.calls, "workspace-update")
	return nil
}
func (r *customViewCreationRepositoryStub) UpdateProject(_ context.Context, _, _, _ string, changes customviewcreation.ProjectViewChanges) error {
	r.calls = append(r.calls, "project-update")
	r.changes = changes
	return nil
}

func customViewContext(timezone string) context.Context {
	return context.WithValue(context.Background(), customViewCreationPrincipalKey{}, customViewCreationPrincipal{userID: "00000000-0000-4000-8000-000000000001", timezone: timezone})
}
func TestCustomViewCreationActivatesTimezoneBeforeHandlerLookup(t *testing.T) {
	repository := &customViewCreationRepositoryStub{}
	serviceValue, err := NewCustomViewCreation(repository, customViewCreationSessionsStub{})
	if err != nil {
		t.Fatal(err)
	}
	err = serviceValue.(*customViewCreationService).CreateWorkspaceView(customViewContext("Not/A_Timezone"), &api.CreateWorkspaceViewPayload{Slug: "demo"}, nil)
	var failure *api.CustomViewCreationErrorResponse
	if !errors.As(err, &failure) || failure.Name != "status_500" {
		t.Fatalf("failure = %#v", err)
	}
	if len(repository.calls) != 0 {
		t.Fatalf("repository calls before timezone activation: %v", repository.calls)
	}
}
func TestCustomViewCreationLooksUpMemberBeforeParsingRequest(t *testing.T) {
	repository := &customViewCreationRepositoryStub{targetErr: customviewcreation.ErrNotFound}
	serviceValue, _ := NewCustomViewCreation(repository, customViewCreationSessionsStub{})
	body := io.NopCloser(strings.NewReader("{"))
	err := serviceValue.(*customViewCreationService).CreateWorkspaceView(customViewContext("UTC"), &api.CreateWorkspaceViewPayload{Slug: "demo"}, body)
	var failure *api.CustomViewCreationErrorResponse
	if !errors.As(err, &failure) || failure.Name != "status_404" {
		t.Fatalf("failure = %#v", err)
	}
	if len(repository.calls) != 1 || repository.calls[0] != "workspace-target" {
		t.Fatalf("lookup/parse order = %v", repository.calls)
	}
}
func TestCustomViewCreationPreservesMissingProjectFields(t *testing.T) {
	repository := &customViewCreationRepositoryStub{}
	serviceValue, _ := NewCustomViewCreation(repository, customViewCreationSessionsStub{})
	err := serviceValue.(*customViewCreationService).CreateProjectView(customViewContext("UTC"), &api.CreateProjectViewPayload{Slug: "demo", ProjectID: "00000000-0000-4000-8000-000000000100"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(repository.calls) != 2 || repository.calls[0] != "project-target" || repository.calls[1] != "project-update" {
		t.Fatalf("calls = %v", repository.calls)
	}
	if repository.changes.HasViewProps || repository.changes.HasPreferences || repository.changes.HasSortOrder {
		t.Fatalf("missing request fields were overwritten: %#v", repository.changes)
	}
}
