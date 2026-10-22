package main

import (
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

func TestRenderResultTypePreservesImportedJSONType(t *testing.T) {
	expression, err := parser.ParseExpr("*json.RawMessage")
	if err != nil {
		t.Fatal(err)
	}
	got, imports, err := renderResultType(expression, map[string]importedPackage{
		"json": {name: "json", path: "encoding/json/v2"},
	})
	if err != nil {
		t.Fatalf("renderResultType() error = %v", err)
	}
	if got != "*json.RawMessage" {
		t.Fatalf("renderResultType() = %q, want *json.RawMessage", got)
	}
	wantImports := []importedPackage{{name: "json", path: "encoding/json/v2"}}
	if !reflect.DeepEqual(imports, wantImports) {
		t.Fatalf("renderResultType() imports = %#v, want %#v", imports, wantImports)
	}
}

func TestRenderResultTypeQualifiesGeneratedAPIType(t *testing.T) {
	expression, err := parser.ParseExpr("*AssetResponse")
	if err != nil {
		t.Fatal(err)
	}
	got, imports, err := renderResultType(expression, nil)
	if err != nil {
		t.Fatalf("renderResultType() error = %v", err)
	}
	if got != "*api.AssetResponse" || len(imports) != 0 {
		t.Fatalf("renderResultType() = %q, imports %#v; want *api.AssetResponse and no extra imports", got, imports)
	}
}

func TestFindStatusTypesPreservesGeneratedErrorReceiverShape(t *testing.T) {
	source := `package generated
type ValueOperationStatus405 map[string]any
func (ValueOperationStatus405) Error() string { return "" }
type PointerOperationStatus400 struct{}
func (*PointerOperationStatus400) Error() string { return "" }
`
	file, err := parser.ParseFile(token.NewFileSet(), "generated.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := findStatusTypes(file, []string{"ValueOperation", "PointerOperation"})
	if got["ValueOperation"]["405"].pointerReceiver {
		t.Fatal("value-receiver error type was classified as a pointer")
	}
	if !got["PointerOperation"]["400"].pointerReceiver {
		t.Fatal("pointer-receiver error type was classified as a value")
	}
	mapper := renderErrorMapper(got)
	if !strings.Contains(mapper, "result := api.ValueOperationStatus405{}") || !strings.Contains(mapper, "result := &api.PointerOperationStatus400{}") {
		t.Fatalf("mapper did not preserve generated receiver forms:\n%s", mapper)
	}
}
