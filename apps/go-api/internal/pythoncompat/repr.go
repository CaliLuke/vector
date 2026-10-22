package pythoncompat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"

	"golang.org/x/text/unicode/rangetable"
)

var pythonUnicode15Assigned = rangetable.Assigned("15.0.0")

// Truthy covers the JSON values accepted by Plane's source serializers.
func Truthy(value any) bool {
	switch value := value.(type) {
	case nil:
		return false
	case bool:
		return value
	case string:
		return value != ""
	case json.Number:
		number, _ := value.Float64()
		return number != 0
	case float64:
		return value != 0
	case int:
		return value != 0
	case int64:
		return value != 0
	case []any:
		return len(value) != 0
	case map[string]any:
		return len(value) != 0
	}
	return true
}

// String matches Python str for JSON values. For composite values, original
// must be the JSON source that produced value. It supplies source string code
// units and object ordering, which Go's decoded maps and strings cannot retain
// for lone surrogates or colliding keys.
func String(value any, original []byte) (string, error) {
	if text, ok := value.(string); ok {
		return text, nil
	}
	if len(original) > 0 && json.Valid(original) {
		switch value.(type) {
		case []any:
			if firstJSONToken(original) == '[' {
				return pythonReprJSON(original)
			}
		case map[string]any:
			if firstJSONToken(original) == '{' {
				return pythonReprJSON(original)
			}
		}
	}
	return repr(value, original)
}

func repr(value any, original []byte) (string, error) {
	switch value := value.(type) {
	case nil:
		return "None", nil
	case bool:
		if value {
			return "True", nil
		}
		return "False", nil
	case string:
		return reprString([]rune(value)), nil
	case json.Number:
		if strings.ContainsAny(string(value), ".eE") {
			number, err := value.Float64()
			if err != nil && !errors.Is(err, strconv.ErrRange) {
				return "", err
			}
			return floatRepr(number), nil
		}
		if value == "-0" {
			return "0", nil
		}
		return string(value), nil
	case float64:
		return floatRepr(value), nil
	case int:
		return strconv.Itoa(value), nil
	case int64:
		return strconv.FormatInt(value, 10), nil
	case []any:
		var originals []json.RawMessage
		_ = json.Unmarshal(original, &originals)
		parts := make([]string, len(value))
		for i, item := range value {
			var source []byte
			if i < len(originals) {
				source = originals[i]
			}
			text, err := repr(item, source)
			if err != nil {
				return "", err
			}
			parts[i] = text
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	case map[string]any:
		keys, originals := objectOrder(original)
		seen := map[string]bool{}
		for _, key := range keys {
			seen[key] = true
		}
		extra := []string{}
		for key := range value {
			if !seen[key] {
				extra = append(extra, key)
			}
		}
		slices.Sort(extra)
		keys = append(keys, extra...)
		parts := []string{}
		for _, key := range keys {
			item, present := value[key]
			if !present {
				continue
			}
			label, _ := repr(key, nil)
			text, err := repr(item, originals[key])
			if err != nil {
				return "", err
			}
			parts = append(parts, label+": "+text)
		}
		return "{" + strings.Join(parts, ", ") + "}", nil
	default:
		return "", fmt.Errorf("unsupported Python string value %T", value)
	}
}

func pythonPrintable(character rune) bool {
	return pythonUnicode15Assigned != nil && unicode.Is(pythonUnicode15Assigned, character) && unicode.IsPrint(character) && (!unicode.IsSpace(character) || character == ' ')
}

func firstJSONToken(raw []byte) byte {
	return bytes.TrimSpace(raw)[0]
}

func reprString(value []rune) string {
	quote := '\''
	if slices.Contains(value, '\'') && !slices.Contains(value, '"') {
		quote = '"'
	}
	var out strings.Builder
	out.WriteRune(quote)
	for _, character := range value {
		switch character {
		case '\\':
			out.WriteString(`\\`)
		case '\n':
			out.WriteString(`\n`)
		case '\r':
			out.WriteString(`\r`)
		case '\t':
			out.WriteString(`\t`)
		default:
			if character == quote {
				out.WriteByte('\\')
				out.WriteRune(character)
			} else if pythonPrintable(character) {
				out.WriteRune(character)
			} else if character <= 0xff {
				fmt.Fprintf(&out, `\x%02x`, character)
			} else if character <= 0xffff {
				fmt.Fprintf(&out, `\u%04x`, character)
			} else {
				fmt.Fprintf(&out, `\U%08x`, character)
			}
		}
	}
	out.WriteRune(quote)
	return out.String()
}

type pythonJSONEntry struct {
	key   []rune
	value json.RawMessage
}

func pythonReprJSON(raw []byte) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return "", err
	}
	switch token := token.(type) {
	case json.Delim:
		if token == '[' {
			parts := []string{}
			for decoder.More() {
				var child json.RawMessage
				if err := decoder.Decode(&child); err != nil {
					return "", err
				}
				text, err := pythonReprJSON(child)
				if err != nil {
					return "", err
				}
				parts = append(parts, text)
			}
			if _, err := decoder.Token(); err != nil {
				return "", err
			}
			return "[" + strings.Join(parts, ", ") + "]", nil
		}
		if token == '{' {
			entries, err := pythonJSONObjectEntries(decoder, raw)
			if err != nil {
				return "", err
			}
			if _, err := decoder.Token(); err != nil {
				return "", err
			}
			parts := make([]string, 0, len(entries))
			for _, entry := range entries {
				key := reprString(pythonCodePoints(entry.key))
				value, err := pythonReprJSON(entry.value)
				if err != nil {
					return "", err
				}
				parts = append(parts, key+": "+value)
			}
			return "{" + strings.Join(parts, ", ") + "}", nil
		}
		return "", fmt.Errorf("unsupported JSON delimiter %q", token)
	case string:
		lexeme, err := jsonStringLexeme(raw)
		if err != nil {
			return "", err
		}
		value, _, err := DecodeJSONStringCodeUnits([]rune(string(lexeme)), 0)
		if err != nil {
			return "", err
		}
		return reprString(pythonCodePoints(value)), nil
	default:
		return repr(token, nil)
	}
}

func pythonJSONObjectEntries(decoder *json.Decoder, raw []byte) ([]pythonJSONEntry, error) {
	entries := []pythonJSONEntry{}
	positions := map[string]int{}
	for decoder.More() {
		start := decoder.InputOffset()
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		end := decoder.InputOffset()
		keyToken, err := jsonStringLexeme(raw[start:end])
		if err != nil {
			return nil, err
		}
		key, _, err := DecodeJSONStringCodeUnits([]rune(string(keyToken)), 0)
		if err != nil {
			return nil, err
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		identity := pythonKeyIdentity(pythonCodePoints(key))
		if position, ok := positions[identity]; ok {
			entries[position].value = value
		} else {
			positions[identity] = len(entries)
			entries = append(entries, pythonJSONEntry{key: key, value: value})
		}
	}
	return entries, nil
}

func jsonStringLexeme(raw []byte) ([]byte, error) {
	start := 0
	for start < len(raw) && (raw[start] == ',' || raw[start] == ' ' || raw[start] == '\t' || raw[start] == '\r' || raw[start] == '\n') {
		start++
	}
	if start >= len(raw) || raw[start] != '"' {
		return nil, fmt.Errorf("JSON string token missing from source")
	}
	_, end, err := DecodeJSONStringCodeUnits([]rune(string(raw)), startRuneIndex(raw, start))
	if err != nil {
		return nil, err
	}
	text := []rune(string(raw))
	return []byte(string(text[startRuneIndex(raw, start):end])), nil
}

func startRuneIndex(raw []byte, byteOffset int) int {
	return len([]rune(string(raw[:byteOffset])))
}

func pythonCodePoints(value []rune) []rune {
	result := make([]rune, 0, len(value))
	for index := 0; index < len(value); index++ {
		character := value[index]
		if character >= 0xd800 && character <= 0xdbff && index+1 < len(value) && value[index+1] >= 0xdc00 && value[index+1] <= 0xdfff {
			result = append(result, utf16.DecodeRune(character, value[index+1]))
			index++
			continue
		}
		result = append(result, character)
	}
	return result
}

func pythonKeyIdentity(value []rune) string {
	var identity strings.Builder
	for _, character := range value {
		fmt.Fprintf(&identity, "%08x", character)
	}
	return identity.String()
}

func floatRepr(value float64) string {
	switch {
	case math.IsNaN(value):
		return "nan"
	case math.IsInf(value, 1):
		return "inf"
	case math.IsInf(value, -1):
		return "-inf"
	default:
		return FloatString(value)
	}
}
