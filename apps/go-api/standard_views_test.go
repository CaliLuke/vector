package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/standard_views"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/standardviews"
)

type standardViewsRepositoryStub struct {
	allowed          bool
	body             json.RawMessage
	err              error
	recentVisitCount int
}

func (s *standardViewsRepositoryStub) AuthorizedWorkspace(context.Context, uuid.UUID, string) (bool, error) {
	return s.allowed, nil
}
func (s *standardViewsRepositoryStub) AuthorizedProject(context.Context, uuid.UUID, string, uuid.UUID) (bool, error) {
	return s.allowed, nil
}
func (s *standardViewsRepositoryStub) ListProject(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error) {
	return s.body, s.err
}
func (s *standardViewsRepositoryStub) GetProject(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (json.RawMessage, error) {
	return s.body, s.err
}
func (s *standardViewsRepositoryStub) CreateProject(context.Context, uuid.UUID, string, uuid.UUID, store.ViewInput) (json.RawMessage, error) {
	return s.body, s.err
}
func (s *standardViewsRepositoryStub) UpdateProject(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, store.ViewInput) (json.RawMessage, error) {
	return s.body, s.err
}
func (s *standardViewsRepositoryStub) PatchProject(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID, store.ViewInput) (json.RawMessage, error) {
	return s.body, s.err
}
func (s *standardViewsRepositoryStub) DeleteProject(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) error {
	return s.err
}
func (s *standardViewsRepositoryStub) ListWorkspace(context.Context, uuid.UUID, string) (json.RawMessage, error) {
	return s.body, s.err
}
func (s *standardViewsRepositoryStub) GetWorkspace(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error) {
	return s.body, s.err
}
func (s *standardViewsRepositoryStub) CreateWorkspace(context.Context, uuid.UUID, string, store.ViewInput) (json.RawMessage, error) {
	return s.body, s.err
}
func (s *standardViewsRepositoryStub) UpdateWorkspace(context.Context, uuid.UUID, string, uuid.UUID, store.ViewInput) (json.RawMessage, error) {
	return s.body, s.err
}
func (s *standardViewsRepositoryStub) PatchWorkspace(context.Context, uuid.UUID, string, uuid.UUID, store.ViewInput) (json.RawMessage, error) {
	return s.body, s.err
}
func (s *standardViewsRepositoryStub) DeleteWorkspace(context.Context, uuid.UUID, string, uuid.UUID) error {
	return s.err
}

func (s *standardViewsRepositoryStub) RecordRecentVisit(context.Context, uuid.UUID, string, *uuid.UUID, uuid.UUID) error {
	s.recentVisitCount++
	return nil
}

type standardViewsSessionStub struct{ err error }

func (s standardViewsSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{UserID: "00000000-0000-4000-8000-000000000010"}, s.err
}

func standardViewsContext(t *testing.T, repository *standardViewsRepositoryStub) (api.Service, context.Context) {
	t.Helper()
	service, err := NewStandardViews(repository, standardViewsSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	return service, ctx
}

func standardViewFixtureBody(t *testing.T, name string) json.RawMessage {
	t.Helper()
	data, err := os.ReadFile("internal/parity/testdata/fixtures/standard_view/" + name)
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Response struct {
			Body json.RawMessage `json:"body"`
		} `json:"response"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture.Response.Body
}

func TestStandardViewsDecodesCommittedProjectFixture(t *testing.T) {
	body := standardViewFixtureBody(t, "project-list.json")
	service, ctx := standardViewsContext(t, &standardViewsRepositoryStub{allowed: true, body: body})
	views, err := service.ListProject(ctx, &api.ListProjectPayload{
		Slug: "standard-view-parity", ProjectID: "10000000-0000-4000-8000-000000000001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(views) == 0 || !views[0].IsFavorite.Present() {
		t.Fatalf("ListProject() did not preserve project is_favorite: %#v", views)
	}
}

type standardViewsRecentVisitCapture struct {
	eventID, slug, entity, entityID, userID, projectID string
}

func (capture *standardViewsRecentVisitCapture) AdmitRecentVisit(_ context.Context, eventID, slug, entity, entityID, userID, projectID string) error {
	capture.eventID, capture.slug, capture.entity, capture.entityID, capture.userID, capture.projectID = eventID, slug, entity, entityID, userID, projectID
	return nil
}

func TestStandardViewsRetrieveAdmitsNativeRecentVisit(t *testing.T) {
	repository := &standardViewsRepositoryStub{allowed: true, body: standardViewFixtureBody(t, "project-retrieve.json")}
	recentVisits := &standardViewsRecentVisitCapture{}
	service, err := NewStandardViews(repository, standardViewsSessionStub{}, recentVisits)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	view, err := service.RetrieveProject(ctx, &api.RetrieveProjectPayload{
		Slug: "standard-view-parity", ProjectID: "10000000-0000-4000-8000-000000000001", Pk: "10000000-0000-4000-8000-000000000002",
	})
	if err != nil {
		t.Fatal(err)
	}
	if view == nil || recentVisits.eventID == "" || recentVisits.slug != "standard-view-parity" || recentVisits.entity != "view" || recentVisits.entityID != "10000000-0000-4000-8000-000000000002" || recentVisits.userID != "00000000-0000-4000-8000-000000000010" || recentVisits.projectID != "10000000-0000-4000-8000-000000000001" {
		t.Fatalf("native recent visit = %+v, view = %#v", recentVisits, view)
	}
	if repository.recentVisitCount != 0 {
		t.Fatalf("legacy SQL visit path ran %d times with native admission configured", repository.recentVisitCount)
	}
}

func TestStandardViewsMapsPatchAndDeleteOwnershipErrors(t *testing.T) {
	repository := &standardViewsRepositoryStub{allowed: true, err: store.ErrLocked}
	service, ctx := standardViewsContext(t, repository)
	_, err := service.PatchWorkspace(ctx, &api.PatchWorkspacePayload{
		Slug: "demo", Pk: "00000000-0000-4000-8000-000000000020",
	})
	var invalid *api.StandardViewInvalid
	if !errors.As(err, &invalid) || invalid.Message != "view is locked" {
		t.Fatalf("PatchWorkspace() error = %#v", err)
	}

	repository.err = store.ErrForbidden
	err = service.DeleteProject(ctx, &api.DeleteProjectPayload{
		Slug: "demo", ProjectID: "00000000-0000-4000-8000-000000000021", Pk: "00000000-0000-4000-8000-000000000022",
	})
	if !errors.As(err, &invalid) || invalid.Message != "Only admin or owner can delete the view" {
		t.Fatalf("DeleteProject() error = %#v", err)
	}
}

func (s *standardViewsRepositoryStub) CheckMutation(context.Context, uuid.UUID, string, *uuid.UUID, uuid.UUID, bool) error {
	return s.err
}
