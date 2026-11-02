//go:build integration

package ratelimit_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/ratelimit"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRedisWindowAdmitsExactlyItsCapacityAcrossClients(t *testing.T) {
	client := testsupport.StartRedis(t)
	store, err := ratelimit.NewRedisStore("redis://" + client.Options().Addr)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	require.NoError(t, store.Ping(t.Context()))
	key, err := ratelimit.BucketKey("concurrency", "shared-identity")
	require.NoError(t, err)
	var admitted atomic.Int64
	var group sync.WaitGroup
	start := make(chan struct{})
	errorsOut := make(chan error, 64)
	for index := range 64 {
		group.Go(func() {
			<-start
			var result ratelimit.Result
			var requestErr error
			if index%2 == 0 {
				result, requestErr = store.Take(t.Context(), key, 7, time.Minute)
			} else {
				result, requestErr = ratelimit.TakeRedis(t.Context(), client, key, 7, time.Minute)
			}
			if requestErr != nil {
				errorsOut <- requestErr
			} else if result.Allowed {
				admitted.Add(1)
			} else if result.Remaining != "0" || result.Wait <= 0 || result.Wait > time.Minute {
				errorsOut <- errors.New("invalid rejection metadata")
			}
		})
	}
	close(start)
	group.Wait()
	close(errorsOut)
	for requestErr := range errorsOut {
		require.NoError(t, requestErr)
	}
	require.EqualValues(t, 7, admitted.Load())
	require.EqualValues(t, 7, client.ZCard(t.Context(), key).Val())
	require.Equal(t, "zset", client.Type(t.Context(), key).Val())
	remaining, err := client.PTTL(t.Context(), key).Result()
	require.NoError(t, err)
	require.Greater(t, remaining, time.Duration(0))
	require.LessOrEqual(t, remaining, time.Minute)
}

func TestRedisWindowExpiresOldEntriesAndDoesNotExtendDeniedBuckets(t *testing.T) {
	client := testsupport.StartRedis(t)
	ctx := t.Context()
	now, err := client.Time(ctx).Result()
	require.NoError(t, err)
	key, err := ratelimit.BucketKey("expiry", "identity")
	require.NoError(t, err)
	require.NoError(t, client.ZAdd(ctx, key,
		redis.Z{Score: float64(now.Add(-time.Minute).UnixMilli()), Member: "expired"},
		redis.Z{Score: float64(now.UnixMilli()), Member: "current"},
	).Err())
	allowed, err := ratelimit.TakeRedis(ctx, client, key, 2, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed.Allowed)
	require.Equal(t, "0", allowed.Remaining)
	require.ErrorIs(t, client.ZScore(ctx, key, "expired").Err(), redis.Nil)
	require.EqualValues(t, 2, client.ZCard(ctx, key).Val())
	require.NoError(t, client.PExpire(ctx, key, 30*time.Second).Err())
	denied, err := ratelimit.TakeRedis(ctx, client, key, 2, time.Minute)
	require.NoError(t, err)
	require.False(t, denied.Allowed)
	require.Greater(t, denied.Wait, time.Duration(0))
	require.LessOrEqual(t, denied.Wait, time.Minute)
	require.Equal(t, allowed.Reset, denied.Reset)
	require.LessOrEqual(t, client.PTTL(ctx, key).Val(), 30*time.Second)
	zero, err := ratelimit.TakeRedis(ctx, client, "zero-rate", 0, time.Minute)
	require.NoError(t, err)
	require.False(t, zero.Allowed)
	require.Equal(t, time.Minute, zero.Wait)
	require.Zero(t, client.Exists(ctx, "zero-rate").Val())
}

func TestRedisWindowScopesIdentitiesAndFailsClosedOnCorruptionOrCancellation(t *testing.T) {
	client := testsupport.StartRedis(t)
	store, err := ratelimit.NewRedisStore("redis://" + client.Options().Addr)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	limiter, err := ratelimit.New(store, "1/minute")
	require.NoError(t, err)
	for _, scope := range []string{"actor", "credential"} {
		result, requestErr := limiter.Allow(t.Context(), scope, "same-identity")
		require.NoError(t, requestErr)
		require.True(t, result.Allowed)
		result, requestErr = limiter.Allow(t.Context(), scope, "same-identity")
		require.NoError(t, requestErr)
		require.False(t, result.Allowed)
	}
	key, err := ratelimit.BucketKey("corrupt", "identity")
	require.NoError(t, err)
	require.NoError(t, client.Set(t.Context(), key, "unexpected format", time.Minute).Err())
	result, err := limiter.Allow(t.Context(), "corrupt", "identity")
	require.Error(t, err)
	require.False(t, result.Allowed)
	require.Equal(t, "unexpected format", client.Get(t.Context(), key).Val())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	result, err = limiter.Allow(ctx, "cancelled", "identity")
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, result.Allowed)
	key, err = ratelimit.BucketKey("cancelled", "identity")
	require.NoError(t, err)
	require.Zero(t, client.Exists(t.Context(), key).Val())
}
