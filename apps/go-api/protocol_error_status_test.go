package systemapi

import (
	"errors"
	"net/http"
	"testing"

	loom "github.com/CaliLuke/loom/pkg"

	accountapi "github.com/CaliLuke/go-plane/accountauthentication/gen/account_authentication"
	collaborationapi "github.com/CaliLuke/go-plane/ticket_collaboration/gen/ticket_collaboration"
	dashboardapi "github.com/CaliLuke/go-plane/workspacepersonal/gen/personal_dashboard"
)

func TestProtocolErrorBridgesNeverAuthorizeMalformedStatusNames(t *testing.T) {
	for _, scenario := range []struct {
		name string
		want int
	}{
		{"status_400", http.StatusBadRequest},
		{"status_401", http.StatusUnauthorized},
		{"status_403", http.StatusForbidden},
		{"status_429", http.StatusTooManyRequests},
		{"status_503", http.StatusServiceUnavailable},
		{"", http.StatusInternalServerError},
		{"status_invalid", http.StatusInternalServerError},
		{"status_0", http.StatusInternalServerError},
		{"status_200", http.StatusInternalServerError},
		{"status_700", http.StatusInternalServerError},
		{"403", http.StatusInternalServerError},
		{"status_403trailing", http.StatusInternalServerError},
		{"status_+403", http.StatusInternalServerError},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			accountStatus, accountBody := accountAuthenticationProtocolError(&accountapi.AccountAuthenticationError{Name: scenario.name, Body: loom.JSONValue(`{"detail":"denied"}`)})
			if accountStatus != scenario.want || accountBody["detail"] != "denied" {
				t.Errorf("account protocol response = %d %v; want status %d with original body", accountStatus, accountBody, scenario.want)
			}
			dashboardStatus, dashboardBody := personalDashboardProtocolError(&dashboardapi.PersonalDashboardErrorResponse{Name: scenario.name, Body: loom.JSONValue(`{"detail":"denied"}`)})
			if dashboardStatus != scenario.want || dashboardBody["detail"] != "denied" {
				t.Errorf("dashboard protocol response = %d %v; want status %d with original body", dashboardStatus, dashboardBody, scenario.want)
			}
		})
	}
}

func TestCollaborationProtocolRejectsPartiallyParsedErrorStatus(t *testing.T) {
	err := collaborationProtocolError(&collaborationapi.CollaborationError{Kind: "status_403trailing", Body: loom.JSONValue(`{"detail":"denied"}`)})
	var named interface{ LoomErrorName() string }
	if !errors.As(err, &named) || named.LoomErrorName() != "status_500" {
		t.Fatalf("converted error=%v; want HTTP 500 for malformed error status", err)
	}
}
