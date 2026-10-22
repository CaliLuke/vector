package assets

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisCacheInvalidator struct{ client *redis.Client }

func NewRedisCacheInvalidator(redisURL string) (*RedisCacheInvalidator, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse assets cache Redis URL: %w", err)
	}
	if options.TLSConfig != nil {
		options.TLSConfig.InsecureSkipVerify = true
	}
	return &RedisCacheInvalidator{client: redis.NewClient(options)}, nil
}

func (cache *RedisCacheInvalidator) Close() error { return cache.client.Close() }

func (cache *RedisCacheInvalidator) Delete(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("assets cache key is required")
	}
	return cache.client.Del(ctx, key).Err()
}
