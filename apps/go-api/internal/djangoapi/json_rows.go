package djangoapi

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

// ReadJSONRows decodes a single JSON object column and preserves an empty result array.
func ReadJSONRows(ctx context.Context, tx interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, query string, args ...any) ([]map[string]any, error) {
	result, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer result.Close()
	out := []map[string]any{}
	for result.Next() {
		var raw []byte
		var row map[string]any
		if err = result.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &row); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, result.Err()
}
