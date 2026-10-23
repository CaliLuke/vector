package authredirects

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/accountauthentication"
)

func TestAdminSignUpLogsInTheAdminWhoCompletedSetup(t *testing.T) {
	accounts := &initialAdminAccounts{
		accountStub: accountStub{userErr: accountauthentication.ErrUserNotFound},
		result: accountauthentication.InitialAdminResult{
			Exists: true, SetupDone: true,
			User: accountauthentication.User{
				ID: "00000000-0000-4000-8000-000000000001", Email: "admin@example.test", Password: "encoded", Active: true,
			},
		},
	}
	service := newTestService(t, &accounts.accountStub, 4)
	service.accounts = accounts

	result := service.AdminSignUp(context.Background(), Request{
		Email: "admin@example.test", Password: "test-password", FirstName: "Admin", BaseURL: "http://localhost:8000/god-mode",
	})

	if result.Status != http.StatusFound || result.Location != "http://localhost:8000/god-mode/general/" {
		t.Fatalf("setup response=%d %q", result.Status, result.Location)
	}
	if result.Session == nil || result.Session.SessionKey == "" {
		t.Fatal("successful first-admin setup did not establish a session")
	}
}

func TestAdminSignUpDoesNotLogInWhenAnotherRequestCompletesSetup(t *testing.T) {
	accounts := &initialAdminAccounts{
		accountStub: accountStub{userErr: accountauthentication.ErrUserNotFound},
		result:      accountauthentication.InitialAdminResult{Exists: true, SetupDone: true, AlreadyExists: true},
	}
	service := newTestService(t, &accounts.accountStub, 4)
	service.accounts = accounts

	result := service.AdminSignUp(context.Background(), Request{
		Email: "admin@example.test", Password: "test-password", FirstName: "Admin", BaseURL: "http://localhost:8000/god-mode",
	})

	if result.Status != http.StatusFound || !strings.Contains(result.Location, "error_code=5150") {
		t.Fatalf("concurrent setup response=%d %q", result.Status, result.Location)
	}
	if result.Session != nil {
		t.Fatal("rejected setup established a session")
	}
}

type initialAdminAccounts struct {
	accountStub
	result accountauthentication.InitialAdminResult
}

var _ AdminAccountRepository = (*initialAdminAccounts)(nil)

func (*initialAdminAccounts) InstanceExists(context.Context) (bool, error) {
	return true, nil
}

func (*initialAdminAccounts) InitialAdminState(context.Context) (bool, bool, bool, error) {
	return true, false, false, nil
}

func (*initialAdminAccounts) IsInstanceAdmin(context.Context, bool, string) (bool, error) {
	return false, nil
}

func (accounts *initialAdminAccounts) CreateInitialAdmin(
	context.Context, string, string, string, string, string, string, string, bool, time.Time,
) (accountauthentication.InitialAdminResult, error) {
	return accounts.result, nil
}
