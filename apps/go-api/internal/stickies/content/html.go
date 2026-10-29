package content

import (
	"bytes"
	"errors"
	"net/url"
	"strings"

	html "github.com/CaliLuke/go-plane/internal/htmlcompat"
	"golang.org/x/net/html/atom"
)

var allowedTags = stringSet(
	"a", "abbr", "acronym", "area", "article", "aside", "b", "bdi", "bdo",
	"blockquote", "br", "caption", "center", "cite", "code", "col", "colgroup",
	"data", "dd", "del", "details", "dfn", "div", "dl", "dt", "em", "figcaption",
	"figure", "footer", "h1", "h2", "h3", "h4", "h5", "h6", "header", "hgroup",
	"hr", "i", "image-component", "img", "input", "ins", "kbd", "label", "li",
	"map", "mark", "mention-component", "nav", "ol", "p", "pre", "q", "rp", "rt",
	"rtc", "ruby", "s", "samp", "small", "span", "strike", "strong", "sub",
	"summary", "sup", "table", "tbody", "td", "th", "thead", "time", "tr", "tt",
	"u", "ul", "var", "wbr",
)

var globalAttributes = stringSet(
	"aria-hidden", "aria-label", "class", "data-background", "data-background-color",
	"data-block-type", "data-checked", "data-emoji-unicode", "data-emoji-url",
	"data-icon-color", "data-icon-name", "data-id", "data-logo-in-use", "data-name",
	"data-node-type", "data-text-color", "data-tight", "data-type", "id", "role",
	"start", "style", "title", "type", "xmlns",
)

var sourceTagAttributes = map[string][]string{
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

var tagAttributes = normalizedAttributeSets(sourceTagAttributes)

var safeURLSchemes = stringSet("http", "https", "mailto", "tel")
var removeWithContents = stringSet("script", "style", "template")

// ValidateHTML sanitizes HTML with Plane's nh3 0.2.18 policy. The bool is false
// only for empty input, matching the source's None clean-HTML result.
func ValidateHTML(input string) (clean string, hasClean bool, err error) {
	if input == "" {
		return "", false, nil
	}
	if len([]byte(input)) > MaxContentBytes {
		return "", false, ErrHTMLTooLarge
	}

	context := &html.Node{Type: html.ElementNode, DataAtom: atom.Body, Data: "body"}
	nodes, parseErr := html.ParseFragmentWithOptions(
		strings.NewReader(input),
		context,
		html.ParseOptionMaxElementDepth(MaxHTMLNestingDepth),
	)
	if parseErr != nil {
		if errors.Is(parseErr, html.ErrElementDepthLimit) {
			return "", false, ErrHTMLTooDeep
		}
		return "", false, ErrHTMLSanitizationFailed
	}
	if exceedsElementDepth(nodes, MaxHTMLNestingDepth) {
		return "", false, ErrHTMLTooDeep
	}

	root := &html.Node{Type: html.DocumentNode}
	for _, node := range nodes {
		root.AppendChild(node)
	}
	sanitizeChildren(root)

	var output bytes.Buffer
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		var rendered bytes.Buffer
		if renderErr := html.Render(&rendered, child); renderErr != nil {
			return "", false, ErrHTMLSanitizationFailed
		}
		output.Write(removeVoidElementSlashes(rendered.Bytes()))
	}
	// html5ever, used by nh3, escapes U+00A0 in text and attributes. Go's
	// renderer leaves that character literal; preserve the source wire value.
	// NH3 retains apostrophes. The renderer always double-quotes attributes,
	// so restoring this character cannot create an attribute delimiter.
	return strings.NewReplacer("\u00a0", "&nbsp;", "&#39;", "'").Replace(output.String()), true, nil
}

func exceedsElementDepth(roots []*html.Node, maxDepth int) bool {
	for _, root := range roots {
		current := root
		depth := 0
		for current != nil {
			if current.Type == html.ElementNode {
				depth++
				if depth > maxDepth {
					return true
				}
			}
			if current.FirstChild != nil {
				current = current.FirstChild
				continue
			}
			for current != nil {
				if current.Type == html.ElementNode {
					depth--
				}
				if current == root {
					current = nil
					break
				}
				if current.NextSibling != nil {
					current = current.NextSibling
					break
				}
				current = current.Parent
			}
		}
	}
	return false
}

var voidTags = stringSet("area", "br", "col", "hr", "img", "input", "wbr")

func removeVoidElementSlashes(rendered []byte) []byte {
	output := make([]byte, 0, len(rendered))
	for position := 0; position < len(rendered); {
		if rendered[position] != '<' || position+1 >= len(rendered) || rendered[position+1] == '/' {
			output = append(output, rendered[position])
			position++
			continue
		}
		nameEnd := position + 1
		for nameEnd < len(rendered) && isHTMLNameByte(rendered[nameEnd]) {
			nameEnd++
		}
		if _, isVoid := voidTags[string(rendered[position+1:nameEnd])]; !isVoid {
			output = append(output, rendered[position])
			position++
			continue
		}

		end := findTagEnd(rendered, nameEnd)
		if end < 0 {
			output = append(output, rendered[position:]...)
			break
		}
		output = append(output, rendered[position:end-1]...)
		if rendered[end-1] != '/' {
			output = append(output, rendered[end-1])
		}
		output = append(output, '>')
		position = end + 1
	}
	return output
}

func findTagEnd(rendered []byte, start int) int {
	var quote byte
	for position := start; position < len(rendered); position++ {
		b := rendered[position]
		if quote != 0 {
			if b == quote {
				quote = 0
			}
			continue
		}
		switch b {
		case '\'', '"':
			quote = b
		case '>':
			return position
		}
	}
	return -1
}

func isHTMLNameByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '-'
}

func sanitizeChildren(parent *html.Node) {
	for node := parent.FirstChild; node != nil; {
		next := node.NextSibling
		switch node.Type {
		case html.ElementNode:
			tag := strings.ToLower(node.Data)
			if _, remove := removeWithContents[tag]; remove {
				parent.RemoveChild(node)
			} else if _, allowed := allowedTags[tag]; !allowed {
				sanitizeChildren(node)
				for child := node.FirstChild; child != nil; {
					childNext := child.NextSibling
					node.RemoveChild(child)
					parent.InsertBefore(child, node)
					child = childNext
				}
				parent.RemoveChild(node)
			} else {
				node.Attr = sanitizeAttributes(tag, node.Attr)
				sanitizeChildren(node)
				if tag == "a" {
					node.Attr = append(node.Attr, html.Attribute{Key: "rel", Val: "noopener noreferrer"})
				}
			}
		case html.CommentNode, html.DoctypeNode:
			parent.RemoveChild(node)
		}
		node = next
	}
}

func sanitizeAttributes(tag string, attributes []html.Attribute) []html.Attribute {
	clean := make([]html.Attribute, 0, len(attributes))
	tagAllowed := tagAttributes[tag]
	for _, attribute := range attributes {
		name := strings.ToLower(attribute.Key)
		_, globallyAllowed := globalAttributes[name]
		_, locallyAllowed := tagAllowed[name]
		if !globallyAllowed && !locallyAllowed {
			continue
		}
		if (name == "href" || name == "src") && !safeURL(attribute.Val) {
			continue
		}
		attribute.Namespace = ""
		attribute.Key = name
		clean = append(clean, attribute)
	}
	return clean
}

func safeURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	if parsed.Scheme == "" {
		return true
	}
	_, allowed := safeURLSchemes[strings.ToLower(parsed.Scheme)]
	return allowed
}

func stringSet(values ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func normalizedAttributeSets(source map[string][]string) map[string]map[string]struct{} {
	sets := make(map[string]map[string]struct{}, len(source))
	for tag, attributes := range source {
		set := make(map[string]struct{}, len(attributes))
		for _, attribute := range attributes {
			set[strings.ToLower(attribute)] = struct{}{}
		}
		sets[tag] = set
	}
	return sets
}
