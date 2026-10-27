package mcptransport

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ActionVisibility is application authorization, not a tool registry. Tool names,
// parameters, defaults and validation still come from the generated declarations.
// An empty action identifies a tool whose upstream contract has no discriminator.
type ActionVisibility map[string]map[string]bool

type actionDiscoverer interface {
	DiscoverActions(context.Context) (ActionVisibility, error)
}

func discoveryMiddleware(app actionDiscoverer) mcpsdk.Middleware {
	return func(next mcpsdk.MethodHandler) mcpsdk.MethodHandler {
		return func(ctx context.Context, method string, request mcpsdk.Request) (mcpsdk.Result, error) {
			if method != "tools/list" {
				return next(ctx, method, request)
			}
			visible, err := app.DiscoverActions(ctx)
			if err != nil {
				return nil, discoveryFailure()
			}
			result, err := next(ctx, method, request)
			if err != nil {
				return nil, err
			}
			listed, ok := result.(*mcpsdk.ListToolsResult)
			if !ok || listed == nil {
				return nil, discoveryFailure()
			}
			filtered := *listed
			filtered.Tools = make([]*mcpsdk.Tool, 0, len(listed.Tools))
			for _, tool := range listed.Tools {
				if tool == nil {
					return nil, discoveryFailure()
				}
				schema, allowed, filterErr := visibleSchema(tool.InputSchema, visible[tool.Name])
				if filterErr != nil {
					return nil, discoveryFailure()
				}
				if !allowed {
					continue
				}
				copyTool := *tool
				copyTool.InputSchema = schema
				filtered.Tools = append(filtered.Tools, &copyTool)
			}
			filtered.TTLMs, filtered.CacheScope = 0, "private"
			return &filtered, nil
		}
	}
}

func discoveryFailure() error {
	return &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: "Tool discovery is unavailable."}
}

type schemaObject map[string]json.RawMessage

// visibleSchema narrows a fresh copy of the generated schema. It never changes
// the shared SDK registration or synthesizes an action/parameter definition.
func visibleSchema(input any, allowed map[string]bool) (json.RawMessage, bool, error) {
	if len(allowed) == 0 {
		return nil, false, nil
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return nil, false, err
	}
	var root schemaObject
	if err = json.Unmarshal(encoded, &root); err != nil || root == nil {
		return nil, false, errors.New("invalid generated tool schema")
	}
	var branches []schemaObject
	if raw, exists := root["oneOf"]; exists {
		if err = json.Unmarshal(raw, &branches); err != nil {
			return nil, false, err
		}
	} else {
		branch, keep, filterErr := visibleBranch(root, allowed)
		if filterErr != nil || !keep {
			return nil, false, filterErr
		}
		raw, marshalErr := json.Marshal(branch)
		return raw, true, marshalErr
	}
	selected := make([]schemaObject, 0, len(branches))
	fields := map[string][]json.RawMessage{}
	for _, branch := range branches {
		narrowed, keep, filterErr := visibleBranch(branch, allowed)
		if filterErr != nil {
			return nil, false, filterErr
		}
		if !keep {
			continue
		}
		selected = append(selected, narrowed)
		var properties schemaObject
		if err = json.Unmarshal(narrowed["properties"], &properties); err != nil {
			return nil, false, err
		}
		for name, schema := range properties {
			fields[name] = appendUniqueSchema(fields[name], schema)
		}
	}
	if len(selected) == 0 {
		return nil, false, nil
	}
	properties := schemaObject{}
	for name, variants := range fields {
		if len(variants) == 1 {
			properties[name] = variants[0]
		} else {
			properties[name], err = json.Marshal(map[string]any{"anyOf": variants})
			if err != nil {
				return nil, false, err
			}
		}
	}
	root["oneOf"], err = json.Marshal(selected)
	if err != nil {
		return nil, false, err
	}
	root["properties"], err = json.Marshal(properties)
	if err != nil {
		return nil, false, err
	}
	raw, err := json.Marshal(root)
	return raw, true, err
}

func visibleBranch(branch schemaObject, allowed map[string]bool) (schemaObject, bool, error) {
	var properties schemaObject
	if err := json.Unmarshal(branch["properties"], &properties); err != nil {
		return nil, false, err
	}
	action, exists := properties["action"]
	if !exists {
		return branch, allowed[""], nil
	}
	var discriminator schemaObject
	if err := json.Unmarshal(action, &discriminator); err != nil {
		return nil, false, err
	}
	var actions []string
	if err := json.Unmarshal(discriminator["enum"], &actions); err != nil || len(actions) == 0 {
		return nil, false, errors.New("missing action enumeration")
	}
	permitted := make([]string, 0, len(actions))
	for _, name := range actions {
		if allowed[name] {
			permitted = append(permitted, name)
		}
	}
	if len(permitted) == 0 {
		return nil, false, nil
	}
	var err error
	discriminator["enum"], err = json.Marshal(permitted)
	if err != nil {
		return nil, false, err
	}
	properties["action"], err = json.Marshal(discriminator)
	if err != nil {
		return nil, false, err
	}
	branch["properties"], err = json.Marshal(properties)
	return branch, true, err
}

func appendUniqueSchema(variants []json.RawMessage, schema json.RawMessage) []json.RawMessage {
	for _, prior := range variants {
		if string(prior) == string(schema) {
			return variants
		}
	}
	return append(variants, schema)
}
