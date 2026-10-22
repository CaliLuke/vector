package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReadServiceMalformedSlashlessPathsKeepResolver404(t *testing.T) {
	policy, err := djangoRuntimeCORSPolicy(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		path   string
		status int
		json   bool
	}{
		{"/api/workspaces/demo/invitations/not-a-uuid", 404, true},
		{"/api/workspaces/demo/invitations/00000000-0000-4000-8000-00000000ABCD/", 404, true},
		{"/api/workspaces/demo/invitations/0000000000004000800000000000abcd/", 404, true},
		{"/api/workspaces/demo/projects/00000000-0000-4000-8000-00000000abcd/invitations/not-a-uuid/", 404, true},
		{"/api/workspaces/demo/members/not-a-uuid/", 404, true},
		// Other families retain their existing fallback and audit ownership.
		{"/api/workspaces/demo/projects/not-a-uuid/modules", 204, false},
		{"/api/v1/workspaces/demo/invitations/not.a.uuid", 404, true},
		{"/api/v1/workspaces/demo/invitations/00000000-0000-4000-8000-00000000abcd.json", 404, true},
		{"/api/v1/workspaces/demo/invitations/0000000000004000800000000000abcd", 204, false},
		{"/api/workspaces/demo/invitations/00000000-0000-4000-8000-00000000abcd", 204, false},
	} {
		t.Run(test.path, func(t *testing.T) {
			handler := readServicesDjangoPathConverterMiddleware(policy, &apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			if response.Code != test.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if test.json {
				if response.Header().Get("Content-Type") != "application/json" || !strings.Contains(response.Header().Get("Vary"), "Cookie") {
					t.Fatalf("source resolver headers=%v", response.Header())
				}
				var body map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["error"] != "Page not found." {
					t.Fatalf("resolver response=%s err=%v", response.Body.String(), err)
				}
			}
		})
	}
}

func TestInvitationsMembershipMetadataMatchesDjangoFixtures(t *testing.T) {
	for pattern, metadata := range invitationsMembershipMetadata {
		data, err := os.ReadFile(filepath.Join("..", "..", "internal", "parity", "testdata", "fixtures", "invitations_membership", metadata.Fixture))
		if err != nil {
			t.Fatal(err)
		}
		var fixture struct {
			Request struct {
				Method string `json:"method"`
			} `json:"request"`
			Response struct {
				Status int `json:"status"`
				Body   any `json:"body"`
			} `json:"response"`
		}
		if err := json.Unmarshal(data, &fixture); err != nil {
			t.Fatal(err)
		}
		var body any
		if err := json.Unmarshal(metadata.Metadata, &body); err != nil {
			t.Fatal(err)
		}
		// Secret-value redaction must not replace an OPTIONS field schema just
		// because its field name is token. Check structure independently of the
		// mirrored fixture so the same corrupted oracle cannot make this pass.
		if object, ok := body.(map[string]any); ok {
			if actions, ok := object["actions"].(map[string]any); ok {
				for method, value := range actions {
					fields, ok := value.(map[string]any)
					if !ok {
						t.Fatalf("%s %s action is not a field map", pattern, method)
					}
					for field, value := range fields {
						info, ok := value.(map[string]any)
						if !ok || info["type"] == nil {
							t.Errorf("%s %s field %s lost its schema: %v", pattern, method, field, value)
						}
					}
				}
			}
		}
		if fixture.Request.Method != http.MethodOptions || fixture.Response.Status != 200 || !reflect.DeepEqual(body, fixture.Response.Body) {
			t.Errorf("%s metadata differs from source fixture %s", pattern, metadata.Fixture)
		}
	}
}

func TestInvitationsMembershipProtocolConverters(t *testing.T) {
	const prefix = "/api/workspaces/workspace/invitations/"
	for _, test := range []struct {
		path      string
		malformed bool
	}{
		{prefix + "00000000-0000-4000-8000-00000000abcd/", false},
		{prefix + "00000000-0000-4000-8000-00000000ABCD/", true},
		{prefix + "0000000000004000800000000000abcd/", true},
		{"/api/v1/workspaces/workspace/invitations/00000000-0000-4000-8000-00000000ABCD/", false},
		{"/api/v1/workspaces/workspace/invitations/0000000000004000800000000000abcd/", false},
		{"/api/v1/workspaces/workspace/invitations/not-a-uuid/", false},
		{"/api/v1/workspaces/workspace/invitations/not.a.uuid/", true},
	} {
		protocol, found := resolveReadServiceProtocol(test.path)
		if !found {
			t.Fatalf("missing protocol for %s", test.path)
		}
		if got := readServiceProtocolHasMalformedUUID(test.path, protocol); got != test.malformed {
			t.Errorf("%s malformed = %t, want %t", test.path, got, test.malformed)
		}
	}
}

func TestInvitationsMembershipProtocolMethodDistinctions(t *testing.T) {
	for _, test := range []struct{ path, allow string }{
		{"/api/v1/workspaces/{slug}/invitations/", "GET, POST, HEAD, OPTIONS"},
		{"/api/v1/workspaces/{slug}/invitations/{pk}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS"},
		{"/api/v1/workspaces/{slug}/members/", "GET"},
		{"/api/v1/workspaces/{slug}/members-lite/", "GET"},
		{"/api/workspaces/{slug}/members/leave/", "POST, OPTIONS"},
		{"/api/workspaces/{slug}/members/{pk}/", "PATCH, DELETE, OPTIONS"},
	} {
		protocol := mustReadServiceProtocol(test.path)
		if protocol.allow != test.allow {
			t.Errorf("%s Allow = %q, want %q", test.path, protocol.allow, test.allow)
		}
		if protocol.derivedHead != readServiceMethodAllowed(test.allow, http.MethodHead) {
			t.Errorf("%s HEAD ownership disagrees with source Allow", test.path)
		}
	}
	// The literal leave path must win over the generic member ID path.
	protocol, ok := resolveReadServiceProtocol("/api/workspaces/workspace/members/leave/")
	if !ok || protocol.allow != "POST, OPTIONS" {
		t.Fatalf("member leave resolved to %#v", protocol)
	}
}
