//go:build integration

package backgroundtasks

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
)

type retryDeletionStore struct {
	*PostgresDeletionStore
	failed           atomic.Bool
	failPlanning     atomic.Bool
	planningFailures atomic.Int32
	applications     atomic.Int32
}

func (s *retryDeletionStore) Apply(ctx context.Context, plan deletionPlan) ([]DeletionInput, error) {
	s.applications.Add(1)
	children, err := s.PostgresDeletionStore.Apply(ctx, plan)
	if err == nil && len(children) > 0 && s.failed.CompareAndSwap(false, true) {
		return nil, errors.New("simulate lost response after committed child marks")
	}
	return children, err
}
func (s *retryDeletionStore) Plan(ctx context.Context, input deletionPlanInput) (deletionPlan, error) {
	if s.failPlanning.Load() {
		s.planningFailures.Add(1)
		return deletionPlan{}, errors.New("injected planning failure")
	}
	return s.PostgresDeletionStore.Plan(ctx, input)
}

func TestPostgresDeletionDurableCascadeRetryAndRetention(t *testing.T) {
	_, executionURL, pool := openBackgroundTaskDatabase(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	owner, workspace, project, sibling, state := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,password,username,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ($1,'!', $2,'Deletion Test','','','',$3,$3,$3,'','',false,false,false,true,false,true,false,false,$2,'','','','',false,'UTC',true)`, owner, owner.String(), now); err != nil {
		t.Fatal(err)
	}
	for _, id := range []uuid.UUID{workspace, sibling} {
		if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id,name,owner_id,slug,timezone,background_color,created_at,updated_at,created_by_id) VALUES ($1,'Deletion',$2,$3,'UTC','',$4,$4,$2)`, id, owner, id.String(), now); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id,name,description,network,workspace_id,identifier,created_by_id,module_view,cycle_view,issue_views_view,page_view,intake_view,archive_in,close_in,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,logo_props,timezone,created_at,updated_at) VALUES ($1,'Deletion','',0,$2,$3,$4,false,false,false,true,false,0,0,false,false,false,'{}'::jsonb,'UTC',$5,$5)`, project, workspace, "DEL", owner, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO states (created_at,updated_at,id,name,description,color,slug,project_id,workspace_id,sequence,"group","default",is_triage,created_by_id) VALUES ($1,$1,$2,'Delete','','','',$3,$4,1,'backlog',false,false,$5);`, now, state, project, workspace, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE projects SET default_state_id=$2 WHERE id=$1`, project, state); err != nil {
		t.Fatal(err)
	}
	base, err := NewPostgresDeletionStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	store := &retryDeletionStore{PostgresDeletionStore: base}
	app := selvedge.NewApp(selvedge.WithDatabase(executionURL), selvedge.WithServiceName("deletion-proof"), selvedge.WithActivityPollInterval(10*time.Millisecond), selvedge.WithWorkflowResumptionInterval(10*time.Millisecond))
	dispatcher, err := RegisterDeletionWorkflows(app, store, 60, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
	if err != nil {
		t.Fatal(err)
	}
	if startErr := app.Start(ctx); startErr != nil {
		t.Fatal(startErr)
	}
	defer func() {
		stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := app.Shutdown(stop); shutdownErr != nil {
			t.Error(shutdownErr)
		}
	}()
	input := DeletionInput{EventID: uuid.NewString(), App: "db", Model: "workspace", ID: workspace.String()}
	instance, err := dispatcher.Admit(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	waitVersionWorkflow(t, ctx, app, instance)
	if !store.failed.Load() {
		t.Fatal("retry boundary was not exercised")
	}
	for table, id := range map[string]uuid.UUID{"workspaces": workspace, "projects": project, "states": state} {
		var deleted bool
		var creator uuid.UUID
		if scanErr := pool.QueryRow(ctx, "SELECT deleted_at IS NOT NULL,created_by_id FROM "+table+" WHERE id=$1", id).Scan(&deleted, &creator); scanErr != nil || !deleted || creator != owner {
			t.Fatalf("%s deleted=%v creator=%s err=%v", table, deleted, creator, scanErr)
		}
	}
	var untouched bool
	if scanErr := pool.QueryRow(ctx, `SELECT deleted_at IS NULL FROM workspaces WHERE id=$1`, sibling).Scan(&untouched); scanErr != nil || !untouched {
		t.Fatalf("sibling changed: %v %v", untouched, scanErr)
	}
	// A separate state deletion exercises SET NULL even on a live project.
	if _, execErr := pool.Exec(ctx, `UPDATE projects SET deleted_at=NULL,default_state_id=$2 WHERE id=$1;`, project, state); execErr != nil {
		t.Fatal(execErr)
	}
	stateRun, err := dispatcher.Admit(ctx, DeletionInput{EventID: uuid.NewString(), App: "db", Model: "state", ID: state.String()})
	if err != nil {
		t.Fatal(err)
	}
	waitVersionWorkflow(t, ctx, app, stateRun)
	var nulled bool
	if scanErr := pool.QueryRow(ctx, `SELECT default_state_id IS NULL FROM projects WHERE id=$1`, project).Scan(&nulled); scanErr != nil || !nulled {
		t.Fatalf("SET NULL not applied: %v %v", nulled, scanErr)
	}
	// An old root physically cascades to a still-live child. Young soft-deleted
	// siblings survive. This proves native FK behavior, not a mocked collector.
	if _, execErr := pool.Exec(ctx, `UPDATE workspaces SET deleted_at=$2 WHERE id=$1`, workspace, now.Add(-61*24*time.Hour)); execErr != nil {
		t.Fatal(execErr)
	}
	if _, execErr := pool.Exec(ctx, `UPDATE workspaces SET deleted_at=$2 WHERE id=$1`, sibling, now); execErr != nil {
		t.Fatal(execErr)
	}
	hard, err := dispatcher.AdmitHardDelete(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	waitVersionWorkflow(t, ctx, app, hard)
	for table, id := range map[string]uuid.UUID{"workspaces": workspace, "projects": project, "states": state} {
		var count int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE id=$1", id).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s hard deletion count=%d err=%v", table, count, err)
		}
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM workspaces WHERE id=$1`, sibling).Scan(&count); err != nil || count != 1 {
		t.Fatalf("retention removed young row: %d %v", count, err)
	}
	// Missing roots and exhausted planning retries are workflow contracts, not
	// incidental coverage from shutting down an unrelated catalogue worker.
	t.Run("missing root is an idempotent no-op", func(t *testing.T) {
		before := store.applications.Load()
		run, admitErr := dispatcher.Admit(ctx, DeletionInput{EventID: uuid.NewString(), App: "db", Model: "workspace", ID: workspace.String()})
		if admitErr != nil {
			t.Fatal(admitErr)
		}
		waitVersionWorkflow(t, ctx, app, run)
		if store.applications.Load() != before {
			t.Fatal("missing root dispatched cascade writes")
		}
	})
	t.Run("exhausted planning retries cannot apply a cascade", func(t *testing.T) {
		before := store.applications.Load()
		store.failPlanning.Store(true)
		defer store.failPlanning.Store(false)
		run, admitErr := dispatcher.Admit(ctx, DeletionInput{EventID: uuid.NewString(), App: "db", Model: "workspace", ID: sibling.String()})
		if admitErr != nil {
			t.Fatal(admitErr)
		}
		result, waitErr := selvedge.WaitWorkflow[struct{}](ctx, app, run)
		if waitErr != nil {
			t.Fatal(waitErr)
		}
		if result.Status != "failed" || result.Error == nil {
			t.Fatalf("planning failure status=%s error=%s", result.Status, result.Error)
		}
		if store.planningFailures.Load() != 3 || store.applications.Load() != before {
			t.Fatalf("planning attempts=%d cascade writes=%d", store.planningFailures.Load(), store.applications.Load()-before)
		}
	})
}
