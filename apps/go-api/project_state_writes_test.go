package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"uuid"

	statessvr "github.com/CaliLuke/go-plane/gen/http/project_state_reads/server"
	api "github.com/CaliLuke/go-plane/gen/project_state_reads"
	store "github.com/CaliLuke/go-plane/internal/projectstatereads"
	loomhttp "github.com/CaliLuke/loom/http"
)

type stateWriteBoundaryRepository struct {
	projectStateReadsRepositoryStub
	allowed        bool
	authorizations int
	writes         []store.StateWriteInput
	failure        error
	lookupFailure  error
	lookups        int
}

func (r *stateWriteBoundaryRepository) InvalidateAppCache(context.Context, string) {}

func (r *stateWriteBoundaryRepository) ResolveForUpdate(context.Context, store.StateWriteInput) error {
	r.lookups++
	return r.lookupFailure
}

func (r *stateWriteBoundaryRepository) AuthorizeApp(context.Context, string, string, string, string) (bool, error) {
	r.authorizations++
	return r.allowed, nil
}

func (r *stateWriteBoundaryRepository) Create(_ context.Context, input store.StateWriteInput) (json.RawMessage, error) {
	r.writes = append(r.writes, input)
	return json.RawMessage(projectStateFixture), r.failure
}

func (r *stateWriteBoundaryRepository) Update(ctx context.Context, input store.StateWriteInput) (json.RawMessage, error) {
	return r.Create(ctx, input)
}

func (r *stateWriteBoundaryRepository) Delete(_ context.Context, input store.StateWriteInput) error {
	r.writes = append(r.writes, input)
	return r.failure
}

func (r *stateWriteBoundaryRepository) MarkDefault(ctx context.Context, input store.StateWriteInput) error {
	return r.Delete(ctx, input)
}

func TestProjectStateWritesAuthorizeBeforeParsingOrMutation(t *testing.T) {
	for _, authenticated := range []bool{false, true} {
		t.Run(map[bool]string{false: "anonymous", true: "non-member"}[authenticated], func(t *testing.T) {
			repository := &stateWriteBoundaryRepository{}
			service := &projectStateReadsService{repository: repository}
			ctx := context.Background()
			if authenticated {
				ctx = context.WithValue(ctx, projectStateReadsPrincipal{}, uuid.New())
			}
			contentType := "application/json"
			_, _, err := service.Create(ctx, &api.CreatePayload{Slug: "demo", ProjectID: uuid.New().String(), ContentType: &contentType}, io.NopCloser(strings.NewReader("{broken")))
			var response *api.ProjectStateWritesAppError
			if !errors.As(err, &response) {
				t.Fatalf("expected permission response, got %T: %v", err, err)
			}
			wantKind, wantAuthorizations := "unauthenticated", 0
			if authenticated {
				wantKind, wantAuthorizations = "permission_denied", 1
			}
			if response.Kind != wantKind || repository.authorizations != wantAuthorizations || len(repository.writes) != 0 {
				t.Fatalf("kind=%s authorization calls=%d writes=%d", response.Kind, repository.authorizations, len(repository.writes))
			}
		})
	}
}

func TestProjectStateWritesMarkDefaultHasNoRequestBody(t *testing.T) {
	repository := &stateWriteBoundaryRepository{allowed: true}
	service := &projectStateReadsService{repository: repository}
	ctx := context.WithValue(context.Background(), projectStateReadsPrincipal{}, uuid.New())
	project, state := uuid.New().String(), uuid.New().String()
	if err := service.MarkDefault(ctx, &api.MarkDefaultPayload{Slug: "demo", ProjectID: project, Pk: state}); err != nil {
		t.Fatal(err)
	}
	if len(repository.writes) != 1 || repository.writes[0].StateID != state || repository.writes[0].ProjectID != project || !repository.writes[0].App {
		t.Fatalf("target mismatch: %+v", repository.writes)
	}
}

func TestProjectStateWritesRejectInvalidUTF8BeforePersistence(t *testing.T) {
	repository := &stateWriteBoundaryRepository{allowed: true}
	service := &projectStateReadsService{repository: repository}
	ctx := context.WithValue(context.Background(), projectStateReadsPrincipal{}, uuid.New())
	contentType := "application/json"
	_, _, err := service.Create(ctx, &api.CreatePayload{Slug: "demo", ProjectID: uuid.New().String(), ContentType: &contentType}, io.NopCloser(strings.NewReader("{\"name\":\"\xff\",\"color\":\"#000000\"}")))
	var response *api.ProjectStateWritesAppError
	if !errors.As(err, &response) || response.Kind != "bad_request" || len(repository.writes) != 0 {
		t.Fatalf("invalid UTF-8 err=%v, writes=%d", err, len(repository.writes))
	}
}

func TestProjectStateWritesMissingTargetPrecedesMalformedBody(t *testing.T) {
	repository := &stateWriteBoundaryRepository{allowed: true, lookupFailure: &store.WriteFailure{Status: 404, Body: map[string]string{"detail": "No State matches the given query."}}}
	service := &projectStateReadsService{repository: repository}
	ctx := context.WithValue(context.Background(), projectStateReadsPrincipal{}, uuid.New())
	contentType := "application/json"
	_, _, err := service.Update(ctx, &api.UpdatePayload{Slug: "demo", ProjectID: uuid.New().String(), Pk: uuid.New().String(), ContentType: &contentType}, io.NopCloser(strings.NewReader("{broken")))
	var response *api.ProjectStateWritesAppError
	if !errors.As(err, &response) || response.Kind != "not_found" || repository.lookups != 1 || len(repository.writes) != 0 {
		t.Fatalf("missing target err=%v, lookups=%d, writes=%d", err, repository.lookups, len(repository.writes))
	}
}

func TestProjectStateWritesPreserveSerializerErrorBody(t *testing.T) {
	want := map[string]any{"name": []string{"This field may not be blank."}, "non_field_errors": []string{"Invalid state."}}
	repository := &stateWriteBoundaryRepository{allowed: true, failure: &store.WriteFailure{Status: 400, Body: want}}
	service := &projectStateReadsService{repository: repository}
	ctx := context.WithValue(context.Background(), projectStateReadsPrincipal{}, uuid.New())
	contentType := "application/json"
	_, _, err := service.Create(ctx, &api.CreatePayload{Slug: "demo", ProjectID: uuid.New().String(), ContentType: &contentType}, io.NopCloser(strings.NewReader(`{"name":"New state","color":"#000000"}`)))
	var response *api.ProjectStateWritesAppError
	if !errors.As(err, &response) || response.Kind != "bad_request" {
		t.Fatalf("error=%T %v", err, err)
	}
	raw, err := json.Marshal(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	expected, _ := json.Marshal(want)
	if string(raw) != string(expected) {
		t.Fatalf("serializer response=%s, want=%s", raw, expected)
	}
}

func TestProjectStateWritesGeneratedBodylessTransport(t *testing.T) {
	for _, operation := range []struct{ name, method, suffix string }{
		{"delete", http.MethodDelete, "/"},
		{"mark default", http.MethodPost, "/mark-default/"},
	} {
		t.Run(operation.name, func(t *testing.T) {
			repository := &stateWriteBoundaryRepository{allowed: true}
			service, err := NewProjectStateReads(repository, projectStateReadsSessionsStub{})
			if err != nil {
				t.Fatal(err)
			}
			mux := loomhttp.NewMuxer()
			server := statessvr.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
			statessvr.Mount(mux, server)
			project, state := uuid.New().String(), uuid.New().String()
			request := httptest.NewRequest(operation.method, "/api/workspaces/demo/projects/"+project+"/states/"+state+operation.suffix, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusNoContent || len(repository.writes) != 1 {
				t.Fatalf("status=%d body=%q writes=%d", response.Code, response.Body.String(), len(repository.writes))
			}
			if repository.writes[0].StateID != state || repository.writes[0].ProjectID != project {
				t.Fatalf("target mismatch: %+v", repository.writes)
			}
		})
	}
}
