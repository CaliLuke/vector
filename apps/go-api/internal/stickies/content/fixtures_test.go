package content

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHTMLFixturesFromDjango(t *testing.T) {
	t.Parallel()

	fixtures := []string{
		"html-custom-editor-tags.json",
		"html-data-attributes.json",
		"html-unsafe-protocols.json",
		"html-script-elements.json",
		"html-malformed-markup.json",
	}
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fixture := readDjangoFixture(t, name)
			input := nestedString(t, fixture, "request", "body", "description_html")
			wantHTML := nestedString(t, fixture, "response", "body", "description_html")
			wantText := nestedString(t, fixture, "response", "body", "description_stripped")

			clean, hasClean, err := ValidateHTML(input)
			if err != nil || !hasClean {
				t.Fatalf("ValidateHTML() = (%q, %t, %v)", clean, hasClean, err)
			}
			if clean != wantHTML {
				t.Fatalf("ValidateHTML() = %q, fixture wants %q", clean, wantHTML)
			}
			if got := StripTags(clean); got != wantText {
				t.Fatalf("StripTags(clean) = %q, fixture wants %q", got, wantText)
			}
		})
	}
}

func TestStripTagsFixturesFromDjango(t *testing.T) {
	t.Parallel()

	fixtures := []string{
		"strip-tags-entity-references.json",
		"strip-tags-unclosed-tags.json",
		"strip-tags-script-bodies.json",
		"strip-tags-malformed-markup.json",
	}
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fixture := readDjangoFixture(t, name)
			input := nestedString(t, fixture, "source_call", "input", "html")
			want := nestedString(t, fixture, "result", "text")
			if got := StripTags(input); got != want {
				t.Fatalf("StripTags() = %q, fixture wants %q", got, want)
			}
		})
	}
}

func TestHTMLBoundaryFixturesFromDjango(t *testing.T) {
	t.Parallel()

	maximumFixture := readDjangoFixture(t, "html-maximum-size.json")
	maximumResult := nestedMap(t, maximumFixture, "result")
	if !directBool(t, maximumResult, "is_valid") {
		t.Fatal("maximum-size Django fixture is not valid")
	}
	clean, hasClean, err := ValidateHTML(strings.Repeat("a", MaxContentBytes))
	if err != nil || !hasClean || len(clean) != MaxContentBytes {
		t.Fatalf("ValidateHTML(maximum) = (len %d, %t, %v)", len(clean), hasClean, err)
	}

	oversizedFixture := readDjangoFixture(t, "html-validator-oversized.json")
	want := nestedString(t, oversizedFixture, "result", "error")
	_, _, err = ValidateHTML(strings.Repeat("a", MaxContentBytes+1))
	assertFixtureError(t, err, want)
}

func TestHTMLDeepNestingFixtureFromDjango(t *testing.T) {
	t.Parallel()

	fixture := readDjangoFixture(t, "html-deep-valid-nesting.json")
	input := nestedMap(t, fixture, "source_call", "input")
	results := nestedMap(t, fixture, "result")
	for _, depth := range intArray(t, input, "depths") {
		name := fmt.Sprint(depth)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			source := strings.Repeat("<dl>", depth) + "deep" + strings.Repeat("</dl>", depth)
			want := nestedMap(t, results, name)
			clean, hasClean, err := ValidateHTML(source)
			if err != nil {
				t.Fatalf("ValidateHTML(depth %d) error = %v", depth, err)
			}
			if !hasClean || !directBool(t, want, "is_valid") {
				t.Fatalf("ValidateHTML(depth %d) valid = %t, fixture wants true", depth, hasClean)
			}
			if !directBool(t, want, "clean_matches_input") || clean != source {
				t.Fatalf("ValidateHTML(depth %d) did not retain valid nesting", depth)
			}
			if got := strings.Count(clean, "<dl>"); got != directInt(t, want, "clean_tag_count") {
				t.Fatalf("ValidateHTML(depth %d) tag count = %d", depth, got)
			}
			if len(source) != directInt(t, want, "input_utf8_bytes") {
				t.Fatalf("ValidateHTML(depth %d) input bytes = %d", depth, len(source))
			}
		})
	}

	unsafeInnerHTML := directString(t, input, "unsafe_inner_html")
	source := strings.Repeat("<dl>", 512) + unsafeInnerHTML + strings.Repeat("</dl>", 512)
	want := nestedMap(t, results, "sanitization_at_512")
	clean, hasClean, err := ValidateHTML(source)
	if err != nil || !hasClean || !directBool(t, want, "is_valid") {
		t.Fatalf("ValidateHTML(deep sanitizer) = (valid %t, error %v)", hasClean, err)
	}
	if !directBool(t, want, "clean_matches_expected") {
		t.Fatal("Django deep-sanitizer fixture did not match its expected output")
	}
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(clean)))
	if digest != directString(t, want, "clean_sha256") {
		t.Fatalf("ValidateHTML(deep sanitizer) SHA-256 = %s", digest)
	}

	structuralCases := nestedMap(t, input, "structural_cases")
	for name, rawCase := range structuralCases {
		caseInput, ok := rawCase.(map[string]any)
		if !ok {
			t.Fatalf("fixture structural case %q is not an object", name)
		}
		source := strings.Repeat("<dl>", 512) + directString(t, caseInput, "html") + strings.Repeat("</dl>", 512)
		want := nestedMap(t, results, name)
		clean, hasClean, err := ValidateHTML(source)
		if err != nil || !hasClean || !directBool(t, want, "is_valid") {
			t.Fatalf("ValidateHTML(%s) = (valid %t, error %v)", name, hasClean, err)
		}
		if !directBool(t, want, "clean_matches_expected") {
			t.Fatalf("Django fixture %q did not match its expected output", name)
		}
		digest := fmt.Sprintf("%x", sha256.Sum256([]byte(clean)))
		if digest != directString(t, want, "clean_sha256") {
			t.Fatalf("ValidateHTML(%s) SHA-256 = %s", name, digest)
		}
	}

	deepContainerCases := nestedMap(t, input, "deep_container_cases")
	for name, rawCase := range deepContainerCases {
		caseInput, ok := rawCase.(map[string]any)
		if !ok {
			t.Fatalf("fixture deep-container case %q is not an object", name)
		}
		repeats := directInt(t, caseInput, "repeats")
		source := strings.Repeat(directString(t, caseInput, "opening_html"), repeats) + "deep" +
			strings.Repeat(directString(t, caseInput, "closing_html"), repeats)
		want := nestedMap(t, results, name)
		clean, hasClean, err := ValidateHTML(source)
		if err != nil || !hasClean || !directBool(t, want, "is_valid") {
			t.Fatalf("ValidateHTML(%s) = (valid %t, error %v)", name, hasClean, err)
		}
		if !directBool(t, want, "clean_matches_input") || clean != source {
			t.Fatalf("ValidateHTML(%s) did not retain valid nesting", name)
		}
		digest := fmt.Sprintf("%x", sha256.Sum256([]byte(clean)))
		if digest != directString(t, want, "clean_sha256") {
			t.Fatalf("ValidateHTML(%s) SHA-256 = %s", name, digest)
		}
	}

	availabilityDepths := intArray(t, input, "availability_depths")
	for _, depth := range availabilityDepths {
		name := fmt.Sprintf("source_availability_%d", depth)
		want := nestedMap(t, results, name)
		if !directBool(t, want, "is_valid") || !directBool(t, want, "clean_matches_input") {
			t.Fatalf("Django fixture %q did not accept its input", name)
		}
		source := strings.Repeat("<div>", depth) + "deep" + strings.Repeat("</div>", depth)
		clean, hasClean, err := ValidateHTML(source)
		if depth == MaxHTMLNestingDepth {
			if err != nil || !hasClean || clean != source {
				t.Fatalf("ValidateHTML(%s) = (len %d, valid %t, error %v)", name, len(clean), hasClean, err)
			}
			continue
		}
		if clean != "" || hasClean || !errors.Is(err, ErrHTMLTooDeep) {
			t.Fatalf("ValidateHTML(%s) = (%q, %t, %v), want safe depth error", name, clean, hasClean, err)
		}
	}
}

func TestBinaryFixturesFromDjango(t *testing.T) {
	t.Parallel()

	t.Run("base64 error", func(t *testing.T) {
		fixture := readDjangoFixture(t, "binary-base64-error.json")
		input := nestedString(t, fixture, "source_call", "input", "value")
		want := nestedString(t, fixture, "result", "error")
		assertFixtureError(t, ValidateBase64Binary(input), want)
	})
	t.Run("too short", func(t *testing.T) {
		fixture := readDjangoFixture(t, "binary-too-short.json")
		want := nestedString(t, fixture, "result", "error")
		input := base64.StdEncoding.EncodeToString([]byte("ABC"))
		assertFixtureError(t, ValidateBase64Binary(input), want)
	})
	t.Run("decoded oversized", func(t *testing.T) {
		fixture := readDjangoFixture(t, "binary-decoded-oversized.json")
		want := nestedString(t, fixture, "result", "error")
		input := base64.StdEncoding.EncodeToString(make([]byte, MaxContentBytes+1))
		assertFixtureError(t, ValidateBase64Binary(input), want)
	})
	t.Run("suspicious patterns", func(t *testing.T) {
		fixture := readDjangoFixture(t, "binary-suspicious-patterns.json")
		result := nestedMap(t, fixture, "result")
		rows, ok := result["results"].([]any)
		if !ok {
			t.Fatal("fixture result.results is not an array")
		}
		for _, value := range rows {
			row, ok := value.(map[string]any)
			if !ok {
				t.Fatal("fixture result row is not an object")
			}
			pattern := directString(t, row, "pattern")
			want := directString(t, row, "error")
			input := base64.StdEncoding.EncodeToString([]byte("prefix-" + pattern + "-suffix"))
			assertFixtureError(t, ValidateBase64Binary(input), want)
		}
	})
	t.Run("valid boundaries", func(t *testing.T) {
		fixture := readDjangoFixture(t, "binary-valid-boundaries.json")
		result := nestedMap(t, fixture, "result")
		inputs := map[string]string{
			"minimum_four_decoded_bytes":        base64.StdEncoding.EncodeToString([]byte("ABCD")),
			"maximum_ten_mib_decoded_bytes":     base64.StdEncoding.EncodeToString(make([]byte, MaxContentBytes)),
			"base64_trailing_non_alphabet_junk": base64.StdEncoding.EncodeToString([]byte("ABCD")) + " !!!",
		}
		for name, input := range inputs {
			if _, ok := result[name]; !ok {
				t.Fatalf("fixture omits result %q", name)
			}
			if err := ValidateBase64Binary(input); err != nil {
				t.Fatalf("ValidateBase64Binary(%s) error = %v", name, err)
			}
		}
	})
	t.Run("character scan boundary", func(t *testing.T) {
		fixture := readDjangoFixture(t, "binary-scan-character-boundary.json")
		result := nestedMap(t, fixture, "result")
		inside := nestedMap(t, result, "pattern_starts_at_byte_200_character_100")
		outside := nestedMap(t, result, "pattern_starts_after_character_200")
		insideInput := base64.StdEncoding.EncodeToString([]byte(repeatRune('é', 100) + "<script"))
		assertFixtureError(t, ValidateBase64Binary(insideInput), directString(t, inside, "error"))
		outsideInput := base64.StdEncoding.EncodeToString([]byte(repeatRune('é', 200) + "<script"))
		if err := ValidateBase64Binary(outsideInput); err != nil {
			t.Fatalf("ValidateBase64Binary(outside scan) error = %v", err)
		}
		if !directBool(t, outside, "is_valid") {
			t.Fatal("outside-scan Django fixture is not valid")
		}
	})
}

func assertFixtureError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("validation error = nil, fixture wants %q", want)
	}
	if err.Error() != want {
		t.Fatalf("validation error = %q, fixture wants %q", err, want)
	}
}

func readDjangoFixture(t *testing.T, name string) map[string]any {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}
	path := filepath.Join(filepath.Dir(currentFile), "../../parity/testdata/fixtures/stickies", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var fixture map[string]any
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("decode fixture %s: %v", name, err)
	}
	return fixture
}

func nestedString(t *testing.T, value map[string]any, path ...string) string {
	t.Helper()
	parent := value
	for _, key := range path[:len(path)-1] {
		parent = nestedMap(t, parent, key)
	}
	return directString(t, parent, path[len(path)-1])
}

func nestedMap(t *testing.T, value map[string]any, path ...string) map[string]any {
	t.Helper()
	current := value
	for _, key := range path {
		next, ok := current[key].(map[string]any)
		if !ok {
			t.Fatalf("fixture path %q is not an object", key)
		}
		current = next
	}
	return current
}

func directString(t *testing.T, value map[string]any, key string) string {
	t.Helper()
	text, ok := value[key].(string)
	if !ok {
		t.Fatalf("fixture path %q is not a string", key)
	}
	return text
}

func directBool(t *testing.T, value map[string]any, key string) bool {
	t.Helper()
	boolean, ok := value[key].(bool)
	if !ok {
		t.Fatalf("fixture path %q is not a boolean", key)
	}
	return boolean
}

func directInt(t *testing.T, value map[string]any, key string) int {
	t.Helper()
	number, ok := value[key].(float64)
	if !ok {
		t.Fatalf("fixture path %q is not a number", key)
	}
	return int(number)
}

func intArray(t *testing.T, value map[string]any, key string) []int {
	t.Helper()
	items, ok := value[key].([]any)
	if !ok {
		t.Fatalf("fixture path %q is not an array", key)
	}
	result := make([]int, len(items))
	for index, item := range items {
		number, ok := item.(float64)
		if !ok {
			t.Fatalf("fixture path %q item %d is not a number", key, index)
		}
		result[index] = int(number)
	}
	return result
}

func repeatRune(value rune, count int) string {
	runes := make([]rune, count)
	for i := range runes {
		runes[i] = value
	}
	return string(runes)
}
