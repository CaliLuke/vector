package pythoncompat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Dumps uses Python json.dumps' default separators, string escaping and numeric
// spelling. The original JSON supplies dictionary insertion order. Duplicate
// keys retain their first position and their last value, matching json.loads.
// This matters for workers that process requested_data in dictionary order.
func Dumps(value any, original []byte) (string, error) {
	var output strings.Builder
	if err := dumpValue(&output, value, original); err != nil {
		return "", err
	}
	return output.String(), nil
}

func dumpValue(output *strings.Builder, value any, original []byte) error {
	switch value := value.(type) {
	case nil:
		output.WriteString("null")
	case bool:
		output.WriteString(strconv.FormatBool(value))
	case string:
		dumpString(output, value)
	case json.Number:
		if strings.ContainsAny(string(value), ".eE") {
			f, _ := strconv.ParseFloat(string(value), 64)
			dumpFloat(output, f)
		} else if value == "-0" {
			output.WriteByte('0')
		} else {
			output.WriteString(string(value))
		}
	case float64:
		dumpFloat(output, value)
	case int:
		output.WriteString(strconv.Itoa(value))
	case int64:
		output.WriteString(strconv.FormatInt(value, 10))
	case map[string]any:
		keys, children := objectOrder(original)
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
		output.WriteByte('{')
		first := true
		for _, key := range keys {
			item, present := value[key]
			if !present {
				continue
			}
			if !first {
				output.WriteString(", ")
			}
			first = false
			dumpString(output, key)
			output.WriteString(": ")
			if err := dumpValue(output, item, children[key]); err != nil {
				return err
			}
		}
		output.WriteByte('}')
	case []any:
		var children []json.RawMessage
		_ = json.Unmarshal(original, &children)
		output.WriteByte('[')
		for index, item := range value {
			if index > 0 {
				output.WriteString(", ")
			}
			var source []byte
			if index < len(children) {
				source = children[index]
			}
			if err := dumpValue(output, item, source); err != nil {
				return err
			}
		}
		output.WriteByte(']')
	default:
		// Named scalar/collection values follow their declared JSON representation;
		// decode with UseNumber so large integer task arguments remain exact.
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		var plain any
		decoder := json.NewDecoder(bytes.NewReader(encoded))
		decoder.UseNumber()
		if err = decoder.Decode(&plain); err != nil {
			return err
		}
		return dumpValue(output, plain, original)
	}
	return nil
}

func objectOrder(raw []byte) ([]string, map[string]json.RawMessage) {
	keys := []string{}
	children := map[string]json.RawMessage{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return keys, children
	}
	for decoder.More() {
		token, err = decoder.Token()
		if err != nil {
			break
		}
		key, ok := token.(string)
		if !ok {
			break
		}
		var child json.RawMessage
		if err = decoder.Decode(&child); err != nil {
			break
		}
		if _, present := children[key]; !present {
			keys = append(keys, key)
		}
		children[key] = child
	}
	return keys, children
}

func dumpFloat(output *strings.Builder, value float64) {
	switch {
	case math.IsNaN(value):
		output.WriteString("NaN")
	case math.IsInf(value, 1):
		output.WriteString("Infinity")
	case math.IsInf(value, -1):
		output.WriteString("-Infinity")
	default:
		output.WriteString(FloatString(value))
	}
}

func dumpString(output *strings.Builder, value string) {
	output.WriteByte('"')
	for _, character := range value {
		switch character {
		case '"', '\\':
			output.WriteByte('\\')
			output.WriteRune(character)
		case '\b':
			output.WriteString(`\b`)
		case '\f':
			output.WriteString(`\f`)
		case '\n':
			output.WriteString(`\n`)
		case '\r':
			output.WriteString(`\r`)
		case '\t':
			output.WriteString(`\t`)
		default:
			if character >= 32 && character < 127 {
				output.WriteRune(character)
			} else if character <= 0xffff {
				fmt.Fprintf(output, `\u%04x`, character)
			} else {
				high, low := utf16.EncodeRune(character)
				fmt.Fprintf(output, `\u%04x\u%04x`, high, low)
			}
		}
	}
	output.WriteByte('"')
}

// DecodeObject preserves JSON integer precision at application-owned body
// boundaries. Stream framing and parser errors are handled by djangoinput.
func DecodeObject(data []byte) (map[string]any, error) {
	var value map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	err := decoder.Decode(&value)
	if err == io.EOF {
		return map[string]any{}, nil
	}
	return value, err
}
