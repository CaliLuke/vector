package pythoncompat

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// DecimalInteger matches Python 3.12 int parsing for decimal strings.
func DecimalInteger(value string) (*big.Int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("integer is empty")
	}
	characters := []rune(value)
	normalized := make([]byte, 0, len(value))
	index := 0
	if characters[0] == '+' || characters[0] == '-' {
		normalized = append(normalized, byte(characters[0]))
		index++
	}
	digits := 0
	previousDigit := false
	for ; index < len(characters); index++ {
		character := characters[index]
		if digit, ok := unicodeDecimalDigit(character); ok {
			if digits == 4300 {
				return nil, fmt.Errorf("decimal integer exceeds Python's 4300-digit limit")
			}
			normalized = append(normalized, '0'+digit)
			digits++
			previousDigit = true
			continue
		}
		nextIsDigit := false
		if index+1 < len(characters) {
			_, nextIsDigit = unicodeDecimalDigit(characters[index+1])
		}
		if character == '_' && previousDigit && nextIsDigit {
			previousDigit = false
			continue
		}
		return nil, fmt.Errorf("invalid decimal integer %q", value)
	}
	if digits == 0 {
		return nil, fmt.Errorf("invalid decimal integer %q", value)
	}
	integer, ok := new(big.Int).SetString(string(normalized), 10)
	if !ok {
		return nil, fmt.Errorf("invalid decimal integer %q", value)
	}
	return integer, nil
}

var pythonFloatPattern = regexp.MustCompile(`^[+-]?(?:(?:[0-9](?:_?[0-9])*)?(?:\.[0-9](?:_?[0-9])*)|[0-9](?:_?[0-9])*(?:\.)?)(?:[eE][+-]?[0-9](?:_?[0-9])*)?$`)

// DecimalFloat64 matches Python 3.12 float parsing for decimal strings.
func DecimalFloat64(value string) (float64, error) {
	characters := strings.TrimSpace(value)
	normalized := make([]byte, 0, len(value))
	for _, character := range characters {
		if digit, ok := unicodeDecimalDigit(character); ok {
			normalized = append(normalized, '0'+digit)
			continue
		}
		if character > 0x7f {
			return 0, fmt.Errorf("invalid decimal float %q", value)
		}
		normalized = append(normalized, byte(character))
	}
	if !pythonFloatPattern.Match(normalized) {
		return 0, fmt.Errorf("invalid decimal float %q", value)
	}
	parsed, err := strconv.ParseFloat(strings.ReplaceAll(string(normalized), "_", ""), 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0, fmt.Errorf("invalid decimal float %q", value)
	}
	return parsed, nil
}

func unicodeDecimalDigit(character rune) (byte, bool) {
	// Python 3.12 uses Unicode 15.0. Keep this table frozen to the runtime
	// that parses Plane's Django settings and renderer parameters.
	for _, digits := range python312DecimalDigitRanges {
		if character >= digits[0] && character <= digits[1] {
			return byte((character - digits[0]) % 10), true
		}
	}
	return 0, false
}

var python312DecimalDigitRanges = [...][2]rune{
	{0x30, 0x39}, {0x660, 0x669}, {0x6F0, 0x6F9}, {0x7C0, 0x7C9},
	{0x966, 0x96F}, {0x9E6, 0x9EF}, {0xA66, 0xA6F}, {0xAE6, 0xAEF},
	{0xB66, 0xB6F}, {0xBE6, 0xBEF}, {0xC66, 0xC6F}, {0xCE6, 0xCEF},
	{0xD66, 0xD6F}, {0xDE6, 0xDEF}, {0xE50, 0xE59}, {0xED0, 0xED9},
	{0xF20, 0xF29}, {0x1040, 0x1049}, {0x1090, 0x1099}, {0x17E0, 0x17E9},
	{0x1810, 0x1819}, {0x1946, 0x194F}, {0x19D0, 0x19D9}, {0x1A80, 0x1A89},
	{0x1A90, 0x1A99}, {0x1B50, 0x1B59}, {0x1BB0, 0x1BB9}, {0x1C40, 0x1C49},
	{0x1C50, 0x1C59}, {0xA620, 0xA629}, {0xA8D0, 0xA8D9}, {0xA900, 0xA909},
	{0xA9D0, 0xA9D9}, {0xA9F0, 0xA9F9}, {0xAA50, 0xAA59}, {0xABF0, 0xABF9},
	{0xFF10, 0xFF19}, {0x104A0, 0x104A9}, {0x10D30, 0x10D39}, {0x11066, 0x1106F},
	{0x110F0, 0x110F9}, {0x11136, 0x1113F}, {0x111D0, 0x111D9}, {0x112F0, 0x112F9},
	{0x11450, 0x11459}, {0x114D0, 0x114D9}, {0x11650, 0x11659}, {0x116C0, 0x116C9},
	{0x11730, 0x11739}, {0x118E0, 0x118E9}, {0x11950, 0x11959}, {0x11C50, 0x11C59},
	{0x11D50, 0x11D59}, {0x11DA0, 0x11DA9}, {0x11F50, 0x11F59}, {0x16A60, 0x16A69},
	{0x16AC0, 0x16AC9}, {0x16B50, 0x16B59}, {0x1D7CE, 0x1D7FF}, {0x1E140, 0x1E149},
	{0x1E2F0, 0x1E2F9}, {0x1E4F0, 0x1E4F9}, {0x1E950, 0x1E959}, {0x1FBF0, 0x1FBF9},
}
