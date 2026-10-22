package pythoncompat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// NormalizeJSONNumbers preserves integer precision and converts decimal and
// exponent tokens through Python's binary64 representation before persistence.
// It leaves strings, object order, whitespace, and null/empty values unchanged.
func NormalizeJSONNumbers(data []byte) ([]byte, error) {
	return normalizeJSONNumbers(data, false)
}

// NormalizeDecodedJSONNumbers applies Python's JSON float decoding to an input
// object while retaining syntactically valid exponents that overflow binary64.
// Python decodes those tokens as infinity, and a downstream Python JSON worker
// can decode the retained token the same way. Persisted JSONFields should use
// NormalizeJSONNumbers, which rejects non-finite values.
func NormalizeDecodedJSONNumbers(data []byte) ([]byte, error) {
	return normalizeJSONNumbers(data, true)
}

func normalizeJSONNumbers(data []byte, preserveOverflow bool) ([]byte, error) {
	if !json.Valid(data) {
		return nil, fmt.Errorf("invalid JSON")
	}
	var out bytes.Buffer
	for i := 0; i < len(data); {
		start := i
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
			i++
			for i < len(data) && strings.ContainsRune("0123456789.eE+-", rune(data[i])) {
				i++
			}
			token := string(data[start:i])
			if strings.ContainsAny(token, ".eE") {
				value, err := strconv.ParseFloat(token, 64)
				if err != nil || math.IsInf(value, 0) {
					if preserveOverflow && (math.IsInf(value, 0) || errors.Is(err, strconv.ErrRange)) {
						out.WriteString(token)
						continue
					}
					return nil, fmt.Errorf("JSON number exceeds finite float range: %s", token)
				}
				out.WriteString(FloatString(value))
			} else if token == "-0" {
				out.WriteByte('0')
			} else {
				out.WriteString(token)
			}
			continue
		} else {
			i++
		}
		out.Write(data[start:i])
	}
	return out.Bytes(), nil
}

// FloatString formats a finite float as Python 3.12 json.dumps does.
func FloatString(value float64) string {
	scientific := strconv.FormatFloat(value, 'e', -1, 64)
	coefficient, exponentText, _ := strings.Cut(scientific, "e")
	exponent, _ := strconv.Atoi(exponentText)
	if exponent < -4 || exponent >= 16 {
		return scientific
	}
	sign := ""
	if strings.HasPrefix(coefficient, "-") {
		sign = "-"
		coefficient = coefficient[1:]
	}
	digits := strings.ReplaceAll(coefficient, ".", "")
	decimal := exponent + 1
	switch {
	case decimal <= 0:
		return sign + "0." + strings.Repeat("0", -decimal) + digits
	case decimal >= len(digits):
		return sign + digits + strings.Repeat("0", decimal-len(digits)) + ".0"
	default:
		return sign + digits[:decimal] + "." + digits[decimal:]
	}
}
