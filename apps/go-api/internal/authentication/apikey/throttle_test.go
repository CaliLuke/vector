package apikey

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/ratelimit"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

func TestThrottleUsesNativeScopedAtomicCapacityAndHeaders(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	store := &testsupport.RateStoreFake{Now: func() time.Time { return now }}
	throttle, err := NewThrottle(store, "2/min")
	require.NoError(t, err)
	first, err := throttle.Allow(t.Context(), "test-credential")
	require.NoError(t, err)
	require.True(t, first.Allowed)
	require.Equal(t, "1", first.Remaining)
	require.Equal(t, now.Unix()+60, first.Reset)
	second, err := throttle.Allow(t.Context(), "test-credential")
	require.NoError(t, err)
	require.True(t, second.Allowed)
	require.Equal(t, "0", second.Remaining)
	third, err := throttle.Allow(t.Context(), "test-credential")
	require.NoError(t, err)
	require.False(t, third.Allowed)
	require.Equal(t, time.Minute, third.Wait)
	require.Len(t, store.Calls, 3)
	require.EqualValues(t, 2, store.Calls[0].Limit)
	require.Equal(t, time.Minute, store.Calls[0].Window)
	require.True(t, strings.HasPrefix(store.Calls[0].Key, "vector:rate:v1:api-key:"))
	require.NotContains(t, store.Calls[0].Key, "test-credential")
	other, err := throttle.Allow(t.Context(), "another-credential")
	require.NoError(t, err)
	require.True(t, other.Allowed)
	require.NotEqual(t, store.Calls[0].Key, store.Calls[3].Key)
}

func TestScopedThrottleSeparatesNativeScopes(t *testing.T) {
	store := &testsupport.RateStoreFake{}
	throttle, err := NewThrottle(store, "1/minute")
	require.NoError(t, err)
	for _, scope := range []string{"asset-id", "email-verification"} {
		rate, requestErr := throttle.AllowScoped(t.Context(), scope, "same-identity")
		require.NoError(t, requestErr)
		require.True(t, rate.Allowed)
	}
	require.NotEqual(t, store.Calls[0].Key, store.Calls[1].Key)
	expected, err := ratelimit.BucketKey("asset-id", "same-identity")
	require.NoError(t, err)
	require.Equal(t, expected, store.Calls[0].Key)
}

func TestThrottleRejectsInvalidConfigurationAndFailsClosed(t *testing.T) {
	_, err := NewThrottle(nil, "1/minute")
	require.Error(t, err)
	for _, rate := range []string{"-1/minute", "1/monkeys", "١/minute", "1_0/minute", "9223372036854775808/minute"} {
		_, rateErr := NewThrottle(&testsupport.RateStoreFake{}, rate)
		require.Error(t, rateErr, rate)
	}
	unavailable := errors.New("rate store unavailable")
	store := &testsupport.RateStoreFake{Err: unavailable}
	throttle, err := NewThrottle(store, "1/minute")
	require.NoError(t, err)
	result, err := throttle.Allow(t.Context(), "identity")
	require.ErrorIs(t, err, unavailable)
	require.False(t, result.Allowed)
	store.Err = nil
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	result, err = throttle.Allow(ctx, "identity")
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, result.Allowed)
	require.Empty(t, store.Counts)
}
