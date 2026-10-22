package content

import (
	"maps"
	"slices"
	"testing"
)

func TestHTMLPolicyMatchesPinnedPythonSource(t *testing.T) {
	t.Parallel()

	wantTags := []string{
		"a", "abbr", "acronym", "area", "article", "aside", "b", "bdi", "bdo",
		"blockquote", "br", "caption", "center", "cite", "code", "col", "colgroup",
		"data", "dd", "del", "details", "dfn", "div", "dl", "dt", "em", "figcaption",
		"figure", "footer", "h1", "h2", "h3", "h4", "h5", "h6", "header", "hgroup",
		"hr", "i", "image-component", "img", "input", "ins", "kbd", "label", "li",
		"map", "mark", "mention-component", "nav", "ol", "p", "pre", "q", "rp", "rt",
		"rtc", "ruby", "s", "samp", "small", "span", "strike", "strong", "sub",
		"summary", "sup", "table", "tbody", "td", "th", "thead", "time", "tr", "tt",
		"u", "ul", "var", "wbr",
	}
	wantGlobalAttributes := []string{
		"aria-hidden", "aria-label", "class", "data-background", "data-background-color",
		"data-block-type", "data-checked", "data-emoji-unicode", "data-emoji-url",
		"data-icon-color", "data-icon-name", "data-id", "data-logo-in-use", "data-name",
		"data-node-type", "data-text-color", "data-tight", "data-type", "id", "role",
		"start", "style", "title", "type", "xmlns",
	}
	wantTagAttributes := map[string][]string{
		"a":                 {"href", "target"},
		"code":              {"language", "spellcheck"},
		"image-component":   {"alignment", "aspectRatio", "aspectratio", "height", "id", "src", "status", "width"},
		"img":               {"alignment", "alt", "aspectRatio", "aspectratio", "height", "src", "title", "width"},
		"input":             {"checked", "type"},
		"mention-component": {"entity_identifier", "entity_name", "id"},
		"pre":               {"language"},
		"td":                {"background", "colspan", "colwidth", "rowspan", "style", "textColor", "textcolor"},
		"th":                {"background", "colspan", "colwidth", "rowspan", "style"},
		"tr":                {"background", "style", "textColor", "textcolor"},
	}
	wantSchemes := []string{"http", "https", "mailto", "tel"}

	assertStringSet(t, allowedTags, wantTags)
	assertStringSet(t, globalAttributes, wantGlobalAttributes)
	assertStringSet(t, safeURLSchemes, wantSchemes)
	if len(sourceTagAttributes) != len(wantTagAttributes) {
		t.Fatalf("tag attribute groups = %d, want %d", len(sourceTagAttributes), len(wantTagAttributes))
	}
	attributeCount := len(globalAttributes)
	for tag, want := range wantTagAttributes {
		got, ok := sourceTagAttributes[tag]
		if !ok {
			t.Fatalf("tag attribute group %q is missing", tag)
		}
		got = append([]string(nil), got...)
		want = append([]string(nil), want...)
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Fatalf("attributes for %q = %v, want %v", tag, got, want)
		}
		attributeCount += len(got)
	}
	if len(allowedTags) != 79 || attributeCount != 67 || len(safeURLSchemes) != 4 {
		t.Fatalf("policy counts = (%d tags, %d attributes, %d schemes), want (79, 67, 4)", len(allowedTags), attributeCount, len(safeURLSchemes))
	}
}

func TestHTMLPolicyMatchesDjangoFixture(t *testing.T) {
	t.Parallel()

	fixture := readDjangoFixture(t, "html-allowlists.json")
	result := nestedMap(t, fixture, "result")
	if got := directString(t, result, "nh3_version"); got != "0.2.18" {
		t.Fatalf("nh3 version = %q, want 0.2.18", got)
	}
	assertStringSet(t, allowedTags, stringArray(t, result, "allowed_tags"))
	assertStringSet(t, safeURLSchemes, stringArray(t, result, "safe_protocols"))

	attributes := nestedMap(t, result, "attributes")
	assertStringSet(t, globalAttributes, stringArray(t, attributes, "*"))
	if len(attributes)-1 != len(sourceTagAttributes) {
		t.Fatalf("fixture tag attribute groups = %d, implementation has %d", len(attributes)-1, len(sourceTagAttributes))
	}
	for tag, got := range sourceTagAttributes {
		want := stringArray(t, attributes, tag)
		got = append([]string(nil), got...)
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Fatalf("attributes for %q = %v, fixture wants %v", tag, got, want)
		}
	}
}

func assertStringSet(t *testing.T, got map[string]struct{}, want []string) {
	t.Helper()
	gotValues := slices.Sorted(maps.Keys(got))
	want = append([]string(nil), want...)
	slices.Sort(want)
	if !slices.Equal(gotValues, want) {
		t.Fatalf("set = %v, want %v", gotValues, want)
	}
}

func stringArray(t *testing.T, parent map[string]any, key string) []string {
	t.Helper()
	values, ok := parent[key].([]any)
	if !ok {
		t.Fatalf("fixture path %q is not an array", key)
	}
	result := make([]string, len(values))
	for i, value := range values {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("fixture path %q item %d is not a string", key, i)
		}
		result[i] = text
	}
	return result
}
