package assets

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/smithy-go"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestStorageDatabaseErrorsPreserveCauses(t *testing.T) {
	for _, code := range []string{"", "X", "23505"} {
		t.Run("code_"+code, func(t *testing.T) {
			cause := &pgconn.PgError{Code: code, Message: "database failure"}
			original := fmt.Errorf("operation: %w", cause)
			classified := classifyStorageMetadataError(original)
			require.ErrorIs(t, classified, ErrPermanentStorageMetadata)
			require.ErrorIs(t, classified, cause)
			require.Equal(t, ErrPermanentStorageMetadata.Error()+": "+original.Error(), classified.Error())
		})
	}

}

func TestMetadataDatabaseRetryStillClassifiesStorageCause(t *testing.T) {
	denied := &smithy.GenericAPIError{Code: "AccessDenied", Fault: smithy.FaultClient}
	original := errors.Join(&pgconn.PgError{Code: "40001"}, denied)
	classified := classifyStorageMetadataError(original)
	require.ErrorIs(t, classified, ErrPermanentStorageMetadata)
	require.ErrorIs(t, classified, original)
	require.ErrorIs(t, classified, denied)
}
