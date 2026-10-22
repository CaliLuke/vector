package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/user_api_tokens"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/userapitokens"
	loom "github.com/CaliLuke/loom/pkg"
)

const tokenUser = "00000000-0000-4000-8000-000000000010"
const tokenID = "00000000-0000-4000-8000-000000000011"
const tokenJSONFixture = `{"id":"00000000-0000-4000-8000-000000000011","created_at":"2026-08-26T21:00:00Z","updated_at":"2026-08-26T21:00:00Z","label":"Fixture","description":"desc","is_active":true,"last_used":null,"created_by":"00000000-0000-4000-8000-000000000010","updated_by":null,"deleted_at":null,"token":"plane_api_secret","user":"00000000-0000-4000-8000-000000000010","user_type":0,"workspace":null,"expired_at":null,"is_service":false,"allowed_rate_limit":"60/min"}`

type apiTokenRepositoryStub struct {
	raw                                            json.RawMessage
	err                                            error
	deleted                                        bool
	createdLabel, createdDescription, createdToken string
	expiredAt                                      *string
	patchLabel, patchDescription                   *string
}

func (s *apiTokenRepositoryStub) List(context.Context, uuid.UUID) (json.RawMessage, error) {
	if s.err != nil {
		return nil, s.err
	}
	return json.RawMessage("[" + string(s.raw) + "]"), nil
}
func (s *apiTokenRepositoryStub) Get(context.Context, uuid.UUID, uuid.UUID) (json.RawMessage, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.raw, nil
}
func (s *apiTokenRepositoryStub) Create(_ context.Context, _ uuid.UUID, label, description, token string, expired *string) (json.RawMessage, error) {
	s.createdLabel, s.createdDescription, s.createdToken, s.expiredAt = label, description, token, expired
	return s.raw, s.err
}
func (s *apiTokenRepositoryStub) Patch(_ context.Context, _ uuid.UUID, _ uuid.UUID, label, description *string) (json.RawMessage, error) {
	s.patchLabel, s.patchDescription = label, description
	return s.raw, s.err
}
func (s *apiTokenRepositoryStub) Delete(context.Context, uuid.UUID, uuid.UUID) error {
	s.deleted = true
	return s.err
}

type apiTokenSessionStub struct{ err error }

func (s apiTokenSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{UserID: tokenUser}, s.err
}
func tokenService(t *testing.T, repo *apiTokenRepositoryStub) (api.Service, context.Context) {
	t.Helper()
	svc, e := NewUserAPITokens(repo, apiTokenSessionStub{})
	if e != nil {
		t.Fatal(e)
	}
	ctx, e := svc.(api.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if e != nil {
		t.Fatal(e)
	}
	return svc, ctx
}

func TestAPITokensAuthDefaultsAndRedaction(t *testing.T) {
	repo := &apiTokenRepositoryStub{raw: json.RawMessage(tokenJSONFixture)}
	svc, ctx := tokenService(t, repo)
	list, e := svc.List(ctx, &api.ListPayload{})
	if e != nil || len(list) != 1 || list[0].ID != tokenID {
		t.Fatalf("List=%#v err=%v", list, e)
	}
	encoded, _ := json.Marshal(list[0])
	if string(encoded) == "" {
		t.Fatal("empty list response")
	}
	expiry := "2026-09-01T00:00:00Z"
	created, e := svc.Create(ctx, &api.CreatePayload{ExpiredAt: loom.NullableValue[string](expiry)})
	if e != nil || created.Token == "" || repo.createdLabel == "" || repo.createdDescription != "" || repo.expiredAt == nil || *repo.expiredAt != expiry || len(repo.createdToken) < len("plane_api_") {
		t.Fatalf("Create=%#v err=%v repo=%#v", created, e, repo)
	}
}

func TestAPITokensPatchIncludesTokenAndDeleteSoftDelete(t *testing.T) {
	repo := &apiTokenRepositoryStub{raw: json.RawMessage(tokenJSONFixture)}
	svc, ctx := tokenService(t, repo)
	label, description := "new", "changed"
	out, e := svc.Patch(ctx, &api.PatchPayload{Pk: tokenID, Label: &label, Description: &description})
	if e != nil || out.Token == "" || repo.patchLabel == nil || *repo.patchLabel != label {
		t.Fatalf("Patch=%#v err=%v", out, e)
	}
	if e = svc.Delete(ctx, &api.DeletePayload{Pk: tokenID}); e != nil || !repo.deleted {
		t.Fatalf("Delete err=%v deleted=%v", e, repo.deleted)
	}
}

func TestAPITokensOwnerAndPhantomErrors(t *testing.T) {
	repo := &apiTokenRepositoryStub{raw: json.RawMessage(tokenJSONFixture), err: store.ErrNotFound}
	svc, ctx := tokenService(t, repo)
	_, e := svc.Retrieve(ctx, &api.RetrievePayload{Pk: tokenID})
	var nf *api.APITokenNotFound
	if !errors.As(e, &nf) {
		t.Fatalf("retrieve error=%v", e)
	}
	if e = svc.PatchCollection(ctx, &api.PatchCollectionPayload{}); e == nil {
		t.Fatal("collection patch returned nil")
	}
	if e = svc.DeleteCollection(ctx, &api.DeleteCollectionPayload{}); e == nil {
		t.Fatal("collection delete returned nil")
	}
	if e = svc.PostDetail(ctx, &api.PostDetailPayload{Pk: tokenID}); e == nil {
		t.Fatal("detail post returned nil")
	}
}

func TestAPITokensRejectInvalidSession(t *testing.T) {
	svc, e := NewUserAPITokens(&apiTokenRepositoryStub{raw: json.RawMessage(tokenJSONFixture)}, apiTokenSessionStub{err: errors.New("invalid")})
	if e != nil {
		t.Fatal(e)
	}
	_, e = svc.(api.Authorizer).APIKeyAuth(context.Background(), "bad", nil)
	var auth *api.APITokenUnauthenticated
	if !errors.As(e, &auth) {
		t.Fatalf("auth error=%v", e)
	}
}
