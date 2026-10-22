package projectmembershipreads

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// InvitationCache implements the key/version and wildcard behavior of
// plane.utils.cache. The service chooses paths and invalidation timing.
type InvitationCache struct{ client *redis.Client }

func NewInvitationCache(redisURL string) (*InvitationCache, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse invitation cache Redis URL: %w", err)
	}
	if options.TLSConfig != nil {
		// Match Django's ssl_cert_reqs=False for the shared application cache.
		options.TLSConfig.InsecureSkipVerify = true
	}
	return &InvitationCache{client: redis.NewClient(options)}, nil
}

func (cache *InvitationCache) Close() error { return cache.client.Close() }

func (cache *InvitationCache) Invalidate(ctx context.Context, path, userID string, multiple bool) error {
	if path == "" {
		return errors.New("invitation cache path is required")
	}
	key := path
	if userID != "" {
		key += ":" + userID
	}
	if !multiple {
		return cache.client.Del(ctx, ":1:"+key).Err()
	}
	keys, err := cache.client.Keys(ctx, ":1:*"+key+"*").Result()
	if err != nil || len(keys) == 0 {
		return err
	}
	return cache.client.Del(ctx, keys...).Err()
}
