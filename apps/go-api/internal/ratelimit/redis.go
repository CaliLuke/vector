package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Redis time orders requests consistently across application processes. The
// script trims expired entries, checks capacity and records admission atomically.
var slidingWindow = redis.NewScript(`
local clock = redis.call('TIME')
local now = tonumber(clock[1]) * 1000 + math.floor(tonumber(clock[2]) / 1000)
local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
if limit == 0 then
  return {0, 0, math.ceil((now + window) / 1000), window}
end
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now - window)
local count = redis.call('ZCARD', KEYS[1])
local first = redis.call('ZRANGE', KEYS[1], 0, 0, 'WITHSCORES')
local expires = now + window
if #first > 0 then expires = tonumber(first[2]) + window end
if count >= limit then
  return {0, 0, math.ceil(expires / 1000), math.max(1, expires - now)}
end
redis.call('ZADD', KEYS[1], now, ARGV[3])
redis.call('PEXPIRE', KEYS[1], window)
return {1, limit - count - 1, math.ceil(expires / 1000), 0}
`)

type RedisStore struct{ client *redis.Client }

func NewRedisStore(address string) (*RedisStore, error) {
	options, err := redis.ParseURL(address)
	if err != nil {
		return nil, errors.New("invalid rate-limit Redis URL")
	}
	return &RedisStore{client: redis.NewClient(options)}, nil
}

func (store *RedisStore) Close() error { return store.client.Close() }

func (store *RedisStore) Ping(ctx context.Context) error { return store.client.Ping(ctx).Err() }

func (store *RedisStore) Take(ctx context.Context, key string, limit int64, window time.Duration) (Result, error) {
	return TakeRedis(ctx, store.client, key, limit, window)
}

// TakeRedis lets a cache owner reuse its existing Redis connection and lifetime.
func TakeRedis(ctx context.Context, client redis.Scripter, key string, limit int64, window time.Duration) (Result, error) {
	if key == "" || limit < 0 || limit > 2147483647 || window < time.Millisecond || window > 24*time.Hour {
		return Result{}, errors.New("invalid rate-limit bucket configuration")
	}
	values, err := slidingWindow.Run(ctx, client, []string{key}, window.Milliseconds(), limit, uuid.NewString()).Int64Slice()
	if err != nil {
		return Result{}, fmt.Errorf("update rate-limit bucket: %w", err)
	}
	if len(values) != 4 || values[0] < 0 || values[0] > 1 || values[1] < 0 || values[1] > limit || values[2] < 0 || values[3] < 0 || values[3] > window.Milliseconds() {
		return Result{}, errors.New("invalid rate-limit store response")
	}
	return Result{
		Allowed: values[0] == 1, Remaining: strconv.FormatInt(values[1], 10), Reset: values[2],
		Wait: time.Duration((values[3]+999)/1000) * time.Second,
	}, nil
}
