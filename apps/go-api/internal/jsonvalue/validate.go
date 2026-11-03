package jsonvalue

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Validate checks JSON and the finite decimal range supported by browser
// consumers. It preserves exact numeric text rather than rounding on write.
func Validate(data []byte) ([]byte, error) {
	if !json.Valid(data) {
		return nil, fmt.Errorf("invalid JSON")
	}
	for i := 0; i < len(data); {
		if data[i] == '"' {
			i++
			for i < len(data) {
				if data[i] == '\\' {
					i += 2
					continue
				}
				if data[i] == '"' {
					i++
					break
				}
				i++
			}
		} else if data[i] == '-' || (data[i] >= '0' && data[i] <= '9') {
			start := i
			i++
			for i < len(data) && strings.ContainsRune("0123456789.eE+-", rune(data[i])) {
				i++
			}
			token := string(data[start:i])
			if strings.ContainsAny(token, ".eE") {
				value, err := strconv.ParseFloat(token, 64)
				if err != nil || math.IsInf(value, 0) {
					return nil, fmt.Errorf("JSON number exceeds finite float range: %s", token)
				}
			}
			continue
		} else {
			i++
		}
	}
	return data, nil
}
