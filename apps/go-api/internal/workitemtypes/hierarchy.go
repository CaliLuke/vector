package workitemtypes

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type HierarchyCatalog struct {
	Catalog
	HierarchyEnabled bool `json:"hierarchy_enabled"`
	DefaultLevel     int  `json:"default_level"`
}
type HierarchyPlacement struct {
	TypeID string `json:"type_id"`
	Level  int    `json:"level"`
}
type HierarchyInput struct {
	Revision     int64                `json:"revision"`
	DefaultLevel int                  `json:"default_level"`
	Placements   []HierarchyPlacement `json:"placements"`
	ReviewID     string               `json:"review_id,omitempty"`
	Confirm      string               `json:"confirm,omitempty"`
}
type hierarchyCommand struct {
	Action       string               `json:"action"`
	Revision     int64                `json:"revision"`
	DefaultLevel int                  `json:"default_level"`
	Placements   []HierarchyPlacement `json:"placements"`
}

type HierarchyPair struct {
	ChildID         string `json:"child_id"`
	ChildName       string `json:"child_name"`
	ChildProjectID  string `json:"child_project_id"`
	ParentID        string `json:"parent_id"`
	ParentName      string `json:"parent_name"`
	ParentProjectID string `json:"parent_project_id"`
}
type HierarchyReview struct {
	ID              string          `json:"id"`
	Links           int             `json:"links"`
	RestrictedLinks int             `json:"restricted_links"`
	Pairs           []HierarchyPair `json:"pairs"`
}

func (r *Repository) HierarchyCatalog(ctx context.Context, actor, slug string) (HierarchyCatalog, error) {
	return readSchemaCatalog(ctx, r, actor, slug, loadHierarchyCatalog)
}

func loadHierarchyCatalog(ctx context.Context, tx pgx.Tx, workspace string, admin bool) (HierarchyCatalog, error) {
	catalog, err := loadCatalog(ctx, tx, workspace, admin)
	result := HierarchyCatalog{Catalog: catalog}
	if err != nil {
		return result, err
	}
	err = tx.QueryRow(ctx, `SELECT COALESCE((SELECT hierarchy_enabled FROM work_item_type_settings WHERE workspace_id=$1),false),COALESCE((SELECT default_level FROM work_item_type_settings WHERE workspace_id=$1),0)`, workspace).Scan(&result.HierarchyEnabled, &result.DefaultLevel)
	return result, err
}
func normalizeHierarchy(input HierarchyInput) (hierarchyCommand, error) {
	command := hierarchyCommand{Action: "hierarchy", Revision: input.Revision, Placements: []HierarchyPlacement{}}
	if input.DefaultLevel < 0 || input.DefaultLevel > 20 {
		return command, reject(422, "Choose a hierarchy level between 0 and 20.")
	}
	assigned := map[string]int{}
	levels := []int{0}
	for _, placement := range input.Placements {
		id, err := uuid.Parse(placement.TypeID)
		if err != nil {
			return command, reject(422, "Every hierarchy type must belong to this workspace.")
		}
		if _, exists := assigned[id.String()]; exists {
			return command, reject(422, "A type can appear in only one hierarchy level.")
		}
		if placement.Level < 0 || placement.Level > 20 {
			return command, reject(422, "Choose a hierarchy level between 0 and 20.")
		}
		assigned[id.String()] = placement.Level
		levels = append(levels, placement.Level)
	}
	slices.Sort(levels)
	levels = slices.Compact(levels)
	for id, original := range assigned {
		level := slices.Index(levels, original)
		command.Placements = append(command.Placements, HierarchyPlacement{TypeID: id, Level: level})
	}
	slices.SortFunc(command.Placements, func(a, b HierarchyPlacement) int {
		if a.TypeID < b.TypeID {
			return -1
		}
		if a.TypeID > b.TypeID {
			return 1
		}
		return 0
	})
	command.DefaultLevel = max(0, slices.Index(levels, input.DefaultLevel))
	return command, nil
}

func validateHierarchyCommand(command hierarchyCommand, catalog HierarchyCatalog) error {
	if !catalog.Enabled {
		return reject(409, "Enable work item types first.")
	}
	if catalog.Revision != command.Revision {
		return reject(409, "Hierarchy configuration changed. Refresh the review.")
	}
	valid := map[string]bool{}
	for _, item := range catalog.Types {
		valid[item.ID] = true
	}
	for _, placement := range command.Placements {
		if !valid[placement.TypeID] {
			return reject(422, "Every hierarchy type must belong to this workspace.")
		}
	}
	return nil
}
