package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/workspace_recent_visits"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workspacerecentvisits"
)

const recentVisitTestUser = "00000000-0000-4000-8000-000000000010"
const recentVisitTestBody = `[{"id":"00000000-0000-4000-8000-000000000011","entity_name":"issue","entity_identifier":null,"entity_data":null,"visited_at":"2026-08-26T00:00:00Z"}]`

type recentVisitRepoStub struct {
	authorized bool
	err        error
	entityName *string
}

func (r *recentVisitRepoStub) Authorized(context.Context, uuid.UUID, string) (bool, error) {
	return r.authorized, r.err
}

func (r *recentVisitRepoStub) List(_ context.Context, _ uuid.UUID, _ string, entityName *string) (json.RawMessage, error) {
	r.entityName = entityName
	return json.RawMessage(recentVisitTestBody), r.err
}

type recentVisitSessionStub struct{ err error }

func (s recentVisitSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{UserID: recentVisitTestUser}, s.err
}

func recentVisitService(t *testing.T, repository *recentVisitRepoStub) (api.Service, context.Context) {
	t.Helper()
	service, err := NewWorkspaceRecentVisits(repository, recentVisitSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	return service, ctx
}

func TestWorkspaceRecentVisitsListAndEntityFilter(t *testing.T) {
	repository := &recentVisitRepoStub{authorized: true}
	service, ctx := recentVisitService(t, repository)
	entityName := "issue"
	visits, err := service.List(ctx, &api.ListPayload{Slug: "demo", EntityName: &entityName})
	if err != nil || len(visits) != 1 || repository.entityName == nil || *repository.entityName != entityName {
		t.Fatalf("List() = %#v, entityName=%v, err=%v", visits, repository.entityName, err)
	}
	if visits[0].EntityData.IsNull() == false {
		t.Fatal("expected null entity_data")
	}
}

func TestWorkspaceRecentVisitsAuthorizationAndFailures(t *testing.T) {
	repository := &recentVisitRepoStub{}
	service, ctx := recentVisitService(t, repository)
	if _, err := service.List(ctx, &api.ListPayload{Slug: "demo"}); err == nil {
		t.Fatal("expected forbidden")
	} else if forbidden, ok := err.(*api.WorkspaceRecentVisitForbidden); !ok || forbidden.Message != "You don't have the required permissions." {
		t.Fatalf("forbidden=%T %+v", err, err)
	}
	repository.authorized = true
	repository.err = errors.New("database unavailable")
	if _, err := service.List(ctx, &api.ListPayload{Slug: "demo"}); err == nil {
		t.Fatal("expected internal error")
	} else if _, ok := err.(*api.WorkspaceRecentVisitInternal); !ok {
		t.Fatalf("error=%T", err)
	}
	invalidSession, err := NewWorkspaceRecentVisits(repository, recentVisitSessionStub{err: errors.New("invalid")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := invalidSession.(api.Authorizer).APIKeyAuth(context.Background(), "bad", nil); err == nil {
		t.Fatal("expected authentication error")
	}
}

var _ workspaceRecentVisitsRepository = (*store.Repository)(nil)
