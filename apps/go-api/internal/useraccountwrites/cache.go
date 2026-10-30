package useraccountwrites

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	ogorek "github.com/kisielk/og-rek"
	"github.com/redis/go-redis/v9"
)

type Cache struct{ client *redis.Client }

func NewCache(redisURL string) (*Cache, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse Django cache URL: %w", err)
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
	return cache.client.Set(ctx, key, value, ttl).Err()
}

func (cache *Cache) Delete(ctx context.Context, key string) error {
	return cache.client.Del(ctx, key).Err()
}

// EncodeEmailVerificationCodeValue uses Django's default Redis cache pickle
// serializer. Protocol 4 is understood by the configured Python runtime and
// avoids protocol 5's out-of-band buffer feature, which is unnecessary here.
func EncodeEmailVerificationCodeValue(code string) ([]byte, error) {
	payload, err := json.Marshal(map[string]string{"token": code})
	if err != nil {
		return nil, fmt.Errorf("encode Django email-code cache JSON: %w", err)
	}
	var encoded bytes.Buffer
	encoder := ogorek.NewEncoderWithConfig(&encoded, &ogorek.EncoderConfig{Protocol: 4})
	if err := encoder.Encode(string(payload)); err != nil {
		return nil, fmt.Errorf("encode Django email-code cache value: %w", err)
	}
	return encoded.Bytes(), nil
}

// DecodeEmailVerificationCodeValue accepts the source's pickled JSON string
// and returns its token without executing Python constructors or callables.
func DecodeEmailVerificationCodeValue(encoded []byte) (string, error) {
	if len(encoded) == 0 || len(encoded) > 64*1024 {
		return "", fmt.Errorf("invalid Django email-code cache value size %d", len(encoded))
	}
	value, err := ogorek.NewDecoder(bytes.NewReader(encoded)).Decode()
	if err != nil {
		return "", fmt.Errorf("decode Django email-code cache value: %w", err)
	}
	serialized, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("Django email-code cache value has type %T, want JSON string", value)
	}
	var fields map[string]any
	if err := json.Unmarshal([]byte(serialized), &fields); err != nil {
		return "", fmt.Errorf("decode Django email-code cache JSON: %w", err)
	}
	code, ok := fields["token"].(string)
	if !ok {
		return "", fmt.Errorf("Django email-code cache value token has type %T, want string", fields["token"])
	}
	return code, nil
}
