//go:build integration

package systemapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/database"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgxpool"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/instancecipher"
	oauth "github.com/CaliLuke/go-plane/internal/mcpoauth"
	server "github.com/CaliLuke/go-plane/mcpoauthlinks/gen/http/mcp_oauth_links/server"
	api "github.com/CaliLuke/go-plane/mcpoauthlinks/gen/mcp_oauth_links"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestGeneratedLinkHTTPUsesCurrentSignedInSession(t *testing.T) {
	db := mcpLinkHTTPDatabase(t)
	_, err := db.Exec(t.Context(), `ALTER TABLE users ADD COLUMN password text NOT NULL DEFAULT 'test-password-hash';
 CREATE TABLE sessions(session_key varchar(128) PRIMARY KEY,session_data text NOT NULL,expire_date timestamptz NOT NULL,user_id varchar(50),device_info jsonb);`)
	if err != nil {
		t.Fatal(err)
	}
	validator, err := djangosession.New(djangosession.NewPostgreSQLReader(db), djangosession.Config{SecretKey: "test-only-session-signing-key", AllowedBackends: []string{"django.contrib.auth.backends.ModelBackend"}})
	if err != nil {
		t.Fatal(err)
	}
	login, err := validator.Login(t.Context(), nil, djangosession.AuthenticatedSession{UserID: "10000000-0000-4000-8000-000000000001", PasswordHash: "test-password-hash", Backend: "django.contrib.auth.backends.ModelBackend"}, 3600, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	issuer := mcpLinkHTTPTestIssuer{}
	links, err := oauth.NewLinks(db, "https://issuer.example")
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := instancecipher.New("test-only-instance-key")
	if err != nil {
		t.Fatal(err)
	}
	linker, err := oauth.NewLinking(links, issuer, cipher)
	if err != nil {
		t.Fatal(err)
	}
	service, err := systemapi.NewMCPOAuthLinks(linker, validator, "https://issuer.example", "https://vector.example/mcp")
	if err != nil {
		t.Fatal(err)
	}
	mux := loomhttp.NewMuxer()
	generated := server.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	metadata, err := loomhttp.NewRequestMetadataPolicy([]string{"Cookie"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	generated.Use(loomhttp.RequestMetadataMiddleware(metadata))
	server.Mount(mux, generated)
	sessionCookie := &http.Cookie{Name: "session-id", Value: login.SessionKey}
	call := func(method, path, contentType, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, "https://vector.example"+path, strings.NewReader(body))
		request.Header.Set("Content-Type", contentType)
		for _, cookie := range cookies {
			request.AddCookie(cookie)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		return response
	}
	if unauth := call("GET", "/api/users/me/mcp-link/", "", ""); unauth.Code != 401 {
		t.Fatalf("anonymous status=%d", unauth.Code)
	}
	start := call("POST", "/api/users/me/mcp-link/start/", "", "", sessionCookie)
	if start.Code != 200 {
		t.Fatalf("start status=%d body=%s", start.Code, start.Body)
	}
	var started struct {
		AuthorizationURL string `json:"authorization_url"`
	}
	if err = json.Unmarshal(start.Body.Bytes(), &started); err != nil {
		t.Fatal(err)
	}
	authorization, err := url.Parse(started.AuthorizationURL)
	if err != nil {
		t.Fatal(err)
	}
	query := authorization.Query()
	callback := call("POST", "/auth/mcp/callback/", "application/x-www-form-urlencoded", url.Values{"state": {query.Get("state")}, "code": {"one-time-code"}}.Encode())
	if callback.Code != 303 || callback.Header().Get("Location") != "https://vector.example/settings/profile/api-tokens/" {
		t.Fatalf("callback status=%d", callback.Code)
	}
	response := callback.Result()
	proofs := response.Cookies()
	testsupport.Close(t, response.Body)
	if len(proofs) != 1 || !proofs[0].Secure || !proofs[0].HttpOnly || proofs[0].SameSite != http.SameSiteStrictMode || proofs[0].Domain != "" || proofs[0].Path != "/" {
		t.Fatal("callback browser proof cookie is not protected")
	}
	proof := proofs[0]
	withoutProof := call("GET", "/api/users/me/mcp-link/", "", "", sessionCookie)
	var hidden struct {
		Pending any `json:"pending"`
	}
	if err = json.Unmarshal(withoutProof.Body.Bytes(), &hidden); err != nil || hidden.Pending != nil {
		t.Fatalf("pending identity leaked without browser proof: %v", err)
	}
	status := call("GET", "/api/users/me/mcp-link/", "", "", sessionCookie, proof)
	var pending struct {
		Pending *struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
		} `json:"pending"`
	}
	if err = json.Unmarshal(status.Body.Bytes(), &pending); err != nil || pending.Pending == nil || pending.Pending.Subject != "person-1" {
		t.Fatalf("pending identity missing: %v", err)
	}
	confirm := call("POST", "/api/users/me/mcp-link/confirm/", "application/json", `{"id":"`+pending.Pending.ID+`"}`, sessionCookie, proof)
	if confirm.Code != 204 {
		t.Fatalf("confirm status=%d", confirm.Code)
	}
	linked, err := links.Resolve(t.Context(), "person-1")
	if err != nil || linked.ID.String() != "10000000-0000-4000-8000-000000000001" {
		t.Fatalf("confirmed identity missing: %v", err)
	}
	if _, err = db.Exec(t.Context(), `DELETE FROM sessions WHERE session_key=$1`, login.SessionKey); err != nil {
		t.Fatal(err)
	}
	if revoked := call("GET", "/api/users/me/mcp-link/", "", "", sessionCookie, proof); revoked.Code != 401 {
		t.Fatalf("revoked browser session status=%d", revoked.Code)
	}
}

type mcpLinkHTTPTestIssuer struct{}

func (mcpLinkHTTPTestIssuer) AuthorizationURL(state, _, _ string) string {
	return "https://issuer.example/authorize?state=" + url.QueryEscape(state)
}
func (mcpLinkHTTPTestIssuer) ExchangeSubject(_ context.Context, code, _ string, _ [32]byte) (string, error) {
	if code != "one-time-code" {
		return "", oauth.ErrInvalidCredential
	}
	return "person-1", nil
}

func mcpLinkHTTPDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	schema, err := os.ReadFile("internal/bootstrap/schema_migrations/000015_mcp_oauth_links.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	ctx, connection := postgres.OpenSQL(t, `CREATE TABLE users(id uuid PRIMARY KEY,is_active boolean NOT NULL,is_bot boolean NOT NULL,user_timezone text NOT NULL);
 INSERT INTO users VALUES ('10000000-0000-4000-8000-000000000001',true,false,'UTC');`+string(schema))
	config, err := pgxpool.ParseConfig(connection.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	if err = database.ConfigurePool(config); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
