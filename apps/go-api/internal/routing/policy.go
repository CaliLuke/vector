// Package routing owns resource method headers shared by native HTTP services.
package routing

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

//go:embed resource-methods.json
var embeddedPolicy []byte

type Policy struct {
	Version   int        `json:"version"`
	Resources []Resource `json:"resources"`
}

// Resource combines the methods of services mounted on the same HTTP path.
type Resource struct {
	Path           string   `json:"path"`
	Methods        []string `json:"methods"`
	UUIDParameters []string `json:"uuid_parameters,omitempty"`
}

func Load() (Policy, error) { return Parse(embeddedPolicy) }

func Parse(data []byte) (Policy, error) {
	var policy Policy
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		return Policy{}, fmt.Errorf("decode resource method policy: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Policy{}, fmt.Errorf("resource method policy must contain one JSON document")
	}
	return policy, policy.Validate()
}

func (policy Policy) Validate() error {
	if policy.Version != 1 || len(policy.Resources) == 0 {
		return fmt.Errorf("resource method policy requires version 1 and resources")
	}
	seen := make(map[string]bool)
	for _, resource := range policy.Resources {
		_, err := resourcePattern(resource)
		if err != nil {
			return err
		}
		shape := parameter.ReplaceAllString(resource.Path, "{}")
		if seen[shape] {
			return fmt.Errorf("resource method policy repeats path %q", resource.Path)
		}
		seen[shape] = true
		methods := make(map[string]bool)
		for _, method := range resource.Methods {
			switch method {
			case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
			default:
				return fmt.Errorf("resource %q has unsupported method %q", resource.Path, method)
			}
			if methods[method] {
				return fmt.Errorf("resource %q repeats method %q", resource.Path, method)
			}
			methods[method] = true
		}
		if len(methods) == 0 {
			return fmt.Errorf("resource %q requires methods", resource.Path)
		}
	}
	return nil
}

var parameter = regexp.MustCompile(`\{[a-zA-Z_][a-zA-Z0-9_]*\}`)

func resourcePattern(resource Resource) (string, error) {
	path := resource.Path
	if (!strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/auth/")) || !strings.HasSuffix(path, "/") || strings.ContainsAny(path, "?#\r\n\t ") {
		return "", fmt.Errorf("invalid resource path %q", path)
	}
	var pattern strings.Builder
	uuidParameters := make(map[string]bool)
	for _, name := range resource.UUIDParameters {
		if uuidParameters[name] || !strings.Contains(path, "{"+name+"}") {
			return "", fmt.Errorf("invalid UUID parameter %q for resource %q", name, path)
		}
		uuidParameters[name] = true
	}
	pattern.WriteByte('^')
	for {
		span := parameter.FindStringIndex(path)
		literal := path
		if span != nil {
			literal = path[:span[0]]
		}
		if strings.ContainsAny(literal, "{}") {
			return "", fmt.Errorf("invalid resource path parameter in %q", path)
		}
		pattern.WriteString(regexp.QuoteMeta(literal))
		if span == nil {
			break
		}
		if uuidParameters[path[span[0]+1:span[1]-1]] {
			// UUID resource IDs must not capture literal actions such as "leave".
			pattern.WriteString(`[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}`)
		} else {
			pattern.WriteString(`[^/]+`)
		}
		path = path[span[1]:]
	}
	pattern.WriteByte('$')
	return pattern.String(), nil
}
