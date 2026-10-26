package notionsync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type notionBlock struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Children bool   `json:"has_children"`
}

func (n *Notion) blocks(ctx context.Context, id string, depth int, budget *int, warnings *[]string) (string, error) {
	if depth > 20 {
		return "", fmt.Errorf("Notion block nesting exceeds 20 levels")
	}
	var out strings.Builder
	cursor := ""
	seen := map[string]bool{}
	for {
		var result struct {
			Results []json.RawMessage `json:"results"`
			HasMore bool              `json:"has_more"`
			Next    string            `json:"next_cursor"`
		}
		path := "/blocks/" + id + "/children?page_size=100"
		if cursor != "" {
			path += "&start_cursor=" + url.QueryEscape(cursor)
		}
		if err := n.api.request(ctx, http.MethodGet, path, nil, &result); err != nil {
			return "", err
		}
		for _, raw := range result.Results {
			*budget--
			if *budget < 0 {
				return "", fmt.Errorf("Notion page exceeds the 5000 block import limit")
			}
			text, err := n.block(ctx, raw, depth, budget, warnings)
			if err != nil {
				return "", err
			}
			out.WriteString(text)
		}
		if !result.HasMore {
			return out.String(), nil
		}
		if result.Next == "" || seen[result.Next] {
			return "", fmt.Errorf("Notion block pagination did not advance")
		}
		seen[result.Next] = true
		cursor = result.Next
	}
}

func (n *Notion) block(ctx context.Context, raw json.RawMessage, depth int, budget *int, warnings *[]string) (string, error) {
	var block notionBlock
	if err := json.Unmarshal(raw, &block); err != nil {
		return "", err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return "", err
	}
	var value struct {
		Text    []richText   `json:"rich_text"`
		Checked bool         `json:"checked"`
		Cells   [][]richText `json:"cells"`
	}
	if err := json.Unmarshal(fields[block.Type], &value); err != nil {
		return "", err
	}
	text := renderRichText(value.Text)
	open, close := "<p>", "</p>"
	switch block.Type {
	case "paragraph", "callout", "toggle":
	case "heading_1":
		open, close = "<h1>", "</h1>"
	case "heading_2":
		open, close = "<h2>", "</h2>"
	case "heading_3":
		open, close = "<h3>", "</h3>"
	case "bulleted_list_item":
		open, close = "<ul><li>", "</li></ul>"
	case "numbered_list_item":
		open, close = "<ol><li>", "</li></ol>"
	case "quote":
		open, close = "<blockquote>", "</blockquote>"
	case "code":
		open, close = "<pre><code>", "</code></pre>"
	case "to_do":
		prefix := "☐ "
		if value.Checked {
			prefix = "☑ "
		}
		text = prefix + text
	case "divider":
		return "<hr>", nil
	case "table":
		open, close = "<table><tbody>", "</tbody></table>"
	case "table_row":
		open, close = "<tr>", "</tr>"
		text = ""
		for _, cell := range value.Cells {
			text += "<td>" + renderRichText(cell) + "</td>"
		}
	case "column_list", "column":
		open, close = "", ""
	default:
		*warnings = append(*warnings, "Unsupported Notion block "+block.Type+" on "+block.ID+"; retained at the source link")
		return "", nil
	}
	if block.Children {
		children, err := n.blocks(ctx, block.ID, depth+1, budget, warnings)
		if err != nil {
			return "", err
		}
		if block.Type == "table" || block.Type == "column" || block.Type == "column_list" {
			text += children
		} else {
			return open + text + close + children, nil
		}
	}
	return open + text + close, nil
}
