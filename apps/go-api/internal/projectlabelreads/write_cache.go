package projectlabelreads

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type WriteCache struct{ client *redis.Client }

func NewWriteCache(redisURL string) (*WriteCache, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	if options.TLSConfig != nil {
		options.TLSConfig.InsecureSkipVerify = true
	} // Django uses ssl_cert_reqs=False for this shared cache.
	return &WriteCache{client: redis.NewClient(options)}, nil
}

func (c *WriteCache) Close() error { return c.client.Close() }

func (c *WriteCache) Invalidate(ctx context.Context, slug string, multiple bool) error {
	key := ":1:/api/workspaces/" + slug + "/labels/"
	if !multiple {
		return c.client.Del(ctx, key).Err()
	}
	// Django's cache.keys('*<path>*') scopes wildcard invalidation to the cache
	// key version, and also removes alternate representations of this workspace.
	keys, err := c.client.Keys(ctx, ":1:*"+"/api/workspaces/"+slug+"/labels/"+"*").Result()
	if err != nil || len(keys) == 0 {
		return err
	}
	return c.client.Del(ctx, keys...).Err()
}
