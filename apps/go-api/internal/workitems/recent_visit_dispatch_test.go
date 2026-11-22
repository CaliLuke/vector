package workitems

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRecentVisitDispatchCoalescesReadsAndDetachesRequestCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dispatch := newRecentVisitDispatcher()
		key := recentVisitKey{slug: "workspace", project: "project", actor: "actor", entity: "issue", id: "item"}
		request, cancel := context.WithCancel(context.Background())
		cancel()

		release := make(chan struct{})
		calls := 0
		admit := func(ctx context.Context) error {
			require.NoError(t, ctx.Err())
			deadline, bounded := ctx.Deadline()
			require.True(t, bounded)
			require.Equal(t, recentVisitTimeout, time.Until(deadline))
			calls++
			<-release
			return nil
		}
		dispatch.submit(request, key, admit)
		dispatch.submit(request, key, admit)
		synctest.Wait()
		require.Equal(t, 1, calls)

		close(release)
		synctest.Wait()
		dispatch.submit(request, key, admit)
		synctest.Wait()
		require.Equal(t, 2, calls)
	})
}

func TestRecentVisitDispatchBoundsConcurrentAdmissionsAndRetriesFailures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dispatch := newRecentVisitDispatcher()
		var calls, timeouts atomic.Int32
		admit := func(ctx context.Context) error {
			calls.Add(1)
			<-ctx.Done()
			require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
			timeouts.Add(1)
			return ctx.Err()
		}
		for i := range recentVisitConcurrency + 1 {
			dispatch.submit(t.Context(), recentVisitKey{entity: "issue", id: fmt.Sprint(i)}, admit)
		}
		synctest.Wait()
		require.Equal(t, recentVisitConcurrency, int(calls.Load()))
		time.Sleep(recentVisitTimeout)
		synctest.Wait()
		require.Equal(t, recentVisitConcurrency, int(timeouts.Load()))
		dispatch.submit(t.Context(), recentVisitKey{entity: "issue", id: "0"}, admit)
		synctest.Wait()
		require.Equal(t, recentVisitConcurrency+1, int(calls.Load()))
		time.Sleep(recentVisitTimeout)
		synctest.Wait()
	})
}
