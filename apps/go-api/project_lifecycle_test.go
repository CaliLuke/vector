package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	storeuuid "uuid"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	store "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	api "github.com/CaliLuke/go-plane/projectlifecycle/gen/project_lifecycle"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type projectLifecycleWriteFake struct {
	projectLifecycleRepository
	authorizeCalls int
	mutationCalls  int
	createCalls    int
	updateCalls    int
}

func (f *projectLifecycleWriteFake) AuthorizeProjectWrite(context.Context, string, string, string, string, bool) error {
	f.authorizeCalls++
	return nil
}

func (f *projectLifecycleWriteFake) CheckProjectMutation(context.Context, string, string, string, bool) error {
	f.mutationCalls++
	return nil
}

func (f *projectLifecycleWriteFake) CreateProject(context.Context, string, string, djangoinput.Input, bool) (json.RawMessage, error) {
	f.createCalls++
	return json.RawMessage(`{}`), nil
}

func (f *projectLifecycleWriteFake) UpdateProject(context.Context, string, string, string, djangoinput.Input, bool, bool) (json.RawMessage, error) {
	f.updateCalls++
	return json.RawMessage(`{}`), nil
}

type projectLifecycleReadFake struct {
	projectLifecycleRepository
	denied  error
	options store.APIProjectListOptions
	calls   int
}

func (f *projectLifecycleReadFake) AuthorizeAPIProjectRead(context.Context, storeuuid.UUID, string, bool) error {
	return f.denied
}
func (f *projectLifecycleReadFake) APIProjects(_ context.Context, _ storeuuid.UUID, _ string, options store.APIProjectListOptions) (json.RawMessage, error) {
	f.calls++
	f.options = options
	return json.RawMessage(`{"results":[{"id":"p"}]}`), nil
}
func TestProjectLifecycleReadChecksPermissionBeforeThrottle(t *testing.T) {
	repo := &projectLifecycleReadFake{denied: store.ErrForbidden}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	s, _ := NewProjectLifecycle(repo, &activitySessionStub{}, &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}}, throttle)
	ctx, err := s.APIKeyAuth(context.Background(), "key", &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = s.GetAPIV1WorkspacesBySlugProjectsLite(ctx, &api.GetAPIV1WorkspacesBySlugProjectsLitePayload{Slug: "space"})
	var failure *api.ProjectLifecycleErrorIdentity
	if !errors.As(err, &failure) || failure.Kind != "status_403" {
		t.Fatalf("error=%v", err)
	}
	if throttle.calls != 0 || repo.calls != 0 {
		t.Fatalf("denied request throttled=%d read=%d", throttle.calls, repo.calls)
	}
}

func TestProjectLifecycleRejectsMalformedUTF8BeforeProjectWrites(t *testing.T) {
	prefix := []byte(`{"name":"Changed","unused":"`)
	tests := []struct {
		name string
		tail []byte
	}{
		{name: "invalid start byte on create", tail: []byte{0xff, '"', '}'}},
		{name: "invalid continuation byte on PATCH", tail: []byte{0xc3, '"', '}'}},
		{name: "truncated UTF-8 at end of JSON string", tail: []byte{0xc3}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &projectLifecycleWriteFake{}
			throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
			service, err := NewProjectLifecycle(
				repo,
				&activitySessionStub{},
				&activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}},
				throttle,
			)
			if err != nil {
				t.Fatal(err)
			}
			ctx, err := service.APIKeyAuth(context.Background(), "key", &security.APIKeyScheme{Name: "ApiKeyAuth"})
			if err != nil {
				t.Fatal(err)
			}
			body := append(append([]byte(nil), prefix...), test.tail...)
			var callErr error
			if test.name == "invalid start byte on create" {
				_, _, callErr = service.create(ctx, "space", io.NopCloser(bytes.NewReader(body)), false)
			} else {
				_, _, callErr = service.update(ctx, "space", "project", io.NopCloser(bytes.NewReader(body)), false, false)
			}
			var failure *api.ProjectLifecycleErrorIdentity
			if !errors.As(callErr, &failure) || failure.Kind != "status_400" {
				t.Fatalf("request error=%v, want HTTP 400", callErr)
			}
			if repo.createCalls != 0 || repo.updateCalls != 0 {
				t.Fatalf("invalid input reached domain writes: create=%d update=%d", repo.createCalls, repo.updateCalls)
			}
		})
	}
}

func TestProjectLifecycleAuthenticationPrecedesMalformedUTF8Parsing(t *testing.T) {
	repo := &projectLifecycleWriteFake{}
	service, err := NewProjectLifecycle(
		repo,
		&activitySessionStub{},
		&activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}},
		&activityThrottleStub{rate: apikey.Rate{Allowed: true}},
	)
	if err != nil {
		t.Fatal(err)
	}
	body := append([]byte(`{"name":"Changed","unused":"`), 0xff, '"', '}')
	_, _, err = service.create(context.Background(), "space", io.NopCloser(bytes.NewReader(body)), false)
	var failure *api.ProjectLifecycleErrorIdentity
	if !errors.As(err, &failure) || failure.Kind != "status_401" {
		t.Fatalf("unauthenticated malformed request error=%v, want HTTP 401", err)
	}
	if repo.authorizeCalls != 0 || repo.mutationCalls != 0 || repo.createCalls != 0 {
		t.Fatalf("unauthenticated request reached authorization/write: auth=%d mutation=%d create=%d", repo.authorizeCalls, repo.mutationCalls, repo.createCalls)
	}
}

func TestProjectLifecycleRejectsIncompleteUTF8AfterCompleteJSONBeforeWrite(t *testing.T) {
	repo := &projectLifecycleWriteFake{}
	service, err := NewProjectLifecycle(
		repo,
		&activitySessionStub{},
		&activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}},
		&activityThrottleStub{rate: apikey.Rate{Allowed: true}},
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.APIKeyAuth(context.Background(), "key", &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err != nil {
		t.Fatal(err)
	}
	body := append([]byte(`{"name":"Changed"}`), 0xc3)
	_, _, err = service.update(ctx, "space", "project", io.NopCloser(bytes.NewReader(body)), false, false)
	var failure *api.ProjectLifecycleErrorIdentity
	if !errors.As(err, &failure) || failure.Kind != "status_400" {
		t.Fatalf("request error=%v, want HTTP 400", err)
	}
	if repo.createCalls != 0 || repo.updateCalls != 0 {
		t.Fatalf("invalid input reached domain writes: create=%d update=%d", repo.createCalls, repo.updateCalls)
	}
}
func TestProjectLifecycleLiteReadPreservesOriginalIntegerQuery(t *testing.T) {
	repo := &projectLifecycleReadFake{}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	s, _ := NewProjectLifecycle(repo, &activitySessionStub{}, &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}}, throttle)
	ctx, err := s.APIKeyAuth(context.Background(), "key", &security.APIKeyScheme{Name: "ApiKeyAuth"})
	if err != nil {
		t.Fatal(err)
	}
	ctx = WithReadServiceQueryPresence(ctx, map[string][]string{"fields": {"id, name,,project_lead"}, "expand": {"project_lead, workspace"}, "per_page": {"invalid"}})
	metadata, body, err := s.GetAPIV1WorkspacesBySlugProjectsLite(ctx, &api.GetAPIV1WorkspacesBySlugProjectsLitePayload{Slug: "space"})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	raw, _ := io.ReadAll(body)
	if metadata.ContentType != "application/json" || string(raw) != `{"results":[{"id":"p"}]}` {
		t.Fatalf("raw=%s metadata=%v", raw, metadata)
	}
	if repo.options.PerPage == nil || *repo.options.PerPage != "invalid" {
		t.Fatalf("options=%+v", repo.options)
	}
}
