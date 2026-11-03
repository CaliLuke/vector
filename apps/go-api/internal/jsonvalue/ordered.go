package jsonvalue

import (
	"bytes"
	"encoding/json"
	"io"
	"slices"
	"strconv"
	"strings"
)

// Ordered serializes JSON while retaining input field order for activity workers.
// New fields are sorted. Already persisted worker payloads and receipt bodies
// are not re-encoded, so their accepted identities remain unchanged.
func Ordered(value any, original []byte) (string, error) {
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
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		output.Write(encoded)
	case float64:
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		output.Write(encoded)
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

func dumpString(output *strings.Builder, value string) {
	encoded, _ := json.Marshal(value)
	output.Write(encoded)
}

// DecodeObject preserves JSON integer precision at application-owned body
// boundaries. Stream framing and parser errors are handled by requestinput.
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
