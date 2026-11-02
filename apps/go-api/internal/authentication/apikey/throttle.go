package apikey

import (
	"context"

	"github.com/CaliLuke/go-plane/internal/ratelimit"
)

// Rate describes remaining capacity and the next retry time.
type Rate = ratelimit.Result

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

// Throttle shares native atomic limits with the other request boundaries.
type Throttle struct{ limiter *ratelimit.Limiter }

func NewThrottle(store ratelimit.Store, rate string) (*Throttle, error) {
	limiter, err := ratelimit.New(store, rate)
	if err != nil {
		return nil, err
	}
	return &Throttle{limiter: limiter}, nil
}

func (throttle *Throttle) Allow(ctx context.Context, token string) (Rate, error) {
	return throttle.AllowScoped(ctx, "api-key", token)
}

func (throttle *Throttle) AllowScoped(ctx context.Context, scope, identity string) (Rate, error) {
	return throttle.limiter.Allow(ctx, scope, identity)
}
