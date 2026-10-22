package content

import (
	"encoding/base64"
	"errors"
	"strings"
	"unicode/utf8"
)

const (
	MaxContentBytes     = 10 * 1024 * 1024
	MaxHTMLNestingDepth = 4096
)

var (
	ErrInvalidBase64          = errors.New("Invalid base64 encoding")
	ErrBinaryTooLarge         = errors.New("Binary data exceeds maximum size limit (10MB)")
	ErrBinaryTooShort         = errors.New("Binary data too short to be valid document format")
	ErrSuspiciousBinary       = errors.New("Binary data contains suspicious content patterns")
	ErrHTMLTooLarge           = errors.New("HTML content exceeds maximum size limit (10MB)")
	ErrHTMLTooDeep            = errors.New("HTML content exceeds maximum nesting depth (4096 elements)")
	ErrHTMLSanitizationFailed = errors.New("Failed to sanitize HTML")
)

var suspiciousBinaryPatterns = [...]string{
	"<html",
	"<!doctype",
	"<script",
	"javascript:",
	"data:",
	"<iframe",
}

// ValidateBinary applies Plane's validation to decoded binary data. Empty data
// is valid because the Django validator treats an empty value as absent.
func ValidateBinary(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if len(data) > MaxContentBytes {
		return ErrBinaryTooLarge
	}
	if len(data) < 4 {
		return ErrBinaryTooShort
	}

	text := firstDecodedRunes(data, 200)
	text = strings.ToLower(text)
	for _, pattern := range suspiciousBinaryPatterns {
		if strings.Contains(text, pattern) {
			return ErrSuspiciousBinary
		}
	}
	return nil
}

// ValidateBase64Binary decodes a string with Python base64.b64decode semantics,
// then applies ValidateBinary.
func ValidateBase64Binary(value string) error {
	_, err := DecodeBase64Binary(value)
	return err
}

// DecodeBase64Binary decodes source-compatible base64 content and applies the
// same document validation used by ValidateBase64Binary.
func DecodeBase64Binary(value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	decoded, err := decodePythonBase64(value)
	if err != nil {
		return nil, ErrInvalidBase64
	}
	if len(decoded) == 0 {
		return nil, ErrBinaryTooShort
	}
	if err := ValidateBinary(decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func decodePythonBase64(value string) ([]byte, error) {
	decoded := make([]byte, 0, base64.StdEncoding.DecodedLen(len(value)))
	quad := make([]byte, 0, 4)
	padding := false
	for i := 0; i < len(value); i++ {
		b := value[i]
		if b >= utf8.RuneSelf {
			return nil, ErrInvalidBase64
		}
		if b == '=' {
			switch len(quad) {
			case 2:
				if padding {
					return decodePaddedQuad(decoded, quad, "==")
				}
				padding = true
			case 3:
				return decodePaddedQuad(decoded, quad, "=")
			}
			continue
		}
		if !isBase64DataByte(b) {
			continue
		}
		if padding {
			return nil, ErrInvalidBase64
		}
		quad = append(quad, b)
		if len(quad) == 4 {
			var block [3]byte
			n, err := base64.StdEncoding.Decode(block[:], quad)
			if err != nil {
				return nil, err
			}
			decoded = append(decoded, block[:n]...)
			quad = quad[:0]
		}
	}
	if len(quad) != 0 || padding {
		return nil, ErrInvalidBase64
	}
	return decoded, nil
}

func decodePaddedQuad(prefix, data []byte, padding string) ([]byte, error) {
	quad := append(append(make([]byte, 0, 4), data...), padding...)
	var block [3]byte
	n, err := base64.StdEncoding.Decode(block[:], quad)
	if err != nil {
		return nil, err
	}
	return append(prefix, block[:n]...), nil
}

func isBase64DataByte(b byte) bool {
	return b >= 'A' && b <= 'Z' ||
		b >= 'a' && b <= 'z' ||
		b >= '0' && b <= '9' ||
		b == '+' || b == '/'
}

func firstDecodedRunes(data []byte, limit int) string {
	var builder strings.Builder
	for len(data) > 0 && limit > 0 {
		r, size := utf8.DecodeRune(data)
		data = data[size:]
		if r == utf8.RuneError && size == 1 {
			continue
		}
		builder.WriteRune(r)
		limit--
	}
	return builder.String()
}
