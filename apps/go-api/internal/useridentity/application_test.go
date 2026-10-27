package useridentity

import (
	"context"
	"encoding/json"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type identityApplicationRepository struct{ actor uuid.UUID }

func (r *identityApplicationRepository) APIV1Identity(_ context.Context, actor uuid.UUID) (json.RawMessage, error) {
	r.actor = actor
	return json.RawMessage(`{"first_name":"Current user"}`), nil
}

func TestIdentityApplicationRequiresAndReadsOnlyItsAuthenticatedActor(t *testing.T) {
	repository := &identityApplicationRepository{}
	reader, err := NewIdentityReader(repository)
	require.NoError(t, err)
	_, err = reader.CurrentIdentity(t.Context(), uuid.UUID{})
	require.ErrorIs(t, err, ErrUnauthenticated)
	assert.Equal(t, uuid.UUID{}, repository.actor)
	actor := uuid.UUID{1}
	body, err := reader.CurrentIdentity(t.Context(), actor)
	require.NoError(t, err)
	assert.Equal(t, actor, repository.actor)
	assert.JSONEq(t, `{"first_name":"Current user"}`, string(body))
	_, err = NewIdentityReader(nil)
	require.Error(t, err)
}
