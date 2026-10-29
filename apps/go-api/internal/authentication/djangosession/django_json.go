package djangosession

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/CaliLuke/go-plane/internal/pythoncompat"
)

type djangoJSONKind uint8

const (
	djangoJSONNull djangoJSONKind = iota
	djangoJSONBool
	djangoJSONString
	djangoJSONInteger
	djangoJSONFloat
	djangoJSONArray
	djangoJSONObject
)

type djangoJSONValue struct {
	kind    djangoJSONKind
	boolean bool
	text    []rune
	number  string
	array   []djangoJSONValue
	object  *djangoJSONObjectValue
}

type djangoJSONObjectValue struct {
	entries []djangoJSONEntry
	index   map[string]int
}

type djangoJSONEntry struct {
	key   []rune
	value djangoJSONValue
}

type djangoOrderedObject struct {
	entries []djangoPlainEntry
}

// Go strings cannot represent lone UTF-16 surrogate code units. Keep such
// JSON strings in rune form so Django's canonical session bytes remain exact
// and JSONField persistence can fail as PostgreSQL does.
type djangoSurrogateString struct {
	text []rune
}

type djangoPlainEntry struct {
	key             string
	keyHasSurrogate bool
	value           any
}

type djangoJSONParser struct {
	input []rune
	index int
}

func decodeDjangoJSON(input []byte) (map[string]any, []byte, error) {
	characters := make([]rune, len(input))
	for index, octet := range input {
		characters[index] = rune(octet)
	}
	parser := djangoJSONParser{input: characters}
	value, err := parser.parseValue()
	if err != nil {
		return nil, nil, err
	}
	parser.skipWhitespace()
	if parser.index != len(parser.input) {
		return nil, nil, ErrInvalidData
	}
	if value.kind != djangoJSONObject {
		return nil, nil, ErrInvalidSessionShape
	}
	plain := make(map[string]any, len(value.object.entries))
	for _, entry := range value.object.entries {
		plain[djangoRunesToString(entry.key)] = djangoJSONPlainValue(entry.value)
	}
	var canonical bytes.Buffer
	writeDjangoJSON(&canonical, value)
	return plain, canonical.Bytes(), nil
}

func (parser *djangoJSONParser) parseValue() (djangoJSONValue, error) {
	parser.skipWhitespace()
	if parser.index >= len(parser.input) {
		return djangoJSONValue{}, ErrInvalidData
	}
	switch parser.input[parser.index] {
	case '{':
		return parser.parseObject()
	case '[':
		return parser.parseArray()
	case '"':
		value, err := parser.parseString()
		return djangoJSONValue{kind: djangoJSONString, text: value}, err
	case 't':
		if parser.consume("true") {
			return djangoJSONValue{kind: djangoJSONBool, boolean: true}, nil
		}
	case 'f':
		if parser.consume("false") {
			return djangoJSONValue{kind: djangoJSONBool}, nil
		}
	case 'n':
		if parser.consume("null") {
			return djangoJSONValue{kind: djangoJSONNull}, nil
		}
	case 'N':
		if parser.consume("NaN") {
			return djangoJSONValue{kind: djangoJSONFloat, number: "NaN"}, nil
		}
	case 'I':
		if parser.consume("Infinity") {
			return djangoJSONValue{kind: djangoJSONFloat, number: "Infinity"}, nil
		}
	case '-':
		if parser.consume("-Infinity") {
			return djangoJSONValue{kind: djangoJSONFloat, number: "-Infinity"}, nil
		}
		return parser.parseNumber()
	default:
		if parser.input[parser.index] >= '0' && parser.input[parser.index] <= '9' {
			return parser.parseNumber()
		}
	}
	return djangoJSONValue{}, ErrInvalidData
}

func (parser *djangoJSONParser) parseObject() (djangoJSONValue, error) {
	parser.index++
	object := &djangoJSONObjectValue{index: make(map[string]int)}
	parser.skipWhitespace()
	if parser.take('}') {
		return djangoJSONValue{kind: djangoJSONObject, object: object}, nil
	}
	for {
		parser.skipWhitespace()
		if parser.index >= len(parser.input) || parser.input[parser.index] != '"' {
			return djangoJSONValue{}, ErrInvalidData
		}
		key, err := parser.parseString()
		if err != nil {
			return djangoJSONValue{}, err
		}
		parser.skipWhitespace()
		if !parser.take(':') {
			return djangoJSONValue{}, ErrInvalidData
		}
		value, err := parser.parseValue()
		if err != nil {
			return djangoJSONValue{}, err
		}
		identity := djangoJSONKeyIdentity(key)
		if existing, found := object.index[identity]; found {
			object.entries[existing].value = value
		} else {
			object.index[identity] = len(object.entries)
			object.entries = append(object.entries, djangoJSONEntry{key: key, value: value})
		}
		parser.skipWhitespace()
		if parser.take('}') {
			return djangoJSONValue{kind: djangoJSONObject, object: object}, nil
		}
		if !parser.take(',') {
			return djangoJSONValue{}, ErrInvalidData
		}
	}
}

func (parser *djangoJSONParser) parseArray() (djangoJSONValue, error) {
	parser.index++
	result := djangoJSONValue{kind: djangoJSONArray}
	parser.skipWhitespace()
	if parser.take(']') {
		return result, nil
	}
	for {
		value, err := parser.parseValue()
		if err != nil {
			return djangoJSONValue{}, err
		}
		result.array = append(result.array, value)
		parser.skipWhitespace()
		if parser.take(']') {
			return result, nil
		}
		if !parser.take(',') {
			return djangoJSONValue{}, ErrInvalidData
		}
	}
}

func (parser *djangoJSONParser) parseString() ([]rune, error) {
	value, next, err := pythoncompat.DecodeJSONStringCodeUnits(parser.input, parser.index)
	if err != nil {
		return nil, ErrInvalidData
	}
	parser.index = next
	return value, nil
}

func (parser *djangoJSONParser) parseNumber() (djangoJSONValue, error) {
	start := parser.index
	parser.take('-')
	if parser.take('0') {
		if parser.index < len(parser.input) && parser.input[parser.index] >= '0' && parser.input[parser.index] <= '9' {
			return djangoJSONValue{}, ErrInvalidData
		}
	} else if !parser.takeDigits() {
		return djangoJSONValue{}, ErrInvalidData
	}
	isFloat := false
	if parser.take('.') {
		isFloat = true
		if !parser.takeDigits() {
			return djangoJSONValue{}, ErrInvalidData
		}
	}
	if parser.take('e') || parser.take('E') {
		isFloat = true
		if !parser.take('+') {
			parser.take('-')
		}
		if !parser.takeDigits() {
			return djangoJSONValue{}, ErrInvalidData
		}
	}
	token := string(parser.input[start:parser.index])
	if isFloat {
		if _, err := strconv.ParseFloat(token, 64); err != nil && !errors.Is(err, strconv.ErrRange) {
			return djangoJSONValue{}, ErrInvalidData
		}
		return djangoJSONValue{kind: djangoJSONFloat, number: token}, nil
	}
	digits := strings.TrimPrefix(token, "-")
	if len(digits) > 4300 {
		return djangoJSONValue{}, ErrInvalidData
	}
	if _, ok := new(big.Int).SetString(token, 10); !ok {
		return djangoJSONValue{}, ErrInvalidData
	}
	return djangoJSONValue{kind: djangoJSONInteger, number: token}, nil
}

func (parser *djangoJSONParser) takeDigits() bool {
	start := parser.index
	for parser.index < len(parser.input) && parser.input[parser.index] >= '0' && parser.input[parser.index] <= '9' {
		parser.index++
	}
	return parser.index > start
}

func (parser *djangoJSONParser) consume(value string) bool {
	characters := []rune(value)
	if parser.index+len(characters) > len(parser.input) {
		return false
	}
	for offset, character := range characters {
		if parser.input[parser.index+offset] != character {
			return false
		}
	}
	parser.index += len(characters)
	return true
}

func (parser *djangoJSONParser) take(character rune) bool {
	if parser.index < len(parser.input) && parser.input[parser.index] == character {
		parser.index++
		return true
	}
	return false
}

func (parser *djangoJSONParser) skipWhitespace() {
	for parser.index < len(parser.input) && strings.ContainsRune(" \t\r\n", parser.input[parser.index]) {
		parser.index++
	}
}

func djangoJSONKeyIdentity(value []rune) string {
	var identity strings.Builder
	for _, character := range value {
		_, _ = fmt.Fprintf(&identity, "%08x", character)
	}
	return identity.String()
}

func djangoJSONPlainValue(value djangoJSONValue) any {
	switch value.kind {
	case djangoJSONNull:
		return nil
	case djangoJSONBool:
		return value.boolean
	case djangoJSONString:
		if djangoRunesContainLoneSurrogate(value.text) {
			return djangoSurrogateString{text: append([]rune(nil), value.text...)}
		}
		return djangoRunesToString(value.text)
	case djangoJSONInteger:
		return json.Number(value.number)
	case djangoJSONFloat:
		switch value.number {
		case "NaN":
			return math.NaN()
		case "Infinity":
			return math.Inf(1)
		case "-Infinity":
			return math.Inf(-1)
		default:
			parsed, _ := strconv.ParseFloat(value.number, 64)
			return parsed
		}
	case djangoJSONArray:
		result := make([]any, 0, len(value.array))
		for _, item := range value.array {
			result = append(result, djangoJSONPlainValue(item))
		}
		return result
	case djangoJSONObject:
		result := djangoOrderedObject{entries: make([]djangoPlainEntry, 0, len(value.object.entries))}
		for _, entry := range value.object.entries {
			result.entries = append(result.entries, djangoPlainEntry{
				key:             djangoRunesToString(entry.key),
				keyHasSurrogate: djangoRunesContainLoneSurrogate(entry.key),
				value:           djangoJSONPlainValue(entry.value),
			})
		}
		return result
	default:
		return nil
	}
}

func djangoRunesToString(value []rune) string {
	var result strings.Builder
	for index := 0; index < len(value); index++ {
		character := value[index]
		if utf16.IsSurrogate(character) {
			if index+1 < len(value) && character >= 0xd800 && character <= 0xdbff && value[index+1] >= 0xdc00 && value[index+1] <= 0xdfff {
				result.WriteRune(utf16.DecodeRune(character, value[index+1]))
				index++
				continue
			}
			result.WriteRune('\ufffd')
			continue
		}
		result.WriteRune(character)
	}
	return result.String()
}

func djangoRunesContainLoneSurrogate(value []rune) bool {
	for index := 0; index < len(value); index++ {
		character := value[index]
		if !utf16.IsSurrogate(character) {
			continue
		}
		if character >= 0xd800 && character <= 0xdbff && index+1 < len(value) && value[index+1] >= 0xdc00 && value[index+1] <= 0xdfff {
			index++
			continue
		}
		return true
	}
	return false
}

func writeDjangoJSON(output *bytes.Buffer, value djangoJSONValue) {
	switch value.kind {
	case djangoJSONNull:
		output.WriteString("null")
	case djangoJSONBool:
		if value.boolean {
			output.WriteString("true")
		} else {
			output.WriteString("false")
		}
	case djangoJSONString:
		writeDjangoJSONString(output, value.text)
	case djangoJSONInteger:
		integer, _ := new(big.Int).SetString(value.number, 10)
		output.WriteString(integer.String())
	case djangoJSONFloat:
		writeDjangoJSONFloat(output, value.number)
	case djangoJSONArray:
		output.WriteByte('[')
		for index, item := range value.array {
			if index > 0 {
				output.WriteByte(',')
			}
			writeDjangoJSON(output, item)
		}
		output.WriteByte(']')
	case djangoJSONObject:
		output.WriteByte('{')
		for index, entry := range value.object.entries {
			if index > 0 {
				output.WriteByte(',')
			}
			writeDjangoJSONString(output, entry.key)
			output.WriteByte(':')
			writeDjangoJSON(output, entry.value)
		}
		output.WriteByte('}')
	}
}

func writeDjangoJSONFloat(output *bytes.Buffer, token string) {
	if token == "NaN" || token == "Infinity" || token == "-Infinity" {
		output.WriteString(token)
		return
	}
	value, _ := strconv.ParseFloat(token, 64)
	if math.IsNaN(value) {
		output.WriteString("NaN")
		return
	}
	if math.IsInf(value, 1) {
		output.WriteString("Infinity")
		return
	}
	if math.IsInf(value, -1) {
		output.WriteString("-Infinity")
		return
	}
	output.WriteString(pythoncompat.FloatString(value))
}

func pythonFloatString(value float64) string {
	return pythoncompat.FloatString(value)
}

func writeDjangoJSONString(output *bytes.Buffer, value []rune) {
	const hex = "0123456789abcdef"
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
			if character >= 0x20 && character < 0x7f {
				output.WriteRune(character)
				continue
			}
			writeCodeUnit := func(code uint16) {
				output.WriteString(`\u`)
				output.WriteByte(hex[code>>12])
				output.WriteByte(hex[(code>>8)&0xf])
				output.WriteByte(hex[(code>>4)&0xf])
				output.WriteByte(hex[code&0xf])
			}
			if character <= 0xffff {
				writeCodeUnit(uint16(character))
			} else {
				high, low := utf16.EncodeRune(character)
				writeCodeUnit(uint16(high))
				writeCodeUnit(uint16(low))
			}
		}
	}
	output.WriteByte('"')
}
