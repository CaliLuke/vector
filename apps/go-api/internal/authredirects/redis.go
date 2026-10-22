package authredirects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/redis/go-redis/v9"
)

const incrementMagicVerifyAttempts = `local count = redis.call("INCR", KEYS[1]); if count == 1 then redis.call("EXPIRE", KEYS[1], tonumber(ARGV[1])); end; return count`

type magicCodeRecord struct {
	CurrentAttempt int    `json:"current_attempt"`
	Email          string `json:"email"`
	Token          string `json:"token"`
}

type RedisMagicCodeVerifier struct {
	client redis.UniversalClient
}

func NewRedisMagicCodeVerifier(client redis.UniversalClient) (*RedisMagicCodeVerifier, error) {
	if client == nil {
		return nil, errors.New("magic-code Redis client is required")
	}
	return &RedisMagicCodeVerifier{client: client}, nil
}

// Verify consumes a valid Django-issued magic code and preserves the source
// five-attempt counter and token-expiry behavior for invalid codes.
func (verifier *RedisMagicCodeVerifier) Verify(ctx context.Context, key, code string, emailExists bool) error {
	encoded, err := verifier.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return magicCodeFailure(emailExists, false, key)
	}
	if err != nil {
		return fmt.Errorf("read magic code: %w", err)
	}
	var record magicCodeRecord
	if err := json.Unmarshal(encoded, &record); err != nil {
		return fmt.Errorf("decode magic code: %w", err)
	}
	if record.Token == code {
		if err := verifier.client.Del(ctx, key, key+":verify_attempts").Err(); err != nil {
			return fmt.Errorf("consume magic code: %w", err)
		}
		return nil
	}
	remaining, err := verifier.client.TTL(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("read magic-code expiry: %w", err)
	}
	remainingSeconds := int64(remaining / time.Second)
	if remainingSeconds <= 0 {
		remainingSeconds = 1
	}
	count, err := verifier.client.Eval(ctx, incrementMagicVerifyAttempts, []string{key + ":verify_attempts"}, remainingSeconds).Int64()
	if err != nil {
		return fmt.Errorf("count invalid magic-code attempt: %w", err)
	}
	if count >= 5 {
		if err := verifier.client.Del(ctx, key, key+":verify_attempts").Err(); err != nil {
			return fmt.Errorf("invalidate exhausted magic code: %w", err)
		}
		return magicCodeFailure(emailExists, true, key)
	}
	return magicCodeFailure(emailExists, false, key)
}

func magicCodeFailure(emailExists, exhausted bool, key string) error {
	email := strings.Replace(key, "magic_", "", 1)
	if emailExists {
		if exhausted {
			return &CodeError{Code: 5100, Message: "EMAIL_CODE_ATTEMPT_EXHAUSTED_SIGN_IN", Email: email}
		}
		return &CodeError{Code: 5090, Message: "INVALID_MAGIC_CODE_SIGN_IN", Email: email}
	}
	if exhausted {
		return &CodeError{Code: 5102, Message: "EMAIL_CODE_ATTEMPT_EXHAUSTED_SIGN_UP", Email: key}
	}
	return &CodeError{Code: 5092, Message: "INVALID_MAGIC_CODE_SIGN_UP", Email: email}
}

type AuthenticationDRFThrottle struct {
	throttle *apikey.Throttle
}

func NewAuthenticationDRFThrottle(redisURL, rate string) (*AuthenticationDRFThrottle, *publicprojectmetadatareads.RedisThrottleCache, error) {
	if rate == "" {
		rate = os.Getenv("AUTHENTICATION_RATE_LIMIT")
		if rate == "" {
			rate = "10/minute"
		}
	}
	cache, err := publicprojectmetadatareads.NewRedisThrottleCache(redisURL)
	if err != nil {
		return nil, nil, err
	}
	throttle, err := apikey.NewThrottle(cache, rate)
	if err != nil {
		_ = cache.Close()
		return nil, nil, err
	}
	return &AuthenticationDRFThrottle{throttle: throttle}, cache, nil
}

func (throttle *AuthenticationDRFThrottle) Allow(ctx context.Context, clientIP string) (bool, error) {
	allowed, _, err := throttle.AllowWithWait(ctx, clientIP)
	return allowed, err
}

// AllowWithWait retains the shared Django throttle's computed retry interval.
func (throttle *AuthenticationDRFThrottle) AllowWithWait(ctx context.Context, clientIP string) (bool, time.Duration, error) {
	rate, err := throttle.throttle.AllowCacheKey(ctx, ":1:throttle_authentication_"+clientIP, "authentication")
	if err != nil {
		return false, 0, err
	}
	return rate.Allowed, rate.Wait, nil
}
