package mcpactions

import (
	"html"
	"regexp"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
	htmlcompat "github.com/CaliLuke/go-plane/internal/htmlcompat"
)

type mentionSpan struct {
	start, end int
	user       string
}

// tokenizeMentions preserves source bytes outside user-mention tags. Empty
// closing pairs are consumed; nonempty contents keep their original closing tag,
// matching the pinned handler's HTMLParser spans rather than reserializing HTML.
func tokenizeMentions(source string) string {
	scanner := htmlcompat.NewTokenizer(strings.NewReader(source))
	spans := []mentionSpan{}
	offset := 0
	for {
		kind := scanner.Next()
		raw := string(scanner.Raw())
		start := offset
		offset += len(raw)
		if kind == htmlcompat.ErrorToken {
			break
		}
		token := scanner.Token()
		switch kind {
		case htmlcompat.StartTagToken, htmlcompat.SelfClosingTagToken:
			if token.Data != "script" && token.Data != "style" && token.Data != "title" && token.Data != "textarea" {
				scanner.NextIsNotRawText()
			}
			if token.Data != "mention-component" {
				continue
			}
			attrs := mentionAttributes(raw)
			user := ""
			if attrs["entity_name"] == "user_mention" {
				if id, err := djangoinput.UUID(strings.TrimSpace(attrs["entity_identifier"])); err == nil {
					user = id.String()
				}
			}
			spans = append(spans, mentionSpan{start: start, end: offset, user: user})
		case htmlcompat.EndTagToken:
			if token.Data == "mention-component" && len(spans) > 0 {
				last := &spans[len(spans)-1]
				if strings.TrimSpace(source[last.end:start]) == "" {
					last.end = offset
				}
			}
		}
	}
	var out strings.Builder
	cursor := 0
	for _, span := range spans {
		out.WriteString(source[cursor:span.start])
		if span.user != "" {
			out.WriteString("@[" + span.user + "]")
		} else {
			out.WriteString(source[span.start:span.end])
		}
		cursor = span.end
	}
	out.WriteString(source[cursor:])
	return out.String()
}

// HTMLParser preserves duplicate attributes and the pinned handler keeps the
// last value. The HTML5 tokenizer intentionally drops later duplicates.
var mentionAttributePattern = regexp.MustCompile(`\s*([^\s/>][^\s/=>]*)(\s*=\s*('[^']*'|"[^"]*"|[^\s>]*))?`)

func mentionAttributes(raw string) map[string]string {
	attrs := map[string]string{}
	raw = raw[len("<mention-component"):]
	for _, match := range mentionAttributePattern.FindAllStringSubmatch(raw, -1) {
		value := match[3]
		if len(value) >= 2 && (value[0] == '\'' || value[0] == '"') {
			value = value[1 : len(value)-1]
		}
		attrs[strings.ToLower(match[1])] = html.UnescapeString(value)
	}
	return attrs
}
