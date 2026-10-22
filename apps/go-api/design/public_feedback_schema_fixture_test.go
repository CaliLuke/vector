package design

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	libvalidator "github.com/pb33f/libopenapi-validator"
)

const (
	publicFeedbackOpenAPI = "../gen/http/openapi.json"
)

func TestPublicFeedbackVoteResponseSchemasMatchPristineFixtures(t *testing.T) {
	validator := publicFeedbackValidator(t)
	defer validator.Release()

	fixtures := []string{
		"vote-post-integer-coercion-boolean-pristine.json",
		"vote-post-integer-coercion-unicode-digit-pristine.json",
		"vote-post-integer-coercion-whitespace-string-pristine.json",
		"vote-post-integer-coercion-float-pristine.json",
	}
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			fixture := readPublicFeedbackVoteFixture(t, name)
			request := publicFeedbackRequest(t, fixture.Request.Method, fixture.Request.Path)
			response := publicFeedbackResponse(t, fixture.Response.Status, fixture.Response.Body)
			valid, validationErrors := validator.ValidateHttpResponse(request, response)
			if !valid {
				t.Fatalf("pristine POST response does not match generated OpenAPI: %v", validationErrors)
			}
		})
	}

	// Exercise the GET list item schema with the same recorded response fields.
	// Integer vote values must pass, while a boolean source scalar must remain
	// rejected by the persisted-list contract.
	fixture := readPublicFeedbackVoteFixture(t, fixtures[0])
	item := make(map[string]json.RawMessage)
	if err := json.Unmarshal(fixture.Response.Body, &item); err != nil {
		t.Fatalf("decode vote fixture response body: %v", err)
	}
	item["vote"] = json.RawMessage("1")
	integerItem, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	getRequest := publicFeedbackRequest(t, http.MethodGet, fixture.Request.Path)
	getResponse := publicFeedbackResponse(t, http.StatusOK, mustJSON(t, []json.RawMessage{integerItem}))
	valid, validationErrors := validator.ValidateHttpResponse(getRequest, getResponse)
	if !valid {
		t.Fatalf("GET vote list rejected integer vote item: %v", validationErrors)
	}

	item["vote"] = json.RawMessage("true")
	booleanItem, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	getResponse = publicFeedbackResponse(t, http.StatusOK, mustJSON(t, []json.RawMessage{booleanItem}))
	valid, validationErrors = validator.ValidateHttpResponse(getRequest, getResponse)
	if valid {
		t.Fatal("GET vote list accepted boolean vote, want persisted integer schema to reject it")
	}
	if len(validationErrors) == 0 {
		t.Fatal("GET vote list rejected boolean vote without reporting a validation error")
	}
}

func publicFeedbackValidator(t *testing.T) libvalidator.Validator {
	t.Helper()
	path := publicFeedbackOpenAPI
	if override := os.Getenv("PUBLIC_FEEDBACK_OPENAPI_PATH"); override != "" {
		path = override
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read OpenAPI document %s: %v", path, err)
	}
	document, err := libopenapi.NewDocument(data)
	if err != nil {
		t.Fatalf("parse OpenAPI document %s: %v", path, err)
	}
	validator, errs := libvalidator.NewValidator(document)
	if len(errs) != 0 {
		t.Fatalf("build OpenAPI validator: %v", errs)
	}
	return validator
}

type publicFeedbackVoteFixture struct {
	Request struct {
		Method string `json:"method"`
		Path   string `json:"path"`
	} `json:"request"`
	Response struct {
		Status int             `json:"status"`
		Body   json.RawMessage `json:"body"`
	} `json:"response"`
}

func readPublicFeedbackVoteFixture(t *testing.T, name string) publicFeedbackVoteFixture {
	t.Helper()
	path := "../internal/parity/testdata/fixtures/public_feedback/" + name
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read pristine fixture %s: %v", path, err)
	}
	var fixture publicFeedbackVoteFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("decode pristine fixture %s: %v", path, err)
	}
	if fixture.Request.Method != http.MethodPost || fixture.Response.Status != http.StatusCreated {
		t.Fatalf("fixture protocol = %s %s -> %d, want POST -> 201", fixture.Request.Method, fixture.Request.Path, fixture.Response.Status)
	}
	fixture.Request.Path = replacePublicFeedbackUUIDAliases(fixture.Request.Path)
	fixture.Response.Body = []byte(replacePublicFeedbackUUIDAliases(string(fixture.Response.Body)))
	return fixture
}

func publicFeedbackRequest(t *testing.T, method, path string) *http.Request {
	t.Helper()
	request, err := http.NewRequest(method, "http://plane.test"+replacePublicFeedbackUUIDAliases(path), nil)
	if err != nil {
		t.Fatalf("create %s validation request: %v", method, err)
	}
	return request
}

func publicFeedbackResponse(t *testing.T, status int, body []byte) *http.Response {
	t.Helper()
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func replacePublicFeedbackUUIDAliases(value string) string {
	return strings.NewReplacer(
		"<uuid-1>", "00000000-0000-4000-8000-000000000001",
		"<uuid-2>", "00000000-0000-4000-8000-000000000002",
		"<uuid-3>", "00000000-0000-4000-8000-000000000003",
		"<uuid-4>", "00000000-0000-4000-8000-000000000004",
		"<uuid-5>", "00000000-0000-4000-8000-000000000005",
		"<uuid-6>", "00000000-0000-4000-8000-000000000006",
		"<uuid-7>", "00000000-0000-4000-8000-000000000007",
		"<uuid-8>", "00000000-0000-4000-8000-000000000008",
	).Replace(value)
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal response fixture: %v", err)
	}
	return data
}
