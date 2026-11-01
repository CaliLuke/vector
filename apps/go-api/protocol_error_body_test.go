package systemapi

import (
	"net/http"
	"testing"

	accountapi "github.com/CaliLuke/go-plane/accountauthentication/gen/account_authentication"
	dashboardapi "github.com/CaliLuke/go-plane/workspacepersonal/gen/personal_dashboard"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestProtocolErrorBridgesReplaceInvalidErrorBodiesWithInternalFailure(t *testing.T) {
	for _, body := range []string{`{`, `[]`, `null`, `"not an error object"`, ``} {
		t.Run(body, func(t *testing.T) {
			accountStatus, accountBody := accountAuthenticationProtocolError(&accountapi.AccountAuthenticationError{Name: "status_403", Body: loom.JSONValue(body)})
			if accountStatus != http.StatusInternalServerError || accountBody["error"] == nil {
				t.Errorf("account protocol response = %d %v; want an internal failure body", accountStatus, accountBody)
			}
			dashboardStatus, dashboardBody := personalDashboardProtocolError(&dashboardapi.PersonalDashboardErrorResponse{Name: "status_403", Body: loom.JSONValue(body)})
			if dashboardStatus != http.StatusInternalServerError || dashboardBody["error"] == nil {
				t.Errorf("dashboard protocol response = %d %v; want an internal failure body", dashboardStatus, dashboardBody)
			}
		})
	}
}
