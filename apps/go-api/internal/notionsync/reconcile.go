package notionsync

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
)

type Report struct {
	ProjectID string   `json:"project_id"`
	Created   int      `json:"created"`
	Updated   int      `json:"updated"`
	Unchanged int      `json:"unchanged"`
	Conflicts []string `json:"conflicts"`
	Warnings  []string `json:"warnings"`
}

type Runner struct {
	Vector  *Vector
	Journal *Journal
	Apply   bool
	Users   map[string]string
	report  Report
}

func equivalent(a, b any) bool {
	// Assignee and membership arrays are sets; API ordering may change.
	if a != nil && b != nil {
		_, aStrings := a.([]string)
		_, aValues := a.([]any)
		_, bStrings := b.([]string)
		_, bValues := b.([]any)
		if (aStrings || aValues) && (bStrings || bValues) {
			left, right := slices.Clone(stringsValue(a)), slices.Clone(stringsValue(b))
			slices.Sort(left)
			slices.Sort(right)
			return slices.Equal(left, right)
		}
	}
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return string(left) == string(right)
}

func fieldChanges(current Object, desired map[string]any, entry *Entry) (map[string]any, []string) {
	changes := map[string]any{}
	var conflicts []string
	for key, value := range desired {
		if equivalent(current[key], value) {
			continue
		}
		base, known := entry.Base[key]
		previousSource, sourceKnown := entry.Source[key]
		// The API may normalize a field (for example, wrapping HTML in a div).
		// Compare both sides with their own last acknowledged representations.
		if known && sourceKnown && equivalent(current[key], base) && equivalent(value, previousSource) {
			continue
		}
		pending, inFlight := entry.Pending[key]
		if known && equivalent(current[key], base) || inFlight && equivalent(current[key], pending) {
			changes[key] = value
		} else {
			conflicts = append(conflicts, key)
		}
	}
	slices.Sort(conflicts)
	return changes, conflicts
}

func (r *Runner) upsert(ctx context.Context, kind, project, sourceID string, desired map[string]any) (string, error) {
	key := kind + ":" + sourceID
	entry := r.Journal.Entries[key]
	var current Object
	var err error
	if project != "preview" {
		current, err = r.Vector.find(ctx, kind, project, sourceID, entry)
	}
	if err != nil {
		return "", err
	}
	if current == nil {
		return r.create(ctx, kind, project, sourceID, desired)
	}
	id, ok := current["id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("Vector returned no object ID")
	}
	if entry == nil {
		entry = &Entry{ID: id, Base: map[string]any{}}
	}
	changes, conflicts := fieldChanges(current, desired, entry)
	if len(conflicts) > 0 {
		r.report.Conflicts = append(r.report.Conflicts, fmt.Sprintf("%s: locally changed or untracked fields %v", key, conflicts))
		return id, nil
	}
	if len(changes) == 0 {
		r.report.Unchanged++
	} else {
		r.report.Updated++
	}
	if !r.Apply {
		return id, nil
	}
	if entry.ID == "" && entry.Pending != nil && kind == "issues" {
		// A create committed before its response was lost. The initial parent
		// is empty; a later local relationship must still be a conflict.
		entry.Base["parent"] = nil
	}
	entry.ID = id
	r.Journal.Entries[key] = entry
	if len(changes) > 0 {
		entry.Pending = maps.Clone(desired)
		if err = r.Journal.Save(); err != nil {
			return "", err
		}
		if err = r.Vector.api.request(ctx, http.MethodPatch, r.Vector.collection(kind, project)+id+"/", changes, nil); err != nil {
			return "", err
		}
		if err = r.Vector.api.request(ctx, http.MethodGet, r.Vector.collection(kind, project)+id+"/", nil, &current); err != nil {
			return "", err
		}
	}
	if entry.Base == nil {
		entry.Base = map[string]any{}
	}
	if entry.Source == nil {
		entry.Source = map[string]any{}
	}
	for field := range desired {
		entry.Base[field] = current[field]
		entry.Source[field] = desired[field]
	}
	entry.Pending = nil
	return id, r.Journal.Save()
}

func (r *Runner) create(ctx context.Context, kind, project, sourceID string, desired map[string]any) (string, error) {
	r.report.Created++
	if !r.Apply {
		return "preview", nil
	}
	key := kind + ":" + sourceID
	entry := r.Journal.Entries[key]
	if entry != nil && entry.ID != "" {
		return "", fmt.Errorf("mapped destination disappeared for %s; refusing to recreate", key)
	}
	if entry == nil {
		entry = &Entry{Base: map[string]any{}}
		r.Journal.Entries[key] = entry
	}
	entry.Pending = maps.Clone(desired)
	if err := r.Journal.Save(); err != nil {
		return "", err
	}
	body := maps.Clone(desired)
	body["external_id"] = sourceID
	body["external_source"] = "notion"
	var current Object
	if err := r.Vector.api.request(ctx, http.MethodPost, r.Vector.collection(kind, project), body, &current); err != nil {
		return "", err
	}
	id, ok := current["id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("created Vector object has no ID")
	}
	entry.ID = id
	if entry.Source == nil {
		entry.Source = map[string]any{}
	}
	for field := range desired {
		entry.Base[field] = current[field]
		entry.Source[field] = desired[field]
	}
	if kind == "issues" {
		// Parents are linked only after every task exists. Record the initial
		// server value so that the second pass can distinguish a local edit.
		entry.Base["parent"] = current["parent"]
	}
	entry.Pending = nil
	return id, r.Journal.Save()
}
