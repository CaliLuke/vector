package workitems

import (
	"maps"

	"github.com/CaliLuke/go-plane/internal/requestinput"
)

func composedItemEditInput(review *itemEditState, input requestinput.Input, row map[string]any) requestinput.Input {
	if review == nil || len(review.plan.Preserved) == 0 {
		return input
	}
	input.Fields = maps.Clone(input.Fields)
	input.Fields["description_html"] = row["description_html"]
	input.Fields["custom_properties"] = row["custom_properties"]
	input.Raw = nil
	return input
}
