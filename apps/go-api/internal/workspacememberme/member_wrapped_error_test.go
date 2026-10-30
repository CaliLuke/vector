package workspacememberme

import (
	"fmt"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestWrappedMissingWorkspaceRetainsInactiveMemberResponse(t *testing.T) {
	database := &memberMeDB{row: memberMeRow(func(...any) error { return fmt.Errorf("query context: %w", pgx.ErrNoRows) })}
	repository, err := NewRepository(database)
	require.NoError(t, err)
	response, err := repository.Get(t.Context(), uuid.Nil(), "missing")
	require.NoError(t, err)
	require.JSONEq(t, `{"company_role":"","created_by":null,"default_props":null,"deleted_at":null,"explored_features":null,"getting_started_checklist":null,"is_active":false,"issue_props":null,"member":null,"role":null,"tips":null,"updated_by":null,"view_props":null,"workspace":null}`, string(response))
}
