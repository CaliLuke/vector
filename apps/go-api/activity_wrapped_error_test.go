package systemapi

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActivityProtocolPreservesWrappedPublicErrors(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 429} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			failure := activityStatusError(status, map[string]any{"detail": "Public response"})
			actual, body := activityErrorStatusAndBody(fmt.Errorf("private diagnostic: %w", failure))
			require.Equal(t, status, actual)
			require.Equal(t, "Public response", body["detail"])
			require.NotContains(t, fmt.Sprint(body), "private diagnostic")
		})
	}
}

func TestActivityProtocolSanitizesWrappedUnknownErrors(t *testing.T) {
	status, body := activityErrorStatusAndBody(fmt.Errorf("private diagnostic: %w", errors.New("private database detail")))
	require.Equal(t, 500, status)
	require.Equal(t, map[string]any{"error": "Something went wrong please try again later"}, body)
}
