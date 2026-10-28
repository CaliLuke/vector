package mcpactions

import (
	"errors"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
)

func receiptOperationError(err error) error {
	switch {
	case errors.Is(err, mutationreceipts.ErrConflict):
		return api.MakeRequestConflict(errors.New("request input changed"))
	case errors.Is(err, mutationreceipts.ErrIndeterminate):
		return api.MakeIndeterminate(errors.New("unacknowledged commit"))
	}
	return nil
}
