package workitems

import (
	"maps"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
)

func (r *Repository) composedItemEditInput(input djangoinput.Input, row map[string]any) djangoinput.Input {
	if r.itemEdit == nil || len(r.itemEdit.plan.Preserved) == 0 {
		return input
	}
	input.Fields = maps.Clone(input.Fields)
	input.Fields["description_html"] = row["description_html"]
	input.Fields["custom_properties"] = row["custom_properties"]
	input.Raw = nil
	return input
}
