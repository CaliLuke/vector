package backgroundtasks

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestPageVersionRejectsInvalidPriorDescriptionBeforeStorage(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://test:test@127.0.0.1:1/test?sslmode=disable&connect_timeout=1")
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	store, err := NewPostgresVersionHistoryStore(pool)
	require.NoError(t, err)
	for _, snapshot := range []string{`{"description_html":42}`, `{"description_html":{}}`, `{"description_html":true}`} {
		err := store.TrackPageVersion(t.Context(), PageVersionInput{PageID: uuid.NewString(), UserID: uuid.NewString(), ExistingInstance: snapshot})
		require.ErrorIs(t, err, ErrPermanentVersionHistory, "malformed stored tasks must stop before database access")
	}
}
