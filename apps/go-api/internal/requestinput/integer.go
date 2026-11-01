package requestinput

import (
	"encoding/json"
	"errors"
	"math/big"
	"strconv"
	"strings"
)

// IntegerField accepts decimal form integers and mathematically integral JSON
// numbers without rounding through float64. Callers enforce destination ranges.
func IntegerField(value any) (*big.Int, error) {
	var text string
	numericJSON := false
	switch value := value.(type) {
	case string:
		text = strings.TrimSpace(value)
	case json.Number:
		text = string(value)
		numericJSON = true
	default:
		return nil, errors.New("integer input must be a decimal integer")
	}
	if len(text) > 1000 {
		return nil, errors.New("integer input is too long")
	}
	if !numericJSON {
		number, ok := new(big.Int).SetString(text, 10)
		if !ok {
			return nil, errors.New("integer input must be a decimal integer")
		}
		return number, nil
	}
	if !json.Valid([]byte(text)) {
		return nil, errors.New("invalid JSON number")
	}
	if index := strings.IndexAny(text, "eE"); index >= 0 {
		exponent, err := strconv.Atoi(text[index+1:])
		if err != nil || exponent > 1000 || exponent < -1000 {
			return nil, errors.New("integer exponent is outside supported range")
		}
	}
	number, ok := new(big.Rat).SetString(text)
	if !ok || !number.IsInt() {
		return nil, errors.New("integer input must be a decimal integer")
	}
	return new(big.Int).Set(number.Num()), nil
}
