package notionsync

import (
	"fmt"
	"unicode/utf8"
)

func validateGraph(graph Graph) error {
	pages := append([]Page{graph.Roadmap}, graph.Modules...)
	pages = append(pages, graph.Tasks...)
	for _, page := range pages {
		if _, err := PageID(page.ID); err != nil {
			return err
		}
		name := textContent(page.Title)
		if name == "" || utf8.RuneCountInString(name) > 255 {
			return fmt.Errorf("Notion page %s needs a title of 1–255 characters", page.ID)
		}
	}
	parents := map[string][]string{}
	for _, task := range graph.Tasks {
		if !task.Archived && task.Status == "" {
			return fmt.Errorf("task %s is missing a status", task.ID)
		}
		parents[task.ID] = task.Relations["Parent"]
	}
	visiting, done := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(id string) error {
		if visiting[id] {
			return fmt.Errorf("Notion task parent cycle at %s", id)
		}
		if done[id] {
			return nil
		}
		visiting[id] = true
		for _, parent := range parents[id] {
			if err := visit(parent); err != nil {
				return err
			}
		}
		visiting[id] = false
		done[id] = true
		return nil
	}
	for id := range parents {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}
