package password

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestEncodeRejectsBiasedEntropyBytesAndRefills(t *testing.T) {
	// A full rejected block requires another read. 247 is the last unbiased
	// byte; 248 through 255 must not become alphabet characters by modulo.
	entropy := bytes.Repeat([]byte{255}, 128)
	copy(entropy[:8], []byte{248, 249, 250, 251, 252, 253, 254, 255})
	entropy[64] = 247
	copy(entropy[65:], []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20})
	hasher, err := New(1000)
	if err != nil {
		t.Fatal(err)
	}
	hasher.random = bytes.NewReader(entropy)
	encoded, err := hasher.Encode("test password")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[2] != "9abcdefghijklmnopqrstu" {
		t.Fatalf("unexpected rejection-sampled salt in %q", encoded)
	}
}

func TestEncodeFailsWithoutCompleteEntropyBlock(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		want error
	}{
		{"empty source", nil, io.EOF},
		{"partial source", []byte{0}, io.ErrUnexpectedEOF},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hasher, err := New(1000)
			if err != nil {
				t.Fatal(err)
			}
			hasher.random = bytes.NewReader(tc.data)
			encoded, err := hasher.Encode("test password")
			if encoded != "" || !errors.Is(err, tc.want) {
				t.Fatalf("incomplete entropy produced hash %q with error %v; want no hash and %v", encoded, err, tc.want)
			}
		})
	}
}
