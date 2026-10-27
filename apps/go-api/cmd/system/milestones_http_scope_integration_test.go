//go:build integration

package main

import (
	"context"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	store "github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type milestoneRevalidatingKeys struct {
	*apikey.Validator
	beforeRepeat func()
	calls        int
}

func (k *milestoneRevalidatingKeys) Validate(ctx context.Context, key string) (apikey.Principal, error) {
	k.calls++
	if k.calls == 2 {
		k.beforeRepeat()
	}
	return k.Validator.Validate(ctx, key)
}

func TestMilestoneHTTPRevalidatesCredentialInsideMutation(t *testing.T) {
	f := newMilestoneHTTPFixture(t)
	keys := &milestoneRevalidatingKeys{Validator: f.keys, beforeRepeat: func() { f.exec(t, `UPDATE api_tokens SET is_active=false`) }}
	service, err := systemapi.NewMilestones(f.repo, keys, milestoneHTTPThrottle{}, "https://vector.example.test")
	require.NoError(t, err)
	f.handler = milestoneTestHTTP(service)
	f.call(t, "POST", "", `{"title":"Revoked","request_id":"`+uuid.NewString()+`"}`, 403)
	require.Equal(t, 2, keys.calls)
	var records, effects, receipts int
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM milestones),(SELECT count(*) FROM work_item_effects),(SELECT count(*) FROM mutation_receipts)`).Scan(&records, &effects, &receipts))
	require.Zero(t, records)
	require.Zero(t, effects)
	require.Zero(t, receipts)
}

type milestoneScopeChange struct {
	*store.Repository
	afterResolve func()
}

func (r milestoneScopeChange) ResolveScope(ctx context.Context, actor, project uuid.UUID, slug, timezone string) (store.Scope, error) {
	scope, err := r.Repository.ResolveScope(ctx, actor, project, slug, timezone)
	if err == nil {
		r.afterResolve()
	}
	return scope, err
}

func TestMilestoneHTTPBindsWorkspaceBeforeSlugReassignment(t *testing.T) {
	for _, method := range []string{"GET", "POST"} {
		t.Run(method, func(t *testing.T) {
			f := newMilestoneHTTPFixture(t)
			replacement := uuid.NewString()
			application := milestoneScopeChange{Repository: f.repo, afterResolve: func() {
				f.exec(t, `UPDATE workspaces SET slug='renamed' WHERE id=$1`, testsupport.LabelWorkspace)
				f.exec(t, `INSERT INTO workspaces SELECT (jsonb_populate_record(NULL::workspaces,to_jsonb(source)||jsonb_build_object('id',$1::uuid,'slug','milestones'))).* FROM workspaces source WHERE id=$2`, replacement, testsupport.LabelWorkspace)
			}}
			service, err := systemapi.NewMilestones(application, f.keys, milestoneHTTPThrottle{}, "https://vector.example.test")
			require.NoError(t, err)
			f.handler = milestoneTestHTTP(service)
			f.call(t, method, "", `{"title":"Wrong owner"}`, 403)
			var owner string
			require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT id::text FROM workspaces WHERE slug='milestones'`).Scan(&owner))
			require.Equal(t, replacement, owner)
			var records int
			require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM milestones`).Scan(&records))
			require.Zero(t, records)
		})
	}
}

func TestMilestoneHTTPThrottleStopsMutationBeforeEffects(t *testing.T) {
	f := newMilestoneHTTPFixture(t)
	service, err := systemapi.NewMilestones(f.repo, f.keys, milestoneHTTPThrottle{denied: true}, "https://vector.example.test")
	require.NoError(t, err)
	f.handler = milestoneTestHTTP(service)
	f.call(t, "POST", "", `{"title":"Throttled"}`, 429)
	var effects int
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM work_item_effects`).Scan(&effects))
	require.Zero(t, effects)
}
