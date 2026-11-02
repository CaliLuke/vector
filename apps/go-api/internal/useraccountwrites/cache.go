package useraccountwrites

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"encoding/json/jsontext"

	"github.com/CaliLuke/go-plane/internal/ratelimit"
	"github.com/redis/go-redis/v9"
)

type Cache struct{ client *redis.Client }

func NewCache(redisURL string) (*Cache, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, errors.New("invalid account-cache Redis URL")
	}
	return &Cache{client: redis.NewClient(options)}, nil
}

func (cache *Cache) Close() error { return cache.client.Close() }

func (cache *Cache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	value, err := cache.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	return value, err == nil, err
}

func (cache *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return errors.New("email-code expiry must be positive")
	}
	_, err := cache.client.TxPipelined(ctx, func(transaction redis.Pipeliner) error {
		transaction.Set(ctx, key, value, ttl)
		transaction.Del(ctx, key+":attempts")
		return nil
	})
	return err
}

func (cache *Cache) Delete(ctx context.Context, key string) error {
	return cache.client.Del(ctx, key, key+":attempts").Err()
}

type EmailCodeStatus int

const (
	EmailCodeMissing EmailCodeStatus = iota
	EmailCodeValid
	EmailCodeInvalid
	EmailCodeExhausted
	EmailCodeMalformed
	emailCodeChanged
)

// Compare the snapshot again before charging a guess, so replacement challenges
// cannot be validated against an older token. The counter shares its expiry.
var verifyEmailCode = redis.NewScript(`
local current = redis.call('GET', KEYS[1])
if not current then return 0 end
if current ~= ARGV[1] then return 5 end
local ttl = redis.call('PTTL', KEYS[1])
if ttl <= 0 then return 4 end
local rawAttempts = redis.call('GET', KEYS[2])
if rawAttempts and not string.match(rawAttempts, '^[0-9]$') then return 4 end
local attempts = tonumber(rawAttempts or '0')
if attempts >= 5 then
  redis.call('DEL', KEYS[1], KEYS[2])
  return 3
end
if ARGV[2] == '1' then return 1 end
attempts = redis.call('INCR', KEYS[2])
if attempts >= 5 then
  redis.call('DEL', KEYS[1], KEYS[2])
  return 3
end
redis.call('PEXPIRE', KEYS[2], ttl)
return 2
`)

func (cache *Cache) VerifyEmailCode(ctx context.Context, key, code string) (EmailCodeStatus, error) {
	for range 3 {
		encoded, found, err := cache.Get(ctx, key)
		if err != nil || !found {
			return EmailCodeMissing, err
		}
		stored, valid := decodeEmailCode(encoded)
		if !valid {
			return EmailCodeMalformed, nil
		}
		matches := subtle.ConstantTimeCompare([]byte(stored), []byte(code))
		value, err := verifyEmailCode.Run(ctx, cache.client, []string{key, key + ":attempts"}, encoded, matches).Int()
		if err != nil {
			return EmailCodeMissing, fmt.Errorf("verify email code: %w", err)
		}
		status := EmailCodeStatus(value)
		if status == emailCodeChanged {
			continue
		}
		if status < EmailCodeMissing || status > EmailCodeMalformed {
			return EmailCodeMissing, errors.New("invalid email-code store response")
		}
		return status, nil
	}
	return EmailCodeMissing, errors.New("email code changed during verification")
}

type emailCodeRecord struct {
	Version int    `json:"version"`
	Token   string `json:"token"`
}

func (cache *Cache) Take(ctx context.Context, key string, limit int64, window time.Duration) (ratelimit.Result, error) {
	return ratelimit.TakeRedis(ctx, cache.client, key, limit, window)
}

// EncodeEmailVerificationCodeValue stores a versioned native JSON record.
func EncodeEmailVerificationCodeValue(code string) ([]byte, error) {
	if !validEmailCode(code) {
		return nil, errors.New("email verification code must contain six ASCII digits")
	}
	return json.Marshal(emailCodeRecord{Version: 1, Token: code})
}

func decodeEmailCode(encoded []byte) (string, bool) {
	if len(encoded) == 0 || len(encoded) > 4096 || !jsontext.Value(encoded).IsValid() {
		return "", false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return "", false
	}
	if len(fields) != 2 || fields["version"] == nil || fields["token"] == nil {
		return "", false
	}
	var record emailCodeRecord
	if err := json.Unmarshal(encoded, &record); err != nil {
		return "", false
	}
	if record.Version != 1 || !validEmailCode(record.Token) {
		return "", false
	}
	return record.Token, true
}

func validEmailCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, digit := range code {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}
