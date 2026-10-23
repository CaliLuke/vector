package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/workspace_member_me"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
)

type memberMeRepoTest struct {
	raw  json.RawMessage
	err  error
	user uuid.UUID
	slug string
}

func (r *memberMeRepoTest) Get(_ context.Context, user uuid.UUID, slug string) (json.RawMessage, error) {
	r.user, r.slug = user, slug
	return r.raw, r.err
}

type memberMeSessionTest struct{ err error }

func (s memberMeSessionTest) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{UserID: "00000000-0000-4000-8000-000000000001"}, s.err
}

func TestWorkspaceMemberMeGetPreservesActiveAndNullMemberShapes(t *testing.T) {
	for _, testCase := range []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "active",
			raw:  `{"company_role":null,"created_at":"2026-01-01T00:00:00Z","created_by":null,"default_props":{},"deleted_at":null,"draft_issue_count":2,"explored_features":{},"getting_started_checklist":{},"id":"00000000-0000-4000-8000-000000000010","is_active":true,"issue_props":{},"member":"00000000-0000-4000-8000-000000000001","role":15,"tips":{},"updated_at":"2026-01-01T00:00:00Z","updated_by":null,"view_props":{},"workspace":"00000000-0000-4000-8000-000000000020"}`,
			want: `"draft_issue_count":2`,
		},
		{
			name: "inactive or non-member",
			raw:  `{"company_role":"","created_by":null,"default_props":null,"deleted_at":null,"explored_features":null,"getting_started_checklist":null,"is_active":false,"issue_props":null,"member":null,"role":null,"tips":null,"updated_by":null,"view_props":null,"workspace":null}`,
			want: `"role":null`,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			repository := &memberMeRepoTest{raw: json.RawMessage(testCase.raw)}
			service, err := NewWorkspaceMemberMe(repository, memberMeSessionTest{})
			if err != nil {
				t.Fatal(err)
			}
			ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "session", nil)
			if err != nil {
				t.Fatal(err)
			}
			result, err := service.Get(ctx, &api.GetPayload{Slug: "demo"})
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			if !json.Valid(encoded) || !containsJSONFragment(string(encoded), testCase.want) {
				t.Fatalf("result = %s, want fragment %s", encoded, testCase.want)
			}
			if repository.slug != "demo" || repository.user.String() != "00000000-0000-4000-8000-000000000001" {
				t.Fatalf("repository call = user %s slug %q", repository.user, repository.slug)
			}
		})
	}
}

func TestWorkspaceMemberMeAuthenticationAndInternalErrors(t *testing.T) {
	if _, err := NewWorkspaceMemberMe(nil, memberMeSessionTest{}); err == nil {
		t.Fatal("expected nil repository error")
	}
	repository := &memberMeRepoTest{err: errors.New("database")}
	service, err := NewWorkspaceMemberMe(repository, memberMeSessionTest{err: errors.New("bad session")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.(api.Authorizer).APIKeyAuth(context.Background(), "bad", nil); err == nil {
		t.Fatal("expected unauthenticated error")
	}
	if _, err = service.Get(context.Background(), &api.GetPayload{Slug: "demo"}); err == nil {
		t.Fatal("expected missing-principal error")
	}

	service, _ = NewWorkspaceMemberMe(repository, memberMeSessionTest{})
	ctx, _ := service.(api.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if _, err = service.Get(ctx, &api.GetPayload{Slug: "demo"}); err == nil || err.Error() != "internal" {
		t.Fatalf("database error = %v", err)
	}
}

func containsJSONFragment(value, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
