package design

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestUserIdentityOpenAPICoversCommittedParityFixtures(t *testing.T) {
	generated := readOpenAPI(t)
	fixtures := []struct{ file, method, path, operationID string }{
		{"../internal/parity/testdata/fixtures/user_identity/session-anonymous.json", "GET", "/api/users/session/", "user_identity.session"},
		{"../internal/parity/testdata/fixtures/user_identity/session-authenticated.json", "GET", "/api/users/session/", "user_identity.session"},
		{"../internal/parity/testdata/fixtures/user_identity/profile-get.json", "GET", "/api/users/me/profile/", "user_identity.profile"},
		{"../internal/parity/testdata/fixtures/user_identity/profile-patch.json", "PATCH", "/api/users/me/profile/", "user_identity.update_profile"},
		{"../internal/parity/testdata/fixtures/user_me_read/authenticated.json", "GET", "/api/users/me/", "user_identity.me"},
		{"../internal/parity/testdata/fixtures/user_me_read/session.json", "GET", "/api/users/me/", "user_identity.me"},
		{"../internal/parity/testdata/fixtures/user_me_read/anonymous.json", "GET", "/api/users/me/", "user_identity.me"},
	}
	for _, fixture := range fixtures {
		data, err := os.ReadFile(fixture.file)
		if err != nil {
			t.Fatalf("read %s: %v", fixture.file, err)
		}
		var document struct {
			Request  struct{ Method, Path string } `json:"request"`
			Response struct {
				Status int `json:"status"`
			} `json:"response"`
		}
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatalf("decode %s: %v", fixture.file, err)
		}
		if document.Request.Method != fixture.method || document.Request.Path != fixture.path {
			t.Errorf("%s request = %s %s, want %s %s", fixture.file, document.Request.Method, document.Request.Path, fixture.method, fixture.path)
		}
		operation, exists := generated.Paths[fixture.path][strings.ToLower(fixture.method)]
		if !exists {
			t.Errorf("fixture %s has no generated %s %s operation", fixture.file, fixture.method, fixture.path)
			continue
		}
		if operation.OperationID != fixture.operationID {
			t.Errorf("%s %s operation ID=%q want=%q", fixture.method, fixture.path, operation.OperationID, fixture.operationID)
		}
		if _, exists := operation.Responses[strconv.Itoa(document.Response.Status)]; !exists {
			t.Errorf("%s has no generated response for fixture status %d", fixture.file, document.Response.Status)
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
