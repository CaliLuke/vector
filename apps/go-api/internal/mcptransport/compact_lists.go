package mcptransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	generated "github.com/CaliLuke/go-plane/mcp/gen/mcp_vector_mcp"
)

type compactListInput struct {
	Action string
	Format string
	Fields string
	Expand string
}

type compactPage struct {
	Results         *[]compactListRow
	TotalCount      *int   `json:"total_count"`
	NextCursor      string `json:"next_cursor"`
	PrevCursor      string `json:"prev_cursor"`
	NextPageResults bool   `json:"next_page_results"`
	PrevPageResults bool   `json:"prev_page_results"`
}

type compactListRow struct {
	ID         string
	Name       string
	Slug       string
	Identifier string
	Group      string
	Priority   string
	ArchivedAt *string `json:"archived_at"`
	Draft      bool    `json:"is_draft"`
	Sequence   int64   `json:"sequence_id"`
	Project    json.RawMessage
	State      struct{ Name string }
}

// compactListContent uses one text block so clients receive each row once.
// Explicit work-item field and expansion selectors retain their JSON projection.
func compactListContent(ctx context.Context, _ generated.ToolCallInterceptorInfo, payload *generated.ToolsCallPayload, next generated.ToolCallHandler) (*generated.ToolsCallResult, error) {
	result, err := next(ctx, payload)
	if err != nil || !supportsCompactList(payload, result) {
		return result, err
	}
	var page compactPage
	if json.Unmarshal(result.StructuredContent, &page) != nil || page.Results == nil {
		return nil, errors.New("invalid compact list result")
	}
	text := formatCompactPage(payload.Name, page)
	result.Content = []*generated.ContentItem{{Type: "text", Text: &text}}
	result.StructuredContent = nil
	return result, nil
}

func supportsCompactList(payload *generated.ToolsCallPayload, result *generated.ToolsCallResult) bool {
	if payload == nil || result == nil || result.IsError != nil && *result.IsError {
		return false
	}
	switch payload.Name {
	case "workspace", "project", "state", "workitem":
		var input compactListInput
		return json.Unmarshal(payload.Arguments, &input) == nil && input.Action == "list" &&
			input.Format == "compact" && input.Fields == "" && input.Expand == ""
	}
	return false
}

func formatCompactPage(tool string, page compactPage) string {
	rows := *page.Results
	var output strings.Builder
	fmt.Fprint(&output, len(rows))
	if page.TotalCount != nil && *page.TotalCount != len(rows) {
		fmt.Fprintf(&output, " of %d", *page.TotalCount)
	}
	fmt.Fprintf(&output, " %s", map[string]string{
		"workspace": "workspaces", "project": "projects", "state": "states", "workitem": "work items",
	}[tool])
	for _, row := range rows {
		output.WriteByte('\n')
		output.WriteString(formatCompactRow(tool, row))
	}
	if page.NextPageResults {
		fmt.Fprintf(&output, "\nNext cursor %s", page.NextCursor)
	}
	if page.PrevPageResults {
		fmt.Fprintf(&output, "\nPrevious cursor %s", page.PrevCursor)
	}
	return output.String()
}

func formatCompactRow(tool string, row compactListRow) string {
	name := strings.Join(strings.Fields(row.Name), " ")
	switch tool {
	case "workspace":
		return row.Slug + "  " + name
	case "project":
		if row.ArchivedAt != nil {
			name += " [archived]"
		}
		return row.Identifier + "  " + name
	case "state":
		return name + "  " + row.Group + "  " + row.ID
	case "workitem":
		return formatCompactWorkitem(row, name)
	}
	return name
}

func formatCompactWorkitem(row compactListRow, name string) string {
	key := row.ID
	var project struct{ Identifier string }
	if !row.Draft && row.Sequence > 0 && json.Unmarshal(row.Project, &project) == nil && project.Identifier != "" {
		key = project.Identifier + "-" + strconv.FormatInt(row.Sequence, 10)
	}
	parts := []string{key, name}
	if row.State.Name != "" {
		parts = append(parts, strings.Join(strings.Fields(row.State.Name), " "))
	}
	if row.Priority != "" && row.Priority != "none" {
		parts = append(parts, row.Priority)
	}
	if row.Draft {
		parts = append(parts, "draft")
	}
	return strings.Join(parts, "  ")
}
