package design

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestUserIdentityDesignCoversCommittedParityFixtures(t *testing.T) {
	design, err := os.ReadFile("user_identity.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		`Service("user_identity"`, `Method("session"`, `Method("me"`, `Method("profile"`, `Method("update_profile"`,
		`GET("/api/users/session/")`, `GET("/api/users/me/")`, `GET("/api/users/me/profile/")`, `PATCH("/api/users/me/profile/")`,
	} {
		if !strings.Contains(string(design), fragment) {
			t.Errorf("user identity design missing %q", fragment)
		}
	}
	fixtures := []struct{ file, method, path string }{
		{"../internal/parity/testdata/fixtures/user_identity/session-anonymous.json", "GET", "/api/users/session/"},
		{"../internal/parity/testdata/fixtures/user_identity/session-authenticated.json", "GET", "/api/users/session/"},
		{"../internal/parity/testdata/fixtures/user_identity/profile-get.json", "GET", "/api/users/me/profile/"},
		{"../internal/parity/testdata/fixtures/user_identity/profile-patch.json", "PATCH", "/api/users/me/profile/"},
		{"../internal/parity/testdata/fixtures/user_me_read/authenticated.json", "GET", "/api/users/me/"},
		{"../internal/parity/testdata/fixtures/user_me_read/session.json", "GET", "/api/users/me/"},
		{"../internal/parity/testdata/fixtures/user_me_read/anonymous.json", "GET", "/api/users/me/"},
	}
	for _, fixture := range fixtures {
		data, err := os.ReadFile(fixture.file)
		if err != nil {
			t.Fatalf("read %s: %v", fixture.file, err)
		}
		var document struct {
			Request struct{ Method, Path string } `json:"request"`
		}
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatalf("decode %s: %v", fixture.file, err)
		}
		if document.Request.Method != fixture.method || document.Request.Path != fixture.path {
			t.Errorf("%s request = %s %s, want %s %s", fixture.file, document.Request.Method, document.Request.Path, fixture.method, fixture.path)
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
