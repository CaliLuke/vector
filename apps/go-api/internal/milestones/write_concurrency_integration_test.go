//go:build integration

package milestones_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestMilestoneConcurrentReceiptCreatesOneResourceAndEffect(t *testing.T) {
	f := newWriteFixture(t)
	requestID := uuid.New()
	f.opts.RequestID = &requestID
	type outcome struct {
		result milestones.MutationResult
		err    error
	}
	results := make(chan outcome, 2)
	start := make(chan struct{})
	for range 2 {
		go func() {
			<-start
			result, err := f.repo.Create(f.ctx, f.opts, milestones.Fields{Title: textField("Release")})
			results <- outcome{result, err}
		}()
	}
	close(start)
	first, second := <-results, <-results
	require.NoError(t, first.err)
	require.NoError(t, second.err)
	require.Equal(t, first.result.MilestoneID, second.result.MilestoneID)
	require.NotEqual(t, *first.result.Replayed, *second.result.Replayed)
	require.Equal(t, first.result.Result, second.result.Result)
	f.counts(t, 1, 1, 1)
}

func TestMilestoneCredentialIsRevalidatedAfterDomainLockWait(t *testing.T) {
	f := newWriteFixture(t)
	tx, err := f.db.Begin(f.ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(context.WithoutCancel(f.ctx)) }()
	_, err = tx.Exec(f.ctx, `SELECT id FROM projects WHERE id=$1 FOR UPDATE`, f.opts.ProjectID)
	require.NoError(t, err)
	var revoked atomic.Bool
	credentialError := errors.New("credential revoked")
	f.opts.Revalidate = func(context.Context) error {
		if revoked.Load() {
			return credentialError
		}
		return nil
	}
	requestID := uuid.New()
	f.opts.RequestID = &requestID
	finished := make(chan error, 1)
	go func() {
		_, failure := f.repo.Create(f.ctx, f.opts, milestones.Fields{Title: textField("Release")})
		finished <- failure
	}()
	f.waitForWriter(t, tx)
	revoked.Store(true)
	require.NoError(t, tx.Rollback(f.ctx))
	require.ErrorIs(t, <-finished, credentialError)
	f.counts(t, 0, 0, 0)
	revoked.Store(false)
	_, err = f.repo.Create(f.ctx, f.opts, milestones.Fields{Title: textField("Release")})
	require.NoError(t, err)
	f.counts(t, 1, 1, 1)
}

func (f writeFixture) waitForWriter(t *testing.T, tx pgx.Tx) {
	t.Helper()
	require.Eventually(t, func() bool {
		// Statistics reads are cached within this controller transaction.
		if _, failure := tx.Exec(f.ctx, `SELECT pg_stat_clear_snapshot()`); failure != nil {
			return false
		}
		var waiting bool
		failure := tx.QueryRow(f.ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND application_name='milestone-writes' AND wait_event_type='Lock')`).Scan(&waiting)
		return failure == nil && waiting
	}, 3*time.Second, 10*time.Millisecond)
}

func TestMilestoneAssociationRechecksEligibilityAndCredentialAfterItemWait(t *testing.T) {
	for _, revoke := range []bool{false, true} {
		t.Run(map[bool]string{false: "archived item", true: "revoked credential"}[revoke], func(t *testing.T) {
			f := newWriteFixture(t)
			created := f.milestone(t, "Release")
			item := f.item(t)
			tx, err := f.db.Begin(f.ctx)
			require.NoError(t, err)
			defer func() { _ = tx.Rollback(context.WithoutCancel(f.ctx)) }()
			_, err = tx.Exec(f.ctx, `UPDATE issues SET archived_at=now() WHERE id=$1`, item)
			require.NoError(t, err)
			credentialError := errors.New("credential revoked")
			var revoked atomic.Bool
			f.opts.Revalidate = func(context.Context) error {
				if revoked.Load() {
					return credentialError
				}
				return nil
			}
			finished := make(chan error, 1)
			go func() {
				_, failure := f.repo.ManageWorkItems(f.ctx, f.opts, uuid.MustParse(created.MilestoneID), []uuid.UUID{item}, nil)
				finished <- failure
			}()
			f.waitForWriter(t, tx)
			revoked.Store(revoke)
			require.NoError(t, tx.Commit(f.ctx))
			want := milestones.ErrInvalidInput
			if revoke {
				want = credentialError
			}
			require.ErrorIs(t, <-finished, want)
			f.counts(t, 1, 1, 0)
		})
	}
}
