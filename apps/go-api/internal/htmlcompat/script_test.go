package html

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

func TestScriptTokenizerPreservesEscapedTextAcrossShortReads(t *testing.T) {
	for _, tc := range []struct {
		name, source, text string
		end                TokenType
	}{
		{"nested script", "<script><!--<script>nested</script>--></script><p>after</p>", "<!--<script>nested</script>-->", EndTagToken},
		{"return to script data", "<script><!--<script>--><b></script><p>after</p>", "<!--<script>--><b>", EndTagToken},
		{"nonmatching end tag", "<script><!--<script>text</other></script>--></script><p>after</p>", "<!--<script>text</other></script>-->", EndTagToken},
		{"unfinished double escape", "<script><!--<script>unterminated", "<!--<script>unterminated", ErrorToken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			z := NewTokenizer(iotest.OneByteReader(strings.NewReader(tc.source)))
			if token := z.Next(); token != StartTagToken || z.Token().Data != "script" {
				t.Fatal("missing script start tag")
			}
			if token := z.Next(); token != TextToken {
				t.Fatalf("script content type = %d, want text", token)
			}
			if text := string(z.Text()); text != tc.text {
				t.Fatalf("script text = %q, want %q", text, tc.text)
			}
			if token := z.Next(); token != tc.end {
				t.Fatalf("next token = %d, want %d", token, tc.end)
			}
			if tc.end == ErrorToken {
				if !errors.Is(z.Err(), io.EOF) {
					t.Fatalf("unfinished script error = %v, want EOF", z.Err())
				}
				return
			}
			if z.Token().Data != "script" {
				t.Fatal("script end tag was consumed as text")
			}
			if token := z.Next(); token != StartTagToken || z.Token().Data != "p" {
				t.Fatal("content following the script was lost")
			}
		})
	}
}
