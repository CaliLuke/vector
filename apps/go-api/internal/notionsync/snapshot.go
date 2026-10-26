package notionsync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// Snapshot supplies normalized pages exported through an authorized connection.
// Missing pages fail discovery before any destination writes occur.
type Snapshot map[string]Page

func ReadSnapshot(reader io.Reader) (Snapshot, error) {
	var pages []Page
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pages); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("snapshot must contain one JSON array")
	}
	snapshot := Snapshot{}
	for _, page := range pages {
		id, err := PageID(page.ID)
		if err != nil {
			return nil, err
		}
		if _, exists := snapshot[id]; exists {
			return nil, fmt.Errorf("duplicate snapshot page %s", id)
		}
		page.ID = id
		for name, related := range page.Relations {
			for index, raw := range related {
				relatedID, relationErr := PageID(raw)
				if relationErr != nil {
					return nil, fmt.Errorf("snapshot relation %s: %w", name, relationErr)
				}
				related[index] = relatedID
			}
		}
		snapshot[id] = page
	}
	return snapshot, nil
}

func (s Snapshot) Page(ctx context.Context, id string) (Page, error) {
	if err := ctx.Err(); err != nil {
		return Page{}, err
	}
	page, ok := s[id]
	if !ok {
		return Page{}, fmt.Errorf("page %s missing from snapshot; export the complete selection", id)
	}
	return page, nil
}
