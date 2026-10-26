// Package notionsync imports a selected Notion roadmap through Vector's public API.
package notionsync

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

type Page struct {
	ID        string              `json:"id"`
	Title     string              `json:"title"`
	URL       string              `json:"url"`
	HTML      string              `json:"html"`
	Status    string              `json:"status"`
	Priority  string              `json:"priority"`
	Start     string              `json:"start"`
	End       string              `json:"end"`
	People    []string            `json:"people"`
	Relations map[string][]string `json:"relations"`
	Archived  bool                `json:"archived"`
	EditedAt  string              `json:"edited_at"`
	Warnings  []string            `json:"warnings,omitempty"`
}

type Source interface {
	Page(context.Context, string) (Page, error)
}

type Graph struct {
	Roadmap  Page                `json:"roadmap"`
	Modules  []Page              `json:"modules"`
	Tasks    []Page              `json:"tasks"`
	Members  map[string][]string `json:"members"`
	Warnings []string            `json:"warnings"`
}

// PageID accepts a Notion URL or UUID, and never uses supplied hosts for requests.
func PageID(raw string) (string, error) {
	if parsed, err := url.Parse(raw); err == nil && parsed.Host != "" {
		raw = strings.TrimRight(parsed.Path, "/")
		raw = raw[strings.LastIndex(raw, "/")+1:]
		if len(raw) > 32 {
			if id, err := uuid.Parse(raw); err == nil {
				return id.String(), nil
			}
			raw = raw[len(raw)-32:]
		}
	}
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return "", fmt.Errorf("invalid Notion page ID")
	}
	return id.String(), nil
}

// Discover follows only the selected roadmap's projects and their task links.
// Ancestors are optional; following a parent never pulls in its other children.
func Discover(ctx context.Context, source Source, root string, includeParents, roadmapOnly bool) (Graph, error) {
	root, err := PageID(root)
	if err != nil {
		return Graph{}, err
	}
	roadmap, err := source.Page(ctx, root)
	if err != nil {
		return Graph{}, err
	}
	if roadmap.Archived {
		return Graph{}, fmt.Errorf("selected roadmap is archived")
	}
	graph := Graph{Roadmap: roadmap, Members: map[string][]string{}, Warnings: roadmap.Warnings}
	if roadmapOnly {
		return graph, nil
	}
	projects, ok := roadmap.Relations["🏗️ Projects"]
	if !ok {
		return Graph{}, fmt.Errorf("roadmap has no 🏗️ Projects relation; check the source schema and connection access")
	}
	seen := map[string]bool{}
	for _, id := range projects {
		module, fetchErr := source.Page(ctx, id)
		if fetchErr != nil {
			return Graph{}, fetchErr
		}
		if module.Archived {
			graph.Warnings = append(graph.Warnings, "Archived source project retained without changes: "+id)
			continue
		}
		tasks, exists := module.Relations["🧑‍💻 Tasks"]
		if !exists {
			return Graph{}, fmt.Errorf("project %s has no 🧑‍💻 Tasks relation", id)
		}
		graph.Modules = append(graph.Modules, module)
		graph.Warnings = append(graph.Warnings, module.Warnings...)
		for _, taskID := range tasks {
			graph.Members[taskID] = append(graph.Members[taskID], id)
			if seen[taskID] {
				continue
			}
			task, taskErr := source.Page(ctx, taskID)
			if taskErr != nil {
				return Graph{}, taskErr
			}
			seen[taskID] = true
			graph.Tasks = append(graph.Tasks, task)
		}
	}
	for index := 0; index < len(graph.Tasks); index++ {
		task := graph.Tasks[index]
		graph.Warnings = append(graph.Warnings, task.Warnings...)
		for _, parent := range task.Relations["Parent"] {
			if seen[parent] {
				continue
			}
			if !includeParents {
				graph.Warnings = append(graph.Warnings, "Parent outside selection: "+parent+" (child "+task.ID+")")
				continue
			}
			ancestor, parentErr := source.Page(ctx, parent)
			if parentErr != nil {
				return Graph{}, parentErr
			}
			seen[parent] = true
			graph.Tasks = append(graph.Tasks, ancestor)
		}
	}
	return graph, nil
}
