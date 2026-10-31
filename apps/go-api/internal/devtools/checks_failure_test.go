package devtools

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLocalHTTPChecksRejectTruncatedSuccessfulResponses(t *testing.T) {
	for _, scenario := range []struct {
		name  string
		check func(context.Context, string) error
	}{
		{"logging probe", func(ctx context.Context, endpoint string) error {
			return localPost(ctx, endpoint, []byte(`{}`), time.Second)
		}},
		{"log export", func(ctx context.Context, endpoint string) error {
			return ExportLogs(ctx, endpoint, []map[string]any{{"scopeLogs": []any{}}})
		}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "128")
				if _, err := io.WriteString(w, `{}`); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			if err := scenario.check(t.Context(), server.URL); !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("truncated successful response error = %v, want unexpected EOF", err)
			}
		})
	}
}

func TestLocalLoginCheckRejectsTruncatedResponses(t *testing.T) {
	for _, path := range []string{"/auth/get-csrf-token/", "/api/users/me/", "/api/users/me/profile/", "/api/users/me/settings/", "/api/users/me/workspaces/"} {
		t.Run(path, func(t *testing.T) {
			server := loginFailureServer(t, path)
			var output strings.Builder
			err := checkLocalLoginAt(t.Context(), "test@example.com", "test-password", server.URL, &output)
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("truncated response error = %v, want unexpected EOF", err)
			}
			if strings.Contains(output.String(), "PASS "+path) {
				t.Fatal("truncated response was reported as passing")
			}
		})
	}
}

func TestLocalLoginCheckReportsFailedOutput(t *testing.T) {
	server := loginFailureServer(t, "")
	err := checkLocalLoginAt(t.Context(), "test@example.com", "test-password", server.URL, failedRunnerOutput{})
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("failed output error = %v, want closed pipe", err)
	}
}

func loginFailureServer(t *testing.T, truncatePath string) *httptest.Server {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := `{}`
		switch r.URL.Path {
		case "/auth/get-csrf-token/":
			body = `{"csrf_token":"test-csrf"}`
		case "/auth/sign-in/":
			http.SetCookie(w, &http.Cookie{Name: "session-id", Value: "test-session", Path: "/"})
			w.Header().Set("Location", server.URL+"/")
			w.WriteHeader(http.StatusFound)
			return
		case "/api/users/me/":
			body = `{"email":"test@example.com"}`
		}
		if r.URL.Path == truncatePath {
			w.Header().Set("Content-Length", "128")
		}
		if _, err := io.WriteString(w, body); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	return server
}
