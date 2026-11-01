package testsupport

import (
	"bytes"
	"encoding/json"
)

// JSONEqual compares valid JSON encodings. Failed encodings never compare equal.
func JSONEqual(actual, expected any) bool {
	left, err := json.Marshal(actual)
	if err != nil {
		return false
	}
	right, err := json.Marshal(expected)
	if err != nil {
		return false
	}
	return bytes.Equal(left, right)
}
