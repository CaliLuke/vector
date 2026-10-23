package djangosession

import (
	"net/http"
	"strings"
)

// CookieFromHeaderValues parses a Django-compatible cookie from the request's
// Cookie header fields. It preserves WSGI header joining, Latin-1 decoding,
// quoted octal escapes, and last-duplicate-wins behavior.
func CookieFromHeaderValues(headerValues []string, name string) *http.Cookie {
	values := make(map[string]string)
	header := latin1String([]byte(strings.Join(headerValues, ",")))
	for _, chunk := range strings.Split(header, ";") {
		key, value, found := strings.Cut(chunk, "=")
		if !found {
			key, value = "", key
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" && value == "" {
			continue
		}
		values[key] = cookieUnquote(value)
	}
	value, found := values[name]
	if !found {
		return nil
	}
	return &http.Cookie{Name: name, Value: value}
}

func cookieUnquote(value string) string {
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return value
	}
	characters := []rune(value[1 : len(value)-1])
	var decoded strings.Builder
	decoded.Grow(len(value))
	for index := 0; index < len(characters); index++ {
		if characters[index] != '\\' || index+1 >= len(characters) {
			decoded.WriteRune(characters[index])
			continue
		}
		if index+3 < len(characters) && characters[index+1] >= '0' && characters[index+1] <= '3' &&
			characters[index+2] >= '0' && characters[index+2] <= '7' && characters[index+3] >= '0' && characters[index+3] <= '7' {
			decoded.WriteRune((characters[index+1]-'0')*64 + (characters[index+2]-'0')*8 + characters[index+3] - '0')
			index += 3
			continue
		}
		decoded.WriteRune(characters[index+1])
		index++
	}
	return decoded.String()
}

func latin1String(value []byte) string {
	characters := make([]rune, len(value))
	for index, octet := range value {
		characters[index] = rune(octet)
	}
	return string(characters)
}
