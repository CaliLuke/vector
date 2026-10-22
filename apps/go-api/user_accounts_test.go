package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	useraccountsapi "github.com/CaliLuke/go-plane/gen/user_accounts"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	useraccountsstore "github.com/CaliLuke/go-plane/internal/useraccounts"
)

const accountJSON = `{"id":"00000000-0000-4000-8000-000000000021","created_at":"2026-08-26T21:00:00Z","updated_at":"2026-08-26T21:00:00Z","user":"00000000-0000-4000-8000-000000000010","provider_account_id":"provider-1","provider":"github","access_token":"secret","access_token_expired_at":null,"refresh_token":null,"refresh_token_expired_at":null,"last_connected_at":"2026-08-26T21:00:00Z","id_token":"","metadata":{}}`

type userAccountsRepositoryStub struct {
	deletedUser    uuid.UUID
	deletedAccount uuid.UUID
	err            error
}

func (stub *userAccountsRepositoryStub) List(context.Context, uuid.UUID) (json.RawMessage, error) {
	return json.RawMessage("[" + accountJSON + "]"), stub.err
}

func (stub *userAccountsRepositoryStub) Get(context.Context, uuid.UUID, uuid.UUID) (json.RawMessage, error) {
	return json.RawMessage(accountJSON), stub.err
}

func (stub *userAccountsRepositoryStub) Delete(_ context.Context, userID, accountID uuid.UUID) error {
	stub.deletedUser = userID
	stub.deletedAccount = accountID
	return stub.err
}

type userAccountsSessionStub struct{ err error }

func (stub userAccountsSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{UserID: "00000000-0000-4000-8000-000000000010"}, stub.err
}

func userAccountsContext(t *testing.T, service useraccountsapi.Service) context.Context {
	t.Helper()
	ctx, err := service.(useraccountsapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TestUserAccountsListRetrieveAndDelete(t *testing.T) {
	repository := &userAccountsRepositoryStub{}
	service, err := NewUserAccounts(repository, userAccountsSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := userAccountsContext(t, service)

	list, err := service.List(ctx, &useraccountsapi.ListPayload{})
	if err != nil || len(list) != 1 || list[0].Provider != "github" || list[0].AccessToken != "secret" {
		t.Fatalf("List() = %#v, %v", list, err)
	}

	account, err := service.Retrieve(ctx, &useraccountsapi.RetrievePayload{Pk: "00000000-0000-4000-8000-000000000021"})
	if err != nil || account.ID != "00000000-0000-4000-8000-000000000021" {
		t.Fatalf("Retrieve() = %#v, %v", account, err)
	}

	err = service.Delete(ctx, &useraccountsapi.DeletePayload{Pk: "00000000-0000-4000-8000-000000000021"})
	if err != nil || repository.deletedUser.String() != "00000000-0000-4000-8000-000000000010" || repository.deletedAccount.String() != "00000000-0000-4000-8000-000000000021" {
		t.Fatalf("Delete() = %v, user %s, account %s", err, repository.deletedUser, repository.deletedAccount)
	}
}

func TestUserAccountsErrors(t *testing.T) {
	service, _ := NewUserAccounts(&userAccountsRepositoryStub{}, userAccountsSessionStub{err: errors.New("invalid session")})
	_, err := service.(useraccountsapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	var unauthenticated *useraccountsapi.UserAccountUnauthenticated
	if !errors.As(err, &unauthenticated) {
		t.Fatalf("auth error = %#v", err)
	}

	repository := &userAccountsRepositoryStub{err: useraccountsstore.ErrNotFound}
	service, _ = NewUserAccounts(repository, userAccountsSessionStub{})
	ctx := userAccountsContext(t, service)
	_, err = service.Retrieve(ctx, &useraccountsapi.RetrievePayload{Pk: "00000000-0000-4000-8000-000000000021"})
	var notFound *useraccountsapi.UserAccountNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("not found error = %#v", err)
	}

	err = service.DeleteCollection(ctx, &useraccountsapi.DeleteCollectionPayload{})
	var internal *useraccountsapi.UserAccountInternalError
	if !errors.As(err, &internal) {
		t.Fatalf("collection delete error = %#v", err)
	}
}

func TestUserAccountEmailUsesDjangoNormalizationAndValidation(t *testing.T) {
	if got, want := normalizeUserAccountEmail("\u2003İ@LOCALHOST\u001c"), "i\u0307@localhost"; got != want {
		t.Fatalf("normalized email=%q, want %q", got, want)
	}
	for email, want := range map[string]bool{
		"person@localhost":       true,
		"person@[2001:db8::1]":   true,
		`"quoted"@example.com`:   true,
		"person@example":         false,
		"Person <a@example.com>": false,
	} {
		if got := validEmail(email); got != want {
			t.Errorf("validEmail(%q)=%t, want %t", email, got, want)
		}
	}
}
