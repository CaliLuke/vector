//go:build integration

package backgroundtasks

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
)

func TestPostgresIssueExportProducesAndStoresArchive(t *testing.T) {
	_, _, pool := openBackgroundTaskDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, workspaceID, projectID, issueID, exporterID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC()
	token := uuid.NewString()
	slug := "export-proof-" + userID.String()[:8]
	username := "issue-export-" + userID.String()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,password,username,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ($1,'!',$2,'Export Actor','Export','Actor','',$3,$3,$3,'','',false,false,false,true,false,true,false,false,$2,'','','','',false,'UTC',true)`, userID, username, now); err != nil {
		t.Fatalf("insert export actor: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES ($1,'Issue Export',$2,$3,'UTC','',$4,$4)`, workspaceID, userID, slug, now); err != nil {
		t.Fatalf("insert export workspace: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspace_members (id,workspace_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,issue_props,explored_features,getting_started_checklist,tips) VALUES ($1,$2,$3,20,true,$4,$4,'{}','{}','{}','{}','{}','{}')`, uuid.New(), workspaceID, userID, now); err != nil {
		t.Fatalf("insert export workspace membership: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at) VALUES ($1,'Export Project','',0,$2,'EXP',false,false,false,true,false,false,false,false,0,0,'{}'::jsonb,'UTC',$3,$3)`, projectID, workspaceID, now); err != nil {
		t.Fatalf("insert export project: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO project_members (id,workspace_id,project_id,member_id,role,view_props,default_props,sort_order,preferences,is_active,created_at,updated_at,deleted_at) VALUES ($1,$2,$3,$4,15,'{}'::jsonb,'{}'::jsonb,0,'{}'::jsonb,true,$5,$5,NULL)`, uuid.New(), workspaceID, projectID, userID, now); err != nil {
		t.Fatalf("insert export project membership: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO issues (id,project_id,workspace_id,created_by_id,name,description_json,description_html,priority,sequence_id,sort_order,is_draft,created_at,updated_at) VALUES ($1,$2,$3,$4,'Export issue','{}'::jsonb,'<p>export</p>','high',27,0,false,$5,$5)`, issueID, projectID, workspaceID, userID, now); err != nil {
		t.Fatalf("insert export issue: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO exporters (id,created_at,updated_at,workspace_id,project,provider,status,reason,key,url,token,initiated_by_id,type) VALUES ($1,$2,$2,$3,$4::uuid[],'csv','queued','','',NULL,$5,$6,'issue_exports')`, exporterID, now, workspaceID, []string{projectID.String()}, token, userID); err != nil {
		t.Fatalf("insert exporter history row: %v", err)
	}

	objects := &issueExportObjectStoreRecorder{url: "https://download.example.test/export.zip"}
	store, err := NewPostgresIssueExportStore(pool, objects)
	if err != nil {
		t.Fatal(err)
	}
	input := IssueExportInput{Provider: "csv", WorkspaceID: workspaceID.String(), ProjectIDs: []string{projectID.String()}, TokenID: token, Slug: slug}
	if generateErr := store.Generate(ctx, input); generateErr != nil {
		t.Fatalf("generate issue export from PostgreSQL: %v", generateErr)
	}

	var status, key, downloadURL string
	if scanErr := pool.QueryRow(ctx, `SELECT status,key,url FROM exporters WHERE id=$1`, exporterID).Scan(&status, &key, &downloadURL); scanErr != nil {
		t.Fatal(scanErr)
	}
	if status != "completed" || key != objects.key || downloadURL != objects.url {
		t.Fatalf("exporter status=%q key=%q url=%q", status, key, downloadURL)
	}
	if !strings.HasPrefix(key, workspaceID.String()+"/export-"+slug+"-") {
		t.Fatalf("stored object key=%q", key)
	}
	archive, err := zip.NewReader(bytes.NewReader(objects.data), int64(len(objects.data)))
	if err != nil {
		t.Fatalf("open uploaded export ZIP: %v", err)
	}
	if len(archive.File) != 1 || archive.File[0].Name != slug+"-"+workspaceID.String()+".csv" {
		t.Fatalf("ZIP entries=%v", archive.File)
	}
	file, err := archive.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(file)
	testsupport.Close(t, file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "Export issue") || !strings.Contains(string(content), "EXP-27") {
		t.Fatalf("CSV content missing issue fields: %s", content)
	}
}

type issueExportObjectStoreRecorder struct {
	key, contentType, url string
	data                  []byte
}

func (recorder *issueExportObjectStoreRecorder) Put(_ context.Context, key, contentType string, data []byte) error {
	recorder.key = key
	recorder.contentType = contentType
	recorder.data = append([]byte(nil), data...)
	return nil
}

func (recorder *issueExportObjectStoreRecorder) PresignGet(_ context.Context, key, _ string, _ *string, _, _ string) (string, error) {
	if key != recorder.key {
		return "", errors.New("pre-signed key differs from uploaded key")
	}
	return recorder.url, nil
}
