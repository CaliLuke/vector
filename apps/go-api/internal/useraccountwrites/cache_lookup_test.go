package useraccountwrites

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// Redis hooks are an existing extension point that can add error context.
// This hook injects lookup results without a network connection.
type cacheLookupHook struct {
	value string
	err   error
}

func (h cacheLookupHook) DialHook(next redis.DialHook) redis.DialHook { return next }
func (h cacheLookupHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h cacheLookupHook) ProcessHook(redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		cmd.(*redis.StringCmd).SetVal(h.value)
		return h.err
	}
}

func TestCacheLookupPreservesWrappedMissAndFailureSemantics(t *testing.T) {
	unavailable := errors.New("cache unavailable")
	for _, tc := range []struct {
		name, value       string
		injected, wantErr error
		found             bool
	}{
		{"value", "stored", nil, nil, true},
		{"empty value", "", nil, nil, true},
		{"direct miss", "", redis.Nil, nil, false},
		{"wrapped miss", "", fmt.Errorf("cache diagnostic: %w", redis.Nil), nil, false},
		{"wrapped failure", "", fmt.Errorf("cache diagnostic: %w", unavailable), unavailable, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cache, err := NewCache("redis://127.0.0.1:1")
			require.NoError(t, err)
			defer testsupport.Close(t, cache)
			cache.client.AddHook(cacheLookupHook{tc.value, tc.injected})
			value, found, err := cache.Get(t.Context(), "key")
			require.ErrorIs(t, err, tc.wantErr)
			require.Equal(t, tc.found, found)
			require.Equal(t, tc.value, string(value))
		})
	}
}
