package workitems

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
)

func (r *Repository) applyItemEditPlanning(ctx context.Context, q Request, command itemEditCommand, relations map[string]any) error {
	for _, kind := range []string{"cycle", "module"} {
		field := kind + "_id"
		if kind == "module" {
			field = "module_ids"
		}
		value, present := command.Changes[field]
		if !present {
			continue
		}
		if kind == "cycle" && value != nil {
			value = []any{value}
		}
		desired, err := rawIDs(value)
		if err != nil {
			return err
		}
		previous := []string{}
		for _, row := range relations[kind].([]map[string]any) {
			previous = append(previous, stringValue(row[kind+"_id"]))
		}
		added, removed := []any{}, []any{}
		for _, id := range desired {
			if !slices.Contains(previous, id) {
				added = append(added, id)
			}
		}
		for _, id := range previous {
			if !slices.Contains(desired, id) {
				removed = append(removed, id)
			}
		}
		if len(added)+len(removed) == 0 {
			continue
		}
		if err = r.authorizePlanning(ctx, q); err != nil {
			return err
		}
		if kind == "module" {
			if _, err = r.issueModules(ctx, q, djangoinput.Input{Fields: map[string]any{"modules": added, "removed_modules": removed}}); err != nil {
				return err
			}
			continue
		}
		for _, id := range removed {
			q.Params = map[string]string{"cycle_id": id.(string)}
			if _, err = r.removePlanning(ctx, q, "cycle", false); err != nil {
				return err
			}
		}
		if len(added) > 0 {
			q.Method = "POST"
			q.Params = map[string]string{"cycle_id": added[0].(string)}
			body, _ := json.Marshal(map[string]any{"issues": []string{q.IssueID}})
			if _, err = r.planningTransaction(ctx, q, bytes.NewReader(body), "application/json"); err != nil {
				return err
			}
		}
	}
	return nil
}
