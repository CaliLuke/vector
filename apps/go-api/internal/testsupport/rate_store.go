package testsupport

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/CaliLuke/go-plane/internal/ratelimit"
)

type RateCall struct {
	Key    string
	Limit  int64
	Window time.Duration
}

// RateStoreFake provides non-expiring capacity for request-boundary tests.
// The native Redis tests own sliding-window expiry and distributed concurrency.
// Configure it before requests, then inspect Calls after those requests finish.
type RateStoreFake struct {
	mu     sync.Mutex
	Counts map[string]int64
	Calls  []RateCall
	Err    error
	Now    func() time.Time
}

func (store *RateStoreFake) Take(ctx context.Context, key string, limit int64, window time.Duration) (ratelimit.Result, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return ratelimit.Result{}, err
	}
	store.Calls = append(store.Calls, RateCall{key, limit, window})
	if store.Err != nil {
		return ratelimit.Result{}, store.Err
	}
	if store.Counts == nil {
		store.Counts = map[string]int64{}
	}
	now := time.Now()
	if store.Now != nil {
		now = store.Now()
	}
	reset := (now.UnixMilli() + window.Milliseconds() + 999) / 1000
	if store.Counts[key] >= limit {
		return ratelimit.Result{Remaining: "0", Reset: reset, Wait: window}, nil
	}
	store.Counts[key]++
	return ratelimit.Result{Allowed: true, Remaining: strconv.FormatInt(limit-store.Counts[key], 10), Reset: reset}, nil
}
