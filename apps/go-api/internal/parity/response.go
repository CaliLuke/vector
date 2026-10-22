package parity

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/google/go-cmp/cmp"
)

// LoadCapturedResponse reads a strict recorder-canonicalized response JSON file.
func LoadCapturedResponse(path string) (CapturedResponse, error) {
	file, err := os.Open(path)
	if err != nil {
		return CapturedResponse{}, fmt.Errorf("open captured response: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var response CapturedResponse
	if err := decoder.Decode(&response); err != nil {
		return CapturedResponse{}, fmt.Errorf("decode captured response: %w", err)
	}
	if err := requireResponseEOF(decoder); err != nil {
		return CapturedResponse{}, err
	}
	return response, nil
}

func requireResponseEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode captured response: multiple JSON values")
		}
		return fmt.Errorf("decode captured response: %w", err)
	}
	return nil
}

// CapturedResponse is a recorder-canonicalized HTTP response supplied by each runtime action.
// Values remain private input; go-cmp sees only SafeResponseProjection.
type CapturedResponse struct {
	Status  int                        `json:"status"`
	Headers map[string]json.RawMessage `json:"headers"`
	Cookies map[string]json.RawMessage `json:"cookies"`
	Body    json.RawMessage            `json:"body"`
}

// SafeResponseProjection locates response differences without exposing response values.
type SafeResponseProjection struct {
	Status      int
	Headers     []NamedFingerprint
	Cookies     []NamedFingerprint
	BodySHA256  string
	BodyPresent bool
}

// NamedFingerprint retains a safe response field name and hashes its value.
type NamedFingerprint struct {
	Name   string
	SHA256 string
}

// SafeResponseOption is the named go-cmp option used for every response comparison.
func SafeResponseOption() cmp.Option {
	return cmp.Transformer("SafeResponseProjection", func(response CapturedResponse) SafeResponseProjection {
		return SafeResponseProjection{
			Status:      response.Status,
			Headers:     projectNamedValues(response.Headers),
			Cookies:     projectNamedValues(response.Cookies),
			BodySHA256:  fingerprint(response.Body),
			BodyPresent: len(response.Body) != 0,
		}
	})
}

func (response CapturedResponse) validate(role string) error {
	if response.Status < 100 || response.Status > 599 {
		return fmt.Errorf("%s response status must be between 100 and 599", role)
	}
	if len(response.Body) == 0 || !json.Valid(response.Body) {
		return fmt.Errorf("%s response body must be valid JSON", role)
	}
	for kind, values := range map[string]map[string]json.RawMessage{
		"header": response.Headers,
		"cookie": response.Cookies,
	} {
		for name, value := range values {
			if name == "" || !json.Valid(value) {
				return fmt.Errorf("%s response %s %q must have a name and valid JSON value", role, kind, name)
			}
		}
	}
	return nil
}

func projectNamedValues(values map[string]json.RawMessage) []NamedFingerprint {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	slices.Sort(names)
	projection := make([]NamedFingerprint, 0, len(names))
	for _, name := range names {
		projection = append(projection, NamedFingerprint{Name: name, SHA256: fingerprint(bytes.TrimSpace(values[name]))})
	}
	return projection
}

func fingerprint(value json.RawMessage) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}
