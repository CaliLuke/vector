//go:build integration

package backgroundtasks

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
)

func TestPostgresDescriptionAssetCopyUsesDurableWorkflowAndStableS3Key(t *testing.T) {
	_, executionURL, pool := openBackgroundTaskDatabase(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	ownerID, workspaceID, projectID := uuid.New(), uuid.New(), uuid.New()
	sourcePageID, targetPageID, assetID := uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC()
	markup := `<p>Image <image-component src="` + assetID.String() + `" alt="example"></image-component></p>`
	username := "description-assets-" + ownerID.String()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,password,username,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ($1,'!', $2,'Asset Test','Asset','Test','',$3,$3,$3,'','',false,false,false,true,false,true,false,false,$2,'','','','',false,'UTC',true)`, ownerID, username, now); err != nil {
		t.Fatalf("insert asset-copy owner: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES ($1,'Asset Copy',$2,$3,'UTC','',$4,$4)`, workspaceID, ownerID, "ac-"+ownerID.String()[:8], now); err != nil {
		t.Fatalf("insert asset-copy workspace: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id,name,description,network,workspace_id,identifier,created_by_id,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at) VALUES ($1,'Asset Copy','',0,$2,$3,$4,false,false,false,true,false,false,false,false,0,0,'{}'::jsonb,'UTC',$5,$5)`, projectID, workspaceID, "AC"+ownerID.String()[:3], ownerID, now); err != nil {
		t.Fatalf("insert asset-copy project: %v", err)
	}
	for _, page := range []struct {
		id   uuid.UUID
		name string
		html string
	}{
		{sourcePageID, "Original", `<p>original</p>`},
		{targetPageID, "Duplicate", markup},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO pages (created_at,updated_at,id,name,description_json,description_html,access,created_by_id,owned_by_id,workspace_id,color,is_locked,view_props,logo_props,is_global,sort_order) VALUES ($1,$1,$2,$3,'{}'::jsonb,$4,0,$5,$5,$6,'',false,'{}'::jsonb,'{}'::jsonb,false,0)`, now, page.id, page.name, page.html, ownerID, workspaceID); err != nil {
			t.Fatalf("insert %s page: %v", page.name, err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO file_assets (created_at,updated_at,id,attributes,asset,created_by_id,workspace_id,is_deleted,is_archived,entity_type,is_uploaded,page_id,project_id,size,storage_metadata) VALUES ($1,$1,$2,'{"name":"image.png","type":"image/png"}'::jsonb,$3,$4,$5,false,false,'PAGE_DESCRIPTION',true,$6,$7,42,'{"etag":"source"}'::jsonb)`, now, assetID, workspaceID.String()+"/source-image.png", ownerID, workspaceID, sourcePageID, projectID); err != nil {
		t.Fatalf("insert source FileAsset: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM file_assets WHERE id=ANY($1::uuid[]) OR page_id=ANY($2::uuid[])`, []uuid.UUID{assetID, uuid.NewSHA1(targetPageID, assetID[:])}, []uuid.UUID{sourcePageID, targetPageID})
		_, _ = pool.Exec(context.Background(), `DELETE FROM pages WHERE id=ANY($1::uuid[])`, []uuid.UUID{sourcePageID, targetPageID})
		_, _ = pool.Exec(context.Background(), `DELETE FROM projects WHERE id=$1`, projectID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM workspaces WHERE id=$1`, workspaceID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, ownerID)
	}()

	var copies []string
	var copiesMu sync.Mutex
	copier := descriptionObjectCopierFunc(func(_ context.Context, source, destination string) error {
		copiesMu.Lock()
		defer copiesMu.Unlock()
		copies = append(copies, source+" -> "+destination)
		return nil
	})
	converter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/convert-document/" {
			t.Errorf("conversion path=%q", r.URL.Path)
		}
		var request map[string]string
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode conversion request: %v", err)
		}
		if request["variant"] != "rich" || !strings.Contains(request["description_html"], `src="`+uuid.NewSHA1(targetPageID, assetID[:]).String()+`"`) {
			t.Errorf("conversion request=%v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"description_json":{"type":"doc","content":[]},"description_binary":"` + base64.StdEncoding.EncodeToString([]byte("converted")) + `"}`))
	}))
	defer converter.Close()
	store := &PostgresDescriptionAssetCopyStore{pool: pool, storage: copier, client: converter.Client(), liveURL: converter.URL}
	input := DescriptionAssetCopyInput{EventID: targetPageID.String(), EntityName: "PAGE", EntityID: targetPageID.String(), ProjectID: projectID.String(), WorkspaceSlug: "ac-test", UserID: ownerID.String()}
	initial, err := store.Prepare(ctx, input)
	if err != nil || len(initial.Assets) != 1 {
		t.Fatalf("prepare original page image: assets=%d error=%v", len(initial.Assets), err)
	}
	app := selvedge.NewApp(selvedge.WithDatabase(executionURL), selvedge.WithServiceName("plane-description-asset-copy-integration-test"), selvedge.WithActivityPollInterval(20*time.Millisecond), selvedge.WithWorkflowResumptionInterval(20*time.Millisecond), selvedge.WithTimerCheckInterval(20*time.Millisecond))
	dispatcher, err := RegisterDescriptionAssetCopyWorkflow(app, store, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := app.Shutdown(shutdownCtx); err != nil {
			t.Errorf("shutdown description asset copy worker: %v", err)
		}
	}()

	instanceID, err := dispatcher.Admit(ctx, input)
	if err != nil {
		t.Fatalf("admit description asset copy: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, instanceID)
	duplicateID, err := dispatcher.Admit(ctx, input)
	if err != nil {
		t.Fatalf("reconcile description asset copy admission: %v", err)
	}
	if duplicateID != instanceID {
		t.Fatalf("duplicate admission ID=%q, want %q", duplicateID, instanceID)
	}
	if len(copies) != 1 {
		t.Fatalf("S3 copied %d times, want exactly once for completed workflow: %v", len(copies), copies)
	}
	copyAssetID := uuid.NewSHA1(targetPageID, assetID[:])
	var copiedKey string
	var uploaded bool
	var copiedPageID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT asset,is_uploaded,page_id FROM file_assets WHERE id=$1`, copyAssetID).Scan(&copiedKey, &uploaded, &copiedPageID); err != nil {
		t.Fatalf("read copied FileAsset: %v", err)
	}
	if !uploaded || copiedPageID != targetPageID || !strings.Contains(copiedKey, copyAssetID.String()[:8]) {
		t.Fatalf("copied FileAsset key=%q uploaded=%t page=%s", copiedKey, uploaded, copiedPageID)
	}
	var updatedHTML string
	var binary []byte
	var descriptionJSON []byte
	if err := pool.QueryRow(ctx, `SELECT description_html,description_binary,description_json FROM pages WHERE id=$1`, targetPageID).Scan(&updatedHTML, &binary, &descriptionJSON); err != nil {
		t.Fatalf("read rewritten page description: %v", err)
	}
	if !strings.Contains(updatedHTML, `src="`+copyAssetID.String()+`"`) || strings.Contains(updatedHTML, assetID.String()) {
		t.Fatalf("rewritten page HTML=%q", updatedHTML)
	}
	var decodedDescription map[string]any
	if err := json.Unmarshal(descriptionJSON, &decodedDescription); err != nil {
		t.Fatalf("decode converted page description JSON: %v", err)
	}
	if string(binary) != "converted" || decodedDescription["type"] != "doc" {
		t.Fatalf("converted page binary=%q JSON=%s", binary, descriptionJSON)
	}

	// Simulate a retry after S3 accepted the copy but before the activity
	// completion was recorded. The destination key and FileAsset ID stay stable.
	replayed, err := store.Copy(ctx, input, initial.Assets[0])
	if err != nil {
		t.Fatalf("replay committed asset copy: %v", err)
	}
	if replayed.CopyID != copyAssetID.String() || len(copies) != 2 || copies[0] != copies[1] {
		t.Fatalf("replayed asset identity=%s copies=%v", replayed.CopyID, copies)
	}
	var copiedRows int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM file_assets WHERE id=$1`, copyAssetID).Scan(&copiedRows); err != nil || copiedRows != 1 {
		t.Fatalf("replayed copy has %d metadata rows, error=%v; want one", copiedRows, err)
	}
}

type descriptionObjectCopier interface {
	Copy(context.Context, string, string) error
}
type descriptionObjectCopierFunc func(context.Context, string, string) error

func (f descriptionObjectCopierFunc) Copy(ctx context.Context, source, destination string) error {
	return f(ctx, source, destination)
}
