package useridentity

import (
	"context"
	"encoding/json"
	"errors"
	"uuid"
)

// IdentityReader is the credential-neutral application boundary for reading
// one's own API identity. Transports authenticate the actor; callers cannot
// choose a different target user. The repository also checks current activity.
type IdentityReader struct{ repository apiIdentityRepository }

type apiIdentityRepository interface {
	APIV1Identity(context.Context, uuid.UUID) (json.RawMessage, error)
}

var ErrUnauthenticated = errors.New("identity requires an authenticated actor")

func NewIdentityReader(repository apiIdentityRepository) (*IdentityReader, error) {
	if repository == nil {
		return nil, errors.New("identity repository is required")
	}
	return &IdentityReader{repository: repository}, nil
}

func (r *IdentityReader) CurrentIdentity(ctx context.Context, actor uuid.UUID) (json.RawMessage, error) {
	if actor == (uuid.UUID{}) {
		return nil, ErrUnauthenticated
	}
	return r.repository.APIV1Identity(ctx, actor)
}
