package notionsync

import (
	"context"
	"fmt"
	"net/http"
	"slices"
)

func stringsValue(value any) []string {
	if values, ok := value.([]string); ok {
		return values
	}
	var result []string
	if values, ok := value.([]any); ok {
		for _, raw := range values {
			if value, ok := raw.(string); ok {
				result = append(result, value)
			}
		}
	}
	return result
}

func (r *Runner) memberships(ctx context.Context, project string, graph Graph, tasks, modules map[string]string) error {
	for _, module := range graph.Modules {
		moduleID := modules[module.ID]
		if moduleID == "preview" {
			continue
		}
		path := r.Vector.collection("modules", project) + moduleID + "/module-issues/"
		rows, err := r.Vector.list(ctx, path)
		if err != nil {
			return err
		}
		current := []string{}
		for _, row := range rows {
			if id, ok := row["id"].(string); ok {
				current = append(current, id)
			}
		}
		desired := []string{}
		for sourceID, moduleSources := range graph.Members {
			id := tasks[sourceID]
			if id == "" {
				// Archived tasks retain their existing memberships.
				if entry := r.Journal.Entries["issues:"+sourceID]; entry != nil {
					id = entry.ID
				}
			}
			if id != "" && id != "preview" && slices.Contains(moduleSources, module.ID) {
				desired = append(desired, id)
			}
		}
		slices.Sort(desired)
		if err = r.syncMembers(ctx, module.ID, path, current, desired); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) syncMembers(ctx context.Context, sourceID, path string, current, desired []string) error {
	key := "member:" + sourceID
	entry := r.Journal.Entries[key]
	if entry == nil {
		entry = &Entry{Base: map[string]any{}}
	}
	previous := stringsValue(entry.Base["issues"])
	pending := stringsValue(entry.Pending["issues"])
	for _, id := range previous {
		removalInFlight := entry.Pending != nil && !slices.Contains(pending, id)
		if !slices.Contains(current, id) && slices.Contains(desired, id) && !removalInFlight {
			r.report.Conflicts = append(r.report.Conflicts, "Module membership removed locally: "+sourceID+" / "+id)
			return nil
		}
	}
	var add, remove []string
	for _, id := range desired {
		if !slices.Contains(current, id) {
			add = append(add, id)
		}
	}
	// A previous attempt may have added links before losing its response.
	// Include that intent when the source changes before a retry.
	managed := append(slices.Clone(previous), pending...)
	slices.Sort(managed)
	for _, id := range slices.Compact(managed) {
		if !slices.Contains(desired, id) && slices.Contains(current, id) {
			remove = append(remove, id)
		}
	}
	if len(add)+len(remove) > 0 {
		r.report.Updated++
	}
	if !r.Apply {
		return nil
	}
	entry.Pending = map[string]any{"issues": desired}
	r.Journal.Entries[key] = entry
	if err := r.Journal.Save(); err != nil {
		return err
	}
	if len(add) > 0 {
		if err := r.Vector.api.request(ctx, http.MethodPost, path, map[string]any{"issues": add}, nil); err != nil {
			return err
		}
	}
	for _, id := range remove {
		if err := r.Vector.api.request(ctx, http.MethodDelete, path+id+"/", nil, nil); err != nil {
			return fmt.Errorf("remove stale module membership: %w", err)
		}
	}
	entry.Base = map[string]any{"issues": desired}
	entry.Pending = nil
	return r.Journal.Save()
}
