// Package commentmentions renders MCP mention tokens without rewriting other HTML.
package commentmentions

import (
	"errors"
	"html"
	"regexp"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
	htmlcompat "github.com/CaliLuke/go-plane/internal/htmlcompat"
	"github.com/google/uuid"
)

type span struct {
	start, end int
	text       bool
	user       string
}
type Document struct {
	source string
	spans  []span
	Users  []uuid.UUID
}

var token = regexp.MustCompile(`@\[[\s\p{Z}\x{0085}\x{001c}-\x{001f}]*([0-9a-fA-F]{8}-(?:[0-9a-fA-F]{4}-){3}[0-9a-fA-F]{12})[\s\p{Z}\x{0085}\x{001c}-\x{001f}]*\]`)
var attribute = regexp.MustCompile(`\s*([^\s/>][^\s/=>]*)(\s*=\s*('[^']*'|"[^"]*"|[^\s>]*))?`)

func Parse(source string) (Document, error) {
	doc := Document{source: source}
	scanner := htmlcompat.NewTokenizer(strings.NewReader(source))
	offset, lastElement := 0, -1
	for {
		kind := scanner.Next()
		raw := string(scanner.Raw())
		start := offset
		offset += len(raw)
		if kind == htmlcompat.ErrorToken {
			break
		}
		item := scanner.Token()
		switch kind {
		case htmlcompat.TextToken:
			doc.spans = append(doc.spans, span{start: start, end: offset, text: true})
		case htmlcompat.StartTagToken, htmlcompat.SelfClosingTagToken:
			if item.Data != "script" && item.Data != "style" && item.Data != "title" && item.Data != "textarea" {
				scanner.NextIsNotRawText()
			}
			if item.Data != "mention-component" {
				continue
			}
			attrs := map[string]string{}
			for _, match := range attribute.FindAllStringSubmatch(raw[len("<mention-component"):], -1) {
				value := match[3]
				if len(value) >= 2 && (value[0] == '\'' || value[0] == '"') {
					value = value[1 : len(value)-1]
				}
				attrs[strings.ToLower(match[1])] = html.UnescapeString(value)
			}
			if attrs["entity_name"] == "" {
				return Document{}, errors.New("mention is missing entity_name")
			}
			user := ""
			if attrs["entity_name"] == "user_mention" {
				if id, err := djangoinput.UUID(strings.TrimSpace(attrs["entity_identifier"])); err == nil {
					user = id.String()
				}
			}
			doc.spans = append(doc.spans, span{start: start, end: offset, user: user})
			lastElement = len(doc.spans) - 1
		case htmlcompat.EndTagToken:
			if item.Data == "mention-component" && lastElement >= 0 && strings.TrimSpace(source[doc.spans[lastElement].end:start]) == "" {
				doc.spans[lastElement].end = offset
				doc.spans = doc.spans[:lastElement+1]
			}
		}
	}
	seen := map[uuid.UUID]bool{}
	add := func(value string) {
		id, err := uuid.Parse(value)
		if err == nil && !seen[id] {
			seen[id] = true
			doc.Users = append(doc.Users, id)
		}
	}
	for _, part := range doc.spans {
		if part.text {
			for _, match := range token.FindAllStringSubmatch(source[part.start:part.end], -1) {
				add(match[1])
			}
		} else if part.user != "" {
			add(part.user)
		}
	}
	return doc, nil
}

// Render allocates chip identities only after a receipt replay has been ruled out.
func (d Document) Render() string {
	element := func(user string) string {
		return `<mention-component id="` + uuid.NewString() + `" entity_identifier="` + user + `" entity_name="user_mention"></mention-component>`
	}
	var out strings.Builder
	cursor := 0
	for _, part := range d.spans {
		out.WriteString(d.source[cursor:part.start])
		raw := d.source[part.start:part.end]
		if part.text {
			out.WriteString(token.ReplaceAllStringFunc(raw, func(value string) string { return element(strings.ToLower(token.FindStringSubmatch(value)[1])) }))
		} else if part.user != "" {
			out.WriteString(element(part.user))
		} else {
			out.WriteString(raw)
		}
		cursor = part.end
	}
	out.WriteString(d.source[cursor:])
	return out.String()
}
