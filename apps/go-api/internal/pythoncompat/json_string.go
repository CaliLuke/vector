package pythoncompat

import "fmt"

// DecodeJSONStringCodeUnits decodes one quoted JSON string from input at start.
// It returns the decoded code units and the index after the closing quote. It
// retains lone UTF-16 surrogates so callers can reproduce Python's behavior.
// Input may be Unicode runes or byte-valued runes, as used by Django's
// Latin-1-compatible session decoder.
func DecodeJSONStringCodeUnits(input []rune, start int) ([]rune, int, error) {
	if start < 0 || start >= len(input) || input[start] != '"' {
		return nil, start, fmt.Errorf("invalid JSON string")
	}
	value := make([]rune, 0)
	for index := start + 1; index < len(input); index++ {
		character := input[index]
		switch character {
		case '"':
			return value, index + 1, nil
		case '\\':
			index++
			if index >= len(input) {
				return nil, start, fmt.Errorf("invalid JSON string")
			}
			switch input[index] {
			case '"', '\\', '/':
				value = append(value, input[index])
			case 'b':
				value = append(value, '\b')
			case 'f':
				value = append(value, '\f')
			case 'n':
				value = append(value, '\n')
			case 'r':
				value = append(value, '\r')
			case 't':
				value = append(value, '\t')
			case 'u':
				code, next, ok := decodeHex4(input, index+1)
				if !ok {
					return nil, start, fmt.Errorf("invalid JSON string")
				}
				value = append(value, rune(code))
				index = next - 1
			default:
				return nil, start, fmt.Errorf("invalid JSON string")
			}
		default:
			if character < 0x20 {
				return nil, start, fmt.Errorf("invalid JSON string")
			}
			value = append(value, character)
		}
	}
	return nil, start, fmt.Errorf("invalid JSON string")
}

func decodeHex4(input []rune, start int) (uint16, int, bool) {
	if start < 0 || start+4 > len(input) {
		return 0, start, false
	}
	var value uint16
	for index := start; index < start+4; index++ {
		value <<= 4
		switch digit := input[index]; {
		case digit >= '0' && digit <= '9':
			value += uint16(digit - '0')
		case digit >= 'a' && digit <= 'f':
			value += uint16(digit-'a') + 10
		case digit >= 'A' && digit <= 'F':
			value += uint16(digit-'A') + 10
		default:
			return 0, start, false
		}
	}
	return value, start + 4, true
}
