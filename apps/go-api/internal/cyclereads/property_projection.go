package cyclereads

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/CaliLuke/go-plane/internal/workitemtypes"
)

// The SQL envelope owns grouping and pagination. Collect only its leaf rows,
// enrich them in one batch, then retain every bucket and count unchanged.
func projectGroupedRows(ctx context.Context, options IssueListOptions, raw []byte) ([]byte, error) {
	var results map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&results); err != nil {
		return nil, err
	}
	depth := 1
	if options.SubGroupBy != "" {
		depth = 2
	}
	rows, err := groupedPropertyRows(results, depth)
	if err != nil {
		return nil, err
	}
	if err = options.snapshot.ProjectStoredRows(ctx, rows, workitemtypes.DiscoveryElevatedProjectVisibility); err != nil {
		return nil, err
	}
	return json.Marshal(results)
}

func groupedPropertyRows(groups map[string]any, depth int) ([]map[string]any, error) {
	rows := []map[string]any{}
	for _, value := range groups {
		bucket, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid cycle property group")
		}
		if depth > 1 {
			subgroups, valid := bucket["results"].(map[string]any)
			if !valid {
				return nil, fmt.Errorf("invalid cycle property subgroups")
			}
			children, err := groupedPropertyRows(subgroups, depth-1)
			if err != nil {
				return nil, err
			}
			rows = append(rows, children...)
			continue
		}
		items, ok := bucket["results"].([]any)
		if !ok {
			return nil, fmt.Errorf("invalid cycle property rows")
		}
		for _, item := range items {
			row, valid := item.(map[string]any)
			if !valid {
				return nil, fmt.Errorf("invalid cycle property row")
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}
