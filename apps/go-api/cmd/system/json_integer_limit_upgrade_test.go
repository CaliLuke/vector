package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	sticky "github.com/CaliLuke/go-plane/gen/stickies"
	"github.com/CaliLuke/go-plane/gen/system"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestStickyHTTPIntegerDigitBoundary(t *testing.T) {
	digits := strings.Repeat("7", 4300)
	for _, value := range []string{digits, "-" + digits, `"` + digits + `7"`} {
		t.Run("value_prefix_"+value[:1], func(t *testing.T) {
			response, stub := stickyIntegerLimitRequest(t, `{"description":`+value+`}`, true)
			if response.Code != http.StatusOK || stub.updated == nil {
				t.Fatalf("status = %d, update called = %t, want 200 and update", response.Code, stub.updated != nil)
			}
			if string(stub.updated.Description) != value {
				t.Fatal("request description did not preserve the exact number or string")
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
				t.Fatal(err)
			}
			if string(fields["description"]) != value {
				t.Fatal("response description did not preserve the exact number or string")
			}
		})
	}
}

func TestStickyHTTPRejectsOversizedIntegerTokensBeforeUpdate(t *testing.T) {
	digits := strings.Repeat("7", 4301)
	for _, test := range []struct {
		name string
		body string
	}{
		{"positive", `{"description":` + digits + `}`},
		{"negative", `{"description":-` + digits + `}`},
		{"nested", `{"description":{"items":[{"value":` + digits + `}]}}`},
		{"unknown_field", `{"ignored":` + digits + `}`},
		{"nested_unknown_field", `{"ignored":{"items":[` + digits + `]}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			response, stub := stickyIntegerLimitRequest(t, test.body, true)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", response.Code)
			}
			if stub.updated != nil {
				t.Fatal("oversized integer reached the update service")
			}
			var problem loomhttp.ProblemResponse
			if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
				t.Fatal(err)
			}
			if problem.Code != loom.DecodePayload || problem.Detail != "invalid request body" {
				t.Fatalf("error code/detail = %q/%q, want existing decode_payload/invalid request body", problem.Code, problem.Detail)
			}
		})
	}
}

func TestStickyHTTPOversizedIntegerKeepsAuthenticationPrecedence(t *testing.T) {
	response, stub := stickyIntegerLimitRequest(t, `{"description":`+strings.Repeat("7", 4301)+`}`, false)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
	if stub.updated != nil {
		t.Fatal("unauthenticated request reached the update service")
	}
}

func TestDjangoRequestDecoderKeepsDecimalAndExponentTokens(t *testing.T) {
	digits := strings.Repeat("7", 4301)
	for _, token := range []string{digits + ".0", digits + "e0", "-" + digits + "E-1"} {
		t.Run(token[len(token)-2:], func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(token))
			request.Header.Set("Content-Type", "application/json")
			var value loom.JSONValue
			if err := djangoRequestDecoder(request).Decode(&value); err != nil {
				t.Fatalf("non-integer number rejected: %v", err)
			}
			if string(value) != token {
				t.Fatal("non-integer number lost its original representation")
			}
		})
	}
}

func TestDjangoRequestDecoderPreservesRequestLimitError(t *testing.T) {
	policy, err := loomhttp.NewRequestBodyPolicy(32)
	if err != nil {
		t.Fatal(err)
	}
	var decodeErr error
	var value loom.JSONValue
	handler := policy.Handler(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		decodeErr = djangoRequestDecoder(request).Decode(&value)
	}))
	request := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(strings.Repeat("7", 4301)))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(httptest.NewRecorder(), request)
	var serviceErr *loom.ServiceError
	if !errors.As(decodeErr, &serviceErr) || serviceErr.Name != loom.RequestBodyTooLarge {
		t.Fatalf("decode error = %v, want request body too large", decodeErr)
	}
	if len(value) != 0 {
		t.Fatal("oversized request body was decoded")
	}
}

func TestDjangoRequestDecoderPreservesNonNullableBodyError(t *testing.T) {
	request := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader("null"))
	request = loomhttp.WithNonNullableBody(request)
	var value loom.JSONValue
	got := djangoRequestDecoder(request).Decode(&value)
	reference := loomhttp.WithNonNullableBody(httptest.NewRequest(http.MethodPatch, "/", strings.NewReader("null")))
	var referenceValue loom.JSONValue
	want := loomhttp.RequestDecoder(reference).Decode(&referenceValue)
	if want == nil || !errors.Is(got, want) {
		t.Fatalf("decode error = %v, want Loom's non-nullable error %v", got, want)
	}
}

func TestDjangoRequestDecoderPreservesUnsupportedMediaError(t *testing.T) {
	request := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(strings.Repeat("7", 4301)))
	request.Header.Set("Content-Type", "application/octet-stream")
	var value loom.JSONValue
	err := djangoRequestDecoder(request).Decode(&value)
	var serviceErr *loom.ServiceError
	if !errors.As(err, &serviceErr) || serviceErr.Name != loom.UnsupportedMediaType {
		t.Fatalf("decode error = %v, want unsupported media type", err)
	}
}

func TestDjangoRequestDecoderPreservesTextBody(t *testing.T) {
	body := strings.Repeat("7", 4301)
	request := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(body))
	request.Header.Set("Content-Type", "text/plain")
	var value string
	if err := djangoRequestDecoder(request).Decode(&value); err != nil {
		t.Fatal(err)
	}
	if value != body {
		t.Fatal("text body did not preserve its exact value")
	}
}

func TestDjangoRequestDecoderLimitsDefaultAndParameterizedJSON(t *testing.T) {
	for _, contentType := range []string{"", "application/json; charset=utf-8"} {
		t.Run(contentType, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(strings.Repeat("7", 4301)))
			request.Header.Set("Content-Type", contentType)
			var value loom.JSONValue
			if err := djangoRequestDecoder(request).Decode(&value); err == nil {
				t.Fatal("oversized JSON integer was accepted")
			}
		})
	}
}

func TestDjangoRequestDecoderPreservesEmptyBodyEOF(t *testing.T) {
	request := httptest.NewRequest(http.MethodPatch, "/", nil)
	request.Header.Set("Content-Type", "application/json")
	var value loom.JSONValue
	if err := djangoRequestDecoder(request).Decode(&value); !errors.Is(err, io.EOF) {
		t.Fatalf("decode error = %v, want EOF", err)
	}
}

func stickyIntegerLimitRequest(t *testing.T, body string, authenticated bool) (*httptest.ResponseRecorder, *stickyJSONRoundTripStub) {
	t.Helper()
	stub := &stickyJSONRoundTripStub{stickyHTTPStub: stickyHTTPStub{success: true}}
	handler := newHTTPHandlerWithStickies(newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), sticky.NewEndpoints(stub), stub.APIKeyAuth)
	request := httptest.NewRequest(http.MethodPatch, "/api/workspaces/test/stickies/0198a28d-d227-7752-82ac-bc4c1c290703/", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if authenticated {
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response, stub
}
