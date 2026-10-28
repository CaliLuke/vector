package property_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/stretchr/testify/require"
)

func TestDatePropertyRetainsDateTimePrecision(t *testing.T) {
	for _, input := range []string{"2026-10-27", "2026-10-27 14:35:59", "2026-10-27T14:35:59.123400+05:30", "2026-10-27T14:35:59+00:00"} {
		t.Run(input, func(t *testing.T) {
			value, err := property.ValidateValue(property.Date, property.Configuration{}, input, false, nil)
			require.NoError(t, err)
			require.Equal(t, input, value)
			read, err := property.ReadValue(property.Date, property.Configuration{}, value)
			require.NoError(t, err)
			require.Equal(t, input, read)
		})
	}
	for _, input := range []string{"2026-02-30 12:00:00", "2026-10-27 25:00:00", "0000-01-01", "2026-10-27 trailing", "9999-12-31T23:59:59-01:00"} {
		t.Run(input, func(t *testing.T) {
			_, err := property.ValidateValue(property.Date, property.Configuration{}, input, false, nil)
			require.Error(t, err)
		})
	}
}
