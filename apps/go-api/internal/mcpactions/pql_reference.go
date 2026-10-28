package mcpactions

import (
	"context"
	_ "embed"
	"errors"

	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
)

// References derived from makeplane/plane-mcp-server at
// dc7cfd9f3c305891909e2372f0cc507291353801. Its MIT notice is retained in
// ../../mcp/testdata/plane-mcp/LICENSE; Vector product naming is applied here.
//
//go:embed pql_brief.txt
var pqlBrief string

//go:embed pql_full.txt
var pqlFull string

func (s *Service) GetPqlReference(ctx context.Context, input *api.GetPqlReferencePayload) (*api.GetPqlReferenceResult, error) {
	if _, _, err := readActor(ctx, nil); err != nil {
		return nil, err
	}
	detail := "full"
	if input != nil && input.Detail != "" {
		detail = input.Detail
	}
	switch detail {
	case "brief":
		return &api.GetPqlReferenceResult{Detail: detail, Reference: pqlBrief}, nil
	case "full":
		return &api.GetPqlReferenceResult{Detail: detail, Reference: pqlFull}, nil
	default:
		return nil, api.MakeInvalidParams(errors.New("detail must be brief or full"))
	}
}
