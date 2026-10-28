package workitemtypes

import (
	"context"
	"regexp"
	"strings"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

var propertyFilterKey = regexp.MustCompile(`^customproperty_([0-9a-f-]{36})(?:_level_([0-9a-f-]{36}))?(?:_catalog_(base|[0-9a-f-]{36}))?$`)

type ScopedPropertyPredicate struct {
	id         string
	comparison *ProjectedPredicate
}

func (p *ScopedPropertyPredicate) Match(values map[string]ProjectedValue) (bool, bool) {
	return p.comparison.Match(values[p.id])
}

// FilterPredicate authorizes identifiers against this discovery snapshot. The
// full effective configuration never leaves this boundary. Cascading keys include
// their stable level and option-catalog owner, not the winning display context.
func (snapshot *PropertyDiscoverySnapshot) FilterPredicate(ctx context.Context, tx pgx.Tx, field string, raw any) (*ScopedPropertyPredicate, bool, error) {
	if !strings.HasPrefix(field, "customproperty_") {
		return nil, false, nil
	}
	key, operator := field, "exact"
	if index := strings.LastIndex(field, "__"); index >= 0 {
		key, operator = field[:index], field[index+2:]
	}
	parts := propertyFilterKey.FindStringSubmatch(key)
	if parts == nil {
		return nil, true, reject(422, "Invalid custom property filter identity.")
	}
	id, level, catalog := parts[1], parts[2], parts[3]
	for _, identity := range []string{id, level, catalog} {
		if identity != "" && identity != "base" {
			parsed, err := uuid.Parse(identity)
			if err != nil || parsed.String() != identity {
				return nil, true, reject(422, "Invalid custom property filter identity.")
			}
		}
	}
	if catalog == "base" {
		catalog = ""
	}
	definition, options, available, err := snapshot.propertyFilterScope(id, level, catalog)
	if err != nil {
		return nil, true, err
	}
	if definition == nil || !available {
		return nil, true, reject(422, "This custom property or catalog is unavailable. Update the filter.")
	}
	if definition.Kind == property.Cascading {
		if level == "" || parts[3] == "" {
			return nil, true, reject(422, "Choose a cascading level and its option catalog.")
		}
	} else if level != "" || parts[3] != "" && definition.Kind != property.Dropdown {
		return nil, true, reject(422, "This property does not have that level or catalog.")
	}
	comparison, err := CompileProjectedPredicate(definition.Kind, definition.Configuration.Result, level, catalog, operator, raw)
	if err != nil {
		return nil, true, reject(422, err.Error())
	}
	if selected, ok := comparison.operand.([]string); ok {
		if definition.Kind == property.Member || definition.Kind == property.Release || definition.Kind == property.Issue {
			options, err = snapshot.predicateReferences(ctx, tx, definition.Kind)
			if err != nil {
				return nil, true, err
			}
		}
		for _, selectedID := range selected {
			if !options[selectedID] {
				return nil, true, reject(422, "A selected property value is unavailable. Update the filter.")
			}
		}
	}
	return &ScopedPropertyPredicate{id: id, comparison: comparison}, true, nil
}

func (snapshot *PropertyDiscoverySnapshot) propertyFilterScope(id, level, catalog string) (*property.Definition, map[string]bool, bool, error) {
	var definition *property.Definition
	options := map[string]bool{}
	available := false
	for typeID, definitions := range snapshot.definitions {
		for _, def := range definitions {
			if def.ID != id {
				continue
			}
			definition = &def
			levelIndex := -1
			for index, item := range def.Configuration.Levels {
				if item.ID == level {
					levelIndex = index
				}
			}
			if def.Kind == property.Cascading && levelIndex < 0 {
				continue
			}
			for projectID := range snapshot.projects {
				config, owner, err := property.OptionCatalog(def, property.Scope{ProjectID: projectID, TypeID: typeID})
				if err != nil {
					return nil, nil, false, err
				}
				if owner != catalog {
					continue
				}
				available = true
				for _, option := range config.Options {
					if def.Kind != property.Cascading || option.Level == levelIndex {
						options[option.ID] = true
					}
				}
			}
		}
	}
	return definition, options, available, nil
}

func (snapshot *PropertyDiscoverySnapshot) predicateReferences(ctx context.Context, tx pgx.Tx, kind property.Kind) (map[string]bool, error) {
	if cached := snapshot.references[kind]; cached != nil {
		return cached, nil
	}
	query := `SELECT member_id::text FROM workspace_members WHERE workspace_id=$1`
	args := []any{snapshot.workspace}
	if kind == property.Release {
		projects := []string{}
		for id := range snapshot.projects {
			projects = append(projects, id)
		}
		query = `SELECT id::text FROM work_item_releases WHERE workspace_id=$1 AND (project_id=ANY($2::uuid[]) OR project_id IS NULL) AND deleted_at IS NULL`
		args = append(args, projects)
	}
	if kind == property.Issue {
		query = `SELECT i.id::text FROM issues i WHERE ` + djangoapi.IssueVisibleSQL("i", "$2::uuid", "$1::uuid")
		args = []any{snapshot.workspace, snapshot.actor}
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := map[string]bool{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	snapshot.references[kind] = ids
	return ids, nil
}
