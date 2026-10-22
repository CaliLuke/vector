package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	api "github.com/CaliLuke/go-plane/gen/estimate_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/estimatereads"
	"github.com/CaliLuke/loom/security"
	"uuid"
)

const estimateReadFixture = `{"id":"00000000-0000-4000-8000-000000000011","created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-02T03:04:05Z","created_by":null,"updated_by":null,"deleted_at":null,"workspace":"00000000-0000-4000-8000-000000000002","project":"00000000-0000-4000-8000-000000000003","name":"Story points","description":"","type":"points","last_used":false,"points":[{"id":"00000000-0000-4000-8000-000000000012","created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-02T03:04:05Z","created_by":null,"updated_by":null,"deleted_at":null,"estimate":"00000000-0000-4000-8000-000000000011","workspace":"00000000-0000-4000-8000-000000000002","project":"00000000-0000-4000-8000-000000000003","key":1,"value":"1","description":""}]}`

type estimateReadsRepositoryStub struct {
	workspaceData json.RawMessage
	projectData   json.RawMessage
	detailData    json.RawMessage
	err           error
	calls         []estimateReadsRepositoryCall
}

type estimateReadsRepositoryCall struct {
	method     string
	userID     uuid.UUID
	slug       string
	projectID  uuid.UUID
	estimateID uuid.UUID
}

func (stub *estimateReadsRepositoryStub) AuthorizeWorkspace(_ context.Context, userID uuid.UUID, slug string) error {
	stub.calls = append(stub.calls, estimateReadsRepositoryCall{method: "authorize_workspace", userID: userID, slug: slug})
	return stub.err
}

func (stub *estimateReadsRepositoryStub) WorkspaceSelected(_ context.Context, userID uuid.UUID, slug string) (json.RawMessage, error) {
	stub.calls = append(stub.calls, estimateReadsRepositoryCall{method: "workspace", userID: userID, slug: slug})
	return stub.workspaceData, stub.err
}

func (stub *estimateReadsRepositoryStub) ProjectList(_ context.Context, userID uuid.UUID, slug string, projectID uuid.UUID) (json.RawMessage, error) {
	stub.calls = append(stub.calls, estimateReadsRepositoryCall{method: "project_list", userID: userID, slug: slug, projectID: projectID})
	return stub.projectData, stub.err
}

func (stub *estimateReadsRepositoryStub) ProjectDetail(_ context.Context, userID uuid.UUID, slug string, projectID, estimateID uuid.UUID) (json.RawMessage, error) {
	stub.calls = append(stub.calls, estimateReadsRepositoryCall{method: "project_detail", userID: userID, slug: slug, projectID: projectID, estimateID: estimateID})
	return stub.detailData, stub.err
}

type estimateReadsSessionsStub struct {
	principal djangosession.Principal
	err       error
}

func (stub estimateReadsSessionsStub) Validate(context.Context, string) (djangosession.Principal, error) {
	if stub.err != nil {
		return djangosession.Principal{}, stub.err
	}
	return stub.principal, nil
}

func TestNewEstimateReadsRequiresDependencies(t *testing.T) {
	sessions := estimateReadsSessionsStub{principal: djangosession.Principal{UserID: uuid.New().String()}}
	if _, err := NewEstimateReads(nil, sessions); err == nil {
		t.Fatal("missing repository returned nil error")
	}
	if _, err := NewEstimateReads(&estimateReadsRepositoryStub{}, nil); err == nil {
		t.Fatal("missing sessions returned nil error")
	}
}

func TestEstimateReadsUsesAuthenticatedUserAndExactTenantPaths(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	projectID := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	estimateID := uuid.MustParse("00000000-0000-4000-8000-000000000011")
	repository := &estimateReadsRepositoryStub{
		workspaceData: json.RawMessage(`[` + estimateReadFixture + `]`),
		projectData:   json.RawMessage(`[` + estimateReadFixture + `]`),
		detailData:    json.RawMessage(estimateReadFixture),
	}
	serviceValue, err := NewEstimateReads(repository, estimateReadsSessionsStub{principal: djangosession.Principal{UserID: userID.String()}})
	if err != nil {
		t.Fatal(err)
	}
	service := serviceValue.(*estimateReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "session", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	search := "ignored"
	workspaceBody := readRawResponse(t, captureRawResponse(service.WorkspaceSelected(ctx, &api.WorkspaceSelectedPayload{Slug: "tenant-one", Search: &search})))
	if workspaceBody != `[`+estimateReadFixture+`]` {
		t.Fatalf("workspace=%s", workspaceBody)
	}
	projectBody := readRawResponse(t, captureRawResponse(service.ProjectList(ctx, &api.ProjectListPayload{Slug: "tenant-two", ProjectID: projectID.String(), Search: &search})))
	if projectBody != `[`+estimateReadFixture+`]` {
		t.Fatalf("project=%s", projectBody)
	}
	detailBody := readRawResponse(t, captureRawResponse(service.ProjectDetail(ctx, &api.ProjectDetailPayload{Slug: "tenant-three", ProjectID: projectID.String(), EstimateID: estimateID.String()})))
	if detailBody != estimateReadFixture {
		t.Fatalf("detail=%s", detailBody)
	}
	if len(repository.calls) != 3 {
		t.Fatalf("calls=%+v", repository.calls)
	}
	wantSlugs := []string{"tenant-one", "tenant-two", "tenant-three"}
	for index, call := range repository.calls {
		if call.userID != userID || call.slug != wantSlugs[index] {
			t.Fatalf("call[%d]=%+v", index, call)
		}
	}
	if repository.calls[1].projectID != projectID || repository.calls[2].projectID != projectID || repository.calls[2].estimateID != estimateID {
		t.Fatalf("project/detail calls=%+v", repository.calls)
	}
}

func TestEstimateReadsWorkspaceOptionsAuthorizeAndMapDatabaseFailure(t *testing.T) {
	ctx := context.WithValue(context.Background(), estimateReadsPrincipal{}, uuid.New())
	repository := &estimateReadsRepositoryStub{}
	service := &estimateReadsService{repository: repository}

	metadata, body, err := service.WorkspaceSelectedOptions(ctx, &api.WorkspaceSelectedOptionsPayload{Slug: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if metadata == nil || metadata.ContentType != "application/json" {
		t.Fatalf("metadata=%#v", metadata)
	}
	gotBody := readRawBody(t, body, nil)
	wantBody := `{"name":"Workspace Estimates Endpoint","description":"","renders":["application/json"],"parses":["application/json","application/x-www-form-urlencoded","multipart/form-data"]}`
	if gotBody != wantBody {
		t.Fatalf("body=%q, want %q", gotBody, wantBody)
	}
	if len(repository.calls) != 1 || repository.calls[0].method != "authorize_workspace" {
		t.Fatalf("authorization calls=%+v", repository.calls)
	}

	repository.err = errors.New("database unavailable")
	_, _, err = service.WorkspaceSelectedOptions(ctx, &api.WorkspaceSelectedOptionsPayload{Slug: "demo"})
	var internal *api.EstimateReadsInternalError
	if !errors.As(err, &internal) || internal.Message != estimateReadsInternalErrorDetail {
		t.Fatalf("database err=%T %v", err, err)
	}
}

func TestEstimateReadsRejectsInvalidSessionBeforeRepositoryAccess(t *testing.T) {
	repository := &estimateReadsRepositoryStub{}
	service := &estimateReadsService{repository: repository, sessions: estimateReadsSessionsStub{err: errors.New("invalid")}}
	_, err := service.APIKeyAuth(context.Background(), "invalid", &security.APIKeyScheme{})
	var unauthenticated *api.EstimateReadsUnauthenticated
	if !errors.As(err, &unauthenticated) || unauthenticated.Detail != unauthenticatedDetail {
		t.Fatalf("err=%T %v", err, err)
	}
	if len(repository.calls) != 0 {
		t.Fatalf("repository calls=%+v", repository.calls)
	}
}

func TestEstimateReadsMapsPermissionAndExactDetailNotFound(t *testing.T) {
	ctx := context.WithValue(context.Background(), estimateReadsPrincipal{}, uuid.New())
	repository := &estimateReadsRepositoryStub{err: store.ErrForbidden}
	service := &estimateReadsService{repository: repository}
	_, _, err := service.ProjectList(ctx, &api.ProjectListPayload{Slug: "demo", ProjectID: uuid.New().String()})
	var forbidden *api.EstimateReadsForbidden
	if !errors.As(err, &forbidden) || forbidden.Detail != "You do not have permission to perform this action." {
		t.Fatalf("err=%T %v", err, err)
	}
	repository.err = store.ErrNotFound
	_, _, err = service.ProjectDetail(ctx, &api.ProjectDetailPayload{Slug: "demo", ProjectID: uuid.New().String(), EstimateID: uuid.New().String()})
	var notFound *api.EstimateReadsNotFound
	if !errors.As(err, &notFound) || notFound.Message != "The required object does not exist." {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestEstimateReadsStreamsArbitraryRepositoryJSON(t *testing.T) {
	ctx := context.WithValue(context.Background(), estimateReadsPrincipal{}, uuid.New())
	service := &estimateReadsService{repository: &estimateReadsRepositoryStub{workspaceData: json.RawMessage(`{`)}}
	body := readRawResponse(t, captureRawResponse(service.WorkspaceSelected(ctx, &api.WorkspaceSelectedPayload{Slug: "demo"})))
	if body != "{" {
		t.Fatalf("body=%q", body)
	}
}
