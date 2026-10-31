package devtools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRecordLogRedactsSecretsAndClassifiesErrors(t *testing.T) {
	item := RecordLog("\x1b[31mERROR password=private token=\"private\"\x1b[0m", "plane-go-api", time.Unix(1, 0))
	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "private") {
		t.Fatalf("secret survived redaction: %s", encoded)
	}
	if !strings.Contains(string(encoded), `"severityNumber":17`) || !strings.Contains(string(encoded), `"stringValue":"plane-go-api"`) {
		t.Fatalf("missing error severity or service attribute: %s", encoded)
	}
}

func TestRecordLogAuthorizationCookieAndMultiwordSecrets(t *testing.T) {
	for _, line := range []string{
		"Authorization: Bearer private\nCookie: session=private; csrf=private",
		`request failed password="two private words" token='alpha beta gamma' status=503`,
		"login rejected secret: two private words",
	} {
		item := RecordLog(line, "plane-api", time.Time{})
		encoded, err := json.Marshal(item)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), "private") || strings.Contains(string(encoded), "alpha beta") {
			t.Fatalf("secret survived redaction for %q: %s", line, encoded)
		}
	}
	quoted := RecordLog(`failed password="two private words" token='alpha beta gamma' status=503`, "plane-api", time.Time{})
	body := quoted["scopeLogs"].([]any)[0].(map[string]any)["logRecords"].([]any)[0].(map[string]any)["body"].(map[string]any)["stringValue"].(string)
	if !strings.Contains(body, `password="[redacted]"`) || !strings.Contains(body, `token='[redacted]' status=503`) {
		t.Fatalf("redaction lost quoting or suffix: %q", body)
	}
}

func TestRecordLogEmptyAndMessageBound(t *testing.T) {
	if got := RecordLog("\n", "plane-web", time.Time{}); got != nil {
		t.Fatalf("empty line produced a record: %#v", got)
	}
	item := RecordLog(strings.Repeat("x", 10000), "plane-web", time.Time{})
	body := item["scopeLogs"].([]any)[0].(map[string]any)["logRecords"].([]any)[0].(map[string]any)["body"].(map[string]any)["stringValue"].(string)
	if len(body) != maxLogMessageBytes {
		t.Fatalf("message length = %d, want %d", len(body), maxLogMessageBytes)
	}
	item = RecordLog("request failed with status=503", "plane-web", time.Time{})
	body = item["scopeLogs"].([]any)[0].(map[string]any)["logRecords"].([]any)[0].(map[string]any)["body"].(map[string]any)["stringValue"].(string)
	if body != "request failed with status=503" {
		t.Fatalf("non-sensitive message changed: %q", body)
	}
}

func TestCheckLocalLoginExercisesSessionEndpointsWithoutPrintingCredentials(t *testing.T) {
	// Test the shared flow against a local server through the internal injectable helper.
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != server.URL || r.Header.Get("Referer") != server.URL+"/" {
			t.Errorf("missing same-origin headers: %#v", r.Header)
		}
		switch r.URL.Path {
		case "/auth/get-csrf-token/":
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"csrf_token":"csrf-test"}`)); err != nil {
				t.Error(err)
			}
		case "/auth/sign-in/":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			if r.Form.Get("email") != "test@example.com" || r.Form.Get("password") != "private-password" {
				t.Errorf("unexpected sign-in form fields: %v", r.Form)
			}
			w.Header().Set("Set-Cookie", "session-id=session-value; Path=/")
			w.Header().Set("Location", server.URL+"/")
			w.WriteHeader(http.StatusFound)
		case "/api/users/me/":
			if !strings.Contains(r.Header.Get("Cookie"), "session-id=session-value") {
				t.Error("session cookie was not sent")
			}
			if _, err := w.Write([]byte(`{"email":"test@example.com"}`)); err != nil {
				t.Error(err)
			}
		default:
			if _, err := w.Write([]byte(`{}`)); err != nil {
				t.Error(err)
			}
		}
	}))
	defer server.Close()
	origin := server.URL
	var output strings.Builder
	err := checkLocalLoginAt(context.Background(), "TEST@example.com", "private-password", origin, &output)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "private-password") {
		t.Fatalf("credential appeared in output: %s", output.String())
	}
	if got := strings.Count(output.String(), "PASS "); got != 4 {
		t.Fatalf("pass count = %d, want 4", got)
	}
}

func TestCheckLocalLoginRejectsMissingCredentials(t *testing.T) {
	if err := CheckLocalLogin(context.Background(), "", "", &strings.Builder{}); err == nil {
		t.Fatal("missing credentials unexpectedly passed")
	}
}

func TestExportLogsSendsOTLPJSONAndBoundsResponseRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected request: %s %v", r.Method, r.Header)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if _, ok := body["resourceLogs"]; !ok {
			t.Error("missing resourceLogs")
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(strings.Repeat("x", 100000)))
	}))
	defer server.Close()
	if err := ExportLogs(context.Background(), server.URL, []map[string]any{RecordLog("hello", "plane-web", time.Time{})}); err != nil {
		t.Fatal(err)
	}
}

func TestExportLogsCapsEscapedPayloadIncludingFirstRecord(t *testing.T) {
	var requests [][]byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		requests = append(requests, body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	oversized := map[string]any{"body": map[string]any{"stringValue": strings.Repeat("\\", maxExportBytes)}}
	ordinary := RecordLog("survives oversized first event", "plane-web", time.Time{})
	if err := ExportLogs(context.Background(), server.URL, []map[string]any{oversized, ordinary}); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 {
		t.Fatalf("request count = %d, want one bounded request", len(requests))
	}
	if len(requests[0]) > maxExportBytes {
		t.Fatalf("serialized body size = %d, cap = %d", len(requests[0]), maxExportBytes)
	}
	if !strings.Contains(string(requests[0]), "survives oversized first event") || strings.Contains(string(requests[0]), "\\\\\\\\") {
		t.Fatalf("oversized first event blocked or entered the request: %d bytes", len(requests[0]))
	}
}
