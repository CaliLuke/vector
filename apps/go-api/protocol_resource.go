package systemapi

import "context"

// ProtocolResource carries router-derived URL parameters into generated
// authentication hooks. Query parameters and request bodies cannot replace it.
type protocolResourceKey struct{}

func WithProtocolResource(ctx context.Context, resource SessionProtocolResource) context.Context {
	return context.WithValue(ctx, protocolResourceKey{}, resource)
}

func ProtocolResourceFromContext(ctx context.Context) (SessionProtocolResource, bool) {
	resource, ok := ctx.Value(protocolResourceKey{}).(SessionProtocolResource)
	return resource, ok
}

type AssetsAPIProtocolAuthorizer interface {
	AuthorizeAssetsAPIProtocol(context.Context, string, SessionProtocolResource) (int, map[string]any)
}
