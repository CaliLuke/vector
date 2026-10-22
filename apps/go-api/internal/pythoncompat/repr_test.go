package pythoncompat

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestStringRetainsSourceJSONContainerRepresentationAndOrder(t *testing.T) {
	for _, vector := range []struct{ raw, want string }{
		{`{"type":"doc","content":[{"type":"paragraph"}]}`, `{'type': 'doc', 'content': [{'type': 'paragraph'}]}`},
		{`["<p>first</p>","<p>second</p>"]`, `['<p>first</p>', '<p>second</p>']`},
		{`[true,false,null,1.0]`, `[True, False, None, 1.0]`},
		{`["can't","a\"b","a'\"b","\u00a0","\n"]`, `["can't", 'a"b', 'a\'"b', '\xa0', '\n']`},
	} {
		var value any
		// UseNumber retains the serializer's integer/float distinction.
		decoder := json.NewDecoder(strings.NewReader(vector.raw))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			t.Fatal(err)
		}
		got, err := String(value, []byte(vector.raw))
		if err != nil || got != vector.want {
			t.Errorf("String(%s)=%s,%v want %s", vector.raw, got, err, vector.want)
		}
	}
}

func TestStringRendersOverflowingJSONNumbersAsPythonInfinity(t *testing.T) {
	const raw = `{"z":1e309,"a":[-1e999,9007199254740993.0]}`
	var value any
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}

	got, err := String(value, []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	const want = `{'z': inf, 'a': [-inf, 9007199254740992.0]}`
	if got != want {
		t.Fatalf("String(%s) = %q, want %q", raw, got, want)
	}
}

func TestStringRendersNonFiniteFloatValuesAsPythonNames(t *testing.T) {
	for _, tc := range []struct {
		value float64
		want  string
	}{
		{math.Inf(1), "inf"},
		{math.Inf(-1), "-inf"},
		{math.NaN(), "nan"},
	} {
		got, err := String(tc.value, nil)
		if err != nil || got != tc.want {
			t.Errorf("String(%v) = %q, %v; want %q", tc.value, got, err, tc.want)
		}
	}
}

func TestStringEscapesNonASCIISpacesInNestedContainers(t *testing.T) {
	const raw = `{"nested":["a\"b","\\","\n","\u00a0","\u2000","é","e\u0301"," "]}`
	var value any
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}

	got, err := String(value, []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	const want = `{'nested': ['a"b', '\\', '\n', '\xa0', '\u2000', 'é', 'é', ' ']}`
	if got != want {
		t.Fatalf("String(%s) = %q, want %q", raw, got, want)
	}
}

func TestStringUsesPythonUnicode15Printability(t *testing.T) {
	const raw = `["𜰀","é","e\u0301"," "]`
	var value any
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}

	got, err := String(value, []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	const want = `['\U0001cc00', 'é', 'é', ' ']`
	if got != want {
		t.Fatalf("String(%s) = %q, want %q", raw, got, want)
	}
}

func TestPythonPrintableMatchesPinnedUnicode15(t *testing.T) {
	hash := sha256.New()
	for character := rune(0); character <= 0x10ffff; character++ {
		value := byte(0)
		if pythonPrintable(character) {
			value = 1
		}
		_, _ = hash.Write([]byte{value})
	}

	const want = "a24a109bc8e347f9490466c59e4160d0dc9dd1006a5be9a4c91af71cd2e7fa99"
	if got := hex.EncodeToString(hash.Sum(nil)); got != want {
		t.Fatalf("Unicode 15 printability digest = %s, want %s", got, want)
	}
}

func TestStringPreservesSurrogateCodeUnitsInCompositeStrings(t *testing.T) {
	const raw = `["\ud800","\udfff","\ud83d\ude00","�","\\ud800"]`
	value := decodeJSONValue(t, raw)
	got, err := String(value, []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	const want = `['\ud800', '\udfff', '😀', '�', '\\ud800']`
	if got != want {
		t.Fatalf("String(%s) = %q, want %q", raw, got, want)
	}
}

func TestStringPreservesDistinctAndDuplicateSourceObjectKeys(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{`{"\ud800":"surrogate","\ufffd":"replacement","\ud83d\ude00":"old","😀":"new"}`, `{'\ud800': 'surrogate', '�': 'replacement', '😀': 'new'}`},
		{`{"a":1,"\u0061":2}`, `{'a': 2}`},
	} {
		value := decodeJSONValue(t, tc.raw)
		got, err := String(value, []byte(tc.raw))
		if err != nil || got != tc.want {
			t.Errorf("String(%s) = %q, %v; want %q", tc.raw, got, err, tc.want)
		}
	}
}

func decodeJSONValue(t *testing.T, raw string) any {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestTruthyPreservesEmptyJSONValues(t *testing.T) {
	for _, value := range []any{nil, false, "", json.Number("0.0"), []any{}, map[string]any{}} {
		if Truthy(value) {
			t.Errorf("empty %#v is truthy", value)
		}
	}
	for _, value := range []any{true, "false", json.Number("1"), []any{nil}, map[string]any{"x": nil}} {
		if !Truthy(value) {
			t.Errorf("nonempty %#v is false", value)
		}
	}
}
