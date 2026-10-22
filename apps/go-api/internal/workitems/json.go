package workitems

import (
	"bytes"
	"encoding/json"
)

func decodeObject(raw []byte) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value map[string]any
	err := decoder.Decode(&value)
	return value, err
}
