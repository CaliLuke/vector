package emailvalidation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBareMailboxContract(t *testing.T) {
	for _, email := range []string{"person@localhost", "person@LOCALHOST", "person@[2001:db8::1]", `"quoted"@example.com`, "i\u0307@example.com", "person@bücher.de", "person@例子.测试", "person@example.c"} {
		require.True(t, Valid(email), email)
	}
	for _, email := range []string{"", "missing-at", "@example.com", `"unclosed@example.com`, "has space@example.com", "comment(test)@example.com", "a@", "a@example.com.", "a@xn--.com", "a@" + strings.Repeat("a", 64) + ".com", string([]byte{0xff}) + "@example.com", "Person <person@example.com>", "a@example.com,b@example.com", "a@example..com", "a@-example.com", "a@example-.com", "a@[fe80::1%zone]", "a\x1c@example.com", "a\n@example.com", "a@example.com\r\nBcc: x@example.com", "a..b@example.com", "a@example", strings.Repeat("a", 244) + "@example.com"} {
		require.False(t, Valid(email), email)
	}
}
func TestNormalizationPreservesStoredUnicodeIdentity(t *testing.T) {
	require.Equal(t, "i\u0307@localhost", Normalize("\u2003İ@LOCALHOST\u2003"))
	require.Equal(t, "a@ος-example.com", Normalize("a@ΟΣ-EXAMPLE.COM"))
	require.Equal(t, "a@bücher.de", Normalize("A@BÜCHER.DE"))
	require.False(t, Valid(Normalize("\u001cA@EXAMPLE.COM")))
}
