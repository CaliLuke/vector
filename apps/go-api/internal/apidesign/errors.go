// Package apidesign holds shared transport policy for authored API designs.
package apidesign

import (
	"strconv"

	"github.com/CaliLuke/loom/dsl"
)

// JSONErrors preserves each endpoint's named status responses and their order.
func JSONErrors(prefix string, statuses ...int) {
	for _, status := range statuses {
		dsl.Response(prefix+strconv.Itoa(status), status, func() {
			dsl.Description("JSON error response.")
			dsl.Meta("openapi:description:errorName", "false")
			dsl.ContentType("application/json")
		})
	}
}
