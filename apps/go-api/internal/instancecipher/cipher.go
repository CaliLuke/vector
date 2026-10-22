// Package instancecipher reads and writes the existing encrypted instance
// configuration format. Fernet owns the encryption and authentication protocol.
package instancecipher

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"unicode/utf8"

	"github.com/fernet/fernet-go"
)

var ErrInvalidToken = errors.New("invalid encrypted instance configuration")

type Cipher struct {
	key fernet.Key
}

func New(secret string) (*Cipher, error) {
	// These parameters are the existing Django storage contract, not new KDF defaults.
	key, err := pbkdf2.Key(sha256.New, secret, []byte("salt"), 100000, 32)
	if err != nil {
		return nil, err
	}
	return &Cipher{key: fernet.Key(key)}, nil
}

func (c *Cipher) Encrypt(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if !utf8.ValidString(value) {
		return "", ErrInvalidToken
	}
	token, err := fernet.EncryptAndSign([]byte(value), &c.key)
	return string(token), err
}

func (c *Cipher) Decrypt(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	raw, err := base64.URLEncoding.DecodeString(value)
	// Fernet v1 has 57 bytes of envelope and at least one 16-byte ciphertext
	// block. Guard malformed frames before the library slices them (#6).
	// Remove this guard only after a pinned release rejects these frames safely:
	// https://github.com/fernet/fernet-go/issues/6
	const envelopeSize, blockSize = 57, 16
	if err != nil || len(raw) < envelopeSize+blockSize || (len(raw)-envelopeSize)%blockSize != 0 {
		return "", ErrInvalidToken
	}
	plaintext := fernet.VerifyAndDecrypt([]byte(value), 0, []*fernet.Key{&c.key})
	if plaintext == nil {
		return "", ErrInvalidToken
	}
	// The library validates repeated padding bytes but misses the legal range.
	// Infer the number it removed without implementing padding or decryption.
	// Remove after a pinned release fixes https://github.com/fernet/fernet-go/issues/20.
	paddingSize := len(raw) - envelopeSize - len(plaintext)
	if paddingSize < 1 || paddingSize > blockSize || !utf8.Valid(plaintext) {
		return "", ErrInvalidToken
	}
	return string(plaintext), nil
}
