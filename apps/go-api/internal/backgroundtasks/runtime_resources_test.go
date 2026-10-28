package backgroundtasks

import (
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type failingRuntimeActivityStore struct {
	issueActivityStore
	err error
}

func (store failingRuntimeActivityStore) Close() error { return store.err }

func TestRuntimeShutdownPreservesBothStoreFailures(t *testing.T) {
	activityErr := errors.New("activity cleanup failed")
	emailOrigin := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	require.NoError(t, emailOrigin.Close())
	runtime := &Runtime{
		IssueActivities: &IssueActivityDispatcher{store: failingRuntimeActivityStore{err: activityErr}},
		emailStore:      &issueEmailStore{origin: emailOrigin},
	}

	err := runtime.Shutdown(t.Context())
	require.ErrorIs(t, err, activityErr)
	require.ErrorIs(t, err, redis.ErrClosed, "the email cleanup error must survive the earlier activity cleanup failure")
}
