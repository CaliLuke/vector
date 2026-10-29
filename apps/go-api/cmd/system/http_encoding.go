package main

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"
)

func djangoJSONColonSpacing(body []byte) []byte {
	spaced := make([]byte, 0, len(body)+bytes.Count(body, []byte(":")))
	inString := false
	escaped := false
	for _, value := range body {
		spaced = append(spaced, value)
		if inString {
			if escaped {
				escaped = false
			} else if value == '\\' {
				escaped = true
			} else if value == '"' {
				inString = false
			}
			continue
		}
		if value == '"' {
			inString = true
		} else if value == ':' {
			spaced = append(spaced, ' ')
		}
	}
	return spaced
}

func prependAcceptEncodingVary(header http.Header) {
	const value = "Accept-Encoding"
	values := []string{value}
	for _, existing := range strings.Split(header.Get("Vary"), ",") {
		existing = strings.TrimSpace(existing)
		if existing != "" && !strings.EqualFold(existing, value) {
			values = append(values, existing)
		}
	}
	header.Set("Vary", strings.Join(values, ", "))
}

func acceptsGzipEncoding(header string) bool {
	for offset := 0; offset < len(header); {
		match := strings.Index(header[offset:], "gzip")
		if match < 0 {
			return false
		}
		start := offset + match
		end := start + len("gzip")
		leftWord := false
		if start > 0 {
			character, _ := utf8.DecodeLastRuneInString(header[:start])
			leftWord = djangoRegexWordCharacter(character)
		}
		rightWord := false
		if end < len(header) {
			character, _ := utf8.DecodeRuneInString(header[end:])
			rightWord = djangoRegexWordCharacter(character)
		}
		if !leftWord && !rightWord {
			return true
		}
		offset = start + 1
	}
	return false
}

func djangoRegexWordCharacter(character rune) bool {
	return character == '_' || unicode.IsLetter(character) || unicode.IsNumber(character)
}

func djangoGzip(body []byte, paddingLength func() (int, error)) ([]byte, error) {
	var compressed bytes.Buffer
	compressor := gzip.NewWriter(&compressed)
	if _, err := compressor.Write(body); err != nil {
		return nil, err
	}
	if err := compressor.Close(); err != nil {
		return nil, err
	}
	if paddingLength == nil {
		paddingLength = func() (int, error) {
			length, err := rand.Int(rand.Reader, big.NewInt(100))
			if err != nil {
				return 0, err
			}
			return int(length.Int64()), nil
		}
	}
	length, err := paddingLength()
	if err != nil {
		return nil, err
	}
	if length < 0 || length >= 100 {
		return nil, fmt.Errorf("invalid gzip padding length %d", length)
	}
	data := compressed.Bytes()
	padded := make([]byte, 0, len(data)+length+1)
	padded = append(padded, data[:3]...)
	padded = append(padded, data[3]|0x08)
	padded = append(padded, data[4:10]...)
	padded = append(padded, bytes.Repeat([]byte{'a'}, length)...)
	padded = append(padded, 0)
	padded = append(padded, data[10:]...)
	return padded, nil
}
