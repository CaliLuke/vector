package property_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	p "github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

const (
	propertyID = "00000000-0000-4000-8000-000000000001"
	projectA   = "00000000-0000-4000-8000-000000000002"
	projectB   = "00000000-0000-4000-8000-000000000003"
	typeA      = "00000000-0000-4000-8000-000000000004"
	typeB      = "00000000-0000-4000-8000-000000000005"
	contextA   = "00000000-0000-4000-8000-000000000006"
	contextB   = "00000000-0000-4000-8000-000000000007"
	optionA    = "00000000-0000-4000-8000-000000000008"
	optionB    = "00000000-0000-4000-8000-000000000009"
	optionC    = "00000000-0000-4000-8000-00000000000a"
)

func definition() p.Definition {
	return p.Definition{ID: propertyID, Name: "effort", Title: "Effort", Kind: p.Number, Configuration: p.Configuration{Active: true}}
}

func TestContextUsesOneMostSpecificOverrideAndBaseFallback(t *testing.T) {
	d := definition()
	d.Configuration.Default = float64(2)
	d.Contexts = []p.Context{
		{ID: contextA, Name: "Project", Projects: []string{projectA}, Overrides: []string{"default_value"}, Configuration: p.Configuration{Default: float64(5)}},
		{ID: contextB, Name: "Type in project", Projects: []string{projectA}, Types: []string{typeA}, Overrides: []string{"active"}, Configuration: p.Configuration{Active: false}},
	}
	if err := p.ValidateDefinition(d, nil, false); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		project, typeID, context string
		active                   bool
		value                    float64
	}{
		{projectA, typeA, contextB, false, 2}, {projectA, typeB, contextA, true, 5}, {projectB, typeA, "", true, 2},
	} {
		t.Run(test.project+test.typeID, func(t *testing.T) {
			got, id, err := p.Effective(d, test.project, test.typeID)
			if err != nil || id != test.context || got.Active != test.active || got.Default != test.value {
				t.Fatalf("effective=%+v context=%s err=%v", got, id, err)
			}
		})
	}
}

func TestContextRejectsAmbiguousScopesAndForbiddenOverrides(t *testing.T) {
	d := definition()
	d.Contexts = []p.Context{
		{ID: contextA, Name: "Project", Projects: []string{projectA}},
		{ID: contextB, Name: "Type", Types: []string{typeA}},
	}
	if err := p.ValidateDefinition(d, nil, false); err == nil {
		t.Fatal("accepted equal-specificity overlap")
	}
	d.Contexts = d.Contexts[:1]
	d.Contexts[0].Overrides = []string{"format"}
	if err := p.ValidateDefinition(d, nil, false); err == nil {
		t.Fatal("accepted unsupported format override")
	}
	d.Contexts[0].Overrides = []string{"multiple"}
	if err := p.ValidateDefinition(d, nil, false); err == nil {
		t.Fatal("accepted cardinality override")
	}
}

func TestRequiredContextClearsInheritedDefault(t *testing.T) {
	d := definition()
	d.Configuration.Default = float64(2)
	d.Contexts = []p.Context{{ID: contextA, Name: "Required", Projects: []string{projectA}, Overrides: []string{"required"}, Configuration: p.Configuration{Required: true}}}
	if err := p.ValidateDefinition(d, nil, false); err != nil {
		t.Fatal(err)
	}
	got, _, err := p.Effective(d, projectA, typeA)
	if err != nil || !got.Required || got.Default != nil {
		t.Fatalf("required effective config: %+v %v", got, err)
	}
}

func TestDefinitionProtectsIdentityAndUsedStructure(t *testing.T) {
	before := definition()
	after := before
	after.Title = "Renamed effort"
	if err := p.ValidateDefinition(after, &before, true); err != nil {
		t.Fatal(err)
	}
	after.Name = "renamed_effort"
	if err := p.ValidateDefinition(after, &before, true); err == nil {
		t.Fatal("internal reference changed on rename")
	}
	after = before
	after.Kind = p.Boolean
	if err := p.ValidateDefinition(after, &before, true); err == nil {
		t.Fatal("used kind changed")
	}
	after = before
	after.Title = "Due_Date"
	if err := p.ValidateDefinition(after, nil, false); err == nil {
		t.Fatal("reserved name accepted")
	}
}

func TestUnusedTextPropertyCanChangeKind(t *testing.T) {
	before := definition()
	before.Kind = p.Text
	before.Configuration = p.Configuration{Active: true, Format: "Single line"}
	after := before
	after.Kind = p.Number
	after.Configuration = p.Configuration{Active: true, Default: float64(0)}
	if err := p.ValidateDefinition(after, &before, false); err != nil {
		t.Fatalf("unused Text to Number conversion rejected: %v", err)
	}
}

func TestUsedTextPropertyCannotChangeKind(t *testing.T) {
	before := definition()
	before.Kind = p.Text
	before.Configuration = p.Configuration{Active: true, Format: "Single line"}
	after := before
	after.Kind = p.Number
	after.Configuration = p.Configuration{Active: true}
	if err := p.ValidateDefinition(after, &before, true); err == nil {
		t.Fatal("used Text property changed kind")
	}
}

func TestUnusedTextPropertyCannotChangeTextFormat(t *testing.T) {
	before := definition()
	before.Kind = p.Text
	before.Configuration = p.Configuration{Active: true, Format: "Single line"}
	after := before
	after.Configuration.Format = "Paragraph"
	if err := p.ValidateDefinition(after, &before, false); err == nil {
		t.Fatal("existing Text property changed format")
	}
}

func TestValuesEnforceKindsAndPreserveZeroAndFalse(t *testing.T) {
	for _, test := range []struct {
		kind        p.Kind
		config      p.Configuration
		input, want any
	}{
		{p.Text, p.Configuration{Format: "Paragraph"}, "hello\nworld", "hello\nworld"},
		{p.Number, p.Configuration{Required: true}, float64(0), float64(0)},
		{p.Boolean, p.Configuration{}, false, false},
		{p.Boolean, p.Configuration{}, nil, false},
		{p.Date, p.Configuration{}, "2024-02-29", "2024-02-29"},
		{p.URL, p.Configuration{}, "https://example.test/path", "https://example.test/path"},
		{p.RichText, p.Configuration{}, `<p>safe<script>alert(1)</script></p>`, `<p>safe</p>`},
		{p.RichText, p.Configuration{Required: true}, `<image-component src="https://example.test/pic.png"></image-component>`, `<image-component src="https://example.test/pic.png"></image-component>`},
		{p.Text, p.Configuration{Format: "Paragraph"}, "  ", nil},
		{p.Dropdown, p.Configuration{Options: []p.Option{{ID: optionA}}}, optionA, optionA},
		{p.Member, p.Configuration{Multiple: true}, []any{optionA}, []string{optionA}},
		{p.Release, p.Configuration{}, optionA, optionA},
	} {
		t.Run(string(test.kind), func(t *testing.T) {
			got, err := p.ValidateValue(test.kind, test.config, test.input, false, func(_ p.Kind, id string) bool { return id == optionA })
			if err != nil || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got %#v err=%v; want %#v", got, err, test.want)
			}
		})
	}
}

func TestInvalidValuesFailWithoutCoercion(t *testing.T) {
	for _, test := range []struct {
		kind   p.Kind
		config p.Configuration
		input  any
	}{
		{p.Number, p.Configuration{}, "12"}, {p.Boolean, p.Configuration{}, "false"},
		{p.Date, p.Configuration{}, "2025-02-29"}, {p.URL, p.Configuration{}, "javascript:alert(1)"},
		{p.URL, p.Configuration{}, "https://secret:password@example.test"},
		{p.Text, p.Configuration{Format: "Single line"}, "one\ntwo"},
		{p.Text, p.Configuration{Format: "Read-only"}, "new text"}, {p.Formula, p.Configuration{}, nil},
		{p.Member, p.Configuration{}, optionB}, {p.Release, p.Configuration{}, optionB},
		{p.Dropdown, p.Configuration{Options: []p.Option{{ID: optionA}}}, optionB},
		{p.Member, p.Configuration{Multiple: true}, []string{optionA, optionA}},
		{p.RichText, p.Configuration{Required: true}, "<script>unsafe</script>"},
		{p.RichText, p.Configuration{Required: true}, `<img alt="">`},
		{p.RichText, p.Configuration{Required: true}, `<image-component src="javascript:alert(1)"></image-component>`},
		{p.Text, p.Configuration{Required: true}, "\t "},
		{p.Text, p.Configuration{}, strings.Repeat("a", 65537)},
	} {
		t.Run(string(test.kind), func(t *testing.T) {
			_, err := p.ValidateValue(test.kind, test.config, test.input, false, func(_ p.Kind, id string) bool { return id == optionA })
			if err == nil {
				t.Fatalf("accepted invalid input %#v", test.input)
			}
		})
	}
}

func TestDefinitionRejectsUnreferenceableNames(t *testing.T) {
	d := definition()
	d.Name = "a}}b"
	if err := p.ValidateDefinition(d, nil, false); err == nil {
		t.Fatal("accepted an internal name that terminates its own formula reference")
	}
}

func TestCascadingLevelsRetainIdentityAndOrder(t *testing.T) {
	before := definition()
	before.Kind = p.Cascading
	before.Configuration = cascade()
	after := before
	after.Configuration.Levels = append([]p.Level(nil), before.Configuration.Levels...)
	after.Configuration.Levels[0], after.Configuration.Levels[1] = after.Configuration.Levels[1], after.Configuration.Levels[0]
	if err := p.ValidateDefinition(after, &before, false); err == nil {
		t.Fatal("accepted reordered level identities")
	}
	after.Configuration.Levels = append([]p.Level(nil), before.Configuration.Levels...)
	after.Configuration.Levels[1].Name = "Renamed region"
	if err := p.ValidateDefinition(after, &before, true); err != nil {
		t.Fatal(err)
	}
}

func TestContextScopeSizeIsBounded(t *testing.T) {
	d := definition()
	d.Contexts = []p.Context{{ID: contextA, Name: "Large scope", Projects: scopeIDs(1000)}}
	if err := p.ValidateDefinition(d, nil, false); err != nil {
		t.Fatal(err)
	}
	d.Contexts[0].Projects = append(d.Contexts[0].Projects, "00000000-0000-4000-8000-ffffffffffff")
	if err := p.ValidateDefinition(d, nil, false); err == nil {
		t.Fatal("accepted oversized scope")
	}
}

func scopeIDs(count int) []string {
	ids := make([]string, count)
	for i := range ids {
		ids[i] = fmt.Sprintf("00000000-0000-4000-8000-%012x", i)
	}
	return ids
}

func cascade() p.Configuration {
	return p.Configuration{Active: true, Levels: []p.Level{
		{ID: projectA, Name: "Country", Required: true}, {ID: projectB, Name: "Region", Required: true},
	}, Options: []p.Option{
		{ID: optionA, Name: "Has regions", Level: 0},
		{ID: optionB, Name: "Leaf country", Level: 0},
		{ID: optionC, Name: "Region", Level: 1, Parent: optionA},
	}}
}

func TestCascadingPathsRequireAncestorsAndPermitLeaves(t *testing.T) {
	c := cascade()
	for _, path := range [][]string{{optionB}, {optionA, optionC}} {
		if _, err := p.ValidateValue(p.Cascading, c, path, false, nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range [][]string{{}, {optionA}, {optionB, optionC}, {optionC}, {optionA, optionA}} {
		if _, err := p.ValidateValue(p.Cascading, c, path, false, nil); err == nil {
			t.Fatalf("accepted invalid path %v", path)
		}
	}
	if _, err := p.ValidateValue(p.Cascading, c, []string{optionB}, true, nil); err == nil {
		t.Fatal("accepted cascading draft value")
	}
}

func TestDraftAllowsMissingRequiredValuesWithoutSupplyingDefaults(t *testing.T) {
	got, err := p.ValidateValue(p.Number, p.Configuration{Active: true, Required: true}, nil, true, nil)
	if err != nil || got != nil {
		t.Fatalf("incomplete draft rejected: %v %v", got, err)
	}
	got, err = p.ValidateValue(p.Number, p.Configuration{Active: true, Default: float64(7)}, nil, false, nil)
	if err != nil || got != nil {
		t.Fatalf("clearing reapplied a default: %v %v", got, err)
	}
}

func TestCascadingDefinitionRejectsBrokenStructure(t *testing.T) {
	d := definition()
	d.Kind = p.Cascading
	d.Configuration = cascade()
	if err := p.ValidateDefinition(d, nil, false); err != nil {
		t.Fatal(err)
	}
	d.Configuration.Levels[0].Required = false
	if err := p.ValidateDefinition(d, nil, false); err == nil {
		t.Fatal("required descendant without required ancestor accepted")
	}
	d.Configuration = cascade()
	d.Configuration.Options[2].Parent = optionC
	if err := p.ValidateDefinition(d, nil, false); err == nil {
		t.Fatal("option cycle accepted")
	}
}

func TestContextUUIDCasingDoesNotChangeScope(t *testing.T) {
	d := definition()
	id := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	upper := strings.ToUpper(id)
	d.Contexts = []p.Context{{ID: contextA, Name: "Project", Projects: []string{upper}, Types: []string{upper}, Overrides: []string{"required"}, Configuration: p.Configuration{Required: true}}}
	for _, scope := range []string{id, upper} {
		got, context, err := p.Effective(d, scope, scope)
		if err != nil || !got.Required || context != contextA {
			t.Fatalf("casing changed scope: %+v %s %v", got, context, err)
		}
	}
	d.Contexts = append(d.Contexts, p.Context{ID: contextB, Name: "Overlap", Projects: []string{id}, Types: []string{id}})
	if err := p.ValidateDefinition(d, nil, false); err == nil {
		t.Fatal("casing hid overlapping contexts")
	}
	d.Contexts = d.Contexts[:1]
	d.Contexts[0].Projects = []string{id, upper}
	if err := p.ValidateDefinition(d, nil, false); err == nil {
		t.Fatal("casing hid duplicate scope IDs")
	}
}
