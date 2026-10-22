package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/projectpublication"
)

type projectPublicationSessionsStub struct{ principal djangosession.Principal }

func (s projectPublicationSessionsStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return s.principal, nil
}

type projectPublicationRepositoryStub struct {
	authorizationErr error
	method           string
}

func (s *projectPublicationRepositoryStub) Authorize(_ context.Context, _ string, method, _, _ string) error {
	s.method = method
	return s.authorizationErr
}
func (*projectPublicationRepositoryStub) Exists(context.Context, string, string, string) error {
	return nil
}
func (*projectPublicationRepositoryStub) Create(context.Context, string, string, string, projectpublication.CreateInput) (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}
func (*projectPublicationRepositoryStub) Update(context.Context, string, string, string, string, projectpublication.Patch) (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}
func (*projectPublicationRepositoryStub) Delete(context.Context, string, string, string, string) error {
	return nil
}

func TestProjectPublicationClassPermissionRunsBeforeTimezoneActivation(t *testing.T) {
	repository := &projectPublicationRepositoryStub{authorizationErr: projectpublication.ErrForbidden}
	serviceValue, err := NewProjectPublication(repository, projectPublicationSessionsStub{principal: djangosession.Principal{UserID: "00000000-0000-4000-8000-000000000001", UserTimezone: "Not/A_Timezone"}})
	if err != nil {
		t.Fatal(err)
	}
	service := serviceValue.(*projectPublicationService)
	status, body := service.AuthorizeSessionResourceProtocol(context.Background(), "session", SessionProtocolResource{Method: "PATCH", Slug: "demo", ProjectID: "00000000-0000-4000-8000-000000000100"})
	if status != 403 || body["detail"] != "You do not have permission to perform this action." {
		t.Fatalf("resource auth = %d %#v", status, body)
	}
	if repository.method != "PATCH" {
		t.Fatalf("permission method = %q", repository.method)
	}
}
func TestProjectPublicationAllowedPermissionActivatesTimezoneBeforeHandler(t *testing.T) {
	repository := &projectPublicationRepositoryStub{}
	serviceValue, _ := NewProjectPublication(repository, projectPublicationSessionsStub{principal: djangosession.Principal{UserID: "00000000-0000-4000-8000-000000000001", UserTimezone: "Not/A_Timezone"}})
	status, _ := serviceValue.(*projectPublicationService).AuthorizeSessionResourceProtocol(context.Background(), "session", SessionProtocolResource{Method: "POST", Slug: "demo", ProjectID: "00000000-0000-4000-8000-000000000100"})
	if status != 400 {
		t.Fatalf("allowed permission with invalid timezone status = %d", status)
	}
}
func TestProjectPublicationProtocolDoesNotPromoteRepositoryErrors(t *testing.T) {
	status, body := projectPublicationProtocolError(errors.New("unexpected"))
	if status != 500 || body["error"] != "Something went wrong please try again later" {
		t.Fatalf("protocol error = %d %#v", status, body)
	}
}
