package notionsync

import (
	"context"
	"fmt"
	"html"
	"maps"
	"slices"
)

// DiscoverTeam maps a complete snapshot to one project for the selected team.
// Roadmaps become modules, their projects become work items, and tasks become
// sub-items. A shared task remains one work item without an arbitrary parent.
func DiscoverTeam(ctx context.Context, source Snapshot, teamID string) (Graph, error) {
	teamID, err := PageID(teamID)
	if err != nil {
		return Graph{}, err
	}
	team, err := source.Page(ctx, teamID)
	if err != nil {
		return Graph{}, err
	}
	if team.Archived {
		return Graph{}, fmt.Errorf("selected team is archived")
	}
	graph := Graph{Roadmap: team, Members: map[string][]string{}, Warnings: slices.Clone(team.Warnings)}
	projects, tasks := map[string]Page{}, map[string]Page{}
	owners := map[string][]string{}
	for _, id := range slices.Sorted(maps.Keys(source)) {
		roadmap := source[id]
		if roadmap.Archived || !slices.Contains(roadmap.Relations["👷 Team"], teamID) {
			continue
		}
		links, exists := roadmap.Relations["🏗️ Projects"]
		if !exists {
			return Graph{}, fmt.Errorf("roadmap %s has no 🏗️ Projects relation", id)
		}
		graph.Modules = append(graph.Modules, roadmap)
		graph.Warnings = append(graph.Warnings, roadmap.Warnings...)
		for _, projectID := range links {
			project, projectErr := source.Page(ctx, projectID)
			if projectErr != nil {
				return Graph{}, projectErr
			}
			if project.Archived {
				graph.Warnings = append(graph.Warnings, "Archived source project omitted: "+projectID)
				continue
			}
			taskLinks, ok := project.Relations["🧑‍💻 Tasks"]
			if !ok {
				return Graph{}, fmt.Errorf("project %s has no 🧑‍💻 Tasks relation", projectID)
			}
			project.Relations = maps.Clone(project.Relations)
			project.Relations["Parent"] = nil
			projects[projectID] = project
			graph.Members[projectID] = appendUnique(graph.Members[projectID], id)
			for _, taskID := range taskLinks {
				task, taskErr := source.Page(ctx, taskID)
				if taskErr != nil {
					return Graph{}, taskErr
				}
				tasks[taskID] = task
				owners[taskID] = appendUnique(owners[taskID], projectID)
				graph.Members[taskID] = appendUnique(graph.Members[taskID], id)
			}
		}
	}
	if len(graph.Modules) == 0 {
		return Graph{}, fmt.Errorf("snapshot contains no roadmaps assigned to the selected team")
	}
	for _, id := range slices.Sorted(maps.Keys(projects)) {
		graph.Tasks = append(graph.Tasks, projects[id])
		graph.Warnings = append(graph.Warnings, projects[id].Warnings...)
	}
	for _, id := range slices.Sorted(maps.Keys(tasks)) {
		if _, collision := projects[id]; collision {
			return Graph{}, fmt.Errorf("page %s is both a project and task", id)
		}
		task := tasks[id]
		task.Relations = maps.Clone(task.Relations)
		if task.Relations == nil {
			task.Relations = map[string][]string{}
		}
		task.Relations["Parent"] = nil
		if len(owners[id]) == 1 {
			task.Relations["Parent"] = slices.Clone(owners[id])
		} else {
			graph.Warnings = append(graph.Warnings, "Shared task has multiple source projects; retained once without a parent: "+id)
			task.HTML += "<p>Linked Notion projects:</p><ul>"
			for _, owner := range owners[id] {
				page := projects[owner]
				task.HTML += "<li><a href=\"" + html.EscapeString(page.URL) + "\">" + html.EscapeString(page.Title) + "</a></li>"
			}
			task.HTML += "</ul>"
		}
		graph.Tasks = append(graph.Tasks, task)
		graph.Warnings = append(graph.Warnings, task.Warnings...)
	}
	return graph, validateGraph(graph)
}

func appendUnique(values []string, value string) []string {
	if !slices.Contains(values, value) {
		return append(values, value)
	}
	return values
}
