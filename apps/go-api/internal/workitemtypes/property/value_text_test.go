package property

import "testing"

func TestHistoricalValueTextPreservesReadableBoundaries(t *testing.T) {
	for _, test := range []struct {
		name  string
		kind  Kind
		value any
		want  string
	}{
		{"multiline", Text, "alpha\nbeta", "alpha\nbeta"},
		{"blank line", Text, "alpha\n\nbeta", "alpha\n\nbeta"},
		{"nested block", RichText, "alpha<div>beta</div>gamma", "alpha\nbeta\ngamma"},
		{"editor image", RichText, `<image-component src="https://example.test/a.png"></image-component>`, "[Image]"},
		{"paragraphs", RichText, "<p>alpha</p><p>beta &amp; gamma</p>", "alpha\nbeta & gamma"},
		{"inline", RichText, "<p>alpha <strong>beta</strong></p>", "alpha beta"},
		{"image", RichText, `<p><img src="https://example.test/image.png" alt="Diagram"></p>`, "[Image: Diagram]"},
		{"zero", Number, float64(0), "0"},
		{"false", Boolean, false, "false"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := ValueText(Definition{Title: "Field", Kind: test.kind}, Configuration{}, test.value, nil)
			if err != nil || got != test.want {
				t.Fatalf("value=%q, error=%v", got, err)
			}
		})
	}
}
