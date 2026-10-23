package apikey

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/pythoncompat"
)

type throttleCache interface {
	Get(context.Context, string) ([]byte, bool, error)
	Set(context.Context, string, []byte, time.Duration) error
}

// Rate describes Django's response headers and optional rejection wait.
type Rate struct {
	Remaining string
	Reset     int64
	Wait      time.Duration
	Allowed   bool
}

// Effects is request-scoped mutable state shared with the HTTP response
// boundary. The endpoint authorizer fills it before response encoding.
type Effects struct {
	Authenticated bool
	Rate          *Rate
}

type effectsContextKey struct{}

func WithEffects(ctx context.Context, effects *Effects) context.Context {
	return context.WithValue(ctx, effectsContextKey{}, effects)
}

func EffectsFromContext(ctx context.Context) *Effects {
	effects, _ := ctx.Value(effectsContextKey{}).(*Effects)
	return effects
}

// Throttle matches DRF SimpleRateThrottle and Django's cache key/pickle
// representation for ApiKeyRateThrottle.
type Throttle struct {
	cache    throttleCache
	requests *big.Int
	window   time.Duration
	now      func() time.Time
}

func NewThrottle(cache throttleCache, rate string) (*Throttle, error) {
	if cache == nil {
		return nil, fmt.Errorf("API-key throttle cache is required")
	}
	requests, window, err := parseRate(rate)
	if err != nil {
		return nil, err
	}
	return &Throttle{cache: cache, requests: requests, window: window, now: time.Now}, nil
}

func (throttle *Throttle) Allow(ctx context.Context, token string) (Rate, error) {
	return throttle.AllowCacheKey(ctx, ":1:api_key:"+token, "API-key")
}

// AllowCacheKey applies the shared DRF throttle window at a caller-supplied
// Django cache key. Callers own the exact key format for their throttle scope.
func (throttle *Throttle) AllowCacheKey(ctx context.Context, key, scope string) (Rate, error) {
	data, found, err := throttle.cache.Get(ctx, key)
	if err != nil {
		return Rate{}, fmt.Errorf("read Django %s throttle: %w", scope, err)
	}
	var history []float64
	if found {
		history, err = decodeThrottlePickle(data)
		if err != nil {
			return Rate{}, fmt.Errorf("decode Django %s throttle: %w", scope, err)
		}
	}
	now := float64(throttle.now().UnixNano()) / float64(time.Second)
	cutoff := now - throttle.window.Seconds()
	for len(history) > 0 && history[len(history)-1] <= cutoff {
		history = history[:len(history)-1]
	}
	historyLength := big.NewInt(int64(len(history)))
	if historyLength.Cmp(throttle.requests) >= 0 {
		availableRequests := new(big.Int).Sub(throttle.requests, historyLength)
		availableRequests.Add(availableRequests, big.NewInt(1))
		if availableRequests.Sign() <= 0 {
			return Rate{Allowed: false}, nil
		}
		remaining := throttle.window.Seconds()
		if len(history) > 0 {
			remaining -= now - history[len(history)-1]
		}
		wait := remaining / float64(availableRequests.Int64())
		return Rate{Allowed: false, Wait: time.Duration(math.Ceil(wait)) * time.Second}, nil
	}
	history = append([]float64{now}, history...)
	if err := throttle.cache.Set(ctx, key, encodeThrottlePickle(history), throttle.window); err != nil {
		return Rate{}, fmt.Errorf("write Django %s throttle: %w", scope, err)
	}
	return Rate{
		Allowed:   true,
		Remaining: new(big.Int).Sub(throttle.requests, big.NewInt(int64(len(history)))).String(),
		Reset:     int64(now + throttle.window.Seconds()),
	}, nil
}

func parseRate(rate string) (*big.Int, time.Duration, error) {
	parts := strings.Split(rate, "/")
	if len(parts) != 2 {
		return nil, 0, fmt.Errorf("invalid API-key rate %q", rate)
	}
	count, err := pythoncompat.DecimalInteger(parts[0])
	if err != nil {
		return nil, 0, fmt.Errorf("invalid API-key rate %q", rate)
	}
	period := parts[1]
	if period == "" {
		return nil, 0, fmt.Errorf("invalid API-key rate period %q", period)
	}
	var window time.Duration
	switch period[0] {
	case 's':
		window = time.Second
	case 'm':
		window = time.Minute
	case 'h':
		window = time.Hour
	case 'd':
		window = 24 * time.Hour
	default:
		return nil, 0, fmt.Errorf("invalid API-key rate period %q", period)
	}
	return count, window, nil
}
func encodeThrottlePickle(values []float64) []byte {
	return pythoncompat.EncodeFloatListPickle(values)
}

func decodeThrottlePickle(data []byte) ([]float64, error) {
	return pythoncompat.DecodeFloatListPickle(data)
}
