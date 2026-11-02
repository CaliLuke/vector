package design

import (
	"encoding/json"
	"os"
	"testing"
)

func TestUserIdentityOpenAPIExposesNativeSessionContracts(t *testing.T) {
	generated := readOpenAPI(t)
	for _, contract := range []struct {
		method, path, id string
		statuses         []string
	}{
		{"get", "/api/users/session/", "user_identity.session", []string{"200"}},
		{"get", "/api/users/me/", "user_identity.me", []string{"200", "401"}},
		{"get", "/api/users/me/profile/", "user_identity.profile", []string{"200", "401"}},
		{"patch", "/api/users/me/profile/", "user_identity.update_profile", []string{"200", "400", "401"}},
	} {
		operation, ok := generated.Paths[contract.path][contract.method]
		if !ok || operation.OperationID != contract.id {
			t.Fatalf("missing native operation %s %s", contract.method, contract.path)
		}
		for _, status := range contract.statuses {
			if _, ok := operation.Responses[status]; !ok {
				t.Errorf("%s missing HTTP %s", contract.path, status)
			}
		}
	}
}

func TestSessionUserEmailIsRequiredAndNullableInOpenAPI(t *testing.T) {
	data, err := os.ReadFile("../gen/http/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					AnyOf []struct {
						Type string `json:"type"`
					} `json:"anyOf"`
				} `json:"properties"`
				Required []string `json:"required"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	schema, ok := document.Components.Schemas["SessionUser"]
	if !ok {
		t.Fatal("SessionUser schema is missing")
	}
	if !containsString(schema.Required, "email") {
		t.Fatal("SessionUser.email must remain required")
	}
	email, ok := schema.Properties["email"]
	if !ok || len(email.AnyOf) != 2 || email.AnyOf[0].Type != "string" || email.AnyOf[1].Type != "null" {
		t.Fatalf("SessionUser.email anyOf = %#v, want string and null", email.AnyOf)
	}
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
