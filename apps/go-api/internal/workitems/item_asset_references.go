package workitems

import (
	"context"
	"slices"
	"strings"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"golang.org/x/net/html"
)

// Inspect only image elements in description and schema-identified rich text.
// Text properties that happen to contain markup are not upload references.
func (r *Repository) itemAssetReferences(ctx context.Context, workspace any, row map[string]any) ([]string, error) {
	return itemAssetReferences(ctx, r.db, workspace, row)
}

func itemAssetReferences(ctx context.Context, db Database, workspace any, row map[string]any) ([]string, error) {
	ids := make([]string, 0)
	visit := func(value string) {
		parser := html.NewTokenizer(strings.NewReader(value))
		for {
			switch parser.Next() {
			case html.ErrorToken:
				return
			case html.StartTagToken, html.SelfClosingTagToken:
				token := parser.Token()
				if token.Data != "image-component" && token.Data != "img" {
					continue
				}
				for _, attr := range token.Attr {
					if attr.Key == "src" {
						if id, err := uuid.Parse(attr.Val); err == nil {
							ids = append(ids, id.String())
						}
					}
				}
			}
		}
	}
	description, _ := row["description_html"].(string)
	visit(description)
	values, _ := row["custom_properties"].(map[string]any)
	if len(values) > 0 {
		keys := make([]string, 0, len(values))
		for id := range values {
			keys = append(keys, id)
		}
		fields, err := db.Query(ctx, `SELECT id::text FROM work_item_properties WHERE workspace_id=$1 AND id::text=ANY($2::text[]) AND definition->>'kind'=$3`, workspace, keys, string(property.RichText))
		if err != nil {
			return nil, err
		}
		defer fields.Close()
		for fields.Next() {
			var id string
			if err = fields.Scan(&id); err != nil {
				return nil, err
			}
			value, _ := values[id].(string)
			visit(value)
		}
		if err = fields.Err(); err != nil {
			return nil, err
		}
	}
	slices.Sort(ids)
	return slices.Compact(ids), nil
}
