package instancecipher

import (
	"errors"
	"testing"
)

func TestReadsPythonConfigurationToken(t *testing.T) {
	c, err := New("fixture-secret")
	if err != nil {
		t.Fatal(err)
	}
	// Python cryptography.Fernet, using the source PBKDF2 parameters. Timestamp
	// 1 also proves storage tokens do not acquire an expiration during migration.
	const token = "gAAAAAAAAAAB3UcPWu87RtPc_flwnj6b3SGMloFWQyR7Ep8NvrKex5Z5rFOvGMKaT1ESlSSeReZNpZ1dxWBCxNbmYztE-VmtwWrfQKchrPxaGQfTus9pqhg="
	got, err := c.Decrypt(token)
	if err != nil || got != "SMTP пароль 🎵" {
		t.Fatalf("Python token: got %q, err %v", got, err)
	}
	for _, value := range []string{"", "password", "SMTP пароль 🎵"} {
		encrypted, err := c.Encrypt(value)
		if err != nil {
			t.Fatal(err)
		}
		got, err := c.Decrypt(encrypted)
		if err != nil || got != value {
			t.Fatalf("round trip: got %q, err %v", got, err)
		}
	}
}

func TestRejectsMalformedAuthenticatedTokens(t *testing.T) {
	c := &Cipher{}
	for i := range c.key {
		c.key[i] = byte(i)
	}
	const valid = "gAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAGWd0uwn1ncxmt8qIH-KbuBv0EowL7JYWjVAGHu4cPnj4oeGE0Ko5aqiSlYJlxs9xg=="
	got, err := c.Decrypt(valid)
	if err != nil || got != "hello" {
		t.Fatalf("valid control: got %q, err %v", got, err)
	}
	for name, token := range map[string]string{
		"zero padding":      "gAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAF6IH-a7_iBsvZnOZZ93mxf2cdOXED4grcijeQeIZoVvFbdcxCHBFCDfAhLgzCeEtQ==",
		"oversized padding": "gAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAABCp7KCV1IPI33HqLkGhzSja-16QfmiCm4HsSn3tgMmTbXjkhFI-ox9hUCP6FX7SI_M_5DcvVS3bM1vUQC0PATo=",
		"short frame":       "gGFiYw==",
		"invalid base64":    valid + "!",
		"invalid version":   "h" + valid[1:],
	} {
		t.Run(name, func(t *testing.T) {
			got, err := c.Decrypt(token)
			if got != "" || !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("malformed token accepted: got %q, err %v", got, err)
			}
		})
	}
	c.key[0] ^= 1
	if got, err := c.Decrypt(valid); got != "" || !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("wrong signing key accepted: got %q, err %v", got, err)
	}
}
