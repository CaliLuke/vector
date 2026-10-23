package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/project_label_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/projectlabelreads"
	"github.com/CaliLuke/loom/security"
)

const projectLabelFixture = `{"parent":null,"name":"Bug","color":"#FF0000","id":"00000000-0000-4000-8000-000000000001","project_id":"00000000-0000-4000-8000-000000000002","workspace_id":"00000000-0000-4000-8000-000000000003","sort_order":100.0}`

type projectLabelReadsRepositoryStub struct {
	list   json.RawMessage
	detail json.RawMessage
	err    error
}

func (stub *projectLabelReadsRepositoryStub) List(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error) {
	return stub.list, stub.err
}

func (stub *projectLabelReadsRepositoryStub) Detail(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error) {
	return stub.detail, stub.err
}

type projectLabelReadsSessionsStub struct{ err error }

func (stub projectLabelReadsSessionsStub) Validate(context.Context, string) (djangosession.Principal, error) {
	if stub.err != nil {
		return djangosession.Principal{}, stub.err
	}
	return djangosession.Principal{UserID: "00000000-0000-4000-8000-000000000010"}, nil
}

func TestNewProjectLabelReadsRequiresDependencies(t *testing.T) {
	if _, err := NewProjectLabelReads(nil, projectLabelReadsSessionsStub{}); err == nil {
		t.Fatal("missing repository returned nil error")
	}
	if _, err := NewProjectLabelReads(&projectLabelReadsRepositoryStub{}, nil); err == nil {
		t.Fatal("missing sessions returned nil error")
	}
}

func TestProjectLabelReadsListAndDetail(t *testing.T) {
	repository := &projectLabelReadsRepositoryStub{
		list:   json.RawMessage(`[` + projectLabelFixture + `]`),
		detail: json.RawMessage(projectLabelFixture),
	}
	serviceValue, _ := NewProjectLabelReads(repository, projectLabelReadsSessionsStub{})
	service := serviceValue.(*projectLabelReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "session", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	capture := &ProjectLabelReadsResponseCapture{}
	ctx = WithProjectLabelReadsResponseCapture(ctx, capture)
	labels, err := service.List(ctx, &api.ListPayload{Slug: "demo", ProjectID: "00000000-0000-4000-8000-000000000002"})
	if err != nil || len(labels) != 1 || labels[0].Name != "Bug" {
		t.Fatalf("labels=%+v err=%v", labels, err)
	}
	if string(capture.Data) != `[`+projectLabelFixture+`]` {
		t.Fatalf("captured=%s", capture.Data)
	}
	label, err := service.Detail(ctx, &api.DetailPayload{
		Slug: "demo", ProjectID: "00000000-0000-4000-8000-000000000002", Pk: "00000000-0000-4000-8000-000000000001",
	})
	if err != nil || label.Name != "Bug" {
		t.Fatalf("label=%+v err=%v", label, err)
	}
}

func TestProjectLabelReadsMapsRepositoryErrors(t *testing.T) {
	ctx := context.WithValue(context.Background(), projectLabelReadsPrincipal{}, uuid.New())
	service := &projectLabelReadsService{repository: &projectLabelReadsRepositoryStub{err: store.ErrForbidden}}
	_, err := service.List(ctx, &api.ListPayload{ProjectID: uuid.New().String()})
	var forbidden *api.ProjectLabelReadsForbidden
	if !errors.As(err, &forbidden) {
		t.Fatalf("err=%T %v", err, err)
	}
	service.repository = &projectLabelReadsRepositoryStub{err: store.ErrNotFound}
	_, err = service.Detail(ctx, &api.DetailPayload{ProjectID: uuid.New().String(), Pk: uuid.New().String()})
	var notFound *api.ProjectLabelReadsNotFound
	if !errors.As(err, &notFound) || notFound.Detail != "No Label matches the given query." {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestProjectLabelReadsRejectInvalidSession(t *testing.T) {
	service := &projectLabelReadsService{
		repository: &projectLabelReadsRepositoryStub{},
		sessions:   projectLabelReadsSessionsStub{err: errors.New("invalid")},
	}
	_, err := service.APIKeyAuth(context.Background(), "bad", &security.APIKeyScheme{})
	var unauthenticated *api.ProjectLabelReadsUnauthenticated
	if !errors.As(err, &unauthenticated) {
		t.Fatalf("err=%T %v", err, err)
	}
}
