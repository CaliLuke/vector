package requestinput

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"mime/multipart"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/google/uuid"
)

type UploadedFile struct{ Name string }

func (f UploadedFile) String() string { return f.Name }

type Input struct {
	Raw    []byte
	Fields map[string]any

	Kind       string
	Null       bool
	Form       bool
	FormValues map[string][]any
	FormKeys   []string
}

func Parse(body io.Reader, contentType string) (Input, error) {
	input := Input{Fields: map[string]any{}}
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
		if !jsontext.Value(raw).IsValid(jsontext.AllowDuplicateNames(true)) {
			return input, Failure(400, "detail", "Invalid JSON body.")
		}
		var value any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return input, Failure(400, "detail", "Invalid JSON body.")
		}
		if fields, ok := value.(map[string]any); ok {
			input.Fields = fields
		} else {
			input.Kind, input.Null = TypeName(value), value == nil
		}
	case "application/x-www-form-urlencoded":
		input.Form = true
		for field := range strings.SplitSeq(string(raw), "&") {
			if field == "" {
				continue
			}
			key, value, _ := strings.Cut(field, "=")
			key, keyErr := url.QueryUnescape(key)
			value, valueErr := url.QueryUnescape(value)
			if keyErr != nil || valueErr != nil || !utf8.ValidString(key) || !utf8.ValidString(value) {
				return input, Failure(400, "detail", "Invalid URL-encoded form body.")
			}
			input.addFormValue(key, value)
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
				return input, Failure(400, "detail", "Invalid multipart form body.")
			}
			data, err := io.ReadAll(part)
			if err != nil {
				return input, err
			}
			if !utf8.ValidString(part.FormName()) || !utf8.ValidString(part.FileName()) {
				return input, Failure(400, "detail", "Invalid UTF-8 form field.")
			}
			if part.FormName() != "" {
				if part.FileName() != "" {
					input.addFormValue(part.FormName(), UploadedFile{Name: part.FileName()})
				} else {
					if !utf8.Valid(data) {
						return input, Failure(400, "detail", "Invalid UTF-8 form field.")
					}
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

func TypeName(value any) string {
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
		return string(value)
	default:
		return fmt.Sprint(value)
	}
}

func CharField(value any, allowBlank bool) (string, []string) {
	return TextField(value, allowBlank, 255)
}

// TextField trims and validates text. A zero maximum leaves its length unconstrained.
func TextField(value any, allowBlank bool, maximum int) (string, []string) {
	if value == nil {
		return "", []string{"This field may not be null."}
	}
	switch value.(type) {
	case string, json.Number:
	default:
		return "", []string{"Not a valid string."}
	}
	// Treat ASCII separators as whitespace alongside Unicode spaces.
	if !utf8.ValidString(ScalarString(value)) {
		return "", []string{"Not a valid UTF-8 string."}
	}
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
	return text, failures
}

// UUID accepts standard textual UUID representations.
func UUID(value any) (uuid.UUID, error) {
	text, ok := value.(string)
	if !ok {
		return uuid.Nil, errors.New("UUID must be a string")
	}
	if err := uuid.Validate(text); err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(text)
}

func FloatField(input any) (float64, string) {
	var text string
	switch value := input.(type) {
	case string:
		if len(value) > 1000 {
			return 0, "String value too large."
		}
		text = strings.TrimSpace(value)
	case json.Number:
		text = string(value)
	default:
		return 0, "A valid number is required."
	}
	value, err := Float(text)
	if err != nil {
		return 0, "A valid number is required."
	}
	return value, ""
}

func Float(text string) (float64, error) {
	if strings.ContainsAny(text, "_xXpP") {
		return 0, errors.New("invalid decimal number")
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, errors.New("number must be finite")
	}
	return value, nil
}

func Failure(status int, field, message string) error {
	return &apicontract.HTTPError{Status: status, Body: map[string]any{field: message}}
}
