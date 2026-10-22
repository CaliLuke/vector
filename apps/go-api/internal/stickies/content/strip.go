package content

import (
	"strings"

	"golang.org/x/net/html"
)

// StripTags returns only data tokens. This matches Plane's MLStripper, including
// entity decoding and retention of script and style bodies.
func StripTags(input string) string {
	tokenizer := html.NewTokenizer(strings.NewReader(input))
	var output strings.Builder
	var rcdataTag string
	for {
		switch tokenizer.Next() {
		case html.TextToken:
			if rcdataTag == "textarea" || rcdataTag == "title" {
				output.WriteString(StripTags(string(tokenizer.Raw())))
			} else {
				output.Write(tokenizer.Text())
			}
		case html.StartTagToken:
			name, _ := tokenizer.TagName()
			tag := strings.ToLower(string(name))
			if tag == "textarea" || tag == "title" {
				rcdataTag = tag
			}
		case html.EndTagToken:
			name, _ := tokenizer.TagName()
			if strings.EqualFold(string(name), rcdataTag) {
				rcdataTag = ""
			}
		case html.ErrorToken:
			return output.String()
		}
	}
}
