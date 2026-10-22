package platform

import (
	"context"
	"net/http"
	"uuid"
)

const RequestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

// RequestID returns the correlation ID assigned to the current request.
func RequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDContextKey{}).(string)
	return requestID, ok
}

// RequestIDMiddleware accepts a valid upstream UUID or assigns a UUIDv7.
// It also returns the canonical value in the X-Request-ID response header.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := requestIDFromHeader(r.Header.Get(RequestIDHeader))
		w.Header().Set(RequestIDHeader, requestID)
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestIDFromHeader(value string) string {
	requestID, err := uuid.Parse(value)
	if err != nil {
		return uuid.NewV7().String()
	}
	return requestID.String()
}
