package main

import (
	"bytes"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"
)

func djangoRequestHeaders(request *http.Request) string {
	values := make(map[string]string, len(request.Header)+4)
	for name, headerValues := range request.Header {
		if strings.ContainsRune(name, '_') {
			continue
		}
		value := djangoLatin1String([]byte(strings.Join(headerValues, ",")))
		if strings.EqualFold(name, "X-Api-Key") || strings.EqualFold(name, "Authorization") || strings.EqualFold(name, "Cookie") {
			value = "[REDACTED]"
		}
		values[http.CanonicalHeaderKey(name)] = value
	}
	if request.Host != "" {
		values["Host"] = djangoLatin1String([]byte(request.Host))
	}
	if request.ContentLength > 0 {
		values["Content-Length"] = strconv.FormatInt(request.ContentLength, 10)
	}
	if len(request.TransferEncoding) > 0 {
		values["Transfer-Encoding"] = strings.Join(request.TransferEncoding, ",")
	}
	if request.Close {
		values["Connection"] = "close"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, pythonStringRepr(key)+": "+pythonStringRepr(values[key]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func pythonStringRepr(value string) string {
	quote := rune('\'')
	if strings.ContainsRune(value, '\'') && !strings.ContainsRune(value, '"') {
		quote = '"'
	}
	var result strings.Builder
	result.WriteRune(quote)
	for _, character := range value {
		switch character {
		case '\\':
			result.WriteString(`\\`)
		case '\t':
			result.WriteString(`\t`)
		case '\n':
			result.WriteString(`\n`)
		case '\r':
			result.WriteString(`\r`)
		default:
			switch {
			case character == quote:
				result.WriteRune('\\')
				result.WriteRune(character)
			case unicode.IsPrint(character):
				result.WriteRune(character)
			case character <= 0xff:
				_, _ = fmt.Fprintf(&result, `\x%02x`, character)
			case character <= 0xffff:
				_, _ = fmt.Fprintf(&result, `\u%04x`, character)
			default:
				_, _ = fmt.Fprintf(&result, `\U%08x`, character)
			}
		}
	}
	result.WriteRune(quote)
	return result.String()
}

func djangoRequestAcceptHeader(request *http.Request) (string, bool) {
	values, present := request.Header[http.CanonicalHeaderKey("Accept")]
	return strings.Join(values, ","), present
}

func projectMembershipReadsJSONNegotiation(accept string, present bool) (bool, int, error) {
	if !present {
		accept = "*/*"
	}
	bestPrecedence := -1
	bestIndent := 0
	for _, value := range strings.Split(accept, ",") {
		mediaType, parameters, err := djangoParseHeaderParameters(strings.TrimSpace(value))
		if err != nil {
			return false, 0, err
		}
		precedence := -1
		switch mediaType {
		case "*/*":
			precedence = 0
		case "application/*":
			precedence = 1
		case "application/json":
			precedence = 2
			for name := range parameters {
				if name != "q" {
					precedence = 3
					break
				}
			}
		}
		if precedence <= bestPrecedence {
			continue
		}
		bestPrecedence = precedence
		bestIndent = djangoJSONIndent(parameters["indent"])
	}
	return bestPrecedence >= 0, bestIndent, nil
}

func djangoParseHeaderParameters(value string) (string, map[string]string, error) {
	parts := make([]string, 0, 2)
	start := 0
	inQuotes := false
	escaped := false
	for index := 0; index < len(value); index++ {
		switch {
		case escaped:
			escaped = false
		case inQuotes && value[index] == '\\':
			escaped = true
		case value[index] == '"':
			inQuotes = !inQuotes
		case value[index] == ';' && !inQuotes:
			parts = append(parts, strings.TrimSpace(value[start:index]))
			start = index + 1
		}
	}
	parts = append(parts, strings.TrimSpace(value[start:]))
	mediaType := strings.ToLower(parts[0])
	parameters := make(map[string]string, len(parts)-1)
	for _, parameter := range parts[1:] {
		name, parameterValue, found := strings.Cut(parameter, "=")
		if !found {
			continue
		}
		name = strings.ToLower(strings.TrimSpace(name))
		parameterValue = strings.TrimSpace(parameterValue)
		if len(parameterValue) >= 2 && parameterValue[0] == '"' && parameterValue[len(parameterValue)-1] == '"' {
			parameterValue = parameterValue[1 : len(parameterValue)-1]
			parameterValue = strings.ReplaceAll(parameterValue, `\\`, `\`)
			parameterValue = strings.ReplaceAll(parameterValue, `\"`, `"`)
		}
		if strings.HasSuffix(name, "*") {
			name = strings.TrimSuffix(name, "*")
			if strings.Count(parameterValue, "'") == 2 {
				extended := strings.Split(parameterValue, "'")
				decodedValue, err := djangoUnquoteHeaderParameter(extended[2], extended[0])
				if err != nil {
					return "", nil, err
				}
				parameterValue = decodedValue
			}
		}
		parameters[name] = parameterValue
	}
	return mediaType, parameters, nil
}

func djangoUnquoteHeaderParameter(value string, encodingName string) (string, error) {
	if !strings.Contains(value, "%") {
		return value, nil
	}
	decoded := make([]byte, 0, len(value))
	for index := 0; index < len(value); index++ {
		if value[index] == '%' && index+2 < len(value) {
			high, highOK := queryHexDigit(value[index+1])
			low, lowOK := queryHexDigit(value[index+2])
			if highOK && lowOK {
				decoded = append(decoded, high<<4|low)
				index += 2
				continue
			}
		}
		decoded = append(decoded, value[index])
	}
	normalizedEncoding := strings.ToLower(strings.ReplaceAll(encodingName, "_", "-"))
	stripUTF8BOM := false
	switch normalizedEncoding {
	case "u8", "utf8", "cp65001":
		normalizedEncoding = "utf-8"
	case "utf-8-sig":
		normalizedEncoding = "utf-8"
		stripUTF8BOM = true
	}
	if stripUTF8BOM && bytes.HasPrefix(decoded, []byte{0xEF, 0xBB, 0xBF}) {
		decoded = decoded[3:]
	}
	if normalizedEncoding == "unicode-escape" || normalizedEncoding == "raw-unicode-escape" {
		return decodePythonUnicodeEscape(decoded, normalizedEncoding == "raw-unicode-escape"), nil
	}
	decoder, err := ianaindex.IANA.Encoding(normalizedEncoding)
	if err != nil {
		decoder, err = htmlindex.Get(normalizedEncoding)
		if err != nil {
			return "", fmt.Errorf("decode Django header parameter with encoding %q: %w", encodingName, err)
		}
	}
	if decoder == nil {
		return "", fmt.Errorf("decode Django header parameter with unsupported encoding %q", encodingName)
	}
	decodedValue, _, err := transform.String(decoder.NewDecoder(), string(decoded))
	if err != nil {
		return "", fmt.Errorf("decode Django header parameter with encoding %q: %w", encodingName, err)
	}
	return decodedValue, nil
}

func decodePythonUnicodeEscape(value []byte, raw bool) string {
	var decoded strings.Builder
	for index := 0; index < len(value); {
		if value[index] != '\\' {
			decoded.WriteRune(rune(value[index]))
			index++
			continue
		}
		if raw {
			runEnd := index
			for runEnd < len(value) && value[runEnd] == '\\' {
				runEnd++
			}
			count := runEnd - index
			if runEnd >= len(value) || (value[runEnd] != 'u' && value[runEnd] != 'U') || count%2 == 0 {
				decoded.WriteString(strings.Repeat("\\", count))
				index = runEnd
				continue
			}
			decoded.WriteString(strings.Repeat("\\", count-1))
			index = runEnd - 1
		}
		if index+1 >= len(value) {
			decoded.WriteRune(utf8.RuneError)
			break
		}
		escape := value[index+1]
		if escape == 'u' || escape == 'U' {
			digits := 4
			if escape == 'U' {
				digits = 8
			}
			if character, ok := pythonHexRune(value, index+2, digits); ok {
				decoded.WriteRune(character)
				index += 2 + digits
				continue
			}
			decoded.WriteRune(utf8.RuneError)
			index += min(2+digits, len(value)-index)
			continue
		}
		if escape == 'x' {
			if character, ok := pythonHexRune(value, index+2, 2); ok {
				decoded.WriteRune(character)
				index += 4
				continue
			}
			decoded.WriteRune(utf8.RuneError)
			index += min(4, len(value)-index)
			continue
		}
		if escape == 'N' {
			if character, consumed, ok := pythonNamedDigit(value[index+2:]); ok {
				decoded.WriteRune(character)
				index += 2 + consumed
				continue
			}
		}
		if escape >= '0' && escape <= '7' {
			end := index + 2
			for end < len(value) && end < index+4 && value[end] >= '0' && value[end] <= '7' {
				end++
			}
			character, _ := strconv.ParseUint(string(value[index+1:end]), 8, 8)
			decoded.WriteRune(rune(character))
			index = end
			continue
		}
		if character, ok := map[byte]byte{
			'\\': '\\', '\'': '\'', '"': '"', 'a': '\a', 'b': '\b', 'f': '\f',
			'n': '\n', 'r': '\r', 't': '\t', 'v': '\v',
		}[escape]; ok {
			decoded.WriteByte(character)
			index += 2
			continue
		}
		decoded.WriteByte('\\')
		decoded.WriteByte(escape)
		index += 2
	}
	return decoded.String()
}

func pythonHexRune(value []byte, start int, digits int) (rune, bool) {
	if start+digits > len(value) {
		return 0, false
	}
	parsed, err := strconv.ParseUint(string(value[start:start+digits]), 16, 32)
	if err != nil || parsed > utf8.MaxRune || parsed >= 0xD800 && parsed <= 0xDFFF {
		return 0, false
	}
	return rune(parsed), true
}

func pythonNamedDigit(value []byte) (rune, int, bool) {
	if len(value) < 3 || value[0] != '{' {
		return 0, 0, false
	}
	end := bytes.IndexByte(value, '}')
	if end < 0 {
		return 0, 0, false
	}
	names := [...]string{"ZERO", "ONE", "TWO", "THREE", "FOUR", "FIVE", "SIX", "SEVEN", "EIGHT", "NINE"}
	name := string(value[1:end])
	for digit, word := range names {
		if name == "DIGIT "+word {
			return rune('0' + digit), end + 1, true
		}
	}
	return 0, 0, false
}

func djangoJSONIndent(value string) int {
	integer, err := pythoncompat.DecimalInteger(value)
	if err != nil || integer.Sign() <= 0 {
		return 0
	}
	maximum := big.NewInt(8)
	if integer.Cmp(maximum) >= 0 {
		return 8
	}
	return int(integer.Int64())
}

func djangoJoinedHeader(request *http.Request, name string) string {
	return djangoLatin1String([]byte(strings.Join(request.Header.Values(name), ",")))
}

func mergeVary(headers http.Header, required ...string) {
	seen := make(map[string]bool)
	values := make([]string, 0, len(required)+len(headers.Values("Vary")))
	for _, line := range append(headers.Values("Vary"), strings.Join(required, ", ")) {
		for _, raw := range strings.Split(line, ",") {
			value := strings.TrimSpace(raw)
			key := strings.ToLower(value)
			if value == "" || seen[key] {
				continue
			}
			seen[key] = true
			values = append(values, value)
		}
	}
	headers.Set("Vary", strings.Join(values, ", "))
}

func removeVary(headers http.Header, removed ...string) {
	blocked := make(map[string]bool, len(removed))
	for _, value := range removed {
		blocked[strings.ToLower(value)] = true
	}
	values := make([]string, 0, len(headers.Values("Vary")))
	seen := make(map[string]bool)
	for _, line := range headers.Values("Vary") {
		for _, raw := range strings.Split(line, ",") {
			value := strings.TrimSpace(raw)
			key := strings.ToLower(value)
			if value == "" || blocked[key] || seen[key] {
				continue
			}
			seen[key] = true
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		headers.Del("Vary")
		return
	}
	headers.Set("Vary", strings.Join(values, ", "))
}
