package notionsync

import (
	"crypto/sha256"
	"fmt"
	"html"
	"strings"

	"github.com/CaliLuke/go-plane/internal/stickies/content"
)

func textContent(value string) string {
	value = strings.NewReplacer("</p>", "\n", "</li>", "\n", "</h1>", "\n", "</h2>", "\n", "</h3>", "\n", "<br>", "\n").Replace(value)
	return strings.TrimSpace(html.UnescapeString(content.StripTags(value)))
}

func identityFields(page Page) map[string]any {
	return map[string]any{"name": textContent(page.Title)}
}

func projectFields(page Page) map[string]any {
	fields := identityFields(page)
	hash := sha256.Sum256([]byte(page.ID))
	fields["identifier"] = fmt.Sprintf("N%X", hash[:5])
	fields["module_view"] = true
	fields["description"] = sourceDescription(page)
	return fields
}

func moduleFields(page Page) map[string]any {
	fields := identityFields(page)
	fields["description"] = sourceDescription(page)
	fields["start_date"] = nullable(page.Start)
	fields["target_date"] = nullable(page.End)
	switch strings.ToLower(page.Status) {
	case "in progress":
		fields["status"] = "in-progress"
	case "done", "completed":
		fields["status"] = "completed"
	case "paused", "blocked", "on hold":
		fields["status"] = "paused"
	case "canceled", "cancelled":
		fields["status"] = "cancelled"
	case "planned", "ready", "not started":
		fields["status"] = "planned"
	default:
		fields["status"] = "backlog"
	}
	return fields
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func sourceDescription(page Page) string {
	prefix := ""
	if page.Status != "" {
		prefix = "Notion status: " + page.Status + "\n\n"
	}
	return prefix + textContent(page.HTML) + "\n\n" + page.URL
}

func (r *Runner) taskFields(page Page, state string) (map[string]any, error) {
	fields := identityFields(page)
	clean, _, err := content.ValidateHTML(page.HTML)
	if err != nil {
		return nil, fmt.Errorf("task %s description: %w", page.ID, err)
	}
	fields["description_html"] = clean
	fields["state"] = state
	fields["start_date"] = nullable(page.Start)
	fields["target_date"] = nullable(page.End)
	priority := "none"
	switch {
	case strings.HasPrefix(page.Priority, "P0"):
		priority = "urgent"
	case strings.HasPrefix(page.Priority, "P1"):
		priority = "high"
	case strings.HasPrefix(page.Priority, "P2"):
		priority = "medium"
	case strings.HasPrefix(page.Priority, "P3"), strings.HasPrefix(page.Priority, "P4"):
		priority = "low"
	}
	fields["priority"] = priority
	people := []string{}
	for _, person := range page.People {
		mapped, ok := r.Users[person]
		if !ok {
			r.report.Warnings = append(r.report.Warnings, "Unmapped Notion user "+person+" on "+page.ID+"; assignees left unchanged")
			return fields, nil
		}
		people = append(people, mapped)
	}
	fields["assignees"] = people
	return fields, nil
}

func stateGroup(name string) string {
	switch strings.ToLower(name) {
	case "done", "completed":
		return "completed"
	case "canceled", "cancelled":
		return "cancelled"
	case "in progress", "in review", "qa", "design review", "blocked":
		return "started"
	case "not started", "ready", "planned", "todo", "to do":
		return "unstarted"
	default:
		return "backlog"
	}
}
