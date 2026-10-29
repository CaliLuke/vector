package content

import (
	"errors"
	"strings"
	"testing"

	html "github.com/CaliLuke/go-plane/internal/htmlcompat"
)

func TestValidateHTMLMatchesDjangoCharacterization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "ordinary", input: "<p>Hello <b>x</b></p>", want: "<p>Hello <b>x</b></p>"},
		{name: "NH3 apostrophes in text and double-quoted attribute", input: `<p title="can't">it's safe</p>`, want: `<p title="can't">it's safe</p>`},
		{name: "NH3 quoted JSON representation", input: `{'type': 'doc'}`, want: `{'type': 'doc'}`},
		{name: "literal entity text stays escaped", input: `&amp;#39;`, want: `&amp;#39;`},
		{name: "apostrophe entity becomes source text", input: `&#39;`, want: `'`},
		{name: "double-escaped markup stays escaped", input: `&amp;lt;script&amp;gt;`, want: `&amp;lt;script&amp;gt;`},
		{name: "apostrophe cannot create another attribute", input: `<p title="a' onmouseover='bad">it's safe</p>`, want: `<p title="a' onmouseover='bad">it's safe</p>`},
		{name: "apostrophes in safe href and title", input: `<a href="https://example.test/a'b" title="can't">it's</a>`, want: `<a href="https://example.test/a'b" title="can't" rel="noopener noreferrer">it's</a>`},
		{name: "apostrophe entities in href title and text", input: `<a href="https://safe/?q=&#39;" title="&#39;">&#39;</a>`, want: `<a href="https://safe/?q='" title="'" rel="noopener noreferrer">'</a>`},
		{name: "encoded unsafe URI is removed before quote rendering", input: `<a href="jav&#x61;script:alert(1)" title="can't">bad</a>`, want: `<a title="can't" rel="noopener noreferrer">bad</a>`},
		{name: "unsafe URI with apostrophe entity is removed", input: `<a href="javascript:alert(&#39;x&#39;)">bad</a>`, want: `<a rel="noopener noreferrer">bad</a>`},
		{
			name: "fixture custom editor tags",
			input: `<mention-component id="u1" entity_identifier="user-1" entity_name="Ada">Ada</mention-component>` +
				`<image-component id="i1" src="https://example.com/a.png" aspectRatio="1.5"></image-component>` +
				`<label>Done <input type="checkbox" checked></label>`,
			want: `<mention-component id="u1" entity_identifier="user-1" entity_name="Ada">Ada</mention-component>` +
				`<image-component id="i1" src="https://example.com/a.png" aspectratio="1.5"></image-component>` +
				`<label>Done <input type="checkbox" checked=""></label>`,
		},
		{name: "script and body removed", input: "<script>alert(1)</script><p>x</p>", want: "<p>x</p>"},
		{name: "style and body removed", input: "<style>bad</style>x", want: "x"},
		{name: "iframe content preserved as escaped text", input: "<iframe>frame<b>x</b></iframe>tail", want: "frame&lt;b&gt;x&lt;/b&gt;tail"},
		{name: "object unwrapped", input: "<object>obj<b>x</b></object>tail", want: "obj<b>x</b>tail"},
		{name: "form unwrapped", input: "<form>form<input></form>", want: "form<input>"},
		{name: "foreign elements removed", input: "<svg><script>x</script><text>svg</text></svg>tail", want: "svgtail"},
		{name: "template and contents removed", input: "<template><p>x</p></template>tail", want: "tail"},
		{name: "noscript contents escaped", input: "<noscript><b>x</b></noscript>tail", want: "&lt;b&gt;x&lt;/b&gt;tail"},
		{name: "unknown tag unwrapped", input: "<foo>x<b>y</b>z</foo>", want: "x<b>y</b>z"},
		{
			name:  "custom mention tag and attributes",
			input: `<mention-component data-id="1" data-custom="x" onclick="bad" entity_identifier="u">M</mention-component>`,
			want:  `<mention-component data-id="1" entity_identifier="u">M</mention-component>`,
		},
		{
			name:  "custom image tag unsafe source",
			input: `<image-component src="javascript:alert(1)" data-extra="q" aspectRatio="2"></image-component>`,
			want:  `<image-component aspectratio="2"></image-component>`,
		},
		{name: "unsafe link", input: `<a href="javascript:bad" target="_blank">x</a>`, want: `<a target="_blank" rel="noopener noreferrer">x</a>`},
		{name: "unsafe link with leading space", input: `<a href=" javaScript:bad">x</a>`, want: `<a rel="noopener noreferrer">x</a>`},
		{name: "unsafe encoded link", input: `<a href="jav&#x61;script:bad">x</a>`, want: `<a rel="noopener noreferrer">x</a>`},
		{name: "FTP link", input: `<a href="ftp://host/x">x</a>`, want: `<a rel="noopener noreferrer">x</a>`},
		{name: "HTTPS link", input: `<a href="https://ok">x</a>`, want: `<a href="https://ok" rel="noopener noreferrer">x</a>`},
		{name: "mailto link", input: `<a href="mailto:user@example.com">x</a>`, want: `<a href="mailto:user@example.com" rel="noopener noreferrer">x</a>`},
		{name: "tel link", input: `<a href="tel:+12025550123">x</a>`, want: `<a href="tel:+12025550123" rel="noopener noreferrer">x</a>`},
		{name: "relative link", input: `<a href="/relative">x</a>`, want: `<a href="/relative" rel="noopener noreferrer">x</a>`},
		{name: "protocol-relative link", input: `<a href="//host/x">x</a>`, want: `<a href="//host/x" rel="noopener noreferrer">x</a>`},
		{name: "fragment link", input: `<a href="#frag">x</a>`, want: `<a href="#frag" rel="noopener noreferrer">x</a>`},
		{name: "unsafe image data URL", input: `<img src="data:image/png;base64,abc" alt="x">`, want: `<img alt="x">`},
		{name: "attributes and text escaped", input: `<p title="a&b">x &amp; y</p>`, want: `<p title="a&amp;b">x &amp; y</p>`},
		{name: "unclosed tags", input: "<p><b>unclosed", want: "<p><b>unclosed</b></p>"},
		{name: "malformed nesting", input: "<p><div>x</p>y", want: "<p></p><div>x<p></p>y</div>"},
		{name: "fixture malformed nesting", input: "<div><p>One<b>Two</div>Three &amp; Four</p>", want: "<div><p>One<b>Two</b></p></div><b>Three &amp; Four<p></p></b>"},
		{name: "broken markup escaped", input: "plain < broken > tail", want: "plain &lt; broken &gt; tail"},
		{name: "comment removed", input: "<!-- hi --><p>x</p>", want: "<p>x</p>"},
		{name: "input attributes", input: "<input checked disabled type=checkbox>", want: `<input checked="" type="checkbox">`},
		{name: "known global attributes", input: `<p data-id=x DATA-TIGHT=y data-foo=z style="position:fixed" onclick=x>x</p>`, want: `<p data-id="x" data-tight="y" style="position:fixed">x</p>`},
		{name: "table normalized", input: "<table><tr><td>x</td></tr></table>", want: "<table><tbody><tr><td>x</td></tr></tbody></table>"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, hasClean, err := ValidateHTML(test.input)
			if err != nil {
				t.Fatalf("ValidateHTML() error = %v", err)
			}
			if !hasClean {
				t.Fatal("ValidateHTML() hasClean = false, want true")
			}
			if got != test.want {
				t.Fatalf("ValidateHTML() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestValidateHTMLEmptyAndSizeBoundary(t *testing.T) {
	t.Parallel()

	clean, hasClean, err := ValidateHTML("")
	if err != nil || clean != "" || hasClean {
		t.Fatalf("ValidateHTML(empty) = (%q, %t, %v), want (empty, false, nil)", clean, hasClean, err)
	}

	atLimit := strings.Repeat("x", MaxContentBytes)
	clean, hasClean, err = ValidateHTML(atLimit)
	if err != nil || !hasClean || clean != atLimit {
		t.Fatalf("ValidateHTML(at limit) = (len %d, %t, %v)", len(clean), hasClean, err)
	}
	_, hasClean, err = ValidateHTML(atLimit + "x")
	if !errors.Is(err, ErrHTMLTooLarge) || hasClean {
		t.Fatalf("ValidateHTML(over limit) = (hasClean %t, %v), want false and %v", hasClean, err, ErrHTMLTooLarge)
	}
}

func TestValidateHTMLNestingBoundary(t *testing.T) {
	t.Parallel()

	boundary := strings.Repeat("<div>", MaxHTMLNestingDepth) + "deep" +
		strings.Repeat("</div>", MaxHTMLNestingDepth)
	clean, hasClean, err := ValidateHTML(boundary)
	if err != nil || !hasClean || clean != boundary {
		t.Fatalf("ValidateHTML(depth boundary) = (len %d, %t, %v)", len(clean), hasClean, err)
	}

	overBoundary := strings.Repeat("<div>", MaxHTMLNestingDepth+1) + "deep" +
		strings.Repeat("</div>", MaxHTMLNestingDepth+1)
	clean, hasClean, err = ValidateHTML(overBoundary)
	if clean != "" || hasClean || !errors.Is(err, ErrHTMLTooDeep) {
		t.Fatalf("ValidateHTML(over depth) = (%q, %t, %v), want safe depth error", clean, hasClean, err)
	}
}

func TestElementDepthCheckRejectsTreeBeforeRecursiveProcessing(t *testing.T) {
	t.Parallel()

	root := &html.Node{Type: html.ElementNode, Data: "div"}
	current := root
	for range MaxHTMLNestingDepth {
		child := &html.Node{Type: html.ElementNode, Data: "div"}
		current.AppendChild(child)
		current = child
	}
	if !exceedsElementDepth([]*html.Node{root}, MaxHTMLNestingDepth) {
		t.Fatal("depth check accepted a tree above the recursive-processing budget")
	}
}

func FuzzValidateHTMLIsIdempotent(f *testing.F) {
	f.Add("<p>Hello <script>bad</script></p>")
	f.Add(`<mention-component data-id="1">user</mention-component>`)
	f.Add("plain < broken > tail")
	f.Add("<A")
	f.Add(strings.Repeat("text", 16*1024+1))
	f.Fuzz(func(t *testing.T, input string) {
		clean, hasClean, err := ValidateHTML(input)
		if err != nil || !hasClean {
			return
		}
		second, secondHasClean, secondErr := ValidateHTML(clean)
		wantSecondClean := clean != ""
		if secondErr != nil || secondHasClean != wantSecondClean || second != clean {
			t.Fatalf("second validation = (%q, %t, %v), first = %q", second, secondHasClean, secondErr, clean)
		}
	})
}

func TestVoidElementSerializationPreservesAttributesAndIncompleteMarkup(t *testing.T) {
	for _, sample := range []struct {
		name  string
		input string
		want  string
	}{
		{"self-closing tag", `<br/>`, `<br>`},
		{"existing HTML tag", `<br>`, `<br>`},
		{"double-quoted delimiter", `<img alt="a/>b"/>`, `<img alt="a/>b">`},
		{"single-quoted delimiter", `<img alt='a/>b'/>`, `<img alt='a/>b'>`},
		{"unfinished tag", `<p>before</p><img alt="unfinished`, `<p>before</p><img alt="unfinished`},
	} {
		t.Run(sample.name, func(t *testing.T) {
			if got := string(removeVoidElementSlashes([]byte(sample.input))); got != sample.want {
				t.Fatalf("serialized HTML=%q, want %q", got, sample.want)
			}
		})
	}
}
