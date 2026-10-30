//go:build integration

package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
)

type seedLostResponseStore struct {
	*PostgresWorkspaceSeedStore
	failed atomic.Bool
}

func (s *seedLostResponseStore) Seed(ctx context.Context, in WorkspaceSeedInput) (struct{}, error) {
	out, err := s.PostgresWorkspaceSeedStore.Seed(ctx, in)
	if err == nil && s.failed.CompareAndSwap(false, true) {
		return out, errors.New("lost committed seed response")
	}
	return out, err
}
func TestPostgresWorkspaceSeedAtomicRetryAndSourceContent(t *testing.T) {
	_, executionURL, pool := openBackgroundTaskDatabase(t)
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	owner, member, workspace := uuid.NewString(), uuid.NewString(), uuid.NewString()
	at := time.Date(2026, 10, 22, 8, 30, 0, 0, time.UTC)
	records, err := decodeSeedRecords()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if rollbackErr := dbutil.RollbackTransaction(ctx, tx.Rollback, nil); rollbackErr != nil {
			t.Errorf("rollback test transaction: %v", rollbackErr)
		}
	}()
	for _, id := range []string{owner, member} {
		row := records.Rows["users"][0]
		row["id"] = id
		row["username"] = id
		row["email"] = id + "@example.test"
		row["is_bot"] = false
		row["bot_type"] = nil
		if err = insertSeedRow(ctx, tx, "users", row); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO workspaces(id,name,slug,owner_id,timezone,background_color,created_at,updated_at) VALUES($1::uuid,'ßabcd workspace',$1::text,$2,'America/Los_Angeles','',$3,$3)`, workspace, owner, at); err != nil {
		t.Fatal(err)
	}
	for _, m := range []struct {
		id     string
		role   int
		active bool
	}{{owner, 20, true}, {member, 5, false}} {
		row := records.Rows["workspace_members"][0]
		row["id"] = uuid.NewString()
		row["workspace_id"] = workspace
		row["member_id"] = m.id
		row["role"] = m.role
		row["is_active"] = m.active
		if err = insertSeedRow(ctx, tx, "workspace_members", row); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	base, err := NewPostgresWorkspaceSeedStore(pool, "https://plane.example.test:8000")
	if err != nil {
		t.Fatal(err)
	}
	// A late database error must leave no earlier seed rows behind.
	if _, err = pool.Exec(ctx, `CREATE FUNCTION public.reject_seed_page() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected seed failure'; END $$; CREATE TRIGGER reject_seed_page BEFORE INSERT ON public.pages FOR EACH ROW EXECUTE FUNCTION public.reject_seed_page()`); err != nil {
		t.Fatal(err)
	}
	_, seedErr := base.Seed(ctx, WorkspaceSeedInput{workspace, at})
	if _, err = pool.Exec(ctx, `DROP TRIGGER reject_seed_page ON public.pages; DROP FUNCTION public.reject_seed_page()`); err != nil {
		t.Fatal(err)
	}
	if seedErr == nil {
		t.Fatal("injected failure did not fail")
	}
	var n int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM projects WHERE workspace_id=$1`, workspace).Scan(&n); err != nil || n != 0 {
		t.Fatalf("partial seed committed %d %v", n, err)
	}
	store := &seedLostResponseStore{PostgresWorkspaceSeedStore: base}
	app := selvedge.NewApp(selvedge.WithDatabase(executionURL), selvedge.WithServiceName("workspace-seed-proof"), selvedge.WithActivityPollInterval(10*time.Millisecond), selvedge.WithWorkflowResumptionInterval(10*time.Millisecond))
	dispatcher, err := RegisterWorkspaceSeedWorkflow(app, store, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
	if err != nil {
		t.Fatal(err)
	}
	if err = app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		stopCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if shutdownErr := app.Shutdown(stopCtx); shutdownErr != nil {
			t.Error(shutdownErr)
		}
	}()
	instance, err := dispatcher.Admit(ctx, WorkspaceSeedInput{workspace, at})
	if err != nil {
		t.Fatal(err)
	}
	waitVersionWorkflow(t, ctx, app, instance)
	if !store.failed.Load() {
		t.Fatal("retry not exercised")
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM issues WHERE workspace_id=$1 AND custom_properties='{}'::jsonb`, workspace).Scan(&n); err != nil || n != 7 {
		t.Fatalf("seeded issue property defaults: count=%d want=7 err=%v", n, err)
	}
	for table, want := range map[string]int{"projects": 1, "states": 5, "issues": 7, "issue_sequences": 7, "cycles": 2, "modules": 3, "pages": 2, "project_members": 3, "project_user_properties": 3, "workspace_members": 3, "issue_activities": 7, "module_issues": 11} {
		if err = pool.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE workspace_id=$1", workspace).Scan(&n); err != nil || n != want {
			t.Errorf("%s count=%d want=%d err=%v", table, n, want, err)
		}
	}
	var identifier, zone string
	if err = pool.QueryRow(ctx, `SELECT identifier,timezone FROM projects WHERE workspace_id=$1`, workspace).Scan(&identifier, &zone); err != nil || identifier != "SSABCD" || zone != "America/Los_Angeles" {
		t.Fatalf("project %q %q %v", identifier, zone, err)
	}
	var role int
	if err = pool.QueryRow(ctx, `SELECT role FROM project_members WHERE workspace_id=$1 AND member_id=$2`, workspace, member).Scan(&role); err != nil || role != 5 {
		t.Fatalf("member role %d %v", role, err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM issue_sequences s JOIN issues i ON i.id=s.issue_id WHERE s.workspace_id=$1 AND s.sequence=i.sequence_id`, workspace).Scan(&n); err != nil || n != 7 {
		t.Fatalf("sequence mismatch %d %v", n, err)
	}
	var start, end time.Time
	if err = pool.QueryRow(ctx, `SELECT start_date,end_date FROM cycles WHERE workspace_id=$1 ORDER BY start_date LIMIT 1`, workspace).Scan(&start, &end); err != nil || !start.Equal(at) || !end.Equal(at.Add(14*24*time.Hour)) {
		t.Fatalf("cycle dates %s %s %v", start, end, err)
	}
	for _, source := range records.Rows["pages"] {
		var html string
		var raw []byte
		if err = pool.QueryRow(ctx, `SELECT description_html,description_json FROM pages WHERE workspace_id=$1 AND name=$2`, workspace, source["name"]).Scan(&html, &raw); err != nil {
			t.Fatal(err)
		}
		var gotJSON any
		if err = json.Unmarshal(raw, &gotJSON); err != nil {
			t.Fatal(err)
		}
		expectedJSON, _ := json.Marshal(source["description_json"])
		actualJSON, _ := json.Marshal(gotJSON)
		if html != source["description_html"] || string(actualJSON) != string(expectedJSON) {
			t.Fatalf("seed page content changed: %s", source["name"])
		}
	}
}
