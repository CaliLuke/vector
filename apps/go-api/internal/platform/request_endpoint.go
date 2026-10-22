package platform

import "context"

type publicRequestEndpointContextKey struct{}

// WithPublicRequestEndpoint carries the browser-visible request origin to storage signers.
func WithPublicRequestEndpoint(ctx context.Context, endpoint string) context.Context {
	return context.WithValue(ctx, publicRequestEndpointContextKey{}, endpoint)
}

// PublicRequestEndpoint returns the browser-visible request origin, when available.
func PublicRequestEndpoint(ctx context.Context) (string, bool) {
	endpoint, ok := ctx.Value(publicRequestEndpointContextKey{}).(string)
	return endpoint, ok && endpoint != ""
}
