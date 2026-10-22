package workspaceprojectreads

import (
	_ "embed"
	"encoding/json"
	"slices"
)

// Frozen from Project.TIMEZONE_CHOICES in the pinned Django source contract.
// This field's choices are narrower than the session timezone policy.
//
//go:embed project_timezones.json
var projectTimezoneData []byte

var projectTimezones = func() []string {
	var values []string
	if err := json.Unmarshal(projectTimezoneData, &values); err != nil {
		panic(err)
	}
	return values
}()

func ValidProjectTimezone(value string) bool {
	_, found := slices.BinarySearch(projectTimezones, value)
	return found
}

func validProjectTimezone(value string) bool { return ValidProjectTimezone(value) }
