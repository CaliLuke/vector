package formula_test

import (
	"math"
	"strings"
	"testing"
	"time"

	f "github.com/CaliLuke/go-plane/internal/workitemtypes/formula"
)

var instant = time.Date(2026, 10, 25, 0, 30, 0, 0, time.UTC)

func evaluate(t *testing.T, source string, kind f.Kind, fields map[string]f.Field, values map[string]f.Value) f.Value {
	t.Helper()
	expression, err := f.Compile(source, kind, fields)
	if err != nil {
		t.Fatal(err)
	}
	result, err := expression.Evaluate(values, instant)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestNumericLanguage(t *testing.T) {
	for _, test := range []struct {
		expression string
		want       float64
	}{
		{"1+2*3", 7}, {"(1+2)*3", 9}, {"-2 * -3", 6}, {"+4 / 2", 2},
		{"8 - 3 - 1", 4}, {"8/2/2", 2}, {"ABS(-3.25)", 3.25},
		{"round(1.25,1)", 1.3}, {"ROUND(-1.25,1)", -1.3}, {"ROUND(125,-1)", 130}, {"ROUND(1.5)", 2},
		{"ROUND(1.005,2)", 1.01}, {"ROUND(-1.005,2)", -1.01},
		{"IF(FALSE, 1/0, 7)", 7}, {"IF(TRUE, 9, 1/0)", 9},
		{`LEN("a🧑é")`, 3}, {"IF(1+2*3=7, 11, 12)", 11},
	} {
		t.Run(test.expression, func(t *testing.T) {
			got := evaluate(t, test.expression, f.Number, nil, nil)
			if got.Empty || got.Number != test.want {
				t.Fatalf("got %+v; want %v", got, test.want)
			}
		})
	}
}

func TestTextAndBooleanLanguage(t *testing.T) {
	for _, test := range []struct{ expression, want string }{
		{`UPPER("café")`, "CAFÉ"}, {`LOWER("ÄBC")`, "äbc"},
		{`"n=" & 2+3`, "n=5"}, {`CONCAT("a",2,TRUE)`, "a2true"},
		{`"quote: \" slash: \\"`, `quote: " slash: \`},
		{`"today " & TODAY()`, "today 2026-10-25"},
	} {
		t.Run(test.expression, func(t *testing.T) {
			got := evaluate(t, test.expression, f.Text, nil, nil)
			if got.Empty || got.Text != test.want {
				t.Fatalf("got %+v; want %q", got, test.want)
			}
		})
	}
	for _, source := range []string{`1=1`, `1!=2`, `1<2`, `1<=1`, `2>1`, `2>=2`, `"a"<"b"`, `TRUE!=FALSE`, `TODAY()<NOW()`} {
		t.Run(source, func(t *testing.T) {
			if got := evaluate(t, source, f.Boolean, nil, nil); got.Empty || !got.Boolean {
				t.Fatalf("want true: %+v", got)
			}
		})
	}
}

func TestFieldsUseStableIdentityAndPreserveFalsyValues(t *testing.T) {
	fields := map[string]f.Field{"Story Points": {ID: "points-id", Kind: f.Number}, "Flag": {ID: "flag-id", Kind: f.Boolean}}
	values := map[string]f.Value{"points-id": {Kind: f.Number, Number: 0}, "flag-id": {Kind: f.Boolean, Boolean: false}}
	got := evaluate(t, `IF({{FLAG}}, 8, {{STORY_POINTS}})`, f.Number, fields, values)
	if got.Empty || got.Number != 0 {
		t.Fatalf("zero/false lost: %+v", got)
	}
	got = evaluate(t, `IF(TRUE, 5, {{story points}})`, f.Number, fields, nil)
	if got.Empty || got.Number != 5 {
		t.Fatalf("untaken empty branch evaluated: %+v", got)
	}
}

func TestEmptyPropagationIsIndependentOfOperandOrder(t *testing.T) {
	fields := map[string]f.Field{"missing": {ID: "id", Kind: f.Number}, "condition": {ID: "bool", Kind: f.Boolean}}
	for _, source := range []string{`{{missing}}+1`, `1+{{missing}}`, `IF({{condition}},1,2)`} {
		t.Run(source, func(t *testing.T) {
			if got := evaluate(t, source, f.Number, fields, nil); !got.Empty {
				t.Fatalf("want empty, got %+v", got)
			}
		})
	}
}

func TestMissingValuesDoNotHideEvaluatedErrors(t *testing.T) {
	fields := map[string]f.Field{"missing": {ID: "id", Kind: f.Number}}
	for _, source := range []string{`1/0+{{missing}}`, `{{missing}}+1/0`} {
		t.Run(source, func(t *testing.T) {
			expression, err := f.Compile(source, f.Number, fields)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = expression.Evaluate(nil, instant); err == nil {
				t.Fatal("missing input hid a runtime error")
			}
		})
	}
}

func TestDatesUseOneUTCInstantAndCalendarArithmetic(t *testing.T) {
	got := evaluate(t, `TODAY()+1`, f.Date, nil, nil)
	if !got.DateOnly || got.Date.Format(time.DateOnly) != "2026-10-26" {
		t.Fatalf("date result: %+v", got)
	}
	got = evaluate(t, `NOW()-0.5`, f.Date, nil, nil)
	if got.DateOnly || !got.Date.Equal(time.Date(2026, 10, 24, 12, 30, 0, 0, time.UTC)) {
		t.Fatalf("timestamp result: %+v", got)
	}
	got = evaluate(t, `(TODAY()+30)-TODAY()`, f.Number, nil, nil)
	if got.Number != 30 {
		t.Fatalf("calendar subtraction: %+v", got)
	}
	expression, err := f.Compile(`TODAY()`, f.Date, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err = expression.Evaluate(nil, time.Date(2026, 10, 24, 18, 30, 0, 0, time.FixedZone("local", -7*3600)))
	if err != nil || got.Date.Format(time.DateOnly) != "2026-10-25" {
		t.Fatalf("UTC boundary: %+v %v", got, err)
	}
}

func TestInvalidExpressionsAreRejectedBeforeEvaluation(t *testing.T) {
	for _, source := range []string{"", "{{unknown}}", "{{unclosed", "1+", "1 2", "1<2<3", "IF(TRUE,1,\"text\")", "IF(1,2,3)", "ROUND()", "ABS(1,2)", "NOW(1)", "UPPER(1)", "CONCAT()", "execute(1)", "1/\"2\"", "TODAY()*2", "TRUE<FALSE", "\"unclosed", "..", "1 == 2", strings.Repeat("(", 34) + "1" + strings.Repeat(")", 34), strings.Repeat("1+", 2050) + "1"} {
		t.Run(source[:min(len(source), 60)], func(t *testing.T) {
			if _, err := f.Compile(source, f.Number, nil); err == nil {
				t.Fatalf("accepted %q", source)
			}
		})
	}
	if _, err := f.Compile("1", f.Text, nil); err == nil {
		t.Fatal("accepted wrong declared result")
	}
	if _, err := f.Compile("1", f.Number, map[string]f.Field{"a b": {ID: "one", Kind: f.Number}, "A_B": {ID: "two", Kind: f.Number}}); err == nil {
		t.Fatal("accepted ambiguous field catalog")
	}
}

func TestRuntimeErrorsAreNotSuccessfulEmptyValues(t *testing.T) {
	for _, test := range []struct {
		source string
		kind   f.Kind
	}{
		{"1/0", f.Number}, {"ROUND(1,1.5)", f.Number}, {"ROUND(1,16)", f.Number}, {"TODAY()+0.5", f.Date}, {"TODAY()+4000000", f.Date},
	} {
		t.Run(test.source, func(t *testing.T) {
			expression, err := f.Compile(test.source, test.kind, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = expression.Evaluate(nil, instant); err == nil {
				t.Fatal("runtime failure was accepted")
			}
		})
	}
	fields := map[string]f.Field{"n": {ID: "n", Kind: f.Number}}
	expression, err := f.Compile(`{{n}}*2`, f.Number, fields)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []f.Value{{Kind: f.Text, Text: "2"}, {Kind: f.Number, Number: math.Inf(1)}, {Kind: f.Number, Number: math.MaxFloat64}} {
		if _, err = expression.Evaluate(map[string]f.Value{"n": value}, instant); err == nil {
			t.Fatalf("accepted invalid input/result %+v", value)
		}
	}
}

func TestOutputExpansionIsBounded(t *testing.T) {
	expression, err := f.Compile(`{{a}} & {{a}}`, f.Text, map[string]f.Field{"a": {ID: "a", Kind: f.Text}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = expression.Evaluate(map[string]f.Value{"a": {Kind: f.Text, Text: strings.Repeat("a", 40000)}}, instant); err == nil {
		t.Fatal("accepted oversized output")
	}
}
