package systemapi

import (
	"encoding/json"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
)

func normalizeJSONFields(fields map[string]json.RawMessage, names ...string) map[string][]string {
	var invalid map[string][]string
	for _, name := range names {
		raw, exists := fields[name]
		if !exists || len(raw) == 0 {
			continue
		}
		normalized, err := pythoncompat.NormalizeJSONNumbers(raw)
		if err != nil {
			if invalid == nil {
				invalid = make(map[string][]string)
			}
			invalid[name] = []string{"Value must be valid JSON."}
		} else {
			fields[name] = normalized
		}
	}
	return invalid
}
