package property

import (
	"strings"
	"testing"
)

func TestPreservationHTMLRetainsParagraphBreaksAndEscapesText(t *testing.T) {
	html, err := PreservationHTML(Definition{Title: "Notes <original>", Kind: Text}, Configuration{}, "First & second\r\n<do not execute>\nLast", nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "<p><strong>Notes &lt;original&gt;</strong></p><p>First &amp; second<br>&lt;do not execute&gt;<br>Last</p>"
	if html != want {
		t.Fatalf("paragraph changed: %s", html)
	}
}
func TestPreservationHTMLUsesEffectiveSelectionLabels(t *testing.T) {
	config := Configuration{Options: []Option{{ID: "one", Name: "First & foremost"}, {ID: "two", Name: "Second"}}}
	html, err := PreservationHTML(Definition{Title: "Area", Kind: Cascading}, config, []any{"one", "two", "unknown"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "First &amp; foremost → Second → Unavailable selection") || strings.Contains(html, "unknown") {
		t.Fatalf("selection labels lost: %s", html)
	}
}

func TestPreservationHTMLKeepsValuesWithoutLeakingSelectionIdentifiers(t *testing.T) {
	for _, test := range []struct {
		name   string
		kind   Kind
		value  any
		labels map[string]string
		want   string
	}{
		{"zero", Number, float64(0), nil, "<p>0</p>"},
		{"false", Boolean, false, nil, "<p>false</p>"},
		{"member", Member, "private-id", map[string]string{"private-id": "Alex <Smith>"}, "<p>Alex &lt;Smith&gt;</p>"},
		{"unavailable release", Release, []string{"private-id"}, nil, "<p>Unavailable selection</p>"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := PreservationHTML(Definition{Title: "Field", Kind: test.kind}, Configuration{}, test.value, test.labels)
			if err != nil || got != "<p><strong>Field</strong></p>"+test.want {
				t.Fatalf("preserved value = %q, %v", got, err)
			}
		})
	}
	got, err := PreservationHTML(Definition{Kind: Number}, Configuration{}, nil, nil)
	if err != nil || got != "" {
		t.Fatalf("absent value created content: %q, %v", got, err)
	}
}

func TestPreservationHTMLSanitizesRichTextAndRejectsMalformedSavedValues(t *testing.T) {
	got, err := PreservationHTML(Definition{Title: "Notes", Kind: RichText}, Configuration{}, `<p onclick="alert(1)">Keep <a href="https://example.test">link</a></p><script>alert(2)</script>`, nil)
	if err != nil || !strings.Contains(got, "Keep") || !strings.Contains(got, "https://example.test") || strings.Contains(got, "onclick") || strings.Contains(got, "<script") {
		t.Fatalf("unsafe or lost rich text: %q, %v", got, err)
	}
	for _, test := range []struct {
		kind  Kind
		value any
	}{
		{RichText, float64(1)}, {Dropdown, []any{"valid", float64(2)}}, {Cascading, map[string]any{"id": "hidden"}},
	} {
		if _, err := PreservationHTML(Definition{Kind: test.kind}, Configuration{}, test.value, nil); err == nil {
			t.Fatalf("accepted malformed %s value %#v", test.kind, test.value)
		}
	}
}
