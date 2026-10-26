package property

import (
	"strings"

	"golang.org/x/net/html"
)

// The compatibility StripTags helper deliberately joins paragraph text. History
// needs readable boundaries so multiline edits remain distinguishable.
func readableValueText(value string) string {
	tokens := html.NewTokenizer(strings.NewReader(value))
	var out strings.Builder
	lineStart := true
	newline := func() {
		if !lineStart {
			out.WriteByte('\n')
			lineStart = true
		}
	}
	for {
		switch tokens.Next() {
		case html.TextToken:
			text := tokens.Text()
			out.Write(text)
			if len(text) > 0 {
				lineStart = text[len(text)-1] == '\n'
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			tag := tokens.Token()
			switch tag.Data {
			case "br":
				out.WriteByte('\n')
				lineStart = true
			case "hr", "p", "div", "li", "ul", "ol", "blockquote", "h1", "h2", "h3", "h4", "h5", "h6", "tr", "pre":
				newline()
			case "img", "image-component":
				label := "Image"
				for _, attr := range tag.Attr {
					if attr.Key == "alt" && strings.TrimSpace(attr.Val) != "" {
						label = "Image: " + attr.Val
					}
				}
				out.WriteString("[" + label + "]")
				lineStart = false
			}
		case html.EndTagToken:
			tag, _ := tokens.TagName()
			switch string(tag) {
			case "p", "div", "li", "ul", "ol", "blockquote", "h1", "h2", "h3", "h4", "h5", "h6", "tr", "pre":
				newline()
			case "td", "th":
				out.WriteByte('\t')
				lineStart = false
			}
		case html.ErrorToken:
			return strings.TrimSpace(out.String())
		}
	}
}
