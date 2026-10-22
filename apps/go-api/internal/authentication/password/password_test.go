package password

import (
	"errors"
	"strings"
	"testing"
)

func TestVerifyPythonGeneratedDjangoPBKDF2SHA256Vector(t *testing.T) {
	hasher, err := New(1000)
	if err != nil {
		t.Fatal(err)
	}
	encoded := "pbkdf2_sha256$1000$abcDEF123xyz$IZ/j26iGvZhrLfeBVUtVy/CeP/Mi+2Lb+u4TC/vpp+Q="
	valid, err := hasher.Verify(encoded, "pässword🔐")
	if err != nil || !valid {
		t.Fatalf("Verify(valid) = (%v, %v), want (true, nil)", valid, err)
	}
	valid, err = hasher.Verify(encoded, "different")
	if err != nil || valid {
		t.Fatalf("Verify(wrong password) = (%v, %v), want (false, nil)", valid, err)
	}
}

func TestEncodeProducesDjangoCompatibleHash(t *testing.T) {
	hasher, err := New(1000)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := hasher.Encode("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != Algorithm || parts[1] != "1000" {
		t.Fatalf("encoded hash has invalid Django structure: %q", encoded)
	}
	if len(parts[2]) != SaltLength {
		t.Fatalf("salt length=%d, want %d", len(parts[2]), SaltLength)
	}
	for _, char := range parts[2] {
		if !strings.ContainsRune(SaltAlphabet, char) {
			t.Fatalf("salt contains unsupported character %q", char)
		}
	}
	valid, err := hasher.Verify(encoded, "correct horse battery staple")
	if err != nil || !valid {
		t.Fatalf("Verify(Encode(password)) = (%v, %v), want (true, nil)", valid, err)
	}
}

func TestVerifyRejectsMalformedDjangoHashes(t *testing.T) {
	hasher := Default()
	for _, encoded := range []string{
		"",
		"!unusable",
		"pbkdf2_sha256$0$salt$hash",
		"pbkdf2_sha256$not-a-number$salt$hash",
		"pbkdf2_sha256$1000$$hash",
		"pbkdf2_sha256$1000$salt$%%%",
		"pbkdf2_sha1$1000$salt$hash",
	} {
		if valid, err := hasher.Verify(encoded, "password"); !errors.Is(err, ErrInvalidHash) || valid {
			t.Errorf("Verify(%q) = (%v, %v), want (false, ErrInvalidHash)", encoded, valid, err)
		}
	}
}

func TestNewRejectsInvalidIterationCount(t *testing.T) {
	for _, iterations := range []int{0, -1} {
		if _, err := New(iterations); !errors.Is(err, ErrIterations) {
			t.Errorf("New(%d) error=%v, want ErrIterations", iterations, err)
		}
	}
}
