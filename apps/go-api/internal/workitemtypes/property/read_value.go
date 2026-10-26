package property

import "fmt"

// ReadValue normalizes saved values without creation defaults, requiredness or
// current reference eligibility. A retained member/release remains a stable ID
// even when it is no longer eligible for new selection; labels are scoped by the
// authorized caller. Read-only content comes from the effective configuration.
func ReadValue(kind Kind, config Configuration, saved any) (any, error) {
	if kind == Text && config.Format == "Read-only" {
		saved = config.Content
	}
	if saved == nil {
		if kind == Boolean {
			return false, nil
		}
		return nil, nil
	}
	switch kind {
	case Dropdown, Member, Release, Cascading:
		multiple := config.Multiple || kind == Cascading
		ids, err := selectionIDs(saved, multiple)
		if err != nil || len(ids) == 0 {
			return nil, err
		}
		if kind == Cascading && len(ids) > len(config.Levels) {
			return nil, fmt.Errorf("saved cascading path exceeds its levels")
		}
		if multiple {
			return append([]string(nil), ids...), nil
		}
		return ids[0], nil
	default:
		return scalarValue(kind, config, saved)
	}
}
