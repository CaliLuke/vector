//go:build integration

package backgroundtasks

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
)

type linkPreviewLostActivityResponse struct {
	*PostgresLinkPreviewStore
	failed atomic.Bool
}

func (store *linkPreviewLostActivityResponse) Crawl(ctx context.Context, input LinkPreviewInput) (struct{}, error) {
	result, err := store.PostgresLinkPreviewStore.Crawl(ctx, input)
	if err == nil && store.failed.CompareAndSwap(false, true) {
		return struct{}{}, errors.New("simulate lost activity response after committed link metadata")
	}
	return result, err
}

func TestPostgresLinkPreviewDurableRetryAndStaleRowGuards(t *testing.T) {
	_, executionURL, pool := openBackgroundTaskDatabase(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()

	ownerID, workspaceID, projectID, issueID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	retryLinkID, staleLinkID, deletedLinkID := uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	username := "link-preview-" + ownerID.String()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,password,username,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ($1,'!', $2,'Link Preview Test','','','',$3,$3,$3,'','',false,false,false,true,false,true,false,false,$2,'','','','',false,'UTC',true)`, ownerID, username, now); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	workspaceSlug := "lp-" + ownerID.String()[:8]
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES ($1,'Link Preview Test',$2,$3,'UTC','',$4,$4)`, workspaceID, ownerID, workspaceSlug, now); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id,name,description,network,workspace_id,identifier,created_by_id,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at) VALUES ($1,'Link Preview Test','',0,$2,$3,$4,false,false,false,true,false,false,false,false,0,0,'{}'::jsonb,'UTC',$5,$5)`, projectID, workspaceID, "LP"+ownerID.String()[:4], ownerID, now); err != nil {
		t.Fatalf("insert project: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO issues (created_at,updated_at,id,name,description_json,priority,sequence_id,project_id,workspace_id,description_html,sort_order,is_draft) VALUES ($1,$1,$2,'Link Preview Issue','{}'::jsonb,'medium',1,$3,$4,'',0,false)`, now, issueID, projectID, workspaceID); err != nil {
		t.Fatalf("insert issue: %v", err)
	}
	defer func() {
		cleanup, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanup, `DELETE FROM issue_links WHERE id=ANY($1::uuid[])`, []uuid.UUID{retryLinkID, staleLinkID, deletedLinkID})
		_, _ = pool.Exec(cleanup, `DELETE FROM issues WHERE id=$1`, issueID)
		_, _ = pool.Exec(cleanup, `DELETE FROM projects WHERE id=$1`, projectID)
		_, _ = pool.Exec(cleanup, `DELETE FROM workspaces WHERE id=$1`, workspaceID)
		_, _ = pool.Exec(cleanup, `DELETE FROM users WHERE id=$1`, ownerID)
	}()

	const host = "link-preview.example.test"
	retryURL := "http://" + host + "/retry"
	staleURL := "http://" + host + "/stale"
	deletedURL := "http://" + host + "/deleted"
	const editedURL = "https://edited.example.test/new-target"
	if _, err := pool.Exec(ctx, `INSERT INTO issue_links (created_at,updated_at,id,title,url,created_by_id,issue_id,project_id,workspace_id,metadata) VALUES ($1,$1,$2,NULL,$3,$4,$5,$6,$7,$8::jsonb),($1,$1,$9,NULL,$10,$4,$5,$6,$7,$11::jsonb),($1,$1,$12,NULL,$13,$4,$5,$6,$7,$14::jsonb)`,
		now, retryLinkID, retryURL, ownerID, issueID, projectID, workspaceID, `{}`, staleLinkID, staleURL, `{"keep":"before-edit"}`, deletedLinkID, deletedURL, `{"keep":"before-delete"}`); err != nil {
		t.Fatalf("insert issue links: %v", err)
	}

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.Host != host {
			t.Errorf("request Host=%q, want %q", request.Host, host)
		}
		if request.UserAgent() != linkPreviewUserAgent {
			t.Errorf("User-Agent=%q, want link preview agent", request.UserAgent())
		}
		switch request.URL.Path {
		case "/retry":
			_, _ = writer.Write([]byte(`<html><title>Retry Preview</title><link rel="icon" href="/icon.png"></html>`))
		case "/stale":
			if _, err := pool.Exec(request.Context(), `UPDATE issue_links SET url=$2,metadata='{"keep":"newer-edit"}'::jsonb WHERE id=$1`, staleLinkID, editedURL); err != nil {
				t.Errorf("edit stale link during crawl: %v", err)
				http.Error(writer, "database edit failed", http.StatusInternalServerError)
				return
			}
			_, _ = writer.Write([]byte(`<html><title>Stale Preview</title><link rel="icon" href="/icon.png"></html>`))
		case "/deleted":
			if _, err := pool.Exec(request.Context(), `UPDATE issue_links SET deleted_at=NOW(),metadata='{"keep":"deleted-row"}'::jsonb WHERE id=$1`, deletedLinkID); err != nil {
				t.Errorf("soft-delete link during crawl: %v", err)
				http.Error(writer, "database delete failed", http.StatusInternalServerError)
				return
			}
			_, _ = writer.Write([]byte(`<html><title>Deleted Preview</title><link rel="icon" href="/icon.png"></html>`))
		case "/icon.png":
			writer.Header().Set("Content-Type", "image/png")
			_, _ = writer.Write([]byte("icon"))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	resolver := &previewResolver{addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8")}}
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != "8.8.8.8:80" {
			return nil, fmt.Errorf("link preview dial used unvalidated address %q", address)
		}
		return (&net.Dialer{}).DialContext(ctx, network, strings.TrimPrefix(server.URL, "http://"))
	}
	store, err := NewPostgresLinkPreviewStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	store.resolver, store.dial = resolver, dial

	// The first activity commits metadata, then loses its response. Selvedge
	// retries from durable execution state; the second update remains idempotent.
	retryStore := &linkPreviewLostActivityResponse{PostgresLinkPreviewStore: store}
	app := selvedge.NewApp(selvedge.WithDatabase(executionURL), selvedge.WithServiceName("plane-link-preview-integration-test"), selvedge.WithActivityPollInterval(10*time.Millisecond), selvedge.WithWorkflowResumptionInterval(10*time.Millisecond), selvedge.WithTimerCheckInterval(10*time.Millisecond))
	dispatcher, err := RegisterLinkPreviewWorkflow(app, retryStore, &retry.Policy{MaxAttempts: 2, InitialInterval: 10 * time.Millisecond, MaxInterval: 10 * time.Millisecond, Multiplier: 1, RandomizationFactor: 0})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		shutdown, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := app.Shutdown(shutdown); err != nil {
			t.Errorf("shutdown link preview worker: %v", err)
		}
	}()
	instanceID, err := dispatcher.Admit(ctx, LinkPreviewInput{ID: retryLinkID.String(), URL: retryURL})
	if err != nil {
		t.Fatalf("admit link preview: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, instanceID)
	if !retryStore.failed.Load() {
		t.Fatal("lost committed activity response was not exercised")
	}
	var title *string
	var metadataURL string
	var faviconURL *string
	if err := pool.QueryRow(ctx, `SELECT metadata->>'title',metadata->>'url',metadata->>'favicon_url' FROM issue_links WHERE id=$1`, retryLinkID).Scan(&title, &metadataURL, &faviconURL); err != nil {
		t.Fatalf("read retried link metadata: %v", err)
	}
	if title == nil || *title != "Retry Preview" || metadataURL != retryURL || faviconURL == nil || *faviconURL != "http://"+host+"/icon.png" {
		t.Fatalf("retried metadata title=%v URL=%q favicon=%v", title, metadataURL, faviconURL)
	}
	if requests.Load() < 4 {
		t.Fatalf("lost response retry made only %d HTTP requests; expected at least two complete crawls", requests.Load())
	}

	// Editing a URL while its response is in flight must preserve the newer URL
	// and metadata. A row deleted during its crawl must retain its tombstone.
	for _, test := range []struct {
		id  uuid.UUID
		url string
	}{
		{staleLinkID, staleURL},
		{deletedLinkID, deletedURL},
	} {
		if _, err := store.Crawl(ctx, LinkPreviewInput{ID: test.id.String(), URL: test.url}); err != nil {
			t.Fatalf("crawl concurrent edit %s: %v", test.id, err)
		}
	}
	var currentURL string
	var staleMetadata string
	if err := pool.QueryRow(ctx, `SELECT url,metadata->>'keep' FROM issue_links WHERE id=$1`, staleLinkID).Scan(&currentURL, &staleMetadata); err != nil || currentURL != editedURL || staleMetadata != "newer-edit" {
		t.Fatalf("stale crawl overwrote edited row: URL=%q metadata=%q err=%v", currentURL, staleMetadata, err)
	}
	var deletedAt *time.Time
	var deletedMetadata string
	if err := pool.QueryRow(ctx, `SELECT deleted_at,metadata->>'keep' FROM issue_links WHERE id=$1`, deletedLinkID).Scan(&deletedAt, &deletedMetadata); err != nil || deletedAt == nil || deletedMetadata != "deleted-row" {
		t.Fatalf("crawl modified deleted row: deleted_at=%v metadata=%q err=%v", deletedAt, deletedMetadata, err)
	}
}
