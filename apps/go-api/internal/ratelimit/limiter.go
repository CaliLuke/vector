// Package ratelimit owns atomic sliding-window request limits.
package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Result struct {
	Remaining string
	Reset     int64
	Wait      time.Duration
	Allowed   bool
}

// Store admits and records a request atomically, using its authoritative clock.
type Store interface {
	Take(context.Context, string, int64, time.Duration) (Result, error)
}

type Limiter struct {
	store  Store
	limit  int64
	window time.Duration
}

func New(store Store, rate string) (*Limiter, error) {
	if store == nil {
		return nil, errors.New("rate-limit store is required")
	}
	limit, window, err := parseRate(rate)
	if err != nil {
		return nil, err
	}
	return &Limiter{store: store, limit: limit, window: window}, nil
}

func (limiter *Limiter) Allow(ctx context.Context, scope, identity string) (Result, error) {
	key, err := BucketKey(scope, identity)
	if err != nil {
		return Result{}, err
	}
	result, err := limiter.store.Take(ctx, key, limiter.limit, limiter.window)
	if err != nil {
		return Result{}, fmt.Errorf("apply %s rate limit: %w", scope, err)
	}
	return result, nil
}

// BucketKey separates request scopes without putting credentials or addresses
// into Redis key names. Versioning keeps obsolete cache formats out of the store.
func BucketKey(scope, identity string) (string, error) {
	if scope == "" || len(scope) > 64 || identity == "" {
		return "", errors.New("rate-limit scope and identity are required")
	}
	for _, c := range scope {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return "", errors.New("invalid rate-limit scope")
		}
	}
	digest := sha256.Sum256([]byte(identity))
	return "vector:rate:v1:" + scope + ":" + hex.EncodeToString(digest[:]), nil
}

func parseRate(rate string) (int64, time.Duration, error) {
	parts := strings.Split(rate, "/")
	if len(parts) != 2 || parts[0] == "" {
		return 0, 0, errors.New("rate must be an ASCII request count and period")
	}
	for _, c := range parts[0] {
		if c < '0' || c > '9' {
			return 0, 0, errors.New("rate count must be an unsigned decimal integer")
		}
	}
	count, err := strconv.ParseUint(parts[0], 10, 31)
	if err != nil {
		return 0, 0, errors.New("rate count exceeds the supported range")
	}
	var window time.Duration
	switch parts[1] {
	case "s", "sec", "second", "seconds":
		window = time.Second
	case "m", "min", "minute", "minutes":
		window = time.Minute
	case "h", "hour", "hours":
		window = time.Hour
	case "d", "day", "days":
		window = 24 * time.Hour
	default:
		return 0, 0, errors.New("rate period must be seconds, minutes, hours or days")
	}
	return int64(count), window, nil
}
