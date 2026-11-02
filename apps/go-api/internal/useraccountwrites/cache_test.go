package useraccountwrites

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmailCodeCacheUsesVersionedNativeJSON(t *testing.T) {
	for _, code := range []string{"000001", "674219", "999999"} {
		encoded, err := EncodeEmailVerificationCodeValue(code)
		require.NoError(t, err)
		require.JSONEq(t, `{"version":1,"token":"`+code+`"}`, string(encoded))
		decoded, valid := decodeEmailCode(encoded)
		require.True(t, valid)
		require.Equal(t, code, decoded)
	}
	for _, code := range []string{"", "1", "1234567", "12a456", "１２３４５６"} {
		_, err := EncodeEmailVerificationCodeValue(code)
		require.Error(t, err)
	}
}

func TestEmailCodeCacheRejectsCorruptOrUnsupportedRecords(t *testing.T) {
	for _, value := range []string{
		"", "not JSON", strings.Repeat("x", 4097), `[]`, `null`,
		`{"token":"674219"}`, `{"version":2,"token":"674219"}`,
		`{"version":1,"token":"674219","extra":true}`, `{"Version":1,"token":"674219"}`,
		`{"version":1,"token":674219}`, `{"version":1.5,"token":"674219"}`,
		`{"version":1,"token":null}`, `{"version":1,"token":"invalid"}`,
		`{"version":1,"token":"674219","token":"123456"}`,
		`{"version":1,"token":"674219"} {}`, string([]byte{0x80, 0x04, 0x95, 0x00}),
	} {
		_, valid := decodeEmailCode([]byte(value))
		require.False(t, valid, value)
	}
}

func TestInvalidAccountCacheConfigurationKeepsCredentialsOutOfErrors(t *testing.T) {
	_, err := NewCache("redis://user:synthetic-password@cache.example.test:invalid/")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "synthetic-password")
}
