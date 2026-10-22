package useraccountwrites

import (
	"bytes"
	"encoding/hex"
	"testing"

	ogorek "github.com/kisielk/og-rek"
)

func TestEmailVerificationCachePickleInteroperatesWithDjango(t *testing.T) {
	// Source stores json.dumps({"token": "674219"}) through Django's
	// RedisCache, which pickles that JSON string using Python pickle protocol 5.
	pythonValue, err := hex.DecodeString("80059517000000000000008c137b22746f6b656e223a2022363734323139227d942e")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeEmailVerificationCodeValue(pythonValue)
	if err != nil || decoded != "674219" {
		t.Fatalf("decoded token=%q err=%v", decoded, err)
	}

	encoded, err := EncodeEmailVerificationCodeValue(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) < 2 || encoded[0] != 0x80 || encoded[1] != 4 {
		t.Fatalf("Django cache value did not use pickle protocol 4: %x", encoded)
	}
	decodedValue, err := ogorek.NewDecoder(bytes.NewReader(encoded)).Decode()
	if err != nil || decodedValue != `{"token":"674219"}` {
		t.Fatalf("encoded cache value=%#v err=%v, want source JSON string", decodedValue, err)
	}
	if roundTrip, err := DecodeEmailVerificationCodeValue(encoded); err != nil || roundTrip != decoded {
		t.Fatalf("round trip token=%q err=%v", roundTrip, err)
	}
}

func TestEmailVerificationCachePickleRejectsUnexpectedValues(t *testing.T) {
	for _, value := range [][]byte{
		{},
		[]byte(`{"token":"674219"}`),
		[]byte("not a pickle"),
	} {
		if _, err := DecodeEmailVerificationCodeValue(value); err == nil {
			t.Fatalf("DecodeEmailVerificationCodeValue(%q) unexpectedly succeeded", value)
		}
	}
}
