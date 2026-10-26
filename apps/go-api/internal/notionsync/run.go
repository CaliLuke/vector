package notionsync

import (
	"context"
	"fmt"
	"net/http"
	"slices"
)

func (r *Runner) Run(ctx context.Context, graph Graph) (Report, error) {
	r.report = Report{Warnings: slices.Clone(graph.Warnings)}
	if err := validateGraph(graph); err != nil {
		return r.report, err
	}
	project, err := r.upsert(ctx, "project", "", graph.Roadmap.ID, projectFields(graph.Roadmap))
	r.report.ProjectID = project
	if err != nil {
		return r.report, err
	}
	states, err := r.states(ctx, project, graph.Tasks)
	if err != nil {
		return r.report, err
	}
	modules := map[string]string{}
	for _, module := range graph.Modules {
		id, writeErr := r.upsert(ctx, "modules", project, module.ID, moduleFields(module))
		if writeErr != nil {
			return r.report, writeErr
		}
		modules[module.ID] = id
	}
	tasks := map[string]string{}
	for _, task := range graph.Tasks {
		if task.Archived {
			r.report.Warnings = append(r.report.Warnings, "Archived source task retained without changes: "+task.ID)
			continue
		}
		fields, fieldErr := r.taskFields(task, states[task.Status])
		if fieldErr != nil {
			return r.report, fieldErr
		}
		id, writeErr := r.upsert(ctx, "issues", project, task.ID, fields)
		if writeErr != nil {
			return r.report, writeErr
		}
		tasks[task.ID] = id
	}
	if err = r.parents(ctx, project, graph.Tasks, tasks); err != nil {
		return r.report, err
	}
	if err = r.memberships(ctx, project, graph, tasks, modules); err != nil {
		return r.report, err
	}
	r.removedSources(graph)
	return r.report, nil
}

func (r *Runner) states(ctx context.Context, project string, tasks []Page) (map[string]string, error) {
	states := map[string]string{}
	if project != "preview" {
		rows, err := r.Vector.list(ctx, r.Vector.collection("states", project))
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			name, _ := row["name"].(string)
			id, _ := row["id"].(string)
			states[name] = id
		}
	}
	for _, task := range tasks {
		if task.Archived {
			continue
		}
		if task.Status == "" {
			return nil, fmt.Errorf("task %s is missing a status", task.ID)
		}
		if states[task.Status] != "" {
			continue
		}
		if !r.Apply {
			states[task.Status] = "preview"
			continue
		}
		var row Object
		fields := map[string]any{"name": task.Status, "group": stateGroup(task.Status), "color": "#60646C"}
		if err := r.Vector.api.request(ctx, http.MethodPost, r.Vector.collection("states", project), fields, &row); err != nil {
			return nil, err
		}
		id, ok := row["id"].(string)
		if !ok || id == "" {
			return nil, fmt.Errorf("created state has no ID")
		}
		states[task.Status] = id
	}
	return states, nil
}

func (r *Runner) parents(ctx context.Context, project string, pages []Page, tasks map[string]string) error {
	for _, task := range pages {
		if task.Archived {
			continue
		}
		parents := task.Relations["Parent"]
		if len(parents) > 1 {
			r.report.Warnings = append(r.report.Warnings, "Multiple parents on "+task.ID+"; parent left unchanged")
			continue
		}
		var parent any
		if len(parents) == 1 {
			id, ok := tasks[parents[0]]
			if !ok {
				continue
			}
			parent = id
		}
		if !r.Apply && tasks[task.ID] == "preview" {
			continue
		}
		if _, err := r.upsert(ctx, "issues", project, task.ID, map[string]any{"parent": parent}); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) removedSources(graph Graph) {
	seen := map[string]bool{"project:" + graph.Roadmap.ID: true}
	for _, module := range graph.Modules {
		seen["modules:"+module.ID] = true
	}
	for _, task := range graph.Tasks {
		seen["issues:"+task.ID] = true
	}
	for key := range r.Journal.Entries {
		if !seen[key] && len(key) > 7 && key[:7] != "member:" {
			r.report.Warnings = append(r.report.Warnings, "Source no longer in selection; destination retained: "+key)
		}
	}
}
