package assets

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestAttachmentApplicationRejectsMalformedOrigin(t *testing.T) {
	// Pool construction is lazy; origin validation must happen without storage
	// or database I/O, so invalid configuration fails at application startup.
	pool, err := pgxpool.New(t.Context(), "postgres://fixture:fixture@unused.example.test/fixture?sslmode=disable")
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	repository, err := NewPostgresRepository(pool)
	require.NoError(t, err)
	for _, origin := range []string{"http://[invalid", "://invalid", "https://example.test/%zz"} {
		t.Run(origin, func(t *testing.T) {
			application, err := NewAttachmentApplication(repository, &Storage{}, nil, origin, AttachmentUploadLimit)
			require.Error(t, err)
			require.Nil(t, application)
		})
	}
}
