package systemapi

import (
	"testing"

	"uuid"

	"github.com/stretchr/testify/require"
)

func TestEmailCodeKeysSeparateAccountsAndAddressesWithoutExposingEither(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	key := userEmailCacheKey(first, "person@example.test")
	require.Contains(t, key, "vector:email-code:v1:")
	require.NotContains(t, key, first.String())
	require.NotContains(t, key, "person@example.test")
	require.Equal(t, key, userEmailCacheKey(first, " PERSON@EXAMPLE.TEST "))
	require.NotEqual(t, key, userEmailCacheKey(second, "person@example.test"))
	require.NotEqual(t, key, userEmailCacheKey(first, "other@example.test"))
}
