// Package password implements Django's default PBKDF2-SHA256 password format.
package password

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

const (
	Algorithm         = "pbkdf2_sha256"
	DefaultIterations = 1_000_000
	SaltLength        = 22
	SaltAlphabet      = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	derivedKeyLength  = sha256.Size
)

var (
	ErrInvalidHash = errors.New("invalid Django PBKDF2 password hash")
	ErrIterations  = errors.New("Django PBKDF2 iteration count is outside supported range")
)

// Hasher encodes and verifies Django's default password hash format.
type Hasher struct {
	iterations int
	random     io.Reader
}

// New creates a Django-compatible hasher using the configured iteration
// count. Django 5.2 uses 1,000,000 iterations by default.
func New(iterations int) (*Hasher, error) {
	if iterations < 1 {
		return nil, ErrIterations
	}
	return &Hasher{iterations: iterations, random: rand.Reader}, nil
}

// Default creates a hasher using Django 5.2's default PBKDF2 iteration count.
func Default() *Hasher {
	return &Hasher{iterations: DefaultIterations, random: rand.Reader}
}

// Encode returns a Django PBKDF2-SHA256 hash with a 128-bit-entropy salt.
func (hasher *Hasher) Encode(raw string) (string, error) {
	if hasher == nil || hasher.iterations < 1 || hasher.random == nil {
		return "", errors.New("Django PBKDF2 password hasher is not initialized")
	}
	salt, err := randomSalt(hasher.random)
	if err != nil {
		return "", fmt.Errorf("generate Django PBKDF2 salt: %w", err)
	}
	return encode(raw, salt, hasher.iterations)
}

// Verify checks an encoded Django PBKDF2-SHA256 hash. A malformed hash is an
// error; a well-formed hash with a different password returns false,nil.
func (hasher *Hasher) Verify(encoded, raw string) (bool, error) {
	if hasher == nil {
		return false, errors.New("Django PBKDF2 password hasher is not initialized")
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != Algorithm || parts[2] == "" || strings.Contains(parts[2], "$") {
		return false, ErrInvalidHash
	}
	iterations, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil || iterations < 1 || iterations > math.MaxInt32 {
		return false, ErrInvalidHash
	}
	if _, err := base64.StdEncoding.DecodeString(parts[3]); err != nil {
		return false, ErrInvalidHash
	}
	calculated, err := encode(raw, parts[2], int(iterations))
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare([]byte(calculated), []byte(encoded)) == 1, nil
}

func encode(raw, salt string, iterations int) (string, error) {
	key, err := pbkdf2.Key(sha256.New, raw, []byte(salt), iterations, derivedKeyLength)
	if err != nil {
		return "", fmt.Errorf("derive Django PBKDF2-SHA256 hash: %w", err)
	}
	return Algorithm + "$" + strconv.Itoa(iterations) + "$" + salt + "$" + base64.StdEncoding.EncodeToString(key), nil
}

func randomSalt(source io.Reader) (string, error) {
	const alphabetSize = byte(len(SaltAlphabet))
	// Rejection sampling avoids the character bias of byte modulo alphabetSize.
	limit := byte(256 - (256 % int(alphabetSize)))
	salt := make([]byte, 0, SaltLength)
	buffer := make([]byte, 64)
	for len(salt) < SaltLength {
		if _, err := io.ReadFull(source, buffer); err != nil {
			return "", err
		}
		for _, value := range buffer {
			if value >= limit {
				continue
			}
			salt = append(salt, SaltAlphabet[int(value%alphabetSize)])
			if len(salt) == SaltLength {
				break
			}
		}
	}
	return string(salt), nil
}
