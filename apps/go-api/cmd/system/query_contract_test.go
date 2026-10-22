package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// These names come from Django GET.get/query_params.get consumers, including
// issue_filters.py, paginator.py, base serializers, and the retired views.
// Keep this independent of readServiceScalarQueryParameters: that map is tested.
var djangoSourceScalarQueryNames = strings.Fields(`
assignees completed_at created_at created_by cursor cycle cycle_view entity_name
estimate_point expand fields filters format group_by grouped inbox_status
intake_status labels logged_by mentions module name order_by owner parent per_page
priority project project_ids query search slug start_date start_target_date state
state_group status sub_group_by sub_issue subscriber target_date track_visit type updated_at
`)

// IssueFilterSet uses scalar widgets, then splits CSV values where applicable.
var djangoSourceScalarFilterAliases = strings.Fields(`
assignee_id assignee_id__in created_at__exact created_at__range created_by_id
created_by_id__in cycle_id__in is_archived is_draft label_id label_id__in mention_id
mention_id__in module_id module_id__in priority__in project_id__in start_date__exact
start_date__range state_group__in state_id state_id__in subscriber_id subscriber_id__in
 target_date__exact target_date__range updated_at__exact updated_at__range
`)

type djangoQueryContractFixture struct {
	LimitFieldCount int               `json:"limit_field_count"`
	RawQuery        string            `json:"raw_query"`
	Values          url.Values        `json:"values"`
	Scalars         map[string]string `json:"scalars"`
	FieldCount      int               `json:"field_count"`
	Accepted        bool              `json:"accepted"`
	ValuesCount     int               `json:"values_count"`
}

func loadDjangoQueryContractFixtures(t *testing.T) []djangoQueryContractFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/query_compatibility.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []djangoQueryContractFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("Django query fixture is empty")
	}
	return fixtures
}

func TestDjangoQueryContractRecordedValues(t *testing.T) {
	for _, fixture := range loadDjangoQueryContractFixtures(t) {
		if fixture.FieldCount != 0 || fixture.LimitFieldCount != 0 {
			continue
		}
		t.Run(fixture.RawQuery, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/api/example/", nil)
			request.URL.RawQuery = fixture.RawQuery
			request.RequestURI = "/api/example/?" + fixture.RawQuery
			originalURL := *request.URL
			originalURI := request.RequestURI
			request = request.WithContext(context.WithValue(request.Context(), readServicesOriginalRawQueryKey{}, fixture.RawQuery))

			decoded := normalizeDjangoTolerantQuery(request)
			if got := decoded.URL.Query(); !reflect.DeepEqual(got, fixture.Values) {
				t.Errorf("decoded query = %#v; Django recorded %#v", got, fixture.Values)
			}
			expected := make(url.Values, len(fixture.Values))
			for key, values := range fixture.Values {
				expected[key] = slices.Clone(values)
				if slices.Contains(djangoSourceScalarQueryNames, key) || slices.Contains(djangoSourceScalarFilterAliases, key) {
					expected[key] = []string{fixture.Scalars[key]}
				}
			}
			normalized := normalizeReadServiceScalarQueryValues(decoded)
			if got := normalized.URL.Query(); !reflect.DeepEqual(got, expected) {
				t.Errorf("normalized query = %#v; want %#v", got, expected)
			}
			if *request.URL != originalURL || request.RequestURI != originalURI {
				t.Error("normalization mutated the original request URL or RequestURI")
			}
			if normalized.RequestURI != originalURI || readServicesOriginalRawQuery(normalized) != fixture.RawQuery {
				t.Error("normalization changed original request evidence")
			}
			again := normalizeReadServiceScalarQueryValues(normalizeDjangoTolerantQuery(normalized))
			if *again.URL != *normalized.URL || again.RequestURI != originalURI || readServicesOriginalRawQuery(again) != fixture.RawQuery {
				t.Error("normalization is not idempotent or changed original request evidence")
			}
		})
	}
}

func TestDjangoQueryContractSourceScalarNames(t *testing.T) {
	names := append(slices.Clone(djangoSourceScalarQueryNames), djangoSourceScalarFilterAliases...)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			for _, last := range []string{"last", "", "%ZZ", "a,b"} {
				request := httptest.NewRequest("GET", "/api/example/", nil)
				request.URL.RawQuery = url.QueryEscape(name) + "=first&" + url.QueryEscape(name) + "=" + last + "&unknown=first&unknown=&unknown=last"
				normalized := normalizeReadServiceScalarQueryValues(normalizeDjangoTolerantQuery(request))
				expected := url.Values{name: {last}, "unknown": {"first", "", "last"}}
				if got := normalized.URL.Query(); !reflect.DeepEqual(got, expected) {
					t.Errorf("last %q: got %#v; want %#v", last, got, expected)
				}
			}
		})
	}
}

func TestDjangoQueryContractRecordedFieldLimit(t *testing.T) {
	for _, fixture := range loadDjangoQueryContractFixtures(t) {
		if fixture.FieldCount == 0 {
			continue
		}
		t.Run(fmt.Sprint(fixture.FieldCount), func(t *testing.T) {
			raw := strings.TrimSuffix(strings.Repeat("query=value&", fixture.FieldCount), "&")
			values, err := djangoQueryValues(raw, "query")
			if fixture.Accepted {
				if err != nil {
					t.Fatal(err)
				}
				if len(values) != fixture.ValuesCount {
					t.Errorf("got %d values; want %d", len(values), fixture.ValuesCount)
				}
			} else if err == nil {
				t.Fatal("accepted a query exceeding Django's original field-count limit")
			}
		})
	}
}

func TestDjangoQueryContractOpenAPIScalarCoverage(t *testing.T) {
	names := make(map[string]bool)
	for _, path := range []string{"../../gen/http/openapi.json", "../../activityreads/gen/http/openapi.json"} {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var document map[string]any
			if err := json.Unmarshal(data, &document); err != nil {
				t.Fatal(err)
			}
			paths, ok := document["paths"].(map[string]any)
			if !ok || len(paths) == 0 {
				t.Fatal("OpenAPI has no paths")
			}
			for route, item := range paths {
				pathItem := resolveQueryContractObject(t, document, item)
				inherited, _ := pathItem["parameters"].([]any)
				for _, method := range []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"} {
					operation, exists := pathItem[method]
					if !exists {
						continue
					}
					operationObject := resolveQueryContractObject(t, document, operation)
					parameters, _ := operationObject["parameters"].([]any)
					parameters = append(slices.Clone(inherited), parameters...)
					for _, value := range parameters {
						parameter := resolveQueryContractObject(t, document, value)
						if parameter["in"] != "query" {
							continue
						}
						name, ok := parameter["name"].(string)
						if !ok || name == "" {
							t.Fatalf("%s %s: query has no name", method, route)
						}
						schema := resolveQueryContractObject(t, document, parameter["schema"])
						_, mapped := readServiceScalarQueryParameters[name]
						if schema["type"] == "array" {
							if mapped {
								t.Errorf("%s %s: array query %q collides with scalar normalization", method, route, name)
							}
							continue
						}
						if schema["type"] == nil {
							t.Fatalf("%s %s: query %q has no resolved schema type", method, route, name)
						}
						names[name] = true
						if !mapped {
							t.Errorf("%s %s: scalar query %q is absent from normalization map", method, route, name)
						}
					}
				}
			}
		})
	}
	// Require every source-characterized name to remain represented, including
	// activity's reference-defined expand parameter.
	for _, name := range djangoSourceScalarQueryNames {
		if !names[name] {
			t.Errorf("source-characterized query %q was not found in either OpenAPI document", name)
		}
	}
}

func resolveQueryContractObject(t *testing.T, document map[string]any, value any) map[string]any {
	t.Helper()
	seen := make(map[string]bool)
	for {
		object, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("expected OpenAPI object, got %T", value)
		}
		reference, exists := object["$ref"]
		if !exists {
			return object
		}
		ref, ok := reference.(string)
		if !ok || !strings.HasPrefix(ref, "#/") || seen[ref] {
			t.Fatalf("unsupported or cyclic OpenAPI reference: %v", reference)
		}
		seen[ref] = true
		value = document
		for _, segment := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
			key := strings.ReplaceAll(strings.ReplaceAll(segment, "~1", "/"), "~0", "~")
			current, ok := value.(map[string]any)
			if !ok {
				t.Fatalf("unresolved OpenAPI reference %q", ref)
			}
			value, ok = current[key]
			if !ok {
				t.Fatalf("unresolved OpenAPI reference %q", ref)
			}
		}
	}
}
