package routing

import (
	"fmt"
	"strings"
)

//go:generate go run ../../cmd/route-policy -output ../../../proxy/api-allow.caddy

// RenderCaddy combines existing upstream Allow headers. Caddyfiles own forwarding.
func RenderCaddy(policy Policy) ([]byte, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	output := []byte("# Code generated from apps/go-api/internal/routing/resource-methods.json. DO NOT EDIT.\n")
	for index, resource := range policy.Resources {
		pattern, err := resourcePattern(resource)
		if err != nil {
			return nil, err
		}
		output = fmt.Appendf(output, "\n@api_allow_%03d {\n\tpath_regexp %s\n}\n", index+1, pattern)
		output = fmt.Appendf(output, "header @api_allow_%03d >Allow %q {\n\tmatch header Allow *\n}\n", index+1, strings.Join(resource.Methods, ", "))
	}
	return output, nil
}
