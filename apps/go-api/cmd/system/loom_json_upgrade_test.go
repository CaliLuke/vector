package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	sticky "github.com/CaliLuke/go-plane/gen/stickies"
	"github.com/CaliLuke/go-plane/gen/system"
	loom "github.com/CaliLuke/loom/pkg"
)

type stickyJSONRoundTripStub struct {
	stickyHTTPStub
	updated *sticky.UpdatePayload
}

func (stub *stickyJSONRoundTripStub) Update(_ context.Context, payload *sticky.UpdatePayload) (*sticky.StickyResponse, error) {
	stub.updated = payload
	result := stickyHTTPSuccessResponse()
	result.Description = payload.Description
	result.LogoProps = payload.LogoProps
	if len(result.Description) == 0 {
		result.Description = loom.JSONValue("null")
		result.LogoProps = loom.JSONValue("null")
	}
	return result, nil
}

func TestStickyHTTPPreservesJSONPresenceAndNumbers(t *testing.T) {
	for _, value := range []string{"", "null", "{}", "[]", `{"id":9007199254740993,"nested":[]}`} {
		t.Run(value, func(t *testing.T) {
			stub := &stickyJSONRoundTripStub{stickyHTTPStub: stickyHTTPStub{success: true}}
			handler := newHTTPHandlerWithStickies(newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false, sticky.NewEndpoints(stub), stub.APIKeyAuth)
			body := `{}`
			if value != "" {
				body = `{"description":` + value + `,"logo_props":` + value + `}`
			}
			request := httptest.NewRequest(http.MethodPatch, "/api/workspaces/test/stickies/0198a28d-d227-7752-82ac-bc4c1c290703/", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK || stub.updated == nil {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			if string(stub.updated.Description) != value || string(stub.updated.LogoProps) != value {
				t.Fatalf("decoded values = %s, %s, want %s", stub.updated.Description, stub.updated.LogoProps, value)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
				t.Fatal(err)
			}
			want := value
			if want == "" {
				want = "null"
			}
			if string(fields["description"]) != want || string(fields["logo_props"]) != want {
				t.Fatalf("response lost JSON values: %s", response.Body.String())
			}
		})
	}
}

func TestStickyFormKeepsJSONNumbers(t *testing.T) {
	const value = `{"id":9007199254740993,"items":[]}`
	body, err := encodeStickyForm(url.Values{"description": {value}, "logo_props": {value}})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["description"]) != value || string(fields["logo_props"]) != value {
		t.Fatalf("form JSON lost precision: %s", body)
	}
}
