package design_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	server "github.com/CaliLuke/go-plane/notificationactions/gen/http/notification_actions/server"
	api "github.com/CaliLuke/go-plane/notificationactions/gen/notification_actions"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

const targetID = "01234567-89ab-cdef-0123-456789abcdef"

// This spike isolates the generated transport from database and source protocol
// middleware; production protocol and durable effects have separate parity tests.
type actionService struct {
	t      *testing.T
	called string
}

func (s *actionService) APIKeyAuth(ctx context.Context, _ string, scheme *security.APIKeyScheme) (context.Context, error) {
	metadata, ok := loomhttp.RequestMetadataFromContext(ctx)
	if !ok || scheme.Name != "PlaneSessionCookie" {
		s.t.Fatal("missing generated session metadata")
	}
	if metadata.Headers().Get("Cookie") != "session-id=spike" {
		return ctx, &api.NotificationError{Name: "status_401", Detail: loom.JSONValue(`"Authentication credentials were not provided."`)}
	}
	return ctx, nil
}
func (s *actionService) record(name, slug, pk string) *api.Notification {
	if slug != "demo" || pk != targetID {
		s.t.Fatalf("wrong target %s/%s", slug, pk)
	}
	s.called = name
	var result api.Notification
	if err := json.Unmarshal([]byte(`{"id":"01234567-89ab-cdef-0123-456789abcdef","workspace":"01234567-89ab-cdef-0123-456789abcdef","receiver":"01234567-89ab-cdef-0123-456789abcdef","created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-02T03:04:05Z","created_by":null,"updated_by":null,"deleted_at":null,"archived_at":null,"read_at":null,"snoozed_till":null,"project":null,"entity_identifier":null,"entity_name":"issue","sender":"Issue changed","title":"","message_html":"","message_stripped":null,"triggered_by":null,"triggered_by_details":null,"data":null,"message":null}`), &result); err != nil {
		s.t.Fatal(err)
	}
	return &result
}
func (s *actionService) Delete(_ context.Context, p *api.DeletePayload) error {
	s.record("Delete", p.Slug, p.Pk)
	return nil
}
func (s *actionService) Archive(_ context.Context, p *api.ArchivePayload, _ io.ReadCloser) (*api.Notification, error) {
	return s.record("Archive", p.Slug, p.Pk), nil
}
func (s *actionService) Unarchive(_ context.Context, p *api.UnarchivePayload) (*api.Notification, error) {
	return s.record("Unarchive", p.Slug, p.Pk), nil
}
func (s *actionService) MarkRead(_ context.Context, p *api.MarkReadPayload, _ io.ReadCloser) (*api.Notification, error) {
	return s.record("MarkRead", p.Slug, p.Pk), nil
}
func (s *actionService) MarkUnread(_ context.Context, p *api.MarkUnreadPayload) (*api.Notification, error) {
	return s.record("MarkUnread", p.Slug, p.Pk), nil
}
func (s *actionService) MarkAllRead(_ context.Context, p *api.MarkAllReadPayload, _ io.ReadCloser) (*api.NotificationActionMessage, error) {
	if p.Slug != "demo" {
		s.t.Fatalf("wrong slug %q", p.Slug)
	}
	s.called = "MarkAllRead"
	return &api.NotificationActionMessage{Message: "Successful"}, nil
}
func (s *actionService) Update(_ context.Context, p *api.UpdatePayload, _ io.ReadCloser) (*api.Notification, error) {
	return s.record("Update", p.Slug, p.Pk), nil
}

func actionTransport(t *testing.T) (http.Handler, *actionService) {
	t.Helper()
	service := &actionService{t: t}
	mux := loomhttp.NewMuxer()
	cors, err := loomhttp.NewRuntimeCORSPolicy(loomhttp.CORSPolicy{Origins: []loomhttp.CORSOrigin{{Pattern: "https://app.example.test", Methods: []string{"POST", "DELETE"}, Headers: []string{"Content-Type"}, Credentials: true}}})
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := loomhttp.NewRequestMetadataPolicy([]string{"Cookie"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	generated := server.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, cors)
	generated.Use(loomhttp.RequestMetadataMiddleware(metadata))
	generated.Mount(mux)
	return mux, service
}

type unreadBody struct{ reads int }

func (b *unreadBody) Read([]byte) (int, error) { b.reads++; return 0, io.ErrUnexpectedEOF }
func (*unreadBody) Close() error               { return nil }

func TestGeneratedNotificationActionsNeverDecodeIgnoredBodies(t *testing.T) {
	for _, tc := range []struct {
		name, method, path string
		status             int
	}{
		{"Delete", "DELETE", server.DeleteNotificationActionsPath("demo", targetID), 204},
		{"Archive", "POST", server.ArchiveNotificationActionsPath("demo", targetID), 200},
		{"Unarchive", "DELETE", server.UnarchiveNotificationActionsPath("demo", targetID), 200},
		{"MarkRead", "POST", server.MarkReadNotificationActionsPath("demo", targetID), 200},
		{"MarkUnread", "DELETE", server.MarkUnreadNotificationActionsPath("demo", targetID), 200},
		{"MarkAllRead", "POST", server.MarkAllReadNotificationActionsPath("demo"), 200},
		{"Update", "PATCH", server.UpdateNotificationActionsPath("demo", targetID), 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler, service := actionTransport(t)
			body := &unreadBody{}
			req := httptest.NewRequest(tc.method, tc.path, body)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Cookie", "session-id=spike")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			if response.Code != tc.status || service.called != tc.name || body.reads != 0 {
				t.Fatalf("status=%d call=%s body reads=%d response=%s", response.Code, service.called, body.reads, response.Body.String())
			}
		})
	}
}

func TestGeneratedNotificationActionsRejectMissingSessionBeforeIgnoredBody(t *testing.T) {
	handler, service := actionTransport(t)
	body := &unreadBody{}
	req := httptest.NewRequest("POST", server.MarkReadNotificationActionsPath("demo", targetID), body)
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != 401 || service.called != "" || body.reads != 0 {
		t.Fatalf("status=%d call=%s body reads=%d", response.Code, service.called, body.reads)
	}
}

func TestGeneratedNotificationActionsValidateCanonicalUUID(t *testing.T) {
	for _, id := range []string{strings.ToUpper(targetID), strings.ReplaceAll(targetID, "-", ""), "not-a-uuid"} {
		t.Run(id, func(t *testing.T) {
			handler, service := actionTransport(t)
			req := httptest.NewRequest("DELETE", server.DeleteNotificationActionsPath("demo", id), nil)
			req.Header.Set("Cookie", "session-id=spike")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			if response.Code != 400 || service.called != "" {
				t.Fatalf("status=%d call=%s", response.Code, service.called)
			}
		})
	}
}

func TestGeneratedNotificationActionsCORSPreflight(t *testing.T) {
	handler, service := actionTransport(t)
	req := httptest.NewRequest("OPTIONS", server.MarkReadNotificationActionsPath("demo", targetID), nil)
	req.Header.Set("Origin", "https://app.example.test")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != 204 || response.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" || response.Header().Get("Access-Control-Allow-Credentials") != "true" || service.called != "" {
		t.Fatalf("status=%d headers=%v call=%s", response.Code, response.Header(), service.called)
	}
}

func TestGeneratedNotificationActionsOpenAPIContainsOnlySelectedMethods(t *testing.T) {
	data, err := os.ReadFile("../gen/http/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	type requestBody struct {
		Ref      string `json:"$ref"`
		Required bool
		Content  map[string]json.RawMessage
	}
	var doc struct {
		Paths      map[string]map[string]json.RawMessage
		Components struct {
			SecuritySchemes map[string]struct{ Type, In, Name string }
			RequestBodies   map[string]requestBody
		}
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	for path, methods := range doc.Paths {
		for _, method := range []string{"get", "head", "post", "patch", "put", "delete", "options"} {
			if _, ok := methods[method]; ok {
				got[path] = append(got[path], method)
			}
		}
	}
	base := "/api/workspaces/{slug}/users/notifications/{pk}/"
	want := map[string][]string{base: {"patch", "delete"}, base + "read/": {"post", "delete"}, base + "archive/": {"post", "delete"}, "/api/workspaces/{slug}/users/notifications/mark-all-read/": {"post"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("operations=%v want=%v", got, want)
	}
	scheme := doc.Components.SecuritySchemes["PlaneSessionCookie"]
	if scheme.Type != "apiKey" || scheme.In != "cookie" || scheme.Name != "session-id" {
		t.Fatalf("session scheme=%+v", scheme)
	}
	for _, path := range []string{base + "read/", base + "archive/"} {
		var operation struct {
			RequestBody requestBody
			Security    []map[string][]string
		}
		if err := json.Unmarshal(doc.Paths[path]["post"], &operation); err != nil {
			t.Fatal(err)
		}
		if operation.RequestBody.Ref != "" {
			operation.RequestBody = doc.Components.RequestBodies[strings.TrimPrefix(operation.RequestBody.Ref, "#/components/requestBodies/")]
		}
		if !operation.RequestBody.Required || len(operation.RequestBody.Content) != 3 || len(operation.Security) != 1 {
			t.Fatalf("missing source documentation contract on %s: %+v", path, operation)
		}
		for _, media := range []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"} {
			if _, ok := operation.RequestBody.Content[media]; !ok {
				t.Fatalf("missing media %s on %s", media, path)
			}
		}
	}
}
