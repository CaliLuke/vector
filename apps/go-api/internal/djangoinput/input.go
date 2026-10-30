package djangoinput

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"mime"
	"mime/multipart"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/google/uuid"
)

type UploadedFile struct{ Name string }

func (f UploadedFile) String() string { return f.Name }

type Input struct {
	Raw        []byte
	Fields     map[string]any
	Surrogates map[string][]string
	Kind       string
	Null       bool
	Form       bool
	FormValues map[string][]any
	FormKeys   []string
}

func Parse(body io.Reader, contentType string) (Input, error) {
	input := Input{Fields: map[string]any{}, Surrogates: map[string][]string{}}
	mediaType, parameters, mediaTypeErr := mime.ParseMediaType(contentType)
	if mediaTypeErr != nil && (mediaType == "application/json" || isJSONContentType(contentType)) {
		return input, Failure(415, "detail", "Invalid Content-Type for application/json.")
	}
	if body == nil {
		return input, nil
	}
	raw, err := io.ReadAll(body)
	if err != nil {
		return input, err
	}
	if len(raw) == 0 {
		return input, nil
	}
	input.Raw = raw
	switch mediaType {
	case "application/json":
		var truncatedTail bool
		var truncatedByte byte
		raw, truncatedByte, truncatedTail, err = validateJSONRequest(raw)
		if err != nil {
			return input, err
		}
		input.Raw = raw
		var value any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			if truncatedTail && (errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)) {
				if syntaxErr, ok := unterminatedJSONStringError(raw); ok {
					return input, syntaxErr
				}
			}
			return input, jsonInputError(raw, err)
		}
		end := int(decoder.InputOffset())
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			for end < len(raw) && strings.ContainsRune(" \t\r\n", rune(raw[end])) {
				end++
			}
			return input, jsonInputErrorAt(raw, end, "Extra data")
		}
		if truncatedTail {
			return input, Failure(400, "detail", fmt.Sprintf("JSON parse error - 'utf-8' codec can't decode byte 0x%02x in position %d: unexpected end of data", truncatedByte, len(raw)))
		}
		if fields, ok := value.(map[string]any); ok {
			input.Fields = fields
			var encoded map[string]json.RawMessage
			if err := json.Unmarshal(raw, &encoded); err != nil {
				return input, err
			}
			for key, data := range encoded {
				input.Surrogates[key] = unpairedSurrogates(data)
			}
		} else {
			input.Kind, input.Null = PythonType(value), value == nil
		}
	case "application/x-www-form-urlencoded":
		input.Form = true
		for field := range strings.SplitSeq(string(raw), "&") {
			if field == "" {
				continue
			}
			key, value, _ := strings.Cut(field, "=")
			input.addFormValue(decodeFormValue(key), decodeFormValue(value))
		}
	case "multipart/form-data":
		input.Form = true
		reader := multipart.NewReader(bytes.NewReader(raw), parameters["boundary"])
		for {
			part, nextErr := reader.NextPart()
			// Only bare EOF ends iteration; wrapped EOF remains a parse failure.
			if nextErr == io.EOF {
				break
			}
			if nextErr != nil {
				return input, Failure(400, "detail", "Multipart form parse error - Invalid boundary in multipart: None")
			}
			data, err := io.ReadAll(part)
			_ = part.Close()
			if err != nil {
				return input, err
			}
			if part.FormName() != "" {
				if part.FileName() != "" {
					input.addFormValue(part.FormName(), UploadedFile{Name: part.FileName()})
				} else {
					input.addFormValue(part.FormName(), string(data))
				}
			}
		}
	default:
		return input, Failure(415, "detail", fmt.Sprintf("Unsupported media type %q in request.", contentType))
	}
	return input, nil
}

func isJSONContentType(contentType string) bool {
	// ParseMediaType clears the type on conflicting parameters. The type itself
	// precedes all parameters, so recover only its identity for the JSON error.
	mediaType, _, _ := strings.Cut(contentType, ";")
	return strings.EqualFold(strings.TrimSpace(mediaType), "application/json")
}

// validateJSONRequest enforces UTF-8 before JSON parsing.
func validateJSONRequest(raw []byte) (validated []byte, truncatedByte byte, truncatedTail bool, err error) {
	if !utf8.Valid(raw) {
		if complete, tailByte, tail := validUTF8Prefix(raw); tail {
			return complete, tailByte, true, nil
		}
		start, end, kind := invalidUTF8Bytes(raw)
		if end > start {
			return raw, 0, false, Failure(400, "detail", fmt.Sprintf("JSON parse error - 'utf-8' codec can't decode bytes in position %d-%d: invalid %s byte", start, end, kind))
		}
		return raw, 0, false, Failure(400, "detail", fmt.Sprintf("JSON parse error - 'utf-8' codec can't decode byte 0x%02x in position %d: invalid %s byte", raw[start], start, kind))
	}
	return raw, 0, false, nil
}

// validUTF8Prefix reports the complete UTF-8 prefix only when the body ends in
// an incomplete code point. Django's buffered decoder drops that trailing code
// point, leaving JSON parsing to report the resulting syntax error.
func validUTF8Prefix(raw []byte) ([]byte, byte, bool) {
	for i := 0; i < len(raw); {
		if !utf8.FullRune(raw[i:]) {
			return raw[:i], raw[i], true
		}
		_, size := utf8.DecodeRune(raw[i:])
		if size == 1 && raw[i] >= utf8.RuneSelf {
			return nil, 0, false
		}
		i += size
	}
	return nil, 0, false
}

func invalidUTF8Bytes(raw []byte) (start, end int, kind string) {
	for i := 0; i < len(raw); {
		if !utf8.FullRune(raw[i:]) {
			return i, i, "continuation"
		}
		r, size := utf8.DecodeRune(raw[i:])
		if r == utf8.RuneError && size == 1 && raw[i] >= utf8.RuneSelf {
			if utf8.FullRune(raw[i : i+1]) {
				return i, i, "start"
			}
			for length := 2; i+length <= len(raw); length++ {
				if utf8.FullRune(raw[i : i+length]) {
					return i, i + length - 2, "continuation"
				}
			}
			return i, i, "continuation"
		}
		i += size
	}
	return 0, 0, "start"
}

func unterminatedJSONStringError(raw []byte) (error, bool) {
	decoder := jsontext.NewDecoder(bytes.NewReader(raw), jsontext.AllowDuplicateNames(true), jsontext.AllowInvalidUTF8(true))
	for {
		before := int(decoder.InputOffset())
		if _, err := decoder.ReadToken(); err != nil {
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				return nil, false
			}
			for before < len(raw) && strings.ContainsRune(" \t\r\n:,", rune(raw[before])) {
				before++
			}
			if before < len(raw) && raw[before] == '"' {
				return jsonInputErrorAt(raw, before, "Unterminated string starting at"), true
			}
			return nil, false
		}
	}
}

func (input *Input) addFormValue(key string, value any) {
	if input.FormValues == nil {
		input.FormValues = map[string][]any{}
	}
	if _, present := input.FormValues[key]; !present {
		input.FormKeys = append(input.FormKeys, key)
	}
	input.FormValues[key] = append(input.FormValues[key], value)
	input.Fields[key] = value
}

func decodeFormValue(value string) string {
	// urllib leaves invalid percent escapes literal and accepts semicolons in
	// values. QueryUnescape needs invalid percent signs protected first.
	var escaped strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == '%' && (i+2 >= len(value) || !hexDigit(value[i+1]) || !hexDigit(value[i+2])) {
			escaped.WriteString("%25")
		} else {
			escaped.WriteByte(value[i])
		}
	}
	decoded, _ := url.QueryUnescape(escaped.String())
	return decoded
}

func hexDigit(value byte) bool {
	return value >= '0' && value <= '9' || value >= 'a' && value <= 'f' || value >= 'A' && value <= 'F'
}

func jsonInputError(raw []byte, err error) error {
	position, reason := 0, "Expecting value"
	var syntax *json.SyntaxError
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		position = len(raw)
		// A sentinel at EOF lets encoding/json report which grammar element is
		// incomplete, rather than losing that state to io.ErrUnexpectedEOF.
		probe := append(append([]byte(nil), raw...), '?')
		var value any
		err = json.Unmarshal(probe, &value)
	}
	if errors.As(err, &syntax) {
		position = int(syntax.Offset) - 1
		switch {
		case strings.Contains(syntax.Error(), "object key string"):
			reason = "Expecting property name enclosed in double quotes"
		case strings.Contains(syntax.Error(), "after object key:value pair"), strings.Contains(syntax.Error(), "after array element"):
			reason = "Expecting ',' delimiter"
		case strings.Contains(syntax.Error(), "after object key"):
			reason = "Expecting ':' delimiter"
		case strings.Contains(syntax.Error(), "in string escape code"), strings.Contains(syntax.Error(), "invalid escape sequence"):
			reason = `Invalid \escape`
			position-- // Python points to the backslash, not its following byte.
		}
	}
	return jsonInputErrorAt(raw, position, reason)
}

func jsonInputErrorAt(raw []byte, position int, reason string) error {
	position = max(0, min(position, len(raw)))
	prefix := string(raw[:position])
	line := strings.Count(prefix, "\n") + 1
	column := utf8.RuneCountInString(prefix[strings.LastIndex(prefix, "\n")+1:]) + 1
	return Failure(400, "detail", fmt.Sprintf("JSON parse error - %s: line %d column %d (char %d)", reason, line, column, utf8.RuneCountInString(prefix)))
}

// encoding/json replaces unpaired UTF-16 escapes. Preserve their identity for
// DRF's field validator instead of confusing them with a literal replacement rune.
func unpairedSurrogates(raw []byte) []string {
	if len(raw) == 0 || raw[0] != '"' {
		return nil
	}
	var result []string
	for i := 1; i < len(raw)-1; i++ {
		if raw[i] != '\\' {
			continue
		}
		if i+5 >= len(raw) || raw[i+1] != 'u' {
			i++
			continue
		}
		value, err := strconv.ParseUint(string(raw[i+2:i+6]), 16, 16)
		if err != nil {
			continue
		}
		if value >= 0xd800 && value <= 0xdbff && i+11 < len(raw) && string(raw[i+6:i+8]) == `\u` {
			low, err := strconv.ParseUint(string(raw[i+8:i+12]), 16, 16)
			if err == nil && low >= 0xdc00 && low <= 0xdfff {
				i += 11
				continue
			}
		}
		if value >= 0xd800 && value <= 0xdfff {
			result = append(result, fmt.Sprintf("U+%04X", value))
		}
		i += 5
	}
	return result
}

func PythonType(value any) string {
	switch typed := value.(type) {
	case nil:
		return "NoneType"
	case []any:
		return "list"
	case bool:
		return "bool"
	case string:
		return "str"
	case json.Number:
		if strings.ContainsAny(string(typed), ".eE") {
			return "float"
		}
		return "int"
	default:
		return "dict"
	}
}

func ScalarString(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case json.Number:
		if strings.ContainsAny(string(value), ".eE") {
			number, _ := value.Float64()
			if math.IsInf(number, 1) {
				return "inf"
			}
			if math.IsInf(number, -1) {
				return "-inf"
			}
			return pythoncompat.FloatString(number)
		}
		if value == "-0" {
			return "0"
		}
		return string(value)
	case bool:
		if value {
			return "True"
		}
		return "False"
	case nil:
		return "None"
	default:
		return fmt.Sprint(value)
	}
}

func CharField(value any, allowBlank bool, surrogates []string) (string, []string) {
	return TextField(value, allowBlank, 255, surrogates)
}

// TextField applies DRF string coercion and validators. A zero maximum leaves
// TextField lengths unconstrained while retaining null and surrogate checks.
func TextField(value any, allowBlank bool, maximum int, surrogates []string) (string, []string) {
	if value == nil {
		return "", []string{"This field may not be null."}
	}
	switch value.(type) {
	case string, json.Number:
	default:
		return "", []string{"Not a valid string."}
	}
	// DRF CharField uses Python str.strip(), which also treats these four
	// ASCII separators as whitespace beyond Unicode's White_Space property.
	text := strings.TrimFunc(ScalarString(value), func(r rune) bool {
		return unicode.IsSpace(r) || r >= '\u001c' && r <= '\u001f'
	})
	if text == "" && !allowBlank {
		return "", []string{"This field may not be blank."}
	}
	var failures []string
	if maximum > 0 && utf8.RuneCountInString(text) > maximum {
		failures = append(failures, fmt.Sprintf("Ensure this field has no more than %d characters.", maximum))
	}
	if strings.ContainsRune(text, 0) {
		failures = append(failures, "Null characters are not allowed.")
	}
	if len(surrogates) > 0 {
		failures = append(failures, "Surrogate characters are not allowed: "+surrogates[0]+".")
	}
	return text, failures
}

// Django UUIDField uses UUID(int=value) for integer primary-key inputs.
func UUID(value any) (uuid.UUID, error) {
	if number, ok := value.(json.Number); ok && !strings.ContainsAny(string(number), ".eE") {
		integer, ok := new(big.Int).SetString(string(number), 10)
		if !ok || integer.Sign() < 0 || integer.BitLen() > 128 {
			return uuid.Nil, errors.New("integer UUID out of range")
		}
		var id uuid.UUID
		integer.FillBytes(id[:])
		return id, nil
	}
	// Python UUID(hex=...) removes these markers before accepting 32 characters.
	text := strings.ReplaceAll(ScalarString(value), "urn:", "")
	text = strings.ReplaceAll(text, "uuid:", "")
	text = strings.ReplaceAll(strings.Trim(text, "{}"), "-", "")
	if utf8.RuneCountInString(text) != 32 {
		return uuid.Nil, errors.New("UUID hex must contain 32 characters")
	}
	var normalized strings.Builder
	for _, character := range text {
		if character <= 127 {
			normalized.WriteRune(character)
		} else if unicode.IsSpace(character) {
			normalized.WriteByte(' ')
		} else {
			digit, err := pythoncompat.DecimalInteger(string(character))
			if err != nil {
				return uuid.Nil, err
			}
			normalized.WriteString(digit.String())
		}
	}
	text = strings.TrimPrefix(strings.TrimSpace(normalized.String()), "+")
	if !strings.HasPrefix(strings.ToLower(text), "0x") {
		if strings.HasPrefix(text, "_") {
			return uuid.Nil, errors.New("invalid UUID hex separator")
		}
		text = "0x" + text
	}
	integer, ok := new(big.Int).SetString(text, 0)
	if !ok || integer.Sign() < 0 || integer.BitLen() > 128 {
		return uuid.Nil, errors.New("invalid UUID hex")
	}
	var id uuid.UUID
	integer.FillBytes(id[:])
	return id, nil
}

func FloatField(input any) (float64, string) {
	text := ""
	integer := false
	switch input := input.(type) {
	case string:
		if utf8.RuneCountInString(input) > 1000 {
			return 0, "String value too large."
		}
		text = input
	case json.Number:
		integer = !strings.ContainsAny(string(input), ".eE")
		text = ScalarString(input)
	case bool:
		if input {
			return 1, ""
		}
		return 0, ""
	default:
		return 0, "A valid number is required."
	}
	value, err := Float(strings.TrimSpace(text))
	if err != nil {
		return 0, "A valid number is required."
	}
	if integer && math.IsInf(value, 0) {
		return 0, "Integer value too large to convert to float"
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, "A valid number is required."
	}
	return value, ""
}

func Float(text string) (float64, error) {
	switch strings.ToLower(text) {
	case "nan", "+nan", "-nan":
		return math.NaN(), nil
	case "inf", "+inf", "infinity", "+infinity":
		return math.Inf(1), nil
	case "-inf", "-infinity":
		return math.Inf(-1), nil
	default:
		return pythoncompat.DecimalFloat64(text)
	}
}

func Failure(status int, field, message string) error {
	return &djangoapi.HTTPError{Status: status, Body: map[string]any{field: message}}
}
