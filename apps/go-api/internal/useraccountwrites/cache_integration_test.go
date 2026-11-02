//go:build integration

package useraccountwrites

import (
	"context"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/ratelimit"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestNativeEmailCodeStorageAndQuotaUseIndependentKeys(t *testing.T) {
	server := testsupport.StartRedis(t)
	cache, err := NewCache("redis://" + server.Options().Addr)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cache.Close()) })
	ctx := t.Context()
	const key = "vector:email-code:v1:storage-test"
	encoded, err := EncodeEmailVerificationCodeValue("012345")
	require.NoError(t, err)
	require.NoError(t, cache.Set(ctx, key, encoded, 10*time.Minute))
	stored, found, err := cache.Get(ctx, key)
	require.NoError(t, err)
	require.True(t, found)
	code, valid := decodeEmailCode(stored)
	require.True(t, valid)
	require.Equal(t, "012345", code)
	limiter, err := ratelimit.New(cache, "1/hour")
	require.NoError(t, err)
	first, err := limiter.Allow(ctx, "email-verification", "actor")
	require.NoError(t, err)
	require.True(t, first.Allowed)
	require.NoError(t, cache.Delete(ctx, key))
	_, found, err = cache.Get(ctx, key)
	require.NoError(t, err)
	require.False(t, found)
	second, err := limiter.Allow(ctx, "email-verification", "actor")
	require.NoError(t, err)
	require.False(t, second.Allowed, "consuming a code must not reset the actor quota")
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	_, _, err = cache.Get(ctx, key)
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, cache.Set(ctx, key, encoded, time.Minute), context.Canceled)
	require.ErrorIs(t, cache.Delete(ctx, key), context.Canceled)
}

func TestEmailCodeGuessBudgetIsAtomicAndReplacementResetsIt(t *testing.T) {
	server := testsupport.StartRedis(t)
	cache, err := NewCache("redis://" + server.Options().Addr)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cache.Close()) })
	const key = "vector:email-code:v1:guess-budget"
	encoded, err := EncodeEmailVerificationCodeValue("012345")
	require.NoError(t, err)
	require.NoError(t, cache.Set(t.Context(), key, encoded, time.Minute))
	start := make(chan struct{})
	results := make(chan EmailCodeStatus, 32)
	failures := make(chan error, 32)
	var group sync.WaitGroup
	for range 32 {
		group.Go(func() {
			<-start
			status, verifyErr := cache.VerifyEmailCode(t.Context(), key, "999999")
			if verifyErr != nil {
				failures <- verifyErr
			} else {
				results <- status
			}
		})
	}
	close(start)
	group.Wait()
	close(results)
	close(failures)
	for verifyErr := range failures {
		require.NoError(t, verifyErr)
	}
	counts := map[EmailCodeStatus]int{}
	for status := range results {
		counts[status]++
	}
	require.Equal(t, 4, counts[EmailCodeInvalid])
	require.Equal(t, 1, counts[EmailCodeExhausted])
	require.Equal(t, 27, counts[EmailCodeMissing])
	status, err := cache.VerifyEmailCode(t.Context(), key, "012345")
	require.NoError(t, err)
	require.Equal(t, EmailCodeMissing, status)
	require.NoError(t, cache.Set(t.Context(), key, encoded, time.Minute))
	status, err = cache.VerifyEmailCode(t.Context(), key, "wrong")
	require.NoError(t, err)
	require.Equal(t, EmailCodeInvalid, status)
	codeTTL, err := server.PTTL(t.Context(), key).Result()
	require.NoError(t, err)
	attemptTTL, err := server.PTTL(t.Context(), key+":attempts").Result()
	require.NoError(t, err)
	require.Greater(t, attemptTTL, time.Duration(0))
	require.LessOrEqual(t, attemptTTL, codeTTL)
	require.NoError(t, cache.Set(t.Context(), key, encoded, time.Minute))
	require.Zero(t, server.Exists(t.Context(), key+":attempts").Val())
	status, err = cache.VerifyEmailCode(t.Context(), key, "012345")
	require.NoError(t, err)
	require.Equal(t, EmailCodeValid, status)
	require.NoError(t, cache.Delete(t.Context(), key))
	require.Zero(t, server.Exists(t.Context(), key, key+":attempts").Val())
}

type verificationHook struct {
	afterLookup func(context.Context) error
	scriptValue *int64
	scriptErr   error
}

func (hook verificationHook) DialHook(next redis.DialHook) redis.DialHook { return next }
func (hook verificationHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (hook verificationHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, command redis.Cmder) error {
		if command.Name() == "evalsha" {
			if hook.scriptErr != nil {
				return hook.scriptErr
			}
			if hook.scriptValue != nil {
				command.(*redis.Cmd).SetVal(*hook.scriptValue)
				return nil
			}
		}
		if err := next(ctx, command); err != nil {
			return err
		}
		if command.Name() == "get" && hook.afterLookup != nil {
			return hook.afterLookup(ctx)
		}
		return nil
	}
}

func TestEmailVerificationRechecksReplacementBeforeAcceptingSnapshot(t *testing.T) {
	server := testsupport.StartRedis(t)
	for _, replacements := range []int{1, 3} {
		t.Run(string(rune('0'+replacements)), func(t *testing.T) {
			cache, err := NewCache("redis://" + server.Options().Addr)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, cache.Close()) })
			key := "vector:email-code:v1:replacement-" + t.Name()
			original, err := EncodeEmailVerificationCodeValue("111111")
			require.NoError(t, err)
			require.NoError(t, cache.Set(t.Context(), key, original, time.Minute))
			changed := 0
			cache.client.AddHook(verificationHook{afterLookup: func(ctx context.Context) error {
				if changed >= replacements {
					return nil
				}
				changed++
				value := "222222"
				if changed%2 == 0 {
					value = "333333"
				}
				encoded, encodeErr := EncodeEmailVerificationCodeValue(value)
				if encodeErr != nil {
					return encodeErr
				}
				return cache.Set(ctx, key, encoded, time.Minute)
			}})
			status, err := cache.VerifyEmailCode(t.Context(), key, "111111")
			if replacements == 1 {
				require.NoError(t, err)
				require.Equal(t, EmailCodeInvalid, status)
				status, err = cache.VerifyEmailCode(t.Context(), key, "222222")
				require.NoError(t, err)
				require.Equal(t, EmailCodeValid, status)
			} else {
				require.Error(t, err)
				require.NotEqual(t, EmailCodeValid, status)
			}
		})
	}
}

func TestEmailCodeVerificationRejectsMalformedStoreStateAndFailures(t *testing.T) {
	server := testsupport.StartRedis(t)
	cache, err := NewCache("redis://" + server.Options().Addr)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cache.Close()) })
	const key = "vector:email-code:v1:malformed"
	require.Error(t, cache.Set(t.Context(), key, []byte("invalid"), 0))
	require.NoError(t, cache.Set(t.Context(), key, []byte("invalid"), time.Minute))
	status, err := cache.VerifyEmailCode(t.Context(), key, "123456")
	require.NoError(t, err)
	require.Equal(t, EmailCodeMalformed, status)
	encoded, err := EncodeEmailVerificationCodeValue("123456")
	require.NoError(t, err)
	require.NoError(t, server.Set(t.Context(), key, encoded, 0).Err())
	status, err = cache.VerifyEmailCode(t.Context(), key, "123456")
	require.NoError(t, err)
	require.Equal(t, EmailCodeMalformed, status, "a challenge requires an expiry")
	require.NoError(t, cache.Set(t.Context(), key, encoded, time.Minute))
	for _, invalid := range []string{"invalid", "1.0", "00", "-1"} {
		require.NoError(t, server.Set(t.Context(), key+":attempts", invalid, time.Minute).Err())
		status, err = cache.VerifyEmailCode(t.Context(), key, "123456")
		require.NoError(t, err)
		require.Equal(t, EmailCodeMalformed, status)
	}
	require.NoError(t, server.Set(t.Context(), key+":attempts", "5", time.Minute).Err())
	status, err = cache.VerifyEmailCode(t.Context(), key, "123456")
	require.NoError(t, err)
	require.Equal(t, EmailCodeExhausted, status)
	require.NoError(t, cache.Set(t.Context(), key, encoded, time.Minute))
	unknown := int64(77)
	cache.client.AddHook(verificationHook{scriptValue: &unknown})
	status, err = cache.VerifyEmailCode(t.Context(), key, "123456")
	require.Error(t, err)
	require.NotEqual(t, EmailCodeValid, status)
	unavailable := errors.New("verification unavailable")
	failedCache, err := NewCache("redis://" + server.Options().Addr)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, failedCache.Close()) })
	failedCache.client.AddHook(verificationHook{scriptErr: unavailable})
	status, err = failedCache.VerifyEmailCode(t.Context(), key, "123456")
	require.ErrorIs(t, err, unavailable)
	require.NotEqual(t, EmailCodeValid, status)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = cache.VerifyEmailCode(ctx, key, "123456")
	require.ErrorIs(t, err, context.Canceled)
}

func TestEmailCodeCacheRejectsUntrustedTLS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	cache, err := NewCache("rediss://" + server.Listener.Addr().String())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cache.Close()) })
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	_, _, err = cache.Get(ctx, "tls-proof")
	var untrusted x509.UnknownAuthorityError
	require.True(t, errors.As(err, &untrusted), "error=%v", err)
}
