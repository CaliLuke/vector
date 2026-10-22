package workitems

import (
	"errors"
	"strings"

	html "github.com/CaliLuke/go-plane/internal/htmlcompat"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"golang.org/x/net/html/atom"
)

var errInvalidAPIHTML = errors.New("Invalid HTML passed")

// API writes select a fragment root before applying the shared nh3 policy.
// App writes apply that policy directly. Keep this distinction at the serializer
// boundary: wrapping after sanitization loses removed roots and table fragments.
func normalizeAPIHTML(value string) (string, error) {
	prepared, first, hasHead := prepareAPIHTML(value)
	if strings.TrimSpace(prepared) == "" {
		return "", errInvalidAPIHTML
	}
	var body *html.Node
	var selected *html.Node
	if context := orphanTableContext(first); context != "" {
		nodes, err := html.ParseFragmentWithOptions(strings.NewReader(prepared), &html.Node{
			Type: html.ElementNode, Data: context, DataAtom: atom.Lookup([]byte(context)),
		}, html.ParseOptionMaxElementDepth(content.MaxHTMLNestingDepth))
		if err != nil {
			return "", errInvalidAPIHTML
		}
		body = &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
		for _, node := range nodes {
			body.AppendChild(node)
		}
	} else {
		doc, err := html.ParseWithOptions(strings.NewReader(prepared), html.ParseOptionMaxElementDepth(content.MaxHTMLNestingDepth))
		if err != nil {
			return "", errInvalidAPIHTML
		}
		for node := range doc.Descendants() {
			if node.Type != html.ElementNode {
				continue
			}
			if node.Data == "head" && node.FirstChild != nil {
				hasHead = true
			}
			if node.Data == "body" {
				body = node
			}
		}
		if first == "html" || hasHead || body == nil {
			selected = doc
		}
	}
	if selected == nil {
		var only *html.Node
		count := 0
		nonblankText := false
		for node := body.FirstChild; node != nil; node = node.NextSibling {
			if node.Type == html.TextNode {
				nonblankText = nonblankText || strings.TrimSpace(node.Data) != ""
			} else {
				count++
				only = node
			}
		}
		if count == 1 && !nonblankText {
			selected = only
		} else {
			body.Data, body.DataAtom = "span", atom.Span
			for node := range body.Descendants() {
				if node.Type == html.ElementNode && apiBlockTag(node.Data) {
					body.Data, body.DataAtom = "div", atom.Div
					break
				}
			}
			selected = body
		}
	}
	var serialized strings.Builder
	if err := html.Render(&serialized, selected); err != nil {
		return "", errInvalidAPIHTML
	}
	// lxml tostring includes an element's tail. In the Go DOM that is the
	// following text sibling, which is not part of Render(element).
	if selected != body && selected.NextSibling != nil && selected.NextSibling.Type == html.TextNode {
		if err := html.Render(&serialized, selected.NextSibling); err != nil {
			return "", errInvalidAPIHTML
		}
	}
	clean, _, err := content.ValidateHTML(serialized.String())
	return clean, err
}

func apiBlockTag(name string) bool {
	// lxml.html.defs.block_tags, used by fromstring's synthetic-root selection.
	switch name {
	case "address", "blockquote", "caption", "center", "col", "colgroup", "dd", "del", "dir", "div", "dl", "dt", "fieldset", "form", "h1", "h2", "h3", "h4", "h5", "h6", "hr", "ins", "isindex", "legend", "li", "menu", "noscript", "ol", "optgroup", "option", "p", "pre", "table", "tbody", "td", "tfoot", "th", "thead", "tr", "ul":
		return true
	}
	return false
}

func orphanTableContext(first string) string {
	switch first {
	case "td", "th":
		return "tr"
	case "tr":
		return "tbody"
	case "thead", "tbody", "tfoot", "caption", "colgroup":
		return "table"
	case "col":
		return "colgroup"
	}
	return ""
}

func prepareAPIHTML(value string) (string, string, bool) {
	tokens := html.NewTokenizer(strings.NewReader(value))
	var result strings.Builder
	first := ""
	started, paragraph, hasHead := false, false, false
	for {
		kind := tokens.Next()
		raw := string(tokens.Raw())
		switch kind {
		case html.ErrorToken:
			return result.String(), first, hasHead
		case html.DoctypeToken:
			continue
		case html.CommentToken:
			if !started {
				continue
			}
		case html.TextToken:
			if !started {
				raw = strings.TrimLeft(raw, " \t\r\n\f")
				if raw == "" {
					continue
				}
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			name, _ := tokens.TagName()
			tag := string(name)
			if !started {
				first = tag
			}
			hasHead = hasHead || tag == "head"
			if tag == "p" {
				paragraph = true
			} else if apiBlockTag(tag) {
				paragraph = false
			}
		case html.EndTagToken:
			name, _ := tokens.TagName()
			if string(name) == "p" {
				// libxml discards an unmatched paragraph close; HTML5 would
				// insert an additional empty paragraph at this point.
				if !paragraph {
					continue
				}
				paragraph = false
			}
		}
		started = true
		result.WriteString(raw)
	}
}
