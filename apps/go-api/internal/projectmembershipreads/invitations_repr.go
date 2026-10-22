package projectmembershipreads

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/CaliLuke/go-plane/internal/pythoncompat"
)

func invitationRawEntry(raw []byte, index int) []byte {
	var fields map[string]json.RawMessage
	var entries []json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || json.Unmarshal(fields["emails"], &entries) != nil || index >= len(entries) {
		return nil
	}
	return entries[index]
}

func invitationPythonEntry(item map[string]any, original []byte) string {
	// Dumps retains input dictionary order and Python numeric spelling. Read its
	// tokens so punctuation and quotes inside string values remain untouched.
	encoded, err := pythoncompat.Dumps(item, original)
	if err != nil {
		return fmt.Sprint(item)
	}
	decoder := json.NewDecoder(strings.NewReader(encoded))
	decoder.UseNumber()
	return invitationPythonToken(decoder)
}

func invitationPythonToken(decoder *json.Decoder) string {
	value, err := decoder.Token()
	if err != nil {
		return ""
	}
	switch value := value.(type) {
	case nil:
		return "None"
	case bool:
		if value {
			return "True"
		}
		return "False"
	case string:
		return invitationPythonString(value)
	case json.Number:
		return string(value)
	case json.Delim:
		var parts []string
		for decoder.More() {
			part := invitationPythonToken(decoder)
			if value == '{' {
				part += ": " + invitationPythonToken(decoder)
			}
			parts = append(parts, part)
		}
		_, _ = decoder.Token()
		if value == '{' {
			return "{" + strings.Join(parts, ", ") + "}"
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return fmt.Sprint(value)
	}
}

func invitationPythonString(value string) string {
	quote := '\''
	if strings.ContainsRune(value, '\'') && !strings.ContainsRune(value, '"') {
		quote = '"'
	}
	var result strings.Builder
	result.WriteRune(quote)
	for _, c := range value {
		switch {
		case c == '\\' || c == quote:
			result.WriteRune('\\')
			result.WriteRune(c)
		case c == '\t':
			result.WriteString(`\t`)
		case c == '\n':
			result.WriteString(`\n`)
		case c == '\r':
			result.WriteString(`\r`)
		case unicode.IsPrint(c):
			result.WriteRune(c)
		case c <= 0xff:
			fmt.Fprintf(&result, `\x%02x`, c)
		case c <= 0xffff:
			fmt.Fprintf(&result, `\u%04x`, c)
		default:
			fmt.Fprintf(&result, `\U%08x`, c)
		}
	}
	result.WriteRune(quote)
	return result.String()
}
