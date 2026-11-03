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

// ValidateBinary checks document size and rejects embedded executable content.
// Empty data represents an absent document.
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

// ValidateBase64Binary validates standard padded base64 document content.
func ValidateBase64Binary(value string) error {
	_, err := DecodeBase64Binary(value)
	return err
}

// DecodeBase64Binary decodes standard padded base64 content and applies the
// same document validation used by ValidateBase64Binary.
func DecodeBase64Binary(value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(value)
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
