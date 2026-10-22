package systemapi

import (
	"bytes"
	"context"
	"io"
	"net/url"
)

type readServiceSessionPrincipalKey struct{}
type readServiceQueryPresenceKey struct{}
type readServiceDjangoHTML500MarkerKey struct{}

type readServiceDjangoHTML500Marker struct {
	failed bool
}

func WithReadServiceDjangoHTML500(ctx context.Context) context.Context {
	return context.WithValue(ctx, readServiceDjangoHTML500MarkerKey{}, &readServiceDjangoHTML500Marker{})
}

func MarkReadServiceDjangoHTML500(ctx context.Context) {
	if marker, ok := ctx.Value(readServiceDjangoHTML500MarkerKey{}).(*readServiceDjangoHTML500Marker); ok {
		marker.failed = true
	}
}

func ReadServiceDjangoHTML500(ctx context.Context) bool {
	marker, ok := ctx.Value(readServiceDjangoHTML500MarkerKey{}).(*readServiceDjangoHTML500Marker)
	return ok && marker.failed
}

func WithReadServiceQueryPresence(ctx context.Context, query url.Values) context.Context {
	return context.WithValue(ctx, readServiceQueryPresenceKey{}, query)
}

func ReadServiceQueryPresent(ctx context.Context, name string) bool {
	query, ok := ctx.Value(readServiceQueryPresenceKey{}).(url.Values)
	if !ok {
		return false
	}
	_, present := query[name]
	return present
}

func ReadServiceQueryValues(ctx context.Context) url.Values {
	query, _ := ctx.Value(readServiceQueryPresenceKey{}).(url.Values)
	return query
}
func readServiceOptionalQuery(ctx context.Context, name string, value *string) *string {
	if value != nil || !ReadServiceQueryPresent(ctx, name) {
		return value
	}
	empty := ""
	return &empty
}

type readServiceSessionIdentity struct {
	userID   string
	timezone string
}

func WithReadServiceSessionPrincipal(ctx context.Context, userID, timezone string) context.Context {
	return context.WithValue(ctx, readServiceSessionPrincipalKey{}, readServiceSessionIdentity{
		userID:   userID,
		timezone: timezone,
	})
}

// ReadServiceSessionUserID returns the user ID validated by the shared Django
// session middleware. Feature domains may use it after that middleware has
// established the principal; it must never be populated from a raw cookie.
func ReadServiceSessionUserID(ctx context.Context) (string, bool) {
	userID, _, ok := readServiceSessionPrincipal(ctx)
	return userID, ok
}

// ReadServiceSessionTimezone returns the authenticated principal's timezone.
func ReadServiceSessionTimezone(ctx context.Context) (string, bool) {
	_, timezone, ok := readServiceSessionPrincipal(ctx)
	return timezone, ok
}

func readServiceSessionPrincipal(ctx context.Context) (string, string, bool) {
	identity, ok := ctx.Value(readServiceSessionPrincipalKey{}).(readServiceSessionIdentity)
	return identity.userID, identity.timezone, ok
}

func rawJSONReadCloser(data []byte) io.ReadCloser {
	if bytes.Contains(data, []byte("\u2028")) {
		data = bytes.ReplaceAll(data, []byte("\u2028"), []byte(`\u2028`))
	}
	if bytes.Contains(data, []byte("\u2029")) {
		data = bytes.ReplaceAll(data, []byte("\u2029"), []byte(`\u2029`))
	}
	return io.NopCloser(bytes.NewReader(data))
}
