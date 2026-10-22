package parity

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
)

//go:embed plane_tables.json
var planeTablesJSON []byte

// PlaneTables returns the application tables in the committed schema inventory.
func PlaneTables() ([]string, error) {
	var tables []string
	if err := json.Unmarshal(planeTablesJSON, &tables); err != nil {
		return nil, fmt.Errorf("decode embedded Plane table inventory: %w", err)
	}
	if len(tables) == 0 {
		return nil, fmt.Errorf("embedded Plane table inventory is empty")
	}
	if !slices.IsSorted(tables) {
		return nil, fmt.Errorf("embedded Plane table inventory is not sorted")
	}
	for index := 1; index < len(tables); index++ {
		if tables[index-1] == tables[index] {
			return nil, fmt.Errorf("embedded Plane table inventory contains duplicate %q", tables[index])
		}
	}
	return slices.Clone(tables), nil
}
