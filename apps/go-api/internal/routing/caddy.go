package routing

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

//go:generate go run ../../cmd/route-policy -inventory route-inventory.json -output ../../../proxy/go-active-routes.caddy

// RenderCaddy renders only inventory operations that are explicitly Go active.
func RenderCaddy(inventory Inventory) ([]byte, error) {
	if err := inventory.Validate(); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	output.WriteString("# Code generated from apps/go-api/internal/routing/route-inventory.json. DO NOT EDIT.\n")
	active := inventory.ActiveOperations()
	if len(active) == 0 {
		output.WriteString("# No application operations are Go active. Django owns the complete application route surface.\n")
		return output.Bytes(), nil
	}
	allowPaths := make(map[string]bool)
	for index, operation := range active {
		mixedDjangoPath := djangoOwnsPathMethod(inventory, operation.Path)
		allMethods := (proxyAllMethods(operation) || operation.State == StateDjangoRemoved) && !mixedDjangoPath
		routedMethods := ""
		if !allMethods {
			routedMethods = operationProxyMethods(operation, mixedDjangoPath)
		}
		fmt.Fprintf(&output, "\n@go_active_%03d {\n", index+1)
		if !allMethods {
			// OPTIONS stays with Django while that path has a Django-owned method,
			// so DRF can describe the complete write schema. A fully retired path
			// sends every request to Go so its compatibility stack owns dispatch.
			fmt.Fprintf(&output, "\tmethod %s\n", routedMethods)
		}
		pathPattern := caddyPathPattern(operation)
		if allMethods {
			basePattern := strings.TrimSuffix(pathPattern, "/$")
			pathPattern = basePattern + "/?$"
			if operation.Implementation != nil && *operation.Implementation == "api_v1_stickies" {
				// DRF DefaultRouter adds lowercase alphanumeric format suffixes,
				// with an optional trailing slash, alongside canonical routes.
				pathPattern = basePattern + `(?:/?|\.[a-z0-9]+/?)$`
			}
		}
		fmt.Fprintf(&output, "\tpath_regexp %s\n", pathPattern)
		output.WriteString("}\n")
		if mixedDjangoPath && !allowPaths[operation.Path] {
			allowPaths[operation.Path] = true
			fmt.Fprintf(&output, "\n@go_active_allow_%03d {\n", index+1)
			fmt.Fprintf(&output, "\tnot method %s\n", strings.Join(goOwnedPathMethods(inventory, operation.Path), " "))
			fmt.Fprintf(&output, "\tpath_regexp %s\n", caddyPathPattern(operation))
			output.WriteString("}\n")
			fmt.Fprintf(
				&output,
				"header @go_active_allow_%03d >Allow %q {\n\tmatch header Allow *\n}\n",
				index+1,
				pathAllowHeader(inventory, operation.Path),
			)
		} else if !mixedDjangoPath && splitGoOwnedPath(inventory, operation.Path) && !allowPaths[operation.Path] {
			allowPaths[operation.Path] = true
			fmt.Fprintf(&output, "\n@go_active_allow_%03d {\n", index+1)
			fmt.Fprintf(&output, "\tpath_regexp %s\n", caddyPathPattern(operation))
			output.WriteString("}\n")
			fmt.Fprintf(
				&output,
				"header @go_active_allow_%03d >Allow %q {\n\tmatch header Allow *\n}\n",
				index+1,
				pathAllowHeader(inventory, operation.Path),
			)
		}
		// Keep response middleware before the terminal proxy handler. This is
		// required by literal-order route blocks such as the AIO Caddyfile.
		fmt.Fprintf(&output, "reverse_proxy @go_active_%03d {$GO_API_UPSTREAM:go-api:8080}\n", index+1)
	}
	// Once Django is fully retired, Go also owns unknown API/auth paths and
	// their 404 responses. Keep this in the generated policy so the static
	// routing gate sees the same fallback as each shipping Caddyfile.
	fullyRetired := len(inventory.Operations) > 0
	for _, operation := range inventory.Operations {
		fullyRetired = fullyRetired && operation.State == StateDjangoRemoved
	}
	if fullyRetired {
		output.WriteString("\n@go_runtime_fallback {\n\tpath_regexp ^/(?:api|auth)/\n}\nreverse_proxy @go_runtime_fallback {$GO_API_UPSTREAM:go-api:8080}\n")
	}
	return output.Bytes(), nil
}

func goOwnedPathMethods(inventory Inventory, path string) []string {
	methods := make(map[string]bool)
	for _, operation := range inventory.Operations {
		if operation.Path == path && operation.GoActive() {
			methods[operation.Method] = true
			for _, companion := range operation.ProtocolCompanions.Values() {
				if companion != "OPTIONS" {
					methods[companion] = true
				}
			}
		}
	}
	ordered := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	result := make([]string, 0, len(methods))
	for _, method := range ordered {
		if methods[method] {
			result = append(result, method)
		}
	}
	return result
}

// splitGoOwnedPath reports whether a fully Go-owned path is split across
// multiple implementations. Their error responses can advertise only the
// methods handled by one implementation, so the proxy restores the resource's
// full Allow header when an upstream already supplied one.
func splitGoOwnedPath(inventory Inventory, path string) bool {
	implementations := make(map[string]bool)
	for _, operation := range inventory.Operations {
		if operation.Path != path {
			continue
		}
		if !operation.GoActive() || operation.Implementation == nil {
			return false
		}
		implementations[*operation.Implementation] = true
	}
	return len(implementations) > 1
}

func operationProxyMethods(operation Operation, leaveOptionsWithDjango bool) string {
	var methods strings.Builder
	methods.WriteString(operation.Method)
	for _, companion := range operation.ProtocolCompanions.Values() {
		if leaveOptionsWithDjango && companion == "OPTIONS" {
			continue
		}
		methods.WriteByte(' ')
		methods.WriteString(companion)
	}
	return methods.String()
}

func proxyAllMethods(operation Operation) bool {
	return operation.Implementation != nil && *operation.Implementation == "public_project_metadata_reads"
}

func pathAllowHeader(inventory Inventory, path string) string {
	methods := make(map[string]bool)
	for _, operation := range inventory.Operations {
		if operation.Path == path {
			methods[operation.Method] = true
			companions := operation.ProtocolCompanions
			if !operation.AllowedProtocolCompanions.IsZero() {
				companions = operation.AllowedProtocolCompanions
			}
			for _, companion := range companions.Values() {
				methods[companion] = true
			}
		}
	}
	ordered := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	allow := make([]string, 0, len(methods))
	for _, method := range ordered {
		if methods[method] {
			allow = append(allow, method)
		}
	}
	return strings.Join(allow, ", ")
}

func djangoOwnsPathMethod(inventory Inventory, path string) bool {
	for _, operation := range inventory.Operations {
		if operation.Path == path && !operation.GoActive() {
			return true
		}
	}
	return false
}

func caddyPathPattern(operation Operation) string {
	path := operation.Path
	var pattern strings.Builder
	pattern.WriteByte('^')
	for len(path) > 0 {
		start := strings.IndexByte(path, '{')
		if start < 0 {
			pattern.WriteString(regexp.QuoteMeta(path))
			break
		}
		pattern.WriteString(regexp.QuoteMeta(path[:start]))
		end := strings.IndexByte(path[start:], '}')
		if end < 0 {
			pattern.WriteString(regexp.QuoteMeta(path[start:]))
			break
		}
		parameter := path[start+1 : start+end]
		if replacementUUIDParameter(operation, parameter) {
			pattern.WriteString(`([0-9A-Fa-f]{8}[0-9A-Fa-f]{4}[0-9A-Fa-f]{4}[89ABab][0-9A-Fa-f]{3}[0-9A-Fa-f]{12}|[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[89ABab][0-9A-Fa-f]{3}-[0-9A-Fa-f]{12}|\{[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[89ABab][0-9A-Fa-f]{3}-[0-9A-Fa-f]{12}\}|urn:uuid:[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[89ABab][0-9A-Fa-f]{3}-[0-9A-Fa-f]{12})`)
		} else if canonicalUUIDParameter(operation, parameter) {
			pattern.WriteString(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
		} else if operation.Implementation != nil && (*operation.Implementation == "invitations_membership" || *operation.Implementation == "api_v1_stickies") && strings.HasPrefix(operation.Path, "/api/v1/") && parameter == "pk" {
			pattern.WriteString(`[^/.]+`)
		} else {
			pattern.WriteString("[^/]+")
		}
		path = path[start+end+1:]
	}
	pattern.WriteByte('$')
	return pattern.String()
}

func replacementUUIDParameter(operation Operation, parameter string) bool {
	if operation.Implementation == nil {
		return false
	}
	return *operation.Implementation == "stickies" && parameter == "pk"
}

func canonicalUUIDParameter(operation Operation, parameter string) bool {
	if operation.Implementation == nil {
		return false
	}
	switch *operation.Implementation {
	case "instance_management":
		return parameter == "pk"
	case "workspace_webhooks":
		return parameter == "pk" || parameter == "webhook_id"
	case "project_configuration", "estimate_reads", "api_label_reads":
		// Estimate point lookup uses a Django str converter; its parent and
		// the API label lookups use UUID converters.
		return parameter == "project_id" || parameter == "estimate_id" || parameter == "pk"
	case "notification_reads", "notification_actions":
		return parameter == "pk"
	case "project_membership_reads", "project_membership_lifecycle":
		return parameter == "pk" || parameter == "project_id" || parameter == "member_id"
	case "invitations_membership":
		// Session paths use Django UUID converters. The API-key invitation
		// router accepts arbitrary lookup strings before model validation.
		return !strings.HasPrefix(operation.Path, "/api/v1/") && (parameter == "pk" || parameter == "project_id")
	case "project_state_reads", "project_label_reads", "project_label_writes", "project_lifecycle":
		return parameter == "pk" || parameter == "project_id"
	case "public_project_metadata_reads":
		return parameter == "project_id"
	case "activity_reads":
		return parameter == "project_id" || parameter == "issue_id" || parameter == "pk"
	case "assets_lifecycle":
		return parameter == "workspace_id" || parameter == "project_id" || parameter == "issue_id" || parameter == "asset_id" || parameter == "entity_id" || parameter == "pk"
	default:
		return false
	}
}
