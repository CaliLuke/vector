//go:build integration

package workitems

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestPublicVotesCommitMembershipAndEffectsAtomically(t *testing.T) {
	ctx := context.Background()
	pairURL := migratedDatabases.SeedDatabase(t, func(context.Context, string) error { return nil })

	db, err := pgxpool.New(ctx, pairURL)
	require.NoError(t, err)
	t.Cleanup(db.Close)
	actor, workspace, project, issue := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	exec := func(query string, args ...any) {
		t.Helper()
		_, execErr := db.Exec(ctx, query, args...)
		require.NoError(t, execErr)
	}
	exec(`INSERT INTO users(id,username,email,password,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,token,user_timezone,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,display_name,is_email_valid,is_password_reset_required) VALUES($1::uuid,$1::text,'voter@example.test','!','','','',now(),now(),now(),'','',false,false,false,true,false,true,false,'','UTC','','','','',false,'Voter',true,false)`, actor)
	exec(`INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Votes',$2,'votes','UTC','',now(),now())`, workspace, actor)
	exec(`INSERT INTO projects(id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at) VALUES($1,'Votes','',0,$2,'VOTE',true,true,false,true,false,false,false,false,0,0,'{}','UTC',now(),now())`, project, workspace)
	exec(`INSERT INTO issues(id,workspace_id,project_id,name,description_json,description_html,priority,sequence_id,sort_order,is_draft,created_at,updated_at,created_by_id) VALUES($1,$2,$3,'Public item','{}','','none',1,0,false,now(),now(),$4)`, issue, workspace, project, actor)
	exec(`INSERT INTO deploy_boards(id,workspace_id,project_id,entity_identifier,entity_name,anchor,is_comments_enabled,is_reactions_enabled,is_votes_enabled,view_props,is_activity_enabled,is_disabled,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$2,'project','votes-board',true,true,true,'{}',true,false,now(),now())`, workspace, project)
	publisher := &versionTaskRecorder{}
	repository, err := NewRepository(db, publisher, "http://example.test")
	require.NoError(t, err)
	scope := PublicFeedbackScope{WorkspaceID: workspace, WorkspaceSlug: "votes", ProjectID: project, Anchor: "votes-board", VotesEnabled: true}
	create := func(body string) (Result, error) {
		return repository.PublicVoteCreate(ctx, scope, issue, actor, strings.NewReader(body), "application/json")
	}
	counts := func(votes, members, effects int) {
		t.Helper()
		var gotVotes, gotMembers, gotEffects int
		require.NoError(t, db.QueryRow(ctx, `SELECT (SELECT count(*) FROM issue_votes WHERE deleted_at IS NULL),(SELECT count(*) FROM project_public_members),(SELECT count(*) FROM work_item_effects)`).Scan(&gotVotes, &gotMembers, &gotEffects))
		require.Equal(t, []int{votes, members, effects}, []int{gotVotes, gotMembers, gotEffects})
	}
	_, err = create(`{"vote":true}`)
	var clientError *apicontract.HTTPError
	require.ErrorAs(t, err, &clientError)
	require.Equal(t, 400, clientError.Status)
	counts(0, 0, 0)

	exec(`ALTER TABLE work_item_effects ADD CONSTRAINT reject_vote_effect CHECK (kind <> 'activity')`)
	_, err = create(`{"vote":-1}`)
	require.Error(t, err)
	counts(0, 0, 0)
	exec(`ALTER TABLE work_item_effects DROP CONSTRAINT reject_vote_effect`)

	const concurrentVotes = 4
	failures := make(chan error, concurrentVotes)
	var writers sync.WaitGroup
	for range concurrentVotes {
		writers.Go(func() {
			_, createErr := create(`{"vote":-1}`)
			failures <- createErr
		})
	}
	writers.Wait()
	close(failures)
	for failure := range failures {
		require.NoError(t, failure)
	}
	counts(1, 1, concurrentVotes)
	var vote int
	require.NoError(t, db.QueryRow(ctx, `SELECT vote FROM issue_votes WHERE deleted_at IS NULL`).Scan(&vote))
	require.Equal(t, -1, vote)
	require.Empty(t, publisher.names, "effects must stay in the durable outbox")

	exec(`ALTER TABLE work_item_effects ADD CONSTRAINT reject_vote_delete CHECK (kind <> 'soft_delete')`)
	_, err = repository.PublicVoteDelete(ctx, scope, issue, actor)
	require.Error(t, err)
	counts(1, 1, concurrentVotes)
	exec(`ALTER TABLE work_item_effects DROP CONSTRAINT reject_vote_delete`)
	result, err := repository.PublicVoteDelete(ctx, scope, issue, actor)
	require.NoError(t, err)
	require.Equal(t, 204, result.Status)
	counts(0, 1, concurrentVotes+2)
	result, err = create(`{"vote":1}`)
	require.NoError(t, err)
	require.Equal(t, 201, result.Status)
	counts(1, 1, concurrentVotes+3)

	exec(`UPDATE deploy_boards SET is_votes_enabled=false`)
	_, err = create(`{"vote":-1}`)
	if !errors.As(err, &clientError) || clientError.Status != 404 {
		t.Fatalf("stale board scope allowed a vote: %v", err)
	}
	counts(1, 1, concurrentVotes+3)
}
