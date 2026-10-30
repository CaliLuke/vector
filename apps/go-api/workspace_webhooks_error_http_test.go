package systemapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workspacewebhooks"
	server "github.com/CaliLuke/go-plane/workspace_webhooks/gen/http/workspace_webhooks/server"
	api "github.com/CaliLuke/go-plane/workspace_webhooks/gen/workspace_webhooks"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type webhookFailureDatabase struct {
	cause         error
	denied        bool
	reads, writes int
}
type webhookFailureRow func(...any) error

func (row webhookFailureRow) Scan(destinations ...any) error { return row(destinations...) }
func (db *webhookFailureDatabase) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	if strings.Contains(query, "SELECT EXISTS") {
		db.reads++
		return webhookFailureRow(func(destinations ...any) error { *destinations[0].(*bool) = !db.denied; return nil })
	}
	if strings.Contains(query, "INSERT INTO webhooks") || strings.Contains(query, "UPDATE webhooks") {
		db.writes++
		return webhookFailureRow(func(...any) error { return db.cause })
	}
	db.reads++
	return webhookFailureRow(func(destinations ...any) error { *destinations[0].(*[]byte) = []byte(`{}`); return nil })
}
func (*webhookFailureDatabase) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("unexpected write")
}

func TestWebhookWriteFailuresRetainSanitizedHTTPContracts(t *testing.T) {
	t.Setenv("WEBHOOK_ALLOWED_HOSTS", "")
	t.Setenv("WEBHOOK_ALLOWED_IPS", "")
	t.Setenv("WEBHOOK_DISALLOWED_DOMAINS", "")
	for _, method := range []string{http.MethodPost, http.MethodPatch} {
		for _, tc := range []struct {
			name   string
			cause  error
			denied bool
			status int
			body   string
		}{
			{"integrity", &pgconn.PgError{Code: "23505", ConstraintName: "other_unique", Message: "private database diagnostic"}, false, 400, `{"error":"The payload is not valid"}`},
			{"duplicate URL", &pgconn.PgError{Code: "23505", ConstraintName: "webhook_url_unique_url_when_deleted_at_null", Message: "private duplicate diagnostic"}, false, 400, `{"error":"The payload is not valid"}`},
			{"database failure", &pgconn.PgError{Code: "08006", Message: "private connection diagnostic"}, false, 500, `{"error":"Something went wrong please try again later"}`},
			{"permission denied", errors.New("must not write"), true, 403, `{"error":"You don't have the required permissions."}`},
		} {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				db := &webhookFailureDatabase{cause: fmt.Errorf("write: %w", tc.cause), denied: tc.denied}
				repository, err := store.NewRepository(db)
				require.NoError(t, err)
				sessions := &activitySessionStub{principal: djangosession.Principal{UserID: "00000000-0000-4000-8000-000000034201", UserTimezone: "UTC"}}
				service, err := NewWorkspaceWebhooks(repository, sessions, unusedWebhookPublisher{})
				require.NoError(t, err)
				mux := loomhttp.NewMuxer()
				handler := server.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
				metadata, err := loomhttp.NewRequestMetadataPolicy([]string{"Content-Type", "Cookie"}, nil)
				require.NoError(t, err)
				handler.Use(loomhttp.RequestMetadataMiddleware(metadata))
				server.Mount(mux, handler)
				path := "/api/workspaces/demo/webhooks/"
				body := `{"url":"https://1.1.1.1/webhook"}`
				if method == http.MethodPatch {
					path += "00000000-0000-4000-8000-000000034202/"
					body = `{}`
				}
				request := httptest.NewRequest(method, path, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "test-session"})
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, request)
				wantStatus, wantBody := tc.status, tc.body
				if method == http.MethodPost && tc.name == "duplicate URL" {
					wantStatus, wantBody = 409, `{"error":"URL already exists for the workspace"}`
				}
				require.Equal(t, wantStatus, response.Code, response.Body.String())
				require.JSONEq(t, wantBody, response.Body.String())
				if tc.denied {
					require.Zero(t, db.writes)
					require.Equal(t, 1, db.reads)
				} else {
					require.Equal(t, 1, db.writes)
					expectedReads := 1
					if method == http.MethodPatch {
						expectedReads++
					}
					require.Equal(t, expectedReads, db.reads)
				}
			})
		}
	}
}
