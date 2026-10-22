package design

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	libvalidator "github.com/pb33f/libopenapi-validator"
)

const pageLifecycleOpenAPI = "../gen/http/openapi.json"

func TestPageLifecycleWriteResponsesMatchOpenAPIFixtures(t *testing.T) {
	data, err := os.ReadFile(pageLifecycleOpenAPI)
	if err != nil {
		t.Fatal(err)
	}
	document, err := libopenapi.NewDocument(data)
	if err != nil {
		t.Fatal(err)
	}
	validator, errs := libvalidator.NewValidator(document)
	if len(errs) != 0 {
		t.Fatalf("build OpenAPI validator: %v", errs)
	}
	defer validator.Release()

	for _, name := range []string{"create-default", "patch-metadata-labels", "duplicate-public"} {
		t.Run(name, func(t *testing.T) {
			fixture := readPageLifecycleSchemaFixture(t, name)
			request, err := http.NewRequest(fixture.Request.Method, "http://plane.test"+fixture.Request.Path, nil)
			if err != nil {
				t.Fatal(err)
			}
			response := &http.Response{
				StatusCode: fixture.Response.Status,
				Header:     http.Header{"Content-Type": {"application/json"}},
				Body:       io.NopCloser(bytes.NewReader(fixture.Response.Body)),
			}
			valid, validationErrors := validator.ValidateHttpResponse(request, response)
			if !valid {
				t.Fatalf("corrected Django success response rejected by generated OpenAPI: %v", validationErrors)
			}
		})
	}

	t.Run("duplicate preserves stored access values outside request choices", func(t *testing.T) {
		fixture := readPageLifecycleSchemaFixture(t, "duplicate-public")
		var body map[string]any
		if err := json.Unmarshal(fixture.Response.Body, &body); err != nil {
			t.Fatal(err)
		}
		body["access"] = float64(2)
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		request, err := http.NewRequest(fixture.Request.Method, "http://plane.test"+fixture.Request.Path, nil)
		if err != nil {
			t.Fatal(err)
		}
		response := &http.Response{StatusCode: fixture.Response.Status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(bytes.NewReader(encoded))}
		valid, validationErrors := validator.ValidateHttpResponse(request, response)
		if !valid {
			t.Fatalf("stored integer access=2 rejected by response schema: %v", validationErrors)
		}
	})
}

func readPageLifecycleSchemaFixture(t *testing.T, name string) pageLifecycleSchemaFixture {
	t.Helper()
	path := "../internal/parity/testdata/fixtures/page_lifecycle_corrected/" + name + ".json"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read corrected source fixture %s: %v", path, err)
	}
	var fixture pageLifecycleSchemaFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("decode corrected source fixture %s: %v", path, err)
	}
	if fixture.Response.Status < 200 || fixture.Response.Status >= 300 || len(fixture.Response.Body) == 0 {
		t.Fatalf("fixture %s is not a successful JSON response", name)
	}
	fixture.Request.Path = normalizePageLifecycleSchemaValues(fixture.Request.Path)
	fixture.Response.Body = []byte(normalizePageLifecycleSchemaValues(string(fixture.Response.Body)))
	return fixture
}

type pageLifecycleSchemaFixture struct {
	Request struct {
		Method string `json:"method"`
		Path   string `json:"path"`
	} `json:"request"`
	Response struct {
		Status int             `json:"status"`
		Body   json.RawMessage `json:"body"`
	} `json:"response"`
}

var pageLifecycleUUIDAlias = regexp.MustCompile(`<uuid-([0-9]+)>`)

func normalizePageLifecycleSchemaValues(value string) string {
	value = pageLifecycleUUIDAlias.ReplaceAllStringFunc(value, func(alias string) string {
		var number int
		_, _ = fmt.Sscanf(alias, "<uuid-%d>", &number)
		return fmt.Sprintf("00000000-0000-4000-8000-%012d", number)
	})
	return strings.ReplaceAll(value, `"<timestamp>"`, `"2026-01-02T03:04:05Z"`)
}
