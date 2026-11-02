package ratelimit

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type takeFunc func(context.Context, string, int64, time.Duration) (Result, error)

func (take takeFunc) Take(ctx context.Context, key string, limit int64, window time.Duration) (Result, error) {
	return take(ctx, key, limit, window)
}

func TestNativeRateConfiguration(t *testing.T) {
	for _, test := range []struct {
		rate   string
		limit  int64
		window time.Duration
	}{
		{"0/minute", 0, time.Minute}, {"2/min", 2, time.Minute}, {"3/hour", 3, time.Hour},
		{"1/s", 1, time.Second}, {"1/seconds", 1, time.Second}, {"1/sec", 1, time.Second},
		{"1/m", 1, time.Minute}, {"1/minutes", 1, time.Minute}, {"1/h", 1, time.Hour},
		{"1/hours", 1, time.Hour}, {"1/d", 1, 24 * time.Hour}, {"1/day", 1, 24 * time.Hour},
		{"2147483647/days", 2147483647, 24 * time.Hour},
	} {
		t.Run(test.rate, func(t *testing.T) {
			limit, window, err := parseRate(test.rate)
			require.NoError(t, err)
			require.Equal(t, test.limit, limit)
			require.Equal(t, test.window, window)
		})
	}
	for _, rate := range []string{"", "1", "/minute", "1/minute/extra", "-1/minute", "+1/minute", " 1/minute", "١/minute", "1_000/minute", "1.0/minute", "1e3/minute", "2147483648/minute", "1/monkeys", "1/", "1/ minute"} {
		_, _, err := parseRate(rate)
		require.Error(t, err, rate)
	}
	_, err := New(nil, "1/minute")
	require.Error(t, err)
	_, err = New(takeFunc(func(context.Context, string, int64, time.Duration) (Result, error) { return Result{}, nil }), "invalid")
	require.Error(t, err)
}

func TestLimiterScopesOpaqueIdentityAndPreservesStoreResults(t *testing.T) {
	want := Result{Allowed: true, Remaining: "2", Reset: 123, Wait: time.Second}
	var received string
	store := takeFunc(func(_ context.Context, key string, limit int64, window time.Duration) (Result, error) {
		received = key
		require.EqualValues(t, 3, limit)
		require.Equal(t, time.Hour, window)
		return want, nil
	})
	limiter, err := New(store, "3/hour")
	require.NoError(t, err)
	got, err := limiter.Allow(t.Context(), "api-key", "private-credential")
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.True(t, strings.HasPrefix(received, "vector:rate:v1:api-key:"))
	require.NotContains(t, received, "private-credential")
	again, err := BucketKey("api-key", "private-credential")
	require.NoError(t, err)
	require.Equal(t, received, again)
	other, err := BucketKey("mcp-credential", "private-credential")
	require.NoError(t, err)
	require.NotEqual(t, received, other)
	for _, input := range [][2]string{{"", "identity"}, {"scope", ""}, {"scope:other", "identity"}, {strings.Repeat("a", 65), "identity"}} {
		_, err := limiter.Allow(t.Context(), input[0], input[1])
		require.Error(t, err)
	}
}

func TestLimiterStoreErrorsFailClosedAndRetainCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	limiter, err := New(takeFunc(func(ctx context.Context, _ string, _ int64, _ time.Duration) (Result, error) {
		return Result{Allowed: true}, ctx.Err()
	}), "1/minute")
	require.NoError(t, err)
	result, err := limiter.Allow(ctx, "api-key", "identity")
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, result.Allowed)
}

type replyScripter struct {
	redis.Scripter
	value any
}

func (server replyScripter) EvalSha(ctx context.Context, _ string, _ []string, _ ...any) *redis.Cmd {
	command := redis.NewCmd(ctx)
	command.SetVal(server.value)
	return command
}

func TestRedisLimiterRejectsInvalidConfigurationAndReplies(t *testing.T) {
	for _, test := range []struct {
		key    string
		limit  int64
		window time.Duration
	}{{"", 1, time.Minute}, {"key", -1, time.Minute}, {"key", 2147483648, time.Minute}, {"key", 1, time.Nanosecond}, {"key", 1, 25 * time.Hour}} {
		result, err := TakeRedis(t.Context(), nil, test.key, test.limit, test.window)
		require.Error(t, err)
		require.False(t, result.Allowed)
	}
	for _, reply := range []any{[]any{}, []any{int64(2), int64(0), int64(1), int64(0)}, []any{int64(1), int64(0), int64(-1), int64(0)}, []any{int64(0), int64(0), int64(1), int64(60001)}, "invalid"} {
		result, err := TakeRedis(t.Context(), replyScripter{value: reply}, "key", 1, time.Minute)
		require.Error(t, err)
		require.False(t, result.Allowed)
	}
	_, err := NewRedisStore("not a Redis URL")
	require.Error(t, err)
	_, err = NewRedisStore("redis://user:synthetic-password@cache.example.test:invalid/")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "synthetic-password")
	store, err := NewRedisStore("redis://127.0.0.1:1")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.True(t, errors.Is(store.Ping(ctx), context.Canceled))
}
