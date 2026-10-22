package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	api "github.com/CaliLuke/go-plane/gen/project_state_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/projectstatereads"
	"github.com/CaliLuke/loom/security"
	"uuid"
)

const projectStateFixture = `{"id":"00000000-0000-4000-8000-000000000001","project_id":"00000000-0000-4000-8000-000000000002","workspace_id":"00000000-0000-4000-8000-000000000003","name":"Backlog","color":"#60646C","group":"backlog","default":true,"description":"First backlog state","sequence":100.0}`

type projectStateReadsRepositoryStub struct {
	list   json.RawMessage
	detail json.RawMessage
	intake json.RawMessage
	err    error
}

func (stub *projectStateReadsRepositoryStub) AuthorizeProject(context.Context, uuid.UUID, string, uuid.UUID) error {
	return stub.err
}

func (stub *projectStateReadsRepositoryStub) List(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error) {
	return stub.list, stub.err
}

func (stub *projectStateReadsRepositoryStub) Detail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error) {
	return stub.detail, stub.err
}

func (stub *projectStateReadsRepositoryStub) Intake(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error) {
	return stub.intake, stub.err
}

type projectStateReadsSessionsStub struct{ err error }

func (stub projectStateReadsSessionsStub) Validate(context.Context, string) (djangosession.Principal, error) {
	if stub.err != nil {
		return djangosession.Principal{}, stub.err
	}
	return djangosession.Principal{UserID: "00000000-0000-4000-8000-000000000010"}, nil
}

func TestNewProjectStateReadsRequiresDependencies(t *testing.T) {
	if _, err := NewProjectStateReads(nil, projectStateReadsSessionsStub{}); err == nil {
		t.Fatal("missing repository returned nil error")
	}
	if _, err := NewProjectStateReads(&projectStateReadsRepositoryStub{}, nil); err == nil {
		t.Fatal("missing sessions returned nil error")
	}
}

func TestProjectStateReadsAuthenticateListAndGroup(t *testing.T) {
	list := json.RawMessage(`[` + projectStateFixture[:len(projectStateFixture)-1] + `,"order":1.0}]`)
	serviceValue, _ := NewProjectStateReads(&projectStateReadsRepositoryStub{list: list}, projectStateReadsSessionsStub{})
	service := serviceValue.(*projectStateReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "session", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.List(ctx, &api.ListPayload{Slug: "demo", ProjectID: "00000000-0000-4000-8000-000000000002"})
	if err != nil {
		t.Fatal(err)
	}
	var states []*api.ProjectStateRead
	if err := json.Unmarshal(result, &states); err != nil {
		t.Fatal(err)
	}
	if len(states) != 1 || states[0].Name != "Backlog" || states[0].Order == nil || *states[0].Order != 1 {
		t.Fatalf("states=%+v", states)
	}
	groupedValue := "true"
	capture := &ProjectStateReadsResponseCapture{}
	ctx = WithProjectStateReadsResponseCapture(ctx, capture)
	result, err = service.List(ctx, &api.ListPayload{Slug: "demo", ProjectID: "00000000-0000-4000-8000-000000000002", Grouped: &groupedValue})
	if err != nil {
		t.Fatal(err)
	}
	var grouped map[string][]*api.ProjectStateRead
	if err := json.Unmarshal(result, &grouped); err != nil {
		t.Fatal(err)
	}
	if len(grouped["backlog"]) != 1 {
		t.Fatalf("grouped=%+v", grouped)
	}
	if len(capture.Data) == 0 || capture.Data[0] != '{' || string(capture.Data[:11]) != `{"backlog":` {
		t.Fatalf("captured grouped response=%s", capture.Data)
	}
}

func TestProjectStateReadsDetailAndIntakeErrorShapes(t *testing.T) {
	ctx := context.WithValue(context.Background(), projectStateReadsPrincipal{}, uuid.New())
	service := &projectStateReadsService{
		repository: &projectStateReadsRepositoryStub{err: store.ErrForbidden},
		sessions:   projectStateReadsSessionsStub{},
	}
	_, err := service.Detail(ctx, &api.DetailPayload{ProjectID: uuid.New().String(), Pk: uuid.New().String()})
	var detailNotFound *api.ProjectStateReadsDetailNotFound
	if !errors.As(err, &detailNotFound) || detailNotFound.Detail != "No State matches the given query." {
		t.Fatalf("detail err=%T %v", err, err)
	}
	_, err = service.Intake(ctx, &api.IntakePayload{ProjectID: uuid.New().String()})
	var denied *api.ProjectStateReadsPermissionDenied
	if !errors.As(err, &denied) || denied.Message != "You don't have the required permissions." {
		t.Fatalf("intake err=%T %v", err, err)
	}
	service.repository = &projectStateReadsRepositoryStub{err: store.ErrNotFound}
	_, err = service.Intake(ctx, &api.IntakePayload{ProjectID: uuid.New().String()})
	var notFound *api.ProjectStateReadsNotFound
	if !errors.As(err, &notFound) || notFound.Message != "Triage state not found" {
		t.Fatalf("intake err=%T %v", err, err)
	}
}

func TestProjectStateReadsRejectInvalidSession(t *testing.T) {
	service := &projectStateReadsService{
		repository: &projectStateReadsRepositoryStub{},
		sessions:   projectStateReadsSessionsStub{err: errors.New("invalid")},
	}
	_, err := service.APIKeyAuth(context.Background(), "bad", &security.APIKeyScheme{})
	var unauthenticated *api.ProjectStateReadsUnauthenticated
	if !errors.As(err, &unauthenticated) {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestProjectStateReadsMetadataRequiresOnlyAuthentication(t *testing.T) {
	service := &projectStateReadsService{
		repository: &projectStateReadsRepositoryStub{err: store.ErrForbidden},
		sessions:   projectStateReadsSessionsStub{},
	}
	if err := service.AuthorizeMetadata(context.Background(), "session"); err != nil {
		t.Fatalf("authenticated non-member metadata err=%v", err)
	}
	service.sessions = projectStateReadsSessionsStub{err: errors.New("invalid")}
	err := service.AuthorizeMetadata(context.Background(), "bad")
	var unauthenticated *api.ProjectStateReadsUnauthenticated
	if !errors.As(err, &unauthenticated) {
		t.Fatalf("anonymous metadata err=%T %v", err, err)
	}
}
