package design

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

type membershipSchema struct {
	Ref        string                      `json:"$ref"`
	Type       string                      `json:"type"`
	Format     string                      `json:"format"`
	Enum       []int                       `json:"enum"`
	Minimum    *float64                    `json:"minimum"`
	Maximum    *float64                    `json:"maximum"`
	Required   []string                    `json:"required"`
	Properties map[string]membershipSchema `json:"properties"`
	Items      *membershipSchema           `json:"items"`
	AllOf      []membershipSchema          `json:"allOf"`
	AnyOf      []membershipSchema          `json:"anyOf"`
}

type membershipMedia struct {
	Schema membershipSchema `json:"schema"`
}
type membershipContent struct {
	Ref     string                     `json:"$ref"`
	Content map[string]membershipMedia `json:"content"`
}
type membershipOperation struct {
	Responses   map[string]membershipContent `json:"responses"`
	RequestBody membershipContent            `json:"requestBody"`
}
type membershipOpenAPI struct {
	Paths      map[string]map[string]membershipOperation `json:"paths"`
	Components struct {
		Schemas       map[string]membershipSchema  `json:"schemas"`
		Responses     map[string]membershipContent `json:"responses"`
		RequestBodies map[string]membershipContent `json:"requestBodies"`
	} `json:"components"`
}

func (doc membershipOpenAPI) contentSchema(content membershipContent) membershipSchema {
	if content.Ref != "" {
		if strings.HasPrefix(content.Ref, "#/components/responses/") {
			content = doc.Components.Responses[strings.TrimPrefix(content.Ref, "#/components/responses/")]
		} else {
			content = doc.Components.RequestBodies[strings.TrimPrefix(content.Ref, "#/components/requestBodies/")]
		}
	}
	return doc.resolve(content.Content["application/json"].Schema)
}

func membershipOpenAPIDocument(t *testing.T) membershipOpenAPI {
	t.Helper()
	data, err := os.ReadFile("../gen/http/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc membershipOpenAPI
	if err = json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func (doc membershipOpenAPI) resolve(schema membershipSchema) membershipSchema {
	if schema.Ref != "" {
		return doc.resolve(doc.Components.Schemas[strings.TrimPrefix(schema.Ref, "#/components/schemas/")])
	}
	if len(schema.AllOf) == 1 && schema.Type == "" {
		return doc.resolve(schema.AllOf[0])
	}
	return schema
}

func TestMembershipResponseSchemasRetainStoredRoleDomain(t *testing.T) {
	doc := membershipOpenAPIDocument(t)
	for _, test := range []struct {
		path, method string
		lite         bool
	}{
		{"/api/v1/workspaces/{slug}/projects/{project_id}/members/{pk}/", "patch", false},
		{"/api/v1/workspaces/{slug}/projects/{project_id}/project-members/{pk}/", "patch", false},
		{"/api/workspaces/{slug}/projects/{project_id}/members/{pk}/", "patch", false},
		{"/api/v1/workspaces/{slug}/projects/{project_id}/project-members-lite/", "get", true},
	} {
		schema := doc.contentSchema(doc.Paths[test.path][test.method].Responses["200"])
		if test.lite {
			items := schema.Properties["results"].Items
			if items == nil {
				t.Fatalf("%s lacks page results", test.path)
			}
			schema = doc.resolve(*items)
		}
		role := doc.resolve(schema.Properties["role"])
		if role.Type != "integer" || len(role.Enum) != 0 || role.Minimum == nil || *role.Minimum != 0 || role.Maximum == nil || *role.Maximum != 32767 {
			t.Fatalf("%s response rejects persisted role 10 or lost stored smallint bounds: %#v", test.path, role)
		}
	}
}

func TestMembershipPatchResponseAllowsNullMemberWithoutWeakeningRequests(t *testing.T) {
	doc := membershipOpenAPIDocument(t)
	for _, alias := range []string{"members", "project-members"} {
		path := "/api/v1/workspaces/{slug}/projects/{project_id}/" + alias + "/{pk}/"
		op := doc.Paths[path]["patch"]
		response := doc.contentSchema(op.Responses["200"])
		member := doc.resolve(response.Properties["member"])
		want := []membershipSchema{{Type: "string", Format: "uuid"}, {Type: "null"}}
		if !slices.Contains(response.Required, "member") || !reflect.DeepEqual(member.AnyOf, want) {
			t.Fatalf("%s response member=%#v required=%v", path, member, response.Required)
		}
		for _, request := range []membershipSchema{
			doc.contentSchema(op.RequestBody),
			doc.contentSchema(doc.Paths[strings.ReplaceAll(path, "{pk}/", "")]["post"].RequestBody),
		} {
			member := doc.resolve(request.Properties["member"])
			role := doc.resolve(request.Properties["role"])
			if member.Type != "string" || member.Format != "uuid" || len(member.AnyOf) > 0 {
				t.Fatalf("%s request member weakened: %#v", path, member)
			}
			if !reflect.DeepEqual(role.Enum, []int{20, 15, 5}) {
				t.Fatalf("%s request role choices changed: %#v", path, role)
			}
		}
	}
}
