package workitems

import (
	"strings"

	"github.com/CaliLuke/go-plane/internal/issuequery"
)

func validateAPIFilterMode(q Request) error {
	if len(q.Params["pql"]) > issuequery.PQLMaxBytes {
		return invalid(map[string]any{"pql": "PQL exceeds the input size limit.", "code": "invalid_pql"})
	}
	if q.Params["filters"] != "" {
		return invalid(map[string]any{"filters": "Structured filters are not supported by this API. Use supported PQL predicates.", "unsupported_parameters": []string{"filters"}})
	}
	if strings.TrimSpace(q.Params["pql"]) != "" && (q.Params["external_id"] != "" || q.Params["external_source"] != "") {
		return invalid(map[string]any{"pql": "PQL cannot be combined with external identity lookup parameters."})
	}
	return nil
}

func (r *Repository) appendPQL(q Request, where []string, args []any) ([]string, []any, error) {
	if !q.API || q.Params["pql"] == "" {
		return where, args, nil
	}
	predicate, err := issuequery.BuildPQL(q.Params["pql"], issuequery.PQLContext{ActorID: q.ActorID, Timezone: q.Timezone, Now: r.now()}, &args)
	if err != nil {
		return nil, args, invalid(map[string]any{"pql": err.Error(), "code": "invalid_pql"})
	}
	if predicate == "" {
		return where, args, nil
	}
	return append(where, predicate), args, nil
}
